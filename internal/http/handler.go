package http

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/ai"
	"selfsystems/internal/service"
	syncapi "selfsystems/internal/sync"
)

const (
	errorCodeInvalidPayload     = "invalid_payload"
	errorCodeValidation         = "validation_error"
	errorCodeNotFound           = "not_found"
	errorCodeInternal           = "internal_error"
	errorCodeServiceUnavailable = "service_unavailable"
	defaultPageLimit            = 50
	maxPageLimit                = 200
)

type Handler struct {
	resources      *service.ResourceService
	categories     *service.CategoryService
	todos          *service.TodoService
	reminders      *service.ReminderService
	graph          *service.GraphService
	chat           *service.ChatService
	syncHub        *syncapi.Hub
	deepProcessor  *service.DeepProcessor
	authMiddleware gin.HandlerFunc
	gbusMonitor    GBUSMonitor
	db             *sql.DB
	aiManager      *ai.Manager
}

// GBUSMonitor is the subset of gbus.Monitor used by the health endpoint.
type GBUSMonitor interface {
	SignalCount() int64
	LastCheckAt() time.Time
}

// GBUSInferenceInfo is implemented by gbus.Inference for status reporting.
type GBUSInferenceInfo interface {
	ModelVersion() string
	ModelStatus() string
}

type HandlerOption func(*Handler)

func WithSyncHub(hub *syncapi.Hub) HandlerOption {
	return func(h *Handler) {
		h.syncHub = hub
	}
}

func WithDeepProcessor(processor *service.DeepProcessor) HandlerOption {
	return func(h *Handler) {
		h.deepProcessor = processor
	}
}

func WithGBUSMonitor(m GBUSMonitor) HandlerOption {
	return func(h *Handler) {
		h.gbusMonitor = m
	}
}

func WithAuthMiddleware(middleware gin.HandlerFunc) HandlerOption {
	return func(h *Handler) {
		h.authMiddleware = middleware
	}
}

// WithDB wires the underlying *sql.DB so /api/v1/health can verify the
// database is writable (Change 12 WS4). nil is a valid value (e.g. Postgres
// repositories that don't expose a shared *sql.DB yet).
func WithDB(db *sql.DB) HandlerOption {
	return func(h *Handler) {
		h.db = db
	}
}

// WithAIManager wires the AI manager so /api/v1/health can report which AI
// providers are registered (Change 12 WS4).
func WithAIManager(manager *ai.Manager) HandlerOption {
	return func(h *Handler) {
		h.aiManager = manager
	}
}

func NewHandler(
	resources *service.ResourceService,
	categories *service.CategoryService,
	todos *service.TodoService,
	reminders *service.ReminderService,
	graph *service.GraphService,
	chat *service.ChatService,
) *Handler {
	return NewHandlerWithOptions(resources, categories, todos, reminders, graph, chat)
}

func NewHandlerWithOptions(
	resources *service.ResourceService,
	categories *service.CategoryService,
	todos *service.TodoService,
	reminders *service.ReminderService,
	graph *service.GraphService,
	chat *service.ChatService,
	options ...HandlerOption,
) *Handler {
	handler := &Handler{
		resources:  resources,
		categories: categories,
		todos:      todos,
		reminders:  reminders,
		graph:      graph,
		chat:       chat,
	}

	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}

	return handler
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// healthDetailed reports per-component status (Change 12 WS4): database
// writability, deep-processing queue liveness, and registered AI providers.
// Returns 200 with status "ok" when all configured components are healthy,
// or 503 with status "degraded" when any configured component is failing.
func (h *Handler) healthDetailed(c *gin.Context) {
	overall := "ok"

	database := gin.H{"status": "not_configured"}
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := h.db.PingContext(ctx); err != nil {
			database["status"] = "error"
			database["error"] = err.Error()
			overall = "degraded"
		} else {
			database["status"] = "ok"
		}
	}

	deepQueue := gin.H{"status": "disabled"}
	if h.deepProcessor != nil {
		dpHealth := h.deepProcessor.Health()
		deepQueue["status"] = dpHealth.Status
		deepQueue["backlog"] = dpHealth.Backlog
		deepQueue["queue_capacity"] = dpHealth.QueueCapacity
	}

	aiStatus := gin.H{"status": "not_configured"}
	if h.aiManager != nil {
		providers := h.aiManager.ProviderNames()
		aiStatus["status"] = "ok"
		aiStatus["providers"] = providers
		if len(providers) == 0 {
			aiStatus["status"] = "degraded"
			overall = "degraded"
		}
	}

	status := http.StatusOK
	if overall != "ok" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": overall,
		"components": gin.H{
			"database":   database,
			"deep_queue": deepQueue,
			"ai":         aiStatus,
		},
	})
}
