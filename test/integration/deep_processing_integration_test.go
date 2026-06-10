package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/ai"
	httpapi "selfsystems/internal/http"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
)

func TestDeepProcessingActivationAndCostMetricsIntegration(t *testing.T) {
	router := newDeepIntegrationRouter(t)

	resourceResponse := doRequest(t, router, http.MethodPost, "/api/v1/resources", `{"url":"https://example.com/deep-int","title":"Deep Integration","summary":"initial summary","category_name":"Research"}`)
	assertStatus(t, resourceResponse, http.StatusCreated)

	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, resourceResponse.Body.Bytes(), &created)
	if strings.TrimSpace(created.Data.ID) == "" {
		t.Fatalf("expected resource id in create response")
	}

	waitForIntegrationCondition(t, 4*time.Second, func() bool {
		metricsResponse := doRequest(t, router, http.MethodGet, "/api/v1/processing/deep/metrics", "")
		if metricsResponse.Code != http.StatusOK {
			return false
		}
		var metrics struct {
			Data service.DeepProcessingMetrics `json:"data"`
		}
		decodeJSON(t, metricsResponse.Body.Bytes(), &metrics)
		return metrics.Data.ProcessedTotal >= 1 && metrics.Data.TokensUsedToday > 0
	})

	resourceByID := doRequest(t, router, http.MethodGet, "/api/v1/resources/"+created.Data.ID, "")
	assertStatus(t, resourceByID, http.StatusOK)
	var loaded struct {
		Data struct {
			Summary string `json:"summary"`
		} `json:"data"`
	}
	decodeJSON(t, resourceByID.Body.Bytes(), &loaded)
	if !strings.Contains(loaded.Data.Summary, "[deep-processing]") {
		t.Fatalf("expected deep-processing marker in summary, got %q", loaded.Data.Summary)
	}

	healthResponse := doRequest(t, router, http.MethodGet, "/api/v1/processing/deep/health", "")
	assertStatus(t, healthResponse, http.StatusOK)
	var health struct {
		Data service.DeepProcessingHealth `json:"data"`
	}
	decodeJSON(t, healthResponse.Body.Bytes(), &health)
	if health.Data.Status != "ok" {
		t.Fatalf("expected deep health status ok, got %q", health.Data.Status)
	}
}

func newDeepIntegrationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "self_systems_deep_integration.db")
	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	todoRepo := sqliterepo.NewTodoRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)

	aiManager := ai.NewManager("heuristic")
	heuristicProvider := ai.NewHeuristicProvider()
	aiManager.Register(heuristicProvider)
	aiManager.SetFallback(heuristicProvider.Name())

	categorySvc := service.NewCategoryService(categoryRepo)
	classifier := service.NewCategoryClassifier(categoryRepo, aiManager)
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc)
	todoSvc := service.NewTodoService(todoRepo)
	reminderSvc := service.NewReminderService(reminderRepo)
	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	chatSvc := service.NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc, graphSvc)

	processor := service.NewDeepProcessor(resourceSvc, categoryRepo, categorySvc, aiManager, service.DeepProcessingSettings{
		Enabled:                 true,
		QueueCapacity:           32,
		WorkerCount:             1,
		MaxTasksPerMinute:       120,
		MaxTokensPerDay:         100000,
		ComplexityThreshold:     5,
		LowCostEstimatedTokens:  50,
		HighCostEstimatedTokens: 200,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	processor.Start(ctx)

	handler := httpapi.NewHandlerWithOptions(resourceSvc, categorySvc, todoSvc, reminderSvc, graphSvc, chatSvc, httpapi.WithDeepProcessor(processor))
	router := gin.New()
	handler.RegisterRoutes(router)
	return router
}

func waitForIntegrationCondition(t *testing.T, timeout time.Duration, predicate func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}

	metricsResponseBody, _ := json.Marshal(map[string]string{"error": "timeout waiting for integration condition"})
	t.Fatalf("condition not satisfied within %s: %s", timeout, string(metricsResponseBody))
}
