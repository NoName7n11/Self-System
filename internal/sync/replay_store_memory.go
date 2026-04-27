package sync

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryReplayStore is a deterministic in-memory store used by tests and fallback wiring.
type MemoryReplayStore struct {
	mu        sync.Mutex
	sequence  int64
	pending   []ReplayMutation
	seen      map[string]ReplayMutation
	conflicts []ConflictRecord
}

func NewMemoryReplayStore() *MemoryReplayStore {
	return &MemoryReplayStore{seen: make(map[string]ReplayMutation)}
}

func (s *MemoryReplayStore) Enqueue(_ context.Context, mutation ReplayMutation) (ReplayMutation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(mutation.OperationID) == "" {
		mutation.OperationID = uuid.NewString()
	}
	mutation.OperationID = strings.TrimSpace(mutation.OperationID)
	if existing, exists := s.seen[mutation.OperationID]; exists {
		clone := existing
		clone.Payload = copyPayload(clone.Payload)
		return clone, nil
	}

	s.sequence++
	if mutation.OccurredAt.IsZero() {
		mutation.OccurredAt = time.Now().UTC()
	}
	if mutation.EnqueuedAt.IsZero() {
		mutation.EnqueuedAt = time.Now().UTC()
	}
	mutation.Sequence = s.sequence
	mutation.EventType = strings.TrimSpace(strings.ToLower(mutation.EventType))
	mutation.EntityID = strings.TrimSpace(mutation.EntityID)
	mutation.Payload = copyPayload(mutation.Payload)

	s.pending = append(s.pending, mutation)
	s.seen[mutation.OperationID] = mutation
	return mutation, nil
}

func (s *MemoryReplayStore) ListPending(_ context.Context, limit int) ([]ReplayMutation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > len(s.pending) {
		limit = len(s.pending)
	}

	items := make([]ReplayMutation, 0, limit)
	for i := 0; i < limit; i++ {
		clone := s.pending[i]
		clone.Payload = copyPayload(clone.Payload)
		items = append(items, clone)
	}
	return items, nil
}

func (s *MemoryReplayStore) MarkApplied(_ context.Context, operationIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(operationIDs) == 0 {
		return nil
	}

	applied := make(map[string]struct{}, len(operationIDs))
	for _, id := range operationIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		applied[trimmed] = struct{}{}
	}

	filtered := s.pending[:0]
	for _, item := range s.pending {
		if _, ok := applied[item.OperationID]; ok {
			continue
		}
		filtered = append(filtered, item)
	}
	s.pending = filtered
	return nil
}

func (s *MemoryReplayStore) RecordConflict(_ context.Context, conflict ConflictRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(conflict.ID) == "" {
		conflict.ID = uuid.NewString()
	}
	if conflict.ResolvedAt.IsZero() {
		conflict.ResolvedAt = time.Now().UTC()
	}
	conflict.EntityID = strings.TrimSpace(conflict.EntityID)
	conflict.EventType = strings.TrimSpace(strings.ToLower(conflict.EventType))
	conflict.Reason = strings.TrimSpace(conflict.Reason)

	s.conflicts = append(s.conflicts, conflict)
	return nil
}

func (s *MemoryReplayStore) ListConflicts(_ context.Context, entityID string, limit int) ([]ConflictRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entityID = strings.TrimSpace(entityID)
	items := make([]ConflictRecord, 0, len(s.conflicts))
	for _, conflict := range s.conflicts {
		if entityID != "" && conflict.EntityID != entityID {
			continue
		}
		items = append(items, conflict)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ResolvedAt.After(items[j].ResolvedAt)
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}
