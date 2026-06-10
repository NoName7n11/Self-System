package sync

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// ObservabilitySnapshot exposes runtime counters for sync/auth/replay signals.
type ObservabilitySnapshot struct {
	AuthFailuresTotal             int64 `json:"auth_failures_total"`
	WebSocketUpgradeFailuresTotal int64 `json:"websocket_upgrade_failures_total"`
	WebSocketConnectionsTotal     int64 `json:"websocket_connections_total"`
	WebSocketDisconnectionsTotal  int64 `json:"websocket_disconnections_total"`

	SyncEventsPublishedTotal int64 `json:"sync_events_published_total"`
	SyncEventRejectedTotal   int64 `json:"sync_event_rejected_total"`

	ReplayEnqueueAcceptedTotal int64 `json:"replay_enqueue_accepted_total"`
	ReplayEnqueueRejectedTotal int64 `json:"replay_enqueue_rejected_total"`

	ReplayRequestsTotal          int64 `json:"replay_requests_total"`
	ReplayRequestsSucceededTotal int64 `json:"replay_requests_succeeded_total"`
	ReplayRequestsFailedTotal    int64 `json:"replay_requests_failed_total"`
	ReplayMutationsReplayedTotal int64 `json:"replay_mutations_replayed_total"`
	ReplayEventsEmittedTotal     int64 `json:"replay_events_emitted_total"`
	ReplayConflictsTotal         int64 `json:"replay_conflicts_total"`
	ReplayQueueDepth             int   `json:"replay_queue_depth"`
	ReplayQueueOldestSeconds     int64 `json:"replay_queue_oldest_seconds"`

	ConflictListRequestsTotal int64 `json:"conflict_list_requests_total"`
	ConflictListFailedTotal   int64 `json:"conflict_list_failed_total"`
}

// Observability stores lightweight in-memory counters for sync runtime events.
type Observability struct {
	authFailures            atomic.Int64
	wsUpgradeFailures       atomic.Int64
	wsConnections           atomic.Int64
	wsDisconnections        atomic.Int64
	syncEventsPublished     atomic.Int64
	syncEventsRejected      atomic.Int64
	replayEnqueueAccepted   atomic.Int64
	replayEnqueueRejected   atomic.Int64
	replayRequests          atomic.Int64
	replayRequestsSucceeded atomic.Int64
	replayRequestsFailed    atomic.Int64
	replayMutationsReplayed atomic.Int64
	replayEventsEmitted     atomic.Int64
	replayConflicts         atomic.Int64
	conflictListRequests    atomic.Int64
	conflictListFailed      atomic.Int64
}

func NewObservability() *Observability {
	return &Observability{}
}

func (o *Observability) Snapshot() ObservabilitySnapshot {
	if o == nil {
		return ObservabilitySnapshot{}
	}

	return ObservabilitySnapshot{
		AuthFailuresTotal:             o.authFailures.Load(),
		WebSocketUpgradeFailuresTotal: o.wsUpgradeFailures.Load(),
		WebSocketConnectionsTotal:     o.wsConnections.Load(),
		WebSocketDisconnectionsTotal:  o.wsDisconnections.Load(),
		SyncEventsPublishedTotal:      o.syncEventsPublished.Load(),
		SyncEventRejectedTotal:        o.syncEventsRejected.Load(),
		ReplayEnqueueAcceptedTotal:    o.replayEnqueueAccepted.Load(),
		ReplayEnqueueRejectedTotal:    o.replayEnqueueRejected.Load(),
		ReplayRequestsTotal:           o.replayRequests.Load(),
		ReplayRequestsSucceededTotal:  o.replayRequestsSucceeded.Load(),
		ReplayRequestsFailedTotal:     o.replayRequestsFailed.Load(),
		ReplayMutationsReplayedTotal:  o.replayMutationsReplayed.Load(),
		ReplayEventsEmittedTotal:      o.replayEventsEmitted.Load(),
		ReplayConflictsTotal:          o.replayConflicts.Load(),
		ConflictListRequestsTotal:     o.conflictListRequests.Load(),
		ConflictListFailedTotal:       o.conflictListFailed.Load(),
	}
}

func (o *Observability) SnapshotWithQueue(queue ReplayQueueSnapshot) ObservabilitySnapshot {
	snapshot := o.Snapshot()
	snapshot.ReplayQueueDepth = queue.PendingCount
	if !queue.OldestEnqueuedAt.IsZero() {
		snapshot.ReplayQueueOldestSeconds = int64(time.Since(queue.OldestEnqueuedAt).Seconds())
		if snapshot.ReplayQueueOldestSeconds < 0 {
			snapshot.ReplayQueueOldestSeconds = 0
		}
	}

	return snapshot
}

// StatusMiddleware tracks unauthorized responses, including auth middleware aborts.
func (o *Observability) StatusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() == http.StatusUnauthorized {
			o.RecordAuthFailure()
		}
	}
}

func (o *Observability) RecordAuthFailure() {
	if o != nil {
		o.authFailures.Add(1)
	}
}

func (o *Observability) RecordWSUpgradeFailure() {
	if o != nil {
		o.wsUpgradeFailures.Add(1)
	}
}

func (o *Observability) RecordWSConnected() {
	if o != nil {
		o.wsConnections.Add(1)
	}
}

func (o *Observability) RecordWSDisconnected() {
	if o != nil {
		o.wsDisconnections.Add(1)
	}
}

func (o *Observability) RecordSyncEventPublished() {
	if o != nil {
		o.syncEventsPublished.Add(1)
	}
}

func (o *Observability) RecordSyncEventRejected() {
	if o != nil {
		o.syncEventsRejected.Add(1)
	}
}

func (o *Observability) RecordReplayEnqueueAccepted() {
	if o != nil {
		o.replayEnqueueAccepted.Add(1)
	}
}

func (o *Observability) RecordReplayEnqueueRejected() {
	if o != nil {
		o.replayEnqueueRejected.Add(1)
	}
}

func (o *Observability) RecordReplayRequest(summary ReplaySummary, replayErr error) {
	if o == nil {
		return
	}

	o.replayRequests.Add(1)
	if replayErr != nil {
		o.replayRequestsFailed.Add(1)
		return
	}

	o.replayRequestsSucceeded.Add(1)
	o.replayMutationsReplayed.Add(int64(summary.ReplayedCount))
	o.replayEventsEmitted.Add(int64(summary.EmittedCount))
	o.replayConflicts.Add(int64(summary.ConflictCount))
}

func (o *Observability) RecordConflictListRequest(listErr error) {
	if o == nil {
		return
	}

	o.conflictListRequests.Add(1)
	if listErr != nil {
		o.conflictListFailed.Add(1)
	}
}
