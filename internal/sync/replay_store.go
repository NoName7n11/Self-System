package sync

import "context"

// OfflineReplayStore persists queued mutations and conflict history.
type OfflineReplayStore interface {
	Enqueue(ctx context.Context, mutation ReplayMutation) (ReplayMutation, error)
	ListPending(ctx context.Context, limit int) ([]ReplayMutation, error)
	MarkApplied(ctx context.Context, operationIDs []string) error
	RecordConflict(ctx context.Context, conflict ConflictRecord) error
	ListConflicts(ctx context.Context, entityID string, limit int) ([]ConflictRecord, error)
}
