package eventstore

import (
	"sync/atomic"
	"time"
)

// EventObservability tracks lightweight in-memory counters for the event
// sourcing runtime. Nil-safe on all methods.
type EventObservability struct {
	appendsTotal        atomic.Int64
	occRetriesTotal     atomic.Int64
	projectorApplyCount atomic.Int64
	projectorLatencyUs  atomic.Int64 // cumulative microseconds for avg calculation
	redactionsTotal     atomic.Int64
}

// EventObservabilitySnapshot is a consistent point-in-time read of all counters.
type EventObservabilitySnapshot struct {
	AppendsTotal           int64   `json:"appends_total"`
	OCCRetriesTotal        int64   `json:"occ_retries_total"`
	ProjectorApplyCount    int64   `json:"projector_apply_count"`
	ProjectorAvgLatencyMs  float64 `json:"projector_avg_latency_ms"`
	RedactionsTotal        int64   `json:"redactions_total"`
}

func NewEventObservability() *EventObservability {
	return &EventObservability{}
}

func (o *EventObservability) RecordAppend() {
	if o != nil {
		o.appendsTotal.Add(1)
	}
}

func (o *EventObservability) RecordOCCRetry() {
	if o != nil {
		o.occRetriesTotal.Add(1)
	}
}

func (o *EventObservability) RecordProjectorLatency(d time.Duration) {
	if o != nil {
		o.projectorApplyCount.Add(1)
		o.projectorLatencyUs.Add(d.Microseconds())
	}
}

func (o *EventObservability) RecordRedaction() {
	if o != nil {
		o.redactionsTotal.Add(1)
	}
}

func (o *EventObservability) Snapshot() EventObservabilitySnapshot {
	if o == nil {
		return EventObservabilitySnapshot{}
	}
	count := o.projectorApplyCount.Load()
	latUs := o.projectorLatencyUs.Load()
	var avgMs float64
	if count > 0 {
		avgMs = float64(latUs) / float64(count) / 1000.0
	}
	return EventObservabilitySnapshot{
		AppendsTotal:          o.appendsTotal.Load(),
		OCCRetriesTotal:       o.occRetriesTotal.Load(),
		ProjectorApplyCount:   count,
		ProjectorAvgLatencyMs: avgMs,
		RedactionsTotal:       o.redactionsTotal.Load(),
	}
}
