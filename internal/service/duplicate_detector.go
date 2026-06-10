package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"selfsystems/internal/domain"
	"selfsystems/internal/eventstore"
)

// DuplicateDetector checks for content-similar resources after embeddings are
// generated. It runs as part of the deep processing pipeline (WS2).
// Similarity is determined by cosine distance; only matches above the threshold
// are linked. No automatic merge — users see the suggestion only.
type DuplicateDetector struct {
	embeddings domain.EmbeddingRepository
	similar    domain.SimilarResourceRepository
	resources  domain.ResourceRepository
	eventStore eventstore.Store
	projectors *eventstore.ProjectorRegistry
	// SimilarityThreshold is the minimum cosine similarity score to flag a pair.
	// Configurable; defaults to 0.92.
	SimilarityThreshold float64
}

// NewDuplicateDetector creates a detector. similarRepo and eventStore may be nil
// when the feature is not yet wired (graceful no-op).
func NewDuplicateDetector(
	embeddings domain.EmbeddingRepository,
	similar domain.SimilarResourceRepository,
	resources domain.ResourceRepository,
	eventStore eventstore.Store,
	projectors *eventstore.ProjectorRegistry,
	threshold float64,
) *DuplicateDetector {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.92
	}
	return &DuplicateDetector{
		embeddings:          embeddings,
		similar:             similar,
		resources:           resources,
		eventStore:          eventStore,
		projectors:          projectors,
		SimilarityThreshold: threshold,
	}
}

// Check runs a similarity search for resourceID and stores any links found.
// It is safe to call even when the resource has no embedding yet — it returns
// silently in that case.
func (d *DuplicateDetector) Check(ctx context.Context, resourceID string) error {
	if d.embeddings == nil || d.similar == nil {
		return nil
	}

	emb, err := d.embeddings.Get(ctx, resourceID)
	if err != nil || emb == nil {
		return nil // no embedding yet — skip
	}

	matches, err := d.embeddings.SearchSimilar(ctx, emb.Vector, emb.ModelVersion, 10, d.SimilarityThreshold)
	if err != nil {
		return fmt.Errorf("duplicate detector search: %w", err)
	}

	for _, m := range matches {
		if m.ResourceID == resourceID {
			continue // skip self
		}

		if err := d.similar.Upsert(ctx, resourceID, m.ResourceID, m.Score); err != nil {
			slog.Warn("duplicate detector: failed to upsert similarity link",
				"resource_id", resourceID, "similar_id", m.ResourceID, "error", err)
			continue
		}

		if d.eventStore != nil && d.projectors != nil {
			d.emitSimilarityEvent(ctx, resourceID, m.ResourceID, m.Score)
		}
	}
	return nil
}

func (d *DuplicateDetector) emitSimilarityEvent(ctx context.Context, resourceID, similarID string, score float64) {
	payload, err := marshalJSON(eventstore.ResourceSimilarityDetectedPayload{
		SimilarResourceID: similarID,
		SimilarityScore:   score,
	})
	if err != nil {
		return
	}

	version, err := aggregateLatestVersion(ctx, d.eventStore, resourceID)
	if err != nil {
		return
	}

	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   resourceID,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceSimilarityDetected,
		EventVersion:  version + 1,
		Payload:       payload,
	}

	committed, err := appendWithTx(ctx, d.eventStore, d.projectors, evt, nil)
	if err != nil {
		slog.Warn("duplicate detector: failed to emit similarity event",
			"resource_id", resourceID, "error", err)
		return
	}
	d.projectors.ApplyAsync(ctx, committed)
}
