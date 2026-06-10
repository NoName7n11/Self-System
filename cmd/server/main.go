package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/ai"
	authapi "selfsystems/internal/auth"
	"selfsystems/internal/config"
	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
	"selfsystems/internal/extractor"
	"selfsystems/internal/gbus"
	httpapi "selfsystems/internal/http"
	postgresrepo "selfsystems/internal/repository/postgres"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	configureLogger(cfg.App.Env)

	categoryRepo, resourceRepo, todoRepo, reminderRepo, embeddingRepo, gbusFeatureStore, replayStore, eventStore, dbClose, err := buildRepositories(cfg)
	if err != nil {
		slog.Error("initialize repositories", "error", err)
		os.Exit(1)
	}
	defer dbClose()

	aiManager := ai.NewManager(cfg.AI.PrimaryProvider)
	heuristicProvider := ai.NewHeuristicProvider()
	aiManager.Register(heuristicProvider)
	aiManager.SetFallback(heuristicProvider.Name())
	aiManager.Register(ai.NewOpenAIProvider(ai.ProviderSettings{
		Enabled:        cfg.AI.OpenAI.Enabled,
		APIKey:         cfg.AI.OpenAI.APIKey,
		Model:          cfg.AI.OpenAI.Model,
		BaseURL:        cfg.AI.OpenAI.BaseURL,
		TimeoutSeconds: cfg.AI.OpenAI.TimeoutSeconds,
	}))
	aiManager.Register(ai.NewAnthropicProvider(ai.ProviderSettings{
		Enabled:        cfg.AI.Anthropic.Enabled,
		APIKey:         cfg.AI.Anthropic.APIKey,
		Model:          cfg.AI.Anthropic.Model,
		BaseURL:        cfg.AI.Anthropic.BaseURL,
		TimeoutSeconds: cfg.AI.Anthropic.TimeoutSeconds,
	}))
	aiManager.Register(ai.NewGeminiProvider(ai.ProviderSettings{
		Enabled:        cfg.AI.Gemini.Enabled,
		APIKey:         cfg.AI.Gemini.APIKey,
		Model:          cfg.AI.Gemini.Model,
		BaseURL:        cfg.AI.Gemini.BaseURL,
		TimeoutSeconds: cfg.AI.Gemini.TimeoutSeconds,
	}))

	// Embedding providers: real OpenAI first (when enabled), local hashing
	// embedder as the always-on offline fallback.
	aiManager.RegisterEmbedding(ai.NewOpenAIEmbeddingProvider(ai.ProviderSettings{
		Enabled:        cfg.AI.OpenAI.Enabled,
		APIKey:         cfg.AI.OpenAI.APIKey,
		BaseURL:        cfg.AI.OpenAI.BaseURL,
		TimeoutSeconds: cfg.AI.OpenAI.TimeoutSeconds,
	}))
	aiManager.RegisterEmbedding(ai.NewLocalEmbeddingProvider())
	embeddingSvc := service.NewEmbeddingService(embeddingRepo, aiManager)

	// Enrichment provider: real AI summary/key-points/entities when configured.
	aiManager.RegisterEnrichment(ai.NewOpenAIEnrichmentProvider(ai.ProviderSettings{
		Enabled:        cfg.AI.OpenAI.Enabled,
		APIKey:         cfg.AI.OpenAI.APIKey,
		Model:          cfg.AI.OpenAI.Model,
		BaseURL:        cfg.AI.OpenAI.BaseURL,
		TimeoutSeconds: cfg.AI.OpenAI.TimeoutSeconds,
	}))

	// Shared event observability — wired into all event-sourced services and the
	// events_health endpoint so metrics accumulate across domain boundaries.
	var eventObs *eventstore.EventObservability
	if eventStore != nil {
		eventObs = eventstore.NewEventObservability()
	}

	catSvcOpts := []service.CategoryServiceOption{}
	if cfg.Features.EventsCategoryEnabled && eventStore != nil {
		reg := eventstore.NewProjectorRegistry()
		reg.SetObservability(eventObs)
		eventstore.RegisterCategoryProjectors(reg, cfg.Database.Type)
		catSvcOpts = append(catSvcOpts,
			service.WithCategoryEventSourcing(eventStore, reg),
			service.WithCategoryEventObservability(eventObs),
		)
	}
	categorySvc := service.NewCategoryService(categoryRepo, catSvcOpts...)
	classifier := service.NewCategoryClassifier(categoryRepo, aiManager)

	resourceSvcOpts := []service.ResourceServiceOption{}
	if cfg.Features.EventsResourceEnabled && eventStore != nil {
		registry := eventstore.NewProjectorRegistry()
		registry.SetObservability(eventObs)
		eventstore.RegisterResourceProjectors(registry, cfg.Database.Type)
		resourceSvcOpts = append(resourceSvcOpts,
			service.WithEventSourcing(eventStore, registry),
			service.WithResourceEventObservability(eventObs),
		)
	}
	// Skim extractor: fires asynchronously after each URL resource is created.
	gbusEmitter := gbus.NewSignalEmitter(eventStore, cfg.GBUS.Enabled)
	gbusInference := gbus.NewInference(cfg.GBUS.ModelPath, cfg.GBUS.InferenceEnabled)
	resourceSvcOpts = append(resourceSvcOpts,
		service.WithSkimExtractor(extractor.NewURLExtractor()),
		service.WithClassificationThreshold(cfg.AI.ClassificationThreshold),
		service.WithResourceEmbeddingService(embeddingSvc),
		service.WithGBUSEmitter(gbusEmitter),
		service.WithGBUSInference(gbusInference),
	)
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc, resourceSvcOpts...)

	todoSvcOpts := []service.TodoServiceOption{}
	if cfg.Features.EventsTodoEnabled && eventStore != nil {
		reg := eventstore.NewProjectorRegistry()
		reg.SetObservability(eventObs)
		eventstore.RegisterTodoProjectors(reg, cfg.Database.Type)
		todoSvcOpts = append(todoSvcOpts,
			service.WithTodoEventSourcing(eventStore, reg),
			service.WithTodoEventObservability(eventObs),
		)
	}
	todoSvc := service.NewTodoService(todoRepo, todoSvcOpts...)

	reminderSvcOpts := []service.ReminderServiceOption{}
	if cfg.Features.EventsReminderEnabled && eventStore != nil {
		reg := eventstore.NewProjectorRegistry()
		reg.SetObservability(eventObs)
		eventstore.RegisterReminderProjectors(reg, cfg.Database.Type)
		reminderSvcOpts = append(reminderSvcOpts,
			service.WithReminderEventSourcing(eventStore, reg),
			service.WithReminderEventObservability(eventObs),
		)
	}
	reminderSvc := service.NewReminderService(reminderRepo, reminderSvcOpts...)
	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	chatSvc := service.NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc, graphSvc)
	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	defer runtimeCancel()

	// Start GBUS aggregator and drift monitor when enabled.
	var gbusMonitor *gbus.Monitor
	if cfg.GBUS.Enabled && gbusFeatureStore != nil && eventStore != nil {
		agg := gbus.NewAggregator(eventStore, gbusFeatureStore, cfg.GBUS.RetentionDays)
		agg.Start(runtimeCtx)
		gbusMonitor = gbus.NewMonitor(eventStore, gbusInference, cfg.GBUS.ModelPath)
		gbusMonitor.Start(runtimeCtx)
	}

	deepProcessor := service.NewDeepProcessor(resourceSvc, categoryRepo, categorySvc, aiManager, service.DeepProcessingSettings{
		Enabled:                     cfg.Features.DeepEnabled && cfg.Processing.Deep.Enabled,
		QueueCapacity:               cfg.Processing.Deep.QueueCapacity,
		WorkerCount:                 cfg.Processing.Deep.WorkerCount,
		BatchSize:                   cfg.Processing.Deep.BatchSize,
		MaxTasksPerMinute:           cfg.Processing.Deep.MaxTasksPerMinute,
		MaxTokensPerDay:             cfg.Processing.Deep.MaxTokensPerDay,
		MinReprocessIntervalSeconds: cfg.Processing.Deep.MinReprocessIntervalSeconds,
		ComplexityThreshold:         cfg.Processing.Deep.ComplexityThreshold,
		LowCostModel:                cfg.Processing.Deep.LowCostModel,
		HighCostModel:               cfg.Processing.Deep.HighCostModel,
		LowCostEstimatedTokens:      cfg.Processing.Deep.LowCostEstimatedTokens,
		HighCostEstimatedTokens:     cfg.Processing.Deep.HighCostEstimatedTokens,
		BudgetStatePath:             cfg.Processing.Deep.BudgetStatePath,
	}).
		WithContentFetcher(extractor.NewContentFetcher()).
		WithPDFExtractor(extractor.NewPDFExtractor()).
		WithImageExtractor(extractor.NewImageExtractor()).
		WithEventDetector(extractor.NewEventDetector()).
		WithReminderService(reminderSvc).
		WithEmbeddingService(embeddingSvc)
	deepProcessor.Start(runtimeCtx)

	syncHub := syncapi.NewHub()
	replayApplier := syncapi.NewServiceMutationApplier(resourceSvc, categorySvc, todoSvc, reminderSvc)
	replayManager := syncapi.NewOfflineReplayManagerWithApplier(replayStore, nil, syncHub, replayApplier)

	// Start the outbox worker when an event store is available. Per ADR 0018 the
	// event log is an audit trail + sync outbox, not a rebuildable source of
	// truth, so there is no snapshot worker.
	var outboxWorker *syncapi.OutboxWorker
	if eventStore != nil {
		outboxWorker = syncapi.NewOutboxWorker(eventStore, syncHub, 0)
		go outboxWorker.Start(runtimeCtx)
	}

	jwtService := authapi.NewJWTService(cfg.Auth)
	handler := httpapi.NewHandlerWithOptions(
		resourceSvc,
		categorySvc,
		todoSvc,
		reminderSvc,
		graphSvc,
		chatSvc,
		httpapi.WithSyncHub(syncHub),
		httpapi.WithDeepProcessor(deepProcessor),
		httpapi.WithAuthMiddleware(jwtService.Middleware()),
		httpapi.WithGBUSMonitor(gbusMonitor),
	)

	syncRouteOpts := []syncapi.BootstrapRouteOption{
		syncapi.WithOfflineReplayManager(replayManager),
	}
	if eventStore != nil {
		syncRouteOpts = append(syncRouteOpts,
			syncapi.WithEventStoreReplay(eventStore),
			syncapi.WithEventObservability(eventObs),
		)
	}
	if outboxWorker != nil {
		syncRouteOpts = append(syncRouteOpts, syncapi.WithOutboxWorker(outboxWorker))
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), httpapi.CORSMiddleware(cfg.Sync.AllowedOrigins))
	handler.RegisterRoutes(router)
	syncapi.RegisterBootstrapRoutes(router, cfg, syncHub, jwtService.Middleware(), syncRouteOpts...)

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("Self Systems API running", "address", cfg.Address())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("start server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	runtimeCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Warn("shutdown error", "error", err)
	}
}

func configureLogger(appEnv string) {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SS_LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	if strings.EqualFold(appEnv, "development") && level == slog.LevelInfo {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler).With("service", "selfsystems-api", "env", appEnv))
}

func buildRepositories(cfg config.Config) (
	categoryRepo domain.CategoryRepository,
	resourceRepo domain.ResourceRepository,
	todoRepo domain.TodoRepository,
	reminderRepo domain.ReminderRepository,
	embeddingRepo domain.EmbeddingRepository,
	gbusStore domain.GBUSFeatureStore,
	replayStore syncapi.OfflineReplayStore,
	evtStore eventstore.Store,
	dbClose func() error,
	err error,
) {
	switch strings.ToLower(strings.TrimSpace(cfg.Database.Type)) {
	case "", "sqlite":
		db, openErr := sqliterepo.Open(cfg.Database.Path)
		if openErr != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("open sqlite: %w", openErr)
		}

		sqliteReplayStore, replayErr := syncapi.NewSQLiteReplayStore(db)
		if replayErr != nil {
			_ = db.Close()
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("open sqlite replay store: %w", replayErr)
		}

		return sqliterepo.NewCategoryRepository(db),
			sqliterepo.NewResourceRepository(db),
			sqliterepo.NewTodoRepository(db),
			sqliterepo.NewReminderRepository(db),
			sqliterepo.NewEmbeddingRepository(db),
			sqliterepo.NewGBUSRepository(db),
			sqliteReplayStore,
			eventstore.NewSQLiteStore(db),
			db.Close,
			nil
	case "postgres", "postgresql":
		db, openErr := postgresrepo.Open(cfg.Database.URL)
		if openErr != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("open postgres: %w", openErr)
		}
		return postgresrepo.NewCategoryRepository(db),
			postgresrepo.NewResourceRepository(db),
			postgresrepo.NewTodoRepository(db),
			postgresrepo.NewReminderRepository(db),
			postgresrepo.NewEmbeddingRepository(db),
			nil, // GBUS feature store: Postgres migration pending
			nil,
			eventstore.NewPostgresStore(db),
			db.Close,
			nil
	default:
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("unsupported database type %q", cfg.Database.Type)
	}
}
