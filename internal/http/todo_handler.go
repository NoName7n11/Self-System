package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

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
