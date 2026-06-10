package sqlite

import (
	"context"
	"database/sql"
	"time"

	"selfsystems/internal/domain"
)

// GBUSRepository implements gbus.FeatureStore against SQLite.
type GBUSRepository struct {
	db *sql.DB
}

func NewGBUSRepository(db *sql.DB) *GBUSRepository {
	return &GBUSRepository{db: db}
}

// UpsertCategoryFeature increments or inserts a per-category signal weight row.
func (r *GBUSRepository) UpsertCategoryFeature(ctx context.Context, categoryID, signalType string, weight float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO gbus_category_features (category_id, signal_type, total_weight, signal_count, last_signal_at)
		VALUES (?, ?, ?, 1, ?)
		ON CONFLICT(category_id, signal_type) DO UPDATE SET
			total_weight  = total_weight + excluded.total_weight,
			signal_count  = signal_count + 1,
			last_signal_at = excluded.last_signal_at
	`, categoryID, signalType, weight, now)
	return err
}

// GetCategoryFeatures returns all feature rows for a category.
func (r *GBUSRepository) GetCategoryFeatures(ctx context.Context, categoryID string) ([]domain.GBUSCategoryFeature, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category_id, signal_type, total_weight, signal_count, last_signal_at
		FROM gbus_category_features
		WHERE category_id = ?
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCategoryFeatures(rows)
}

// ListAllCategoryFeatures returns every category feature row — used by the training pipeline.
func (r *GBUSRepository) ListAllCategoryFeatures(ctx context.Context) ([]domain.GBUSCategoryFeature, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category_id, signal_type, total_weight, signal_count, last_signal_at
		FROM gbus_category_features
		ORDER BY category_id, signal_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCategoryFeatures(rows)
}

func scanCategoryFeatures(rows *sql.Rows) ([]domain.GBUSCategoryFeature, error) {
	var out []domain.GBUSCategoryFeature
	for rows.Next() {
		var f domain.GBUSCategoryFeature
		var lastAt string
		if err := rows.Scan(&f.CategoryID, &f.SignalType, &f.TotalWeight, &f.SignalCount, &lastAt); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, lastAt); err == nil {
			f.LastSignalAt = t
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// UpsertResourceFeature increments or inserts a per-resource signal weight row.
func (r *GBUSRepository) UpsertResourceFeature(ctx context.Context, resourceID, signalType string, weight float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO gbus_resource_features (resource_id, signal_type, total_weight, signal_count, last_signal_at)
		VALUES (?, ?, ?, 1, ?)
		ON CONFLICT(resource_id, signal_type) DO UPDATE SET
			total_weight   = total_weight + excluded.total_weight,
			signal_count   = signal_count + 1,
			last_signal_at = excluded.last_signal_at
	`, resourceID, signalType, weight, now)
	return err
}

// GetResourceFeatures returns all feature rows for a resource.
func (r *GBUSRepository) GetResourceFeatures(ctx context.Context, resourceID string) ([]domain.GBUSResourceFeature, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT resource_id, signal_type, total_weight, signal_count, last_signal_at
		FROM gbus_resource_features
		WHERE resource_id = ?
	`, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GBUSResourceFeature
	for rows.Next() {
		var f domain.GBUSResourceFeature
		var lastAt string
		if err := rows.Scan(&f.ResourceID, &f.SignalType, &f.TotalWeight, &f.SignalCount, &lastAt); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, lastAt); err == nil {
			f.LastSignalAt = t
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// DeleteResourceFeatures removes all feature rows for a resource (called on delete).
func (r *GBUSRepository) DeleteResourceFeatures(ctx context.Context, resourceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM gbus_resource_features WHERE resource_id = ?`, resourceID)
	return err
}

// PruneOlderThan deletes feature rows whose last_signal_at is before cutoff.
func (r *GBUSRepository) PruneOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	cutoffStr := cutoff.UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM gbus_category_features WHERE last_signal_at < ?
	`, cutoffStr)
	if err != nil {
		return 0, err
	}
	n1, _ := res.RowsAffected()

	res2, err := r.db.ExecContext(ctx, `
		DELETE FROM gbus_resource_features WHERE last_signal_at < ?
	`, cutoffStr)
	if err != nil {
		return int(n1), err
	}
	n2, _ := res2.RowsAffected()
	return int(n1 + n2), nil
}
