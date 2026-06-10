package sync

import (
	"context"
	"time"
)

type ReplayQueueSnapshot struct {
	PendingCount     int       `json:"pending_count"`
	OldestEnqueuedAt time.Time `json:"oldest_enqueued_at"`
}

// OfflineReplayStore persists queued mutations and conflict history.
type OfflineReplayStore interface {
	Enqueue(ctx context.Context, mutation ReplayMutation) (ReplayMutation, error)
	ListPending(ctx context.Context, limit int) ([]ReplayMutation, error)
	QueueSnapshot(ctx context.Context) (ReplayQueueSnapshot, error)
	MarkApplied(ctx context.Context, operationIDs []string) error
	RecordConflict(ctx context.Context, conflict ConflictRecord) error
	ListConflicts(ctx context.Context, entityID string, limit int) ([]ConflictRecord, error)
}
