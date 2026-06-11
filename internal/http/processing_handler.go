package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/service"
)

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
