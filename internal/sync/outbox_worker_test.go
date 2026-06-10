package sync

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"selfsystems/internal/eventstore"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

// ── test helpers ─────────────────────────────────────────────────────────────

func newOutboxTestStore(t *testing.T) eventstore.Store {
	t.Helper()
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "outbox.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return eventstore.NewSQLiteStore(db)
}

func appendResourceEvent(t *testing.T, store eventstore.Store, eventID, resourceID, eventType string, version int) eventstore.AppendResult {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"url": "https://example.com/" + resourceID, "host": "example.com",
		"title": "T", "summary": "", "category_id": "cat-1", "category_name": "Cat",
		"user_override": false,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
	})
	result, err := store.Append(context.Background(), eventstore.Event{
		EventID:       eventID,
		AggregateID:   resourceID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventType,
		EventVersion:  version,
		Payload:       json.RawMessage(payload),
	})
	if err != nil {
		t.Fatalf("append %s: %v", eventType, err)
	}
	return result
}

// drainHub collects up to n events from the hub's subscribe channel within timeout.
func drainHub(t *testing.T, hub *Hub, n int, timeout time.Duration) []Event {
	t.Helper()
	ch, unsub := hub.Subscribe(n + 4)
	defer unsub()

	var got []Event
	deadline := time.After(timeout)
	for len(got) < n {
		select {
		case e := <-ch:
			got = append(got, e)
		case <-deadline:
			return got
		}
	}
	return got
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestOutboxWorkerPublishesResourceCreated(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	evtID := uuid.NewString()
	rid := uuid.NewString()
	appendResourceEvent(t, store, evtID, rid, eventstore.EventTypeResourceCreated, 1)

	worker := NewOutboxWorker(store, hub, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go worker.Start(ctx)

	events := drainHub(t, hub, 1, 2*time.Second)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventTypeResourceCreated {
		t.Fatalf("expected sync.resource.created, got %s", events[0].Type)
	}
	payload := events[0].Payload.(map[string]any)
	if payload["entity_id"] != rid {
		t.Fatalf("expected entity_id=%s, got %v", rid, payload["entity_id"])
	}
}

func TestOutboxWorkerPublishesResourceImported(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	evtID := uuid.NewString()
	rid := uuid.NewString()
	appendResourceEvent(t, store, evtID, rid, eventstore.EventTypeResourceImported, 1)

	worker := NewOutboxWorker(store, hub, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go worker.Start(ctx)

	events := drainHub(t, hub, 1, 2*time.Second)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// ResourceImported maps to sync.resource.created
	if events[0].Type != EventTypeResourceCreated {
		t.Fatalf("expected sync.resource.created for Imported, got %s", events[0].Type)
	}
}

func TestOutboxWorkerPublishesResourceUpdated(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	rid := uuid.NewString()
	appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceCreated, 1)
	appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceUpdated, 2)

	worker := NewOutboxWorker(store, hub, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go worker.Start(ctx)

	events := drainHub(t, hub, 2, 2*time.Second)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].Type != EventTypeResourceUpdated {
		t.Fatalf("expected sync.resource.updated, got %s", events[1].Type)
	}
}

func TestOutboxWorkerPublishesResourceDeleted(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	rid := uuid.NewString()
	appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceCreated, 1)
	appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceDeleted, 2)

	worker := NewOutboxWorker(store, hub, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go worker.Start(ctx)

	events := drainHub(t, hub, 2, 2*time.Second)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].Type != EventTypeResourceDeleted {
		t.Fatalf("expected sync.resource.deleted, got %s", events[1].Type)
	}
}

func TestOutboxWorkerSkipsUnknownEventTypes(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	// Append a GBUS signal event (aggregate_type = gbus_signal) — not mapped to a sync type.
	_, err := store.Append(context.Background(), eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   "signal-1",
		AggregateType: "gbus_signal",
		EventType:     "gbus.resource_opened",
		EventVersion:  1,
		Payload:       json.RawMessage(`{"resource_id":"r1"}`),
	})
	if err != nil {
		t.Fatalf("append gbus event: %v", err)
	}

	// Append one known event after.
	rid := uuid.NewString()
	appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceCreated, 1)

	worker := NewOutboxWorker(store, hub, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go worker.Start(ctx)

	// Should only receive the ResourceCreated event, not the gbus signal.
	events := drainHub(t, hub, 1, 2*time.Second)
	if len(events) != 1 {
		t.Fatalf("expected 1 event (ResourceCreated only), got %d", len(events))
	}
	if events[0].Type != EventTypeResourceCreated {
		t.Fatalf("unexpected event type: %s", events[0].Type)
	}
}

func TestOutboxWorkerSetsHubSequenceFromEventStore(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	rid := uuid.NewString()
	result := appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceCreated, 1)
	estoreSeq := result.Sequence

	worker := NewOutboxWorker(store, hub, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go worker.Start(ctx)

	events := drainHub(t, hub, 1, 2*time.Second)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	// The hub event must carry the eventstore sequence for since_sequence alignment.
	if events[0].Sequence != estoreSeq {
		t.Fatalf("expected hub sequence=%d (eventstore seq), got %d", estoreSeq, events[0].Sequence)
	}
}

func TestOutboxWorkerLastSequence(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	rid := uuid.NewString()
	r1 := appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceCreated, 1)

	worker := NewOutboxWorker(store, hub, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go worker.Start(ctx)

	drainHub(t, hub, 1, 2*time.Second)

	if worker.LastSequence() != r1.Sequence {
		t.Fatalf("expected LastSequence=%d, got %d", r1.Sequence, worker.LastSequence())
	}
	if worker.Published() < 1 {
		t.Fatalf("expected Published >= 1, got %d", worker.Published())
	}
}

func TestOutboxWorkerPicksUpNewEvents(t *testing.T) {
	store := newOutboxTestStore(t)
	hub := NewHub()

	// Start worker with empty store.
	worker := NewOutboxWorker(store, hub, 50*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go worker.Start(ctx)

	// Give worker time to settle at sequence=0.
	time.Sleep(100 * time.Millisecond)

	// Append a new event after worker is running.
	rid := uuid.NewString()
	appendResourceEvent(t, store, uuid.NewString(), rid, eventstore.EventTypeResourceCreated, 1)

	events := drainHub(t, hub, 1, 2*time.Second)
	if len(events) != 1 {
		t.Fatalf("expected 1 new event, got %d", len(events))
	}
	if events[0].Type != EventTypeResourceCreated {
		t.Fatalf("unexpected type: %s", events[0].Type)
	}
}

// ── mergeDurableAndHubReplay (Finding 3 residual edge case) ────────────────────

// TestMergeReplay_SequenceCollisionKeepsHubEvent is the regression test for the
// Finding 3 residual edge case: a directly-published hub event (category update
// from http.mutation) whose minted sequence collides with an unrelated
// events-table sequence must NOT be dropped during reconnect merge.
func TestMergeReplay_SequenceCollisionKeepsHubEvent(t *testing.T) {
	// Events table: a resource event at sequence 4 (outbox-originated).
	stored := []eventstore.Event{
		{
			Sequence:    4,
			AggregateID: "res-1",
			EventType:   eventstore.EventTypeResourceCreated,
			Payload:     json.RawMessage(`{"url":"https://example.com"}`),
		},
	}
	// Hub history: a category update minted the SAME sequence (4) by a direct
	// publisher — a different origin (http.mutation), not outbox.worker.
	hubEvents := []Event{
		{
			Sequence: 4,
			Type:     EventTypeCategoryUpdated,
			Payload: map[string]any{
				PayloadKeyEntityID:    "cat-1",
				PayloadKeyEventSource: EventSourceHTTPMutation,
			},
		},
	}

	merged := mergeDurableAndHubReplay(stored, hubEvents)

	if len(merged) != 2 {
		t.Fatalf("expected 2 merged events (collision must not drop the hub event), got %d", len(merged))
	}
	var sawResource, sawCategory bool
	for _, e := range merged {
		switch e.Type {
		case EventTypeResourceCreated:
			sawResource = true
		case EventTypeCategoryUpdated:
			sawCategory = true
		}
	}
	if !sawResource || !sawCategory {
		t.Errorf("expected both events; resource=%v category=%v", sawResource, sawCategory)
	}
}

// TestMergeReplay_DedupesOutboxOriginatedHubEvents verifies the dual property:
// a hub event that came from the outbox worker IS deduped against the events
// table read (it is the same event, written durably and also broadcast).
func TestMergeReplay_DedupesOutboxOriginatedHubEvents(t *testing.T) {
	stored := []eventstore.Event{
		{
			Sequence:    1,
			AggregateID: "res-1",
			EventType:   eventstore.EventTypeResourceCreated,
			Payload:     json.RawMessage(`{"url":"https://example.com"}`),
		},
	}
	// The same logical event sits in hub history tagged outbox.worker.
	hubEvents := []Event{
		{
			Sequence: 1,
			Type:     EventTypeResourceCreated,
			Payload: map[string]any{
				PayloadKeyEntityID:    "res-1",
				PayloadKeyEventSource: EventSourceOutboxWorker,
			},
		},
	}

	merged := mergeDurableAndHubReplay(stored, hubEvents)
	if len(merged) != 1 {
		t.Fatalf("expected outbox-originated hub event to be deduped, got %d events", len(merged))
	}
}

// TestMergeReplay_SkipsUntranslatableStored verifies untranslatable events-table
// rows (e.g. gbus signals) are dropped from the merge without affecting others.
func TestMergeReplay_SkipsUntranslatableStored(t *testing.T) {
	stored := []eventstore.Event{
		{Sequence: 1, AggregateID: "sig-1", EventType: "gbus.resource_opened", Payload: json.RawMessage(`{}`)},
		{Sequence: 2, AggregateID: "res-1", EventType: eventstore.EventTypeResourceCreated, Payload: json.RawMessage(`{"url":"https://example.com"}`)},
	}
	merged := mergeDurableAndHubReplay(stored, nil)
	if len(merged) != 1 {
		t.Fatalf("expected 1 translatable event, got %d", len(merged))
	}
	if merged[0].Type != EventTypeResourceCreated {
		t.Fatalf("expected resource.created, got %s", merged[0].Type)
	}
}

// TestMergeReplay_SkippedRowInterleaving is the regression test for the
// Finding 3 residual edge case: an untranslatable event-store row, a direct
// hub event around it, then a durable replay — asserting no sequence reuse
// and no dropped event.
func TestMergeReplay_SkippedRowInterleaving(t *testing.T) {
	stored := []eventstore.Event{
		{Sequence: 1, AggregateID: "sig-1", EventType: "gbus.resource_opened", Payload: json.RawMessage(`{}`)}, // Untranslatable, skipped
		{Sequence: 3, AggregateID: "res-1", EventType: eventstore.EventTypeResourceCreated, Payload: json.RawMessage(`{"url":"https://example.com"}`)}, // Translatable, kept
	}
	hubEvents := []Event{
		{Sequence: 2, Type: EventTypeCategoryUpdated, Payload: map[string]any{PayloadKeyEventSource: EventSourceHTTPMutation}}, // Direct hub event, kept
	}

	merged := mergeDurableAndHubReplay(stored, hubEvents)

	if len(merged) != 2 {
		t.Fatalf("expected 2 merged events, got %d", len(merged))
	}
	if merged[0].Sequence != 2 {
		t.Fatalf("expected first event sequence 2, got %d", merged[0].Sequence)
	}
	if merged[0].Type != EventTypeCategoryUpdated {
		t.Fatalf("expected first event CategoryUpdated, got %s", merged[0].Type)
	}
	if merged[1].Sequence != 3 {
		t.Fatalf("expected second event sequence 3, got %d", merged[1].Sequence)
	}
	if merged[1].Type != EventTypeResourceCreated {
		t.Fatalf("expected second event ResourceCreated, got %s", merged[1].Type)
	}
}

// TestMergeReplay_SortedBySequence verifies the merged output is sequence-ordered.
func TestMergeReplay_SortedBySequence(t *testing.T) {
	stored := []eventstore.Event{
		{Sequence: 5, AggregateID: "res-2", EventType: eventstore.EventTypeResourceCreated, Payload: json.RawMessage(`{"url":"https://example.com/2"}`)},
	}
	hubEvents := []Event{
		{Sequence: 2, Type: EventTypeCategoryUpdated, Payload: map[string]any{PayloadKeyEventSource: EventSourceHTTPMutation}},
	}
	merged := mergeDurableAndHubReplay(stored, hubEvents)
	if len(merged) != 2 {
		t.Fatalf("expected 2 events, got %d", len(merged))
	}
	if merged[0].Sequence > merged[1].Sequence {
		t.Errorf("merged events not sorted: %d before %d", merged[0].Sequence, merged[1].Sequence)
	}
}

// ── outboxTranslate unit tests ────────────────────────────────────────────────

func TestOutboxTranslateMapping(t *testing.T) {
	cases := []struct {
		evtType  string
		syncType string
		wantOK   bool
	}{
		{eventstore.EventTypeResourceCreated, EventTypeResourceCreated, true},
		{eventstore.EventTypeResourceImported, EventTypeResourceCreated, true},
		{eventstore.EventTypeResourceUpdated, EventTypeResourceUpdated, true},
		{eventstore.EventTypeResourceCategoryAssigned, EventTypeResourceUpdated, true},
		{eventstore.EventTypeResourceDeleted, EventTypeResourceDeleted, true},
		{"gbus.resource_opened", "", false},
		{"unknown.event", "", false},
	}

	for _, tc := range cases {
		e := eventstore.Event{
			AggregateID: "r1",
			EventType:   tc.evtType,
			Sequence:    42,
			Payload:     json.RawMessage(`{"url":"https://example.com","host":"example.com"}`),
		}
		syncEvt, ok := outboxTranslate(e)
		if ok != tc.wantOK {
			t.Errorf("%s: wantOK=%v, got %v", tc.evtType, tc.wantOK, ok)
			continue
		}
		if !ok {
			continue
		}
		if syncEvt.Type != tc.syncType {
			t.Errorf("%s: expected sync type %s, got %s", tc.evtType, tc.syncType, syncEvt.Type)
		}
		if syncEvt.Sequence != 42 {
			t.Errorf("%s: expected sequence=42, got %d", tc.evtType, syncEvt.Sequence)
		}
		payload := syncEvt.Payload.(map[string]any)
		if payload["entity_id"] != "r1" {
			t.Errorf("%s: missing entity_id in payload: %v", tc.evtType, payload)
		}
		if payload["event_source"] != EventSourceOutboxWorker {
			t.Errorf("%s: expected event_source=%s, got %v", tc.evtType, EventSourceOutboxWorker, payload["event_source"])
		}
	}
}
