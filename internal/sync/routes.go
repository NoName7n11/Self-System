package sync

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"selfsystems/internal/config"
)

type bootstrapRouteOptions struct {
	replayManager *OfflineReplayManager
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
			"status":                     "ok",
			"auth_enabled":               cfg.Auth.Enabled,
			"google_oauth_configured":    strings.TrimSpace(cfg.Auth.GoogleClientID) != "" && strings.TrimSpace(cfg.Auth.GoogleClientSecret) != "",
			"jwt_signing_key_configured": strings.TrimSpace(cfg.Auth.JWTSecret) != "",
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

	wsHandler := NewWSHandler(hub, cfg.Sync.AllowedOrigins, cfg.Sync.HeartbeatSeconds, observability)
	router.GET(cfg.Sync.WebSocketPath, observability.StatusMiddleware(), authMiddleware, wsHandler.ServeHTTP)

	api.GET("/sync/health", func(c *gin.Context) {
		hubStats := hub.Stats()
		c.JSON(http.StatusOK, gin.H{
			"status":            "ok",
			"sync_enabled":      true,
			"websocket_path":    cfg.Sync.WebSocketPath,
			"connected_clients": hubStats.ConnectedClients,
			"offline_replay":    routeOptions.replayManager != nil,
			"hub":               hubStats,
			"metrics":           observability.Snapshot(),
		})
	})

	authorizedSync := api.Group("/sync")
	authorizedSync.Use(observability.StatusMiddleware(), authMiddleware)
	authorizedSync.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": observability.Snapshot()})
	})

	authorizedSync.POST("/events", func(c *gin.Context) {
		var request struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			observability.RecordSyncEventRejected()
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "code": "invalid_payload"})
			return
		}

		eventType := strings.ToLower(strings.TrimSpace(request.Type))
		if err := ValidateIncomingEvent(eventType, request.Payload); err != nil {
			observability.RecordSyncEventRejected()
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "validation_error"})
			return
		}

		request.Payload = BuildEventPayload(request.Payload, ExtractEntityID(request.Payload), EventSourceSyncPublish)

		hub.Publish(NewEvent(eventType, request.Payload))
		observability.RecordSyncEventPublished()
		c.JSON(http.StatusAccepted, gin.H{"message": "event published"})
	})

	authorizedSync.POST("/offline-queue/enqueue", func(c *gin.Context) {
		if routeOptions.replayManager == nil {
			observability.RecordReplayEnqueueRejected()
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "code": "invalid_payload"})
			return
		}

		mutation := ReplayMutation{
			OperationID: strings.TrimSpace(request.OperationID),
			EventType:   strings.TrimSpace(strings.ToLower(request.Type)),
			Payload:     request.Payload,
		}

		if strings.TrimSpace(request.OccurredAt) != "" {
			occurredAt, err := time.Parse(time.RFC3339, strings.TrimSpace(request.OccurredAt))
			if err != nil {
				observability.RecordReplayEnqueueRejected()
				c.JSON(http.StatusBadRequest, gin.H{"error": "occurred_at must be RFC3339", "code": "validation_error"})
				return
			}
			mutation.OccurredAt = occurredAt.UTC()
		}

		enqueued, err := routeOptions.replayManager.Enqueue(c.Request.Context(), mutation)
		if err != nil {
			observability.RecordReplayEnqueueRejected()
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "validation_error"})
			return
		}

		observability.RecordReplayEnqueueAccepted()
		c.JSON(http.StatusAccepted, gin.H{"data": enqueued})
	})

	authorizedSync.POST("/offline-queue/replay", func(c *gin.Context) {
		if routeOptions.replayManager == nil {
			observability.RecordReplayRequest(ReplaySummary{}, errors.New("offline replay is not configured"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "offline replay is not configured", "code": "service_unavailable"})
			return
		}

		var request struct {
			Limit int `json:"limit"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			observability.RecordReplayRequest(ReplaySummary{}, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "code": "invalid_payload"})
			return
		}

		summary, err := routeOptions.replayManager.Replay(c.Request.Context(), request.Limit)
		observability.RecordReplayRequest(summary, err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "internal_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": summary})
	})

	authorizedSync.GET("/conflicts", func(c *gin.Context) {
		if routeOptions.replayManager == nil {
			observability.RecordConflictListRequest(errors.New("offline replay is not configured"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "offline replay is not configured", "code": "service_unavailable"})
			return
		}

		limit, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("limit", "50")))
		if limit <= 0 {
			limit = 50
		}

		conflicts, err := routeOptions.replayManager.ListConflicts(c.Request.Context(), strings.TrimSpace(c.Query("entity_id")), limit)
		observability.RecordConflictListRequest(err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "code": "internal_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": conflicts})
	})
}
