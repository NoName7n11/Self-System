package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrConcurrencyConflict = errors.New("event concurrency conflict")
	ErrEventNotFound       = errors.New("event not found")
	ErrInvalidPayload      = errors.New("invalid event payload")
)

type Event struct {
	Sequence             int64
	EventID              string
	AggregateID          string
	AggregateType        string
	EventType            string
	EventVersion         int
	Payload              json.RawMessage
	PayloadSchemaVersion int
	OccurredAt           time.Time
	RecordedAt           time.Time
	DeviceID             *string
	ActorID              *string
	Redacted             bool
	CorrelationID        *string
}

type AppendResult struct {
	Sequence int64
	Applied  bool
}

type Snapshot struct {
	AggregateID     string
	AggregateType   string
	SnapshotVersion int
	Payload         json.RawMessage
	CreatedAt       time.Time
}

// TxConn is the minimal database interface satisfied by *sql.Tx.
// Sync projectors receive it to write to projection tables in the
// same transaction as the event append (P8 — projector classification).
type TxConn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// TxStore is a Store scoped to a live database transaction.
// Passed to synchronous projectors so they can write to the same
// transaction as the event append (P8 — projector classification).
type TxStore interface {
	Store
	// Commit finalises the transaction. Called by WithTx on success.
	Commit() error
	// Rollback discards the transaction. Called by WithTx on any error.
	Rollback() error
	// Conn returns the raw transaction connection for sync projectors
	// that need to write to non-event tables in the same transaction.
	Conn() TxConn
}

// Store is the core event store interface.
// For simple, non-projected writes use Append directly.
// For writes that must drive synchronous projectors in the same
// transaction, use WithTx.
type Store interface {
	// Append writes a single event. If the event_id already exists the
	// write is a no-op and AppendResult.Applied is false (P2 idempotency).
	// Returns ErrConcurrencyConflict when aggregate_id + event_version
	// collides with an existing row (P1 OCC).
	Append(ctx context.Context, event Event) (AppendResult, error)

	// ReadByAggregate returns events for aggregateID with
	// event_version > afterVersion, ordered by event_version ASC.
	// limit=0 means no limit.
	ReadByAggregate(ctx context.Context, aggregateID string, afterVersion, limit int) ([]Event, error)

	// ReadBySequence returns events with sequence > afterSequence,
	// ordered by sequence ASC. Used by the outbox worker (P4).
	// limit=0 means no limit.
	ReadBySequence(ctx context.Context, afterSequence int64, limit int) ([]Event, error)

	// Snapshot persists an aggregate snapshot. Retained for audit/redaction use only; NOT used for projection rebuild — see ADR 0018. Do not build rebuild logic on this.
	Snapshot(ctx context.Context, snapshot Snapshot) error

	// Redact replaces the payload of eventID with {"redacted":true} and
	// sets the redacted flag, preserving the event envelope (P6).
	Redact(ctx context.Context, eventID string) error

	// LatestSequence returns the highest sequence in the events table, or 0
	// if no events exist. Used by the events_health endpoint to compute
	// outbox lag (P4 — outbox pattern).
	LatestSequence(ctx context.Context) (int64, error)

	// WithTx opens a database transaction, creates a TxStore scoped to
	// it, and calls fn. Commits on nil return, rolls back otherwise.
	// Synchronous projectors (P8) receive the TxStore so they participate
	// in the same transaction as the append.
	WithTx(ctx context.Context, fn func(tx TxStore) error) error
}

func normalizeEvent(event Event) (Event, error) {
	event.EventID = strings.TrimSpace(event.EventID)
	event.AggregateID = strings.TrimSpace(event.AggregateID)
	event.AggregateType = strings.TrimSpace(event.AggregateType)
	event.EventType = strings.TrimSpace(event.EventType)

	if event.EventID == "" {
		return event, fmt.Errorf("event_id is required")
	}
	if _, err := uuid.Parse(event.EventID); err != nil {
		return event, fmt.Errorf("event_id must be a valid UUID: %w", err)
	}
	if event.AggregateID == "" {
		return event, fmt.Errorf("aggregate_id is required")
	}
	if event.AggregateType == "" {
		return event, fmt.Errorf("aggregate_type is required")
	}
	if event.EventType == "" {
		return event, fmt.Errorf("event_type is required")
	}
	if event.EventVersion <= 0 {
		return event, fmt.Errorf("event_version must be positive")
	}
	if event.PayloadSchemaVersion <= 0 {
		event.PayloadSchemaVersion = 1
	}
	if len(event.Payload) == 0 {
		return event, ErrInvalidPayload
	}
	if !json.Valid(event.Payload) {
		return event, ErrInvalidPayload
	}

	now := time.Now().UTC()
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	// RecordedAt is always the server's perspective; never trust the caller.
	event.RecordedAt = now

	return event, nil
}

func normalizeSnapshot(snapshot Snapshot) (Snapshot, error) {
	snapshot.AggregateID = strings.TrimSpace(snapshot.AggregateID)
	snapshot.AggregateType = strings.TrimSpace(snapshot.AggregateType)

	if snapshot.AggregateID == "" {
		return snapshot, fmt.Errorf("aggregate_id is required")
	}
	if snapshot.AggregateType == "" {
		return snapshot, fmt.Errorf("aggregate_type is required")
	}
	if snapshot.SnapshotVersion <= 0 {
		return snapshot, fmt.Errorf("snapshot_version must be positive")
	}
	if len(snapshot.Payload) == 0 {
		return snapshot, ErrInvalidPayload
	}
	if !json.Valid(snapshot.Payload) {
		return snapshot, ErrInvalidPayload
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now().UTC()
	}

	return snapshot, nil
}
