# ADR 0016: Dual-Write Transition Policy

## Status
Accepted

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
The system must migrate incrementally without breaking read paths or existing
API behavior. A phased transition is required.

## Decision
Use a feature flag that controls only the write path. When enabled, writes
append events and update the existing state projection in the same transaction.
Reads continue to use the state table until the sync read path switch
completes. Backfill and projection rebuild run only when the flag is on for the
aggregate type.

## Consequences

Positive:
- Safe incremental rollout
- Preserves read-path stability

Negative:
- Temporary dual-write complexity
- Requires strict transactional boundaries

## Alternatives Considered

### Big-bang cutover
Pros:
- Simpler long-term architecture

Cons:
- High risk and difficult rollback

### Read-switch first
Pros:
- Validates projection early

Cons:
- Drift risk without event-first writes

## Notes
- Pattern decision P7 for Change 3.
