package domain

import (
	"context"
	"errors"
	"time"
)

// ErrDuplicateResource is returned when a resource with the same URL already
// exists. The caller receives the existing resource and should inspect it.
var ErrDuplicateResource = errors.New("duplicate resource")

type CategoryRepository interface {
	List(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetByName(ctx context.Context, name string) (*Category, error)
	Create(ctx context.Context, c *Category) error
	Update(ctx context.Context, c *Category) error
	Delete(ctx context.Context, id string) error
	IncrementAccept(ctx context.Context, id string) error
	IncrementOverride(ctx context.Context, id string) error
}

type ResourceRepository interface {
	GetByID(ctx context.Context, id string) (*Resource, error)
	Create(ctx context.Context, r *Resource) error
	Update(ctx context.Context, r *Resource) error
	Delete(ctx context.Context, id string) error
	// List returns non-archived resources ordered by created_at DESC.
	List(ctx context.Context, limit, offset int) ([]Resource, error)
	// ListArchived returns only archived resources ordered by archived_at DESC.
	ListArchived(ctx context.Context, limit, offset int) ([]Resource, error)
	Search(ctx context.Context, query string, limit int) ([]Resource, error)
	UpdateCategory(ctx context.Context, resourceID, categoryID string, userOverride bool) error
	// UpdateExtractedData writes only the extracted_data column for a resource.
	// Called by background extraction workers; never blocks the primary write path.
	UpdateExtractedData(ctx context.Context, resourceID string, data ResourceExtractedData) error
	// FindByURL returns the existing resource for a given normalized URL, or nil
	// when not found. Used for exact-match duplicate detection.
	FindByURL(ctx context.Context, url string) (*Resource, error)
	// IncrementCounter increments save_count for the resource with the given ID.
	IncrementCounter(ctx context.Context, id string) error
	// Archive soft-archives a resource with the given reason.
	Archive(ctx context.Context, id string, reason ArchiveReason) error
	// Restore removes the archived state from a resource.
	Restore(ctx context.Context, id string) error
	// BulkArchive soft-archives up to 100 resources atomically.
	BulkArchive(ctx context.Context, ids []string, reason ArchiveReason) error
	// BulkRestore restores up to 100 archived resources atomically.
	BulkRestore(ctx context.Context, ids []string) error
}

// SimilarResourceRepository stores content-similarity links between resources.
type SimilarResourceRepository interface {
	Upsert(ctx context.Context, resourceID, similarID string, score float64) error
	ListByResource(ctx context.Context, resourceID string) ([]SimilarResource, error)
	Delete(ctx context.Context, resourceID, similarID string) error
}

type TodoRepository interface {
	GetByID(ctx context.Context, id string) (*Todo, error)
	Create(ctx context.Context, t *Todo) error
	Update(ctx context.Context, t *Todo) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Todo, error)
}

type ReminderRepository interface {
	GetByID(ctx context.Context, id string) (*Reminder, error)
	Create(ctx context.Context, r *Reminder) error
	Update(ctx context.Context, r *Reminder) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]Reminder, error)
}

// EmbeddingRepository stores and retrieves resource embedding vectors.
// SearchSimilar performs brute-force cosine similarity in Go (Change 7 WS3) —
// pure-Go, no native vector extension. Only vectors sharing modelVersion are
// comparable, so SearchSimilar filters by the query vector's model version.
type EmbeddingRepository interface {
	Upsert(ctx context.Context, emb ResourceEmbedding) error
	Get(ctx context.Context, resourceID string) (*ResourceEmbedding, error)
	Delete(ctx context.Context, resourceID string) error
	SearchSimilar(ctx context.Context, vector []float32, modelVersion string, limit int, threshold float64) ([]EmbeddingMatch, error)
}

// EmbeddingMatch is a single similarity search result.
type EmbeddingMatch struct {
	ResourceID string
	Score      float64 // cosine similarity in [-1, 1]
}

// GBUSFeatureStore is the persistence interface for GBUS feature tables.
// Implemented by the SQLite repo; Postgres pending.
type GBUSFeatureStore interface {
	UpsertCategoryFeature(ctx context.Context, userID, categoryID, signalType string, weight float64) error
	GetCategoryFeatures(ctx context.Context, userID, categoryID string) ([]GBUSCategoryFeature, error)
	ListAllCategoryFeatures(ctx context.Context) ([]GBUSCategoryFeature, error)
	UpsertResourceFeature(ctx context.Context, userID, resourceID, signalType string, weight float64) error
	GetResourceFeatures(ctx context.Context, userID, resourceID string) ([]GBUSResourceFeature, error)
	DeleteResourceFeatures(ctx context.Context, resourceID string) error
	PruneOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}
