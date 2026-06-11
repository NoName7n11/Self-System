package http

import (
	"strings"

	syncapi "selfsystems/internal/sync"
)

func (h *Handler) publishSyncEvent(eventType, entityID string, payload map[string]any) {
	h.publishSyncEventWithSource(eventType, entityID, syncapi.EventSourceHTTPMutation, payload)
}

func (h *Handler) publishSyncEventWithSource(eventType, entityID, source string, payload map[string]any) {
	if h.syncHub == nil || strings.TrimSpace(eventType) == "" {
		return
	}

	enrichedPayload := syncapi.BuildEventPayload(payload, entityID, source)
	h.syncHub.Publish(syncapi.NewEvent(eventType, enrichedPayload))
}
