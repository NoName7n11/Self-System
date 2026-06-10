package service

import (
	"context"
	"strings"
	"testing"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
)

// ---- in-memory fakes --------------------------------------------------------

type fakeCategoryRepo struct {
	byID   map[string]domain.Category
	byName map[string]domain.Category
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{
		byID:   map[string]domain.Category{},
		byName: map[string]domain.Category{},
	}
}

func (r *fakeCategoryRepo) List(_ context.Context) ([]domain.Category, error) {
	items := make([]domain.Category, 0, len(r.byID))
	for _, c := range r.byID {
		items = append(items, c)
	}
	return items, nil
}
func (r *fakeCategoryRepo) GetByID(_ context.Context, id string) (*domain.Category, error) {
	if c, ok := r.byID[id]; ok {
		return &c, nil
	}
	return nil, nil
}
func (r *fakeCategoryRepo) GetByName(_ context.Context, name string) (*domain.Category, error) {
	if c, ok := r.byName[strings.ToLower(strings.TrimSpace(name))]; ok {
		return &c, nil
	}
	return nil, nil
}
func (r *fakeCategoryRepo) Create(_ context.Context, c *domain.Category) error {
	r.byID[c.ID] = *c
	r.byName[strings.ToLower(strings.TrimSpace(c.Name))] = *c
	return nil
}
func (r *fakeCategoryRepo) Update(_ context.Context, c *domain.Category) error {
	r.byID[c.ID] = *c
	r.byName[strings.ToLower(strings.TrimSpace(c.Name))] = *c
	return nil
}
func (r *fakeCategoryRepo) Delete(_ context.Context, id string) error {
	delete(r.byID, id)
	return nil
}
func (r *fakeCategoryRepo) IncrementAccept(_ context.Context, _ string) error   { return nil }
func (r *fakeCategoryRepo) IncrementOverride(_ context.Context, _ string) error { return nil }

type fakeResourceRepo struct {
	items map[string]domain.Resource
}

func newFakeResourceRepo() *fakeResourceRepo {
	return &fakeResourceRepo{items: map[string]domain.Resource{}}
}

func (r *fakeResourceRepo) GetByID(_ context.Context, id string) (*domain.Resource, error) {
	if res, ok := r.items[id]; ok {
		return &res, nil
	}
	return nil, nil
}
func (r *fakeResourceRepo) Create(_ context.Context, res *domain.Resource) error {
	r.items[res.ID] = *res
	return nil
}
func (r *fakeResourceRepo) Update(_ context.Context, res *domain.Resource) error {
	r.items[res.ID] = *res
	return nil
}
func (r *fakeResourceRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}
func (r *fakeResourceRepo) List(_ context.Context, _, _ int) ([]domain.Resource, error) {
	items := make([]domain.Resource, 0, len(r.items))
	for _, res := range r.items {
		items = append(items, res)
	}
	return items, nil
}
func (r *fakeResourceRepo) Search(_ context.Context, _ string, _ int) ([]domain.Resource, error) {
	return nil, nil
}
func (r *fakeResourceRepo) UpdateCategory(_ context.Context, id, catID string, override bool) error {
	if res, ok := r.items[id]; ok {
		res.CategoryID = catID
		res.UserOverride = override
		r.items[id] = res
	}
	return nil
}
func (r *fakeResourceRepo) UpdateExtractedData(_ context.Context, id string, data domain.ResourceExtractedData) error {
	if res, ok := r.items[id]; ok {
		res.ExtractedData = data
		r.items[id] = res
	}
	return nil
}
func (r *fakeResourceRepo) FindByURL(_ context.Context, _ string) (*domain.Resource, error) {
	return nil, nil
}
func (r *fakeResourceRepo) IncrementCounter(_ context.Context, _ string) error { return nil }
func (r *fakeResourceRepo) ListArchived(_ context.Context, _, _ int) ([]domain.Resource, error) {
	return []domain.Resource{}, nil
}
func (r *fakeResourceRepo) Archive(_ context.Context, _ string, _ domain.ArchiveReason) error {
	return nil
}
func (r *fakeResourceRepo) Restore(_ context.Context, _ string) error { return nil }
func (r *fakeResourceRepo) BulkArchive(_ context.Context, _ []string, _ domain.ArchiveReason) error {
	return nil
}
func (r *fakeResourceRepo) BulkRestore(_ context.Context, _ []string) error { return nil }

// fakeProvider is a configurable ai.Provider for tests.
type fakeProvider struct {
	name   string
	output ai.ClassificationOutput
	err    error
}

func (p fakeProvider) Name() string { return p.name }
func (p fakeProvider) ClassifySkim(_ context.Context, _ ai.ClassificationInput) (ai.ClassificationOutput, error) {
	if p.err != nil {
		return ai.ClassificationOutput{}, p.err
	}
	return p.output, nil
}

// ---- tests ------------------------------------------------------------------

func TestClassifier_AIPath_SetsSourceAI(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	mgr := ai.NewManager("openai")
	mgr.Register(fakeProvider{
		name: "openai",
		output: ai.ClassificationOutput{
			SuggestedCategory: "Technology",
			Confidence:        0.91,
			Reason:            "looks like tech",
		},
	})

	classifier := NewCategoryClassifier(catRepo, mgr)
	suggestion, err := classifier.Suggest(context.Background(), "https://example.com/go", "Go programming")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}

	if suggestion.Source != ClassificationSourceAI {
		t.Errorf("Source = %q, want %q", suggestion.Source, ClassificationSourceAI)
	}
	if suggestion.Score != 0.91 {
		t.Errorf("Score = %v, want 0.91", suggestion.Score)
	}
	if suggestion.Category.Name != "Technology" {
		t.Errorf("Category = %q, want Technology", suggestion.Category.Name)
	}
}

func TestClassifier_FallbackToHeuristic_SetsSourceHeuristic(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	// Provider is unavailable → classifier falls back to keyword heuristics.
	mgr := ai.NewManager("openai")
	mgr.Register(fakeProvider{name: "openai", err: ai.ErrProviderUnavailable})

	classifier := NewCategoryClassifier(catRepo, mgr)
	suggestion, err := classifier.Suggest(context.Background(), "https://github.com/foo/bar", "A code repo")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}

	if suggestion.Source != ClassificationSourceHeuristic {
		t.Errorf("Source = %q, want %q", suggestion.Source, ClassificationSourceHeuristic)
	}
}

func TestResourceService_Create_LowConfidence_FlagsNeedsReview(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	mgr := ai.NewManager("openai")
	mgr.Register(fakeProvider{
		name: "openai",
		output: ai.ClassificationOutput{
			SuggestedCategory: "Technology",
			Confidence:        0.50, // below default 0.85 threshold
		},
	})
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)
	svc := NewResourceService(resRepo, catRepo, classifier, catSvc)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL: "https://example.com/article",
		// No CategoryID / CategoryName → auto-classified path.
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !resource.ExtractedData.NeedsReview {
		t.Error("expected NeedsReview = true for confidence 0.50 < 0.85")
	}
	if resource.ExtractedData.ClassificationSource != ClassificationSourceAI {
		t.Errorf("ClassificationSource = %q, want ai", resource.ExtractedData.ClassificationSource)
	}
	if resource.ExtractedData.ClassificationConfidence != 0.50 {
		t.Errorf("ClassificationConfidence = %v, want 0.50", resource.ExtractedData.ClassificationConfidence)
	}
}

func TestResourceService_Create_HighConfidence_NoReview(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	mgr := ai.NewManager("openai")
	mgr.Register(fakeProvider{
		name: "openai",
		output: ai.ClassificationOutput{
			SuggestedCategory: "Technology",
			Confidence:        0.95, // above threshold
		},
	})
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)
	svc := NewResourceService(resRepo, catRepo, classifier, catSvc)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL: "https://example.com/article",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if resource.ExtractedData.NeedsReview {
		t.Error("expected NeedsReview = false for confidence 0.95 ≥ 0.85")
	}
}

func TestResourceService_Create_ManualCategory_NoReview(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	mgr := ai.NewManager("heuristic")
	mgr.Register(ai.NewHeuristicProvider())
	mgr.SetFallback("heuristic")
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)
	svc := NewResourceService(resRepo, catRepo, classifier, catSvc)

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL:          "https://example.com/article",
		CategoryName: "My Manual Category",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if resource.ExtractedData.NeedsReview {
		t.Error("manual category assignment should never need review")
	}
	if resource.ExtractedData.ClassificationSource != ClassificationSourceUser {
		t.Errorf("ClassificationSource = %q, want user", resource.ExtractedData.ClassificationSource)
	}
	if resource.ExtractedData.ClassificationConfidence != 1.0 {
		t.Errorf("ClassificationConfidence = %v, want 1.0", resource.ExtractedData.ClassificationConfidence)
	}
}

func TestResourceService_CustomThreshold(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	mgr := ai.NewManager("openai")
	mgr.Register(fakeProvider{
		name:   "openai",
		output: ai.ClassificationOutput{SuggestedCategory: "Technology", Confidence: 0.70},
	})
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)
	// Lower the threshold to 0.65 → confidence 0.70 should NOT need review.
	svc := NewResourceService(resRepo, catRepo, classifier, catSvc, WithClassificationThreshold(0.65))

	resource, err := svc.Create(context.Background(), CreateResourceInput{
		URL: "https://example.com/article",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if resource.ExtractedData.NeedsReview {
		t.Error("expected NeedsReview = false for confidence 0.70 ≥ custom threshold 0.65")
	}
}
