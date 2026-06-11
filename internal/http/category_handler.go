package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

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
