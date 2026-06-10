package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"selfsystems/internal/domain"
)

// setupVectorTestDB opens a temp DB and seeds a category plus n resources so
// embeddings (which FK to resources) can be inserted.
func setupVectorTestDB(t *testing.T, n int) (*EmbeddingRepository, []string) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "vec_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	catRepo := NewCategoryRepository(db)
	cat := domain.Category{ID: "cat-1", Name: "Test", Source: domain.CategorySourceManual}
	if err := catRepo.Create(ctx, &cat); err != nil {
		t.Fatalf("create category: %v", err)
	}

	resRepo := NewResourceRepository(db)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("res-%d", i)
		ids[i] = id
		res := domain.Resource{
			ID:         id,
			URL:        fmt.Sprintf("https://example.com/%d", i),
			Host:       "example.com",
			Title:      fmt.Sprintf("Resource %d", i),
			CategoryID: cat.ID,
		}
		if err := resRepo.Create(ctx, &res); err != nil {
			t.Fatalf("create resource %d: %v", i, err)
		}
	}

	return NewEmbeddingRepository(db), ids
}

func TestEmbeddingRepository_UpsertAndGet(t *testing.T) {
	repo, ids := setupVectorTestDB(t, 1)
	ctx := context.Background()

	vec := []float32{0.1, 0.2, 0.3, 0.4}
	if err := repo.Upsert(ctx, domain.ResourceEmbedding{
		ResourceID:   ids[0],
		Vector:       vec,
		ModelVersion: "local-hash-v1",
		Dim:          4,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := repo.Get(ctx, ids[0])
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected embedding, got nil")
	}
	if got.ModelVersion != "local-hash-v1" || got.Dim != 4 {
		t.Errorf("metadata mismatch: %+v", got)
	}
	if len(got.Vector) != 4 {
		t.Fatalf("vector len = %d, want 4", len(got.Vector))
	}
	for i := range vec {
		if got.Vector[i] != vec[i] {
			t.Errorf("vector[%d] = %v, want %v (round-trip mismatch)", i, got.Vector[i], vec[i])
		}
	}
}

func TestEmbeddingRepository_UpsertOverwrites(t *testing.T) {
	repo, ids := setupVectorTestDB(t, 1)
	ctx := context.Background()

	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[0], Vector: []float32{1, 0}, ModelVersion: "v1", Dim: 2})
	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[0], Vector: []float32{0, 1, 0}, ModelVersion: "v2", Dim: 3})

	got, err := repo.Get(ctx, ids[0])
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ModelVersion != "v2" || got.Dim != 3 {
		t.Errorf("expected overwrite to v2/dim3, got %s/dim%d", got.ModelVersion, got.Dim)
	}
}

func TestEmbeddingRepository_SearchSimilar_Ordering(t *testing.T) {
	repo, ids := setupVectorTestDB(t, 3)
	ctx := context.Background()

	// Query will be [1, 0, 0]. Set up resources with decreasing similarity.
	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[0], Vector: []float32{1, 0, 0}, ModelVersion: "m", Dim: 3}) // identical
	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[1], Vector: []float32{0.7, 0.7, 0}, ModelVersion: "m", Dim: 3})
	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[2], Vector: []float32{0, 0, 1}, ModelVersion: "m", Dim: 3}) // orthogonal

	matches, err := repo.SearchSimilar(ctx, []float32{1, 0, 0}, "m", 3, -1.0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
	// Best match must be the identical vector.
	if matches[0].ResourceID != ids[0] {
		t.Errorf("top match = %s, want %s", matches[0].ResourceID, ids[0])
	}
	// Scores must be descending.
	for i := 1; i < len(matches); i++ {
		if matches[i].Score > matches[i-1].Score {
			t.Errorf("scores not descending: %v at %d > %v at %d", matches[i].Score, i, matches[i-1].Score, i-1)
		}
	}
	// Identical vector cosine ~ 1.
	if matches[0].Score < 0.99 {
		t.Errorf("identical vector score = %v, want ~1", matches[0].Score)
	}
}

func TestEmbeddingRepository_SearchSimilar_ThresholdFilters(t *testing.T) {
	repo, ids := setupVectorTestDB(t, 2)
	ctx := context.Background()

	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[0], Vector: []float32{1, 0, 0}, ModelVersion: "m", Dim: 3})
	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[1], Vector: []float32{0, 0, 1}, ModelVersion: "m", Dim: 3})

	// Threshold 0.5 should exclude the orthogonal vector (score 0).
	matches, err := repo.SearchSimilar(ctx, []float32{1, 0, 0}, "m", 10, 0.5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match above threshold, got %d", len(matches))
	}
	if matches[0].ResourceID != ids[0] {
		t.Errorf("match = %s, want %s", matches[0].ResourceID, ids[0])
	}
}

func TestEmbeddingRepository_SearchSimilar_ModelVersionIsolation(t *testing.T) {
	repo, ids := setupVectorTestDB(t, 2)
	ctx := context.Background()

	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[0], Vector: []float32{1, 0, 0}, ModelVersion: "model-a", Dim: 3})
	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[1], Vector: []float32{1, 0, 0}, ModelVersion: "model-b", Dim: 3})

	// Search within model-a only — model-b vector must not appear.
	matches, err := repo.SearchSimilar(ctx, []float32{1, 0, 0}, "model-a", 10, -1.0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match (model-a only), got %d", len(matches))
	}
	if matches[0].ResourceID != ids[0] {
		t.Errorf("match = %s, want %s", matches[0].ResourceID, ids[0])
	}
}

func TestEmbeddingRepository_Delete(t *testing.T) {
	repo, ids := setupVectorTestDB(t, 1)
	ctx := context.Background()

	_ = repo.Upsert(ctx, domain.ResourceEmbedding{ResourceID: ids[0], Vector: []float32{1, 0}, ModelVersion: "m", Dim: 2})
	if err := repo.Delete(ctx, ids[0]); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err := repo.Get(ctx, ids[0])
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}
