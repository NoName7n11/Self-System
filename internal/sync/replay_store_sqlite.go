package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const sqliteReplayStoreSchema = `
CREATE TABLE IF NOT EXISTS sync_offline_queue (
	sequence INTEGER PRIMARY KEY AUTOINCREMENT,
	operation_id TEXT NOT NULL UNIQUE,
	event_type TEXT NOT NULL,
	entity_id TEXT NOT NULL,
	payload_json TEXT NOT NULL,
	occurred_at TEXT NOT NULL,
	enqueued_at TEXT NOT NULL,
	applied_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_offline_queue_applied ON sync_offline_queue(applied_at, sequence);

CREATE TABLE IF NOT EXISTS sync_conflicts (
	id TEXT PRIMARY KEY,
	entity_id TEXT NOT NULL,
	event_type TEXT NOT NULL,
	existing_operation_id TEXT NOT NULL,
	incoming_operation_id TEXT NOT NULL,
	winner_operation_id TEXT NOT NULL,
	reason TEXT NOT NULL,
	resolved_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sync_conflicts_entity_resolved ON sync_conflicts(entity_id, resolved_at DESC);
`

type SQLiteReplayStore struct {
	db *sql.DB
}

func NewSQLiteReplayStore(db *sql.DB) (*SQLiteReplayStore, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlite replay store db is required")
	}

	if _, err := db.Exec(sqliteReplayStoreSchema); err != nil {
		return nil, fmt.Errorf("ensure sqlite replay schema: %w", err)
	}

	return &SQLiteReplayStore{db: db}, nil
}

func (s *SQLiteReplayStore) Enqueue(ctx context.Context, mutation ReplayMutation) (ReplayMutation, error) {
	if strings.TrimSpace(mutation.OperationID) == "" {
		mutation.OperationID = uuid.NewString()
	}
	mutation.OperationID = strings.TrimSpace(mutation.OperationID)
	mutation.EventType = strings.TrimSpace(strings.ToLower(mutation.EventType))
	mutation.EntityID = strings.TrimSpace(mutation.EntityID)
	if mutation.OccurredAt.IsZero() {
		mutation.OccurredAt = time.Now().UTC()
	}
	if mutation.EnqueuedAt.IsZero() {
		mutation.EnqueuedAt = time.Now().UTC()
	}

	payload := copyPayload(mutation.Payload)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return ReplayMutation{}, fmt.Errorf("marshal replay payload: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_offline_queue (operation_id, event_type, entity_id, payload_json, occurred_at, enqueued_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(operation_id) DO NOTHING
	`, mutation.OperationID, mutation.EventType, mutation.EntityID, string(payloadBytes), mutation.OccurredAt.UTC().Format(time.RFC3339Nano), mutation.EnqueuedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return ReplayMutation{}, fmt.Errorf("enqueue replay mutation: %w", err)
	}

	stored, err := s.getByOperationID(ctx, mutation.OperationID)
	if err != nil {
		return ReplayMutation{}, err
	}
	stored.Payload = copyPayload(stored.Payload)
	return stored, nil
}

func (s *SQLiteReplayStore) ListPending(ctx context.Context, limit int) ([]ReplayMutation, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT sequence, operation_id, event_type, entity_id, payload_json, occurred_at, enqueued_at
		FROM sync_offline_queue
		WHERE applied_at IS NULL
		ORDER BY sequence ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending replay mutations: %w", err)
	}
	defer rows.Close()

	items := make([]ReplayMutation, 0)
	for rows.Next() {
		mutation, scanErr := scanSQLiteReplayMutation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, mutation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending replay mutations: %w", err)
	}

	return items, nil
}

func (s *SQLiteReplayStore) MarkApplied(ctx context.Context, operationIDs []string) error {
	if len(operationIDs) == 0 {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, id := range operationIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE sync_offline_queue
			SET applied_at = ?
			WHERE operation_id = ? AND applied_at IS NULL
		`, now, trimmed); err != nil {
			return fmt.Errorf("mark replay mutation applied: %w", err)
		}
	}

	return nil
}

func (s *SQLiteReplayStore) RecordConflict(ctx context.Context, conflict ConflictRecord) error {
	if strings.TrimSpace(conflict.ID) == "" {
		conflict.ID = uuid.NewString()
	}
	if conflict.ResolvedAt.IsZero() {
		conflict.ResolvedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_conflicts (id, entity_id, event_type, existing_operation_id, incoming_operation_id, winner_operation_id, reason, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, strings.TrimSpace(conflict.ID), strings.TrimSpace(conflict.EntityID), strings.TrimSpace(strings.ToLower(conflict.EventType)), strings.TrimSpace(conflict.ExistingOperationID), strings.TrimSpace(conflict.IncomingOperationID), strings.TrimSpace(conflict.WinnerOperationID), strings.TrimSpace(conflict.Reason), conflict.ResolvedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record sync conflict: %w", err)
	}
	return nil
}

func (s *SQLiteReplayStore) ListConflicts(ctx context.Context, entityID string, limit int) ([]ConflictRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	entityID = strings.TrimSpace(entityID)
	items := make([]ConflictRecord, 0)

	if entityID == "" {
		rows, err := s.db.QueryContext(ctx, `
			SELECT id, entity_id, event_type, existing_operation_id, incoming_operation_id, winner_operation_id, reason, resolved_at
			FROM sync_conflicts
			ORDER BY resolved_at DESC
			LIMIT ?
		`, limit)
		if err != nil {
			return nil, fmt.Errorf("list sync conflicts: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			item, scanErr := scanSQLiteConflict(rows)
			if scanErr != nil {
				return nil, scanErr
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate sync conflicts: %w", err)
		}
		return items, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, entity_id, event_type, existing_operation_id, incoming_operation_id, winner_operation_id, reason, resolved_at
		FROM sync_conflicts
		WHERE entity_id = ?
		ORDER BY resolved_at DESC
		LIMIT ?
	`, entityID, limit)
	if err != nil {
		return nil, fmt.Errorf("list sync conflicts by entity: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, scanErr := scanSQLiteConflict(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sync conflicts by entity: %w", err)
	}

	return items, nil
}

func scanSQLiteReplayMutation(row interface{ Scan(dest ...any) error }) (ReplayMutation, error) {
	var (
		item       ReplayMutation
		payloadRaw string
		occurredAt string
		enqueuedAt string
	)

	if err := row.Scan(&item.Sequence, &item.OperationID, &item.EventType, &item.EntityID, &payloadRaw, &occurredAt, &enqueuedAt); err != nil {
		return ReplayMutation{}, fmt.Errorf("scan replay mutation: %w", err)
	}

	item.EventType = strings.TrimSpace(strings.ToLower(item.EventType))
	item.EntityID = strings.TrimSpace(item.EntityID)

	if err := json.Unmarshal([]byte(payloadRaw), &item.Payload); err != nil {
		return ReplayMutation{}, fmt.Errorf("unmarshal replay payload: %w", err)
	}

	parsedOccurredAt, err := time.Parse(time.RFC3339Nano, occurredAt)
	if err != nil {
		return ReplayMutation{}, fmt.Errorf("parse replay occurred_at: %w", err)
	}
	parsedEnqueuedAt, err := time.Parse(time.RFC3339Nano, enqueuedAt)
	if err != nil {
		return ReplayMutation{}, fmt.Errorf("parse replay enqueued_at: %w", err)
	}

	item.OccurredAt = parsedOccurredAt.UTC()
	item.EnqueuedAt = parsedEnqueuedAt.UTC()
	return item, nil
}

func (s *SQLiteReplayStore) getByOperationID(ctx context.Context, operationID string) (ReplayMutation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT sequence, operation_id, event_type, entity_id, payload_json, occurred_at, enqueued_at
		FROM sync_offline_queue
		WHERE operation_id = ?
	`, strings.TrimSpace(operationID))

	item, err := scanSQLiteReplayMutation(row)
	if err != nil {
		return ReplayMutation{}, fmt.Errorf("load replay mutation by operation_id: %w", err)
	}
	return item, nil
}

func scanSQLiteConflict(row interface{ Scan(dest ...any) error }) (ConflictRecord, error) {
	var (
		item       ConflictRecord
		resolvedAt string
	)

	if err := row.Scan(&item.ID, &item.EntityID, &item.EventType, &item.ExistingOperationID, &item.IncomingOperationID, &item.WinnerOperationID, &item.Reason, &resolvedAt); err != nil {
		return ConflictRecord{}, fmt.Errorf("scan sync conflict: %w", err)
	}

	parsedResolvedAt, err := time.Parse(time.RFC3339Nano, resolvedAt)
	if err != nil {
		return ConflictRecord{}, fmt.Errorf("parse conflict resolved_at: %w", err)
	}

	item.EventType = strings.TrimSpace(strings.ToLower(item.EventType))
	item.EntityID = strings.TrimSpace(item.EntityID)
	item.Reason = strings.TrimSpace(item.Reason)
	item.ResolvedAt = parsedResolvedAt.UTC()
	return item, nil
}

func copyPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}
