package http

import (
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.health)

	api := router.Group("/api/v1")
	api.GET("/health", h.healthDetailed)
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
