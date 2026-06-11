package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/domain"
)

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
