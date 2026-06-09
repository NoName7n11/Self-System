# ADR 0012: Event Payload Versioning and Upcasters

## Status
Accepted

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
Event payloads are immutable once written, but schemas evolve. We need a safe
way to read older events without rewriting history.

## Decision
Add `payload_schema_version` to each event. Maintain read-time upcasters that
convert older payloads to the current shape before applying projections. When
semantics change, use a new event type.

## Consequences

Positive:
- Preserves immutability while supporting schema evolution
- Keeps projections consistent across versions

Negative:
- Ongoing maintenance of upcasters
- More testing required for version transitions

## Alternatives Considered

### Rewrite historical events
Pros:
- Single current schema

Cons:
- Breaks immutability and auditability

### Freeze payload schema forever
Pros:
- No versioning complexity

Cons:
- Blocks model evolution and fixes

## Notes
- Pattern decision P3 for Change 3.
