package sync

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestLastWriteWinsResolverPrefersNewerTimestamp(t *testing.T) {
	resolver := NewLastWriteWinsResolver()

	existing := ReplayMutation{
		OperationID: "op-1",
		EventType:   EventTypeResourceUpdated,
		EntityID:    "res-1",
		OccurredAt:  time.Date(2026, time.April, 14, 10, 0, 0, 0, time.UTC),
	}
	incoming := ReplayMutation{
		OperationID: "op-2",
		EventType:   EventTypeResourceUpdated,
		EntityID:    "res-1",
		OccurredAt:  time.Date(2026, time.April, 14, 10, 5, 0, 0, time.UTC),
	}

	winner, conflict, err := resolver.Resolve(existing, incoming)
	if err != nil {
		t.Fatalf("resolve conflict: %v", err)
	}
	if winner.OperationID != "op-2" {
		t.Fatalf("expected newer operation op-2 to win, got %q", winner.OperationID)
	}
	if conflict == nil {
		t.Fatalf("expected conflict record")
	}
	if conflict.WinnerOperationID != "op-2" {
		t.Fatalf("expected conflict winner op-2, got %q", conflict.WinnerOperationID)
	}
}

func TestOfflineReplayManagerReplaysFIFOAndRecordsConflict(t *testing.T) {
	hub := NewHub()
	store := NewMemoryReplayStore()
	manager := NewOfflineReplayManager(store, NewLastWriteWinsResolver(), hub)
	ctx := context.Background()

	enqueuedFirst, err := manager.Enqueue(ctx, ReplayMutation{
		OperationID: "op-1",
		EventType:   EventTypeResourceUpdated,
		OccurredAt:  time.Date(2026, time.April, 14, 9, 0, 0, 0, time.UTC),
		Payload: map[string]any{
			PayloadKeyEntityID: "res-1",
			"title":            "old",
		},
	})
	if err != nil {
		t.Fatalf("enqueue first mutation: %v", err)
	}
	if enqueuedFirst.Sequence == 0 {
		t.Fatalf("expected sequence to be assigned")
	}

	if _, err := manager.Enqueue(ctx, ReplayMutation{
		OperationID: "op-2",
		EventType:   EventTypeResourceUpdated,
		OccurredAt:  time.Date(2026, time.April, 14, 9, 5, 0, 0, time.UTC),
		Payload: map[string]any{
			PayloadKeyEntityID: "res-1",
			"title":            "new",
		},
	}); err != nil {
		t.Fatalf("enqueue second mutation: %v", err)
	}

	events, unsubscribe := hub.Subscribe(4)
	defer unsubscribe()

	summary, err := manager.Replay(ctx, 10)
	if err != nil {
		t.Fatalf("replay mutations: %v", err)
	}
	if summary.ReplayedCount != 2 {
		t.Fatalf("expected replayed_count 2, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1, got %d", summary.EmittedCount)
	}
	if summary.ConflictCount != 1 {
		t.Fatalf("expected conflict_count 1, got %d", summary.ConflictCount)
	}

	select {
	case event := <-events:
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("expected map payload, got %T", event.Payload)
		}
		if payload["title"] != "new" {
			t.Fatalf("expected winner payload title new, got %v", payload["title"])
		}
		if payload[PayloadKeyEventSource] != EventSourceSyncReplay {
			t.Fatalf("expected replay event source %q, got %v", EventSourceSyncReplay, payload[PayloadKeyEventSource])
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timed out waiting for replay event")
	}

	conflicts, err := manager.ListConflicts(ctx, "res-1", 10)
	if err != nil {
		t.Fatalf("list conflicts: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected one conflict record, got %d", len(conflicts))
	}
	if conflicts[0].WinnerOperationID != "op-2" {
		t.Fatalf("expected conflict winner op-2, got %q", conflicts[0].WinnerOperationID)
	}
}

func TestOfflineReplayManagerEnqueueIsIdempotentByOperationID(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryReplayStore()
	manager := NewOfflineReplayManager(store, NewLastWriteWinsResolver(), NewHub())

	first, err := manager.Enqueue(ctx, ReplayMutation{
		OperationID: "op-dup",
		EventType:   EventTypeResourceUpdated,
		OccurredAt:  time.Date(2026, time.April, 15, 9, 0, 0, 0, time.UTC),
		Payload: map[string]any{
			PayloadKeyEntityID: "res-dup",
			"title":            "first",
		},
	})
	if err != nil {
		t.Fatalf("enqueue first duplicate mutation: %v", err)
	}

	second, err := manager.Enqueue(ctx, ReplayMutation{
		OperationID: "op-dup",
		EventType:   EventTypeResourceUpdated,
		OccurredAt:  time.Date(2026, time.April, 15, 9, 1, 0, 0, time.UTC),
		Payload: map[string]any{
			PayloadKeyEntityID: "res-dup",
			"title":            "second",
		},
	})
	if err != nil {
		t.Fatalf("enqueue second duplicate mutation: %v", err)
	}
	if first.Sequence == 0 {
		t.Fatalf("expected first enqueue to assign sequence")
	}
	if second.Sequence != first.Sequence {
		t.Fatalf("expected duplicate enqueue to reuse sequence %d, got %d", first.Sequence, second.Sequence)
	}

	summary, err := manager.Replay(ctx, 10)
	if err != nil {
		t.Fatalf("replay idempotent queue: %v", err)
	}
	if summary.QueuedCount != 1 {
		t.Fatalf("expected queued_count 1, got %d", summary.QueuedCount)
	}
	if summary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1, got %d", summary.EmittedCount)
	}

	pending, err := store.ListPending(ctx, 10)
	if err != nil {
		t.Fatalf("list pending after replay: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending items after replay, got %d", len(pending))
	}

	third, err := manager.Enqueue(ctx, ReplayMutation{
		OperationID: "op-dup",
		EventType:   EventTypeResourceUpdated,
		OccurredAt:  time.Date(2026, time.April, 15, 9, 2, 0, 0, time.UTC),
		Payload: map[string]any{
			PayloadKeyEntityID: "res-dup",
			"title":            "third",
		},
	})
	if err != nil {
		t.Fatalf("enqueue duplicate after apply: %v", err)
	}
	if third.Sequence != first.Sequence {
		t.Fatalf("expected duplicate after apply to reuse sequence %d, got %d", first.Sequence, third.Sequence)
	}

	pending, err = store.ListPending(ctx, 10)
	if err != nil {
		t.Fatalf("list pending after duplicate enqueue: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected duplicate enqueue after apply to keep queue empty, got %d", len(pending))
	}
}

func TestOfflineReplayManagerStopsOnApplyFailureWithoutDroppingRemaining(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()
	store := NewMemoryReplayStore()
	manager := NewOfflineReplayManagerWithApplier(store, NewLastWriteWinsResolver(), hub, ReplayMutationApplierFunc(func(_ context.Context, mutation ReplayMutation) error {
		if mutation.EntityID == "res-b" {
			return fmt.Errorf("forced apply failure")
		}
		return nil
	}))

	for _, mutation := range []ReplayMutation{
		{
			OperationID: "op-a",
			EventType:   EventTypeResourceUpdated,
			OccurredAt:  time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC),
			Payload: map[string]any{
				PayloadKeyEntityID: "res-a",
				"title":            "A",
			},
		},
		{
			OperationID: "op-b",
			EventType:   EventTypeResourceUpdated,
			OccurredAt:  time.Date(2026, time.April, 16, 10, 1, 0, 0, time.UTC),
			Payload: map[string]any{
				PayloadKeyEntityID: "res-b",
				"title":            "B",
			},
		},
		{
			OperationID: "op-c",
			EventType:   EventTypeResourceUpdated,
			OccurredAt:  time.Date(2026, time.April, 16, 10, 2, 0, 0, time.UTC),
			Payload: map[string]any{
				PayloadKeyEntityID: "res-c",
				"title":            "C",
			},
		},
	} {
		if _, err := manager.Enqueue(ctx, mutation); err != nil {
			t.Fatalf("enqueue mutation %q: %v", mutation.OperationID, err)
		}
	}

	events, unsubscribe := hub.Subscribe(4)
	defer unsubscribe()

	summary, err := manager.Replay(ctx, 10)
	if err == nil {
		t.Fatalf("expected replay to fail on forced apply error")
	}
	if summary.QueuedCount != 3 {
		t.Fatalf("expected queued_count 3, got %d", summary.QueuedCount)
	}
	if summary.ReplayedCount != 1 {
		t.Fatalf("expected replayed_count 1 before failure, got %d", summary.ReplayedCount)
	}
	if summary.EmittedCount != 1 {
		t.Fatalf("expected emitted_count 1 before failure, got %d", summary.EmittedCount)
	}

	select {
	case event := <-events:
		payload, ok := event.Payload.(map[string]any)
		if !ok {
			t.Fatalf("expected map payload, got %T", event.Payload)
		}
		if payload[PayloadKeyEntityID] != "res-a" {
			t.Fatalf("expected only first entity replayed before failure, got %v", payload[PayloadKeyEntityID])
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("expected replay event for first mutation")
	}

	pending, listErr := store.ListPending(ctx, 10)
	if listErr != nil {
		t.Fatalf("list pending after failed replay: %v", listErr)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending mutations after partial replay failure, got %d", len(pending))
	}
	if pending[0].OperationID != "op-b" || pending[1].OperationID != "op-c" {
		t.Fatalf("expected pending mutations [op-b op-c], got [%s %s]", pending[0].OperationID, pending[1].OperationID)
	}
}
