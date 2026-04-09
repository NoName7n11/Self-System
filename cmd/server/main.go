package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/ai"
	"selfsystems/internal/config"
	httpapi "selfsystems/internal/http"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := sqliterepo.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	todoRepo := sqliterepo.NewTodoRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)

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

	handler := httpapi.NewHandler(resourceSvc, categorySvc, todoSvc, reminderSvc, graphSvc, chatSvc)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler.RegisterRoutes(router)

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
