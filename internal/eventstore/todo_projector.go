package eventstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RegisterTodoProjectors adds sync projectors for the Todo aggregate.
func RegisterTodoProjectors(registry *ProjectorRegistry, dialect string) {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		registry.RegisterSync(EventTypeTodoCreated, pgTodoCreate)
		registry.RegisterSync(EventTypeTodoUpdated, pgTodoUpdate)
		registry.RegisterSync(EventTypeTodoDeleted, pgTodoDelete)
	default: // sqlite
		registry.RegisterSync(EventTypeTodoCreated, sqliteTodoCreate)
		registry.RegisterSync(EventTypeTodoUpdated, sqliteTodoUpdate)
		registry.RegisterSync(EventTypeTodoDeleted, sqliteTodoDelete)
	}
}

// ── SQLite projectors ────────────────────────────────────────────────────────

func sqliteTodoCreate(ctx context.Context, event Event, conn TxConn) error {
	var p TodoCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode TodoCreated payload: %w", err)
	}
	var dueAt interface{}
	if p.DueAt != nil {
		dueAt = rfc3339(*p.DueAt)
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO todos (id, title, details, status, due_at, resource_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, event.AggregateID, p.Title, p.Details, p.Status, dueAt, optStr(p.ResourceID), rfc3339(p.CreatedAt), rfc3339(p.UpdatedAt))
	if err != nil {
		return fmt.Errorf("project TodoCreated (sqlite): %w", err)
	}
	return nil
}

func sqliteTodoUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p TodoUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode TodoUpdated payload: %w", err)
	}
	var dueAt interface{}
	if p.DueAt != nil {
		dueAt = rfc3339(*p.DueAt)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE todos SET title=?, details=?, status=?, due_at=?, resource_id=?, updated_at=? WHERE id=?
	`, p.Title, p.Details, p.Status, dueAt, optStr(p.ResourceID), rfc3339(p.UpdatedAt), event.AggregateID)
	if err != nil {
		return fmt.Errorf("project TodoUpdated (sqlite): %w", err)
	}
	return nil
}

func sqliteTodoDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM todos WHERE id=?`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project TodoDeleted (sqlite): %w", err)
	}
	return nil
}

// ── Postgres projectors ──────────────────────────────────────────────────────

func pgTodoCreate(ctx context.Context, event Event, conn TxConn) error {
	var p TodoCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode TodoCreated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO todos (id, title, details, status, due_at, resource_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, event.AggregateID, p.Title, p.Details, p.Status, p.DueAt, p.ResourceID, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("project TodoCreated (postgres): %w", err)
	}
	return nil
}

func pgTodoUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p TodoUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode TodoUpdated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE todos SET title=$1, details=$2, status=$3, due_at=$4, resource_id=$5, updated_at=$6 WHERE id=$7
	`, p.Title, p.Details, p.Status, p.DueAt, p.ResourceID, p.UpdatedAt, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project TodoUpdated (postgres): %w", err)
	}
	return nil
}

func pgTodoDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM todos WHERE id=$1`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project TodoDeleted (postgres): %w", err)
	}
	return nil
}

// optStr converts *string to nil interface when nil (for SQL NULL).
func optStr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}
