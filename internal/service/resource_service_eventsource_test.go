package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

// newTestDB opens a fresh SQLite DB in a temp dir with all migrations applied.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedCategory inserts a category directly and returns it.
func seedCategory(t *testing.T, db *sql.DB, id, name string) domain.Category {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO categories (id, name, description, source, created_at, updated_at)
		VALUES (?, ?, '', 'manual', ?, ?)
	`, id, name, now, now)
	if err != nil {
		t.Fatalf("seed category: %v", err)
	}
	return domain.Category{ID: id, Name: name, Source: domain.CategorySourceManual}
}

// newEventSourcedResourceService wires up a ResourceService backed by a real
// SQLite event store and SQLite resource/category repos, with events enabled.
func newEventSourcedResourceService(t *testing.T, db *sql.DB) *ResourceService {
	t.Helper()
	store := eventstore.NewSQLiteStore(db)
	registry := eventstore.NewProjectorRegistry()
	eventstore.RegisterResourceProjectors(registry, "sqlite")

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)

	// Minimal classifier that returns the first category from the repo.
	classifier := NewCategoryClassifier(categoryRepo, nil)

	catSvc := NewCategoryService(categoryRepo)

	return NewResourceService(
		resourceRepo,
		categoryRepo,
		classifier,
		catSvc,
		WithEventSourcing(store, registry),
	)
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestResourceServiceEventSourcedCreate(t *testing.T) {
	db := newTestDB(t)
	cat := seedCategory(t, db, "cat-1", "Technology")
	svc := newEventSourcedResourceService(t, db)
	store := eventstore.NewSQLiteStore(db)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:        "https://example.com/article",
		Title:      "Event Sourcing 101",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if resource.ID == "" {
		t.Fatal("expected non-empty resource ID")
	}
	if resource.CategoryID != cat.ID {
		t.Fatalf("expected category_id=%s, got %s", cat.ID, resource.CategoryID)
	}

	// Event must exist in the event store.
	events, err := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	if err != nil {
		t.Fatalf("ReadByAggregate: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	evt := events[0]
	if evt.EventType != eventstore.EventTypeResourceCreated {
		t.Fatalf("expected EventTypeResourceCreated, got %s", evt.EventType)
	}
	if evt.EventVersion != 1 {
		t.Fatalf("expected event_version=1, got %d", evt.EventVersion)
	}

	// Projection (resources table) must be populated.
	repo := sqliterepo.NewResourceRepository(db)
	got, err := repo.GetByID(context.Background(), resource.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected resource in projection table")
	}
	if got.Title != "Event Sourcing 101" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
	if got.CategoryID != cat.ID {
		t.Fatalf("unexpected category_id: %s", got.CategoryID)
	}
}

func TestResourceServiceEventSourcedCreatePayload(t *testing.T) {
	db := newTestDB(t)
	cat := seedCategory(t, db, "cat-2", "AI")
	svc := newEventSourcedResourceService(t, db)
	store := eventstore.NewSQLiteStore(db)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:        "https://example.com/ai",
		Title:      "LLM Guide",
		Summary:    "How LLMs work",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	events, _ := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	var payload eventstore.ResourceCreatedPayload
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Title != "LLM Guide" {
		t.Fatalf("unexpected payload title: %s", payload.Title)
	}
	if payload.Summary != "How LLMs work" {
		t.Fatalf("unexpected payload summary: %s", payload.Summary)
	}
	if payload.CategoryID != cat.ID {
		t.Fatalf("unexpected payload category_id: %s", payload.CategoryID)
	}
}

// ── Update ───────────────────────────────────────────────────────────────────

func TestResourceServiceEventSourcedUpdate(t *testing.T) {
	db := newTestDB(t)
	cat := seedCategory(t, db, "cat-3", "Research")
	svc := newEventSourcedResourceService(t, db)
	store := eventstore.NewSQLiteStore(db)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:        "https://example.com/research",
		Title:      "Original Title",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := svc.Update(context.Background(), UpdateResourceInput{
		ID:         resource.ID,
		URL:        resource.URL,
		Title:      "Updated Title",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Fatalf("unexpected title: %s", updated.Title)
	}

	// Two events must exist: ResourceCreated + ResourceUpdated.
	events, err := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	if err != nil {
		t.Fatalf("ReadByAggregate: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].EventType != eventstore.EventTypeResourceUpdated {
		t.Fatalf("expected EventTypeResourceUpdated, got %s", events[1].EventType)
	}
	if events[1].EventVersion != 2 {
		t.Fatalf("expected event_version=2, got %d", events[1].EventVersion)
	}

	// Projection must reflect the update.
	repo := sqliterepo.NewResourceRepository(db)
	got, _ := repo.GetByID(context.Background(), resource.ID)
	if got.Title != "Updated Title" {
		t.Fatalf("projection not updated: %s", got.Title)
	}
}

// ── Delete ───────────────────────────────────────────────────────────────────

func TestResourceServiceEventSourcedDelete(t *testing.T) {
	db := newTestDB(t)
	cat := seedCategory(t, db, "cat-4", "Tools")
	svc := newEventSourcedResourceService(t, db)
	store := eventstore.NewSQLiteStore(db)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:        "https://example.com/tool",
		Title:      "Tool Article",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	deleted, err := svc.Delete(context.Background(), resource.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}

	// ResourceDeleted event must exist.
	events, _ := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].EventType != eventstore.EventTypeResourceDeleted {
		t.Fatalf("expected EventTypeResourceDeleted, got %s", events[1].EventType)
	}

	// Projection must be gone.
	repo := sqliterepo.NewResourceRepository(db)
	got, err := repo.GetByID(context.Background(), resource.ID)
	if err != nil {
		t.Fatalf("GetByID after delete: %v", err)
	}
	if got != nil {
		t.Fatal("expected resource removed from projection table")
	}
}

// ── Flag OFF compatibility ────────────────────────────────────────────────────

func TestResourceServiceFlagOffUsesDirectRepo(t *testing.T) {
	db := newTestDB(t)
	cat := seedCategory(t, db, "cat-5", "Misc")
	store := eventstore.NewSQLiteStore(db)

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	classifier := NewCategoryClassifier(categoryRepo, nil)
	catSvc := NewCategoryService(categoryRepo)

	// Events NOT enabled — use plain constructor with no options.
	svc := NewResourceService(resourceRepo, categoryRepo, classifier, catSvc)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:        "https://example.com/misc",
		Title:      "Misc Article",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Resource must exist in the state table.
	repo := sqliterepo.NewResourceRepository(db)
	got, err := repo.GetByID(context.Background(), resource.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected resource in state table")
	}

	// Event store must be empty (no events written).
	events, err := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	if err != nil {
		t.Fatalf("ReadByAggregate: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events with flag OFF, got %d", len(events))
	}
}

// ── OCC retry ────────────────────────────────────────────────────────────────

func TestResourceServiceOCCRetryOnConcurrentUpdate(t *testing.T) {
	db := newTestDB(t)
	cat := seedCategory(t, db, "cat-6", "Science")
	svc := newEventSourcedResourceService(t, db)
	store := eventstore.NewSQLiteStore(db)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:        "https://example.com/science",
		Title:      "Science Article",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Manually inject a version-2 event to simulate a concurrent write
	// that lands between our latestResourceVersion read and our append.
	ghostEvt := eventstore.Event{
		EventID:       "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		AggregateID:   resource.ID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceUpdated,
		EventVersion:  2,
		Payload:       json.RawMessage(`{"url":"https://example.com/science","host":"example.com","title":"Concurrent Write","summary":"","category_id":"cat-6","category_name":"Science","user_override":false,"updated_at":"2026-01-01T00:00:00Z"}`),
	}
	if _, err := store.Append(context.Background(), ghostEvt); err != nil {
		t.Fatalf("inject concurrent event: %v", err)
	}

	// Now do an update. The service will try version=2, hit OCC, re-read (max=2),
	// and retry with version=3 — which should succeed.
	updated, err := svc.Update(context.Background(), UpdateResourceInput{
		ID:         resource.ID,
		URL:        resource.URL,
		Title:      "Retry Succeeded",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Update with OCC retry: %v", err)
	}
	if updated.Title != "Retry Succeeded" {
		t.Fatalf("unexpected title: %s", updated.Title)
	}

	// Three events: Created (v1), ghost (v2), Updated via retry (v3).
	events, _ := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[2].EventVersion != 3 {
		t.Fatalf("expected event_version=3, got %d", events[2].EventVersion)
	}

	// Projection must reflect the retried update.
	repo := sqliterepo.NewResourceRepository(db)
	got, _ := repo.GetByID(context.Background(), resource.ID)
	if got.Title != "Retry Succeeded" {
		t.Fatalf("projection not updated after retry: %s", got.Title)
	}
}

// ── idempotency guard ─────────────────────────────────────────────────────────

func TestResourceServiceEventSourcedIdempotentEventID(t *testing.T) {
	db := newTestDB(t)
	cat := seedCategory(t, db, "cat-7", "Dev")
	svc := newEventSourcedResourceService(t, db)
	store := eventstore.NewSQLiteStore(db)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:        "https://example.com/dev",
		Title:      "Dev Article",
		CategoryID: cat.ID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	events, _ := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	// Re-append the same event (same event_id) — must be idempotent.
	result, err := store.Append(context.Background(), events[0])
	if err != nil {
		t.Fatalf("idempotent append: %v", err)
	}
	if result.Applied {
		t.Fatal("expected Applied=false for duplicate event_id")
	}
	if result.Sequence != events[0].Sequence {
		t.Fatalf("expected same sequence, got %d vs %d", result.Sequence, events[0].Sequence)
	}

	// Still only one event.
	events2, _ := store.ReadByAggregate(context.Background(), resource.ID, 0, 0)
	if len(events2) != 1 {
		t.Fatalf("expected 1 event after idempotent append, got %d", len(events2))
	}
}
