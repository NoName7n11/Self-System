# ADR 0011: Idempotent Event Appends

## Status
Accepted

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
Sync retries and client replays can submit the same event more than once. The
append path must be safe under retries and network duplication.

## Decision
Require a client-supplied UUID `event_id` and enforce idempotency with
UNIQUE(event_id). Inserts use ON CONFLICT DO NOTHING.

## Consequences

Positive:
- Safe retries for sync and offline replay
- Simplifies duplicate handling

Negative:
- Reuse of event_id can mask bugs
- Requires callers to generate stable ids

## Alternatives Considered

### Server-generated ids
Pros:
- Simpler client implementation

Cons:
- Hard to dedupe retries and replays

### Deduping by payload hash
Pros:
- No client id requirement

Cons:
- Collisions and expensive hashing

## Notes
- Pattern decision P2 for Change 3.
