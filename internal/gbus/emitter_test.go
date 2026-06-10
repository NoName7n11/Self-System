package gbus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"selfsystems/internal/eventstore"
)

// stubStore is a minimal in-memory event store for emitter tests.
type stubStore struct {
	events []eventstore.Event
}

func (s *stubStore) Append(_ context.Context, evt eventstore.Event) (eventstore.AppendResult, error) {
	s.events = append(s.events, evt)
	return eventstore.AppendResult{Applied: true}, nil
}
func (s *stubStore) ReadByAggregate(_ context.Context, _ string, _, _ int) ([]eventstore.Event, error) {
	return nil, nil
}
func (s *stubStore) ReadBySequence(_ context.Context, _ int64, _ int) ([]eventstore.Event, error) {
	return s.events, nil
}
func (s *stubStore) Snapshot(_ context.Context, _ eventstore.Snapshot) error   { return nil }
func (s *stubStore) Redact(_ context.Context, _ string) error                   { return nil }
func (s *stubStore) LatestSequence(_ context.Context) (int64, error)            { return 0, nil }
func (s *stubStore) WithTx(_ context.Context, fn func(eventstore.TxStore) error) error {
	return fn(nil)
}

func waitForEvents(store *stubStore, want int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(store.events) >= want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func TestEmitter_Disabled_NoEvents(t *testing.T) {
	store := &stubStore{}
	emitter := NewSignalEmitter(store, false)
	emitter.Emit(context.Background(), GBUSSignalPayload{SignalType: SignalResourceSaved})
	time.Sleep(20 * time.Millisecond)
	if len(store.events) != 0 {
		t.Errorf("expected 0 events when disabled, got %d", len(store.events))
	}
}

func TestEmitter_NilStore_NoOp(t *testing.T) {
	emitter := NewSignalEmitter(nil, true)
	// Should not panic.
	emitter.Emit(context.Background(), GBUSSignalPayload{SignalType: SignalResourceSaved})
}

func TestEmitter_EmitsManualClassificationSignal(t *testing.T) {
	store := &stubStore{}
	emitter := NewSignalEmitter(store, true)
	emitter.Emit(context.Background(), GBUSSignalPayload{
		SignalType: SignalManualClassification,
		CategoryID: "cat-1",
		ResourceID: "res-1",
	})
	if !waitForEvents(store, 1, time.Second) {
		t.Fatal("expected event to be emitted")
	}
	evt := store.events[0]
	if evt.AggregateType != AggregateTypeGBUS {
		t.Errorf("aggregate_type = %q, want %q", evt.AggregateType, AggregateTypeGBUS)
	}
	expectedType := EventTypeGBUSBase + "." + SignalManualClassification
	if evt.EventType != expectedType {
		t.Errorf("event_type = %q, want %q", evt.EventType, expectedType)
	}
	var payload GBUSSignalPayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Weight != SignalWeights[SignalManualClassification] {
		t.Errorf("weight = %v, want %v", payload.Weight, SignalWeights[SignalManualClassification])
	}
}

func TestEmitter_AllCoreSignalTypes(t *testing.T) {
	signalTypes := []string{
		SignalManualClassification,
		SignalCategoryCorrection,
		SignalAutoClassification,
		SignalResourceSaved,
		SignalResourceDeleted,
		SignalResourceRevisited,
		SignalCounterIncremented,
	}
	for _, st := range signalTypes {
		store := &stubStore{}
		emitter := NewSignalEmitter(store, true)
		emitter.Emit(context.Background(), GBUSSignalPayload{SignalType: st})
		if !waitForEvents(store, 1, time.Second) {
			t.Errorf("signal type %q: event not emitted", st)
			continue
		}
		var payload GBUSSignalPayload
		if err := json.Unmarshal(store.events[0].Payload, &payload); err != nil {
			t.Errorf("signal type %q: unmarshal: %v", st, err)
			continue
		}
		if payload.SignalType != st {
			t.Errorf("signal type %q: payload.SignalType = %q", st, payload.SignalType)
		}
		if payload.Weight == 0 {
			t.Errorf("signal type %q: weight should be non-zero", st)
		}
		if payload.OccurredAt.IsZero() {
			t.Errorf("signal type %q: occurred_at should be set", st)
		}
	}
}

func TestEmitter_ExplicitWeight_NotOverridden(t *testing.T) {
	store := &stubStore{}
	emitter := NewSignalEmitter(store, true)
	emitter.Emit(context.Background(), GBUSSignalPayload{
		SignalType: SignalResourceSaved,
		Weight:     0.99,
	})
	if !waitForEvents(store, 1, time.Second) {
		t.Fatal("event not emitted")
	}
	var payload GBUSSignalPayload
	if err := json.Unmarshal(store.events[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Weight != 0.99 {
		t.Errorf("explicit weight overridden: got %v, want 0.99", payload.Weight)
	}
}

func TestEmitter_AsyncNonBlocking(t *testing.T) {
	store := &stubStore{}
	emitter := NewSignalEmitter(store, true)
	start := time.Now()
	for i := 0; i < 10; i++ {
		emitter.Emit(context.Background(), GBUSSignalPayload{SignalType: SignalResourceSaved})
	}
	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Errorf("Emit calls should return immediately; took %v", elapsed)
	}
	waitForEvents(store, 10, time.Second)
}
