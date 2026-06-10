package sync

// Sync sequence replay tests.
//
// These tests verify that a client reconnecting at an arbitrary sequence offset
// receives exactly the events it missed — no more, no fewer. This covers the
// hub's in-memory history path and (where possible) the events-table durable
// replay path introduced in WS4.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"selfsystems/internal/eventstore"
)

// ── Hub in-memory replay at arbitrary offsets ────────────────────────────────

// TestReconnectReplaySinceZero verifies that a client reconnecting at sequence 0
// (first connection ever) receives all events in the hub history.
func TestReconnectReplaySinceZero(t *testing.T) {
	hub := NewHub()
	publishN(hub, 5)

	events := hub.ReplaySince(0, 0)
	if len(events) != 5 {
		t.Fatalf("want 5 events since 0, got %d", len(events))
	}
	for i, e := range events {
		if e.Sequence != int64(i+1) {
			t.Fatalf("[%d] want seq %d, got %d", i, i+1, e.Sequence)
		}
	}
}

// TestReconnectReplaySinceMidpoint verifies that a client reconnecting at
// sequence k receives only events with sequence > k.
func TestReconnectReplaySinceMidpoint(t *testing.T) {
	hub := NewHub()
	publishN(hub, 10)

	// Reconnect after having received sequences 1..5.
	events := hub.ReplaySince(5, 0)
	if len(events) != 5 {
		t.Fatalf("want 5 events since 5, got %d", len(events))
	}
	for _, e := range events {
		if e.Sequence <= 5 {
			t.Fatalf("got event with sequence %d ≤ 5", e.Sequence)
		}
	}
}

// TestReconnectReplaySinceLatest verifies that a client reconnecting at the
// latest sequence receives no events (already up-to-date).
func TestReconnectReplaySinceLatest(t *testing.T) {
	hub := NewHub()
	publishN(hub, 7)

	events := hub.ReplaySince(7, 0)
	if len(events) != 0 {
		t.Fatalf("want 0 events since latest, got %d", len(events))
	}
}

// TestReconnectReplaySinceBeyondLatest verifies that a client providing a
// sequence beyond what the hub has seen receives no events (handles clock skew
// or future-sequence scenarios).
func TestReconnectReplaySinceBeyondLatest(t *testing.T) {
	hub := NewHub()
	publishN(hub, 3)

	events := hub.ReplaySince(999, 0)
	if len(events) != 0 {
		t.Fatalf("want 0 events since 999, got %d", len(events))
	}
}

// TestReconnectReplaySinceWithLimit verifies that the limit parameter caps
// the number of replayed events even when more are available.
func TestReconnectReplaySinceWithLimit(t *testing.T) {
	hub := NewHub()
	publishN(hub, 20)

	events := hub.ReplaySince(0, 5)
	if len(events) != 5 {
		t.Fatalf("want 5 events (limited), got %d", len(events))
	}
	// Should be the first 5.
	for i, e := range events {
		if e.Sequence != int64(i+1) {
			t.Fatalf("[%d] want seq %d, got %d", i, i+1, e.Sequence)
		}
	}
}

// TestReconnectSequenceContinuity verifies that sequences assigned by the hub
// form a contiguous range with no gaps.
func TestReconnectSequenceContinuity(t *testing.T) {
	hub := NewHub()
	publishN(hub, 15)

	events := hub.ReplaySince(0, 0)
	if len(events) != 15 {
		t.Fatalf("want 15 events, got %d", len(events))
	}
	for i, e := range events {
		if e.Sequence != int64(i+1) {
			t.Fatalf("gap at index %d: expected seq %d, got %d", i, i+1, e.Sequence)
		}
	}
}

// TestReconnectExplicitSequencePreserved verifies that when events are
// published with an explicit Sequence value (e.g. from the outbox worker),
// the hub preserves that sequence rather than auto-assigning a new one.
func TestReconnectExplicitSequencePreserved(t *testing.T) {
	hub := NewHub()

	// Publish with explicit sequences mirroring an eventstore (e.g. after catch-up).
	for _, seq := range []int64{10, 20, 30} {
		hub.Publish(Event{
			Sequence:  seq,
			Type:      EventTypeResourceCreated,
			Timestamp: time.Now(),
		})
	}

	events := hub.ReplaySince(0, 0)
	if len(events) != 3 {
		t.Fatalf("want 3 events, got %d", len(events))
	}
	for i, want := range []int64{10, 20, 30} {
		if events[i].Sequence != want {
			t.Fatalf("[%d] want seq %d, got %d", i, want, events[i].Sequence)
		}
	}

	// ReplaySince(10) should return only seq 20 and 30.
	after10 := hub.ReplaySince(10, 0)
	if len(after10) != 2 {
		t.Fatalf("want 2 events since 10, got %d", len(after10))
	}
	if after10[0].Sequence != 20 || after10[1].Sequence != 30 {
		t.Fatalf("unexpected sequences: %d %d", after10[0].Sequence, after10[1].Sequence)
	}
}

// TestReconnectEventsTableReplay tests the durable replay path: events written
// to the events table are queryable via LatestSequence and ReadBySequence,
// matching what the outbox worker would deliver.
func TestReconnectEventsTableReplay(t *testing.T) {
	db, cleanup := openReconnectTestDB(t)
	defer cleanup()

	store := eventstore.NewSQLiteStore(db)
	ctx := context.Background()

	// Write 8 events.
	for i := 1; i <= 8; i++ {
		payload, _ := json.Marshal(map[string]any{"i": i})
		_, err := store.Append(ctx, eventstore.Event{
			EventID:       uuid.NewString(),
			AggregateID:   "agg-replay",
			AggregateType: "resource",
			EventType:     "ResourceCreated",
			EventVersion:  i,
			Payload:       payload,
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	latest, err := store.LatestSequence(ctx)
	if err != nil {
		t.Fatalf("LatestSequence: %v", err)
	}
	if latest != 8 {
		t.Fatalf("want latest=8, got %d", latest)
	}

	// Simulate client reconnect at various sequence offsets.
	for _, sinceSeq := range []int64{0, 3, 7} {
		events, err := store.ReadBySequence(ctx, sinceSeq, 0)
		if err != nil {
			t.Fatalf("ReadBySequence(%d): %v", sinceSeq, err)
		}
		expected := int(8 - sinceSeq)
		if len(events) != expected {
			t.Fatalf("since=%d: want %d events, got %d", sinceSeq, expected, len(events))
		}
		for _, e := range events {
			if e.Sequence <= sinceSeq {
				t.Fatalf("since=%d: got event with seq %d (not > since)", sinceSeq, e.Sequence)
			}
		}
	}
}

// TestReconnectFuzzArbitraryOffsets verifies the replay correctness across
// many random offset values using the hub's in-memory history.
func FuzzReconnectReplaySince(f *testing.F) {
	// Seeds: (totalEvents, sinceSeq)
	f.Add(10, int64(0))
	f.Add(10, int64(5))
	f.Add(10, int64(9))
	f.Add(10, int64(10))
	f.Add(10, int64(11))
	f.Add(1, int64(0))
	f.Add(50, int64(25))

	f.Fuzz(func(t *testing.T, total int, sinceSeq int64) {
		if total < 1 || total > 100 {
			t.Skip("out of range")
		}
		if sinceSeq < 0 {
			sinceSeq = 0
		}

		hub := NewHub()
		publishN(hub, total)

		events := hub.ReplaySince(sinceSeq, 0)

		expected := int64(total) - sinceSeq
		if expected < 0 {
			expected = 0
		}
		if int64(len(events)) != expected {
			t.Fatalf("total=%d since=%d: want %d events, got %d",
				total, sinceSeq, expected, len(events))
		}
		for _, e := range events {
			if e.Sequence <= sinceSeq {
				t.Fatalf("got event seq %d which is ≤ since %d", e.Sequence, sinceSeq)
			}
		}
	})
}

// ── helpers ─────────────────────────────────────────────────────────────────

// publishN publishes n events to hub with sequences 1..n.
func publishN(hub *Hub, n int) {
	for i := 1; i <= n; i++ {
		hub.Publish(Event{
			Type:      EventTypeResourceCreated,
			Payload:   fmt.Sprintf(`{"i":%d}`, i),
			Timestamp: time.Now(),
		})
	}
}

// openReconnectTestDB opens a minimal SQLite DB with just the events table.
func openReconnectTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "reconnect.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, err = db.Exec(`PRAGMA journal_mode=WAL;`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			sequence              INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id              TEXT NOT NULL UNIQUE,
			aggregate_id          TEXT NOT NULL,
			aggregate_type        TEXT NOT NULL,
			event_type            TEXT NOT NULL,
			event_version         INTEGER NOT NULL,
			payload               TEXT NOT NULL,
			payload_schema_version INTEGER NOT NULL DEFAULT 1,
			occurred_at           TEXT NOT NULL,
			recorded_at           TEXT NOT NULL,
			device_id             TEXT,
			actor_id              TEXT,
			redacted              INTEGER NOT NULL DEFAULT 0,
			correlation_id        TEXT,
			UNIQUE (aggregate_id, event_version)
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}
	return db, func() { _ = db.Close() }
}
