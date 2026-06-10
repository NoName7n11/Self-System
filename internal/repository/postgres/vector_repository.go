package postgres

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"selfsystems/internal/domain"
)

// EmbeddingRepository is the Postgres implementation of domain.EmbeddingRepository.
// Vectors are stored as little-endian float32 BYTEA; similarity is brute-force
// cosine in Go (Change 7 WS3 — pure-Go, no pgvector dependency).
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
	createdAt := emb.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO resource_embeddings (resource_id, vector, model_version, dim, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (resource_id) DO UPDATE SET
			vector = EXCLUDED.vector,
			model_version = EXCLUDED.model_version,
			dim = EXCLUDED.dim,
			created_at = EXCLUDED.created_at
	`, strings.TrimSpace(emb.ResourceID), blob, emb.ModelVersion, len(emb.Vector), createdAt)
	if err != nil {
		return fmt.Errorf("upsert embedding: %w", err)
	}
	return nil
}

func (r *EmbeddingRepository) Get(ctx context.Context, resourceID string) (*domain.ResourceEmbedding, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT resource_id, vector, model_version, dim, created_at
		FROM resource_embeddings WHERE resource_id = $1
	`, strings.TrimSpace(resourceID))

	var (
		id        string
		blob      []byte
		model     string
		dim       int
		createdAt time.Time
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
		CreatedAt:    createdAt,
	}, nil
}

func (r *EmbeddingRepository) Delete(ctx context.Context, resourceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM resource_embeddings WHERE resource_id = $1`, strings.TrimSpace(resourceID))
	if err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}
	return nil
}

func (r *EmbeddingRepository) SearchSimilar(ctx context.Context, vector []float32, modelVersion string, limit int, threshold float64) ([]domain.EmbeddingMatch, error) {
	if len(vector) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT resource_id, vector FROM resource_embeddings WHERE model_version = $1
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
			continue
		}
		score := cosineSimilarity(vector, candidate)
		if score >= threshold {
			matches = append(matches, domain.EmbeddingMatch{ResourceID: id, Score: score})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate embeddings: %w", err)
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].Score > matches[j].Score })
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

// ---- vector encoding (shared shape with the sqlite repo) --------------------

func encodeVector(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func decodeVector(b []byte) []float32 {
	n := len(b) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}

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
