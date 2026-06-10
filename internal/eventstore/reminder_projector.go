package eventstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RegisterReminderProjectors adds sync projectors for the Reminder aggregate.
func RegisterReminderProjectors(registry *ProjectorRegistry, dialect string) {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql":
		registry.RegisterSync(EventTypeReminderCreated, pgReminderCreate)
		registry.RegisterSync(EventTypeReminderUpdated, pgReminderUpdate)
		registry.RegisterSync(EventTypeReminderDeleted, pgReminderDelete)
	default: // sqlite
		registry.RegisterSync(EventTypeReminderCreated, sqliteReminderCreate)
		registry.RegisterSync(EventTypeReminderUpdated, sqliteReminderUpdate)
		registry.RegisterSync(EventTypeReminderDeleted, sqliteReminderDelete)
	}
}

// ── SQLite projectors ────────────────────────────────────────────────────────

func sqliteReminderCreate(ctx context.Context, event Event, conn TxConn) error {
	var p ReminderCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ReminderCreated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO reminders (id, title, message, remind_at, status, resource_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, event.AggregateID, p.Title, p.Message, rfc3339(p.RemindAt), p.Status, optStr(p.ResourceID), rfc3339(p.CreatedAt), rfc3339(p.UpdatedAt))
	if err != nil {
		return fmt.Errorf("project ReminderCreated (sqlite): %w", err)
	}
	return nil
}

func sqliteReminderUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p ReminderUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ReminderUpdated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE reminders SET title=?, message=?, remind_at=?, status=?, resource_id=?, updated_at=? WHERE id=?
	`, p.Title, p.Message, rfc3339(p.RemindAt), p.Status, optStr(p.ResourceID), rfc3339(p.UpdatedAt), event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ReminderUpdated (sqlite): %w", err)
	}
	return nil
}

func sqliteReminderDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM reminders WHERE id=?`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ReminderDeleted (sqlite): %w", err)
	}
	return nil
}

// ── Postgres projectors ──────────────────────────────────────────────────────

func pgReminderCreate(ctx context.Context, event Event, conn TxConn) error {
	var p ReminderCreatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ReminderCreated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		INSERT INTO reminders (id, title, message, remind_at, status, resource_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, event.AggregateID, p.Title, p.Message, p.RemindAt, p.Status, p.ResourceID, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("project ReminderCreated (postgres): %w", err)
	}
	return nil
}

func pgReminderUpdate(ctx context.Context, event Event, conn TxConn) error {
	var p ReminderUpdatedPayload
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		return fmt.Errorf("decode ReminderUpdated payload: %w", err)
	}
	_, err := conn.ExecContext(ctx, `
		UPDATE reminders SET title=$1, message=$2, remind_at=$3, status=$4, resource_id=$5, updated_at=$6 WHERE id=$7
	`, p.Title, p.Message, p.RemindAt, p.Status, p.ResourceID, p.UpdatedAt, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ReminderUpdated (postgres): %w", err)
	}
	return nil
}

func pgReminderDelete(ctx context.Context, event Event, conn TxConn) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM reminders WHERE id=$1`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("project ReminderDeleted (postgres): %w", err)
	}
	return nil
}
