package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"selfsystems/internal/eventstore"
)

// appendWithTx appends an event and runs all sync projectors in one transaction.
// On ErrConcurrencyConflict it re-reads the latest version for aggregateID and
// retries up to maxRetries times (OCC retry per P1).
// obs is nil-safe; pass nil when observability is not wired.
func appendWithTx(
	ctx context.Context,
	store eventstore.Store,
	projectors *eventstore.ProjectorRegistry,
	evt eventstore.Event,
	obs *eventstore.EventObservability,
) (eventstore.Event, error) {
	const maxRetries = 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var appliedSeq int64
		err := store.WithTx(ctx, func(tx eventstore.TxStore) error {
			result, appErr := tx.Append(ctx, evt)
			if appErr != nil {
				return appErr
			}
			appliedSeq = result.Sequence
			if !result.Applied {
				// Duplicate event_id: the original append already committed its
				// projection atomically. Re-projecting would corrupt the
				// projection with this call's in-memory payload. Skip.
				return nil
			}
			return projectors.ApplySync(ctx, evt, tx.Conn())
		})
		if err == nil {
			evt.Sequence = appliedSeq
			obs.RecordAppend()
			return evt, nil
		}
		if !errors.Is(err, eventstore.ErrConcurrencyConflict) {
			return eventstore.Event{}, err
		}
		obs.RecordOCCRetry()
		version, readErr := aggregateLatestVersion(ctx, store, evt.AggregateID)
		if readErr != nil {
			return eventstore.Event{}, readErr
		}
		evt.EventVersion = version + 1
		evt.EventID = uuid.NewString()
	}
	return eventstore.Event{}, eventstore.ErrConcurrencyConflict
}

func aggregateLatestVersion(ctx context.Context, store eventstore.Store, aggregateID string) (int, error) {
	events, err := store.ReadByAggregate(ctx, aggregateID, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("read aggregate version: %w", err)
	}
	if len(events) == 0 {
		return 0, nil
	}
	return events[len(events)-1].EventVersion, nil
}
