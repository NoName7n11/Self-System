package sync

import (
	"fmt"
	"strings"
	"time"
)

// ReplayMutation represents a queued offline mutation awaiting replay.
type ReplayMutation struct {
	Sequence    int64          `json:"sequence"`
	OperationID string         `json:"operation_id"`
	EventType   string         `json:"event_type"`
	EntityID    string         `json:"entity_id"`
	Payload     map[string]any `json:"payload"`
	OccurredAt  time.Time      `json:"occurred_at"`
	EnqueuedAt  time.Time      `json:"enqueued_at"`
}

// ConflictRecord captures deterministic resolution details for concurrent writes.
type ConflictRecord struct {
	ID                  string    `json:"id"`
	EntityID            string    `json:"entity_id"`
	EventType           string    `json:"event_type"`
	ExistingOperationID string    `json:"existing_operation_id"`
	IncomingOperationID string    `json:"incoming_operation_id"`
	WinnerOperationID   string    `json:"winner_operation_id"`
	Reason              string    `json:"reason"`
	ResolvedAt          time.Time `json:"resolved_at"`
}

// ConflictResolver picks a winner when multiple queued mutations target one entity.
type ConflictResolver interface {
	Resolve(existing, incoming ReplayMutation) (winner ReplayMutation, conflict *ConflictRecord, err error)
}

// LastWriteWinsResolver applies timestamp-based last-write-wins with deterministic tie-breaks.
type LastWriteWinsResolver struct{}

func NewLastWriteWinsResolver() *LastWriteWinsResolver {
	return &LastWriteWinsResolver{}
}

func (r *LastWriteWinsResolver) Resolve(existing, incoming ReplayMutation) (ReplayMutation, *ConflictRecord, error) {
	if strings.TrimSpace(incoming.OperationID) == "" {
		return existing, nil, fmt.Errorf("incoming operation_id is required")
	}

	if strings.TrimSpace(existing.OperationID) == "" {
		return incoming, nil, nil
	}

	if strings.TrimSpace(existing.EntityID) == "" || strings.TrimSpace(incoming.EntityID) == "" {
		return incoming, nil, fmt.Errorf("entity_id is required")
	}

	if existing.EntityID != incoming.EntityID {
		// Different entities are not a conflict pair.
		return incoming, nil, nil
	}

	if existing.EventType != incoming.EventType {
		// Preserve deterministic behavior while still recording the mixed-type conflict.
		winner := pickWinner(existing, incoming)
		conflict := &ConflictRecord{
			EntityID:            incoming.EntityID,
			EventType:           incoming.EventType,
			ExistingOperationID: existing.OperationID,
			IncomingOperationID: incoming.OperationID,
			WinnerOperationID:   winner.OperationID,
			Reason:              "mixed_event_types_last_write_wins",
			ResolvedAt:          time.Now().UTC(),
		}
		return winner, conflict, nil
	}

	winner := pickWinner(existing, incoming)
	if winner.OperationID == existing.OperationID && existing.OperationID == incoming.OperationID {
		return winner, nil, nil
	}

	reason := "existing_newer_timestamp"
	if winner.OperationID == incoming.OperationID {
		reason = "incoming_newer_timestamp"
	}
	if existing.OccurredAt.Equal(incoming.OccurredAt) {
		reason = "timestamp_tie_operation_id"
	}

	conflict := &ConflictRecord{
		EntityID:            incoming.EntityID,
		EventType:           incoming.EventType,
		ExistingOperationID: existing.OperationID,
		IncomingOperationID: incoming.OperationID,
		WinnerOperationID:   winner.OperationID,
		Reason:              reason,
		ResolvedAt:          time.Now().UTC(),
	}

	return winner, conflict, nil
}

func pickWinner(existing, incoming ReplayMutation) ReplayMutation {
	existingAt := existing.OccurredAt.UTC()
	incomingAt := incoming.OccurredAt.UTC()

	switch {
	case incomingAt.After(existingAt):
		return incoming
	case incomingAt.Before(existingAt):
		return existing
	default:
		// Deterministic tie-break: lexicographically larger operation_id wins.
		if strings.Compare(incoming.OperationID, existing.OperationID) > 0 {
			return incoming
		}
		return existing
	}
}
