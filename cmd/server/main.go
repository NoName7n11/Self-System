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

	categorySvc := service.NewCategoryService(categoryRepo)
	classifier := service.NewCategoryClassifier(categoryRepo)
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc)
	todoSvc := service.NewTodoService(todoRepo)
	reminderSvc := service.NewReminderService(reminderRepo)
	chatSvc := service.NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc)

	handler := httpapi.NewHandler(resourceSvc, categorySvc, todoSvc, reminderSvc, chatSvc)

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
