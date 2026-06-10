package eventstore

import (
	"context"
	"testing"
	"time"
)

func TestEventObservabilityNilSafe(t *testing.T) {
	var obs *EventObservability
	obs.RecordAppend()
	obs.RecordOCCRetry()
	obs.RecordProjectorLatency(time.Millisecond)
	obs.RecordRedaction()
	snap := obs.Snapshot()
	if snap.AppendsTotal != 0 {
		t.Fatalf("expected 0 appends on nil obs, got %d", snap.AppendsTotal)
	}
}

func TestEventObservabilityCounters(t *testing.T) {
	obs := NewEventObservability()
	obs.RecordAppend()
	obs.RecordAppend()
	obs.RecordOCCRetry()
	obs.RecordRedaction()

	snap := obs.Snapshot()
	if snap.AppendsTotal != 2 {
		t.Fatalf("appends: want 2, got %d", snap.AppendsTotal)
	}
	if snap.OCCRetriesTotal != 1 {
		t.Fatalf("occ_retries: want 1, got %d", snap.OCCRetriesTotal)
	}
	if snap.RedactionsTotal != 1 {
		t.Fatalf("redactions: want 1, got %d", snap.RedactionsTotal)
	}
}

func TestEventObservabilityProjectorLatencyAvg(t *testing.T) {
	obs := NewEventObservability()
	obs.RecordProjectorLatency(2 * time.Millisecond)
	obs.RecordProjectorLatency(4 * time.Millisecond)

	snap := obs.Snapshot()
	if snap.ProjectorApplyCount != 2 {
		t.Fatalf("apply_count: want 2, got %d", snap.ProjectorApplyCount)
	}
	// avg should be 3ms; accept 2–4ms range to tolerate rounding
	if snap.ProjectorAvgLatencyMs < 2.0 || snap.ProjectorAvgLatencyMs > 4.5 {
		t.Fatalf("avg_latency_ms out of expected range: %f", snap.ProjectorAvgLatencyMs)
	}
}

func TestProjectorRegistryNoLatencyOnUnmatchedType(t *testing.T) {
	obs := NewEventObservability()
	reg := NewProjectorRegistry()
	reg.SetObservability(obs)

	reg.RegisterSync("KnownEvent", func(_ context.Context, _ Event, _ TxConn) error {
		return nil
	})

	_ = reg.ApplySync(context.Background(), Event{EventType: "OtherEvent"}, nil)
	if obs.Snapshot().ProjectorApplyCount != 0 {
		t.Fatal("projector_apply_count should be 0 for unregistered event type")
	}
}

func TestProjectorRegistryRecordsLatencyOnMatch(t *testing.T) {
	obs := NewEventObservability()
	reg := NewProjectorRegistry()
	reg.SetObservability(obs)

	reg.RegisterSync("MyEvent", func(_ context.Context, _ Event, _ TxConn) error {
		return nil
	})

	_ = reg.ApplySync(context.Background(), Event{EventType: "MyEvent"}, nil)
	snap := obs.Snapshot()
	if snap.ProjectorApplyCount != 1 {
		t.Fatalf("projector_apply_count: want 1, got %d", snap.ProjectorApplyCount)
	}
}
