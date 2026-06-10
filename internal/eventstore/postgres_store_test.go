package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

// postgresEventstoreDSN skips the test when no Postgres DSN is configured,
// matching the convention in internal/repository/postgres.
func postgresEventstoreDSN(t *testing.T) string {
	t.Helper()
	for _, key := range []string{"SS_POSTGRES_TEST_DSN", "SS_DATABASE_URL"} {
		if dsn := strings.TrimSpace(os.Getenv(key)); dsn != "" {
			return dsn
		}
	}
	t.Skip("set SS_POSTGRES_TEST_DSN or SS_DATABASE_URL to run postgres eventstore tests")
	return ""
}

func newPostgresStore(t *testing.T) (*PostgresStore, func()) {
	t.Helper()
	dsn := postgresEventstoreDSN(t)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres db: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Fatalf("ping postgres: %v", err)
	}
	resetPostgresEventTables(t, db)
	return NewPostgresStore(db), func() { _ = db.Close() }
}

func resetPostgresEventTables(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE TABLE projection_snapshots, events RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate event tables: %v", err)
	}
}

func newPGResourceEvent(eventID string, version int) Event {
	return Event{
		EventID:              eventID,
		AggregateID:          "00000000-0000-0000-0001-000000000001",
		AggregateType:        "resource",
		EventType:            "ResourceCreated",
		EventVersion:         version,
		Payload:              json.RawMessage(`{"summary":"pg-test"}`),
		PayloadSchemaVersion: 1,
	}
}

// ── Append ──────────────────────────────────────────────────────────────────

func TestPostgresStoreAppendAndRead(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	evtID := "660e8400-e29b-41d4-a716-446655440000"
	result, err := store.Append(context.Background(), newPGResourceEvent(evtID, 1))
	if err != nil {
		t.Fatalf("append event: %v", err)
	}
	if !result.Applied {
		t.Fatalf("expected applied=true")
	}
	if result.Sequence <= 0 {
		t.Fatalf("expected sequence > 0, got %d", result.Sequence)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0001-000000000001", 0, 0)
	if err != nil {
		t.Fatalf("read by aggregate: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventID != evtID {
		t.Fatalf("unexpected event_id: %s", events[0].EventID)
	}
}

func TestPostgresStoreAppendIdempotent(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	evtID := "660e8400-e29b-41d4-a716-446655440001"
	first, err := store.Append(context.Background(), newPGResourceEvent(evtID, 1))
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	second, err := store.Append(context.Background(), newPGResourceEvent(evtID, 1))
	if err != nil {
		t.Fatalf("append duplicate: %v", err)
	}
	if second.Applied {
		t.Fatalf("expected applied=false for duplicate")
	}
	if second.Sequence != first.Sequence {
		t.Fatalf("expected same sequence, got %d vs %d", first.Sequence, second.Sequence)
	}
}

func TestPostgresStoreAppendConcurrencyConflict(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	_, err := store.Append(context.Background(), newPGResourceEvent("660e8400-e29b-41d4-a716-446655440002", 1))
	if err != nil {
		t.Fatalf("first append: %v", err)
	}

	_, err = store.Append(context.Background(), newPGResourceEvent("660e8400-e29b-41d4-a716-446655440003", 1))
	if !errors.Is(err, ErrConcurrencyConflict) {
		t.Fatalf("expected ErrConcurrencyConflict, got %v", err)
	}
}

func TestPostgresStoreAppendRejectsInvalidPayload(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	evt := newPGResourceEvent("660e8400-e29b-41d4-a716-446655440004", 1)
	evt.Payload = nil
	if _, err := store.Append(context.Background(), evt); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload for nil payload, got %v", err)
	}

	evt.Payload = json.RawMessage(`not-json`)
	if _, err := store.Append(context.Background(), evt); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload for malformed JSON, got %v", err)
	}
}

// ── ReadBySequence ───────────────────────────────────────────────────────────

func TestPostgresStoreReadBySequence(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	ids := []string{
		"660e8400-e29b-41d4-a716-446655440010",
		"660e8400-e29b-41d4-a716-446655440011",
		"660e8400-e29b-41d4-a716-446655440012",
	}
	sequences := make([]int64, len(ids))
	for i, id := range ids {
		evt := newPGResourceEvent(id, i+1)
		r, err := store.Append(context.Background(), evt)
		if err != nil {
			t.Fatalf("append %s: %v", id, err)
		}
		sequences[i] = r.Sequence
	}

	after0, err := store.ReadBySequence(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("read by sequence: %v", err)
	}
	if len(after0) < 3 {
		t.Fatalf("expected >= 3 events, got %d", len(after0))
	}

	limited, err := store.ReadBySequence(context.Background(), 0, 2)
	if err != nil {
		t.Fatalf("read limited: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 events with limit=2, got %d", len(limited))
	}
}

// ── Snapshot ─────────────────────────────────────────────────────────────────

func TestPostgresStoreSnapshot(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	snap := Snapshot{
		AggregateID:     "00000000-0000-0000-0001-000000000002",
		AggregateType:   "resource",
		SnapshotVersion: 5,
		Payload:         json.RawMessage(`{"title":"pg-snap"}`),
	}
	if err := store.Snapshot(context.Background(), snap); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	snap.Payload = json.RawMessage(`{"title":"pg-snap-updated"}`)
	if err := store.Snapshot(context.Background(), snap); err != nil {
		t.Fatalf("upsert snapshot: %v", err)
	}
}

// ── Redact ───────────────────────────────────────────────────────────────────

func TestPostgresStoreRedact(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	evtID := "660e8400-e29b-41d4-a716-446655440020"
	if _, err := store.Append(context.Background(), newPGResourceEvent(evtID, 1)); err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := store.Redact(context.Background(), evtID); err != nil {
		t.Fatalf("redact: %v", err)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0001-000000000001", 0, 0)
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
		}
	}
	if !found {
		t.Fatalf("redacted event not found")
	}

	if err := store.Redact(context.Background(), "660e8400-e29b-41d4-a716-446655440099"); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}
}

// ── WithTx ───────────────────────────────────────────────────────────────────

func TestPostgresStoreWithTxCommit(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	evtID := "660e8400-e29b-41d4-a716-446655440030"
	var capturedSeq int64

	err := store.WithTx(context.Background(), func(tx TxStore) error {
		result, err := tx.Append(context.Background(), newPGResourceEvent(evtID, 1))
		if err != nil {
			return err
		}
		capturedSeq = result.Sequence
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0001-000000000001", 0, 0)
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

func TestPostgresStoreWithTxRollback(t *testing.T) {
	store, cleanup := newPostgresStore(t)
	defer cleanup()

	evtID := "660e8400-e29b-41d4-a716-446655440031"
	sentinelErr := errors.New("deliberate rollback")

	err := store.WithTx(context.Background(), func(tx TxStore) error {
		if _, err := tx.Append(context.Background(), newPGResourceEvent(evtID, 1)); err != nil {
			return err
		}
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected sentinelErr, got %v", err)
	}

	events, err := store.ReadByAggregate(context.Background(), "00000000-0000-0000-0001-000000000001", 0, 0)
	if err != nil {
		t.Fatalf("read after rollback: %v", err)
	}
	for _, e := range events {
		if e.EventID == evtID {
			t.Fatalf("event should not exist after rollback")
		}
	}
}
