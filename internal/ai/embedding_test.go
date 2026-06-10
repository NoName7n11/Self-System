package ai

import (
	"context"
	"testing"
)

func TestLocalEmbeddingProvider_Deterministic(t *testing.T) {
	p := NewLocalEmbeddingProvider()
	ctx := context.Background()

	a, err := p.GenerateEmbedding(ctx, "machine learning neural networks")
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}
	b, err := p.GenerateEmbedding(ctx, "machine learning neural networks")
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}

	if a.Dim != LocalEmbeddingDim {
		t.Errorf("Dim = %d, want %d", a.Dim, LocalEmbeddingDim)
	}
	if a.ModelVersion != LocalEmbeddingModelVersion {
		t.Errorf("ModelVersion = %q, want %q", a.ModelVersion, LocalEmbeddingModelVersion)
	}
	if len(a.Vector) != LocalEmbeddingDim {
		t.Fatalf("vector len = %d, want %d", len(a.Vector), LocalEmbeddingDim)
	}
	// Determinism: identical input → identical vector.
	for i := range a.Vector {
		if a.Vector[i] != b.Vector[i] {
			t.Fatalf("non-deterministic at dim %d: %v vs %v", i, a.Vector[i], b.Vector[i])
		}
	}
}

func TestLocalEmbeddingProvider_Normalized(t *testing.T) {
	p := NewLocalEmbeddingProvider()
	emb, err := p.GenerateEmbedding(context.Background(), "some text content here")
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}
	// L2 norm should be ~1 for non-empty input.
	var sumSq float64
	for _, x := range emb.Vector {
		sumSq += float64(x) * float64(x)
	}
	if sumSq < 0.99 || sumSq > 1.01 {
		t.Errorf("vector not normalized: sumSq = %v, want ~1.0", sumSq)
	}
}

func TestLocalEmbedding_SimilarTextHigherSimilarity(t *testing.T) {
	p := NewLocalEmbeddingProvider()
	ctx := context.Background()

	base, _ := p.GenerateEmbedding(ctx, "artificial intelligence and machine learning research")
	similar, _ := p.GenerateEmbedding(ctx, "machine learning and artificial intelligence study")
	different, _ := p.GenerateEmbedding(ctx, "cooking pasta recipes for dinner tonight")

	simScore := CosineSimilarity(base.Vector, similar.Vector)
	diffScore := CosineSimilarity(base.Vector, different.Vector)

	if simScore <= diffScore {
		t.Errorf("expected similar text to score higher: similar=%v different=%v", simScore, diffScore)
	}
}

func TestManager_GenerateEmbedding_FallsBackToLocal(t *testing.T) {
	mgr := NewManager("heuristic")
	// Register only the local provider.
	mgr.RegisterEmbedding(NewLocalEmbeddingProvider())

	emb, err := mgr.GenerateEmbedding(context.Background(), "fallback test")
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}
	if emb.ModelVersion != LocalEmbeddingModelVersion {
		t.Errorf("ModelVersion = %q, want local", emb.ModelVersion)
	}
}

func TestManager_GenerateEmbedding_SkipsUnavailable(t *testing.T) {
	mgr := NewManager("heuristic")
	// An unavailable provider first, then local — manager should skip to local.
	mgr.RegisterEmbedding(unavailableEmbeddingProvider{})
	mgr.RegisterEmbedding(NewLocalEmbeddingProvider())

	emb, err := mgr.GenerateEmbedding(context.Background(), "skip unavailable")
	if err != nil {
		t.Fatalf("GenerateEmbedding: %v", err)
	}
	if emb.ModelVersion != LocalEmbeddingModelVersion {
		t.Errorf("expected fallback to local, got %q", emb.ModelVersion)
	}
}

func TestManager_GenerateEmbedding_NoProviders(t *testing.T) {
	mgr := NewManager("heuristic")
	_, err := mgr.GenerateEmbedding(context.Background(), "no providers")
	if err == nil {
		t.Error("expected error when no embedding providers registered")
	}
}

func TestCosineSimilarity_EdgeCases(t *testing.T) {
	if got := CosineSimilarity(nil, nil); got != 0 {
		t.Errorf("nil vectors: got %v, want 0", got)
	}
	if got := CosineSimilarity([]float32{1, 0}, []float32{1, 0, 0}); got != 0 {
		t.Errorf("mismatched lengths: got %v, want 0", got)
	}
	if got := CosineSimilarity([]float32{1, 0}, []float32{1, 0}); got < 0.99 {
		t.Errorf("identical vectors: got %v, want ~1", got)
	}
}

type unavailableEmbeddingProvider struct{}

func (unavailableEmbeddingProvider) Name() string { return "unavailable" }
func (unavailableEmbeddingProvider) GenerateEmbedding(_ context.Context, _ string) (Embedding, error) {
	return Embedding{}, ErrProviderUnavailable
}
