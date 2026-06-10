package service

import (
	"context"
	"testing"

	"selfsystems/internal/domain"
)

type graphCategoryRepoStub struct {
	items []domain.Category
}

func (s graphCategoryRepoStub) List(ctx context.Context) ([]domain.Category, error) {
	return s.items, nil
}
func (s graphCategoryRepoStub) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	return nil, nil
}
func (s graphCategoryRepoStub) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	return nil, nil
}
func (s graphCategoryRepoStub) Create(ctx context.Context, c *domain.Category) error   { return nil }
func (s graphCategoryRepoStub) Update(ctx context.Context, c *domain.Category) error   { return nil }
func (s graphCategoryRepoStub) Delete(ctx context.Context, id string) error            { return nil }
func (s graphCategoryRepoStub) IncrementAccept(ctx context.Context, id string) error   { return nil }
func (s graphCategoryRepoStub) IncrementOverride(ctx context.Context, id string) error { return nil }

type graphResourceRepoStub struct {
	items []domain.Resource
}

func (s graphResourceRepoStub) GetByID(ctx context.Context, id string) (*domain.Resource, error) {
	return nil, nil
}
func (s graphResourceRepoStub) Create(ctx context.Context, r *domain.Resource) error { return nil }
func (s graphResourceRepoStub) Update(ctx context.Context, r *domain.Resource) error { return nil }
func (s graphResourceRepoStub) Delete(ctx context.Context, id string) error          { return nil }
func (s graphResourceRepoStub) List(ctx context.Context, limit, offset int) ([]domain.Resource, error) {
	return s.items, nil
}
func (s graphResourceRepoStub) Search(ctx context.Context, query string, limit int) ([]domain.Resource, error) {
	return nil, nil
}
func (s graphResourceRepoStub) UpdateCategory(ctx context.Context, resourceID, categoryID string, userOverride bool) error {
	return nil
}
func (s graphResourceRepoStub) UpdateExtractedData(_ context.Context, _ string, _ domain.ResourceExtractedData) error {
	return nil
}
func (s graphResourceRepoStub) FindByURL(_ context.Context, _ string) (*domain.Resource, error) {
	return nil, nil
}
func (s graphResourceRepoStub) IncrementCounter(_ context.Context, _ string) error { return nil }
func (s graphResourceRepoStub) ListArchived(_ context.Context, _, _ int) ([]domain.Resource, error) {
	return []domain.Resource{}, nil
}
func (s graphResourceRepoStub) Archive(_ context.Context, _ string, _ domain.ArchiveReason) error {
	return nil
}
func (s graphResourceRepoStub) Restore(_ context.Context, _ string) error { return nil }
func (s graphResourceRepoStub) BulkArchive(_ context.Context, _ []string, _ domain.ArchiveReason) error {
	return nil
}
func (s graphResourceRepoStub) BulkRestore(_ context.Context, _ []string) error { return nil }

func TestGraphBuildCreatesCategoryAndResourceNodes(t *testing.T) {
	categoryID := "cat-1"
	resourceID := "res-1"
	service := NewGraphService(
		graphCategoryRepoStub{items: []domain.Category{{ID: categoryID, Name: "AI"}}},
		graphResourceRepoStub{items: []domain.Resource{{ID: resourceID, Title: "AI Article", URL: "https://example.com", CategoryID: categoryID, CategoryName: "AI"}}},
	)

	graph, err := service.Build(context.Background(), 100)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if graph.Stats.CategoryCount != 1 {
		t.Fatalf("expected 1 category, got %d", graph.Stats.CategoryCount)
	}
	if graph.Stats.ResourceCount != 1 {
		t.Fatalf("expected 1 resource, got %d", graph.Stats.ResourceCount)
	}
	if graph.Stats.EdgeCount != 1 {
		t.Fatalf("expected 1 edge, got %d", graph.Stats.EdgeCount)
	}
}

func TestGraphBuildCreatesFallbackCategoryNodeForMissingCategory(t *testing.T) {
	service := NewGraphService(
		graphCategoryRepoStub{items: []domain.Category{}},
		graphResourceRepoStub{items: []domain.Resource{{ID: "res-1", Title: "Untyped", URL: "https://example.com", CategoryID: "missing-cat", CategoryName: "Recovered"}}},
	)

	graph, err := service.Build(context.Background(), 100)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if graph.Stats.CategoryCount != 1 {
		t.Fatalf("expected fallback category node, got %d", graph.Stats.CategoryCount)
	}
	if graph.Stats.EdgeCount != 1 {
		t.Fatalf("expected edge to fallback category, got %d", graph.Stats.EdgeCount)
	}
}
