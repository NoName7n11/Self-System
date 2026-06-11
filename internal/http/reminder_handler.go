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
