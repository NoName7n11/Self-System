// Rollback drill script for the event sourcing migration (Change 3 WS7).
//
// This program exercises the feature-flag rollback procedure documented in
// DEPLOYMENT.md Section 8 without touching any production data. It:
//
//   Phase 1 — Flag ON:  creates resources via the event-sourced write path
//             and verifies events appear in the events table and the projection
//             table is populated.
//
//   Phase 2 — Flag OFF: creates resources via the direct-write path and
//             verifies NO new events are appended for those resources.
//
//   Phase 3 — Parity: runs CheckResourceParity and verifies that the Phase 1
//             resources are clean (parity) while Phase 2 resources appear as
//             "extra in projection" (no backing events), confirming the drill
//             matches expected post-rollback state.
//
// Exit code 0 = drill passed.  Exit code 1 = drill failed (reason printed).
//
// Usage:
//
//	go run ./scripts/rollback_drill
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
	"selfsystems/internal/migration"
	sqliterepo "selfsystems/internal/repository/sqlite"
	"selfsystems/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "rollback drill FAILED:", err)
		os.Exit(1)
	}
	fmt.Println("rollback drill PASSED")
}

func run() error {
	ctx := context.Background()

	// Open a temporary SQLite database — no production data is touched.
	tmpDir, err := os.MkdirTemp("", "rollback-drill-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := sqliterepo.Open(filepath.Join(tmpDir, "drill.db"))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	store := eventstore.NewSQLiteStore(db)
	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)

	// ── Phase 1: events_resource_enabled = true ───────────────────────────

	fmt.Println("Phase 1: flag ON — event-sourced writes")

	flagOnSvc, err := buildEventSourcingService(db, store, categoryRepo, resourceRepo)
	if err != nil {
		return fmt.Errorf("build event-sourced service: %w", err)
	}

	var phase1IDs []string
	for i := 1; i <= 3; i++ {
		r, createErr := flagOnSvc.Create(ctx, service.CreateResourceInput{
			URL:          fmt.Sprintf("https://drill.example.com/phase1/%d", i),
			Title:        fmt.Sprintf("Phase1 Resource %d", i),
			CategoryName: "drill-category",
		})
		if createErr != nil {
			return fmt.Errorf("phase1 create %d: %w", i, createErr)
		}
		phase1IDs = append(phase1IDs, r.ID)
		fmt.Printf("  created %s (flag ON)\n", r.ID)
	}

	// Verify each Phase 1 resource has at least one event.
	for _, id := range phase1IDs {
		events, readErr := store.ReadByAggregate(ctx, id, 0, 0)
		if readErr != nil {
			return fmt.Errorf("read events for %s: %w", id, readErr)
		}
		if len(events) == 0 {
			return fmt.Errorf("Phase 1 FAIL: resource %s has no events (expected ≥1)", id)
		}
		fmt.Printf("  verified %s: %d event(s) in event log\n", id, len(events))
	}

	// ── Phase 2: events_resource_enabled = false ──────────────────────────

	fmt.Println("Phase 2: flag OFF — direct-write path")

	// Build a service WITHOUT event sourcing (simulates flag flip + restart).
	// catSvc is still needed for EnsureByName on the category path.
	flagOffCatSvc := service.NewCategoryService(categoryRepo)
	flagOffSvc := service.NewResourceService(resourceRepo, categoryRepo, nil, flagOffCatSvc)

	var phase2IDs []string
	for i := 1; i <= 3; i++ {
		r, createErr := flagOffSvc.Create(ctx, service.CreateResourceInput{
			URL:          fmt.Sprintf("https://drill.example.com/phase2/%d", i),
			Title:        fmt.Sprintf("Phase2 Resource %d", i),
			CategoryName: "drill-category",
		})
		if createErr != nil {
			return fmt.Errorf("phase2 create %d: %w", i, createErr)
		}
		phase2IDs = append(phase2IDs, r.ID)
		fmt.Printf("  created %s (flag OFF)\n", r.ID)
	}

	// Verify Phase 2 resources have NO events.
	for _, id := range phase2IDs {
		events, readErr := store.ReadByAggregate(ctx, id, 0, 0)
		if readErr != nil {
			return fmt.Errorf("read events for %s: %w", id, readErr)
		}
		if len(events) != 0 {
			return fmt.Errorf("Phase 2 FAIL: resource %s has %d events (expected 0)", id, len(events))
		}
		fmt.Printf("  verified %s: 0 events (correct — flag was OFF)\n", id)
	}

	// ── Phase 3: Parity check ─────────────────────────────────────────────

	fmt.Println("Phase 3: parity check")

	report, parityErr := migration.CheckResourceParity(ctx, db, store)
	if parityErr != nil {
		return fmt.Errorf("parity check: %w", parityErr)
	}

	fmt.Println(migration.FormatReport(report))

	// Phase 1 resources should be in both projection and events → 0 divergences.
	// Phase 2 resources should be in projection only → ExtraInProjection.
	if report.Checked == 0 {
		return fmt.Errorf("parity: expected >0 aggregates checked")
	}
	if len(report.Divergences) > 0 {
		return fmt.Errorf("parity: unexpected field-level divergences: %v", report.Divergences)
	}
	if len(report.ExtraInProjection) != len(phase2IDs) {
		return fmt.Errorf("parity: expected %d extra-in-projection rows (Phase 2), got %d",
			len(phase2IDs), len(report.ExtraInProjection))
	}
	// All Phase 2 IDs should appear in ExtraInProjection.
	extraSet := make(map[string]bool, len(report.ExtraInProjection))
	for _, id := range report.ExtraInProjection {
		extraSet[id] = true
	}
	for _, id := range phase2IDs {
		if !extraSet[id] {
			return fmt.Errorf("parity: Phase 2 resource %s not in ExtraInProjection", id)
		}
	}

	fmt.Printf("  Phase 1 (%d resources): all in event log ✓\n", len(phase1IDs))
	fmt.Printf("  Phase 2 (%d resources): projection-only, no events ✓ (matches post-rollback state)\n", len(phase2IDs))
	fmt.Println("  Parity check confirmed rollback behavior is correct.")

	return nil
}

// buildEventSourcingService constructs a ResourceService with event sourcing
// enabled — identical to how main.go wires it when events_resource_enabled=true.
func buildEventSourcingService(
	db *sql.DB,
	store eventstore.Store,
	categoryRepo domain.CategoryRepository,
	resourceRepo domain.ResourceRepository,
) (*service.ResourceService, error) {
	registry := eventstore.NewProjectorRegistry()
	eventstore.RegisterResourceProjectors(registry, "sqlite")

	// Seed a category for FK integrity (resources require a category or can be
	// created without one — use the no-category path for simplicity).
	catSvc := service.NewCategoryService(categoryRepo)

	// Build the classifier with nil AI manager (drill doesn't need AI).
	classifier := service.NewCategoryClassifier(categoryRepo, nil)
	_ = classifier

	svc := service.NewResourceService(
		resourceRepo,
		categoryRepo,
		nil, // no classifier needed — we're not adding categories
		catSvc,
		service.WithEventSourcing(store, registry),
	)
	return svc, nil
}

func init() {
	_ = uuid.NewString // ensure import is used
}
