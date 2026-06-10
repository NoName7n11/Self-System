package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"selfsystems/internal/ai"
	sqliterepo "selfsystems/internal/repository/sqlite"
)

func TestParseCommandIDStrict(t *testing.T) {
	id, err := parseCommandID("abc")
	if err != nil || id != "abc" {
		t.Fatalf("expected single-token success, got id=%q err=%v", id, err)
	}

	id, err = parseCommandID("")
	if err != nil || id != "" {
		t.Fatalf("expected empty-input no-error, got id=%q err=%v", id, err)
	}

	if _, err = parseCommandID("abc def"); err == nil {
		t.Fatalf("expected error on multi-token input")
	}
}

func TestApplyAllowlistDropsUnknownKeys(t *testing.T) {
	parsed := parsePipePayload("https://x.com | category=AI | injected=danger | title=ok")
	parsed = applyAllowlist(parsed, "category", "title")

	if parsed["category"] != "AI" || parsed["title"] != "ok" {
		t.Fatalf("expected allowed keys preserved, got %#v", parsed)
	}
	if _, ok := parsed["injected"]; ok {
		t.Fatalf("expected unknown key 'injected' to be dropped, got %#v", parsed)
	}
	if parsed["url"] == "" {
		t.Fatalf("expected parser-internal 'url' to be preserved")
	}
}

func TestChatServiceRejectsUnknownPipeKeysWithoutSideEffect(t *testing.T) {
	chatSvc, ctx := newChatServiceTestFixture(t)

	res, err := chatSvc.Execute(ctx, "create todo plan | injected=evil | title=Real Title")
	if err != nil {
		t.Fatalf("execute create todo: %v", err)
	}
	if res.Action != "todo_created" || res.Todo == nil {
		t.Fatalf("expected todo_created, got action=%q", res.Action)
	}
	if res.Todo.Title != "Real Title" {
		t.Fatalf("expected explicit title to win, got %q", res.Todo.Title)
	}
}

func TestChatServiceRejectsMultiTokenIDsExplicitly(t *testing.T) {
	chatSvc, ctx := newChatServiceTestFixture(t)

	res, err := chatSvc.Execute(ctx, "get resource abc def")
	if err != nil {
		t.Fatalf("execute get resource: %v", err)
	}
	if res.Action != "resource_error" {
		t.Fatalf("expected resource_error, got action=%q", res.Action)
	}
	if !strings.Contains(res.Message, "single id token") {
		t.Fatalf("expected explicit multi-token error message, got %q", res.Message)
	}
}

func TestParsePipePayload(t *testing.T) {
	payload := "https://example.com | category=AI | title=Example"
	parsed := parsePipePayload(payload)

	if parsed["url"] != "https://example.com" {
		t.Fatalf("expected url to be parsed")
	}
	if parsed["category"] != "AI" {
		t.Fatalf("expected category to be parsed")
	}
	if parsed["title"] != "Example" {
		t.Fatalf("expected title to be parsed")
	}
}

func TestFirstNonEmpty(t *testing.T) {
	result := firstNonEmpty("", "   ", "ok")
	if result != "ok" {
		t.Fatalf("expected ok, got %q", result)
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback int
		want     int
	}{
		{name: "empty uses fallback", raw: "", fallback: 20, want: 20},
		{name: "invalid uses fallback", raw: "x", fallback: 20, want: 20},
		{name: "negative uses fallback", raw: "-10", fallback: 20, want: 20},
		{name: "valid value", raw: "15", fallback: 20, want: 15},
		{name: "clamped to max", raw: "120", fallback: 20, want: 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLimit(tc.raw, tc.fallback)
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestParseGraphLimit(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback int
		want     int
	}{
		{name: "empty uses fallback", raw: "", fallback: 1000, want: 1000},
		{name: "invalid uses fallback", raw: "x", fallback: 1000, want: 1000},
		{name: "negative uses fallback", raw: "-1", fallback: 1000, want: 1000},
		{name: "valid value", raw: "1200", fallback: 1000, want: 1200},
		{name: "clamped to max", raw: "10000", fallback: 1000, want: 5000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseGraphLimit(tc.raw, tc.fallback)
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestParsePipePayloadQueryMode(t *testing.T) {
	payload := "ai graphs | query=knowledge graph | limit=12"
	parsed := parsePipePayload(payload)

	if parsed["value"] != "ai graphs" {
		t.Fatalf("expected value to be parsed")
	}
	if parsed["query"] != "knowledge graph" {
		t.Fatalf("expected query to be parsed")
	}
	if parsed["limit"] != "12" {
		t.Fatalf("expected limit to be parsed")
	}
}

func TestChatServiceCategoryCRUDCommands(t *testing.T) {
	chatSvc, ctx := newChatServiceTestFixture(t)

	created, err := chatSvc.Execute(ctx, "create category research | notes")
	if err != nil {
		t.Fatalf("execute create category: %v", err)
	}
	if created.Action != "category_created" || created.Category == nil {
		t.Fatalf("expected category_created with payload, got action=%q", created.Action)
	}

	id := created.Category.ID
	retrieved, err := chatSvc.Execute(ctx, "get category "+id)
	if err != nil {
		t.Fatalf("execute get category: %v", err)
	}
	if retrieved.Action != "category_retrieved" || retrieved.Category == nil || retrieved.Category.ID != id {
		t.Fatalf("expected category_retrieved for id %q, got action=%q", id, retrieved.Action)
	}

	updated, err := chatSvc.Execute(ctx, "update category "+id+" | name=knowledge systems | description=updated")
	if err != nil {
		t.Fatalf("execute update category: %v", err)
	}
	if updated.Action != "category_updated" || updated.Category == nil {
		t.Fatalf("expected category_updated with payload, got action=%q", updated.Action)
	}
	if updated.Category.Name != "Knowledge Systems" {
		t.Fatalf("expected updated category name Knowledge Systems, got %q", updated.Category.Name)
	}

	deleted, err := chatSvc.Execute(ctx, "delete category "+id)
	if err != nil {
		t.Fatalf("execute delete category: %v", err)
	}
	if deleted.Action != "category_deleted" {
		t.Fatalf("expected category_deleted, got %q", deleted.Action)
	}

	missing, err := chatSvc.Execute(ctx, "get category "+id)
	if err != nil {
		t.Fatalf("execute get deleted category: %v", err)
	}
	if missing.Action != "category_error" || !strings.Contains(missing.Message, "not found") {
		t.Fatalf("expected category_error not found, got action=%q message=%q", missing.Action, missing.Message)
	}
}

func TestChatServiceResourceCRUDCommands(t *testing.T) {
	chatSvc, ctx := newChatServiceTestFixture(t)

	created, err := chatSvc.Execute(ctx, "resource: https://example.com/resource | title=Resource Title | category=Research")
	if err != nil {
		t.Fatalf("execute create resource: %v", err)
	}
	if created.Action != "resource_created" || created.Resource == nil {
		t.Fatalf("expected resource_created with payload, got action=%q", created.Action)
	}

	id := created.Resource.ID
	retrieved, err := chatSvc.Execute(ctx, "get resource "+id)
	if err != nil {
		t.Fatalf("execute get resource: %v", err)
	}
	if retrieved.Action != "resource_retrieved" || retrieved.Resource == nil || retrieved.Resource.ID != id {
		t.Fatalf("expected resource_retrieved for id %q, got action=%q", id, retrieved.Action)
	}

	updated, err := chatSvc.Execute(ctx, "update resource "+id+" | title=Updated Resource | summary=updated summary")
	if err != nil {
		t.Fatalf("execute update resource: %v", err)
	}
	if updated.Action != "resource_updated" || updated.Resource == nil {
		t.Fatalf("expected resource_updated with payload, got action=%q", updated.Action)
	}
	if updated.Resource.Title != "Updated Resource" {
		t.Fatalf("expected updated resource title, got %q", updated.Resource.Title)
	}

	deleted, err := chatSvc.Execute(ctx, "delete resource "+id)
	if err != nil {
		t.Fatalf("execute delete resource: %v", err)
	}
	if deleted.Action != "resource_deleted" {
		t.Fatalf("expected resource_deleted, got %q", deleted.Action)
	}

	missing, err := chatSvc.Execute(ctx, "get resource "+id)
	if err != nil {
		t.Fatalf("execute get deleted resource: %v", err)
	}
	if missing.Action != "resource_error" || !strings.Contains(missing.Message, "not found") {
		t.Fatalf("expected resource_error not found, got action=%q message=%q", missing.Action, missing.Message)
	}
}

func TestChatServiceTodoCRUDCommands(t *testing.T) {
	chatSvc, ctx := newChatServiceTestFixture(t)

	created, err := chatSvc.Execute(ctx, "create todo draft spec | details=phase 2 | due=2026-04-21T10:00:00Z")
	if err != nil {
		t.Fatalf("execute create todo: %v", err)
	}
	if created.Action != "todo_created" || created.Todo == nil {
		t.Fatalf("expected todo_created with payload, got action=%q", created.Action)
	}

	id := created.Todo.ID
	retrieved, err := chatSvc.Execute(ctx, "get todo "+id)
	if err != nil {
		t.Fatalf("execute get todo: %v", err)
	}
	if retrieved.Action != "todo_retrieved" || retrieved.Todo == nil || retrieved.Todo.ID != id {
		t.Fatalf("expected todo_retrieved for id %q, got action=%q", id, retrieved.Action)
	}

	updated, err := chatSvc.Execute(ctx, "update todo "+id+" | title=draft spec updated | details=done | status=done | due=2026-04-22T10:00:00Z")
	if err != nil {
		t.Fatalf("execute update todo: %v", err)
	}
	if updated.Action != "todo_updated" || updated.Todo == nil {
		t.Fatalf("expected todo_updated with payload, got action=%q", updated.Action)
	}
	if string(updated.Todo.Status) != "done" {
		t.Fatalf("expected updated todo status done, got %q", updated.Todo.Status)
	}

	deleted, err := chatSvc.Execute(ctx, "delete todo "+id)
	if err != nil {
		t.Fatalf("execute delete todo: %v", err)
	}
	if deleted.Action != "todo_deleted" {
		t.Fatalf("expected todo_deleted, got %q", deleted.Action)
	}

	missing, err := chatSvc.Execute(ctx, "get todo "+id)
	if err != nil {
		t.Fatalf("execute get deleted todo: %v", err)
	}
	if missing.Action != "todo_error" || !strings.Contains(missing.Message, "not found") {
		t.Fatalf("expected todo_error not found, got action=%q message=%q", missing.Action, missing.Message)
	}
}

func TestChatServiceReminderCRUDCommands(t *testing.T) {
	chatSvc, ctx := newChatServiceTestFixture(t)

	created, err := chatSvc.Execute(ctx, "create reminder follow up | at=2026-04-21T10:00:00Z | message=ping")
	if err != nil {
		t.Fatalf("execute create reminder: %v", err)
	}
	if created.Action != "reminder_created" || created.Reminder == nil {
		t.Fatalf("expected reminder_created with payload, got action=%q", created.Action)
	}

	id := created.Reminder.ID
	retrieved, err := chatSvc.Execute(ctx, "get reminder "+id)
	if err != nil {
		t.Fatalf("execute get reminder: %v", err)
	}
	if retrieved.Action != "reminder_retrieved" || retrieved.Reminder == nil || retrieved.Reminder.ID != id {
		t.Fatalf("expected reminder_retrieved for id %q, got action=%q", id, retrieved.Action)
	}

	updated, err := chatSvc.Execute(ctx, "update reminder "+id+" | title=follow up updated | at=2026-04-22T11:00:00Z | status=sent | message=sent")
	if err != nil {
		t.Fatalf("execute update reminder: %v", err)
	}
	if updated.Action != "reminder_updated" || updated.Reminder == nil {
		t.Fatalf("expected reminder_updated with payload, got action=%q", updated.Action)
	}
	if string(updated.Reminder.Status) != "sent" {
		t.Fatalf("expected updated reminder status sent, got %q", updated.Reminder.Status)
	}

	deleted, err := chatSvc.Execute(ctx, "delete reminder "+id)
	if err != nil {
		t.Fatalf("execute delete reminder: %v", err)
	}
	if deleted.Action != "reminder_deleted" {
		t.Fatalf("expected reminder_deleted, got %q", deleted.Action)
	}

	missing, err := chatSvc.Execute(ctx, "get reminder "+id)
	if err != nil {
		t.Fatalf("execute get deleted reminder: %v", err)
	}
	if missing.Action != "reminder_error" || !strings.Contains(missing.Message, "not found") {
		t.Fatalf("expected reminder_error not found, got action=%q message=%q", missing.Action, missing.Message)
	}
}

func newChatServiceTestFixture(t *testing.T) (*ChatService, context.Context) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "chat_service_test.db")
	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	categoryRepo := sqliterepo.NewCategoryRepository(db)
	resourceRepo := sqliterepo.NewResourceRepository(db)
	todoRepo := sqliterepo.NewTodoRepository(db)
	reminderRepo := sqliterepo.NewReminderRepository(db)

	aiManager := ai.NewManager("heuristic")
	heuristicProvider := ai.NewHeuristicProvider()
	aiManager.Register(heuristicProvider)
	aiManager.SetFallback(heuristicProvider.Name())

	categorySvc := NewCategoryService(categoryRepo)
	classifier := NewCategoryClassifier(categoryRepo, aiManager)
	resourceSvc := NewResourceService(resourceRepo, categoryRepo, classifier, categorySvc)
	todoSvc := NewTodoService(todoRepo)
	reminderSvc := NewReminderService(reminderRepo)

	return NewChatService(categorySvc, resourceSvc, todoSvc, reminderSvc, nil), context.Background()
}
