package sync

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReplaySummary struct {
	QueuedCount   int `json:"queued_count"`
	ReplayedCount int `json:"replayed_count"`
	EmittedCount  int `json:"emitted_count"`
	ConflictCount int `json:"conflict_count"`
}

type replayEntityBatch struct {
	firstSequence int64
	mutations     []ReplayMutation
	winner        ReplayMutation
	conflicts     []ConflictRecord
}

// OfflineReplayManager coordinates queue persistence, conflict resolution, and replay emission.
type OfflineReplayManager struct {
	store    OfflineReplayStore
	resolver ConflictResolver
	hub      *Hub
	applier  ReplayMutationApplier
}

func NewOfflineReplayManager(store OfflineReplayStore, resolver ConflictResolver, hub *Hub) *OfflineReplayManager {
	return NewOfflineReplayManagerWithApplier(store, resolver, hub, nil)
}

func NewOfflineReplayManagerWithApplier(store OfflineReplayStore, resolver ConflictResolver, hub *Hub, applier ReplayMutationApplier) *OfflineReplayManager {
	if store == nil {
		store = NewMemoryReplayStore()
	}
	if resolver == nil {
		resolver = NewLastWriteWinsResolver()
	}
	if hub == nil {
		hub = NewHub()
	}

	return &OfflineReplayManager{store: store, resolver: resolver, hub: hub, applier: applier}
}

func (m *OfflineReplayManager) Enqueue(ctx context.Context, mutation ReplayMutation) (ReplayMutation, error) {
	if err := ValidateIncomingEvent(mutation.EventType, mutation.Payload); err != nil {
		return ReplayMutation{}, err
	}

	entityID := strings.TrimSpace(mutation.EntityID)
	if entityID == "" {
		entityID = inferEntityID(mutation.Payload)
	}
	if entityID == "" {
		return ReplayMutation{}, fmt.Errorf("payload.entity_id is required for offline replay")
	}

	mutation.EventType = strings.TrimSpace(strings.ToLower(mutation.EventType))
	mutation.EntityID = entityID
	mutation.Payload = BuildEventPayload(mutation.Payload, entityID, EventSourceSyncReplay)
	if mutation.OccurredAt.IsZero() {
		mutation.OccurredAt = time.Now().UTC()
	}
	mutation.EnqueuedAt = time.Now().UTC()

	return m.store.Enqueue(ctx, mutation)
}

func (m *OfflineReplayManager) Replay(ctx context.Context, limit int) (ReplaySummary, error) {
	logger := syncLogger()

	pending, err := m.store.ListPending(ctx, limit)
	if err != nil {
		logger.Warn("sync replay list pending failed", "error", err)
		return ReplaySummary{}, err
	}

	summary := ReplaySummary{QueuedCount: len(pending)}
	if len(pending) == 0 {
		return summary, nil
	}

	batchesByEntity := make(map[string]*replayEntityBatch)
	orderedBatches := make([]*replayEntityBatch, 0)

	for _, mutation := range pending {
		entityID := strings.TrimSpace(firstNonEmpty(mutation.EntityID, inferEntityID(mutation.Payload)))
		if entityID == "" {
			return summary, fmt.Errorf("replay mutation %q is missing entity_id", mutation.OperationID)
		}
		mutation.EntityID = entityID

		batch, exists := batchesByEntity[entityID]
		if !exists {
			batch = &replayEntityBatch{
				firstSequence: mutation.Sequence,
				winner:        mutation,
				mutations:     []ReplayMutation{mutation},
			}
			batchesByEntity[entityID] = batch
			orderedBatches = append(orderedBatches, batch)
			continue
		}

		batch.mutations = append(batch.mutations, mutation)

		winner, conflict, resolveErr := m.resolver.Resolve(batch.winner, mutation)
		if resolveErr != nil {
			logger.Warn("sync replay conflict resolution failed", "entity_id", entityID, "event_type", mutation.EventType, "error", resolveErr)
			return summary, resolveErr
		}
		if conflict != nil {
			logger.Info(
				"sync replay conflict resolved",
				"entity_id", conflict.EntityID,
				"event_type", conflict.EventType,
				"winner_operation_id", conflict.WinnerOperationID,
				"existing_operation_id", conflict.ExistingOperationID,
				"incoming_operation_id", conflict.IncomingOperationID,
				"reason", conflict.Reason,
			)
			batch.conflicts = append(batch.conflicts, *conflict)
		}
		batch.winner = winner
	}

	sort.SliceStable(orderedBatches, func(i, j int) bool {
		return orderedBatches[i].firstSequence < orderedBatches[j].firstSequence
	})

	for _, batch := range orderedBatches {
		winner := batch.winner
		if m.applier != nil {
			if applyErr := m.applier.Apply(ctx, winner); applyErr != nil {
				logger.Warn("sync replay apply failed", "operation_id", winner.OperationID, "entity_id", winner.EntityID, "event_type", winner.EventType, "error", applyErr)
				return summary, fmt.Errorf("apply replay mutation %q: %w", winner.OperationID, applyErr)
			}
		}

		operationIDs := make([]string, 0, len(batch.mutations))
		for _, mutation := range batch.mutations {
			operationIDs = append(operationIDs, mutation.OperationID)
		}
		if markErr := m.store.MarkApplied(ctx, operationIDs); markErr != nil {
			logger.Warn("sync replay mark applied failed", "operation_ids", operationIDs, "error", markErr)
			return summary, fmt.Errorf("mark replay mutations applied: %w", markErr)
		}
		summary.ReplayedCount += len(batch.mutations)

		for _, conflict := range batch.conflicts {
			if recordErr := m.store.RecordConflict(ctx, conflict); recordErr != nil {
				logger.Warn("sync replay conflict record failed", "conflict_id", conflict.ID, "entity_id", conflict.EntityID, "error", recordErr)
				return summary, fmt.Errorf("record replay conflict: %w", recordErr)
			}
			summary.ConflictCount++
		}

		payload := BuildEventPayload(winner.Payload, winner.EntityID, EventSourceSyncReplay)
		m.hub.Publish(NewEvent(winner.EventType, payload))
		summary.EmittedCount++
	}

	logger.Info(
		"sync replay completed",
		"queued_count", summary.QueuedCount,
		"replayed_count", summary.ReplayedCount,
		"emitted_count", summary.EmittedCount,
		"conflict_count", summary.ConflictCount,
	)

	return summary, nil
}

func (m *OfflineReplayManager) QueueSnapshot(ctx context.Context) (ReplayQueueSnapshot, error) {
	if m == nil || m.store == nil {
		return ReplayQueueSnapshot{}, nil
	}

	return m.store.QueueSnapshot(ctx)
}

func (m *OfflineReplayManager) ListConflicts(ctx context.Context, entityID string, limit int) ([]ConflictRecord, error) {
	return m.store.ListConflicts(ctx, strings.TrimSpace(entityID), limit)
}

func inferEntityID(payload map[string]any) string {
	return ExtractEntityID(payload)
}
