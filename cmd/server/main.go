package main

import (
	"context"
	"fmt"
	"log"
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
	httpapi "selfsystems/internal/http"
	postgresrepo "selfsystems/internal/repository/postgres"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	categoryRepo, resourceRepo, todoRepo, reminderRepo, replayStore, dbClose, err := buildRepositories(cfg)
	if err != nil {
		log.Fatalf("initialize repositories: %v", err)
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

	categorySvc := service.NewCategoryService(categoryRepo)
	classifier := service.NewCategoryClassifier(categoryRepo, aiManager)
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc)
	todoSvc := service.NewTodoService(todoRepo)
	reminderSvc := service.NewReminderService(reminderRepo)
	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	chatSvc := service.NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc, graphSvc)
	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	defer runtimeCancel()

	deepProcessor := service.NewDeepProcessor(resourceSvc, categoryRepo, categorySvc, aiManager, service.DeepProcessingSettings{
		Enabled:                 cfg.Features.DeepEnabled && cfg.Processing.Deep.Enabled,
		QueueCapacity:           cfg.Processing.Deep.QueueCapacity,
		WorkerCount:             cfg.Processing.Deep.WorkerCount,
		MaxTasksPerMinute:       cfg.Processing.Deep.MaxTasksPerMinute,
		MaxTokensPerDay:         cfg.Processing.Deep.MaxTokensPerDay,
		ComplexityThreshold:     cfg.Processing.Deep.ComplexityThreshold,
		LowCostModel:            cfg.Processing.Deep.LowCostModel,
		HighCostModel:           cfg.Processing.Deep.HighCostModel,
		LowCostEstimatedTokens:  cfg.Processing.Deep.LowCostEstimatedTokens,
		HighCostEstimatedTokens: cfg.Processing.Deep.HighCostEstimatedTokens,
	})
	deepProcessor.Start(runtimeCtx)

	syncHub := syncapi.NewHub()
	replayApplier := syncapi.NewServiceMutationApplier(resourceSvc, categorySvc, todoSvc, reminderSvc)
	replayManager := syncapi.NewOfflineReplayManagerWithApplier(replayStore, nil, syncHub, replayApplier)
	handler := httpapi.NewHandlerWithOptions(resourceSvc, categorySvc, todoSvc, reminderSvc, graphSvc, chatSvc, httpapi.WithSyncHub(syncHub), httpapi.WithDeepProcessor(deepProcessor))
	jwtService := authapi.NewJWTService(cfg.Auth)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler.RegisterRoutes(router)
	syncapi.RegisterBootstrapRoutes(router, cfg, syncHub, jwtService.Middleware(), syncapi.WithOfflineReplayManager(replayManager))

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Self Systems API running on http://%s", cfg.Address())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	runtimeCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func buildRepositories(cfg config.Config) (
	categoryRepo domain.CategoryRepository,
	resourceRepo domain.ResourceRepository,
	todoRepo domain.TodoRepository,
	reminderRepo domain.ReminderRepository,
	replayStore syncapi.OfflineReplayStore,
	dbClose func() error,
	err error,
) {
	switch strings.ToLower(strings.TrimSpace(cfg.Database.Type)) {
	case "", "sqlite":
		db, openErr := sqliterepo.Open(cfg.Database.Path)
		if openErr != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("open sqlite: %w", openErr)
		}

		sqliteReplayStore, replayErr := syncapi.NewSQLiteReplayStore(db)
		if replayErr != nil {
			_ = db.Close()
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("open sqlite replay store: %w", replayErr)
		}

		return sqliterepo.NewCategoryRepository(db), sqliterepo.NewResourceRepository(db), sqliterepo.NewTodoRepository(db), sqliterepo.NewReminderRepository(db), sqliteReplayStore, db.Close, nil
	case "postgres", "postgresql":
		db, openErr := postgresrepo.Open(cfg.Database.URL)
		if openErr != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("open postgres: %w", openErr)
		}
		return postgresrepo.NewCategoryRepository(db), postgresrepo.NewResourceRepository(db), postgresrepo.NewTodoRepository(db), postgresrepo.NewReminderRepository(db), nil, db.Close, nil
	default:
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unsupported database type %q", cfg.Database.Type)
	}
}
