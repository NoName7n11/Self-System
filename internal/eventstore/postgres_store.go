package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// postgresConn is the minimal interface satisfied by *sql.DB and *sql.Tx.
type postgresConn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// PostgresStore implements Store against a *sql.DB.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// PostgresTxStore is a Store scoped to an open *sql.Tx. Implements TxStore.
type PostgresTxStore struct {
	tx *sql.Tx
}

func (s *PostgresTxStore) Commit() error   { return s.tx.Commit() }
func (s *PostgresTxStore) Rollback() error { return s.tx.Rollback() }
func (s *PostgresTxStore) Conn() TxConn    { return s.tx }

func (s *PostgresTxStore) Append(ctx context.Context, event Event) (AppendResult, error) {
	return pgAppendEvent(ctx, s.tx, event)
}
func (s *PostgresTxStore) ReadByAggregate(ctx context.Context, id string, after, limit int) ([]Event, error) {
	return pgReadByAggregate(ctx, s.tx, id, after, limit)
}
func (s *PostgresTxStore) ReadBySequence(ctx context.Context, after int64, limit int) ([]Event, error) {
	return pgReadBySequence(ctx, s.tx, after, limit)
}
func (s *PostgresTxStore) Snapshot(ctx context.Context, snap Snapshot) error {
	return pgSaveSnapshot(ctx, s.tx, snap)
}
func (s *PostgresTxStore) Redact(ctx context.Context, eventID string) error {
	return pgRedactEvent(ctx, s.tx, eventID)
}
func (s *PostgresTxStore) LatestSequence(ctx context.Context) (int64, error) {
	return pgLatestSequence(ctx, s.tx)
}
func (s *PostgresTxStore) WithTx(ctx context.Context, fn func(tx TxStore) error) error {
	// Already inside a transaction — run fn directly without nesting.
	return fn(s)
}

// WithTx opens a *sql.Tx, scopes a PostgresTxStore to it, and calls fn.
func (s *PostgresStore) WithTx(ctx context.Context, fn func(tx TxStore) error) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("postgres eventstore is nil")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	txStore := &PostgresTxStore{tx: tx}
	if err := fn(txStore); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) Append(ctx context.Context, event Event) (AppendResult, error) {
	if s == nil || s.db == nil {
		return AppendResult{}, fmt.Errorf("postgres eventstore is nil")
	}
	return pgAppendEvent(ctx, s.db, event)
}

func (s *PostgresStore) ReadByAggregate(ctx context.Context, aggregateID string, afterVersion, limit int) ([]Event, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("postgres eventstore is nil")
	}
	return pgReadByAggregate(ctx, s.db, aggregateID, afterVersion, limit)
}

func (s *PostgresStore) ReadBySequence(ctx context.Context, afterSequence int64, limit int) ([]Event, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("postgres eventstore is nil")
	}
	return pgReadBySequence(ctx, s.db, afterSequence, limit)
}

func (s *PostgresStore) Snapshot(ctx context.Context, snap Snapshot) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("postgres eventstore is nil")
	}
	return pgSaveSnapshot(ctx, s.db, snap)
}

func (s *PostgresStore) Redact(ctx context.Context, eventID string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("postgres eventstore is nil")
	}
	return pgRedactEvent(ctx, s.db, eventID)
}

func (s *PostgresStore) LatestSequence(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("postgres eventstore is nil")
	}
	return pgLatestSequence(ctx, s.db)
}

// ── shared helpers ──────────────────────────────────────────────────────────

func pgLatestSequence(ctx context.Context, conn postgresConn) (int64, error) {
	var seq int64
	err := conn.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) FROM events`).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("latest sequence: %w", err)
	}
	return seq, nil
}

func pgAppendEvent(ctx context.Context, conn postgresConn, event Event) (AppendResult, error) {
	normalized, err := normalizeEvent(event)
	if err != nil {
		return AppendResult{}, err
	}

	const query = `
		INSERT INTO events (
			event_id, aggregate_id, aggregate_type, event_type, event_version,
			payload, payload_schema_version, occurred_at, recorded_at,
			device_id, actor_id, redacted, correlation_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (event_id) DO NOTHING
		RETURNING sequence
	`

	var sequence int64
	err = conn.QueryRowContext(
		ctx, query,
		normalized.EventID,
		normalized.AggregateID,
		normalized.AggregateType,
		normalized.EventType,
		normalized.EventVersion,
		normalized.Payload,
		normalized.PayloadSchemaVersion,
		normalized.OccurredAt,
		normalized.RecordedAt,
		optionalString(normalized.DeviceID),
		optionalString(normalized.ActorID),
		normalized.Redacted,
		optionalString(normalized.CorrelationID),
	).Scan(&sequence)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			seq, lookupErr := pgLookupSequence(ctx, conn, normalized.EventID)
			if lookupErr != nil {
				return AppendResult{}, lookupErr
			}
			return AppendResult{Sequence: seq, Applied: false}, nil
		}
		if isPostgresConcurrencyConflict(err) {
			return AppendResult{}, ErrConcurrencyConflict
		}
		return AppendResult{}, fmt.Errorf("append event: %w", err)
	}

	return AppendResult{Sequence: sequence, Applied: true}, nil
}

func pgReadByAggregate(ctx context.Context, conn postgresConn, aggregateID string, afterVersion, limit int) ([]Event, error) {
	aggregateID = strings.TrimSpace(aggregateID)
	if aggregateID == "" {
		return []Event{}, nil
	}

	query := `
		SELECT sequence, event_id, aggregate_id, aggregate_type, event_type, event_version,
			payload, payload_schema_version, occurred_at, recorded_at, device_id, actor_id,
			redacted, correlation_id
		FROM events
		WHERE aggregate_id = $1 AND event_version > $2
		ORDER BY event_version ASC
	`
	args := []any{aggregateID, afterVersion}
	if limit > 0 {
		query += " LIMIT $3"
		args = append(args, limit)
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read events by aggregate: %w", err)
	}
	defer rows.Close()
	return pgScanEvents(rows)
}

func pgReadBySequence(ctx context.Context, conn postgresConn, afterSequence int64, limit int) ([]Event, error) {
	query := `
		SELECT sequence, event_id, aggregate_id, aggregate_type, event_type, event_version,
			payload, payload_schema_version, occurred_at, recorded_at, device_id, actor_id,
			redacted, correlation_id
		FROM events
		WHERE sequence > $1
		ORDER BY sequence ASC
	`
	args := []any{afterSequence}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read events by sequence: %w", err)
	}
	defer rows.Close()
	return pgScanEvents(rows)
}

func pgSaveSnapshot(ctx context.Context, conn postgresConn, snapshot Snapshot) error {
	normalized, err := normalizeSnapshot(snapshot)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO projection_snapshots (
			aggregate_id, aggregate_type, snapshot_version, payload, created_at
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (aggregate_id, snapshot_version) DO UPDATE
		SET aggregate_type = EXCLUDED.aggregate_type,
			payload        = EXCLUDED.payload,
			created_at     = EXCLUDED.created_at
	`
	_, err = conn.ExecContext(
		ctx, query,
		normalized.AggregateID,
		normalized.AggregateType,
		normalized.SnapshotVersion,
		normalized.Payload,
		normalized.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

func pgRedactEvent(ctx context.Context, conn postgresConn, eventID string) error {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return fmt.Errorf("event_id is required")
	}

	payload := json.RawMessage(`{"redacted":true}`)
	result, err := conn.ExecContext(
		ctx,
		`UPDATE events SET payload = $1, redacted = TRUE WHERE event_id = $2`,
		payload, eventID,
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

func pgLookupSequence(ctx context.Context, conn postgresConn, eventID string) (int64, error) {
	var sequence int64
	err := conn.QueryRowContext(ctx, `SELECT sequence FROM events WHERE event_id = $1`, eventID).Scan(&sequence)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrEventNotFound
		}
		return 0, fmt.Errorf("lookup event sequence: %w", err)
	}
	return sequence, nil
}

func pgScanEvents(rows *sql.Rows) ([]Event, error) {
	items := make([]Event, 0)
	for rows.Next() {
		item, err := scanPostgresEvent(rows)
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

func scanPostgresEvent(row interface{ Scan(dest ...any) error }) (Event, error) {
	var item Event
	var payload []byte
	var deviceID, actorID, correlationID sql.NullString

	if err := row.Scan(
		&item.Sequence, &item.EventID, &item.AggregateID, &item.AggregateType,
		&item.EventType, &item.EventVersion, &payload, &item.PayloadSchemaVersion,
		&item.OccurredAt, &item.RecordedAt, &deviceID, &actorID,
		&item.Redacted, &correlationID,
	); err != nil {
		return Event{}, fmt.Errorf("scan event: %w", err)
	}

	item.Payload = json.RawMessage(payload)
	item.DeviceID = nullToPtr(deviceID)
	item.ActorID = nullToPtr(actorID)
	item.CorrelationID = nullToPtr(correlationID)
	return item, nil
}

func isPostgresConcurrencyConflict(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	return pqErr.Code == "23505" && pqErr.Constraint == "events_aggregate_version_unique"
}
