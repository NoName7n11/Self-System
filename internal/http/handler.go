package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

const (
	errorCodeInvalidPayload     = "invalid_payload"
	errorCodeValidation         = "validation_error"
	errorCodeNotFound           = "not_found"
	errorCodeInternal           = "internal_error"
	errorCodeServiceUnavailable = "service_unavailable"
	defaultPageLimit            = 50
	maxPageLimit                = 200
)

type Handler struct {
	resources      *service.ResourceService
	categories     *service.CategoryService
	todos          *service.TodoService
	reminders      *service.ReminderService
	graph          *service.GraphService
	chat           *service.ChatService
	syncHub        *syncapi.Hub
	deepProcessor  *service.DeepProcessor
	authMiddleware gin.HandlerFunc
	gbusMonitor    GBUSMonitor
}

// GBUSMonitor is the subset of gbus.Monitor used by the health endpoint.
type GBUSMonitor interface {
	SignalCount() int64
	LastCheckAt() time.Time
}

// GBUSInferenceInfo is implemented by gbus.Inference for status reporting.
type GBUSInferenceInfo interface {
	ModelVersion() string
	ModelStatus() string
}

type HandlerOption func(*Handler)

func WithSyncHub(hub *syncapi.Hub) HandlerOption {
	return func(h *Handler) {
		h.syncHub = hub
	}
}

func WithDeepProcessor(processor *service.DeepProcessor) HandlerOption {
	return func(h *Handler) {
		h.deepProcessor = processor
	}
}

func WithGBUSMonitor(m GBUSMonitor) HandlerOption {
	return func(h *Handler) {
		h.gbusMonitor = m
	}
}

func WithAuthMiddleware(middleware gin.HandlerFunc) HandlerOption {
	return func(h *Handler) {
		h.authMiddleware = middleware
	}
}

func NewHandler(
	resources *service.ResourceService,
	categories *service.CategoryService,
	todos *service.TodoService,
	reminders *service.ReminderService,
	graph *service.GraphService,
	chat *service.ChatService,
) *Handler {
	return NewHandlerWithOptions(resources, categories, todos, reminders, graph, chat)
}

func NewHandlerWithOptions(
	resources *service.ResourceService,
	categories *service.CategoryService,
	todos *service.TodoService,
	reminders *service.ReminderService,
	graph *service.GraphService,
	chat *service.ChatService,
	options ...HandlerOption,
) *Handler {
	handler := &Handler{
		resources:  resources,
		categories: categories,
		todos:      todos,
		reminders:  reminders,
		graph:      graph,
		chat:       chat,
	}

	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}

	return handler
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.health)

	api := router.Group("/api/v1")
	if h.authMiddleware != nil {
		api.Use(h.authMiddleware)
	}
	api.Use(MaxBodyBytesMiddleware(defaultMaxRequestBodyBytes))
	mutationLimiter := MutationRateLimitMiddleware(120, time.Minute)
	api.POST("/resources", mutationLimiter, h.createResource)
	api.GET("/resources", h.listResources)
	api.GET("/resources/search", h.searchResources)
	api.GET("/resources/semantic-search", h.semanticSearchResources)
	api.GET("/resources/:id", h.getResourceByID)
	api.PUT("/resources/:id", mutationLimiter, h.updateResource)
	api.DELETE("/resources/:id", mutationLimiter, h.deleteResource)
	api.GET("/graph", h.getGraph)
	api.PATCH("/resources/:id/category", mutationLimiter, h.updateResourceCategory)
	api.POST("/resources/:id/archive", mutationLimiter, h.archiveResource)
	api.POST("/resources/:id/restore", mutationLimiter, h.restoreResource)
	api.POST("/resources/bulk-archive", mutationLimiter, h.bulkArchiveResources)
	api.POST("/resources/bulk-restore", mutationLimiter, h.bulkRestoreResources)

	api.POST("/categories", mutationLimiter, h.createCategory)
	api.GET("/categories", h.listCategories)
	api.GET("/categories/:id", h.getCategoryByID)
	api.PUT("/categories/:id", mutationLimiter, h.updateCategory)
	api.DELETE("/categories/:id", mutationLimiter, h.deleteCategory)

	api.POST("/todos", mutationLimiter, h.createTodo)
	api.GET("/todos", h.listTodos)
	api.GET("/todos/:id", h.getTodoByID)
	api.PUT("/todos/:id", mutationLimiter, h.updateTodo)
	api.DELETE("/todos/:id", mutationLimiter, h.deleteTodo)

	api.POST("/reminders", mutationLimiter, h.createReminder)
	api.GET("/reminders", h.listReminders)
	api.GET("/reminders/:id", h.getReminderByID)
	api.PUT("/reminders/:id", mutationLimiter, h.updateReminder)
	api.DELETE("/reminders/:id", mutationLimiter, h.deleteReminder)

	api.GET("/processing/deep/health", h.deepProcessingHealth)
	api.GET("/gbus/health", h.gbusHealth)
	api.GET("/processing/deep/metrics", h.deepProcessingMetrics)
	api.POST("/processing/deep/reprocess/:id", mutationLimiter, h.reprocessDeepResource)

	api.POST("/chat/commands", mutationLimiter, h.executeChatCommand)
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type createResourceRequest struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
}

func (h *Handler) createResource(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	var req createResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("url", req.URL, maxURLLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("title", req.Title, maxTitleLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("summary", req.Summary, maxSummaryLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("category_id", req.CategoryID, maxIDLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("category_name", req.CategoryName, maxCategoryNameLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	resource, err := h.resources.Create(c.Request.Context(), service.CreateResourceInput{
		URL:          req.URL,
		Title:        req.Title,
		Summary:      req.Summary,
		CategoryID:   req.CategoryID,
		CategoryName: req.CategoryName,
	})
	if errors.Is(err, domain.ErrDuplicateResource) {
		// Return the existing resource with a duplicate flag instead of 201.
		c.JSON(http.StatusOK, gin.H{"data": resource, "duplicate": true})
		return
	}
	if err != nil {
		respondOperationError(c, err)
		return
	}

	// When event sourcing is active the outbox worker delivers the sync event;
	// skip the direct publish to avoid duplicate delivery.
	if !h.resources.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeResourceCreated, resource.ID, map[string]any{
			"url":         resource.URL,
			"category_id": resource.CategoryID,
		})
	}
	h.enqueueDeepProcessing(resource.ID)

	c.JSON(http.StatusCreated, gin.H{"data": resource})
}

func (h *Handler) gbusHealth(c *gin.Context) {
	resp := gin.H{
		"enabled":       h.gbusMonitor != nil,
		"signal_count":  int64(0),
		"last_check_at": nil,
	}
	if h.gbusMonitor != nil {
		resp["signal_count"] = h.gbusMonitor.SignalCount()
		if t := h.gbusMonitor.LastCheckAt(); !t.IsZero() {
			resp["last_check_at"] = t
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *Handler) deepProcessingHealth(c *gin.Context) {
	if h.deepProcessor == nil {
		respondError(c, http.StatusServiceUnavailable, "deep processing is not configured")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": h.deepProcessor.Health()})
}

func (h *Handler) deepProcessingMetrics(c *gin.Context) {
	if h.deepProcessor == nil {
		respondError(c, http.StatusServiceUnavailable, "deep processing is not configured")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": h.deepProcessor.Metrics()})
}

func (h *Handler) reprocessDeepResource(c *gin.Context) {
	if h.deepProcessor == nil {
		respondError(c, http.StatusServiceUnavailable, "deep processing is not configured")
		return
	}

	resourceID := strings.TrimSpace(c.Param("id"))
	if resourceID == "" {
		respondError(c, http.StatusBadRequest, "resource id is required")
		return
	}

	if err := h.deepProcessor.Reprocess(c.Request.Context(), resourceID); err != nil {
		switch {
		case errors.Is(err, service.ErrDeepProcessingDisabled):
			respondError(c, http.StatusServiceUnavailable, err.Error())
			return
		case errors.Is(err, service.ErrDeepProcessingQueueFull):
			respondErrorCode(c, http.StatusTooManyRequests, "rate_limited", err.Error())
			return
		default:
			respondOperationError(c, err)
			return
		}
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "deep reprocess queued", "resource_id": resourceID})
}

func (h *Handler) listResources(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	limit, offset := pagination(c)

	// ?archived=true returns the archive view; default returns non-archived.
	var resources []domain.Resource
	var err error
	if c.Query("archived") == "true" {
		resources, err = h.resources.ListArchived(c.Request.Context(), limit, offset)
	} else {
		resources, err = h.resources.List(c.Request.Context(), limit, offset)
	}
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resources})
}

func (h *Handler) getResourceByID(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	resource, err := h.resources.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if resource == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "resource not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resource})
}

type updateResourceRequest struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
}

func (h *Handler) updateResource(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	var req updateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("url", req.URL, maxURLLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("title", req.Title, maxTitleLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("summary", req.Summary, maxSummaryLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("category_id", req.CategoryID, maxIDLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("category_name", req.CategoryName, maxCategoryNameLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.resources.Update(c.Request.Context(), service.UpdateResourceInput{
		ID:           c.Param("id"),
		URL:          req.URL,
		Title:        req.Title,
		Summary:      req.Summary,
		CategoryID:   req.CategoryID,
		CategoryName: req.CategoryName,
	})
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if updated == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "resource not found")
		return
	}

	if !h.resources.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeResourceUpdated, updated.ID, map[string]any{
			"url":         updated.URL,
			"category_id": updated.CategoryID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) deleteResource(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	deleted, err := h.resources.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if !deleted {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "resource not found")
		return
	}

	if !h.resources.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeResourceDeleted, c.Param("id"), nil)
	}

	c.JSON(http.StatusOK, gin.H{"message": "resource deleted"})
}

func (h *Handler) searchResources(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		respondError(c, http.StatusBadRequest, "q is required")
		return
	}

	mode := strings.ToLower(strings.TrimSpace(c.DefaultQuery("mode", "keyword")))
	limit := parseBoundedInt(c.DefaultQuery("limit", "25"), 25, 1, 100)

	var resources []domain.Resource
	var err error

	switch mode {
	case "semantic":
		resources, err = h.resources.SemanticSearch(c.Request.Context(), query, limit)
	case "hybrid":
		resources, err = h.resources.HybridSearch(c.Request.Context(), query, limit)
	default: // "keyword" and any unknown mode
		resources, err = h.resources.Search(c.Request.Context(), query, limit)
	}

	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resources})
}

func (h *Handler) semanticSearchResources(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		respondError(c, http.StatusBadRequest, "q is required")
		return
	}

	limit := parseBoundedInt(c.DefaultQuery("limit", "10"), 10, 1, 100)
	resources, err := h.resources.SemanticSearch(c.Request.Context(), query, limit)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resources})
}

func (h *Handler) getGraph(c *gin.Context) {
	if h.graph == nil {
		respondError(c, http.StatusServiceUnavailable, "graph service is not configured")
		return
	}

	limit := parseBoundedInt(c.DefaultQuery("limit", "1000"), 1000, 1, 5000)
	graph, err := h.graph.Build(c.Request.Context(), limit)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": graph})
}

type updateResourceCategoryRequest struct {
	CategoryID string `json:"category_id"`
}

func (h *Handler) updateResourceCategory(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	resourceID := c.Param("id")
	var req updateResourceCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("category_id", req.CategoryID, maxIDLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.resources.UpdateCategory(c.Request.Context(), service.UpdateResourceCategoryInput{
		ResourceID: resourceID,
		CategoryID: req.CategoryID,
	}); err != nil {
		respondOperationError(c, err)
		return
	}

	if !h.resources.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeResourceUpdated, resourceID, map[string]any{"category_id": req.CategoryID})
	}

	c.JSON(http.StatusOK, gin.H{"message": "resource category updated"})
}

type createCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) createCategory(c *gin.Context) {
	if h.categories == nil {
		respondError(c, http.StatusServiceUnavailable, "category service is not configured")
		return
	}

	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("name", req.Name, maxCategoryNameLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("description", req.Description, maxDescriptionLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	category, err := h.categories.Create(c.Request.Context(), service.CreateCategoryInput{
		Name:        req.Name,
		Description: req.Description,
		Source:      domain.CategorySourceManual,
	})
	if err != nil {
		respondOperationError(c, err)
		return
	}

	if !h.categories.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeCategoryUpdated, category.ID, map[string]any{"name": category.Name})
	}

	c.JSON(http.StatusCreated, gin.H{"data": category})
}

func (h *Handler) listCategories(c *gin.Context) {
	if h.categories == nil {
		respondError(c, http.StatusServiceUnavailable, "category service is not configured")
		return
	}

	categories, err := h.categories.List(c.Request.Context())
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
}

func (h *Handler) getCategoryByID(c *gin.Context) {
	if h.categories == nil {
		respondError(c, http.StatusServiceUnavailable, "category service is not configured")
		return
	}

	category, err := h.categories.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if category == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "category not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": category})
}

type updateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *Handler) updateCategory(c *gin.Context) {
	if h.categories == nil {
		respondError(c, http.StatusServiceUnavailable, "category service is not configured")
		return
	}

	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("name", req.Name, maxCategoryNameLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("description", req.Description, maxDescriptionLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.categories.Update(c.Request.Context(), service.UpdateCategoryInput{
		ID:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if updated == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "category not found")
		return
	}

	if !h.categories.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeCategoryUpdated, updated.ID, map[string]any{"name": updated.Name})
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) deleteCategory(c *gin.Context) {
	if h.categories == nil {
		respondError(c, http.StatusServiceUnavailable, "category service is not configured")
		return
	}

	deleted, err := h.categories.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if !deleted {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "category not found")
		return
	}

	if !h.categories.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeCategoryUpdated, c.Param("id"), map[string]any{"deleted": true})
	}

	c.JSON(http.StatusOK, gin.H{"message": "category deleted"})
}

type createTodoRequest struct {
	Title      string `json:"title"`
	Details    string `json:"details"`
	DueAt      string `json:"due_at"`
	ResourceID string `json:"resource_id"`
}

func (h *Handler) createTodo(c *gin.Context) {
	if h.todos == nil {
		respondError(c, http.StatusServiceUnavailable, "todo service is not configured")
		return
	}

	var req createTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("title", req.Title, maxTitleLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("details", req.Details, maxDetailsLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("resource_id", req.ResourceID, maxIDLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var dueAt *time.Time
	if strings.TrimSpace(req.DueAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueAt)
		if err != nil {
			respondError(c, http.StatusBadRequest, "due_at must be RFC3339")
			return
		}
		dueAt = &parsed
	}

	var resourceID *string
	if strings.TrimSpace(req.ResourceID) != "" {
		temp := strings.TrimSpace(req.ResourceID)
		resourceID = &temp
	}

	todo, err := h.todos.Create(c.Request.Context(), service.CreateTodoInput{
		Title:      req.Title,
		Details:    req.Details,
		DueAt:      dueAt,
		ResourceID: resourceID,
	})
	if err != nil {
		respondOperationError(c, err)
		return
	}

	if !h.todos.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeTodoUpdated, todo.ID, map[string]any{"status": todo.Status})
	}

	c.JSON(http.StatusCreated, gin.H{"data": todo})
}

func (h *Handler) listTodos(c *gin.Context) {
	if h.todos == nil {
		respondError(c, http.StatusServiceUnavailable, "todo service is not configured")
		return
	}

	limit, offset := pagination(c)
	todos, err := h.todos.List(c.Request.Context(), limit, offset)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": todos})
}

func (h *Handler) getTodoByID(c *gin.Context) {
	if h.todos == nil {
		respondError(c, http.StatusServiceUnavailable, "todo service is not configured")
		return
	}

	todo, err := h.todos.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if todo == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "todo not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": todo})
}

type updateTodoRequest struct {
	Title      string `json:"title"`
	Details    string `json:"details"`
	Status     string `json:"status"`
	DueAt      string `json:"due_at"`
	ResourceID string `json:"resource_id"`
}

func (h *Handler) updateTodo(c *gin.Context) {
	if h.todos == nil {
		respondError(c, http.StatusServiceUnavailable, "todo service is not configured")
		return
	}

	var req updateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("title", req.Title, maxTitleLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("details", req.Details, maxDetailsLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("resource_id", req.ResourceID, maxIDLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var dueAt *time.Time
	if strings.TrimSpace(req.DueAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueAt)
		if err != nil {
			respondError(c, http.StatusBadRequest, "due_at must be RFC3339")
			return
		}
		dueAt = &parsed
	}

	var resourceID *string
	if strings.TrimSpace(req.ResourceID) != "" {
		temp := strings.TrimSpace(req.ResourceID)
		resourceID = &temp
	}

	updated, err := h.todos.Update(c.Request.Context(), service.UpdateTodoInput{
		ID:         c.Param("id"),
		Title:      req.Title,
		Details:    req.Details,
		Status:     domain.TodoStatus(strings.TrimSpace(req.Status)),
		DueAt:      dueAt,
		ResourceID: resourceID,
	})
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if updated == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "todo not found")
		return
	}

	if !h.todos.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeTodoUpdated, updated.ID, map[string]any{"status": updated.Status})
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) deleteTodo(c *gin.Context) {
	if h.todos == nil {
		respondError(c, http.StatusServiceUnavailable, "todo service is not configured")
		return
	}

	deleted, err := h.todos.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if !deleted {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "todo not found")
		return
	}

	if !h.todos.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeTodoUpdated, c.Param("id"), map[string]any{"deleted": true})
	}

	c.JSON(http.StatusOK, gin.H{"message": "todo deleted"})
}

type createReminderRequest struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	RemindAt   string `json:"remind_at"`
	ResourceID string `json:"resource_id"`
}

func (h *Handler) createReminder(c *gin.Context) {
	if h.reminders == nil {
		respondError(c, http.StatusServiceUnavailable, "reminder service is not configured")
		return
	}

	var req createReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("title", req.Title, maxTitleLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("message", req.Message, maxMessageLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("resource_id", req.ResourceID, maxIDLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	remindAt, err := time.Parse(time.RFC3339, req.RemindAt)
	if err != nil {
		respondError(c, http.StatusBadRequest, "remind_at must be RFC3339")
		return
	}

	var resourceID *string
	if strings.TrimSpace(req.ResourceID) != "" {
		temp := strings.TrimSpace(req.ResourceID)
		resourceID = &temp
	}

	reminder, err := h.reminders.Create(c.Request.Context(), service.CreateReminderInput{
		Title:      req.Title,
		Message:    req.Message,
		RemindAt:   remindAt,
		ResourceID: resourceID,
	})
	if err != nil {
		respondOperationError(c, err)
		return
	}

	if !h.reminders.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeReminderUpdated, reminder.ID, map[string]any{"status": reminder.Status})
	}

	c.JSON(http.StatusCreated, gin.H{"data": reminder})
}

func (h *Handler) listReminders(c *gin.Context) {
	if h.reminders == nil {
		respondError(c, http.StatusServiceUnavailable, "reminder service is not configured")
		return
	}

	limit, offset := pagination(c)
	reminders, err := h.reminders.List(c.Request.Context(), limit, offset)
	if err != nil {
		respondInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reminders})
}

func (h *Handler) getReminderByID(c *gin.Context) {
	if h.reminders == nil {
		respondError(c, http.StatusServiceUnavailable, "reminder service is not configured")
		return
	}

	reminder, err := h.reminders.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if reminder == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "reminder not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": reminder})
}

type updateReminderRequest struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	RemindAt   string `json:"remind_at"`
	Status     string `json:"status"`
	ResourceID string `json:"resource_id"`
}

func (h *Handler) updateReminder(c *gin.Context) {
	if h.reminders == nil {
		respondError(c, http.StatusServiceUnavailable, "reminder service is not configured")
		return
	}

	var req updateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("title", req.Title, maxTitleLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("message", req.Message, maxMessageLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateLength("resource_id", req.ResourceID, maxIDLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	remindAt, err := time.Parse(time.RFC3339, req.RemindAt)
	if err != nil {
		respondError(c, http.StatusBadRequest, "remind_at must be RFC3339")
		return
	}

	var resourceID *string
	if strings.TrimSpace(req.ResourceID) != "" {
		temp := strings.TrimSpace(req.ResourceID)
		resourceID = &temp
	}

	updated, err := h.reminders.Update(c.Request.Context(), service.UpdateReminderInput{
		ID:         c.Param("id"),
		Title:      req.Title,
		Message:    req.Message,
		RemindAt:   remindAt,
		Status:     domain.ReminderStatus(strings.TrimSpace(req.Status)),
		ResourceID: resourceID,
	})
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if updated == nil {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "reminder not found")
		return
	}

	if !h.reminders.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeReminderUpdated, updated.ID, map[string]any{"status": updated.Status})
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) deleteReminder(c *gin.Context) {
	if h.reminders == nil {
		respondError(c, http.StatusServiceUnavailable, "reminder service is not configured")
		return
	}

	deleted, err := h.reminders.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondOperationError(c, err)
		return
	}
	if !deleted {
		respondErrorCode(c, http.StatusNotFound, errorCodeNotFound, "reminder not found")
		return
	}

	if !h.reminders.EventsEnabled() {
		h.publishSyncEvent(syncapi.EventTypeReminderUpdated, c.Param("id"), map[string]any{"deleted": true})
	}

	c.JSON(http.StatusOK, gin.H{"message": "reminder deleted"})
}

type chatCommandRequest struct {
	Message string `json:"message"`
}

func (h *Handler) executeChatCommand(c *gin.Context) {
	var req chatCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if err := validateLength("message", req.Message, maxMessageLength); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if h.chat == nil {
		respondError(c, http.StatusServiceUnavailable, "chat service is not configured")
		return
	}

	result, err := h.chat.Execute(c.Request.Context(), req.Message)
	if err != nil {
		respondOperationError(c, err)
		return
	}

	h.publishSyncFromChatResult(result)

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) publishSyncEvent(eventType, entityID string, payload map[string]any) {
	h.publishSyncEventWithSource(eventType, entityID, syncapi.EventSourceHTTPMutation, payload)
}

func (h *Handler) enqueueDeepProcessing(resourceID string) {
	if h.deepProcessor == nil {
		return
	}

	_ = h.deepProcessor.Enqueue(service.DeepTask{ResourceID: resourceID})
}

func (h *Handler) publishSyncEventWithSource(eventType, entityID, source string, payload map[string]any) {
	if h.syncHub == nil || strings.TrimSpace(eventType) == "" {
		return
	}

	enrichedPayload := syncapi.BuildEventPayload(payload, entityID, source)
	h.syncHub.Publish(syncapi.NewEvent(eventType, enrichedPayload))
}

func (h *Handler) publishSyncFromChatResult(result service.ChatResult) {
	switch result.Action {
	case "resource_created":
		if result.Resource != nil {
			h.publishSyncEventWithSource(syncapi.EventTypeResourceCreated, result.Resource.ID, syncapi.EventSourceChatCommand, map[string]any{"url": result.Resource.URL, "category_id": result.Resource.CategoryID})
		}
	case "resource_updated":
		if result.Resource != nil {
			h.publishSyncEventWithSource(syncapi.EventTypeResourceUpdated, result.Resource.ID, syncapi.EventSourceChatCommand, map[string]any{"url": result.Resource.URL, "category_id": result.Resource.CategoryID})
		}
	case "resource_deleted":
		if result.Resource != nil {
			h.publishSyncEventWithSource(syncapi.EventTypeResourceDeleted, result.Resource.ID, syncapi.EventSourceChatCommand, map[string]any{"deleted": true})
		}
	case "category_created", "category_updated":
		if result.Category != nil {
			h.publishSyncEventWithSource(syncapi.EventTypeCategoryUpdated, result.Category.ID, syncapi.EventSourceChatCommand, map[string]any{"name": result.Category.Name})
		}
	case "todo_created", "todo_updated":
		if result.Todo != nil {
			h.publishSyncEventWithSource(syncapi.EventTypeTodoUpdated, result.Todo.ID, syncapi.EventSourceChatCommand, map[string]any{"status": result.Todo.Status})
		}
	case "reminder_created", "reminder_updated":
		if result.Reminder != nil {
			h.publishSyncEventWithSource(syncapi.EventTypeReminderUpdated, result.Reminder.ID, syncapi.EventSourceChatCommand, map[string]any{"status": result.Reminder.Status})
		}
	}
}

func pagination(c *gin.Context) (int, int) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultPageLimit)))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func parseBoundedInt(raw string, fallback, min, max int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	if parsed < min {
		return fallback
	}
	if parsed > max {
		return max
	}
	return parsed
}

func respondOperationError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	message := err.Error()
	if isValidationErrorMessage(message) {
		respondError(c, http.StatusBadRequest, message)
		return
	}

	respondInternalError(c, err)
}

func respondInternalError(c *gin.Context, err error) {
	slog.Error("operation failed", "error", err)
	respondError(c, http.StatusInternalServerError, "internal server error")
}

func isValidationErrorMessage(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	validationHints := []string{
		"required",
		"invalid",
		"not found",
		"must be",
		"empty",
	}

	for _, hint := range validationHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}

	return false
}

func respondError(c *gin.Context, status int, message string) {
	code := errorCodeInternal
	switch status {
	case http.StatusBadRequest:
		code = errorCodeValidation
	case http.StatusNotFound:
		code = errorCodeNotFound
	case http.StatusServiceUnavailable:
		code = errorCodeServiceUnavailable
	}
	respondErrorCode(c, status, code, message)
}

func respondErrorCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}

// ── Archive / Restore handlers (WS3) ────────────────────────────────────────

type archiveResourceRequest struct {
	Reason string `json:"reason"` // "manual" | "dead_link" | "expired"; defaults to "manual"
}

func (h *Handler) archiveResource(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "resource id is required")
		return
	}

	var req archiveResourceRequest
	_ = c.ShouldBindJSON(&req) // optional body
	reason := domain.ArchiveReason(strings.TrimSpace(req.Reason))
	if reason == "" {
		reason = domain.ArchiveReasonManual
	}

	if err := h.resources.Archive(c.Request.Context(), id, reason); err != nil {
		respondOperationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resource archived", "resource_id": id})
}

func (h *Handler) restoreResource(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondError(c, http.StatusBadRequest, "resource id is required")
		return
	}

	if err := h.resources.Restore(c.Request.Context(), id); err != nil {
		respondOperationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resource restored", "resource_id": id})
}

type bulkArchiveRequest struct {
	IDs    []string `json:"ids"`
	Reason string   `json:"reason"` // defaults to "manual"
}

func (h *Handler) bulkArchiveResources(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	var req bulkArchiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if len(req.IDs) == 0 {
		respondError(c, http.StatusBadRequest, "ids list is required")
		return
	}

	reason := domain.ArchiveReason(strings.TrimSpace(req.Reason))
	if reason == "" {
		reason = domain.ArchiveReasonManual
	}

	if err := h.resources.BulkArchive(c.Request.Context(), req.IDs, reason); err != nil {
		respondOperationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resources archived", "count": len(req.IDs)})
}

type bulkRestoreRequest struct {
	IDs []string `json:"ids"`
}

func (h *Handler) bulkRestoreResources(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	var req bulkRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondErrorCode(c, http.StatusBadRequest, errorCodeInvalidPayload, "invalid payload")
		return
	}
	if len(req.IDs) == 0 {
		respondError(c, http.StatusBadRequest, "ids list is required")
		return
	}

	if err := h.resources.BulkRestore(c.Request.Context(), req.IDs); err != nil {
		respondOperationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resources restored", "count": len(req.IDs)})
}
