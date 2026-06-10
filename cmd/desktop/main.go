package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"selfsystems/internal/ai"
	"selfsystems/internal/config"
	"selfsystems/internal/desktop"
	"selfsystems/internal/eventstore"
	"selfsystems/internal/extractor"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

// frontendDistFS resolves the built frontend (frontend/dist) relative to this
// source file, so the asset server works regardless of the process's cwd.
func frontendDistFS() fs.FS {
	_, thisFile, _, _ := runtime.Caller(0)
	return os.DirFS(filepath.Join(filepath.Dir(thisFile), "..", "..", "frontend", "dist"))
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := sqliterepo.Open(cfg.Database.Path)
	if err != nil {
		slog.Error("open sqlite", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	todoRepo := sqliterepo.NewTodoRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)
	embeddingRepo := sqliterepo.NewEmbeddingRepository(db)

	replayStore, err := syncapi.NewSQLiteReplayStore(db)
	if err != nil {
		slog.Error("open replay store", "error", err)
		os.Exit(1)
	}
	_ = replayStore

	evtStore := eventstore.NewSQLiteStore(db)

	aiManager := ai.NewManager(cfg.AI.PrimaryProvider)
	heuristic := ai.NewHeuristicProvider()
	aiManager.Register(heuristic)
	aiManager.SetFallback(heuristic.Name())
	aiManager.RegisterEmbedding(ai.NewLocalEmbeddingProvider())
	embeddingSvc := service.NewEmbeddingService(embeddingRepo, aiManager)

	catSvc := service.NewCategoryService(categoryRepo)
	classifier := service.NewCategoryClassifier(categoryRepo, aiManager)
	resourceSvc := service.NewResourceService(
		resourceRepo, categoryRepo, classifier, catSvc,
		service.WithSkimExtractor(extractor.NewURLExtractor()),
		service.WithClassificationThreshold(cfg.AI.ClassificationThreshold),
		service.WithResourceEmbeddingService(embeddingSvc),
	)
	todoSvc := service.NewTodoService(todoRepo)
	reminderSvc := service.NewReminderService(reminderRepo)

	if cfg.Features.EventsResourceEnabled {
		reg := eventstore.NewProjectorRegistry()
		eventstore.RegisterResourceProjectors(reg, "sqlite")
		resourceSvc = service.NewResourceService(
			resourceRepo, categoryRepo, classifier, catSvc,
			service.WithEventSourcing(evtStore, reg),
			service.WithSkimExtractor(extractor.NewURLExtractor()),
			service.WithClassificationThreshold(cfg.AI.ClassificationThreshold),
			service.WithResourceEmbeddingService(embeddingSvc),
		)
	}

	app := desktop.NewApp(desktop.AppOptions{
		Resources:  resourceSvc,
		Categories: catSvc,
		Todos:      todoSvc,
		Reminders:  reminderSvc,
	})

	if err := wails.Run(&options.App{
		Title:  "Self Systems",
		Width:  1280,
		Height: 820,
		AssetServer: &assetserver.Options{
			Assets: frontendDistFS(),
		},
		BackgroundColour:        &options.RGBA{R: 17, G: 17, B: 17, A: 255},
		OnStartup:               app.Startup,
		OnShutdown:              app.Shutdown,
		Bind:                    []interface{}{app},
		EnableDefaultContextMenu: false,
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
	}); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func configureLogger(appEnv string) {
	level := slog.LevelInfo
	if strings.EqualFold(appEnv, "development") {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler).With("service", "selfsystems-desktop", "env", appEnv))
}
