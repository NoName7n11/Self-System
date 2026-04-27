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

func TestAPICategoryResourceCRUDIntegration(t *testing.T) {
	router := newIntegrationRouter(t)

	categoryID := createCategory(t, router, `{"name":"Research","description":"Long-form notes"}`)
	resourceID := createResource(t, router, `{"url":"https://example.com/research","title":"Research Note","category_id":"`+categoryID+`"}`)

	getCategory := doRequest(t, router, http.MethodGet, "/api/v1/categories/"+categoryID, "")
	assertStatus(t, getCategory, http.StatusOK)

	updateCategory := doRequest(t, router, http.MethodPut, "/api/v1/categories/"+categoryID, `{"name":"knowledge systems","description":"Updated description"}`)
	assertStatus(t, updateCategory, http.StatusOK)
	var categoryPayload struct {
		Data struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	decodeJSON(t, updateCategory.Body.Bytes(), &categoryPayload)
	if categoryPayload.Data.Name != "Knowledge Systems" {
		t.Fatalf("expected normalized category name Knowledge Systems, got %q", categoryPayload.Data.Name)
	}

	getResource := doRequest(t, router, http.MethodGet, "/api/v1/resources/"+resourceID, "")
	assertStatus(t, getResource, http.StatusOK)

	updateResource := doRequest(t, router, http.MethodPut, "/api/v1/resources/"+resourceID, `{"url":"https://example.com/research-updated","title":"Research Updated","summary":"Updated summary","category_id":"`+categoryID+`"}`)
	assertStatus(t, updateResource, http.StatusOK)
	var resourcePayload struct {
		Data struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		} `json:"data"`
	}
	decodeJSON(t, updateResource.Body.Bytes(), &resourcePayload)
	if resourcePayload.Data.URL != "https://example.com/research-updated" {
		t.Fatalf("expected updated resource url, got %q", resourcePayload.Data.URL)
	}
	if resourcePayload.Data.Title != "Research Updated" {
		t.Fatalf("expected updated resource title, got %q", resourcePayload.Data.Title)
	}

	deleteResource := doRequest(t, router, http.MethodDelete, "/api/v1/resources/"+resourceID, "")
	assertStatus(t, deleteResource, http.StatusOK)

	getDeletedResource := doRequest(t, router, http.MethodGet, "/api/v1/resources/"+resourceID, "")
	assertStatus(t, getDeletedResource, http.StatusNotFound)
	assertErrorEnvelope(t, getDeletedResource.Body.Bytes(), "not_found")

	deleteCategory := doRequest(t, router, http.MethodDelete, "/api/v1/categories/"+categoryID, "")
	assertStatus(t, deleteCategory, http.StatusOK)

	getDeletedCategory := doRequest(t, router, http.MethodGet, "/api/v1/categories/"+categoryID, "")
	assertStatus(t, getDeletedCategory, http.StatusNotFound)
	assertErrorEnvelope(t, getDeletedCategory.Body.Bytes(), "not_found")
}

func TestAPITodoReminderCRUDIntegration(t *testing.T) {
	router := newIntegrationRouter(t)

	categoryID := createCategory(t, router, `{"name":"Planning","description":"Planning tasks"}`)
	resourceID := createResource(t, router, `{"url":"https://example.com/plan","title":"Planning Doc","category_id":"`+categoryID+`"}`)
	todoID := createTodoWithID(t, router, `{"title":"Initial todo","details":"todo details","due_at":"2026-04-21T10:00:00Z","resource_id":"`+resourceID+`"}`)
	reminderID := createReminderWithID(t, router, `{"title":"Initial reminder","message":"follow up","remind_at":"2026-04-22T10:00:00Z","resource_id":"`+resourceID+`"}`)

	getTodo := doRequest(t, router, http.MethodGet, "/api/v1/todos/"+todoID, "")
	assertStatus(t, getTodo, http.StatusOK)

	updateTodo := doRequest(t, router, http.MethodPut, "/api/v1/todos/"+todoID, `{"title":"Updated todo","details":"updated","status":"done","due_at":"2026-04-23T12:00:00Z","resource_id":"`+resourceID+`"}`)
	assertStatus(t, updateTodo, http.StatusOK)
	var todoPayload struct {
		Data struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"data"`
	}
	decodeJSON(t, updateTodo.Body.Bytes(), &todoPayload)
	if todoPayload.Data.Title != "Updated todo" {
		t.Fatalf("expected updated todo title, got %q", todoPayload.Data.Title)
	}
	if todoPayload.Data.Status != "done" {
		t.Fatalf("expected updated todo status done, got %q", todoPayload.Data.Status)
	}

	getReminder := doRequest(t, router, http.MethodGet, "/api/v1/reminders/"+reminderID, "")
	assertStatus(t, getReminder, http.StatusOK)

	updateReminder := doRequest(t, router, http.MethodPut, "/api/v1/reminders/"+reminderID, `{"title":"Updated reminder","message":"updated message","remind_at":"2026-04-24T09:15:00Z","status":"dismissed","resource_id":"`+resourceID+`"}`)
	assertStatus(t, updateReminder, http.StatusOK)
	var reminderPayload struct {
		Data struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"data"`
	}
	decodeJSON(t, updateReminder.Body.Bytes(), &reminderPayload)
	if reminderPayload.Data.Title != "Updated reminder" {
		t.Fatalf("expected updated reminder title, got %q", reminderPayload.Data.Title)
	}
	if reminderPayload.Data.Status != "dismissed" {
		t.Fatalf("expected updated reminder status dismissed, got %q", reminderPayload.Data.Status)
	}

	deleteTodo := doRequest(t, router, http.MethodDelete, "/api/v1/todos/"+todoID, "")
	assertStatus(t, deleteTodo, http.StatusOK)

	getDeletedTodo := doRequest(t, router, http.MethodGet, "/api/v1/todos/"+todoID, "")
	assertStatus(t, getDeletedTodo, http.StatusNotFound)
	assertErrorEnvelope(t, getDeletedTodo.Body.Bytes(), "not_found")

	deleteReminder := doRequest(t, router, http.MethodDelete, "/api/v1/reminders/"+reminderID, "")
	assertStatus(t, deleteReminder, http.StatusOK)

	getDeletedReminder := doRequest(t, router, http.MethodGet, "/api/v1/reminders/"+reminderID, "")
	assertStatus(t, getDeletedReminder, http.StatusNotFound)
	assertErrorEnvelope(t, getDeletedReminder.Body.Bytes(), "not_found")
}

func TestAPIChatCRUDCommandsIntegration(t *testing.T) {
	router := newIntegrationRouter(t)

	createdCategory := executeChatCommand(t, router, "create category research | notes")
	if createdCategory.Action != "category_created" || createdCategory.Category == nil {
		t.Fatalf("expected category_created with payload, got action=%q", createdCategory.Action)
	}
	categoryID := createdCategory.Category.ID

	if result := executeChatCommand(t, router, "get category "+categoryID); result.Action != "category_retrieved" {
		t.Fatalf("expected category_retrieved, got %q", result.Action)
	}
	if result := executeChatCommand(t, router, "update category "+categoryID+" | name=knowledge systems | description=updated"); result.Action != "category_updated" {
		t.Fatalf("expected category_updated, got %q", result.Action)
	}

	createdResource := executeChatCommand(t, router, "resource: https://example.com/chat-item | title=Chat Item | category=Knowledge Systems")
	if createdResource.Action != "resource_created" || createdResource.Resource == nil {
		t.Fatalf("expected resource_created with payload, got action=%q", createdResource.Action)
	}
	resourceID := createdResource.Resource.ID

	if result := executeChatCommand(t, router, "get resource "+resourceID); result.Action != "resource_retrieved" {
		t.Fatalf("expected resource_retrieved, got %q", result.Action)
	}
	if result := executeChatCommand(t, router, "update resource "+resourceID+" | title=Chat Item Updated | summary=updated"); result.Action != "resource_updated" {
		t.Fatalf("expected resource_updated, got %q", result.Action)
	}

	createdTodo := executeChatCommand(t, router, "create todo chat task | details=integration | due=2026-04-25T10:00:00Z")
	if createdTodo.Action != "todo_created" || createdTodo.Todo == nil {
		t.Fatalf("expected todo_created with payload, got action=%q", createdTodo.Action)
	}
	todoID := createdTodo.Todo.ID

	if result := executeChatCommand(t, router, "get todo "+todoID); result.Action != "todo_retrieved" {
		t.Fatalf("expected todo_retrieved, got %q", result.Action)
	}
	if result := executeChatCommand(t, router, "update todo "+todoID+" | title=chat task updated | status=done | due=2026-04-26T10:00:00Z"); result.Action != "todo_updated" {
		t.Fatalf("expected todo_updated, got %q", result.Action)
	}

	createdReminder := executeChatCommand(t, router, "create reminder chat reminder | at=2026-04-25T11:00:00Z | message=ping")
	if createdReminder.Action != "reminder_created" || createdReminder.Reminder == nil {
		t.Fatalf("expected reminder_created with payload, got action=%q", createdReminder.Action)
	}
	reminderID := createdReminder.Reminder.ID

	if result := executeChatCommand(t, router, "get reminder "+reminderID); result.Action != "reminder_retrieved" {
		t.Fatalf("expected reminder_retrieved, got %q", result.Action)
	}
	if result := executeChatCommand(t, router, "update reminder "+reminderID+" | title=chat reminder updated | at=2026-04-26T11:00:00Z | status=sent"); result.Action != "reminder_updated" {
		t.Fatalf("expected reminder_updated, got %q", result.Action)
	}

	if result := executeChatCommand(t, router, "delete resource "+resourceID); result.Action != "resource_deleted" {
		t.Fatalf("expected resource_deleted, got %q", result.Action)
	}
	if result := executeChatCommand(t, router, "delete todo "+todoID); result.Action != "todo_deleted" {
		t.Fatalf("expected todo_deleted, got %q", result.Action)
	}
	if result := executeChatCommand(t, router, "delete reminder "+reminderID); result.Action != "reminder_deleted" {
		t.Fatalf("expected reminder_deleted, got %q", result.Action)
	}
	if result := executeChatCommand(t, router, "delete category "+categoryID); result.Action != "category_deleted" {
		t.Fatalf("expected category_deleted, got %q", result.Action)
	}
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

func createTodoWithID(t *testing.T, router *gin.Engine, payload string) string {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/api/v1/todos", payload)
	assertStatus(t, resp, http.StatusCreated)
	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, resp.Body.Bytes(), &body)
	if strings.TrimSpace(body.Data.ID) == "" {
		t.Fatalf("expected non-empty todo id")
	}
	return body.Data.ID
}

func createReminder(t *testing.T, router *gin.Engine, payload string) {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/api/v1/reminders", payload)
	assertStatus(t, resp, http.StatusCreated)
}

func createReminderWithID(t *testing.T, router *gin.Engine, payload string) string {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/api/v1/reminders", payload)
	assertStatus(t, resp, http.StatusCreated)
	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, resp.Body.Bytes(), &body)
	if strings.TrimSpace(body.Data.ID) == "" {
		t.Fatalf("expected non-empty reminder id")
	}
	return body.Data.ID
}

func doRequest(t *testing.T, router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if method == http.MethodPost || method == http.MethodPatch || method == http.MethodPut {
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

func executeChatCommand(t *testing.T, router *gin.Engine, message string) service.ChatResult {
	t.Helper()

	requestBody, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		t.Fatalf("marshal chat command payload: %v", err)
	}

	response := doRequest(t, router, http.MethodPost, "/api/v1/chat/commands", string(requestBody))
	assertStatus(t, response, http.StatusOK)

	var payload struct {
		Data service.ChatResult `json:"data"`
	}
	decodeJSON(t, response.Body.Bytes(), &payload)
	return payload.Data
}
