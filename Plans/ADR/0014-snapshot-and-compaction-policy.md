# ADR 0014: Snapshot and Compaction Policy

## Status
Superseded by ADR 0018

> **Superseded (2026-06-07):** ADR 0018 demoted the event log to an audit trail +
> sync outbox; the mutable tables are the source of truth and there is no
> rebuild-from-events guarantee. Snapshots existed solely to bound rebuild time,
> so the snapshot worker has been deleted. The `projection_snapshots` table and
> the `Store.Snapshot` method remain in place but unused (dropping the table would
> need a destructive migration not worth writing). The cadence policy below is no
> longer in force.

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
Event logs grow unbounded. Rebuild time must remain predictable as history
expands.

## Decision
Create snapshots every 100 events or every 30 days, whichever comes first.
Store snapshots in `projection_snapshots` keyed by (aggregate_id,
snapshot_version). Rebuild loads latest snapshot and applies newer events. Raw
events are retained 365 days, then archived to cold storage.

## Consequences

Positive:
- Predictable rebuild times
- Controlled storage growth

Negative:
- Snapshot storage adds complexity
- Archival process must be maintained

## Alternatives Considered

### No snapshots
Pros:
- Simpler system

Cons:
- Rebuild time grows without bound

### Snapshot on every event
Pros:
- Fastest rebuilds

Cons:
- Excessive storage and write overhead

## Notes
- Pattern decision P5 for Change 3.
