package gbus

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"selfsystems/internal/eventstore"
)

// Aggregator tails gbus_signal events from the event store and maintains
// per-category and per-resource feature tables for the inference engine.
// One aggregation cycle is bounded to 30 seconds.
type Aggregator struct {
	eventStore    eventstore.Store
	featureStore  FeatureStore
	retentionDays int
	lastSequence  int64
}

// NewAggregator creates an Aggregator. retentionDays controls how long raw
// signal rows are retained (0 = keep forever).
func NewAggregator(es eventstore.Store, fs FeatureStore, retentionDays int) *Aggregator {
	return &Aggregator{
		eventStore:    es,
		featureStore:  fs,
		retentionDays: retentionDays,
	}
}

// Run executes one aggregation cycle: reads new gbus_signal events since the
// last run and upserts feature rows. Returns the number of events processed.
func (a *Aggregator) Run(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	const batchSize = 1000
	events, err := a.eventStore.ReadBySequence(ctx, a.lastSequence, batchSize)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, evt := range events {
		// Track sequence regardless of whether we process the event.
		seq := evt.Sequence

		if evt.AggregateType != AggregateTypeGBUS || !strings.HasPrefix(evt.EventType, EventTypeGBUSBase+".") {
			a.lastSequence = seq
			continue
		}

		var payload GBUSSignalPayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			slog.Warn("gbus aggregator: parse payload", "event_id", evt.EventID, "error", err)
			a.lastSequence = seq
			continue
		}

		if payload.CategoryID != "" {
			if uErr := a.featureStore.UpsertCategoryFeature(ctx, payload.CategoryID, payload.SignalType, payload.Weight); uErr != nil {
				slog.Warn("gbus aggregator: upsert category feature", "error", uErr)
			}
		}
		if payload.ResourceID != "" {
			if uErr := a.featureStore.UpsertResourceFeature(ctx, payload.ResourceID, payload.SignalType, payload.Weight); uErr != nil {
				slog.Warn("gbus aggregator: upsert resource feature", "error", uErr)
			}
		}

		a.lastSequence = seq
		processed++
	}

	// Prune feature rows older than the retention window.
	if a.retentionDays > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -a.retentionDays)
		if n, pErr := a.featureStore.PruneOlderThan(ctx, cutoff); pErr != nil {
			slog.Warn("gbus aggregator: prune old features", "error", pErr)
		} else if n > 0 {
			slog.Info("gbus aggregator: pruned old features", "count", n)
		}
	}

	return processed, nil
}

// Start runs the aggregator on a daily ticker until ctx is cancelled.
// One catch-up cycle runs immediately at startup.
func (a *Aggregator) Start(ctx context.Context) {
	go func() {
		if n, err := a.Run(ctx); err != nil {
			slog.Warn("gbus aggregator: startup run", "error", err)
		} else if n > 0 {
			slog.Info("gbus aggregator: startup run complete", "processed", n)
		}

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := a.Run(ctx); err != nil {
					slog.Warn("gbus aggregator: daily run", "error", err)
				} else {
					slog.Info("gbus aggregator: daily run complete", "processed", n)
				}
			}
		}
	}()
}
