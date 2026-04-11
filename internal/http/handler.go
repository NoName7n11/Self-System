package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
)

const (
	errorCodeInvalidPayload     = "invalid_payload"
	errorCodeValidation         = "validation_error"
	errorCodeInternal           = "internal_error"
	errorCodeServiceUnavailable = "service_unavailable"
)

type Handler struct {
	resources  *service.ResourceService
	categories *service.CategoryService
	todos      *service.TodoService
	reminders  *service.ReminderService
	graph      *service.GraphService
	chat       *service.ChatService
}

func NewHandler(
	resources *service.ResourceService,
	categories *service.CategoryService,
	todos *service.TodoService,
	reminders *service.ReminderService,
	graph *service.GraphService,
	chat *service.ChatService,
) *Handler {
	return &Handler{
		resources:  resources,
		categories: categories,
		todos:      todos,
		reminders:  reminders,
		graph:      graph,
		chat:       chat,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.health)

	api := router.Group("/api/v1")
	api.POST("/resources", h.createResource)
	api.GET("/resources", h.listResources)
	api.GET("/resources/search", h.searchResources)
	api.GET("/resources/semantic-search", h.semanticSearchResources)
	api.GET("/graph", h.getGraph)
	api.PATCH("/resources/:id/category", h.updateResourceCategory)

	api.POST("/categories", h.createCategory)
	api.GET("/categories", h.listCategories)

	api.POST("/todos", h.createTodo)
	api.GET("/todos", h.listTodos)

	api.POST("/reminders", h.createReminder)
	api.GET("/reminders", h.listReminders)

	api.POST("/chat/commands", h.executeChatCommand)
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

	resource, err := h.resources.Create(c.Request.Context(), service.CreateResourceInput{
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

	c.JSON(http.StatusCreated, gin.H{"data": resource})
}

func (h *Handler) listResources(c *gin.Context) {
	if h.resources == nil {
		respondError(c, http.StatusServiceUnavailable, "resource service is not configured")
		return
	}

	limit, offset := pagination(c)
	resources, err := h.resources.List(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resources})
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

	limit := parseBoundedInt(c.DefaultQuery("limit", "25"), 25, 1, 100)
	resources, err := h.resources.Search(c.Request.Context(), query, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
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
		respondError(c, http.StatusInternalServerError, err.Error())
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
		respondError(c, http.StatusInternalServerError, err.Error())
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

	if err := h.resources.UpdateCategory(c.Request.Context(), service.UpdateResourceCategoryInput{
		ResourceID: resourceID,
		CategoryID: req.CategoryID,
	}); err != nil {
		respondOperationError(c, err)
		return
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

	category, err := h.categories.Create(c.Request.Context(), service.CreateCategoryInput{
		Name:        req.Name,
		Description: req.Description,
		Source:      domain.CategorySourceManual,
	})
	if err != nil {
		respondOperationError(c, err)
		return
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
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
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
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": todos})
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
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reminders})
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

	if h.chat == nil {
		respondError(c, http.StatusServiceUnavailable, "chat service is not configured")
		return
	}

	result, err := h.chat.Execute(c.Request.Context(), req.Message)
	if err != nil {
		respondOperationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func pagination(c *gin.Context) (int, int) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 50
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

	respondError(c, http.StatusInternalServerError, message)
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
	case http.StatusServiceUnavailable:
		code = errorCodeServiceUnavailable
	}
	respondErrorCode(c, status, code, message)
}

func respondErrorCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": message, "code": code})
}
