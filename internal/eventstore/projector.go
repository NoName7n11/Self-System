package eventstore

import (
	"context"
	"time"
)

// SyncProjector runs inside the event append transaction (P8).
// conn is the same database connection used for the event append, so the
// projector can write to projection tables atomically.
type SyncProjector func(ctx context.Context, event Event, conn TxConn) error

// AsyncProjector runs after the event append transaction commits (P8).
// It must not assume it will be retried on failure.
type AsyncProjector func(ctx context.Context, event Event)

// ProjectorRegistry maps event types to ordered lists of projectors.
type ProjectorRegistry struct {
	sync  map[string][]SyncProjector
	async map[string][]AsyncProjector
	obs   *EventObservability
}

func NewProjectorRegistry() *ProjectorRegistry {
	return &ProjectorRegistry{
		sync:  make(map[string][]SyncProjector),
		async: make(map[string][]AsyncProjector),
	}
}

// SetObservability wires an EventObservability into the registry so that
// ApplySync records projector latency automatically.
func (r *ProjectorRegistry) SetObservability(obs *EventObservability) {
	if r != nil {
		r.obs = obs
	}
}

// RegisterSync adds a sync projector for eventType.
func (r *ProjectorRegistry) RegisterSync(eventType string, p SyncProjector) {
	r.sync[eventType] = append(r.sync[eventType], p)
}

// RegisterAsync adds an async projector for eventType.
func (r *ProjectorRegistry) RegisterAsync(eventType string, p AsyncProjector) {
	r.async[eventType] = append(r.async[eventType], p)
}

// ApplySync runs all sync projectors for event.EventType in registration order.
// conn is the event append's transaction connection.
// Returns the first error encountered; remaining projectors are skipped.
// Records total latency across all projectors to the wired EventObservability.
func (r *ProjectorRegistry) ApplySync(ctx context.Context, event Event, conn TxConn) error {
	projectors := r.sync[event.EventType]
	if len(projectors) == 0 {
		return nil
	}
	start := time.Now()
	for _, p := range projectors {
		if err := p(ctx, event, conn); err != nil {
			return err
		}
	}
	r.obs.RecordProjectorLatency(time.Since(start))
	return nil
}

// ApplyAsync runs all async projectors for event.EventType after the tx commits.
func (r *ProjectorRegistry) ApplyAsync(ctx context.Context, event Event) {
	for _, p := range r.async[event.EventType] {
		p(ctx, event)
	}
}
