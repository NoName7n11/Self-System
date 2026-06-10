package sync

import (
	"testing"
	"time"
)

func TestHubPublishDeliversToSubscriber(t *testing.T) {
	hub := NewHub()
	events, unsubscribe := hub.Subscribe(1)
	defer unsubscribe()

	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-1"}))

	select {
	case event := <-events:
		if event.Type != "sync.update" {
			t.Fatalf("expected event type sync.update, got %q", event.Type)
		}
		if event.Sequence == 0 {
			t.Fatalf("expected non-zero event sequence")
		}
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("expected payload map, got %T", event.Payload)
		}
		if payload["id"] != "res-1" {
			t.Fatalf("expected payload id res-1, got %v", payload["id"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestHubTracksSequenceAndDropsForSlowSubscribers(t *testing.T) {
	hub := NewHub()
	events, unsubscribe := hub.Subscribe(1)
	defer unsubscribe()

	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-1"}))

	select {
	case first := <-events:
		if first.Sequence != 1 {
			t.Fatalf("expected first sequence 1, got %d", first.Sequence)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-2"}))
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-3"}))

	select {
	case second := <-events:
		if second.Sequence != 2 {
			t.Fatalf("expected buffered sequence 2, got %d", second.Sequence)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for second event")
	}

	stats := hub.Stats()
	if stats.LastSequence != 3 {
		t.Fatalf("expected last sequence 3, got %d", stats.LastSequence)
	}
	if stats.HistoryDepth != 3 {
		t.Fatalf("expected history depth 3, got %d", stats.HistoryDepth)
	}
	if stats.PublishedTotal != 3 {
		t.Fatalf("expected published total 3, got %d", stats.PublishedTotal)
	}
	if stats.DroppedTotal != 1 {
		t.Fatalf("expected dropped total 1, got %d", stats.DroppedTotal)
	}
}

func TestHubReplaySinceReturnsOrderedEvents(t *testing.T) {
	hub := NewHub()
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-1"}))
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-2"}))
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-3"}))
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-4"}))

	replayed := hub.ReplaySince(2, 0)
	if len(replayed) != 2 {
		t.Fatalf("expected 2 replayed events, got %d", len(replayed))
	}
	if replayed[0].Sequence != 3 || replayed[1].Sequence != 4 {
		t.Fatalf("expected replay sequences [3 4], got [%d %d]", replayed[0].Sequence, replayed[1].Sequence)
	}

	limited := hub.ReplaySince(0, 2)
	if len(limited) != 2 {
		t.Fatalf("expected limited replay size 2, got %d", len(limited))
	}
	if limited[0].Sequence != 1 || limited[1].Sequence != 2 {
		t.Fatalf("expected limited replay sequences [1 2], got [%d %d]", limited[0].Sequence, limited[1].Sequence)
	}
}

func TestHubReplaySinceWithMetadataFlagsTruncation(t *testing.T) {
	hub := NewHub()
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-1"}))
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-2"}))
	hub.Publish(NewEvent("sync.update", map[string]any{"id": "res-3"}))

	events, truncated := hub.ReplaySinceWithMetadata(0, 2)
	if !truncated {
		t.Fatalf("expected replay to be truncated")
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Sequence != 1 || events[1].Sequence != 2 {
		t.Fatalf("expected earliest two sequences [1 2], got [%d %d]", events[0].Sequence, events[1].Sequence)
	}
}

func TestHubUnsubscribeRemovesClient(t *testing.T) {
	hub := NewHub()
	_, unsubscribe := hub.Subscribe(1)

	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 connected client, got %d", hub.ClientCount())
	}

	unsubscribe()

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 connected clients after unsubscribe, got %d", hub.ClientCount())
	}
}
