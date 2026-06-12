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

// UpsertCategoryFeature increments or inserts a per-(user, category) signal
// weight row. evidence_count tracks explicit-intent signals (weight >=
// domain.ExplicitIntentWeightThreshold); confidence ramps linearly with
// evidence_count up to domain.ConfidenceEvidenceThreshold.
func (r *GBUSRepository) UpsertCategoryFeature(ctx context.Context, userID, categoryID, signalType string, weight float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	evidenceDelta := 0
	if weight >= domain.ExplicitIntentWeightThreshold {
		evidenceDelta = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO gbus_category_features (user_id, category_id, signal_type, total_weight, signal_count, evidence_count, confidence, last_signal_at)
		VALUES (?, ?, ?, ?, 1, ?, MIN(1.0, ?/?), ?)
		ON CONFLICT(category_id, signal_type) DO UPDATE SET
			user_id        = excluded.user_id,
			total_weight   = total_weight + excluded.total_weight,
			signal_count   = signal_count + 1,
			evidence_count = evidence_count + ?,
			confidence     = MIN(1.0, (evidence_count + ?) * 1.0 / ?),
			last_signal_at = excluded.last_signal_at
	`, userID, categoryID, signalType, weight, evidenceDelta, float64(evidenceDelta), float64(domain.ConfidenceEvidenceThreshold), now,
		evidenceDelta, evidenceDelta, domain.ConfidenceEvidenceThreshold)
	return err
}

// GetCategoryFeatures returns all feature rows for a (user, category).
func (r *GBUSRepository) GetCategoryFeatures(ctx context.Context, userID, categoryID string) ([]domain.GBUSCategoryFeature, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, category_id, signal_type, total_weight, signal_count, evidence_count, confidence, last_signal_at
		FROM gbus_category_features
		WHERE user_id = ? AND category_id = ?
	`, userID, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCategoryFeatures(rows)
}

// ListAllCategoryFeatures returns every category feature row — used by the training pipeline.
func (r *GBUSRepository) ListAllCategoryFeatures(ctx context.Context) ([]domain.GBUSCategoryFeature, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, category_id, signal_type, total_weight, signal_count, evidence_count, confidence, last_signal_at
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
		if err := rows.Scan(&f.UserID, &f.CategoryID, &f.SignalType, &f.TotalWeight, &f.SignalCount, &f.EvidenceCount, &f.Confidence, &lastAt); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339, lastAt); err == nil {
			f.LastSignalAt = t
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// UpsertResourceFeature increments or inserts a per-(user, resource) signal weight row.
func (r *GBUSRepository) UpsertResourceFeature(ctx context.Context, userID, resourceID, signalType string, weight float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO gbus_resource_features (user_id, resource_id, signal_type, total_weight, signal_count, last_signal_at)
		VALUES (?, ?, ?, ?, 1, ?)
		ON CONFLICT(resource_id, signal_type) DO UPDATE SET
			user_id        = excluded.user_id,
			total_weight   = total_weight + excluded.total_weight,
			signal_count   = signal_count + 1,
			last_signal_at = excluded.last_signal_at
	`, userID, resourceID, signalType, weight, now)
	return err
}

// GetResourceFeatures returns all feature rows for a (user, resource).
func (r *GBUSRepository) GetResourceFeatures(ctx context.Context, userID, resourceID string) ([]domain.GBUSResourceFeature, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, resource_id, signal_type, total_weight, signal_count, last_signal_at
		FROM gbus_resource_features
		WHERE user_id = ? AND resource_id = ?
	`, userID, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.GBUSResourceFeature
	for rows.Next() {
		var f domain.GBUSResourceFeature
		var lastAt string
		if err := rows.Scan(&f.UserID, &f.ResourceID, &f.SignalType, &f.TotalWeight, &f.SignalCount, &lastAt); err != nil {
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