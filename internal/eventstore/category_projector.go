package eventstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RegisterCategoryProjectors adds sync projectors for the Category aggregate.
// dialect is "sqlite" or "postgres".
func RegisterCategoryProjectors(registry *ProjectorRegistry, dialect string) {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		registry.RegisterSync(EventTypeCategoryCreated, pgCategoryCreate)
		registry.RegisterSync(EventTypeCategoryUpdated, pgCategoryUpdate)
		registry.RegisterSync(EventTypeCategoryDeleted, pgCategoryDelete)
	default: // sqlite
		registry.RegisterSync(EventTypeCategoryCreated, sqliteCategoryCreate)
		registry.RegisterSync(EventTypeCategoryUpdated, sqliteCategoryUpdate)
		registry.RegisterSync(EventTypeCategoryDeleted, sqliteCategoryDelete)
	}
}

// ── SQLite projectors ────────────────────────────────────────────────────────

func sqliteCategoryCreate(ctx context.Context, event Event, conn TxConn) error {
	var p CategoryCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode CategoryCreated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO categories (id, name, description, source, accept_count, override_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, 0, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, event.AggregateID, p.Name, p.Description, p.Source, rfc3339(p.CreatedAt), rfc3339(p.UpdatedAt))
	if err != nil {
		return fmt.Errorf("project CategoryCreated (sqlite): %w", err)
	}
	return nil
}

func sqliteCategoryUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p CategoryUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode CategoryUpdated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE categories SET name=?, description=?, updated_at=? WHERE id=?
	`, p.Name, p.Description, rfc3339(p.UpdatedAt), event.AggregateID)
	if err != nil {
		return fmt.Errorf("project CategoryUpdated (sqlite): %w", err)
	}
	return nil
}

func sqliteCategoryDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM categories WHERE id=?`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project CategoryDeleted (sqlite): %w", err)
	}
	return nil
}

// ── Postgres projectors ──────────────────────────────────────────────────────

func pgCategoryCreate(ctx context.Context, event Event, conn TxConn) error {
	var p CategoryCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode CategoryCreated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO categories (id, name, description, source, accept_count, override_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, 0, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`, event.AggregateID, p.Name, p.Description, p.Source, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("project CategoryCreated (postgres): %w", err)
	}
	return nil
}

func pgCategoryUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p CategoryUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode CategoryUpdated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE categories SET name=$1, description=$2, updated_at=$3 WHERE id=$4
	`, p.Name, p.Description, p.UpdatedAt, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project CategoryUpdated (postgres): %w", err)
	}
	return nil
}

func pgCategoryDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM categories WHERE id=$1`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project CategoryDeleted (postgres): %w", err)
	}
	return nil
}
