# ADR 0010: Optimistic Concurrency Control for Events

## Status
Accepted

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
Concurrent writes to the same aggregate must not silently overwrite each other.
We need a deterministic, low-overhead way to enforce per-aggregate ordering.

## Decision
Use optimistic concurrency control with a UNIQUE constraint on
(aggregate_id, event_version). Writers read the latest version, compute the
next version, attempt INSERT, and retry on conflict.

## Consequences

Positive:
- Prevents lost updates without heavy locking
- Enforces strict per-aggregate ordering

Negative:
- Write contention can cause retry loops
- Writers must read current version before append

## Alternatives Considered

### Advisory locks or serializable transactions
Pros:
- Strong ordering without retries

Cons:
- Higher latency and operational complexity

### Last-write-wins without OCC
Pros:
- Simpler write path

Cons:
- Lost updates and non-deterministic ordering

## Notes
- Pattern decision P1 for Change 3.
