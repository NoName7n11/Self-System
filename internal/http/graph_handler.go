package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
