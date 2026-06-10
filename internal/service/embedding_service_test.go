package service

import (
	"context"
	"sort"
	"testing"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
)

type fakeEmbeddingRepo struct {
	stored map[string]domain.ResourceEmbedding
}

func newFakeEmbeddingRepo() *fakeEmbeddingRepo {
	return &fakeEmbeddingRepo{stored: map[string]domain.ResourceEmbedding{}}
}

func (r *fakeEmbeddingRepo) Upsert(_ context.Context, emb domain.ResourceEmbedding) error {
	r.stored[emb.ResourceID] = emb
	return nil
}
func (r *fakeEmbeddingRepo) Get(_ context.Context, id string) (*domain.ResourceEmbedding, error) {
	if e, ok := r.stored[id]; ok {
		return &e, nil
	}
	return nil, nil
}
func (r *fakeEmbeddingRepo) Delete(_ context.Context, id string) error {
	delete(r.stored, id)
	return nil
}
func (r *fakeEmbeddingRepo) SearchSimilar(_ context.Context, vector []float32, modelVersion string, limit int, threshold float64) ([]domain.EmbeddingMatch, error) {
	matches := make([]domain.EmbeddingMatch, 0)
	for id, e := range r.stored {
		if e.ModelVersion != modelVersion || len(e.Vector) != len(vector) {
			continue
		}
		score := ai.CosineSimilarity(vector, e.Vector)
		if score >= threshold {
			matches = append(matches, domain.EmbeddingMatch{ResourceID: id, Score: score})
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Score > matches[j].Score })
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func newLocalEmbeddingManager() *ai.Manager {
	mgr := ai.NewManager("heuristic")
	mgr.RegisterEmbedding(ai.NewLocalEmbeddingProvider())
	return mgr
}

func TestEmbeddingService_EmbedResource_StoresVector(t *testing.T) {
	repo := newFakeEmbeddingRepo()
	svc := NewEmbeddingService(repo, newLocalEmbeddingManager())

	result, err := svc.EmbedResource(context.Background(), "res-1", "machine learning and AI")
	if err != nil {
		t.Fatalf("EmbedResource: %v", err)
	}
	if result.ModelVersion != ai.LocalEmbeddingModelVersion {
		t.Errorf("ModelVersion = %q, want local", result.ModelVersion)
	}
	if result.Dim != ai.LocalEmbeddingDim {
		t.Errorf("Dim = %d, want %d", result.Dim, ai.LocalEmbeddingDim)
	}
	if result.EstimatedTokens <= 0 {
		t.Error("expected EstimatedTokens > 0")
	}

	stored, _ := repo.Get(context.Background(), "res-1")
	if stored == nil {
		t.Fatal("embedding not stored")
	}
	if len(stored.Vector) != ai.LocalEmbeddingDim {
		t.Errorf("stored vector dim = %d, want %d", len(stored.Vector), ai.LocalEmbeddingDim)
	}
}

func TestEmbeddingService_EmbedResource_EmptyText(t *testing.T) {
	svc := NewEmbeddingService(newFakeEmbeddingRepo(), newLocalEmbeddingManager())
	_, err := svc.EmbedResource(context.Background(), "res-1", "   ")
	if err == nil {
		t.Error("expected error for empty text")
	}
}

func TestEmbeddingService_SearchSimilar(t *testing.T) {
	repo := newFakeEmbeddingRepo()
	svc := NewEmbeddingService(repo, newLocalEmbeddingManager())
	ctx := context.Background()

	// Store two resources: one about AI, one about cooking.
	if _, err := svc.EmbedResource(ctx, "ai-res", "artificial intelligence machine learning neural networks"); err != nil {
		t.Fatalf("embed ai-res: %v", err)
	}
	if _, err := svc.EmbedResource(ctx, "food-res", "cooking pasta italian recipes dinner"); err != nil {
		t.Fatalf("embed food-res: %v", err)
	}

	// Query close to the AI resource.
	matches, err := svc.SearchSimilar(ctx, "machine learning artificial intelligence", 5, 0.0)
	if err != nil {
		t.Fatalf("SearchSimilar: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	if matches[0].ResourceID != "ai-res" {
		t.Errorf("top match = %s, want ai-res", matches[0].ResourceID)
	}
}
