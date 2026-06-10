// Package migration provides tools for seeding the event log from existing
// state tables and verifying projection parity.
package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"selfsystems/internal/eventstore"
)

// BackfillConfig controls the resource backfill job.
type BackfillConfig struct {
	// BatchSize is the number of events appended per transaction. Defaults to 500.
	BatchSize int
	// CorrelationID links all events created in this run. Auto-generated when empty.
	CorrelationID string
	// OnProgress is called after each batch with (processed, total).
	OnProgress func(processed, total int)
}

// BackfillResult reports the outcome of a backfill run.
type BackfillResult struct {
	Processed     int // ResourceImported events appended
	Skipped       int // resources that already had events
	CorrelationID string
	Duration      time.Duration
}

// resourceRow holds fields read from the resources+categories join.
type resourceRow struct {
	id           string
	url          string
	host         string
	title        string
	summary      string
	categoryID   string
	categoryName string
	userOverride bool
	createdAt    time.Time
	updatedAt    time.Time
}

// RunResourceBackfill reads all rows from the resources table that have no
// events in the event store and appends a ResourceImported event for each.
//
// Idempotent: resources that already have at least one event are skipped.
// Resumable: if interrupted, a second run continues where the first left off.
func RunResourceBackfill(ctx context.Context, db *sql.DB, store eventstore.Store, cfg BackfillConfig) (BackfillResult, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.CorrelationID == "" {
		cfg.CorrelationID = uuid.NewString()
	}
	start := time.Now()

	seeded, err := resourceIDsWithEvents(ctx, db)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("query existing resource events: %w", err)
	}

	rows, err := allResourceRows(ctx, db)
	if err != nil {
		return BackfillResult{}, fmt.Errorf("query resources: %w", err)
	}

	var toFill []resourceRow
	skipped := 0
	for _, r := range rows {
		if seeded[r.id] {
			skipped++
		} else {
			toFill = append(toFill, r)
		}
	}
	total := len(toFill)

	processed := 0
	for batchStart := 0; batchStart < len(toFill); batchStart += cfg.BatchSize {
		end := batchStart + cfg.BatchSize
		if end > len(toFill) {
			end = len(toFill)
		}
		batch := toFill[batchStart:end]

		if err := store.WithTx(ctx, func(tx eventstore.TxStore) error {
			for _, r := range batch {
				evt, buildErr := buildImportEvent(r, cfg.CorrelationID)
				if buildErr != nil {
					return buildErr
				}
				if _, appendErr := tx.Append(ctx, evt); appendErr != nil {
					return appendErr
				}
			}
			return nil
		}); err != nil {
			return BackfillResult{}, fmt.Errorf("append batch starting at %d: %w", batchStart, err)
		}

		processed += len(batch)
		if cfg.OnProgress != nil {
			cfg.OnProgress(processed, total)
		}
	}

	return BackfillResult{
		Processed:     processed,
		Skipped:       skipped,
		CorrelationID: cfg.CorrelationID,
		Duration:      time.Since(start),
	}, nil
}

func buildImportEvent(r resourceRow, correlationID string) (eventstore.Event, error) {
	payload, err := json.Marshal(eventstore.ResourceCreatedPayload{
		URL:          r.url,
		Host:         r.host,
		Title:        r.title,
		Summary:      r.summary,
		CategoryID:   r.categoryID,
		CategoryName: r.categoryName,
		UserOverride: r.userOverride,
		CreatedAt:    r.createdAt,
		UpdatedAt:    r.updatedAt,
	})
	if err != nil {
		return eventstore.Event{}, fmt.Errorf("marshal import payload for %s: %w", r.id, err)
	}
	corr := correlationID
	return eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   r.id,
		AggregateType: eventstore.AggregateTypeResource,
		EventType:     eventstore.EventTypeResourceImported,
		EventVersion:  1,
		Payload:       json.RawMessage(payload),
		CorrelationID: &corr,
	}, nil
}

// resourceIDsWithEvents returns the set of aggregate_ids in the events table
// for aggregate_type = 'resource'. One query regardless of table size.
func resourceIDsWithEvents(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT DISTINCT aggregate_id FROM events WHERE aggregate_type = 'resource'`)
	if err != nil {
		return nil, fmt.Errorf("distinct resource events: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}

// allResourceRows reads every row from resources JOIN categories.
// Compatible with both SQLite (TEXT timestamps, INTEGER bool) and
// Postgres (TIMESTAMPTZ, BOOLEAN) via interface{} scanning.
func allResourceRows(ctx context.Context, db *sql.DB) ([]resourceRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.id, r.url, r.host, r.title, r.summary, r.category_id, c.name,
		       r.user_override, r.created_at, r.updated_at
		FROM resources r
		JOIN categories c ON c.id = r.category_id
		ORDER BY r.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query all resources: %w", err)
	}
	defer rows.Close()

	var result []resourceRow
	for rows.Next() {
		var r resourceRow
		var userOverrideRaw, createdAtRaw, updatedAtRaw interface{}
		if err := rows.Scan(
			&r.id, &r.url, &r.host, &r.title, &r.summary,
			&r.categoryID, &r.categoryName,
			&userOverrideRaw, &createdAtRaw, &updatedAtRaw,
		); err != nil {
			return nil, fmt.Errorf("scan resource row: %w", err)
		}
		r.userOverride = scanBool(userOverrideRaw)
		r.createdAt = scanTime(createdAtRaw)
		r.updatedAt = scanTime(updatedAtRaw)
		result = append(result, r)
	}
	return result, rows.Err()
}

// scanBool converts a driver value to bool (SQLite int64 or Postgres bool).
func scanBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case int64:
		return b != 0
	case int32:
		return b != 0
	}
	return false
}

// scanTime converts a driver value to time.Time (SQLite string or Postgres time.Time).
func scanTime(v interface{}) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t.UTC()
	case string:
		parsed, _ := time.Parse(time.RFC3339, t)
		return parsed
	case []byte:
		parsed, _ := time.Parse(time.RFC3339, string(t))
		return parsed
	}
	return time.Time{}
}
