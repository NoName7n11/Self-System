package sync

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"selfsystems/internal/eventstore"
)

type WSHandler struct {
	hub               *Hub
	eventStore        eventstore.Store // optional; enables durable replay from events table
	observability     *Observability
	upgrader          websocket.Upgrader
	heartbeatInterval time.Duration
	maxConnections    int
	connMu            sync.Mutex
	connByClient      map[string]int
}

// SetEventStore enables events-table-backed replay for reconnecting clients.
// When set, since_sequence queries the events table before falling back to hub history.
func (h *WSHandler) SetEventStore(store eventstore.Store) {
	h.eventStore = store
}

const (
	defaultReplayLimit            = 200
	maxReplayLimit                = 2000
	maxInboundMessagesPerMinute   = 120
	inboundMessageLimitWindowSize = time.Minute
)

func NewWSHandler(hub *Hub, allowedOrigins []string, heartbeatSeconds, maxConnectionsPerClient int, observability *Observability) *WSHandler {
	if hub == nil {
		hub = NewHub()
	}
	if heartbeatSeconds <= 0 {
		heartbeatSeconds = 30
	}
	if maxConnectionsPerClient <= 0 {
		maxConnectionsPerClient = 5
	}

	normalizedOrigins := normalizeOrigins(allowedOrigins)
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if len(normalizedOrigins) == 0 {
				return true
			}

			origin := strings.TrimSpace(strings.ToLower(r.Header.Get("Origin")))
			for _, allowed := range normalizedOrigins {
				if origin == allowed {
					return true
				}
			}

			return false
		},
	}

	return &WSHandler{
		hub:               hub,
		observability:     observability,
		upgrader:          upgrader,
		heartbeatInterval: time.Duration(heartbeatSeconds) * time.Second,
		maxConnections:    maxConnectionsPerClient,
		connByClient:      map[string]int{},
	}
}

func normalizeOrigins(origins []string) []string {
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(strings.ToLower(origin))
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func (h *WSHandler) ServeHTTP(c *gin.Context) {
	logger := syncLogger()
	clientIP := c.ClientIP()
	if !h.acquireClientConnection(clientIP) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "connection limit exceeded", "code": "rate_limited"})
		return
	}
	defer h.releaseClientConnection(clientIP)

	sinceSequence := parseSinceSequence(c.Query("since_sequence"))
	replayLimit := parseReplayLimit(c.Query("replay_limit"))

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.observability.RecordWSUpgradeFailure()
		logger.Warn("sync websocket upgrade failed", "client_ip", clientIP, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "websocket upgrade failed", "code": "invalid_request"})
		return
	}
	h.observability.RecordWSConnected()
	defer func() {
		h.observability.RecordWSDisconnected()
		logger.Info("sync websocket disconnected", "client_ip", clientIP)
		_ = conn.Close()
	}()

	logger.Info("sync websocket connected", "client_ip", clientIP, "since_sequence", sinceSequence, "replay_limit", replayLimit)

	events, unsubscribe := h.hub.Subscribe(16)
	defer unsubscribe()

	done := make(chan struct{})
	go h.readLoop(conn, done)

	lastReplayedSequence := int64(0)
	if sinceSequence > 0 {
		var replayed []Event
		truncated := false

		if h.eventStore != nil {
			// Durable replay: query events table first. The outbox worker aligns
			// hub sequences with eventstore sequences, so since_sequence works
			// for both sources. This survives server restarts (hub history does not).
			storedEvents, readErr := h.eventStore.ReadBySequence(c.Request.Context(), sinceSequence, replayLimit)
			if readErr != nil {
				logger.Warn("sync durable replay read failed, falling back to hub history", "error", readErr)
			} else {
				hubEvents, hubTruncated := h.hub.ReplaySinceWithMetadata(sinceSequence, replayLimit)
				truncated = hubTruncated
				replayed = mergeDurableAndHubReplay(storedEvents, hubEvents)
			}
		}

		// Fall back to hub-only history if eventStore not configured or read failed.
		if len(replayed) == 0 && h.eventStore == nil {
			replayed, truncated = h.hub.ReplaySinceWithMetadata(sinceSequence, replayLimit)
		}

		reconnectPayload := gin.H{
			"since_sequence": sinceSequence,
			"replayed_count": len(replayed),
			"truncated":      truncated,
		}
		if len(replayed) > 0 {
			lastReplayedSequence = replayed[len(replayed)-1].Sequence
			reconnectPayload["last_replayed_sequence"] = lastReplayedSequence
		}

		if err := conn.WriteJSON(NewEvent(EventTypeReconnected, reconnectPayload)); err != nil {
			return
		}

		for _, replayEvent := range replayed {
			if err := conn.WriteJSON(replayEvent); err != nil {
				return
			}
		}

		logger.Info(
			"sync websocket replayed",
			"client_ip", clientIP,
			"since_sequence", sinceSequence,
			"replay_limit", replayLimit,
			"replayed_count", len(replayed),
			"last_replayed_sequence", lastReplayedSequence,
			"durable", h.eventStore != nil,
		)
	}

	if err := conn.WriteJSON(NewEvent(EventTypeConnected, gin.H{"message": "sync websocket connected"})); err != nil {
		return
	}

	ticker := time.NewTicker(h.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if lastReplayedSequence > 0 && event.Sequence > 0 && event.Sequence <= lastReplayedSequence {
				continue
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}

func (h *WSHandler) acquireClientConnection(clientID string) bool {
	trimmed := strings.TrimSpace(clientID)
	if trimmed == "" {
		trimmed = "unknown"
	}

	h.connMu.Lock()
	defer h.connMu.Unlock()

	current := h.connByClient[trimmed]
	if current >= h.maxConnections {
		return false
	}
	h.connByClient[trimmed] = current + 1
	return true
}

func (h *WSHandler) releaseClientConnection(clientID string) {
	trimmed := strings.TrimSpace(clientID)
	if trimmed == "" {
		trimmed = "unknown"
	}

	h.connMu.Lock()
	defer h.connMu.Unlock()

	current := h.connByClient[trimmed]
	if current <= 1 {
		delete(h.connByClient, trimmed)
		return
	}
	h.connByClient[trimmed] = current - 1
}

func parseSinceSequence(raw string) int64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}

	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || parsed <= 0 {
		return 0
	}

	return parsed
}

func parseReplayLimit(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultReplayLimit
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return defaultReplayLimit
	}
	if parsed > maxReplayLimit {
		return maxReplayLimit
	}

	return parsed
}

func (h *WSHandler) readLoop(conn *websocket.Conn, done chan<- struct{}) {
	defer close(done)
	logger := syncLogger()

	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(2 * h.heartbeatInterval))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(2 * h.heartbeatInterval))
		return nil
	})
	limiter := newInboundMessageLimiter(maxInboundMessagesPerMinute, inboundMessageLimitWindowSize)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		if !limiter.Allow(time.Now().UTC()) {
			logger.Warn("sync websocket inbound message rate exceeded", "limit", maxInboundMessagesPerMinute, "window_seconds", int(inboundMessageLimitWindowSize.Seconds()))
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "inbound message rate exceeded"),
				time.Now().Add(2*time.Second),
			)
			return
		}
	}
}

type inboundMessageLimiter struct {
	windowStart time.Time
	count       int
	limit       int
	window      time.Duration
}

func newInboundMessageLimiter(limit int, window time.Duration) *inboundMessageLimiter {
	if limit <= 0 {
		limit = maxInboundMessagesPerMinute
	}
	if window <= 0 {
		window = inboundMessageLimitWindowSize
	}
	return &inboundMessageLimiter{
		windowStart: time.Now().UTC(),
		limit:       limit,
		window:      window,
	}
}

func (l *inboundMessageLimiter) Allow(now time.Time) bool {
	if now.Sub(l.windowStart) >= l.window {
		l.windowStart = now
		l.count = 0
	}
	l.count++
	return l.count <= l.limit
}
