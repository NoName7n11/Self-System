package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

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

func (h *Handler) enqueueDeepProcessing(resourceID string) {
	if h.deepProcessor == nil {
		return
	}

	_ = h.deepProcessor.Enqueue(service.DeepTask{ResourceID: resourceID})
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
