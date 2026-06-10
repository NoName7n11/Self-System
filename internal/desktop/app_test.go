package desktop

import (
	"context"
	"testing"

	"selfsystems/internal/domain"
	"selfsystems/internal/service"
)

// ── minimal in-memory fakes ───────────────────────────────────────────────────

type stubResourceRepo struct{ items []domain.Resource }

func (r *stubResourceRepo) GetByID(_ context.Context, id string) (*domain.Resource, error) {
	for i := range r.items {
		if r.items[i].ID == id {
			return &r.items[i], nil
		}
	}
	return nil, nil
}
func (r *stubResourceRepo) Create(_ context.Context, res *domain.Resource) error {
	r.items = append(r.items, *res)
	return nil
}
func (r *stubResourceRepo) Update(_ context.Context, res *domain.Resource) error {
	for i := range r.items {
		if r.items[i].ID == res.ID {
			r.items[i] = *res
		}
	}
	return nil
}
func (r *stubResourceRepo) Delete(_ context.Context, id string) error {
	out := r.items[:0]
	for _, item := range r.items {
		if item.ID != id {
			out = append(out, item)
		}
	}
	r.items = out
	return nil
}
func (r *stubResourceRepo) List(_ context.Context, limit, _ int) ([]domain.Resource, error) {
	if limit > 0 && len(r.items) > limit {
		return r.items[:limit], nil
	}
	return r.items, nil
}
func (r *stubResourceRepo) Search(_ context.Context, _ string, _ int) ([]domain.Resource, error) {
	return nil, nil
}
func (r *stubResourceRepo) UpdateCategory(_ context.Context, _, _ string, _ bool) error { return nil }
func (r *stubResourceRepo) UpdateExtractedData(_ context.Context, _ string, _ domain.ResourceExtractedData) error {
	return nil
}
func (r *stubResourceRepo) FindByURL(_ context.Context, _ string) (*domain.Resource, error) {
	return nil, nil
}
func (r *stubResourceRepo) IncrementCounter(_ context.Context, _ string) error { return nil }
func (r *stubResourceRepo) ListArchived(_ context.Context, _, _ int) ([]domain.Resource, error) {
	return nil, nil
}
func (r *stubResourceRepo) Archive(_ context.Context, _ string, _ domain.ArchiveReason) error {
	return nil
}
func (r *stubResourceRepo) Restore(_ context.Context, _ string) error { return nil }
func (r *stubResourceRepo) BulkArchive(_ context.Context, _ []string, _ domain.ArchiveReason) error {
	return nil
}
func (r *stubResourceRepo) BulkRestore(_ context.Context, _ []string) error { return nil }

type stubCategoryRepo struct{ items []domain.Category }

func (r *stubCategoryRepo) List(_ context.Context) ([]domain.Category, error) { return r.items, nil }
func (r *stubCategoryRepo) GetByID(_ context.Context, id string) (*domain.Category, error) {
	for i := range r.items {
		if r.items[i].ID == id {
			return &r.items[i], nil
		}
	}
	return nil, nil
}
func (r *stubCategoryRepo) GetByName(_ context.Context, _ string) (*domain.Category, error) {
	return nil, nil
}
func (r *stubCategoryRepo) Create(_ context.Context, c *domain.Category) error {
	r.items = append(r.items, *c)
	return nil
}
func (r *stubCategoryRepo) Update(_ context.Context, c *domain.Category) error {
	for i := range r.items {
		if r.items[i].ID == c.ID {
			r.items[i] = *c
		}
	}
	return nil
}
func (r *stubCategoryRepo) Delete(_ context.Context, id string) error {
	out := r.items[:0]
	for _, item := range r.items {
		if item.ID != id {
			out = append(out, item)
		}
	}
	r.items = out
	return nil
}
func (r *stubCategoryRepo) IncrementAccept(_ context.Context, _ string) error   { return nil }
func (r *stubCategoryRepo) IncrementOverride(_ context.Context, _ string) error { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestApp() *App {
	resRepo := &stubResourceRepo{}
	catRepo := &stubCategoryRepo{}
	catSvc := service.NewCategoryService(catRepo)
	classifier := service.NewCategoryClassifier(catRepo, nil)
	resSvc := service.NewResourceService(resRepo, catRepo, classifier, catSvc)
	todoSvc := service.NewTodoService(nil)
	reminderSvc := service.NewReminderService(nil)

	app := NewApp(AppOptions{
		Resources:  resSvc,
		Categories: catSvc,
		Todos:      todoSvc,
		Reminders:  reminderSvc,
	})
	app.ctx = context.Background()
	return app
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestApp_Startup_SetsContext(t *testing.T) {
	app := &App{}
	app.Startup(context.Background())
	if app.ctx == nil {
		t.Error("expected ctx to be set after Startup")
	}
}

func TestApp_GetResources_ReturnsEmpty(t *testing.T) {
	app := newTestApp()
	resources, err := app.GetResources(10, 0)
	if err != nil {
		t.Fatalf("GetResources: %v", err)
	}
	if resources == nil {
		resources = []domain.Resource{}
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}
}

func TestApp_CreateAndGetResource(t *testing.T) {
	app := newTestApp()
	created, err := app.CreateResource("https://example.com", "Test", "summary", "")
	if err != nil {
		t.Fatalf("CreateResource: %v", err)
	}
	if created.ID == "" {
		t.Error("expected resource ID to be set")
	}
	if created.URL != "https://example.com" {
		t.Errorf("URL = %q, want https://example.com", created.URL)
	}

	resources, err := app.GetResources(10, 0)
	if err != nil {
		t.Fatalf("GetResources: %v", err)
	}
	if len(resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(resources))
	}
}

func TestApp_DeleteResource(t *testing.T) {
	app := newTestApp()
	created, _ := app.CreateResource("https://example.com/del", "Del", "", "")

	deleted, err := app.DeleteResource(created.ID)
	if err != nil {
		t.Fatalf("DeleteResource: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	resources, _ := app.GetResources(10, 0)
	if len(resources) != 0 {
		t.Errorf("expected 0 resources after delete, got %d", len(resources))
	}
}

func TestApp_GetCategories_ReturnsEmpty(t *testing.T) {
	app := newTestApp()
	cats, err := app.GetCategories()
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if cats == nil {
		cats = []domain.Category{}
	}
	if len(cats) != 0 {
		t.Errorf("expected 0 categories, got %d", len(cats))
	}
}
