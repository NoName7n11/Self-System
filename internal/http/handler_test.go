package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

type graphCategoryRepoStub struct {
	items   []domain.Category
	listErr error
}

func (s graphCategoryRepoStub) List(ctx context.Context) ([]domain.Category, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

func (s graphCategoryRepoStub) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	return nil, nil
}

func (s graphCategoryRepoStub) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	return nil, nil
}

func (s graphCategoryRepoStub) Create(ctx context.Context, c *domain.Category) error {
	return nil
}

func (s graphCategoryRepoStub) Update(ctx context.Context, c *domain.Category) error {
	return nil
}

func (s graphCategoryRepoStub) Delete(ctx context.Context, id string) error {
	return nil
}

func (s graphCategoryRepoStub) IncrementAccept(ctx context.Context, id string) error {
	return nil
}

func (s graphCategoryRepoStub) IncrementOverride(ctx context.Context, id string) error {
	return nil
}

type graphResourceRepoStub struct {
	items           []domain.Resource
	lastLimit       int
	lastOffset      int
	listErr         error
	getByIDErr      error
	createErr       error
	searchItems     []domain.Resource
	lastSearchQuery string
	lastSearchLimit int
	searchErr       error
	created         []domain.Resource
	updateErr       error
	deleteErr       error
	lastUpdateID    string
	lastUpdateCatID string
	lastUpdateOver  bool
}

func (s *graphResourceRepoStub) GetByID(ctx context.Context, id string) (*domain.Resource, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (s *graphResourceRepoStub) Create(ctx context.Context, r *domain.Resource) error {
	if s.createErr != nil {
		return s.createErr
	}
	if r != nil {
		s.created = append(s.created, *r)
	}
	return nil
}

func (s *graphResourceRepoStub) Update(ctx context.Context, r *domain.Resource) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if r == nil {
		return nil
	}
	for i := range s.items {
		if s.items[i].ID == r.ID {
			s.items[i] = *r
			return nil
		}
	}
	s.items = append(s.items, *r)
	return nil
}

func (s *graphResourceRepoStub) Delete(ctx context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			break
		}
	}
	return nil
}

func (s *graphResourceRepoStub) List(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

func (s *graphResourceRepoStub) Search(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	s.lastSearchQuery = query
	s.lastSearchLimit = limit
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	if s.searchItems != nil {
		return s.searchItems, nil
	}
	return nil, nil
}

func (s *graphResourceRepoStub) UpdateCategory(ctx context.Context, resourceID, categoryID string, userOverride bool) error {
	s.lastUpdateID = resourceID
	s.lastUpdateCatID = categoryID
	s.lastUpdateOver = userOverride
	if s.updateErr != nil {
		return s.updateErr
	}
	return nil
}

func (s *graphResourceRepoStub) UpdateExtractedData(_ context.Context, _ string, _ domain.ResourceExtractedData) error {
	return nil
}
func (s *graphResourceRepoStub) FindByURL(_ context.Context, _ string) (*domain.Resource, error) {
	return nil, nil
}
func (s *graphResourceRepoStub) IncrementCounter(_ context.Context, _ string) error { return nil }
func (s *graphResourceRepoStub) ListArchived(_ context.Context, _, _ int) ([]domain.Resource, error) {
	return []domain.Resource{}, nil
}
func (s *graphResourceRepoStub) Archive(_ context.Context, _ string, _ domain.ArchiveReason) error {
	return nil
}
func (s *graphResourceRepoStub) Restore(_ context.Context, _ string) error { return nil }
func (s *graphResourceRepoStub) BulkArchive(_ context.Context, _ []string, _ domain.ArchiveReason) error {
	return nil
}
func (s *graphResourceRepoStub) BulkRestore(_ context.Context, _ []string) error { return nil }

type categoryRepoCRUDStub struct {
	items        []domain.Category
	createErr    error
	getByIDErr   error
	listErr      error
	getByNameErr error
	updateErr    error
	deleteErr    error
}

func (s *categoryRepoCRUDStub) List(ctx context.Context) ([]domain.Category, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

func (s *categoryRepoCRUDStub) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (s *categoryRepoCRUDStub) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	if s.getByNameErr != nil {
		return nil, s.getByNameErr
	}
	for i := range s.items {
		if strings.EqualFold(strings.TrimSpace(s.items[i].Name), strings.TrimSpace(name)) {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (s *categoryRepoCRUDStub) Create(ctx context.Context, c *domain.Category) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.items = append(s.items, *c)
	return nil
}

func (s *categoryRepoCRUDStub) Update(ctx context.Context, c *domain.Category) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	for i := range s.items {
		if s.items[i].ID == c.ID {
			s.items[i] = *c
			return nil
		}
	}
	return nil
}

func (s *categoryRepoCRUDStub) Delete(ctx context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			break
		}
	}
	return nil
}

func (s *categoryRepoCRUDStub) IncrementAccept(ctx context.Context, id string) error {
	return nil
}

func (s *categoryRepoCRUDStub) IncrementOverride(ctx context.Context, id string) error {
	return nil
}

type todoRepoCRUDStub struct {
	items      []domain.Todo
	getByIDErr error
	createErr  error
	updateErr  error
	deleteErr  error
	listErr    error
	lastLimit  int
	lastOffset int
}

func (s *todoRepoCRUDStub) GetByID(ctx context.Context, id string) (*domain.Todo, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (s *todoRepoCRUDStub) Create(ctx context.Context, t *domain.Todo) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.items = append(s.items, *t)
	return nil
}

func (s *todoRepoCRUDStub) Update(ctx context.Context, t *domain.Todo) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	for i := range s.items {
		if s.items[i].ID == t.ID {
			s.items[i] = *t
			return nil
		}
	}
	return nil
}

func (s *todoRepoCRUDStub) Delete(ctx context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			break
		}
	}
	return nil
}

func (s *todoRepoCRUDStub) List(ctx context.Context, limit, offset int) ([]domain.Todo, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

type reminderRepoCRUDStub struct {
	items      []domain.Reminder
	getByIDErr error
	createErr  error
	updateErr  error
	deleteErr  error
	listErr    error
	lastLimit  int
	lastOffset int
}

func (s *reminderRepoCRUDStub) GetByID(ctx context.Context, id string) (*domain.Reminder, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (s *reminderRepoCRUDStub) Create(ctx context.Context, r *domain.Reminder) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.items = append(s.items, *r)
	return nil
}

func (s *reminderRepoCRUDStub) Update(ctx context.Context, r *domain.Reminder) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	for i := range s.items {
		if s.items[i].ID == r.ID {
			s.items[i] = *r
			return nil
		}
	}
	return nil
}

func (s *reminderRepoCRUDStub) Delete(ctx context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			break
		}
	}
	return nil
}

func (s *reminderRepoCRUDStub) List(ctx context.Context, limit, offset int) ([]domain.Reminder, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

func TestGetGraphReturnsGraphData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := graphCategoryRepoStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{
		ID:           "res-1",
		Title:        "AI Article",
		URL:          "https://example.com",
		CategoryID:   "cat-1",
		CategoryName: "AI",
	}}}

	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	handler := NewHandler(nil, nil, nil, nil, graphSvc, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/graph?limit=10", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if resourceRepo.lastLimit != 10 {
		t.Fatalf("expected limit 10, got %d", resourceRepo.lastLimit)
	}

	var response struct {
		Data service.GraphData `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Data.Stats.CategoryCount != 1 {
		t.Fatalf("expected 1 category node, got %d", response.Data.Stats.CategoryCount)
	}
	if response.Data.Stats.ResourceCount != 1 {
		t.Fatalf("expected 1 resource node, got %d", response.Data.Stats.ResourceCount)
	}
	if response.Data.Stats.EdgeCount != 1 {
		t.Fatalf("expected 1 edge, got %d", response.Data.Stats.EdgeCount)
	}
}

func TestGetGraphDefaultsLimitWhenInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := graphCategoryRepoStub{}
	resourceRepo := &graphResourceRepoStub{}

	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	handler := NewHandler(nil, nil, nil, nil, graphSvc, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/graph?limit=invalid", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if resourceRepo.lastLimit != 1000 {
		t.Fatalf("expected default limit 1000, got %d", resourceRepo.lastLimit)
	}
}

func TestGetGraphReturnsInternalServerErrorOnBuildFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := graphCategoryRepoStub{}
	resourceRepo := &graphResourceRepoStub{listErr: errors.New("db unavailable")}

	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	handler := NewHandler(nil, nil, nil, nil, graphSvc, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/graph", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	errorMessage := response["error"]
	if errorMessage != "internal server error" {
		t.Fatalf("expected generic internal error, got %q", errorMessage)
	}

	if response["code"] != "internal_error" {
		t.Fatalf("expected code internal_error, got %q", response["code"])
	}
}

func TestGetGraphReturnsServiceUnavailableWhenGraphNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/graph", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "graph service is not configured" {
		t.Fatalf("expected graph service unavailable error, got %q", response["error"])
	}

	if response["code"] != "service_unavailable" {
		t.Fatalf("expected code service_unavailable, got %q", response["code"])
	}
}

func TestSearchResourcesReturnsResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{searchItems: []domain.Resource{{
		ID:    "res-1",
		URL:   "https://example.com",
		Title: "AI Search Result",
	}}}
	resourceSvc := service.NewResourceService(resourceRepo, graphCategoryRepoStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/search?q=ai&limit=7", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if resourceRepo.lastSearchQuery != "ai" {
		t.Fatalf("expected query ai, got %q", resourceRepo.lastSearchQuery)
	}
	if resourceRepo.lastSearchLimit != 7 {
		t.Fatalf("expected search limit 7, got %d", resourceRepo.lastSearchLimit)
	}

	var response struct {
		Data []domain.Resource `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(response.Data))
	}
}

func TestSearchResourcesReturnsBadRequestWhenQueryMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, graphCategoryRepoStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/search", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "q is required" {
		t.Fatalf("expected missing q error, got %q", response["error"])
	}

	if response["code"] != "validation_error" {
		t.Fatalf("expected code validation_error, got %q", response["code"])
	}
}

func TestSearchResourcesReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{searchErr: errors.New("search failure")}
	resourceSvc := service.NewResourceService(resourceRepo, graphCategoryRepoStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/search?q=ai", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "internal server error" {
		t.Fatalf("expected generic internal error, got %q", response["error"])
	}

	if response["code"] != "internal_error" {
		t.Fatalf("expected code internal_error, got %q", response["code"])
	}
}

func TestSearchResourcesReturnsServiceUnavailableWhenResourceServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/search?q=ai", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "resource service is not configured" {
		t.Fatalf("expected resource service unavailable error, got %q", response["error"])
	}

	if response["code"] != "service_unavailable" {
		t.Fatalf("expected code service_unavailable, got %q", response["code"])
	}
}

func TestExecuteChatCommandGraphReturnsGraphData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := graphCategoryRepoStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{
		ID:           "res-1",
		Title:        "AI Article",
		URL:          "https://example.com",
		CategoryID:   "cat-1",
		CategoryName: "AI",
	}}}

	graphSvc := service.NewGraphService(categoryRepo, resourceRepo)
	chatSvc := service.NewChatService(nil, nil, nil, nil, graphSvc)
	handler := NewHandler(nil, nil, nil, nil, graphSvc, chatSvc)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/commands", strings.NewReader(`{"message":"graph | limit=25"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if resourceRepo.lastLimit != 25 {
		t.Fatalf("expected graph command limit 25, got %d", resourceRepo.lastLimit)
	}

	var response struct {
		Data struct {
			Action string             `json:"action"`
			Graph  *service.GraphData `json:"graph"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Data.Action != "graph_data" {
		t.Fatalf("expected action graph_data, got %q", response.Data.Action)
	}

	if response.Data.Graph == nil {
		t.Fatalf("expected graph payload")
	}

	if response.Data.Graph.Stats.CategoryCount != 1 {
		t.Fatalf("expected 1 category node, got %d", response.Data.Graph.Stats.CategoryCount)
	}
	if response.Data.Graph.Stats.ResourceCount != 1 {
		t.Fatalf("expected 1 resource node, got %d", response.Data.Graph.Stats.ResourceCount)
	}
	if response.Data.Graph.Stats.EdgeCount != 1 {
		t.Fatalf("expected 1 edge, got %d", response.Data.Graph.Stats.EdgeCount)
	}
}

func TestExecuteChatCommandReturnsServiceUnavailableWhenChatNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/commands", strings.NewReader(`{"message":"graph"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "chat service is not configured" {
		t.Fatalf("expected chat service unavailable error, got %q", response["error"])
	}

	if response["code"] != "service_unavailable" {
		t.Fatalf("expected code service_unavailable, got %q", response["code"])
	}
}

func TestExecuteChatCommandReturnsBadRequestOnEmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	chatSvc := service.NewChatService(nil, nil, nil, nil, nil)
	handler := NewHandler(nil, nil, nil, nil, nil, chatSvc)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/commands", strings.NewReader(`{"message":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if !strings.Contains(response["error"], "message is required") {
		t.Fatalf("expected validation message, got %q", response["error"])
	}

	if response["code"] != "validation_error" {
		t.Fatalf("expected code validation_error, got %q", response["code"])
	}
}

func TestExecuteChatCommandReturnsBadRequestOnInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/commands", strings.NewReader("{"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "invalid payload" {
		t.Fatalf("expected invalid payload error, got %q", response["error"])
	}

	if response["code"] != "invalid_payload" {
		t.Fatalf("expected code invalid_payload, got %q", response["code"])
	}
}

func TestSemanticSearchResourcesReturnsResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := graphCategoryRepoStub{}
	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{
		ID:           "res-1",
		Title:        "Knowledge Graph Systems for AI Agents",
		Summary:      "Building graph memory for assistants",
		URL:          "https://example.com/graph-ai",
		CategoryID:   "cat-1",
		CategoryName: "AI",
	}}}

	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/semantic-search?q=knowledge+graph&limit=5", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if resourceRepo.lastLimit != 500 {
		t.Fatalf("expected semantic candidate fetch limit 500, got %d", resourceRepo.lastLimit)
	}

	var response struct {
		Data []domain.Resource `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 semantic search result, got %d", len(response.Data))
	}

	if response.Data[0].ID != "res-1" {
		t.Fatalf("expected top result res-1, got %q", response.Data[0].ID)
	}
}

func TestSemanticSearchResourcesReturnsBadRequestWhenQueryMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, graphCategoryRepoStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/semantic-search", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "q is required" {
		t.Fatalf("expected missing q error, got %q", response["error"])
	}

	if response["code"] != "validation_error" {
		t.Fatalf("expected code validation_error, got %q", response["code"])
	}
}

func TestSemanticSearchResourcesReturnsServiceUnavailableWhenResourceServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/semantic-search?q=graph", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "resource service is not configured" {
		t.Fatalf("expected resource service unavailable error, got %q", response["error"])
	}

	if response["code"] != "service_unavailable" {
		t.Fatalf("expected code service_unavailable, got %q", response["code"])
	}
}

func TestCreateResourceReturnsServiceUnavailableWhenResourceServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{"url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "resource service is not configured")
}

func TestListResourcesReturnsServiceUnavailableWhenResourceServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "resource service is not configured")
}

func TestUpdateResourceCategoryReturnsServiceUnavailableWhenResourceServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/resources/res-1/category", strings.NewReader(`{"category_id":"cat-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "resource service is not configured")
}

func TestCreateCategoryReturnsServiceUnavailableWhenCategoryServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"AI"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "category service is not configured")
}

func TestListCategoriesReturnsServiceUnavailableWhenCategoryServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "category service is not configured")
}

func TestCreateTodoReturnsServiceUnavailableWhenTodoServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/todos", strings.NewReader(`{"title":"Task"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "todo service is not configured")
}

func TestListTodosReturnsServiceUnavailableWhenTodoServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "todo service is not configured")
}

func TestCreateReminderReturnsServiceUnavailableWhenReminderServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/reminders", strings.NewReader(`{"title":"Follow up","remind_at":"2026-04-20T10:00:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "reminder service is not configured")
}

func TestListRemindersReturnsServiceUnavailableWhenReminderServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reminders", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "reminder service is not configured")
}

func TestCreateCategoryReturnsCreatedCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"research","description":"notes"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response struct {
		Data domain.Category `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse category response: %v", err)
	}

	if response.Data.Name != "Research" {
		t.Fatalf("expected normalized name Research, got %q", response.Data.Name)
	}
	if len(categoryRepo.items) != 1 {
		t.Fatalf("expected category repo to store one item, got %d", len(categoryRepo.items))
	}
}

func TestListCategoriesReturnsCategoryData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}, {ID: "cat-2", Name: "Research"}}}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Data []domain.Category `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse categories response: %v", err)
	}

	if len(response.Data) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(response.Data))
	}
}

func TestCreateTodoReturnsCreatedTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/todos", strings.NewReader(`{"title":"Write tests","details":"batch session","due_at":"2026-04-20T10:00:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response struct {
		Data domain.Todo `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse todo response: %v", err)
	}

	if response.Data.Title != "Write tests" {
		t.Fatalf("expected todo title Write tests, got %q", response.Data.Title)
	}
	if response.Data.Status != domain.TodoStatusOpen {
		t.Fatalf("expected todo status open, got %q", response.Data.Status)
	}
	if len(todoRepo.items) != 1 {
		t.Fatalf("expected todo repo to store one item, got %d", len(todoRepo.items))
	}
}

func TestListTodosReturnsTodoDataWithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Task", Status: domain.TodoStatusOpen}}}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/todos?limit=5&offset=2", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if todoRepo.lastLimit != 5 || todoRepo.lastOffset != 2 {
		t.Fatalf("expected pagination limit=5 offset=2, got limit=%d offset=%d", todoRepo.lastLimit, todoRepo.lastOffset)
	}

	var response struct {
		Data []domain.Todo `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse todos response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 todo item, got %d", len(response.Data))
	}
}

func TestListTodosDefaultsPaginationWhenInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Task", Status: domain.TodoStatusOpen}}}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/todos?limit=0&offset=-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if todoRepo.lastLimit != 50 || todoRepo.lastOffset != 0 {
		t.Fatalf("expected default pagination limit=50 offset=0, got limit=%d offset=%d", todoRepo.lastLimit, todoRepo.lastOffset)
	}
}

func TestCreateReminderReturnsCreatedReminder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/reminders", strings.NewReader(`{"title":"Follow up","message":"Ping","remind_at":"2026-04-20T10:00:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response struct {
		Data domain.Reminder `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse reminder response: %v", err)
	}

	if response.Data.Title != "Follow up" {
		t.Fatalf("expected reminder title Follow up, got %q", response.Data.Title)
	}
	if response.Data.Status != domain.ReminderStatusScheduled {
		t.Fatalf("expected reminder status scheduled, got %q", response.Data.Status)
	}
	if len(reminderRepo.items) != 1 {
		t.Fatalf("expected reminder repo to store one item, got %d", len(reminderRepo.items))
	}
}

func TestListRemindersReturnsReminderDataWithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{items: []domain.Reminder{{ID: "rem-1", Title: "Follow up", Status: domain.ReminderStatusScheduled}}}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reminders?limit=3&offset=1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if reminderRepo.lastLimit != 3 || reminderRepo.lastOffset != 1 {
		t.Fatalf("expected pagination limit=3 offset=1, got limit=%d offset=%d", reminderRepo.lastLimit, reminderRepo.lastOffset)
	}

	var response struct {
		Data []domain.Reminder `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse reminders response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 reminder item, got %d", len(response.Data))
	}
}

func TestListRemindersDefaultsPaginationWhenInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{items: []domain.Reminder{{ID: "rem-1", Title: "Follow up", Status: domain.ReminderStatusScheduled}}}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reminders?limit=0&offset=-9", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if reminderRepo.lastLimit != 50 || reminderRepo.lastOffset != 0 {
		t.Fatalf("expected default pagination limit=50 offset=0, got limit=%d offset=%d", reminderRepo.lastLimit, reminderRepo.lastOffset)
	}
}

func TestCreateCategoryReturnsBadRequestOnValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "category name is required")
}

func TestCreateCategoryReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{createErr: errors.New("category write failure")}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"AI"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "category write failure")
}

func TestListCategoriesReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{listErr: errors.New("category list failure")}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "category list failure")
}

func TestCreateTodoReturnsBadRequestOnValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/todos", strings.NewReader(`{"title":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "todo title is required")
}

func TestCreateTodoReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{createErr: errors.New("todo write failure")}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/todos", strings.NewReader(`{"title":"Task"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "todo write failure")
}

func TestListTodosReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{listErr: errors.New("todo list failure")}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "todo list failure")
}

func TestCreateReminderReturnsBadRequestOnValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/reminders", strings.NewReader(`{"title":"Follow up","remind_at":"not-a-time"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "remind_at must be RFC3339")
}

func TestCreateReminderReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{createErr: errors.New("reminder write failure")}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/reminders", strings.NewReader(`{"title":"Follow up","remind_at":"2026-04-20T10:00:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "reminder write failure")
}

func TestListRemindersReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{listErr: errors.New("reminder list failure")}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reminders", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "reminder list failure")
}

func TestCreateResourceReturnsCreatedResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{}
	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{"url":"https://example.com/article","title":"AI Article","category_id":"cat-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	if len(resourceRepo.created) != 1 {
		t.Fatalf("expected one created resource, got %d", len(resourceRepo.created))
	}

	var response struct {
		Data domain.Resource `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse resource response: %v", err)
	}

	if response.Data.URL != "https://example.com/article" {
		t.Fatalf("expected normalized URL, got %q", response.Data.URL)
	}
	if response.Data.CategoryID != "cat-1" {
		t.Fatalf("expected category cat-1, got %q", response.Data.CategoryID)
	}
}

func TestCreateResourceReturnsBadRequestOnValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{"url":"not-a-url"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "invalid url")
}

func TestCreateResourceReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{createErr: errors.New("resource write failure")}
	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{"url":"https://example.com","category_id":"cat-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "resource write failure")
}

func TestDeepProcessingHealthReturnsServiceUnavailableWhenNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/processing/deep/health", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "deep processing is not configured")
}

func TestDeepProcessingMetricsReturnsServiceUnavailableWhenNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/processing/deep/metrics", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "deep processing is not configured")
}

func TestReprocessDeepResourceReturnsServiceUnavailableWhenNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/processing/deep/reprocess/res-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "deep processing is not configured")
}

func TestReprocessDeepResourceReturnsServiceUnavailableWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	deepProcessor := service.NewDeepProcessor(nil, nil, nil, nil, service.DeepProcessingSettings{Enabled: false})
	handler := NewHandlerWithOptions(nil, nil, nil, nil, nil, nil, WithDeepProcessor(deepProcessor))
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/processing/deep/reprocess/res-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "deep processing is disabled")
}

func TestReprocessDeepResourceReturnsRateLimitedWhenQueueFull(t *testing.T) {
	gin.SetMode(gin.TestMode)

	deepProcessor := service.NewDeepProcessor(nil, nil, nil, nil, service.DeepProcessingSettings{Enabled: true, QueueCapacity: 1, WorkerCount: 1})
	if err := deepProcessor.Reprocess(context.Background(), "res-1"); err != nil {
		t.Fatalf("expected initial reprocess to succeed, got %v", err)
	}

	handler := NewHandlerWithOptions(nil, nil, nil, nil, nil, nil, WithDeepProcessor(deepProcessor))
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/processing/deep/reprocess/res-2", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusTooManyRequests, "rate_limited", "deep processing queue is full")
}

func TestReprocessDeepResourceReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	deepProcessor := service.NewDeepProcessor(nil, nil, nil, nil, service.DeepProcessingSettings{Enabled: true, QueueCapacity: 2, WorkerCount: 1})
	handler := NewHandlerWithOptions(nil, nil, nil, nil, nil, nil, WithDeepProcessor(deepProcessor))
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/processing/deep/reprocess/res-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, recorder.Code)
	}

	var response struct {
		Message    string `json:"message"`
		ResourceID string `json:"resource_id"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response.Message != "deep reprocess queued" {
		t.Fatalf("expected message deep reprocess queued, got %q", response.Message)
	}
	if response.ResourceID != "res-1" {
		t.Fatalf("expected resource_id res-1, got %q", response.ResourceID)
	}
}

func TestListResourcesReturnsResourceDataWithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{ID: "res-1", URL: "https://example.com/1"}, {ID: "res-2", URL: "https://example.com/2"}}}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources?limit=4&offset=2", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if resourceRepo.lastLimit != 4 || resourceRepo.lastOffset != 2 {
		t.Fatalf("expected pagination limit=4 offset=2, got limit=%d offset=%d", resourceRepo.lastLimit, resourceRepo.lastOffset)
	}

	var response struct {
		Data []domain.Resource `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse resources response: %v", err)
	}

	if len(response.Data) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(response.Data))
	}
}

func TestListResourcesDefaultsPaginationWhenInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{ID: "res-1", URL: "https://example.com/1"}}}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources?limit=0&offset=-4", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if resourceRepo.lastLimit != 50 || resourceRepo.lastOffset != 0 {
		t.Fatalf("expected default pagination limit=50 offset=0, got limit=%d offset=%d", resourceRepo.lastLimit, resourceRepo.lastOffset)
	}
}

func TestListResourcesReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{listErr: errors.New("resource list failure")}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "resource list failure")
}

func TestUpdateResourceCategoryReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{}
	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/resources/res-1/category", strings.NewReader(`{"category_id":"cat-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if resourceRepo.lastUpdateID != "res-1" || resourceRepo.lastUpdateCatID != "cat-1" || !resourceRepo.lastUpdateOver {
		t.Fatalf("expected update args id=res-1 category=cat-1 override=true, got id=%q category=%q override=%t", resourceRepo.lastUpdateID, resourceRepo.lastUpdateCatID, resourceRepo.lastUpdateOver)
	}
}

func TestUpdateResourceCategoryReturnsBadRequestWhenCategoryMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/resources/res-1/category", strings.NewReader(`{"category_id":"missing"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "category not found")
}

func TestUpdateResourceCategoryReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{updateErr: errors.New("resource update failure")}
	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/resources/res-1/category", strings.NewReader(`{"category_id":"cat-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "resource update failure")
}

func TestErrorEnvelopeIncludesCodeForRepresentativeFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{}
	todoRepo := &todoRepoCRUDStub{}
	reminderRepo := &reminderRepoCRUDStub{}
	resourceRepo := &graphResourceRepoStub{listErr: errors.New("semantic list failure")}

	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	categorySvc := service.NewCategoryService(categoryRepo)
	todoSvc := service.NewTodoService(todoRepo)
	reminderSvc := service.NewReminderService(reminderRepo)

	configuredHandler := NewHandler(resourceSvc, categorySvc, todoSvc, reminderSvc, nil, service.NewChatService(nil, nil, nil, nil, nil))
	configuredRouter := gin.New()
	configuredHandler.RegisterRoutes(configuredRouter)

	unavailableHandler := NewHandler(nil, nil, nil, nil, nil, nil)
	unavailableRouter := gin.New()
	unavailableHandler.RegisterRoutes(unavailableRouter)

	type testCase struct {
		name      string
		router    *gin.Engine
		method    string
		path      string
		body      string
		status    int
		errorCode string
	}

	tests := []testCase{
		{name: "service unavailable envelope", router: unavailableRouter, method: http.MethodGet, path: "/api/v1/graph", status: http.StatusServiceUnavailable, errorCode: "service_unavailable"},
		{name: "invalid payload envelope", router: configuredRouter, method: http.MethodPost, path: "/api/v1/chat/commands", body: "{", status: http.StatusBadRequest, errorCode: "invalid_payload"},
		{name: "validation envelope", router: configuredRouter, method: http.MethodGet, path: "/api/v1/resources/search", status: http.StatusBadRequest, errorCode: "validation_error"},
		{name: "internal envelope", router: configuredRouter, method: http.MethodGet, path: "/api/v1/resources/semantic-search?q=graph", status: http.StatusInternalServerError, errorCode: "internal_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader *strings.Reader
			if tc.body == "" {
				bodyReader = strings.NewReader("")
			} else {
				bodyReader = strings.NewReader(tc.body)
			}
			request := httptest.NewRequest(tc.method, tc.path, bodyReader)
			if tc.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			recorder := httptest.NewRecorder()

			tc.router.ServeHTTP(recorder, request)

			if recorder.Code != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, recorder.Code)
			}

			var response map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}

			if strings.TrimSpace(response["code"]) == "" {
				t.Fatalf("expected non-empty code field")
			}
			if response["code"] != tc.errorCode {
				t.Fatalf("expected code %q, got %q", tc.errorCode, response["code"])
			}
			if strings.TrimSpace(response["error"]) == "" {
				t.Fatalf("expected non-empty error field")
			}
		})
	}
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedCode, expectedError string) {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if expectedCode == "internal_error" {
		if response["error"] != "internal server error" {
			t.Fatalf("expected generic internal error, got %q", response["error"])
		}
	} else if response["error"] != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, response["error"])
	}

	if response["code"] != expectedCode {
		t.Fatalf("expected code %q, got %q", expectedCode, response["code"])
	}
}

func TestSemanticSearchResourcesReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := graphCategoryRepoStub{}
	resourceRepo := &graphResourceRepoStub{listErr: errors.New("semantic list failure")}

	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/semantic-search?q=graph", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if response["error"] != "internal server error" {
		t.Fatalf("expected generic internal error, got %q", response["error"])
	}

	if response["code"] != "internal_error" {
		t.Fatalf("expected code internal_error, got %q", response["code"])
	}
}

func TestPaginationClampsLimitToMax(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/resources?limit=10000000&offset=3", nil)

	limit, offset := pagination(c)
	if limit != 200 {
		t.Fatalf("expected limit 200, got %d", limit)
	}
	if offset != 3 {
		t.Fatalf("expected offset 3, got %d", offset)
	}
}

func TestPaginationUsesDefaultWhenLimitInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/resources?limit=0&offset=-5", nil)

	limit, offset := pagination(c)
	if limit != 50 {
		t.Fatalf("expected default limit 50, got %d", limit)
	}
	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}
}

func TestGetCategoryByIDReturnsCategoryData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI", Description: "Artificial Intelligence"}}}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories/cat-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Data domain.Category `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse category response: %v", err)
	}

	if response.Data.ID != "cat-1" {
		t.Fatalf("expected category id cat-1, got %q", response.Data.ID)
	}
}

func TestGetCategoryByIDReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categorySvc := service.NewCategoryService(&categoryRepoCRUDStub{})
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "category not found")
}

func TestGetCategoryByIDReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{getByIDErr: errors.New("category lookup failure")}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories/cat-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "category lookup failure")
}

func TestUpdateCategoryReturnsUpdatedCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI", Description: "Old"}}}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/categories/cat-1", strings.NewReader(`{"name":"productivity systems","description":"Updated"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Data domain.Category `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse category response: %v", err)
	}

	if response.Data.Name != "Productivity Systems" {
		t.Fatalf("expected normalized category name Productivity Systems, got %q", response.Data.Name)
	}
	if categoryRepo.items[0].Description != "Updated" {
		t.Fatalf("expected category description to be updated")
	}
}

func TestUpdateCategoryReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categorySvc := service.NewCategoryService(&categoryRepoCRUDStub{})
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/categories/missing", strings.NewReader(`{"name":"AI"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "category not found")
}

func TestUpdateCategoryReturnsBadRequestOnValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/categories/cat-1", strings.NewReader(`{"name":"   "}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "category name is required")
}

func TestDeleteCategoryReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/cat-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(categoryRepo.items) != 0 {
		t.Fatalf("expected category to be removed from repository")
	}
}

func TestDeleteCategoryReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categorySvc := service.NewCategoryService(&categoryRepoCRUDStub{})
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "category not found")
}

func TestDeleteCategoryReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}, deleteErr: errors.New("category delete failure")}
	categorySvc := service.NewCategoryService(categoryRepo)
	handler := NewHandler(nil, categorySvc, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/cat-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "category delete failure")
}

func TestGetResourceByIDReturnsResourceData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{
		ID:           "res-1",
		URL:          "https://example.com/ai",
		Host:         "example.com",
		Title:        "AI",
		CategoryID:   "cat-1",
		CategoryName: "AI",
	}}}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/res-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Data domain.Resource `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse resource response: %v", err)
	}

	if response.Data.ID != "res-1" {
		t.Fatalf("expected resource id res-1, got %q", response.Data.ID)
	}
}

func TestGetResourceByIDReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "resource not found")
}

func TestGetResourceByIDReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{getByIDErr: errors.New("resource lookup failure")}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/resources/res-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "resource lookup failure")
}

func TestUpdateResourceReturnsUpdatedResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{
		ID:           "res-1",
		URL:          "https://example.com/old",
		Host:         "example.com",
		Title:        "Old Title",
		Summary:      "Old Summary",
		CategoryID:   "cat-1",
		CategoryName: "AI",
	}}}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/resources/res-1", strings.NewReader(`{"url":"https://example.com/new","title":"New Title","summary":"New Summary"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Data domain.Resource `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse resource response: %v", err)
	}

	if response.Data.URL != "https://example.com/new" {
		t.Fatalf("expected updated resource url, got %q", response.Data.URL)
	}
	if response.Data.Title != "New Title" {
		t.Fatalf("expected updated title New Title, got %q", response.Data.Title)
	}
}

func TestUpdateResourceReturnsBadRequestOnValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{ID: "res-1", URL: "https://example.com/old", Host: "example.com", CategoryID: "cat-1", CategoryName: "AI"}}}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/resources/res-1", strings.NewReader(`{"url":"not-a-url"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "invalid url")
}

func TestUpdateResourceReturnsBadRequestOnInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/resources/res-1", strings.NewReader("{"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "invalid_payload", "invalid payload")
}

func TestUpdateResourceReturnsServiceUnavailableWhenResourceServiceNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/resources/res-1", strings.NewReader(`{"title":"Updated"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "service_unavailable", "resource service is not configured")
}

func TestUpdateResourceReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/resources/missing", strings.NewReader(`{"title":"Updated"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "resource not found")
}

func TestDeleteResourceReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{ID: "res-1", URL: "https://example.com", CategoryID: "cat-1", CategoryName: "AI"}}}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/resources/res-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(resourceRepo.items) != 0 {
		t.Fatalf("expected resource to be deleted")
	}
}

func TestDeleteResourceReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceSvc := service.NewResourceService(&graphResourceRepoStub{}, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/resources/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "resource not found")
}

func TestDeleteResourceReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resourceRepo := &graphResourceRepoStub{items: []domain.Resource{{ID: "res-1", URL: "https://example.com", CategoryID: "cat-1", CategoryName: "AI"}}, deleteErr: errors.New("resource delete failure")}
	resourceSvc := service.NewResourceService(resourceRepo, &categoryRepoCRUDStub{}, nil, nil)
	handler := NewHandler(resourceSvc, nil, nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/resources/res-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "resource delete failure")
}

func TestGetTodoByIDReturnsTodoData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Review", Status: domain.TodoStatusOpen}}}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/todos/todo-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestGetTodoByIDReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoSvc := service.NewTodoService(&todoRepoCRUDStub{})
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/todos/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "todo not found")
}

func TestGetTodoByIDReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{getByIDErr: errors.New("todo lookup failure")}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/todos/todo-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "todo lookup failure")
}

func TestUpdateTodoReturnsUpdatedTodo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Review", Status: domain.TodoStatusOpen}}}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/todos/todo-1", strings.NewReader(`{"title":"Updated Task","details":"Done","status":"in_progress","due_at":"2026-04-20T10:00:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if todoRepo.items[0].Title != "Updated Task" {
		t.Fatalf("expected todo title Updated Task, got %q", todoRepo.items[0].Title)
	}
	if todoRepo.items[0].Status != domain.TodoStatusInProgress {
		t.Fatalf("expected todo status in_progress, got %q", todoRepo.items[0].Status)
	}
}

func TestUpdateTodoReturnsBadRequestOnInvalidDueAt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoSvc := service.NewTodoService(&todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Task", Status: domain.TodoStatusOpen}}})
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/todos/todo-1", strings.NewReader(`{"title":"Task","status":"open","due_at":"bad-time"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "due_at must be RFC3339")
}

func TestUpdateTodoReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoSvc := service.NewTodoService(&todoRepoCRUDStub{})
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/todos/missing", strings.NewReader(`{"title":"Task","status":"open"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "todo not found")
}

func TestDeleteTodoReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Task", Status: domain.TodoStatusOpen}}}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/todo-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(todoRepo.items) != 0 {
		t.Fatalf("expected todo to be deleted")
	}
}

func TestDeleteTodoReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoSvc := service.NewTodoService(&todoRepoCRUDStub{})
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "todo not found")
}

func TestDeleteTodoReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	todoRepo := &todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Task", Status: domain.TodoStatusOpen}}, deleteErr: errors.New("todo delete failure")}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandler(nil, nil, todoSvc, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/todo-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "todo delete failure")
}

func TestGetReminderByIDReturnsReminderData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{items: []domain.Reminder{{ID: "rem-1", Title: "Follow up", RemindAt: time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC), Status: domain.ReminderStatusScheduled}}}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reminders/rem-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestGetReminderByIDReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderSvc := service.NewReminderService(&reminderRepoCRUDStub{})
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reminders/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "reminder not found")
}

func TestGetReminderByIDReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{getByIDErr: errors.New("reminder lookup failure")}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/reminders/rem-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "reminder lookup failure")
}

func TestUpdateReminderReturnsUpdatedReminder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{items: []domain.Reminder{{ID: "rem-1", Title: "Follow up", RemindAt: time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC), Status: domain.ReminderStatusScheduled}}}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/reminders/rem-1", strings.NewReader(`{"title":"Follow up updated","message":"Ping","remind_at":"2026-04-22T08:30:00Z","status":"sent"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if reminderRepo.items[0].Title != "Follow up updated" {
		t.Fatalf("expected updated reminder title, got %q", reminderRepo.items[0].Title)
	}
	if reminderRepo.items[0].Status != domain.ReminderStatusSent {
		t.Fatalf("expected updated reminder status sent, got %q", reminderRepo.items[0].Status)
	}
}

func TestUpdateReminderReturnsBadRequestOnInvalidTime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderSvc := service.NewReminderService(&reminderRepoCRUDStub{items: []domain.Reminder{{ID: "rem-1", Title: "Follow up", RemindAt: time.Now().UTC(), Status: domain.ReminderStatusScheduled}}})
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/reminders/rem-1", strings.NewReader(`{"title":"Reminder","remind_at":"bad-time"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "validation_error", "remind_at must be RFC3339")
}

func TestUpdateReminderReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderSvc := service.NewReminderService(&reminderRepoCRUDStub{})
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/reminders/missing", strings.NewReader(`{"title":"Reminder","remind_at":"2026-04-22T08:30:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "reminder not found")
}

func TestDeleteReminderReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{items: []domain.Reminder{{ID: "rem-1", Title: "Follow up", RemindAt: time.Now().UTC(), Status: domain.ReminderStatusScheduled}}}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/reminders/rem-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if len(reminderRepo.items) != 0 {
		t.Fatalf("expected reminder to be deleted")
	}
}

func TestDeleteReminderReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderSvc := service.NewReminderService(&reminderRepoCRUDStub{})
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/reminders/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found", "reminder not found")
}

func TestDeleteReminderReturnsInternalServerErrorOnServiceFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reminderRepo := &reminderRepoCRUDStub{items: []domain.Reminder{{ID: "rem-1", Title: "Follow up", RemindAt: time.Now().UTC(), Status: domain.ReminderStatusScheduled}}, deleteErr: errors.New("reminder delete failure")}
	reminderSvc := service.NewReminderService(reminderRepo)
	handler := NewHandler(nil, nil, nil, reminderSvc, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/reminders/rem-1", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusInternalServerError, "internal_error", "reminder delete failure")
}

func TestCreateResourcePublishesSyncEventWhenHubConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := syncapi.NewHub()
	events, unsubscribe := hub.Subscribe(1)
	defer unsubscribe()

	resourceRepo := &graphResourceRepoStub{}
	categoryRepo := &categoryRepoCRUDStub{items: []domain.Category{{ID: "cat-1", Name: "AI"}}}
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, nil)
	handler := NewHandlerWithOptions(resourceSvc, nil, nil, nil, nil, nil, WithSyncHub(hub))
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/resources", strings.NewReader(`{"url":"https://example.com/article","title":"AI Article","category_id":"cat-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	event := readSyncEventFromHub(t, events)
	if event.Type != syncapi.EventTypeResourceCreated {
		t.Fatalf("expected sync event type %q, got %q", syncapi.EventTypeResourceCreated, event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if strings.TrimSpace(payload["entity_id"].(string)) == "" {
		t.Fatalf("expected payload.entity_id to be populated")
	}
	if payload[syncapi.PayloadKeyEventSource] != syncapi.EventSourceHTTPMutation {
		t.Fatalf("expected event source %q, got %v", syncapi.EventSourceHTTPMutation, payload[syncapi.PayloadKeyEventSource])
	}
	if payload[syncapi.PayloadKeyEventVersion] != syncapi.EventVersionCurrent {
		t.Fatalf("expected event version %d, got %v", syncapi.EventVersionCurrent, payload[syncapi.PayloadKeyEventVersion])
	}
}

func TestUpdateTodoPublishesSyncEventWhenHubConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := syncapi.NewHub()
	events, unsubscribe := hub.Subscribe(1)
	defer unsubscribe()

	todoRepo := &todoRepoCRUDStub{items: []domain.Todo{{ID: "todo-1", Title: "Task", Status: domain.TodoStatusOpen}}}
	todoSvc := service.NewTodoService(todoRepo)
	handler := NewHandlerWithOptions(nil, nil, todoSvc, nil, nil, nil, WithSyncHub(hub))
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/todos/todo-1", strings.NewReader(`{"title":"Task Updated","status":"done"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	event := readSyncEventFromHub(t, events)
	if event.Type != syncapi.EventTypeTodoUpdated {
		t.Fatalf("expected sync event type %q, got %q", syncapi.EventTypeTodoUpdated, event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload["entity_id"] != "todo-1" {
		t.Fatalf("expected payload.entity_id todo-1, got %v", payload["entity_id"])
	}
	if payload[syncapi.PayloadKeyEventSource] != syncapi.EventSourceHTTPMutation {
		t.Fatalf("expected event source %q, got %v", syncapi.EventSourceHTTPMutation, payload[syncapi.PayloadKeyEventSource])
	}
}

func TestExecuteChatCommandPublishesSyncEventForResourceCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := syncapi.NewHub()
	events, unsubscribe := hub.Subscribe(1)
	defer unsubscribe()

	resourceRepo := &graphResourceRepoStub{}
	categoryRepo := &categoryRepoCRUDStub{}
	categorySvc := service.NewCategoryService(categoryRepo)
	resourceSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, categorySvc)
	todoSvc := service.NewTodoService(&todoRepoCRUDStub{})
	reminderSvc := service.NewReminderService(&reminderRepoCRUDStub{})
	chatSvc := service.NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc, nil)

	handler := NewHandlerWithOptions(resourceSvc, categorySvc, todoSvc, reminderSvc, nil, chatSvc, WithSyncHub(hub))
	router := gin.New()
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/commands", strings.NewReader(`{"message":"resource: https://example.com/chat-item | category=AI"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	event := readSyncEventFromHub(t, events)
	if event.Type != syncapi.EventTypeResourceCreated {
		t.Fatalf("expected sync event type %q, got %q", syncapi.EventTypeResourceCreated, event.Type)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", event.Payload)
	}
	if payload[syncapi.PayloadKeyEventSource] != syncapi.EventSourceChatCommand {
		t.Fatalf("expected event source %q, got %v", syncapi.EventSourceChatCommand, payload[syncapi.PayloadKeyEventSource])
	}
}

func readSyncEventFromHub(t *testing.T, events <-chan syncapi.Event) syncapi.Event {
	t.Helper()

	select {
	case event := <-events:
		return event
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for sync event")
		return syncapi.Event{}
	}
}

func TestRegisterRoutesAppliesAuthMiddlewareWhenConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authMiddleware := func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer test-token" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "code": "unauthorized"})
			return
		}
		c.Next()
	}

	handler := NewHandlerWithOptions(
		service.NewResourceService(&graphResourceRepoStub{}, graphCategoryRepoStub{}, nil, nil),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithAuthMiddleware(authMiddleware),
	)
	router := gin.New()
	handler.RegisterRoutes(router)

	unauthorized := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	unauthorizedRecorder := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedRecorder, unauthorized)
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, unauthorizedRecorder.Code)
	}

	authorized := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	authorized.Header.Set("Authorization", "Bearer test-token")
	authorizedRecorder := httptest.NewRecorder()
	router.ServeHTTP(authorizedRecorder, authorized)
	if authorizedRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, authorizedRecorder.Code)
	}
}

func TestCreateCategoryRejectsOverlongName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(nil, service.NewCategoryService(&categoryRepoCRUDStub{}), nil, nil, nil, nil)
	router := gin.New()
	handler.RegisterRoutes(router)

	longName := strings.Repeat("a", maxCategoryNameLength+1)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"`+longName+`","description":"desc"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}
