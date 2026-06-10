package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// sqliteConn is the minimal interface satisfied by *sql.DB and *sql.Tx.
type sqliteConn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sqliteTxConn extends sqliteConn with transaction control.
type sqliteTxConn interface {
	sqliteConn
	Commit() error
	Rollback() error
}

// SQLiteStore implements Store against a *sql.DB.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// SQLiteTxStore is a Store scoped to an open *sql.Tx. Implements TxStore.
type SQLiteTxStore struct {
	tx *sql.Tx
}

func (s *SQLiteTxStore) Commit() error   { return s.tx.Commit() }
func (s *SQLiteTxStore) Rollback() error { return s.tx.Rollback() }
func (s *SQLiteTxStore) Conn() TxConn   { return s.tx }

func (s *SQLiteTxStore) Append(ctx context.Context, event Event) (AppendResult, error) {
	return appendEvent(ctx, s.tx, event)
}
func (s *SQLiteTxStore) ReadByAggregate(ctx context.Context, id string, after, limit int) ([]Event, error) {
	return readByAggregate(ctx, s.tx, id, after, limit)
}
func (s *SQLiteTxStore) ReadBySequence(ctx context.Context, after int64, limit int) ([]Event, error) {
	return readBySequence(ctx, s.tx, after, limit)
}
func (s *SQLiteTxStore) Snapshot(ctx context.Context, snap Snapshot) error {
	return saveSnapshot(ctx, s.tx, snap)
}
func (s *SQLiteTxStore) Redact(ctx context.Context, eventID string) error {
	return redactEvent(ctx, s.tx, eventID)
}
func (s *SQLiteTxStore) LatestSequence(ctx context.Context) (int64, error) {
	return latestSequence(ctx, s.tx)
}
func (s *SQLiteTxStore) WithTx(ctx context.Context, fn func(tx TxStore) error) error {
	// Already inside a transaction — run fn directly without nesting.
	return fn(s)
}

// WithTx opens a *sql.Tx, scopes a SQLiteTxStore to it, and calls fn.
func (s *SQLiteStore) WithTx(ctx context.Context, fn func(tx TxStore) error) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite eventstore is nil")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txStore := &SQLiteTxStore{tx: tx}
	if err := fn(txStore); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) Append(ctx context.Context, event Event) (AppendResult, error) {
	if s == nil || s.db == nil {
		return AppendResult{}, fmt.Errorf("sqlite eventstore is nil")
	}
	return appendEvent(ctx, s.db, event)
}

func (s *SQLiteStore) ReadByAggregate(ctx context.Context, aggregateID string, afterVersion, limit int) ([]Event, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite eventstore is nil")
	}
	return readByAggregate(ctx, s.db, aggregateID, afterVersion, limit)
}

func (s *SQLiteStore) ReadBySequence(ctx context.Context, afterSequence int64, limit int) ([]Event, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite eventstore is nil")
	}
	return readBySequence(ctx, s.db, afterSequence, limit)
}

func (s *SQLiteStore) Snapshot(ctx context.Context, snap Snapshot) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite eventstore is nil")
	}
	return saveSnapshot(ctx, s.db, snap)
}

func (s *SQLiteStore) Redact(ctx context.Context, eventID string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite eventstore is nil")
	}
	return redactEvent(ctx, s.db, eventID)
}

func (s *SQLiteStore) LatestSequence(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("sqlite eventstore is nil")
	}
	return latestSequence(ctx, s.db)
}

// ── shared helpers (work against *sql.DB and *sql.Tx) ──────────────────────

func appendEvent(ctx context.Context, conn sqliteConn, event Event) (AppendResult, error) {
	normalized, err := normalizeEvent(event)
	if err != nil {
		return AppendResult{}, err
	}

	const query = `
		INSERT INTO events (
			event_id, aggregate_id, aggregate_type, event_type, event_version,
			payload, payload_schema_version, occurred_at, recorded_at,
			device_id, actor_id, redacted, correlation_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(event_id) DO NOTHING
		RETURNING sequence
	`

	var sequence int64
	err = conn.QueryRowContext(
		ctx,
		query,
		normalized.EventID,
		normalized.AggregateID,
		normalized.AggregateType,
		normalized.EventType,
		normalized.EventVersion,
		string(normalized.Payload),
		normalized.PayloadSchemaVersion,
		formatTime(normalized.OccurredAt),
		formatTime(normalized.RecordedAt),
		optionalString(normalized.DeviceID),
		optionalString(normalized.ActorID),
		boolToInt(normalized.Redacted),
		optionalString(normalized.CorrelationID),
	).Scan(&sequence)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Idempotent: event_id already exists; return its sequence.
			seq, lookupErr := lookupSequence(ctx, conn, normalized.EventID)
			if lookupErr != nil {
				return AppendResult{}, lookupErr
			}
			return AppendResult{Sequence: seq, Applied: false}, nil
		}
		if isSQLiteConcurrencyConflict(err) {
			return AppendResult{}, ErrConcurrencyConflict
		}
		return AppendResult{}, fmt.Errorf("append event: %w", err)
	}

	return AppendResult{Sequence: sequence, Applied: true}, nil
}

func readByAggregate(ctx context.Context, conn sqliteConn, aggregateID string, afterVersion, limit int) ([]Event, error) {
	aggregateID = strings.TrimSpace(aggregateID)
	if aggregateID == "" {
		return []Event{}, nil
	}

	query := `
		SELECT sequence, event_id, aggregate_id, aggregate_type, event_type, event_version,
			payload, payload_schema_version, occurred_at, recorded_at, device_id, actor_id,
			redacted, correlation_id
		FROM events
		WHERE aggregate_id = ? AND event_version > ?
		ORDER BY event_version ASC
	`
	args := []any{aggregateID, afterVersion}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read events by aggregate: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func readBySequence(ctx context.Context, conn sqliteConn, afterSequence int64, limit int) ([]Event, error) {
	query := `
		SELECT sequence, event_id, aggregate_id, aggregate_type, event_type, event_version,
			payload, payload_schema_version, occurred_at, recorded_at, device_id, actor_id,
			redacted, correlation_id
		FROM events
		WHERE sequence > ?
		ORDER BY sequence ASC
	`
	args := []any{afterSequence}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read events by sequence: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func saveSnapshot(ctx context.Context, conn sqliteConn, snapshot Snapshot) error {
	normalized, err := normalizeSnapshot(snapshot)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO projection_snapshots (
			aggregate_id, aggregate_type, snapshot_version, payload, created_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(aggregate_id, snapshot_version) DO UPDATE
		SET aggregate_type = excluded.aggregate_type,
			payload        = excluded.payload,
			created_at     = excluded.created_at
	`
	_, err = conn.ExecContext(
		ctx, query,
		normalized.AggregateID,
		normalized.AggregateType,
		normalized.SnapshotVersion,
		string(normalized.Payload),
		formatTime(normalized.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

func redactEvent(ctx context.Context, conn sqliteConn, eventID string) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return fmt.Errorf("event_id is required")
	}

	result, err := conn.ExecContext(
		ctx,
		`UPDATE events SET payload = ?, redacted = 1 WHERE event_id = ?`,
		`{"redacted":true}`,
		eventID,
	)
	if err != nil {
		return fmt.Errorf("redact event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("redact event rows: %w", err)
	}
	if affected == 0 {
		return ErrEventNotFound
	}
	return nil
}

func lookupSequence(ctx context.Context, conn sqliteConn, eventID string) (int64, error) {
	var sequence int64
	err := conn.QueryRowContext(ctx, `SELECT sequence FROM events WHERE event_id = ?`, eventID).Scan(&sequence)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrEventNotFound
		}
		return 0, fmt.Errorf("lookup event sequence: %w", err)
	}
	return sequence, nil
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	items := make([]Event, 0)
	for rows.Next() {
		item, err := scanSQLiteEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return items, nil
}

func scanSQLiteEvent(row interface{ Scan(dest ...any) error }) (Event, error) {
	var item Event
	var payload string
	var occurredAt, recordedAt string
	var deviceID, actorID, correlationID sql.NullString
	var redactedInt int

	if err := row.Scan(
		&item.Sequence, &item.EventID, &item.AggregateID, &item.AggregateType,
		&item.EventType, &item.EventVersion, &payload, &item.PayloadSchemaVersion,
		&occurredAt, &recordedAt, &deviceID, &actorID, &redactedInt, &correlationID,
	); err != nil {
		return Event{}, fmt.Errorf("scan event: %w", err)
	}

	item.Payload = json.RawMessage(payload)
	item.OccurredAt = parseTime(occurredAt)
	item.RecordedAt = parseTime(recordedAt)
	item.DeviceID = nullToPtr(deviceID)
	item.ActorID = nullToPtr(actorID)
	item.CorrelationID = nullToPtr(correlationID)
	item.Redacted = redactedInt == 1
	return item, nil
}

// ── time helpers ────────────────────────────────────────────────────────────

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	// Fallback: SQLite datetime() without timezone.
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

// ── misc helpers ─────────────────────────────────────────────────────────────

func optionalString(v *string) any {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullToPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(v.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func latestSequence(ctx context.Context, conn sqliteConn) (int64, error) {
	var seq int64
	err := conn.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) FROM events`).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("latest sequence: %w", err)
	}
	return seq, nil
}

func isSQLiteConcurrencyConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") &&
		strings.Contains(msg, "events.aggregate_id") &&
		strings.Contains(msg, "events.event_version")
}
