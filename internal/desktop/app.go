package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	ctx        context.Context
	resources  *service.ResourceService
	categories *service.CategoryService
	todos      *service.TodoService
	reminders  *service.ReminderService
	statePath  string
}

// windowState is the persisted window geometry, saved on shutdown and
// restored on startup so the app reopens where the user left it.
type windowState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	X      int `json:"x"`
	Y      int `json:"y"`
}

// windowStatePath returns the file used to persist window geometry. It lives
// under the OS user-config dir; on failure we fall back to the working dir so
// the feature degrades gracefully rather than panicking.
func windowStatePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "selfsystems-window.json"
	}
	appDir := filepath.Join(dir, "SelfSystems")
	if mkErr := os.MkdirAll(appDir, 0o755); mkErr != nil {
		return "selfsystems-window.json"
	}
	return filepath.Join(appDir, "window.json")
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
		statePath:  windowStatePath(),
	}
}

// Startup is called when the application starts. The context is saved so
// Wails runtime functions (notifications, window ops) can be called later.
// It also restores the saved window geometry, initializes the OS notification
// subsystem, and registers the native file-drop handler.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.wireRuntime(ctx)
	slog.Info("Self Systems desktop app started")
}

// wireRuntime restores window geometry and registers notifications, file-drop,
// and the system tray. These call Wails runtime functions that log.Fatal (which
// calls os.Exit) when handed anything other than the real lifecycle context, so
// we first confirm the context carries the Wails frontend value. Unit tests pass
// a bare context.Background and fall through this guard untouched.
func (a *App) wireRuntime(ctx context.Context) {
	if ctx == nil || ctx.Value("frontend") == nil {
		slog.Debug("non-Wails context; skipping desktop runtime wiring")
		return
	}

	a.restoreWindowState()

	if err := runtime.InitializeNotifications(ctx); err != nil {
		slog.Warn("initialize notifications", "error", err)
	}

	runtime.OnFileDrop(ctx, func(_, _ int, paths []string) {
		// Emit dropped file paths to the frontend, which creates a resource
		// per file via the CreateResource IPC binding.
		runtime.EventsEmit(ctx, "files:dropped", paths)
	})

	a.StartTray()
}

// Shutdown is called when the application is about to quit. It persists the
// current window geometry so the next launch restores it.
func (a *App) Shutdown(_ context.Context) {
	a.saveWindowState()
	slog.Info("Self Systems desktop app shutting down")
}

// restoreWindowState reads the persisted geometry and applies it. Missing or
// unreadable state is ignored (first launch uses the defaults from main.go).
func (a *App) restoreWindowState() {
	if a.ctx == nil || a.statePath == "" {
		return
	}
	data, err := os.ReadFile(a.statePath)
	if err != nil {
		return
	}
	var st windowState
	if err := json.Unmarshal(data, &st); err != nil {
		return
	}
	if st.Width > 0 && st.Height > 0 {
		runtime.WindowSetSize(a.ctx, st.Width, st.Height)
	}
	runtime.WindowSetPosition(a.ctx, st.X, st.Y)
}

// saveWindowState writes the current geometry to disk. Errors are logged but
// not fatal — a failed save just means the next launch uses defaults.
func (a *App) saveWindowState() {
	if a.ctx == nil || a.statePath == "" {
		return
	}
	w, h := runtime.WindowGetSize(a.ctx)
	x, y := runtime.WindowGetPosition(a.ctx)
	data, err := json.Marshal(windowState{Width: w, Height: h, X: x, Y: y})
	if err != nil {
		slog.Warn("marshal window state", "error", err)
		return
	}
	if err := os.WriteFile(a.statePath, data, 0o644); err != nil {
		slog.Warn("write window state", "error", err)
	}
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

	// Fire a native OS notification when available so the user is alerted even
	// when the window is minimized to the tray / background.
	if runtime.IsNotificationAvailable(a.ctx) {
		if err := runtime.SendNotification(a.ctx, runtime.NotificationOptions{
			ID:    fmt.Sprintf("processing-complete-%d", time.Now().UnixNano()),
			Title: "Processing complete",
			Body:  fmt.Sprintf("%q finished deep processing.", resourceTitle),
		}); err != nil {
			slog.Warn("send notification", "error", err)
		}
	}
}
