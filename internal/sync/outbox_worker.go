package sync

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"selfsystems/internal/eventstore"
)

const (
	defaultOutboxPollInterval = 250 * time.Millisecond
	defaultOutboxBatchSize    = 100
)

// OutboxWorker tails the events table and publishes resource events to the
// sync hub (P4 — outbox pattern for sync emission).
//
// On startup it replays all events from sequence 0, which:
//   - Populates the hub's in-memory history cache
//   - Sets hub sequence numbers equal to eventstore sequence numbers, allowing
//     reconnecting clients to use the same since_sequence for both hub history
//     and events-table replay.
//
// In steady state it polls every pollInterval for newly appended events.
type OutboxWorker struct {
	store        eventstore.Store
	hub          *Hub
	pollInterval time.Duration
	batchSize    int
	lastSeq      atomic.Int64
	published    atomic.Int64
}

func NewOutboxWorker(store eventstore.Store, hub *Hub, pollInterval time.Duration) *OutboxWorker {
	if pollInterval <= 0 {
		pollInterval = defaultOutboxPollInterval
	}
	return &OutboxWorker{
		store:        store,
		hub:          hub,
		pollInterval: pollInterval,
		batchSize:    defaultOutboxBatchSize,
	}
}

// Start begins polling. Blocks until ctx is cancelled; call in a goroutine.
func (w *OutboxWorker) Start(ctx context.Context) {
	logger := syncLogger()
	logger.Info("outbox worker starting", "poll_interval", w.pollInterval)

	// Catch-up loop: drain all historical events quickly before settling
	// into the steady-state polling cadence.
	for {
		n := w.poll(ctx)
		if n < w.batchSize {
			break // caught up
		}
		if ctx.Err() != nil {
			logger.Info("outbox worker stopped during catch-up")
			return
		}
	}
	logger.Info("outbox worker caught up", "last_seq", w.lastSeq.Load(), "published", w.published.Load())

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("outbox worker stopped")
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

// poll reads one batch from the events table and publishes translatable events.
// Returns the number of events read (may be 0 if nothing new).
func (w *OutboxWorker) poll(ctx context.Context) int {
	events, err := w.store.ReadBySequence(ctx, w.lastSeq.Load(), w.batchSize)
	if err != nil {
		syncLogger().Warn("outbox worker read failed", "error", err)
		return 0
	}

	for _, e := range events {
		// Always advance lastSeq so we never re-read the same event.
		w.lastSeq.Store(e.Sequence)

		syncEvt, ok := outboxTranslate(e)
		if !ok {
			continue
		}
		w.hub.Publish(syncEvt)
		w.published.Add(1)
	}
	return len(events)
}

// LastSequence returns the highest eventstore sequence delivered so far.
func (w *OutboxWorker) LastSequence() int64 { return w.lastSeq.Load() }

// Published returns the total number of sync events published to the hub.
func (w *OutboxWorker) Published() int64 { return w.published.Load() }

// outboxTranslate converts an eventstore event to a sync hub event.
// Returns false for event types that have no sync representation.
func outboxTranslate(e eventstore.Event) (Event, bool) {
	var syncType string
	switch e.EventType {
	case eventstore.EventTypeResourceCreated, eventstore.EventTypeResourceImported:
		syncType = EventTypeResourceCreated
	case eventstore.EventTypeResourceUpdated, eventstore.EventTypeResourceCategoryAssigned:
		syncType = EventTypeResourceUpdated
	case eventstore.EventTypeResourceDeleted:
		syncType = EventTypeResourceDeleted
	case eventstore.EventTypeCategoryCreated, eventstore.EventTypeCategoryUpdated, eventstore.EventTypeCategoryDeleted:
		syncType = EventTypeCategoryUpdated
	case eventstore.EventTypeTodoCreated, eventstore.EventTypeTodoUpdated, eventstore.EventTypeTodoDeleted:
		syncType = EventTypeTodoUpdated
	case eventstore.EventTypeReminderCreated, eventstore.EventTypeReminderUpdated, eventstore.EventTypeReminderDeleted:
		syncType = EventTypeReminderUpdated
	default:
		return Event{}, false
	}

	payload := BuildEventPayload(map[string]any{
		"url": extractPayloadString(e.Payload, "url"),
	}, e.AggregateID, EventSourceOutboxWorker)

	return Event{
		// Use eventstore sequence so hub sequence space aligns with events
		// table, enabling direct events-table queries during WS reconnect.
		Sequence:  e.Sequence,
		Type:      syncType,
		Payload:   payload,
		Timestamp: e.RecordedAt,
	}, true
}

// eventOriginIsOutbox reports whether a hub event was published by the outbox
// worker (i.e. it originates from the events table). Such events are already
// represented by the durable events-table read during reconnect.
func eventOriginIsOutbox(e Event) bool {
	m, ok := e.Payload.(map[string]any)
	if !ok {
		return false
	}
	src, _ := m[PayloadKeyEventSource].(string)
	return src == EventSourceOutboxWorker
}

// mergeDurableAndHubReplay merges the durable events-table replay with hub
// history during reconnect.
//
// Hub events are deduped by ORIGIN, not by raw sequence. A hub event is dropped
// only when it came from the outbox worker (already covered by the events-table
// read). Deduping by raw sequence — the previous behaviour — was unsound: the
// hub mints independent sequence numbers for directly-published events
// (category/todo/reminder/chat mutations, sync.publish, replay fanout), and
// those numbers can collide with unrelated events-table sequences, silently
// dropping a real event from the reconnect stream (Finding 3 residual edge case).
func mergeDurableAndHubReplay(stored []eventstore.Event, hubEvents []Event) []Event {
	replayed := make([]Event, 0, len(stored)+len(hubEvents))
	for _, e := range stored {
		if syncEvt, ok := outboxTranslate(e); ok {
			replayed = append(replayed, syncEvt)
		}
	}
	for _, he := range hubEvents {
		if eventOriginIsOutbox(he) {
			continue // already represented by the durable events-table read
		}
		replayed = append(replayed, he)
	}
	sortEventsBySequence(replayed)
	return replayed
}

// extractPayloadString reads a top-level string field from raw JSON.
// Returns "" on any parse error rather than failing.
func extractPayloadString(raw []byte, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
