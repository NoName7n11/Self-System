package sync

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub               *Hub
	observability     *Observability
	upgrader          websocket.Upgrader
	heartbeatInterval time.Duration
}

const (
	defaultReplayLimit = 200
	maxReplayLimit     = 2000
)

func NewWSHandler(hub *Hub, allowedOrigins []string, heartbeatSeconds int, observability *Observability) *WSHandler {
	if hub == nil {
		hub = NewHub()
	}
	if heartbeatSeconds <= 0 {
		heartbeatSeconds = 30
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
	sinceSequence := parseSinceSequence(c.Query("since_sequence"))
	replayLimit := parseReplayLimit(c.Query("replay_limit"))

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.observability.RecordWSUpgradeFailure()
		c.JSON(http.StatusBadRequest, gin.H{"error": "websocket upgrade failed", "code": "invalid_request"})
		return
	}
	h.observability.RecordWSConnected()
	defer func() {
		h.observability.RecordWSDisconnected()
		_ = conn.Close()
	}()

	events, unsubscribe := h.hub.Subscribe(16)
	defer unsubscribe()

	done := make(chan struct{})
	go h.readLoop(conn, done)

	lastReplayedSequence := int64(0)
	if sinceSequence > 0 {
		replayed := h.hub.ReplaySince(sinceSequence, replayLimit)
		reconnectPayload := gin.H{
			"since_sequence": sinceSequence,
			"replayed_count": len(replayed),
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

	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(2 * h.heartbeatInterval))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(2 * h.heartbeatInterval))
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
