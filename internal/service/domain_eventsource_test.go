package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

// ── common helpers ────────────────────────────────────────────────────────────

func openDomainDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := sqliterepo.Open(filepath.Join(t.TempDir(), "domain.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db, func() { _ = db.Close() }
}

// ── CategoryService ───────────────────────────────────────────────────────────

func newESCategorySvc(t *testing.T, db *sql.DB) *CategoryService {
	t.Helper()
	store := eventstore.NewSQLiteStore(db)
	reg := eventstore.NewProjectorRegistry()
	eventstore.RegisterCategoryProjectors(reg, "sqlite")
	return NewCategoryService(sqliterepo.NewCategoryRepository(db), WithCategoryEventSourcing(store, reg))
}

func TestCategoryServiceESCreate(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESCategorySvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	cat, err := svc.Create(context.Background(), CreateCategoryInput{Name: "Technology", Source: domain.CategorySourceManual})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cat.ID == "" {
		t.Fatal("expected non-empty id")
	}

	// Event appended.
	events, _ := store.ReadByAggregate(context.Background(), cat.ID, 0, 0)
	if len(events) != 1 || events[0].EventType != eventstore.EventTypeCategoryCreated {
		t.Fatalf("expected CategoryCreated event, got %v", events)
	}

	// Projection populated.
	repo := sqliterepo.NewCategoryRepository(db)
	got, _ := repo.GetByID(context.Background(), cat.ID)
	if got == nil || got.Name != "Technology" {
		t.Fatalf("expected category in projection, got %v", got)
	}
}

func TestCategoryServiceESCreateIdempotentByName(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESCategorySvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	first, _ := svc.Create(context.Background(), CreateCategoryInput{Name: "Science"})
	second, err := svc.Create(context.Background(), CreateCategoryInput{Name: "Science"})
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}
	if first.ID != second.ID {
		t.Fatal("expected same id for duplicate name")
	}
	// Only one event.
	events, _ := store.ReadByAggregate(context.Background(), first.ID, 0, 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestCategoryServiceESUpdate(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESCategorySvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	cat, _ := svc.Create(context.Background(), CreateCategoryInput{Name: "Old Name"})
	updated, err := svc.Update(context.Background(), UpdateCategoryInput{ID: cat.ID, Name: "New Name", Description: "Updated"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("unexpected name: %s", updated.Name)
	}

	events, _ := store.ReadByAggregate(context.Background(), cat.ID, 0, 0)
	if len(events) != 2 || events[1].EventType != eventstore.EventTypeCategoryUpdated {
		t.Fatalf("expected CategoryUpdated event: %v", events)
	}

	repo := sqliterepo.NewCategoryRepository(db)
	got, _ := repo.GetByID(context.Background(), cat.ID)
	if got.Name != "New Name" {
		t.Fatalf("projection not updated: %s", got.Name)
	}
}

func TestCategoryServiceESDelete(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESCategorySvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	cat, _ := svc.Create(context.Background(), CreateCategoryInput{Name: "ToDelete"})
	deleted, err := svc.Delete(context.Background(), cat.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}

	events, _ := store.ReadByAggregate(context.Background(), cat.ID, 0, 0)
	if len(events) != 2 || events[1].EventType != eventstore.EventTypeCategoryDeleted {
		t.Fatalf("expected CategoryDeleted event: %v", events)
	}

	repo := sqliterepo.NewCategoryRepository(db)
	got, _ := repo.GetByID(context.Background(), cat.ID)
	if got != nil {
		t.Fatal("expected category removed from projection")
	}
}

func TestCategoryServiceESFlagOff(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	store := eventstore.NewSQLiteStore(db)

	// No event-sourcing option — flag OFF.
	svc := NewCategoryService(sqliterepo.NewCategoryRepository(db))
	cat, _ := svc.Create(context.Background(), CreateCategoryInput{Name: "DirectWrite"})

	events, _ := store.ReadByAggregate(context.Background(), cat.ID, 0, 0)
	if len(events) != 0 {
		t.Fatalf("expected 0 events with flag OFF, got %d", len(events))
	}

	repo := sqliterepo.NewCategoryRepository(db)
	got, _ := repo.GetByID(context.Background(), cat.ID)
	if got == nil {
		t.Fatal("expected category in projection")
	}
}

func TestCategoryServiceESEnsureByName(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESCategorySvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	cat, err := svc.EnsureByName(context.Background(), "Auto Category", domain.CategorySourceAuto)
	if err != nil {
		t.Fatalf("EnsureByName: %v", err)
	}

	events, _ := store.ReadByAggregate(context.Background(), cat.ID, 0, 0)
	if len(events) != 1 || events[0].EventType != eventstore.EventTypeCategoryCreated {
		t.Fatalf("expected CategoryCreated: %v", events)
	}

	// Second call — returns existing, no new event.
	cat2, _ := svc.EnsureByName(context.Background(), "Auto Category", domain.CategorySourceAuto)
	if cat2.ID != cat.ID {
		t.Fatal("expected same id on second call")
	}
	events2, _ := store.ReadByAggregate(context.Background(), cat.ID, 0, 0)
	if len(events2) != 1 {
		t.Fatalf("expected still 1 event, got %d", len(events2))
	}
}

// ── TodoService ───────────────────────────────────────────────────────────────

func newESTodoSvc(t *testing.T, db *sql.DB) *TodoService {
	t.Helper()
	store := eventstore.NewSQLiteStore(db)
	reg := eventstore.NewProjectorRegistry()
	eventstore.RegisterTodoProjectors(reg, "sqlite")
	return NewTodoService(sqliterepo.NewTodoRepository(db), WithTodoEventSourcing(store, reg))
}

func seedCategoryForTodo(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	db.Exec(`INSERT INTO categories (id, name, description, source, created_at, updated_at) VALUES (?, 'Test', '', 'manual', ?, ?)`, id, now, now)
	return id
}

func TestTodoServiceESCreate(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESTodoSvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	todo, err := svc.Create(context.Background(), CreateTodoInput{Title: "Buy milk"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	events, _ := store.ReadByAggregate(context.Background(), todo.ID, 0, 0)
	if len(events) != 1 || events[0].EventType != eventstore.EventTypeTodoCreated {
		t.Fatalf("expected TodoCreated: %v", events)
	}

	repo := sqliterepo.NewTodoRepository(db)
	got, _ := repo.GetByID(context.Background(), todo.ID)
	if got == nil || got.Title != "Buy milk" {
		t.Fatalf("projection not populated: %v", got)
	}
}

func TestTodoServiceESUpdate(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESTodoSvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	todo, _ := svc.Create(context.Background(), CreateTodoInput{Title: "Original"})
	updated, err := svc.Update(context.Background(), UpdateTodoInput{
		ID: todo.ID, Title: "Updated", Status: domain.TodoStatusInProgress,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("unexpected title: %s", updated.Title)
	}

	events, _ := store.ReadByAggregate(context.Background(), todo.ID, 0, 0)
	if len(events) != 2 || events[1].EventType != eventstore.EventTypeTodoUpdated {
		t.Fatalf("expected TodoUpdated: %v", events)
	}

	repo := sqliterepo.NewTodoRepository(db)
	got, _ := repo.GetByID(context.Background(), todo.ID)
	if got.Title != "Updated" || got.Status != domain.TodoStatusInProgress {
		t.Fatalf("projection not updated: %v", got)
	}
}

func TestTodoServiceESDelete(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESTodoSvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	todo, _ := svc.Create(context.Background(), CreateTodoInput{Title: "Delete me"})
	deleted, err := svc.Delete(context.Background(), todo.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}

	events, _ := store.ReadByAggregate(context.Background(), todo.ID, 0, 0)
	if len(events) != 2 || events[1].EventType != eventstore.EventTypeTodoDeleted {
		t.Fatalf("expected TodoDeleted: %v", events)
	}

	repo := sqliterepo.NewTodoRepository(db)
	got, _ := repo.GetByID(context.Background(), todo.ID)
	if got != nil {
		t.Fatal("expected todo removed from projection")
	}
}

func TestTodoServiceESFlagOff(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	store := eventstore.NewSQLiteStore(db)

	svc := NewTodoService(sqliterepo.NewTodoRepository(db))
	todo, _ := svc.Create(context.Background(), CreateTodoInput{Title: "Direct"})

	events, _ := store.ReadByAggregate(context.Background(), todo.ID, 0, 0)
	if len(events) != 0 {
		t.Fatalf("expected 0 events with flag OFF, got %d", len(events))
	}
	repo := sqliterepo.NewTodoRepository(db)
	got, _ := repo.GetByID(context.Background(), todo.ID)
	if got == nil {
		t.Fatal("expected todo in projection")
	}
}

// ── ReminderService ───────────────────────────────────────────────────────────

func newESReminderSvc(t *testing.T, db *sql.DB) *ReminderService {
	t.Helper()
	store := eventstore.NewSQLiteStore(db)
	reg := eventstore.NewProjectorRegistry()
	eventstore.RegisterReminderProjectors(reg, "sqlite")
	return NewReminderService(sqliterepo.NewReminderRepository(db), WithReminderEventSourcing(store, reg))
}

func TestReminderServiceESCreate(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESReminderSvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	remindAt := time.Now().Add(24 * time.Hour).UTC()
	r, err := svc.Create(context.Background(), CreateReminderInput{Title: "Dentist", RemindAt: remindAt})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	events, _ := store.ReadByAggregate(context.Background(), r.ID, 0, 0)
	if len(events) != 1 || events[0].EventType != eventstore.EventTypeReminderCreated {
		t.Fatalf("expected ReminderCreated: %v", events)
	}

	repo := sqliterepo.NewReminderRepository(db)
	got, _ := repo.GetByID(context.Background(), r.ID)
	if got == nil || got.Title != "Dentist" {
		t.Fatalf("projection not populated: %v", got)
	}
}

func TestReminderServiceESUpdate(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESReminderSvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	remindAt := time.Now().Add(24 * time.Hour).UTC()
	r, _ := svc.Create(context.Background(), CreateReminderInput{Title: "Original", RemindAt: remindAt})
	updated, err := svc.Update(context.Background(), UpdateReminderInput{
		ID: r.ID, Title: "Updated", RemindAt: remindAt, Status: domain.ReminderStatusScheduled,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("unexpected title: %s", updated.Title)
	}

	events, _ := store.ReadByAggregate(context.Background(), r.ID, 0, 0)
	if len(events) != 2 || events[1].EventType != eventstore.EventTypeReminderUpdated {
		t.Fatalf("expected ReminderUpdated: %v", events)
	}
}

func TestReminderServiceESDelete(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	svc := newESReminderSvc(t, db)
	store := eventstore.NewSQLiteStore(db)

	remindAt := time.Now().Add(24 * time.Hour).UTC()
	r, _ := svc.Create(context.Background(), CreateReminderInput{Title: "Delete me", RemindAt: remindAt})
	deleted, err := svc.Delete(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}

	events, _ := store.ReadByAggregate(context.Background(), r.ID, 0, 0)
	if len(events) != 2 || events[1].EventType != eventstore.EventTypeReminderDeleted {
		t.Fatalf("expected ReminderDeleted: %v", events)
	}

	repo := sqliterepo.NewReminderRepository(db)
	got, _ := repo.GetByID(context.Background(), r.ID)
	if got != nil {
		t.Fatal("expected reminder removed from projection")
	}
}

func TestReminderServiceESFlagOff(t *testing.T) {
	db, cleanup := openDomainDB(t)
	defer cleanup()
	store := eventstore.NewSQLiteStore(db)

	svc := NewReminderService(sqliterepo.NewReminderRepository(db))
	remindAt := time.Now().Add(24 * time.Hour).UTC()
	r, _ := svc.Create(context.Background(), CreateReminderInput{Title: "Direct", RemindAt: remindAt})

	events, _ := store.ReadByAggregate(context.Background(), r.ID, 0, 0)
	if len(events) != 0 {
		t.Fatalf("expected 0 events with flag OFF, got %d", len(events))
	}
	repo := sqliterepo.NewReminderRepository(db)
	got, _ := repo.GetByID(context.Background(), r.ID)
	if got == nil {
		t.Fatal("expected reminder in projection")
	}
}
