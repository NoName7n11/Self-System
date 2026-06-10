package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	sqliterepo "selfsystems/internal/repository/sqlite"
)

func newSQLiteStore(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return NewSQLiteStore(db), func() { _ = db.Close() }
}

func newResourceEvent(eventID string, version int) Event {
	return Event{
		EventID:              eventID,
		AggregateID:          "00000000-0000-0000-0000-000000000001",
		AggregateType:        "resource",
		EventType:            "ResourceCreated",
		EventVersion:         version,
		Payload:              json.RawMessage(`{"summary":"test"}`),
		PayloadSchemaVersion: 1,
	}
}

// ── Append ──────────────────────────────────────────────────────────────────

func TestSQLiteStoreAppendAndRead(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	result, err := store.Append(context.Background(), newResourceEvent("550e8400-e29b-41d4-a716-446655440000", 1))
	if err != nil {
		t.Fatalf("append event: %v", err)
	}
	if !result.Applied {
		t.Fatalf("expected applied=true")
	}
	if result.Sequence <= 0 {
		t.Fatalf("expected sequence > 0, got %d", result.Sequence)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0000-000000000001", 0, 0)
	if err != nil {
		t.Fatalf("read by aggregate: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected event_id: %s", events[0].EventID)
	}
}

func TestSQLiteStoreAppendIdempotent(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	first, err := store.Append(context.Background(), newResourceEvent("550e8400-e29b-41d4-a716-446655440001", 1))
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	second, err := store.Append(context.Background(), newResourceEvent("550e8400-e29b-41d4-a716-446655440001", 1))
	if err != nil {
		t.Fatalf("append duplicate: %v", err)
	}
	if second.Applied {
		t.Fatalf("expected applied=false for duplicate")
	}
	if second.Sequence != first.Sequence {
		t.Fatalf("expected same sequence, got %d vs %d", first.Sequence, second.Sequence)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0000-000000000001", 0, 0)
	if err != nil {
		t.Fatalf("read by aggregate: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after duplicate, got %d", len(events))
	}
}

func TestSQLiteStoreAppendConcurrencyConflict(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	_, err := store.Append(context.Background(), newResourceEvent("550e8400-e29b-41d4-a716-446655440002", 1))
	if err != nil {
		t.Fatalf("first append: %v", err)
	}

	_, err = store.Append(context.Background(), newResourceEvent("550e8400-e29b-41d4-a716-446655440003", 1))
	if !errors.Is(err, ErrConcurrencyConflict) {
		t.Fatalf("expected ErrConcurrencyConflict, got %v", err)
	}
}

func TestSQLiteStoreAppendRejectsInvalidPayload(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	evt := newResourceEvent("550e8400-e29b-41d4-a716-446655440004", 1)
	evt.Payload = nil
	if _, err := store.Append(context.Background(), evt); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload for nil payload, got %v", err)
	}

	evt.Payload = json.RawMessage(`not-json`)
	if _, err := store.Append(context.Background(), evt); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload for malformed JSON, got %v", err)
	}
}

func TestSQLiteStoreAppendRejectsInvalidUUID(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	evt := newResourceEvent("not-a-uuid", 1)
	if _, err := store.Append(context.Background(), evt); err == nil {
		t.Fatalf("expected error for non-UUID event_id")
	}
}

func TestSQLiteStoreAppendRejectsMissingFields(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	base := newResourceEvent("550e8400-e29b-41d4-a716-446655440005", 1)

	noID := base
	noID.EventID = ""
	if _, err := store.Append(context.Background(), noID); err == nil {
		t.Fatalf("expected error for empty event_id")
	}

	noAggregate := base
	noAggregate.AggregateID = ""
	if _, err := store.Append(context.Background(), noAggregate); err == nil {
		t.Fatalf("expected error for empty aggregate_id")
	}

	zeroVersion := base
	zeroVersion.EventVersion = 0
	if _, err := store.Append(context.Background(), zeroVersion); err == nil {
		t.Fatalf("expected error for zero event_version")
	}
}

// ── ReadByAggregate ─────────────────────────────────────────────────────────

func TestSQLiteStoreReadByAggregateMultipleEvents(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	aggID := "00000000-0000-0000-0000-000000000002"
	for _, v := range []int{1, 2, 3} {
		evt := newResourceEvent("", v)
		evt.AggregateID = aggID
		switch v {
		case 1:
			evt.EventID = "550e8400-e29b-41d4-a716-446655440010"
		case 2:
			evt.EventID = "550e8400-e29b-41d4-a716-446655440011"
		case 3:
			evt.EventID = "550e8400-e29b-41d4-a716-446655440012"
		}
		if _, err := store.Append(context.Background(), evt); err != nil {
			t.Fatalf("append v%d: %v", v, err)
		}
	}

	all, err := store.ReadByAggregate(context.Background(), aggID, 0, 0)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 events, got %d", len(all))
	}
	if all[0].EventVersion != 1 || all[1].EventVersion != 2 || all[2].EventVersion != 3 {
		t.Fatalf("unexpected versions: %v", []int{all[0].EventVersion, all[1].EventVersion, all[2].EventVersion})
	}

	// afterVersion filter
	after1, err := store.ReadByAggregate(context.Background(), aggID, 1, 0)
	if err != nil {
		t.Fatalf("read after v1: %v", err)
	}
	if len(after1) != 2 {
		t.Fatalf("expected 2 events after v1, got %d", len(after1))
	}

	// limit
	limited, err := store.ReadByAggregate(context.Background(), aggID, 0, 2)
	if err != nil {
		t.Fatalf("read limited: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 events with limit=2, got %d", len(limited))
	}
}

// ── ReadBySequence ───────────────────────────────────────────────────────────

func TestSQLiteStoreReadBySequence(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	ids := []string{
		"550e8400-e29b-41d4-a716-446655440020",
		"550e8400-e29b-41d4-a716-446655440021",
		"550e8400-e29b-41d4-a716-446655440022",
	}
	sequences := make([]int64, len(ids))
	for i, id := range ids {
		evt := newResourceEvent(id, i+1)
		result, err := store.Append(context.Background(), evt)
		if err != nil {
			t.Fatalf("append %s: %v", id, err)
		}
		sequences[i] = result.Sequence
	}

	after0, err := store.ReadBySequence(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("read by sequence: %v", err)
	}
	if len(after0) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(after0))
	}

	afterFirst, err := store.ReadBySequence(context.Background(), sequences[0], 0)
	if err != nil {
		t.Fatalf("read after first sequence: %v", err)
	}
	for _, e := range afterFirst {
		if e.Sequence <= sequences[0] {
			t.Fatalf("expected sequence > %d, got %d", sequences[0], e.Sequence)
		}
	}

	limited, err := store.ReadBySequence(context.Background(), 0, 2)
	if err != nil {
		t.Fatalf("read limited by sequence: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 events with limit=2, got %d", len(limited))
	}
}

// ── Snapshot ─────────────────────────────────────────────────────────────────

func TestSQLiteStoreSnapshot(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	snap := Snapshot{
		AggregateID:     "00000000-0000-0000-0000-000000000003",
		AggregateType:   "resource",
		SnapshotVersion: 5,
		Payload:         json.RawMessage(`{"title":"snapped"}`),
	}

	if err := store.Snapshot(context.Background(), snap); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	// Upsert with updated payload — should not error.
	snap.Payload = json.RawMessage(`{"title":"updated"}`)
	if err := store.Snapshot(context.Background(), snap); err != nil {
		t.Fatalf("upsert snapshot: %v", err)
	}
}

func TestSQLiteStoreSnapshotRejectsInvalidPayload(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	snap := Snapshot{
		AggregateID:     "00000000-0000-0000-0000-000000000004",
		AggregateType:   "resource",
		SnapshotVersion: 1,
		Payload:         json.RawMessage(`not-json`),
	}
	if err := store.Snapshot(context.Background(), snap); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
}

// ── Redact ───────────────────────────────────────────────────────────────────

func TestSQLiteStoreRedact(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	evtID := "550e8400-e29b-41d4-a716-446655440030"
	if _, err := store.Append(context.Background(), newResourceEvent(evtID, 1)); err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := store.Redact(context.Background(), evtID); err != nil {
		t.Fatalf("redact: %v", err)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0000-000000000001", 0, 0)
	if err != nil {
		t.Fatalf("read after redact: %v", err)
	}
	var found bool
	for _, e := range events {
		if e.EventID == evtID {
			found = true
			if !e.Redacted {
				t.Fatalf("expected redacted=true")
			}
			if string(e.Payload) != `{"redacted":true}` {
				t.Fatalf("unexpected payload: %s", e.Payload)
			}
		}
	}
	if !found {
		t.Fatalf("redacted event not found in read results")
	}

	if err := store.Redact(context.Background(), "550e8400-e29b-41d4-a716-446655440099"); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}
}

// ── WithTx ───────────────────────────────────────────────────────────────────

func TestSQLiteStoreWithTxCommit(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	evtID := "550e8400-e29b-41d4-a716-446655440040"
	var capturedSeq int64

	err := store.WithTx(context.Background(), func(tx TxStore) error {
		result, err := tx.Append(context.Background(), newResourceEvent(evtID, 1))
		if err != nil {
			return err
		}
		capturedSeq = result.Sequence
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0000-000000000001", 0, 0)
	if err != nil {
		t.Fatalf("read after WithTx: %v", err)
	}
	var found bool
	for _, e := range events {
		if e.EventID == evtID && e.Sequence == capturedSeq {
			found = true
		}
	}
	if !found {
		t.Fatalf("event not persisted after WithTx commit")
	}
}

func TestSQLiteStoreWithTxRollback(t *testing.T) {
	store, cleanup := newSQLiteStore(t)
	defer cleanup()

	evtID := "550e8400-e29b-41d4-a716-446655440041"
	sentinelErr := errors.New("deliberate rollback")

	err := store.WithTx(context.Background(), func(tx TxStore) error {
		if _, err := tx.Append(context.Background(), newResourceEvent(evtID, 1)); err != nil {
			return err
		}
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected sentinelErr, got %v", err)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0000-000000000001", 0, 0)
	if err != nil {
		t.Fatalf("read after rollback: %v", err)
	}
	for _, e := range events {
		if e.EventID == evtID {
			t.Fatalf("event should not exist after rollback")
		}
	}
}
