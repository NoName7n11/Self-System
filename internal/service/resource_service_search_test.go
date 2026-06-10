package service

import (
	"context"
	"testing"
	"time"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
)

// ---- helpers ----------------------------------------------------------------

func makeResourceSvc(t *testing.T) (*ResourceService, *fakeResourceRepo, *fakeEmbeddingRepo) {
	t.Helper()
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	embRepo := newFakeEmbeddingRepo()

	mgr := ai.NewManager("heuristic")
	mgr.Register(ai.NewHeuristicProvider())
	mgr.SetFallback("heuristic")
	mgr.RegisterEmbedding(ai.NewLocalEmbeddingProvider())

	embSvc := NewEmbeddingService(embRepo, mgr)
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)

	svc := NewResourceService(resRepo, catRepo, classifier, catSvc,
		WithResourceEmbeddingService(embSvc),
	)
	return svc, resRepo, embRepo
}

func seedResource(t *testing.T, resRepo *fakeResourceRepo, embRepo *fakeEmbeddingRepo, mgr *ai.Manager, id, title, catID string) {
	t.Helper()
	resRepo.items[id] = domain.Resource{
		ID:           id,
		URL:          "https://example.com/" + id,
		Host:         "example.com",
		Title:        title,
		CategoryID:   catID,
		CategoryName: "Test",
		CreatedAt:    time.Now().UTC(),
	}
	// Generate and store an embedding for the resource.
	emb, err := mgr.GenerateEmbedding(context.Background(), title)
	if err != nil {
		t.Fatalf("generate embedding for %s: %v", id, err)
	}
	_ = embRepo.Upsert(context.Background(), domain.ResourceEmbedding{
		ResourceID:   id,
		Vector:       emb.Vector,
		ModelVersion: emb.ModelVersion,
		Dim:          emb.Dim,
	})
}

// ---- tests ------------------------------------------------------------------

func TestResourceService_SemanticSearch_VectorPath(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	embRepo := newFakeEmbeddingRepo()

	mgr := ai.NewManager("heuristic")
	mgr.Register(ai.NewHeuristicProvider())
	mgr.SetFallback("heuristic")
	mgr.RegisterEmbedding(ai.NewLocalEmbeddingProvider())
	embSvc := NewEmbeddingService(embRepo, mgr)
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)
	svc := NewResourceService(resRepo, catRepo, classifier, catSvc,
		WithResourceEmbeddingService(embSvc),
	)

	seedResource(t, resRepo, embRepo, mgr, "ai-1", "artificial intelligence and machine learning", "cat-1")
	seedResource(t, resRepo, embRepo, mgr, "cook-1", "pasta recipes and italian cooking", "cat-2")

	results, err := svc.SemanticSearch(context.Background(), "machine learning", 10)
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].ID != "ai-1" {
		t.Errorf("top result = %s, want ai-1", results[0].ID)
	}
}

func TestResourceService_SemanticSearch_FallbackWhenNoEmbeddings(t *testing.T) {
	// embeddingSvc configured but no embeddings stored → falls back to token search
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	embRepo := newFakeEmbeddingRepo() // empty

	mgr := ai.NewManager("heuristic")
	mgr.Register(ai.NewHeuristicProvider())
	mgr.SetFallback("heuristic")
	mgr.RegisterEmbedding(ai.NewLocalEmbeddingProvider())
	embSvc := NewEmbeddingService(embRepo, mgr)
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)
	svc := NewResourceService(resRepo, catRepo, classifier, catSvc,
		WithResourceEmbeddingService(embSvc),
	)

	resRepo.items["r1"] = domain.Resource{
		ID: "r1", Title: "neural network tutorial", URL: "https://example.com/r1",
		CategoryID: "cat-1", CategoryName: "AI", CreatedAt: time.Now(),
	}

	// No embeddings stored → vector search returns 0 results → token fallback fires
	results, err := svc.SemanticSearch(context.Background(), "neural network", 10)
	if err != nil {
		t.Fatalf("SemanticSearch: %v", err)
	}
	// Token search should still find the resource by title keywords
	if len(results) == 0 {
		t.Error("expected token search fallback to return results")
	}
}

func TestResourceService_SemanticSearch_EmptyQuery(t *testing.T) {
	svc, _, _ := makeResourceSvc(t)
	results, err := svc.SemanticSearch(context.Background(), "  ", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestResourceService_HybridSearch_MergesResults(t *testing.T) {
	catRepo := newFakeCategoryRepo()
	resRepo := newFakeResourceRepo()
	embRepo := newFakeEmbeddingRepo()

	mgr := ai.NewManager("heuristic")
	mgr.Register(ai.NewHeuristicProvider())
	mgr.SetFallback("heuristic")
	mgr.RegisterEmbedding(ai.NewLocalEmbeddingProvider())
	embSvc := NewEmbeddingService(embRepo, mgr)
	catSvc := NewCategoryService(catRepo)
	classifier := NewCategoryClassifier(catRepo, mgr)
	svc := NewResourceService(resRepo, catRepo, classifier, catSvc,
		WithResourceEmbeddingService(embSvc),
	)

	seedResource(t, resRepo, embRepo, mgr, "ai-1", "artificial intelligence research", "cat-1")
	seedResource(t, resRepo, embRepo, mgr, "ai-2", "machine learning algorithms", "cat-1")
	seedResource(t, resRepo, embRepo, mgr, "cook-1", "pasta and italian food", "cat-2")

	results, err := svc.HybridSearch(context.Background(), "artificial intelligence", 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results from hybrid search")
	}
	// Top result should be AI-related (both paths favour it)
	if results[0].ID == "cook-1" {
		t.Error("cooking resource should not rank first for AI query")
	}
}

func TestMergeHybridResults_Deduplication(t *testing.T) {
	r1 := domain.Resource{ID: "r1", Title: "shared"}
	r2 := domain.Resource{ID: "r2", Title: "keyword only"}
	r3 := domain.Resource{ID: "r3", Title: "semantic only"}

	keyword := []domain.Resource{r1, r2}
	semantic := []domain.Resource{r1, r3} // r1 appears in both

	merged := mergeHybridResults(keyword, semantic, 10)

	seen := map[string]int{}
	for _, r := range merged {
		seen[r.ID]++
	}
	for id, count := range seen {
		if count > 1 {
			t.Errorf("resource %s appears %d times (expected 1)", id, count)
		}
	}
	if len(merged) != 3 {
		t.Errorf("expected 3 unique results, got %d", len(merged))
	}
}
