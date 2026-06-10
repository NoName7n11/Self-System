package sync

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/config"
	"selfsystems/internal/eventstore"
)

// EventsHealthSnapshot is the response body for GET /api/v1/sync/events/health.
type EventsHealthSnapshot struct {
	LatestStoreSequence   int64   `json:"latest_store_sequence"`
	LastPublishedSeq      int64   `json:"last_published_sequence"`
	OutboxLagSequences    int64   `json:"outbox_lag_sequences"`
	AppendsTotal          int64   `json:"appends_total"`
	OCCRetriesTotal       int64   `json:"occ_retries_total"`
	ProjectorApplyCount   int64   `json:"projector_apply_count"`
	ProjectorAvgLatencyMs float64 `json:"projector_avg_latency_ms"`
	RedactionsTotal       int64   `json:"redactions_total"`
}

const (
	defaultReplayRequestLimit = 100
	defaultConflictListLimit  = 50
	maxSyncListLimit          = 200
)

type bootstrapRouteOptions struct {
	replayManager *OfflineReplayManager
	eventStore    eventstore.Store               // optional; enables durable WS reconnect replay
	outboxWorker  *OutboxWorker                  // optional; exposes outbox lag in events_health
	eventObs      *eventstore.EventObservability // optional; exposes event metrics in events_health
}

type BootstrapRouteOption func(*bootstrapRouteOptions)

func WithOfflineReplayManager(manager *OfflineReplayManager) BootstrapRouteOption {
	return func(options *bootstrapRouteOptions) {
		if options == nil {
			return
		}
		options.replayManager = manager
	}
}

// WithEventStoreReplay wires the events table as the durable replay source for
// reconnecting WebSocket clients. When provided, since_sequence queries the
// events table rather than (only) the hub's in-memory history.
func WithEventStoreReplay(store eventstore.Store) BootstrapRouteOption {
	return func(options *bootstrapRouteOptions) {
		if options == nil {
			return
		}
		options.eventStore = store
	}
}

// WithOutboxWorker wires the OutboxWorker so its last-published sequence can
// be reported on the events_health endpoint.
func WithOutboxWorker(w *OutboxWorker) BootstrapRouteOption {
	return func(options *bootstrapRouteOptions) {
		if options != nil {
			options.outboxWorker = w
		}
	}
}

// WithEventObservability wires EventObservability counters into the
// events_health endpoint.
func WithEventObservability(obs *eventstore.EventObservability) BootstrapRouteOption {
	return func(options *bootstrapRouteOptions) {
		if options != nil {
			options.eventObs = obs
		}
	}
}

func RegisterBootstrapRoutes(router *gin.Engine, cfg config.Config, hub *Hub, authMiddleware gin.HandlerFunc, options ...BootstrapRouteOption) {
	if router == nil {
		return
	}
	if hub == nil {
		hub = NewHub()
	}
	if authMiddleware == nil {
		authMiddleware = func(c *gin.Context) { c.Next() }
	}

	logger := syncLogger()

	routeOptions := bootstrapRouteOptions{}
	for _, option := range options {
		if option != nil {
			option(&routeOptions)
		}
	}
	if routeOptions.replayManager == nil {
		routeOptions.replayManager = NewOfflineReplayManager(nil, nil, hub)
	}

	observability := NewObservability()
	api := router.Group("/api/v1")

	api.GET("/auth/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	if !cfg.Sync.Enabled {
		api.GET("/sync/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":       "ok",
				"sync_enabled": false,
			})
		})
		return
	}

	wsHandler := NewWSHandler(hub, cfg.Sync.AllowedOrigins, cfg.Sync.HeartbeatSeconds, cfg.Sync.MaxConnectionsPerClient, observability)
	wsHandler.SetEventStore(routeOptions.eventStore)
	router.GET(cfg.Sync.WebSocketPath, observability.StatusMiddleware(), authMiddleware, wsHandler.ServeHTTP)

	api.GET("/sync/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "ok",
			"sync_enabled": true,
		})
	})

	authorizedSync := api.Group("/sync")
	authorizedSync.Use(observability.StatusMiddleware(), authMiddleware)
	authorizedSync.GET("/metrics", func(c *gin.Context) {
		queueSnapshot := ReplayQueueSnapshot{}
		if routeOptions.replayManager != nil {
			snapshot, queueErr := routeOptions.replayManager.QueueSnapshot(c.Request.Context())
			if queueErr != nil {
				logger.Warn("sync replay queue snapshot failed", "error", queueErr)
			} else {
				queueSnapshot = snapshot
			}
		}
		c.JSON(http.StatusOK, gin.H{"data": observability.SnapshotWithQueue(queueSnapshot)})
	})

	authorizedSync.GET("/events/health", func(c *gin.Context) {
		snap := EventsHealthSnapshot{}

		if routeOptions.eventStore != nil {
			latestSeq, seqErr := routeOptions.eventStore.LatestSequence(c.Request.Context())
			if seqErr != nil {
				logger.Warn("events_health latest sequence failed", "error", seqErr)
			} else {
				snap.LatestStoreSequence = latestSeq
			}
		}

		if routeOptions.outboxWorker != nil {
			snap.LastPublishedSeq = routeOptions.outboxWorker.LastSequence()
		}
		snap.OutboxLagSequences = snap.LatestStoreSequence - snap.LastPublishedSeq
		if snap.OutboxLagSequences < 0 {
			snap.OutboxLagSequences = 0
		}

		if routeOptions.eventObs != nil {
			evtSnap := routeOptions.eventObs.Snapshot()
			snap.AppendsTotal = evtSnap.AppendsTotal
			snap.OCCRetriesTotal = evtSnap.OCCRetriesTotal
			snap.ProjectorApplyCount = evtSnap.ProjectorApplyCount
			snap.ProjectorAvgLatencyMs = evtSnap.ProjectorAvgLatencyMs
			snap.RedactionsTotal = evtSnap.RedactionsTotal
		}

		c.JSON(http.StatusOK, gin.H{"data": snap})
	})

	authorizedSync.POST("/events", func(c *gin.Context) {
		var request struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			observability.RecordSyncEventRejected()
			logger.Warn("sync event rejected", "reason", "invalid_payload", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "code": "invalid_payload"})
			return
		}

		eventType := strings.ToLower(strings.TrimSpace(request.Type))
		if err := ValidateIncomingEvent(eventType, request.Payload); err != nil {
			observability.RecordSyncEventRejected()
			logger.Warn("sync event rejected", "event_type", eventType, "reason", "validation_error", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "validation_error"})
			return
		}

		request.Payload = SanitizeIncomingPayload(eventType, request.Payload)
		entityID := ExtractEntityID(request.Payload)
		request.Payload = BuildEventPayload(request.Payload, entityID, EventSourceSyncPublish)

		hub.Publish(NewEvent(eventType, request.Payload))
		observability.RecordSyncEventPublished()
		logger.Info("sync event published", "event_type", eventType, "entity_id", entityID, "source", EventSourceSyncPublish)
		c.JSON(http.StatusAccepted, gin.H{"message": "event published"})
	})

	authorizedSync.POST("/offline-queue/enqueue", func(c *gin.Context) {
		if routeOptions.replayManager == nil {
			observability.RecordReplayEnqueueRejected()
			logger.Warn("sync offline enqueue rejected", "reason", "offline_replay_not_configured")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "offline replay is not configured", "code": "service_unavailable"})
			return
		}

		var request struct {
			OperationID string         `json:"operation_id"`
			Type        string         `json:"type"`
			Payload     map[string]any `json:"payload"`
			OccurredAt  string         `json:"occurred_at"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			observability.RecordReplayEnqueueRejected()
			logger.Warn("sync offline enqueue rejected", "reason", "invalid_payload", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "code": "invalid_payload"})
			return
		}

		mutation := ReplayMutation{
			OperationID: strings.TrimSpace(request.OperationID),
			EventType:   strings.TrimSpace(strings.ToLower(request.Type)),
			Payload:     SanitizeIncomingPayload(request.Type, request.Payload),
		}

		if strings.TrimSpace(request.OccurredAt) != "" {
			occurredAt, err := time.Parse(time.RFC3339, strings.TrimSpace(request.OccurredAt))
			if err != nil {
				observability.RecordReplayEnqueueRejected()
				logger.Warn("sync offline enqueue rejected", "reason", "invalid_occurred_at", "error", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "occurred_at must be RFC3339", "code": "validation_error"})
				return
			}
			mutation.OccurredAt = occurredAt.UTC()
		}

		enqueued, err := routeOptions.replayManager.Enqueue(c.Request.Context(), mutation)
		if err != nil {
			observability.RecordReplayEnqueueRejected()
			logger.Warn("sync offline enqueue rejected", "reason", "validation_error", "error", err, "event_type", mutation.EventType, "operation_id", mutation.OperationID)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "validation_error"})
			return
		}

		observability.RecordReplayEnqueueAccepted()
		logger.Info("sync offline enqueue accepted", "operation_id", enqueued.OperationID, "event_type", enqueued.EventType, "entity_id", enqueued.EntityID)
		c.JSON(http.StatusAccepted, gin.H{"data": enqueued})
	})

	authorizedSync.POST("/offline-queue/replay", func(c *gin.Context) {
		if routeOptions.replayManager == nil {
			observability.RecordReplayRequest(ReplaySummary{}, errors.New("offline replay is not configured"))
			logger.Warn("sync offline replay rejected", "reason", "offline_replay_not_configured")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "offline replay is not configured", "code": "service_unavailable"})
			return
		}

		var request struct {
			Limit int `json:"limit"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			observability.RecordReplayRequest(ReplaySummary{}, err)
			logger.Warn("sync offline replay rejected", "reason", "invalid_payload", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "code": "invalid_payload"})
			return
		}

		if request.Limit <= 0 {
			request.Limit = defaultReplayRequestLimit
		}
		if request.Limit > maxSyncListLimit {
			request.Limit = maxSyncListLimit
		}

		summary, err := routeOptions.replayManager.Replay(c.Request.Context(), request.Limit)
		observability.RecordReplayRequest(summary, err)
		if err != nil {
			logger.Warn("sync offline replay failed", "error", err, "limit", request.Limit)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "internal_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": summary})
	})

	authorizedSync.GET("/conflicts", func(c *gin.Context) {
		if routeOptions.replayManager == nil {
			observability.RecordConflictListRequest(errors.New("offline replay is not configured"))
			logger.Warn("sync conflict list rejected", "reason", "offline_replay_not_configured")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "offline replay is not configured", "code": "service_unavailable"})
			return
		}

		limit, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("limit", strconv.Itoa(defaultConflictListLimit))))
		if limit <= 0 {
			limit = defaultConflictListLimit
		}
		if limit > maxSyncListLimit {
			limit = maxSyncListLimit
		}

		conflicts, err := routeOptions.replayManager.ListConflicts(c.Request.Context(), strings.TrimSpace(c.Query("entity_id")), limit)
		observability.RecordConflictListRequest(err)
		if err != nil {
			logger.Warn("sync conflict list failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "internal_error"})
			return
		}

		logger.Info("sync conflict list", "count", len(conflicts), "entity_id", strings.TrimSpace(c.Query("entity_id")))
		c.JSON(http.StatusOK, gin.H{"data": conflicts})
	})
}
