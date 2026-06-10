package sqlite

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"selfsystems/internal/domain"
)

// EmbeddingRepository stores resource embedding vectors and performs brute-force
// cosine similarity search in pure Go (Change 7 WS3). No native vector extension
// is used — vectors are stored as little-endian float32 BLOBs.
type EmbeddingRepository struct {
	db *sql.DB
}

var _ domain.EmbeddingRepository = (*EmbeddingRepository)(nil)

func NewEmbeddingRepository(db *sql.DB) *EmbeddingRepository {
	return &EmbeddingRepository{db: db}
}

func (r *EmbeddingRepository) Upsert(ctx context.Context, emb domain.ResourceEmbedding) error {
	if strings.TrimSpace(emb.ResourceID) == "" {
		return fmt.Errorf("resource id is required")
	}
	if len(emb.Vector) == 0 {
		return fmt.Errorf("vector is empty")
	}
	blob := encodeVector(emb.Vector)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO resource_embeddings (resource_id, vector, model_version, dim, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(resource_id) DO UPDATE SET
			vector = excluded.vector,
			model_version = excluded.model_version,
			dim = excluded.dim,
			created_at = excluded.created_at
	`, strings.TrimSpace(emb.ResourceID), blob, emb.ModelVersion, len(emb.Vector), nowRFC3339())
	if err != nil {
		return fmt.Errorf("upsert embedding: %w", err)
	}
	return nil
}

func (r *EmbeddingRepository) Get(ctx context.Context, resourceID string) (*domain.ResourceEmbedding, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT resource_id, vector, model_version, dim, created_at
		FROM resource_embeddings WHERE resource_id = ?
	`, strings.TrimSpace(resourceID))

	var (
		id        string
		blob      []byte
		model     string
		dim       int
		createdAt string
	)
	if err := row.Scan(&id, &blob, &model, &dim, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get embedding: %w", err)
	}

	return &domain.ResourceEmbedding{
		ResourceID:   id,
		Vector:       decodeVector(blob),
		ModelVersion: model,
		Dim:          dim,
		CreatedAt:    parseRFC3339(createdAt),
	}, nil
}

func (r *EmbeddingRepository) Delete(ctx context.Context, resourceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM resource_embeddings WHERE resource_id = ?`, strings.TrimSpace(resourceID))
	if err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}
	return nil
}

// SearchSimilar loads all vectors of the given model version and returns the
// top-`limit` by cosine similarity above `threshold`, sorted descending.
// Brute force is acceptable at personal-KMS scale (thousands of vectors).
func (r *EmbeddingRepository) SearchSimilar(ctx context.Context, vector []float32, modelVersion string, limit int, threshold float64) ([]domain.EmbeddingMatch, error) {
	if len(vector) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT resource_id, vector FROM resource_embeddings WHERE model_version = ?
	`, modelVersion)
	if err != nil {
		return nil, fmt.Errorf("query embeddings: %w", err)
	}
	defer rows.Close()

	matches := make([]domain.EmbeddingMatch, 0)
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, fmt.Errorf("scan embedding: %w", err)
		}
		candidate := decodeVector(blob)
		if len(candidate) != len(vector) {
			continue // dimension mismatch — not comparable
		}
		score := cosineSimilarity(vector, candidate)
		if score >= threshold {
			matches = append(matches, domain.EmbeddingMatch{ResourceID: id, Score: score})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate embeddings: %w", err)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

// ---- vector encoding --------------------------------------------------------

// encodeVector serialises a float32 slice as little-endian bytes (4 bytes each).
func encodeVector(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// decodeVector reverses encodeVector.
func decodeVector(b []byte) []float32 {
	n := len(b) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}

// cosineSimilarity returns the cosine similarity of two equal-length vectors.
func cosineSimilarity(a, b []float32) float64 {
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
