package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

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
