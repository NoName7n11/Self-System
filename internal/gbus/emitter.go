package gbus

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"selfsystems/internal/eventstore"
)

const (
	// AggregateTypeGBUS is the aggregate_type written for all GBUS signal events.
	AggregateTypeGBUS = "gbus_signal"
	// EventTypeGBUSBase is the event_type prefix; the signal type is appended.
	EventTypeGBUSBase = "gbus.signal"
)

// SignalEmitter writes GBUS interaction signals as events in the event store.
// All emissions are fire-and-forget — they run in a background goroutine and
// never block the primary operation.
type SignalEmitter struct {
	eventStore eventstore.Store
	enabled    bool
	// sessionID identifies this process run. Per Phase_3_GBUS_Signals_Feature_Store.md
	// a session is "app start or a 30-minute inactivity gap" — approximated here as
	// one session per emitter (process) lifetime.
	sessionID string
}

// NewSignalEmitter creates a SignalEmitter. When enabled=false or store=nil,
// Emit is a no-op; callers need not check the flag themselves.
func NewSignalEmitter(store eventstore.Store, enabled bool) *SignalEmitter {
	return &SignalEmitter{eventStore: store, enabled: enabled, sessionID: uuid.NewString()}
}

// Emit writes a GBUS signal event asynchronously. Returns immediately.
func (e *SignalEmitter) Emit(ctx context.Context, payload GBUSSignalPayload) {
	if !e.enabled || e.eventStore == nil {
		return
	}
	if payload.OccurredAt.IsZero() {
		payload.OccurredAt = time.Now().UTC()
	}
	if payload.Weight == 0 {
		if w, ok := SignalWeights[payload.SignalType]; ok {
			payload.Weight = w
		}
	}
	if payload.UserID == "" {
		payload.UserID = DefaultUserID
	}
	if payload.SessionID == "" {
		payload.SessionID = e.sessionID
	}
	go e.emit(payload)
}

func (e *SignalEmitter) emit(payload GBUSSignalPayload) {
	b, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("gbus: marshal signal payload", "signal_type", payload.SignalType, "error", err)
		return
	}
	evt := eventstore.Event{
		EventID:       uuid.NewString(),
		AggregateID:   uuid.NewString(), // each signal is its own aggregate
		AggregateType: AggregateTypeGBUS,
		EventType:     EventTypeGBUSBase + "." + payload.SignalType,
		EventVersion:  1,
		Payload:       json.RawMessage(b),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := e.eventStore.Append(ctx, evt); err != nil {
		slog.Warn("gbus: emit signal", "signal_type", payload.SignalType, "error", err)
	}
}
