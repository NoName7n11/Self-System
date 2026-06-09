# ADR 0004: Adopt Last-Write-Wins for Phase 2 Sync Conflicts

## Status
Accepted

## Date
2026-05-27

## Decision Date
Approx. 2026-04-14

## Context
Phase 2 introduces multi-device sync with offline edits. The system needs a
predictable conflict resolution strategy that is simple to implement, test, and
explain, while preserving a conflict history for recovery.

## Decision
Adopt a deterministic last-write-wins (LWW) strategy for conflict resolution.
The winning change is the write with the latest timestamp, and overwritten data
is retained in conflict history for recovery.

## Consequences

Positive:
- Deterministic and simple resolution
- Easy to test and explain
- Works well with offline replay pipelines

Negative:
- Older writes can be lost without manual review
- Relies on timestamps and can be sensitive to clock skew

## Alternatives Considered

### Manual conflict resolution UI
Pros:
- Preserves user intent on conflicts

Cons:
- Higher UX and implementation complexity

### Vector clocks or CRDTs
Pros:
- Stronger consistency semantics

Cons:
- Higher complexity and operational cost

## Notes
- This ADR is retrospective; reasoning is reconstructed from plans and current context.
