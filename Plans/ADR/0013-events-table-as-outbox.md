# ADR 0013: Events Table as Outbox

## Status
Accepted

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
Sync emission must be consistent with persisted state. A separate event stream
risks drift between writes and notifications.

## Decision
Treat the `events` table as the outbox. A worker tails by `sequence` and
publishes to the sync hub. Delivery is at-least-once, and subscribers dedupe by
`event_id`.

## Consequences

Positive:
- Eliminates drift between state and sync events
- Decouples write latency from network emission

Negative:
- Requires outbox lag monitoring
- Adds an async worker dependency

## Alternatives Considered

### Emit directly on write
Pros:
- Lower end-to-end latency

Cons:
- Can publish events that later roll back

### Separate outbox table
Pros:
- Isolates sync traffic

Cons:
- Duplicates data and logic

## Notes
- Pattern decision P4 for Change 3.
