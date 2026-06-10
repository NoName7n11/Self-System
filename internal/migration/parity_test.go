package migration

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"selfsystems/internal/eventstore"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

func openParityDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "parity.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ── ParityReport helpers ──────────────────────────────────────────────────────

func TestParityReportIsClean(t *testing.T) {
	r := ParityReport{Checked: 5}
	if !r.IsClean() {
		t.Fatal("empty report should be clean")
	}
	r.Divergences = append(r.Divergences, Divergence{})
	if r.IsClean() {
		t.Fatal("report with divergences should not be clean")
	}
}

// ── empty DB ─────────────────────────────────────────────────────────────────

func TestParityEmptyDB(t *testing.T) {
	db := openParityDB(t)
	store := eventstore.NewSQLiteStore(db)

	report, err := CheckResourceParity(context.Background(), db, store)
	if err != nil {
		t.Fatalf("parity check: %v", err)
	}
	if !report.IsClean() {
		t.Fatalf("expected clean report on empty DB: %s", FormatReport(report))
	}
	if report.Checked != 0 {
		t.Fatalf("expected 0 checked, got %d", report.Checked)
	}
}

// ── after backfill ────────────────────────────────────────────────────────────

func TestParityCleanAfterBackfill(t *testing.T) {
	db := openParityDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "pcat-1", "Technology")
	for i := 0; i < 5; i++ {
		insertResource(t, db, uuid.NewString(), "pcat-1")
	}

	if _, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{}); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	report, err := CheckResourceParity(context.Background(), db, store)
	if err != nil {
		t.Fatalf("parity check: %v", err)
	}
	if !report.IsClean() {
		t.Fatalf("expected clean report after backfill:\n%s", FormatReport(report))
	}
	if report.Checked != 5 {
		t.Fatalf("expected 5 checked, got %d", report.Checked)
	}
}

// ── divergence detection ──────────────────────────────────────────────────────

func TestParityDetectsDivergence(t *testing.T) {
	db := openParityDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "pcat-2", "Science")
	rid := uuid.NewString()
	insertResource(t, db, rid, "pcat-2")

	// Backfill creates event with "Test Title".
	if _, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{}); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	// Manually diverge the live projection: change the title in the DB
	// without a corresponding event (simulates a bug or direct edit).
	if _, err := db.Exec(`UPDATE resources SET title='DIVERGED' WHERE id=?`, rid); err != nil {
		t.Fatalf("update resources: %v", err)
	}

	report, err := CheckResourceParity(context.Background(), db, store)
	if err != nil {
		t.Fatalf("parity check: %v", err)
	}
	if report.IsClean() {
		t.Fatal("expected divergence to be detected")
	}
	if len(report.Divergences) == 0 {
		t.Fatal("expected at least one divergence")
	}
	found := false
	for _, d := range report.Divergences {
		if d.ResourceID == rid && d.Field == "title" {
			found = true
			if d.LiveValue != "DIVERGED" {
				t.Fatalf("expected live title=DIVERGED, got %q", d.LiveValue)
			}
			if d.EventValue != "Test Title" {
				t.Fatalf("expected event title=Test Title, got %q", d.EventValue)
			}
		}
	}
	if !found {
		t.Fatalf("expected divergence for resource %s field title", rid)
	}
}

// ── extra-in-projection ───────────────────────────────────────────────────────

func TestParityDetectsExtraInProjection(t *testing.T) {
	db := openParityDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "pcat-3", "Math")
	rid := uuid.NewString()
	insertResource(t, db, rid, "pcat-3")
	// Do NOT backfill — resource has no events.

	report, err := CheckResourceParity(context.Background(), db, store)
	if err != nil {
		t.Fatalf("parity check: %v", err)
	}
	if report.IsClean() {
		t.Fatal("expected extra-in-projection to be detected")
	}
	found := false
	for _, id := range report.ExtraInProjection {
		if id == rid {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s in ExtraInProjection", rid)
	}
}

// ── extra-in-events ───────────────────────────────────────────────────────────

func TestParityDetectsExtraInEvents(t *testing.T) {
	db := openParityDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "pcat-4", "History")
	rid := uuid.NewString()
	insertResource(t, db, rid, "pcat-4")

	// Backfill creates the ResourceImported event.
	if _, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{}); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	// Delete from projection directly (no ResourceDeleted event).
	if _, err := db.Exec(`DELETE FROM resources WHERE id=?`, rid); err != nil {
		t.Fatalf("delete from projection: %v", err)
	}

	report, err := CheckResourceParity(context.Background(), db, store)
	if err != nil {
		t.Fatalf("parity check: %v", err)
	}
	if report.IsClean() {
		t.Fatal("expected extra-in-events to be detected")
	}
	found := false
	for _, id := range report.ExtraInEvents {
		if id == rid {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s in ExtraInEvents", rid)
	}
}

// ── ResourceDeleted in events removes from rebuilt projection ─────────────────

func TestParityDeletedNotInRebuilt(t *testing.T) {
	db := openParityDB(t)
	store := eventstore.NewSQLiteStore(db)

	insertCategory(t, db, "pcat-5", "Tools")
	rid := uuid.NewString()
	insertResource(t, db, rid, "pcat-5")

	// Backfill to create version=1 event.
	if _, err := RunResourceBackfill(context.Background(), db, store, BackfillConfig{}); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	// Append a ResourceDeleted event (version=2).
	if _, err := store.Append(context.Background(), eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   rid,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceDeleted,
		EventVersion:  2,
		Payload:       []byte(`{"id":"` + rid + `"}`),
	}); err != nil {
		t.Fatalf("append deleted event: %v", err)
	}

	// Also delete from projection to keep it consistent.
	if _, err := db.Exec(`DELETE FROM resources WHERE id=?`, rid); err != nil {
		t.Fatalf("delete from projection: %v", err)
	}

	// Parity should be clean: deleted resource is absent from both.
	report, err := CheckResourceParity(context.Background(), db, store)
	if err != nil {
		t.Fatalf("parity check: %v", err)
	}
	if !report.IsClean() {
		t.Fatalf("expected clean report when deleted resource absent from both:\n%s", FormatReport(report))
	}
}

// ── FormatReport ─────────────────────────────────────────────────────────────

func TestFormatReportPass(t *testing.T) {
	r := ParityReport{Checked: 10}
	out := FormatReport(r)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	// Should contain "PASS"
	if len(out) < 4 || out[:4] != "PASS" {
		t.Fatalf("expected PASS prefix, got: %q", out)
	}
}

func TestFormatReportFail(t *testing.T) {
	r := ParityReport{
		Checked:     3,
		Divergences: []Divergence{{ResourceID: "abc", Field: "title", LiveValue: "A", EventValue: "B"}},
	}
	out := FormatReport(r)
	if len(out) < 4 || out[:4] != "FAIL" {
		t.Fatalf("expected FAIL prefix, got: %q", out)
	}
}
