package sync

import (
	"fmt"
	"strings"
)

const (
	EventTypeConnected       = "sync.connected"
	EventTypeReconnected     = "sync.reconnected"
	EventTypeHeartbeat       = "sync.heartbeat"
	EventTypeUpdate          = "sync.update"
	EventTypeResourceCreated = "sync.resource.created"
	EventTypeResourceUpdated = "sync.resource.updated"
	EventTypeResourceDeleted = "sync.resource.deleted"
	EventTypeCategoryUpdated = "sync.category.updated"
	EventTypeTodoUpdated     = "sync.todo.updated"
	EventTypeReminderUpdated = "sync.reminder.updated"

	PayloadKeyEntityID     = "entity_id"
	PayloadKeyEventVersion = "event_version"
	PayloadKeyEventSource  = "event_source"

	EventVersionCurrent = 1

	EventSourceHTTPMutation = "http.mutation"
	EventSourceChatCommand  = "chat.command"
	EventSourceSyncPublish  = "sync.publish"
	EventSourceSyncReplay   = "sync.replay"
)

type eventTypeRule struct {
	requireEntityID bool
}

var incomingEventRules = map[string]eventTypeRule{
	EventTypeUpdate:          {requireEntityID: false},
	EventTypeReconnected:     {requireEntityID: false},
	EventTypeResourceCreated: {requireEntityID: true},
	EventTypeResourceUpdated: {requireEntityID: true},
	EventTypeResourceDeleted: {requireEntityID: true},
	EventTypeCategoryUpdated: {requireEntityID: true},
	EventTypeTodoUpdated:     {requireEntityID: true},
	EventTypeReminderUpdated: {requireEntityID: true},
}

func ValidateIncomingEvent(eventType string, payload map[string]any) error {
	normalizedType := strings.ToLower(strings.TrimSpace(eventType))
	if normalizedType == "" {
		return fmt.Errorf("type is required")
	}

	rule, ok := incomingEventRules[normalizedType]
	if !ok {
		return fmt.Errorf("unsupported sync event type")
	}

	if rule.requireEntityID {
		if payload == nil {
			return fmt.Errorf("payload.entity_id is required")
		}
		value, exists := payload[PayloadKeyEntityID]
		if !exists || strings.TrimSpace(fmt.Sprint(value)) == "" {
			return fmt.Errorf("payload.entity_id is required")
		}
	}

	return nil
}

func ExtractEntityID(payload map[string]any) string {
	if payload == nil {
		return ""
	}

	value, exists := payload[PayloadKeyEntityID]
	if !exists {
		return ""
	}

	return strings.TrimSpace(fmt.Sprint(value))
}

func BuildEventPayload(payload map[string]any, entityID, eventSource string) map[string]any {
	enriched := make(map[string]any)
	for key, value := range payload {
		enriched[key] = value
	}

	if strings.TrimSpace(entityID) != "" {
		enriched[PayloadKeyEntityID] = strings.TrimSpace(entityID)
	}

	source := strings.TrimSpace(eventSource)
	if source == "" {
		source = EventSourceSyncPublish
	}

	enriched[PayloadKeyEventVersion] = EventVersionCurrent
	enriched[PayloadKeyEventSource] = source

	return enriched
}
