# ADR 0017: Projector Classification and Registry

## Status
Accepted

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
Event handling spans sync emission, projections, deep processing, and GBUS
aggregation. Some handlers must run inside the write transaction, others should
run asynchronously.

## Decision
Introduce a ProjectorRegistry that registers handlers as synchronous or
asynchronous. Synchronous projectors run inside the append transaction for
critical projections. Asynchronous projectors run out of band for side effects
like sync emission, deep processing enqueue, and GBUS aggregation.

## Consequences

Positive:
- Clear separation of critical vs side-effect handlers
- Consistent registration and execution model

Negative:
- Async lag must be monitored
- Requires idempotent projector design

## Alternatives Considered

### All projectors synchronous
Pros:
- Simpler consistency model

Cons:
- Slower writes and higher coupling

### All projectors asynchronous
Pros:
- Faster writes

Cons:
- Risk of stale projections during reads

## Notes
- Pattern decision P8 for Change 3.
