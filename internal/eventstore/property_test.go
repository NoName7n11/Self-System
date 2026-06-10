package eventstore

// Property-based tests for the event sourcing core invariants.
//
// These are implemented using Go's built-in fuzz testing (testing.F, Go 1.18+)
// which serves the same purpose as pgregory.net/rapid:
//   - Run as a unit test (seed corpus only): go test -run 'Fuzz' ./internal/eventstore/
//   - Run in continuous fuzz mode: go test -fuzz 'Fuzz' ./internal/eventstore/
//
// Two invariants are tested:
//   1. Version monotonicity: N appends to one aggregate → versions 1..N, no gaps.
//   2. Projection determinism: events interleaved randomly across aggregates
//      (within per-aggregate version order) produce the same final projection as
//      sequential per-aggregate application.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func openPropertyTestDB(t testing.TB) (*sql.DB, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "prop.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			sequence             INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id             TEXT NOT NULL UNIQUE,
			aggregate_id         TEXT NOT NULL,
			aggregate_type       TEXT NOT NULL,
			event_type           TEXT NOT NULL,
			event_version        INTEGER NOT NULL,
			payload              TEXT NOT NULL,
			payload_schema_version INTEGER NOT NULL DEFAULT 1,
			occurred_at          TEXT NOT NULL,
			recorded_at          TEXT NOT NULL,
			device_id            TEXT,
			actor_id             TEXT,
			redacted             INTEGER NOT NULL DEFAULT 0,
			correlation_id       TEXT,
			UNIQUE (aggregate_id, event_version)
		);
		CREATE TABLE IF NOT EXISTS projection_snapshots (
			aggregate_id      TEXT NOT NULL,
			aggregate_type    TEXT NOT NULL,
			snapshot_version  INTEGER NOT NULL,
			payload           TEXT NOT NULL,
			created_at        TEXT NOT NULL,
			PRIMARY KEY (aggregate_id, snapshot_version)
		);
		CREATE TABLE IF NOT EXISTS categories (
			id             TEXT PRIMARY KEY,
			name           TEXT NOT NULL UNIQUE,
			description    TEXT NOT NULL DEFAULT '',
			source         TEXT NOT NULL DEFAULT 'manual',
			accept_count   INTEGER NOT NULL DEFAULT 0,
			override_count INTEGER NOT NULL DEFAULT 0,
			created_at     TEXT NOT NULL,
			updated_at     TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}
	return db, func() { _ = db.Close() }
}

func makeCategoryCreatedEvent(aggID string, version int, name string) Event {
	now := time.Now().UTC()
	payload, _ := json.Marshal(CategoryCreatedPayload{
		Name:        name,
		Description: "desc-" + aggID,
		Source:      "manual",
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	return Event{
		EventID:       uuid.NewString(),
		AggregateID:   aggID,
		AggregateType: AggregateTypeCategory,
		EventType:     EventTypeCategoryCreated,
		EventVersion:  version,
		Payload:       payload,
	}
}

func makeCategoryUpdatedEvent(aggID string, version int, name string) Event {
	payload, _ := json.Marshal(CategoryUpdatedPayload{
		Name:        name,
		Description: "updated-" + aggID,
		UpdatedAt:   time.Now().UTC(),
	})
	return Event{
		EventID:       uuid.NewString(),
		AggregateID:   aggID,
		AggregateType: AggregateTypeCategory,
		EventType:     EventTypeCategoryUpdated,
		EventVersion:  version,
		Payload:       payload,
	}
}

// applyEvent appends an event AND runs the sync projector in one transaction.
func applyEventWithProjector(ctx context.Context, store Store, reg *ProjectorRegistry, evt Event) error {
	return store.WithTx(ctx, func(tx TxStore) error {
		_, err := tx.Append(ctx, evt)
		if err != nil {
			return err
		}
		return reg.ApplySync(ctx, evt, tx.Conn())
	})
}

// readCategoryState reads (name, description) for an aggregate from the projection table.
func readCategoryState(db *sql.DB, aggID string) (name, description string, found bool) {
	row := db.QueryRow(`SELECT name, description FROM categories WHERE id = ?`, aggID)
	err := row.Scan(&name, &description)
	if err != nil {
		return "", "", false
	}
	return name, description, true
}

// ── Fuzz: version monotonicity ───────────────────────────────────────────────

// FuzzEventVersionMonotonicity verifies that N sequential appends to a single
// aggregate produce event_versions 1, 2, …, N with no gaps.
//
// Corpus entries cover boundary values and typical usage.
func FuzzEventVersionMonotonicity(f *testing.F) {
	f.Add(1)
	f.Add(2)
	f.Add(5)
	f.Add(20)
	f.Add(50)

	f.Fuzz(func(t *testing.T, n int) {
		if n < 1 || n > 50 {
			t.Skip("out of range")
		}

		db, cleanup := openPropertyTestDB(t)
		defer cleanup()
		store := NewSQLiteStore(db)
		ctx := context.Background()
		aggID := uuid.NewString()

		for v := 1; v <= n; v++ {
			payload, _ := json.Marshal(map[string]any{"v": v})
			evt := Event{
				EventID:       uuid.NewString(),
				AggregateID:   aggID,
				AggregateType: "test",
				EventType:     "TestEvent",
				EventVersion:  v,
				Payload:       payload,
			}
			if _, err := store.Append(ctx, evt); err != nil {
				t.Fatalf("append v=%d: %v", v, err)
			}
		}

		events, err := store.ReadByAggregate(ctx, aggID, 0, 0)
		if err != nil {
			t.Fatalf("ReadByAggregate: %v", err)
		}
		if len(events) != n {
			t.Fatalf("expected %d events, got %d", n, len(events))
		}
		for i, e := range events {
			if e.EventVersion != i+1 {
				t.Fatalf("event[%d]: want version %d, got %d", i, i+1, e.EventVersion)
			}
		}

		// Latest sequence must be ≥ n.
		latest, err := store.LatestSequence(ctx)
		if err != nil {
			t.Fatalf("LatestSequence: %v", err)
		}
		if latest < int64(n) {
			t.Fatalf("LatestSequence %d < %d", latest, n)
		}
	})
}

// ── Fuzz: projection determinism ────────────────────────────────────────────

// FuzzProjectionDeterminism verifies that applying events from M aggregates in
// a random valid cross-aggregate interleaving (preserving per-aggregate version
// order) produces the same final projection state as sequential per-aggregate
// application.
//
// This is the core event sourcing invariant: final projection state depends
// only on each aggregate's event sequence, not the cross-aggregate order.
func FuzzProjectionDeterminism(f *testing.F) {
	// Seeds: (m=num_aggregates, k=events_per_aggregate, seed=random_seed)
	f.Add(2, 2, int64(1))
	f.Add(3, 4, int64(42))
	f.Add(5, 3, int64(99))
	f.Add(2, 1, int64(7))
	f.Add(4, 5, int64(12345))

	f.Fuzz(func(t *testing.T, m, k int, seed int64) {
		if m < 1 || m > 6 || k < 1 || k > 8 {
			t.Skip("out of range")
		}

		ctx := context.Background()

		// ── Sequential DB: apply each aggregate's events one at a time ──────
		seqDB, cleanupSeq := openPropertyTestDB(t)
		defer cleanupSeq()
		seqStore := NewSQLiteStore(seqDB)
		seqReg := NewProjectorRegistry()
		RegisterCategoryProjectors(seqReg, "sqlite")

		// ── Interleaved DB: apply same events in random valid interleaving ──
		intDB, cleanupInt := openPropertyTestDB(t)
		defer cleanupInt()
		intStore := NewSQLiteStore(intDB)
		intReg := NewProjectorRegistry()
		RegisterCategoryProjectors(intReg, "sqlite")

		// Pre-generate events for all aggregates.
		type aggPlan struct {
			aggID  string
			events []Event
		}
		plans := make([]aggPlan, m)
		for i := range plans {
			aggID := fmt.Sprintf("cat-%d-%d", seed, i)
			plans[i].aggID = aggID
			for v := 1; v <= k; v++ {
				var evt Event
				if v == 1 {
					evt = makeCategoryCreatedEvent(aggID, v, fmt.Sprintf("Cat%d-seed%d-v%d", i, seed, v))
				} else {
					evt = makeCategoryUpdatedEvent(aggID, v, fmt.Sprintf("Cat%d-seed%d-v%d", i, seed, v))
				}
				plans[i].events = append(plans[i].events, evt)
			}
		}

		// Apply sequentially: all events for agg0, then all for agg1, etc.
		for _, plan := range plans {
			for _, evt := range plan.events {
				if err := applyEventWithProjector(ctx, seqStore, seqReg, evt); err != nil {
					t.Fatalf("seq apply %s v%d: %v", plan.aggID, evt.EventVersion, err)
				}
			}
		}

		// Build a random valid interleaving using the provided seed.
		// Use a "pointer" per aggregate tracking how many events have been applied.
		pointers := make([]int, m)
		rng := rand.New(rand.NewSource(seed))
		for {
			// Find aggregates with remaining events.
			available := make([]int, 0, m)
			for i, ptr := range pointers {
				if ptr < len(plans[i].events) {
					available = append(available, i)
				}
			}
			if len(available) == 0 {
				break
			}
			// Pick one at random and apply its next event.
			pick := available[rng.Intn(len(available))]
			evt := plans[pick].events[pointers[pick]]
			pointers[pick]++
			if err := applyEventWithProjector(ctx, intStore, intReg, evt); err != nil {
				t.Fatalf("interleaved apply %s v%d: %v", plans[pick].aggID, evt.EventVersion, err)
			}
		}

		// Compare projection state for each aggregate.
		for _, plan := range plans {
			seqName, seqDesc, seqFound := readCategoryState(seqDB, plan.aggID)
			intName, intDesc, intFound := readCategoryState(intDB, plan.aggID)

			if seqFound != intFound {
				t.Fatalf("aggregate %s: seq_found=%v interleaved_found=%v",
					plan.aggID, seqFound, intFound)
			}
			if !seqFound {
				continue
			}
			if seqName != intName || seqDesc != intDesc {
				t.Fatalf("aggregate %s: projection mismatch\nseq:  name=%q desc=%q\nint:  name=%q desc=%q",
					plan.aggID, seqName, seqDesc, intName, intDesc)
			}
		}
	})
}

// ── Table-driven property checks (unit-test mode, always runs) ──────────────

// TestEventVersionMonotonicityVariants runs the version-monotonicity check
// across a fixed set of n values. This complements the fuzz test with
// deterministic coverage.
func TestEventVersionMonotonicityVariants(t *testing.T) {
	for _, n := range []int{1, 2, 5, 15, 30} {
		n := n
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			db, cleanup := openPropertyTestDB(t)
			defer cleanup()
			store := NewSQLiteStore(db)
			ctx := context.Background()
			aggID := uuid.NewString()

			for v := 1; v <= n; v++ {
				payload, _ := json.Marshal(map[string]any{"v": v})
				_, err := store.Append(ctx, Event{
					EventID:       uuid.NewString(),
					AggregateID:   aggID,
					AggregateType: "test",
					EventType:     "TestEvent",
					EventVersion:  v,
					Payload:       payload,
				})
				if err != nil {
					t.Fatalf("append v=%d: %v", v, err)
				}
			}

			events, err := store.ReadByAggregate(ctx, aggID, 0, 0)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if len(events) != n {
				t.Fatalf("want %d events, got %d", n, len(events))
			}
			for i, e := range events {
				if e.EventVersion != i+1 {
					t.Fatalf("[%d] want version %d, got %d", i, i+1, e.EventVersion)
				}
			}
		})
	}
}

// TestProjectionDeterminismSeeds runs the projection-determinism property
// across a fixed set of seeds — the same cases the fuzz test uses as corpus.
func TestProjectionDeterminismSeeds(t *testing.T) {
	cases := []struct{ m, k int; seed int64 }{
		{2, 2, 1},
		{3, 4, 42},
		{5, 3, 99},
		{2, 1, 7},
		{4, 5, 12345},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("m%d_k%d_seed%d", tc.m, tc.k, tc.seed), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			seqDB, cleanupSeq := openPropertyTestDB(t)
			defer cleanupSeq()
			seqStore := NewSQLiteStore(seqDB)
			seqReg := NewProjectorRegistry()
			RegisterCategoryProjectors(seqReg, "sqlite")

			intDB, cleanupInt := openPropertyTestDB(t)
			defer cleanupInt()
			intStore := NewSQLiteStore(intDB)
			intReg := NewProjectorRegistry()
			RegisterCategoryProjectors(intReg, "sqlite")

			type aggPlan struct {
				aggID  string
				events []Event
			}
			plans := make([]aggPlan, tc.m)
			for i := range plans {
				aggID := fmt.Sprintf("cat-%d-%d", tc.seed, i)
				plans[i].aggID = aggID
				for v := 1; v <= tc.k; v++ {
					var evt Event
					if v == 1 {
						evt = makeCategoryCreatedEvent(aggID, v, fmt.Sprintf("C%d-s%d-v%d", i, tc.seed, v))
					} else {
						evt = makeCategoryUpdatedEvent(aggID, v, fmt.Sprintf("C%d-s%d-v%d", i, tc.seed, v))
					}
					plans[i].events = append(plans[i].events, evt)
				}
			}

			for _, plan := range plans {
				for _, evt := range plan.events {
					if err := applyEventWithProjector(ctx, seqStore, seqReg, evt); err != nil {
						t.Fatalf("seq apply: %v", err)
					}
				}
			}

			pointers := make([]int, tc.m)
			rng := rand.New(rand.NewSource(tc.seed))
			for {
				available := make([]int, 0, tc.m)
				for i, ptr := range pointers {
					if ptr < len(plans[i].events) {
						available = append(available, i)
					}
				}
				if len(available) == 0 {
					break
				}
				pick := available[rng.Intn(len(available))]
				evt := plans[pick].events[pointers[pick]]
				pointers[pick]++
				if err := applyEventWithProjector(ctx, intStore, intReg, evt); err != nil {
					t.Fatalf("interleaved apply: %v", err)
				}
			}

			for _, plan := range plans {
				seqName, seqDesc, seqFound := readCategoryState(seqDB, plan.aggID)
				intName, intDesc, intFound := readCategoryState(intDB, plan.aggID)
				if seqFound != intFound {
					t.Fatalf("%s: found mismatch seq=%v int=%v", plan.aggID, seqFound, intFound)
				}
				if !seqFound {
					continue
				}
				if seqName != intName || seqDesc != intDesc {
					t.Fatalf("%s: seq name=%q desc=%q; int name=%q desc=%q",
						plan.aggID, seqName, seqDesc, intName, intDesc)
				}
			}
		})
	}
}
