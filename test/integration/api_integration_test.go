package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/ai"
	httpapi "selfsystems/internal/http"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
)

func TestAPIFullFlowIntegration(t *testing.T) {
	router := newIntegrationRouter(t)

	categoryID := createCategory(t, router, `{"name":"AI","description":"Artificial intelligence"}`)
	resourceID := createResource(t, router, `{"url":"https://example.com/ai-graph","title":"AI Graph Systems","category_id":"`+categoryID+`"}`)
	createTodo(t, router, `{"title":"Review AI Graph article","resource_id":"`+resourceID+`"}`)
	createReminder(t, router, `{"title":"Follow up","remind_at":"2026-04-20T10:00:00Z","resource_id":"`+resourceID+`"}`)

	searchResponse := doRequest(t, router, http.MethodGet, "/api/v1/resources/search?q=AI", "")
	assertStatus(t, searchResponse, http.StatusOK)

	semanticResponse := doRequest(t, router, http.MethodGet, "/api/v1/resources/semantic-search?q=knowledge+graph", "")
	assertStatus(t, semanticResponse, http.StatusOK)

	graphResponse := doRequest(t, router, http.MethodGet, "/api/v1/graph?limit=100", "")
	assertStatus(t, graphResponse, http.StatusOK)
	var graphPayload struct {
		Data struct {
			Nodes []any `json:"nodes"`
			Edges []any `json:"edges"`
			Stats struct {
				CategoryCount int `json:"category_count"`
				ResourceCount int `json:"resource_count"`
				EdgeCount     int `json:"edge_count"`
			} `json:"stats"`
		} `json:"data"`
	}
	decodeJSON(t, graphResponse.Body.Bytes(), &graphPayload)
	if graphPayload.Data.Stats.CategoryCount < 1 || graphPayload.Data.Stats.ResourceCount < 1 || graphPayload.Data.Stats.EdgeCount < 1 {
		t.Fatalf("expected non-empty graph stats, got %+v", graphPayload.Data.Stats)
	}

	chatGraphResponse := doRequest(t, router, http.MethodPost, "/api/v1/chat/commands", `{"message":"graph | limit=100"}`)
	assertStatus(t, chatGraphResponse, http.StatusOK)
	var chatPayload struct {
		Data struct {
			Action string `json:"action"`
		} `json:"data"`
	}
	decodeJSON(t, chatGraphResponse.Body.Bytes(), &chatPayload)
	if chatPayload.Data.Action != "graph_data" {
		t.Fatalf("expected chat graph action graph_data, got %q", chatPayload.Data.Action)
	}
}

func TestAPIErrorEnvelopeIntegration(t *testing.T) {
	router := newIntegrationRouter(t)

	invalidResourceResponse := doRequest(t, router, http.MethodPost, "/api/v1/resources", `{"url":"not-a-url"}`)
	assertStatus(t, invalidResourceResponse, http.StatusBadRequest)
	assertErrorEnvelope(t, invalidResourceResponse.Body.Bytes(), "validation_error")

	invalidChatPayload := doRequest(t, router, http.MethodPost, "/api/v1/chat/commands", "{")
	assertStatus(t, invalidChatPayload, http.StatusBadRequest)
	assertErrorEnvelope(t, invalidChatPayload.Body.Bytes(), "invalid_payload")
}

func newIntegrationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "self_systems_integration.db")
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

	handler := httpapi.NewHandler(resourceSvc, categorySvc, todoSvc, reminderSvc, graphSvc, chatSvc)
	router := gin.New()
	handler.RegisterRoutes(router)
	return router
}

func createCategory(t *testing.T, router *gin.Engine, payload string) string {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/api/v1/categories", payload)
	assertStatus(t, resp, http.StatusCreated)
	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, resp.Body.Bytes(), &body)
	if strings.TrimSpace(body.Data.ID) == "" {
		t.Fatalf("expected non-empty category id")
	}
	return body.Data.ID
}

func createResource(t *testing.T, router *gin.Engine, payload string) string {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/api/v1/resources", payload)
	assertStatus(t, resp, http.StatusCreated)
	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, resp.Body.Bytes(), &body)
	if strings.TrimSpace(body.Data.ID) == "" {
		t.Fatalf("expected non-empty resource id")
	}
	return body.Data.ID
}

func createTodo(t *testing.T, router *gin.Engine, payload string) {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/api/v1/todos", payload)
	assertStatus(t, resp, http.StatusCreated)
}

func createReminder(t *testing.T, router *gin.Engine, payload string) {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/api/v1/reminders", payload)
	assertStatus(t, resp, http.StatusCreated)
}

func doRequest(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if method == http.MethodPost || method == http.MethodPatch {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if response.Code != expected {
		t.Fatalf("expected status %d, got %d, body=%s", expected, response.Code, response.Body.String())
	}
}

func assertErrorEnvelope(t *testing.T, body []byte, expectedCode string) {
	t.Helper()
	var response map[string]string
	decodeJSON(t, body, &response)
	if strings.TrimSpace(response["error"]) == "" {
		t.Fatalf("expected non-empty error field")
	}
	if response["code"] != expectedCode {
		t.Fatalf("expected code %q, got %q", expectedCode, response["code"])
	}
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode json: %v, body=%s", err, string(body))
	}
}
