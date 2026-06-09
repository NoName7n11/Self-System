# ADR 0015: Redaction via Envelope-Preserving Rewrite

## Status
Accepted

## Date
2026-05-30

## Decision Date
2026-05-30

## Context
Privacy requirements can require deleting sensitive payloads while preserving
ordering and audit metadata.

## Decision
Implement redaction as a dedicated event that triggers a controlled rewrite of
the target event payload to {"redacted": true}. Preserve the envelope fields and
set `redacted = true`. Rebuild projections for affected aggregates.

## Consequences

Positive:
- Preserves audit trail and sequence integrity
- Meets data removal requirements

Negative:
- Not strictly append-only
- Requires controlled rewrite path and tooling

## Alternatives Considered

### Hard delete events
Pros:
- Strong data removal

Cons:
- Breaks sequence integrity and audit history

### Encrypt payloads with key deletion
Pros:
- Append-only preserved

Cons:
- Complex key management and recovery

## Notes
- Pattern decision P6 for Change 3.
