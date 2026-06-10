package gbus

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"selfsystems/internal/eventstore"
)

// Monitor tracks drift between the active GBUS model and recent user signals.
// Drift > 10% accuracy degradation triggers a warning log and model reload.
type Monitor struct {
	eventStore  eventstore.Store
	inference   *Inference
	modelPath   string
	signalCount atomic.Int64
	lastCheckAt time.Time
}

// NewMonitor creates a Monitor. It shares the same Inference instance so it can
// trigger a Reload when a promoted artifact is detected on disk.
func NewMonitor(es eventstore.Store, inf *Inference, modelPath string) *Monitor {
	return &Monitor{
		eventStore: es,
		inference:  inf,
		modelPath:  modelPath,
	}
}

// SignalCount returns the total number of GBUS signals ingested since startup.
func (m *Monitor) SignalCount() int64 {
	return m.signalCount.Load()
}

// LastCheckAt returns the time of the last drift check.
func (m *Monitor) LastCheckAt() time.Time {
	return m.lastCheckAt
}

// CheckDrift reads recent signal events, estimates current model top-1 accuracy
// on those signals, and compares it to the stored baseline accuracy.
// Returns (currentAccuracy, driftPct, error).
func (m *Monitor) CheckDrift(ctx context.Context) (float64, float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Read most recent 500 signals.
	events, err := m.eventStore.ReadBySequence(ctx, 0, 500)
	if err != nil {
		return 0, 0, err
	}

	correct := 0
	total := 0
	for _, evt := range events {
		if evt.AggregateType != AggregateTypeGBUS {
			continue
		}
		if !strings.HasPrefix(evt.EventType, EventTypeGBUSBase+".") {
			continue
		}
		var payload GBUSSignalPayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			continue
		}
		m.signalCount.Add(1)
		// "Correct" means the category with highest GBUS score matches the
		// signal's category (proxy for whether inference would agree).
		if payload.CategoryID != "" && payload.SignalType == SignalManualClassification {
			score := m.inference.CategoryScore(payload.CategoryID)
			if score >= 0.5 {
				correct++
			}
			total++
		}
	}
	m.lastCheckAt = time.Now().UTC()

	if total == 0 {
		return 1.0, 0, nil
	}

	currentAccuracy := float64(correct) / float64(total)

	// Load baseline from model file to compare.
	var baseline float64
	if b, readErr := os.ReadFile(m.modelPath); readErr == nil {
		var model GBUSModel
		if jsonErr := json.Unmarshal(b, &model); jsonErr == nil {
			baseline = model.ValidationAccuracy
		}
	}

	driftPct := 0.0
	if baseline > 0 {
		driftPct = math.Abs(baseline-currentAccuracy) / baseline * 100
	}
	if driftPct > 10 {
		slog.Warn("gbus monitor: accuracy drift exceeds 10%",
			"current", currentAccuracy,
			"baseline", baseline,
			"drift_pct", driftPct,
		)
		// Attempt to reload in case a new model was promoted on disk.
		m.inference.Reload()
	}

	return currentAccuracy, driftPct, nil
}

// Start runs drift checks on a daily ticker until ctx is cancelled.
func (m *Monitor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, _, err := m.CheckDrift(ctx); err != nil {
					slog.Warn("gbus monitor: drift check", "error", err)
				}
			}
		}
	}()
}
