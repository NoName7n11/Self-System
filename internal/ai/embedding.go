package ai

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
)

// Embedding is a generated vector representation of text, tagged with the model
// version that produced it. Vectors from different model versions are not
// directly comparable.
type Embedding struct {
	Vector       []float32
	ModelVersion string
	Dim          int
}

// EmbeddingProvider produces embedding vectors for text. Real providers (OpenAI)
// and the deterministic LocalEmbeddingProvider both satisfy this interface.
type EmbeddingProvider interface {
	Name() string
	GenerateEmbedding(ctx context.Context, text string) (Embedding, error)
}

// RegisterEmbedding adds an embedding provider to the manager. Providers are
// tried in registration order; the first to succeed wins.
func (m *Manager) RegisterEmbedding(provider EmbeddingProvider) {
	if provider == nil {
		return
	}
	m.embeddingProviders = append(m.embeddingProviders, provider)
}

// GenerateEmbedding produces an embedding for text by trying each registered
// embedding provider in order. A provider returning ErrProviderUnavailable is
// skipped. Returns an error only if no provider produced an embedding.
func (m *Manager) GenerateEmbedding(ctx context.Context, text string) (Embedding, error) {
	var lastErr error
	for _, p := range m.embeddingProviders {
		emb, err := p.GenerateEmbedding(ctx, text)
		if err != nil {
			if err == ErrProviderUnavailable {
				continue
			}
			lastErr = err
			continue
		}
		if len(emb.Vector) > 0 {
			return emb, nil
		}
	}
	if lastErr != nil {
		return Embedding{}, lastErr
	}
	return Embedding{}, ErrProviderUnavailable
}

// ---- Local deterministic embedding provider --------------------------------

// LocalEmbeddingModelVersion identifies the local hashing embedder. Bump this
// when the algorithm changes so stale vectors can be detected and re-embedded.
const LocalEmbeddingModelVersion = "local-hash-v1"

// LocalEmbeddingDim is the fixed dimensionality of local embeddings.
const LocalEmbeddingDim = 256

// LocalEmbeddingProvider produces deterministic embeddings using feature hashing
// (the "hashing trick"): each token is hashed to a dimension and accumulated,
// then the vector is L2-normalised. This is offline, dependency-free, and gives
// meaningful cosine similarity based on shared vocabulary. It is the always-on
// fallback so the embedding pipeline works without any external API.
type LocalEmbeddingProvider struct {
	dim int
}

// NewLocalEmbeddingProvider returns a LocalEmbeddingProvider with the default dim.
func NewLocalEmbeddingProvider() *LocalEmbeddingProvider {
	return &LocalEmbeddingProvider{dim: LocalEmbeddingDim}
}

func (p *LocalEmbeddingProvider) Name() string { return "local-embedding" }

func (p *LocalEmbeddingProvider) GenerateEmbedding(_ context.Context, text string) (Embedding, error) {
	dim := p.dim
	if dim <= 0 {
		dim = LocalEmbeddingDim
	}
	vec := make([]float32, dim)

	tokens := embeddingTokenize(text)
	for _, tok := range tokens {
		h := fnv.New32a()
		_, _ = h.Write([]byte(tok))
		sum := h.Sum32()
		idx := int(sum % uint32(dim))
		// Sign bit derived from a second hash so collisions partially cancel.
		if (sum>>31)&1 == 1 {
			vec[idx] -= 1
		} else {
			vec[idx] += 1
		}
	}

	NormalizeVector(vec)

	return Embedding{
		Vector:       vec,
		ModelVersion: LocalEmbeddingModelVersion,
		Dim:          dim,
	}, nil
}

func embeddingTokenize(text string) []string {
	lower := strings.ToLower(text)
	fields := strings.FieldsFunc(lower, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) >= 2 { // drop single chars
			out = append(out, f)
		}
	}
	return out
}

// NormalizeVector L2-normalises v in place. A zero vector is left unchanged.
// Exported so test helpers in integration packages can normalize mock vectors.
func NormalizeVector(v []float32) {
	var sumSq float64
	for _, x := range v {
		sumSq += float64(x) * float64(x)
	}
	if sumSq == 0 {
		return
	}
	norm := float32(math.Sqrt(sumSq))
	for i := range v {
		v[i] /= norm
	}
}

// CosineSimilarity returns the cosine similarity of two equal-length vectors.
// Returns 0 if lengths differ or either vector is zero.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
