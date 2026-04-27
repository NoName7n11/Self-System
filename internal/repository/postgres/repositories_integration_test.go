package postgres

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"selfsystems/internal/domain"
)

func TestRepositoriesCRUDIntegration(t *testing.T) {
	dsn := postgresIntegrationDSN(t)

	db, err := Open(dsn)
	if err != nil {
		t.Fatalf("open postgres integration db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	resetPostgresIntegrationTables(t, db)

	ctx := context.Background()
	categoryRepo := NewCategoryRepository(db)
	resourceRepo := NewResourceRepository(db)
	todoRepo := NewTodoRepository(db)
	reminderRepo := NewReminderRepository(db)

	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405"), " ", "")
	primaryCategoryID := "cat-primary-" + suffix
	secondaryCategoryID := "cat-secondary-" + suffix
	resourceID := "res-" + suffix
	todoID := "todo-" + suffix
	reminderID := "rem-" + suffix

	primaryCategory := &domain.Category{
		ID:          primaryCategoryID,
		Name:        "Integration Primary " + suffix,
		Description: "Primary category",
		Source:      domain.CategorySourceManual,
	}
	if err := categoryRepo.Create(ctx, primaryCategory); err != nil {
		t.Fatalf("create primary category: %v", err)
	}

	if err := categoryRepo.IncrementAccept(ctx, primaryCategoryID); err != nil {
		t.Fatalf("increment accept count: %v", err)
	}
	if err := categoryRepo.IncrementOverride(ctx, primaryCategoryID); err != nil {
		t.Fatalf("increment override count: %v", err)
	}

	loadedPrimaryCategory, err := categoryRepo.GetByID(ctx, primaryCategoryID)
	if err != nil {
		t.Fatalf("get primary category by id: %v", err)
	}
	if loadedPrimaryCategory == nil {
		t.Fatal("expected primary category to exist")
	}
	if loadedPrimaryCategory.AcceptCount != 1 {
		t.Fatalf("expected accept count 1, got %d", loadedPrimaryCategory.AcceptCount)
	}
	if loadedPrimaryCategory.OverrideCount != 1 {
		t.Fatalf("expected override count 1, got %d", loadedPrimaryCategory.OverrideCount)
	}

	primaryCategory.Name = "Integration Primary Updated " + suffix
	primaryCategory.Description = "Primary category updated"
	if err := categoryRepo.Update(ctx, primaryCategory); err != nil {
		t.Fatalf("update primary category: %v", err)
	}

	loadedByName, err := categoryRepo.GetByName(ctx, primaryCategory.Name)
	if err != nil {
		t.Fatalf("get primary category by name: %v", err)
	}
	if loadedByName == nil {
		t.Fatal("expected category lookup by name to return a category")
	}
	if loadedByName.ID != primaryCategoryID {
		t.Fatalf("expected category id %q from name lookup, got %q", primaryCategoryID, loadedByName.ID)
	}

	secondaryCategory := &domain.Category{
		ID:          secondaryCategoryID,
		Name:        "Integration Secondary " + suffix,
		Description: "Secondary category",
		Source:      domain.CategorySourceAuto,
	}
	if err := categoryRepo.Create(ctx, secondaryCategory); err != nil {
		t.Fatalf("create secondary category: %v", err)
	}

	categories, err := categoryRepo.List(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) < 2 {
		t.Fatalf("expected at least 2 categories, got %d", len(categories))
	}

	resource := &domain.Resource{
		ID:         resourceID,
		URL:        "https://example.com/postgres-integration-" + suffix,
		Host:       "example.com",
		Title:      "Postgres Integration",
		Summary:    "Resource summary",
		CategoryID: primaryCategoryID,
	}
	if err := resourceRepo.Create(ctx, resource); err != nil {
		t.Fatalf("create resource: %v", err)
	}

	loadedResource, err := resourceRepo.GetByID(ctx, resourceID)
	if err != nil {
		t.Fatalf("get resource by id: %v", err)
	}
	if loadedResource == nil {
		t.Fatal("expected resource to exist")
	}
	if loadedResource.CategoryID != primaryCategoryID {
		t.Fatalf("expected resource category %q, got %q", primaryCategoryID, loadedResource.CategoryID)
	}

	resource.Title = "Postgres Integration Updated"
	resource.Summary = "Updated summary"
	if err := resourceRepo.Update(ctx, resource); err != nil {
		t.Fatalf("update resource: %v", err)
	}

	if err := resourceRepo.UpdateCategory(ctx, resourceID, secondaryCategoryID, true); err != nil {
		t.Fatalf("update resource category: %v", err)
	}

	updatedResource, err := resourceRepo.GetByID(ctx, resourceID)
	if err != nil {
		t.Fatalf("reload updated resource: %v", err)
	}
	if updatedResource == nil {
		t.Fatal("expected updated resource to exist")
	}
	if updatedResource.CategoryID != secondaryCategoryID {
		t.Fatalf("expected updated resource category %q, got %q", secondaryCategoryID, updatedResource.CategoryID)
	}
	if !updatedResource.UserOverride {
		t.Fatal("expected updated resource user override to be true")
	}

	resources, err := resourceRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(resources) < 1 {
		t.Fatalf("expected at least 1 resource, got %d", len(resources))
	}

	searchResults, err := resourceRepo.Search(ctx, "Integration Updated", 10)
	if err != nil {
		t.Fatalf("search resources: %v", err)
	}
	if len(searchResults) < 1 {
		t.Fatalf("expected at least 1 search result, got %d", len(searchResults))
	}

	dueAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	todo := &domain.Todo{
		ID:         todoID,
		Title:      "Integration Todo",
		Details:    "Todo details",
		Status:     domain.TodoStatusOpen,
		DueAt:      &dueAt,
		ResourceID: &resourceID,
	}
	if err := todoRepo.Create(ctx, todo); err != nil {
		t.Fatalf("create todo: %v", err)
	}

	loadedTodo, err := todoRepo.GetByID(ctx, todoID)
	if err != nil {
		t.Fatalf("get todo by id: %v", err)
	}
	if loadedTodo == nil {
		t.Fatal("expected todo to exist")
	}
	if loadedTodo.Status != domain.TodoStatusOpen {
		t.Fatalf("expected todo status %q, got %q", domain.TodoStatusOpen, loadedTodo.Status)
	}
	if loadedTodo.DueAt == nil || !loadedTodo.DueAt.Equal(dueAt) {
		t.Fatalf("expected todo due date %s, got %v", dueAt, loadedTodo.DueAt)
	}

	todo.Status = domain.TodoStatusDone
	todo.Title = "Integration Todo Done"
	if err := todoRepo.Update(ctx, todo); err != nil {
		t.Fatalf("update todo: %v", err)
	}

	todos, err := todoRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list todos: %v", err)
	}
	if len(todos) < 1 {
		t.Fatalf("expected at least 1 todo, got %d", len(todos))
	}

	remindAt := time.Date(2026, 4, 21, 9, 30, 0, 0, time.UTC)
	reminder := &domain.Reminder{
		ID:         reminderID,
		Title:      "Integration Reminder",
		Message:    "Reminder message",
		RemindAt:   remindAt,
		Status:     domain.ReminderStatusScheduled,
		ResourceID: &resourceID,
	}
	if err := reminderRepo.Create(ctx, reminder); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	loadedReminder, err := reminderRepo.GetByID(ctx, reminderID)
	if err != nil {
		t.Fatalf("get reminder by id: %v", err)
	}
	if loadedReminder == nil {
		t.Fatal("expected reminder to exist")
	}
	if loadedReminder.Status != domain.ReminderStatusScheduled {
		t.Fatalf("expected reminder status %q, got %q", domain.ReminderStatusScheduled, loadedReminder.Status)
	}
	if !loadedReminder.RemindAt.Equal(remindAt) {
		t.Fatalf("expected reminder time %s, got %s", remindAt, loadedReminder.RemindAt)
	}

	reminder.Status = domain.ReminderStatusDismissed
	reminder.Title = "Integration Reminder Updated"
	if err := reminderRepo.Update(ctx, reminder); err != nil {
		t.Fatalf("update reminder: %v", err)
	}

	reminders, err := reminderRepo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list reminders: %v", err)
	}
	if len(reminders) < 1 {
		t.Fatalf("expected at least 1 reminder, got %d", len(reminders))
	}

	if err := reminderRepo.Delete(ctx, reminderID); err != nil {
		t.Fatalf("delete reminder: %v", err)
	}
	deletedReminder, err := reminderRepo.GetByID(ctx, reminderID)
	if err != nil {
		t.Fatalf("get deleted reminder by id: %v", err)
	}
	if deletedReminder != nil {
		t.Fatal("expected deleted reminder to be nil")
	}

	if err := todoRepo.Delete(ctx, todoID); err != nil {
		t.Fatalf("delete todo: %v", err)
	}
	deletedTodo, err := todoRepo.GetByID(ctx, todoID)
	if err != nil {
		t.Fatalf("get deleted todo by id: %v", err)
	}
	if deletedTodo != nil {
		t.Fatal("expected deleted todo to be nil")
	}

	if err := resourceRepo.Delete(ctx, resourceID); err != nil {
		t.Fatalf("delete resource: %v", err)
	}
	deletedResource, err := resourceRepo.GetByID(ctx, resourceID)
	if err != nil {
		t.Fatalf("get deleted resource by id: %v", err)
	}
	if deletedResource != nil {
		t.Fatal("expected deleted resource to be nil")
	}

	if err := categoryRepo.Delete(ctx, secondaryCategoryID); err != nil {
		t.Fatalf("delete secondary category: %v", err)
	}
	if err := categoryRepo.Delete(ctx, primaryCategoryID); err != nil {
		t.Fatalf("delete primary category: %v", err)
	}

	deletedPrimaryCategory, err := categoryRepo.GetByID(ctx, primaryCategoryID)
	if err != nil {
		t.Fatalf("get deleted primary category by id: %v", err)
	}
	if deletedPrimaryCategory != nil {
		t.Fatal("expected deleted primary category to be nil")
	}
}

func postgresIntegrationDSN(t *testing.T) string {
	t.Helper()

	if dsn := strings.TrimSpace(os.Getenv("SS_POSTGRES_TEST_DSN")); dsn != "" {
		return dsn
	}
	if dsn := strings.TrimSpace(os.Getenv("SS_DATABASE_URL")); dsn != "" {
		return dsn
	}

	t.Skip("set SS_POSTGRES_TEST_DSN or SS_DATABASE_URL to run postgres integration tests")
	return ""
}

func resetPostgresIntegrationTables(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		TRUNCATE TABLE
			chat_events,
			sync_metadata,
			reminders,
			todos,
			resources,
			categories
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate integration tables: %v", err)
	}
}
