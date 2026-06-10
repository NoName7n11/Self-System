package sync

import "testing"

func TestValidateIncomingEventAcceptsLifecycleEvents(t *testing.T) {
	if err := ValidateIncomingEvent(EventTypeUpdate, map[string]any{"id": "res-1"}); err != nil {
		t.Fatalf("expected sync.update to be valid, got %v", err)
	}
	if err := ValidateIncomingEvent(EventTypeReconnected, map[string]any{"status": "ok"}); err != nil {
		t.Fatalf("expected sync.reconnected to be valid, got %v", err)
	}
}

func TestValidateIncomingEventRequiresEntityIDForEntityEvents(t *testing.T) {
	err := ValidateIncomingEvent(EventTypeResourceUpdated, map[string]any{"title": "Updated"})
	if err == nil {
		t.Fatalf("expected validation error for missing entity_id")
	}
	if err.Error() != "payload.entity_id is required" {
		t.Fatalf("expected payload.entity_id error, got %q", err.Error())
	}

	if err := ValidateIncomingEvent(EventTypeResourceUpdated, map[string]any{"entity_id": "res-1", "title": "Updated"}); err != nil {
		t.Fatalf("expected resource event to be valid with entity_id, got %v", err)
	}
}

func TestValidateIncomingEventRejectsUnsupportedType(t *testing.T) {
	err := ValidateIncomingEvent("sync.unknown", map[string]any{"entity_id": "res-1"})
	if err == nil {
		t.Fatalf("expected unsupported type validation error")
	}
	if err.Error() != "unsupported sync event type" {
		t.Fatalf("expected unsupported sync event type error, got %q", err.Error())
	}
}

func TestBuildEventPayloadAddsVersionAndSourceMetadata(t *testing.T) {
	enriched := BuildEventPayload(map[string]any{"title": "Updated"}, "res-1", EventSourceHTTPMutation)

	if enriched[PayloadKeyEntityID] != "res-1" {
		t.Fatalf("expected entity_id res-1, got %v", enriched[PayloadKeyEntityID])
	}
	if enriched[PayloadKeyEventSource] != EventSourceHTTPMutation {
		t.Fatalf("expected event source %q, got %v", EventSourceHTTPMutation, enriched[PayloadKeyEventSource])
	}
	if enriched[PayloadKeyEventVersion] != EventVersionCurrent {
		t.Fatalf("expected event version %d, got %v", EventVersionCurrent, enriched[PayloadKeyEventVersion])
	}
	if enriched["title"] != "Updated" {
		t.Fatalf("expected payload field to be preserved, got %v", enriched["title"])
	}
}

func TestBuildEventPayloadDefaultsSourceWhenEmpty(t *testing.T) {
	enriched := BuildEventPayload(nil, "", "")

	if enriched[PayloadKeyEventSource] != EventSourceSyncPublish {
		t.Fatalf("expected default event source %q, got %v", EventSourceSyncPublish, enriched[PayloadKeyEventSource])
	}
	if enriched[PayloadKeyEventVersion] != EventVersionCurrent {
		t.Fatalf("expected event version %d, got %v", EventVersionCurrent, enriched[PayloadKeyEventVersion])
	}
}

func TestExtractEntityID(t *testing.T) {
	if id := ExtractEntityID(map[string]any{PayloadKeyEntityID: "res-1"}); id != "res-1" {
		t.Fatalf("expected extracted entity id res-1, got %q", id)
	}
	if id := ExtractEntityID(map[string]any{"other": "value"}); id != "" {
		t.Fatalf("expected empty entity id for missing key, got %q", id)
	}
}

func TestSanitizeIncomingPayload(t *testing.T) {
	payload := map[string]any{
		"entity_id": "res-1",
		"title":     "Allowed",
		"evil":      "drop-me",
	}

	sanitized := SanitizeIncomingPayload(EventTypeResourceUpdated, payload)
	if sanitized["entity_id"] != "res-1" {
		t.Fatalf("expected entity_id to be preserved")
	}
	if sanitized["title"] != "Allowed" {
		t.Fatalf("expected title to be preserved")
	}
	if _, exists := sanitized["evil"]; exists {
		t.Fatalf("expected unknown key to be removed")
	}
}
