package migration

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"selfsystems/internal/eventstore"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

// ── test helpers ─────────────────────────────────────────────────────────────

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func insertCategory(t *testing.T, db *sql.DB, id, name string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO categories (id, name, description, source, created_at, updated_at)
		VALUES (?, ?, '', 'manual', ?, ?)
	`, id, name, now, now); err != nil {
		t.Fatalf("insert category: %v", err)
	}
}

// insertResource inserts directly into the resources state table (simulates
// pre-event-sourcing data that needs backfilling).
func insertResource(t *testing.T, db *sql.DB, id, categoryID string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO resources (id, url, host, title, summary, category_id, user_override, created_at, updated_at)
		VALUES (?, ?, 'example.com', 'Test Title', 'Test Summary', ?, 0, ?, ?)
	`, id, "https://example.com/"+id, categoryID, now, now); err != nil {
		t.Fatalf("insert resource: %v", err)
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestBackfillEmptyDB(t *testing.T) {
	db := openDB(t)
	store := eventstore.NewSQLiteStore(db)

	result, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.Processed != 0 {
		t.Fatalf("expected 0 processed, got %d", result.Processed)
	}
	if result.Skipped != 0 {
		t.Fatalf("expected 0 skipped, got %d", result.Skipped)
	}
	if result.CorrelationID == "" {
		t.Fatal("expected non-empty correlation_id")
	}
}

func TestBackfillSeeds(t *testing.T) {
	db := openDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "cat-1", "Tech")
	r1 := uuid.NewString()
	r2 := uuid.NewString()
	insertResource(t, db, r1, "cat-1")
	insertResource(t, db, r2, "cat-1")

	result, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.Processed != 2 {
		t.Fatalf("expected 2 processed, got %d", result.Processed)
	}
	if result.Skipped != 0 {
		t.Fatalf("expected 0 skipped, got %d", result.Skipped)
	}

	// Verify events exist.
	for _, id := range []string{r1, r2} {
		events, err := store.ReadByAggregate(context.Background(), id, 0, 0)
		if err != nil {
			t.Fatalf("ReadByAggregate %s: %v", id, err)
		}
		if len(events) != 1 {
			t.Fatalf("expected 1 event for %s, got %d", id, len(events))
		}
		if events[0].EventType != eventstore.EventTypeResourceImported {
			t.Fatalf("expected EventTypeResourceImported, got %s", events[0].EventType)
		}
		if events[0].EventVersion != 1 {
			t.Fatalf("expected event_version=1, got %d", events[0].EventVersion)
		}
		if events[0].CorrelationID == nil || *events[0].CorrelationID != result.CorrelationID {
			t.Fatalf("expected correlation_id=%s", result.CorrelationID)
		}
	}
}

func TestBackfillSkipsAlreadyEvented(t *testing.T) {
	db := openDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "cat-2", "Science")
	r1 := uuid.NewString()
	r2 := uuid.NewString()
	insertResource(t, db, r1, "cat-2")
	insertResource(t, db, r2, "cat-2")

	// Seed r1 with a ResourceCreated event (simulates event-sourced create).
	_, err := store.Append(context.Background(), eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   r1,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceCreated,
		EventVersion:  1,
		Payload:       []byte(`{"url":"https://example.com/r1","host":"example.com","title":"R1","summary":"","category_id":"cat-2","category_name":"Science","user_override":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`),
	})
	if err != nil {
		t.Fatalf("seed event for r1: %v", err)
	}

	result, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("expected 1 processed (r2 only), got %d", result.Processed)
	}
	if result.Skipped != 1 {
		t.Fatalf("expected 1 skipped (r1), got %d", result.Skipped)
	}

	// r1 should still have exactly 1 event (ResourceCreated, not ResourceImported).
	events, _ := store.ReadByAggregate(context.Background(), r1, 0, 0)
	if len(events) != 1 || events[0].EventType != eventstore.EventTypeResourceCreated {
		t.Fatalf("r1 events incorrect: %v", events)
	}
}

func TestBackfillIdempotent(t *testing.T) {
	db := openDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "cat-3", "Arts")
	r1 := uuid.NewString()
	insertResource(t, db, r1, "cat-3")

	// First run.
	r1first, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{})
	if err != nil {
		t.Fatalf("first backfill: %v", err)
	}
	if r1first.Processed != 1 {
		t.Fatalf("expected 1 processed on first run, got %d", r1first.Processed)
	}

	// Second run — r1 already has an event, so it should be skipped.
	r1second, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{})
	if err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if r1second.Processed != 0 {
		t.Fatalf("expected 0 processed on second run, got %d", r1second.Processed)
	}
	if r1second.Skipped != 1 {
		t.Fatalf("expected 1 skipped on second run, got %d", r1second.Skipped)
	}

	// Still only one event for r1.
	events, _ := store.ReadByAggregate(context.Background(), r1, 0, 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 event after two runs, got %d", len(events))
	}
}

func TestBackfillCorrelationID(t *testing.T) {
	db := openDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "cat-4", "Food")
	for i := 0; i < 3; i++ {
		insertResource(t, db, uuid.NewString(), "cat-4")
	}

	correlationID := "c0rr3lat10n-0000-0000-0000-000000000001"
	result, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{
		CorrelationID: correlationID,
	})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.CorrelationID != correlationID {
		t.Fatalf("expected correlation_id=%s, got %s", correlationID, result.CorrelationID)
	}

	// Verify all events share the correlation_id.
	allEvents, err := store.ReadBySequence(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("ReadBySequence: %v", err)
	}
	for _, e := range allEvents {
		if e.CorrelationID == nil || *e.CorrelationID != correlationID {
			t.Fatalf("event %s missing correlation_id", e.EventID)
		}
	}
}

func TestBackfillBatchSize(t *testing.T) {
	db := openDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "cat-5", "Misc")
	n := 25
	for i := 0; i < n; i++ {
		insertResource(t, db, uuid.NewString(), "cat-5")
	}

	result, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{
		BatchSize: 7, // forces multiple batches
	})
	if err != nil {
		t.Fatalf("backfill with batch_size=7: %v", err)
	}
	if result.Processed != n {
		t.Fatalf("expected %d processed, got %d", n, result.Processed)
	}
}

func TestBackfillProgressCallback(t *testing.T) {
	db := openDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "cat-6", "Progress")
	for i := 0; i < 10; i++ {
		insertResource(t, db, uuid.NewString(), "cat-6")
	}

	var callCount, lastProcessed int
	_, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{
		BatchSize: 3,
		OnProgress: func(processed, total int) {
			callCount++
			lastProcessed = processed
			if processed > total {
				t.Errorf("processed %d > total %d", processed, total)
			}
		},
	})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if callCount == 0 {
		t.Fatal("expected OnProgress to be called")
	}
	if lastProcessed != 10 {
		t.Fatalf("expected lastProcessed=10, got %d", lastProcessed)
	}
}

// ── benchmark ─────────────────────────────────────────────────────────────────

// BenchmarkBackfill100K seeds 100K resources and times the backfill.
// Run with: go test -run=^$ -bench=BenchmarkBackfill100K -benchtime=1x ./internal/migration/...
// Performance gate: must complete in under 30 minutes (1800s).
func BenchmarkBackfill100K(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 100K backfill benchmark in short mode")
	}

	db, err := sqliterepo.Open(filepath.Join(b.TempDir(), "bench.db"))
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Seed one category.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO categories (id, name, description, source, created_at, updated_at) VALUES ('bench-cat', 'Bench', '', 'manual', ?, ?)`, now, now); err != nil {
		b.Fatalf("seed category: %v", err)
	}

	// Batch-insert 100K resources for speed.
	const n = 100_000
	b.Logf("seeding %d resources...", n)
	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin tx: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO resources (id, url, host, title, summary, category_id, user_override, created_at, updated_at) VALUES (?, ?, 'example.com', 'Title', '', 'bench-cat', 0, ?, ?)`)
	if err != nil {
		b.Fatalf("prepare: %v", err)
	}
	for i := 0; i < n; i++ {
		if _, err := stmt.Exec(uuid.NewString(), fmt.Sprintf("https://example.com/%d", i), now, now); err != nil {
			b.Fatalf("insert resource %d: %v", i, err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit seed: %v", err)
	}
	b.Logf("seeded %d resources, running backfill...", n)

	store := eventstore.NewSQLiteStore(db)
	b.ResetTimer()

	result, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{
		BatchSize: 500,
		OnProgress: func(processed, total int) {
			if processed%10000 == 0 {
				b.Logf("  progress: %d/%d", processed, total)
			}
		},
	})
	b.StopTimer()

	if err != nil {
		b.Fatalf("backfill: %v", err)
	}
	if result.Processed != n {
		b.Fatalf("expected %d processed, got %d", n, result.Processed)
	}

	const budget = 30 * time.Minute
	b.Logf("backfilled %d resources in %s (budget: %s)", n, result.Duration, budget)
	if result.Duration > budget {
		b.Errorf("FAIL: backfill took %s, exceeds %s budget", result.Duration, budget)
	}
}
