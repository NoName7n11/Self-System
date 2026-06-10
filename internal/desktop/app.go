package desktop

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
)

func parseOptionalTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("invalid RFC3339 time %q: %w", s, err)
	}
	return &t, nil
}

func parseRequiredTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid RFC3339 time %q: %w", s, err)
	}
	return t, nil
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// App is the Wails application struct. Every exported method becomes an IPC
// binding callable from the frontend via the generated TypeScript bindings.
type App struct {
	ctx       context.Context
	resources *service.ResourceService
	categories *service.CategoryService
	todos      *service.TodoService
	reminders  *service.ReminderService
}

// AppOptions contains all services the desktop app needs.
type AppOptions struct {
	Resources  *service.ResourceService
	Categories *service.CategoryService
	Todos      *service.TodoService
	Reminders  *service.ReminderService
}

// NewApp creates the Wails App with all services wired in.
func NewApp(opts AppOptions) *App {
	return &App{
		resources:  opts.Resources,
		categories: opts.Categories,
		todos:      opts.Todos,
		reminders:  opts.Reminders,
	}
}

// Startup is called when the application starts. The context is saved so
// Wails runtime functions (notifications, window ops) can be called later.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	slog.Info("Self Systems desktop app started")
}

// Shutdown is called when the application is about to quit.
func (a *App) Shutdown(_ context.Context) {
	slog.Info("Self Systems desktop app shutting down")
}

// ── Resource IPC methods ──────────────────────────────────────────────────────

func (a *App) GetResources(limit, offset int) ([]domain.Resource, error) {
	if limit <= 0 {
		limit = 50
	}
	return a.resources.List(a.ctx, limit, offset)
}

func (a *App) GetResourceByID(id string) (*domain.Resource, error) {
	return a.resources.GetByID(a.ctx, id)
}

func (a *App) CreateResource(url, title, summary, categoryName string) (domain.Resource, error) {
	res, err := a.resources.Create(a.ctx, service.CreateResourceInput{
		URL:          url,
		Title:        title,
		Summary:      summary,
		CategoryName: categoryName,
	})
	if err != nil {
		return domain.Resource{}, err
	}
	return res, nil
}

func (a *App) UpdateResource(id, url, title, summary, categoryName string) (*domain.Resource, error) {
	return a.resources.Update(a.ctx, service.UpdateResourceInput{
		ID:           id,
		URL:          url,
		Title:        title,
		Summary:      summary,
		CategoryName: categoryName,
	})
}

func (a *App) DeleteResource(id string) (bool, error) {
	return a.resources.Delete(a.ctx, id)
}

func (a *App) SearchResources(query string, limit int) ([]domain.Resource, error) {
	return a.resources.Search(a.ctx, query, limit)
}

func (a *App) ArchiveResource(id string) error {
	return a.resources.Archive(a.ctx, id, domain.ArchiveReasonManual)
}

func (a *App) RestoreResource(id string) error {
	return a.resources.Restore(a.ctx, id)
}

// ── Category IPC methods ──────────────────────────────────────────────────────

func (a *App) GetCategories() ([]domain.Category, error) {
	return a.categories.List(a.ctx)
}

func (a *App) CreateCategory(name, description string) (domain.Category, error) {
	return a.categories.Create(a.ctx, service.CreateCategoryInput{
		Name:        name,
		Description: description,
	})
}

func (a *App) UpdateCategory(id, name, description string) (*domain.Category, error) {
	return a.categories.Update(a.ctx, service.UpdateCategoryInput{
		ID:          id,
		Name:        name,
		Description: description,
	})
}

func (a *App) DeleteCategory(id string) (bool, error) {
	return a.categories.Delete(a.ctx, id)
}

// ── Todo IPC methods ──────────────────────────────────────────────────────────

func (a *App) GetTodos(limit, offset int) ([]domain.Todo, error) {
	if limit <= 0 {
		limit = 50
	}
	return a.todos.List(a.ctx, limit, offset)
}

func (a *App) CreateTodo(title, details, dueAt, resourceId string) (domain.Todo, error) {
	t, err := parseOptionalTime(dueAt)
	if err != nil {
		return domain.Todo{}, err
	}
	return a.todos.Create(a.ctx, service.CreateTodoInput{
		Title:      title,
		Details:    details,
		DueAt:      t,
		ResourceID: optionalString(resourceId),
	})
}

func (a *App) UpdateTodo(id, title, details, dueAt, status, resourceId string) (*domain.Todo, error) {
	t, err := parseOptionalTime(dueAt)
	if err != nil {
		return nil, err
	}
	return a.todos.Update(a.ctx, service.UpdateTodoInput{
		ID:         id,
		Title:      title,
		Details:    details,
		Status:     domain.TodoStatus(status),
		DueAt:      t,
		ResourceID: optionalString(resourceId),
	})
}

func (a *App) DeleteTodo(id string) (bool, error) {
	return a.todos.Delete(a.ctx, id)
}

// ── Reminder IPC methods ──────────────────────────────────────────────────────

func (a *App) GetReminders(limit, offset int) ([]domain.Reminder, error) {
	if limit <= 0 {
		limit = 50
	}
	return a.reminders.List(a.ctx, limit, offset)
}

func (a *App) CreateReminder(title, message, remindAt, resourceId string) (domain.Reminder, error) {
	t, err := parseRequiredTime(remindAt)
	if err != nil {
		return domain.Reminder{}, err
	}
	return a.reminders.Create(a.ctx, service.CreateReminderInput{
		Title:      title,
		Message:    message,
		RemindAt:   t,
		ResourceID: optionalString(resourceId),
	})
}

func (a *App) UpdateReminder(id, title, message, remindAt, status, resourceId string) (*domain.Reminder, error) {
	t, err := parseRequiredTime(remindAt)
	if err != nil {
		return nil, err
	}
	return a.reminders.Update(a.ctx, service.UpdateReminderInput{
		ID:         id,
		Title:      title,
		Message:    message,
		RemindAt:   t,
		Status:     domain.ReminderStatus(status),
		ResourceID: optionalString(resourceId),
	})
}

func (a *App) DeleteReminder(id string) (bool, error) {
	return a.reminders.Delete(a.ctx, id)
}

// ── Desktop-native helpers ────────────────────────────────────────────────────

// NotifyProcessingComplete triggers a native OS notification when deep
// processing finishes for a resource.
func (a *App) NotifyProcessingComplete(resourceTitle string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "processing:complete", map[string]any{
		"title": resourceTitle,
	})
}
