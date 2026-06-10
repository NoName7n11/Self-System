package eventstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RegisterResourceProjectors adds sync projectors for the Resource aggregate
// to registry. dialect must be "sqlite" or "postgres".
// Projectors run inside the event append transaction (P8) and keep the
// resources projection table aligned with the event log.
func RegisterResourceProjectors(registry *ProjectorRegistry, dialect string) {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		registry.RegisterSync(EventTypeResourceCreated, pgResourceCreate)
		registry.RegisterSync(EventTypeResourceImported, pgResourceCreate) // same projection as Created
		registry.RegisterSync(EventTypeResourceUpdated, pgResourceUpdate)
		registry.RegisterSync(EventTypeResourceDeleted, pgResourceDelete)
		registry.RegisterSync(EventTypeResourceCategoryAssigned, pgResourceCategory)
	default: // sqlite
		registry.RegisterSync(EventTypeResourceCreated, sqliteResourceCreate)
		registry.RegisterSync(EventTypeResourceImported, sqliteResourceCreate) // same projection as Created
		registry.RegisterSync(EventTypeResourceUpdated, sqliteResourceUpdate)
		registry.RegisterSync(EventTypeResourceDeleted, sqliteResourceDelete)
		registry.RegisterSync(EventTypeResourceCategoryAssigned, sqliteResourceCategory)
	}
}

// ── SQLite projectors ────────────────────────────────────────────────────────
// SQLite stores booleans as 0/1 and timestamps as RFC3339 strings.

func sqliteResourceCreate(ctx context.Context, event Event, conn TxConn) error {
	var p ResourceCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ResourceCreated payload: %w", err)
	}
	createdAt := rfc3339(p.CreatedAt)
	updatedAt := rfc3339(p.UpdatedAt)
	userOverride := projBoolToInt(p.UserOverride)
	extractedData := projExtractedData(p.ExtractedDataJSON)

	_, err := conn.ExecContext(ctx, `
		INSERT INTO resources (id, url, host, title, summary, category_id, user_override, extracted_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, event.AggregateID, p.URL, p.Host, p.Title, p.Summary, p.CategoryID, userOverride, extractedData, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("project ResourceCreated (sqlite): %w", err)
	}
	return nil
}

func sqliteResourceUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p ResourceUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ResourceUpdated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE resources
		SET url=?, host=?, title=?, summary=?, category_id=?, user_override=?, updated_at=?
		WHERE id=?
	`, p.URL, p.Host, p.Title, p.Summary, p.CategoryID, projBoolToInt(p.UserOverride), rfc3339(p.UpdatedAt), event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ResourceUpdated (sqlite): %w", err)
	}
	return nil
}

func sqliteResourceDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM resources WHERE id=?`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ResourceDeleted (sqlite): %w", err)
	}
	return nil
}

func sqliteResourceCategory(ctx context.Context, event Event, conn TxConn) error {
	var p ResourceCategoryAssignedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ResourceCategoryAssigned payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE resources SET category_id=?, user_override=?, updated_at=? WHERE id=?
	`, p.CategoryID, projBoolToInt(p.UserOverride), rfc3339(p.UpdatedAt), event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ResourceCategoryAssigned (sqlite): %w", err)
	}
	return nil
}

// ── Postgres projectors ──────────────────────────────────────────────────────
// Postgres stores booleans as BOOLEAN and timestamps as TIMESTAMPTZ.

func pgResourceCreate(ctx context.Context, event Event, conn TxConn) error {
	var p ResourceCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ResourceCreated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO resources (id, url, host, title, summary, category_id, user_override, extracted_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO NOTHING
	`, event.AggregateID, p.URL, p.Host, p.Title, p.Summary, p.CategoryID, p.UserOverride, projExtractedData(p.ExtractedDataJSON), p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("project ResourceCreated (postgres): %w", err)
	}
	return nil
}

func pgResourceUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p ResourceUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ResourceUpdated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE resources
		SET url=$1, host=$2, title=$3, summary=$4, category_id=$5, user_override=$6, updated_at=$7
		WHERE id=$8
	`, p.URL, p.Host, p.Title, p.Summary, p.CategoryID, p.UserOverride, p.UpdatedAt, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ResourceUpdated (postgres): %w", err)
	}
	return nil
}

func pgResourceDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM resources WHERE id=$1`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ResourceDeleted (postgres): %w", err)
	}
	return nil
}

func pgResourceCategory(ctx context.Context, event Event, conn TxConn) error {
	var p ResourceCategoryAssignedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ResourceCategoryAssigned payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE resources SET category_id=$1, user_override=$2, updated_at=$3 WHERE id=$4
	`, p.CategoryID, p.UserOverride, p.UpdatedAt, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ResourceCategoryAssigned (postgres): %w", err)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return t.UTC().Format(time.RFC3339)
}

func projBoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// projExtractedData returns the extracted_data JSON to write into the projection,
// defaulting to "{}" when the event carried no extracted_data (older events).
func projExtractedData(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	return raw
}
