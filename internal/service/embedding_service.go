package service

import (
	"context"
	"fmt"
	"strings"

	"selfsystems/internal/ai"
	"selfsystems/internal/domain"
)

// EmbeddingService generates embedding vectors via the AI manager and stores
// them through the embedding repository (Change 7 WS2). It is transport-agnostic
// and used by the deep processing worker after content extraction.
type EmbeddingService struct {
	repo    domain.EmbeddingRepository
	manager *ai.Manager
}

// NewEmbeddingService builds an EmbeddingService. Both dependencies are required.
func NewEmbeddingService(repo domain.EmbeddingRepository, manager *ai.Manager) *EmbeddingService {
	return &EmbeddingService{repo: repo, manager: manager}
}

// EmbedResourceResult reports what was produced by an embedding call.
type EmbedResourceResult struct {
	ModelVersion    string
	Dim             int
	EstimatedTokens int
}

// EmbedResource generates an embedding for the given text and stores it against
// resourceID. Returns metadata for token-budget accounting. A blank text or a
// missing provider is a no-op error the caller may treat as non-fatal.
func (s *EmbeddingService) EmbedResource(ctx context.Context, resourceID, text string) (EmbedResourceResult, error) {
	if s == nil || s.repo == nil || s.manager == nil {
		return EmbedResourceResult{}, fmt.Errorf("embedding service is not configured")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return EmbedResourceResult{}, fmt.Errorf("embedding text is empty")
	}

	emb, err := s.manager.GenerateEmbedding(ctx, trimmed)
	if err != nil {
		return EmbedResourceResult{}, err
	}

	if err := s.repo.Upsert(ctx, domain.ResourceEmbedding{
		ResourceID:   resourceID,
		Vector:       emb.Vector,
		ModelVersion: emb.ModelVersion,
		Dim:          emb.Dim,
	}); err != nil {
		return EmbedResourceResult{}, err
	}

	return EmbedResourceResult{
		ModelVersion:    emb.ModelVersion,
		Dim:             emb.Dim,
		EstimatedTokens: estimateEmbeddingTokens(trimmed),
	}, nil
}

// GenerateQueryEmbedding embeds a search query on the fly (not stored).
func (s *EmbeddingService) GenerateQueryEmbedding(ctx context.Context, query string) (ai.Embedding, error) {
	if s == nil || s.manager == nil {
		return ai.Embedding{}, fmt.Errorf("embedding service is not configured")
	}
	return s.manager.GenerateEmbedding(ctx, strings.TrimSpace(query))
}

// SearchSimilar embeds the query and returns the nearest resource IDs.
func (s *EmbeddingService) SearchSimilar(ctx context.Context, query string, limit int, threshold float64) ([]domain.EmbeddingMatch, error) {
	emb, err := s.GenerateQueryEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	return s.repo.SearchSimilar(ctx, emb.Vector, emb.ModelVersion, limit, threshold)
}

// estimateEmbeddingTokens approximates token count as ~1 token per 4 characters.
func estimateEmbeddingTokens(text string) int {
	n := len(text) / 4
	if n < 1 {
		n = 1
	}
	return n
}
