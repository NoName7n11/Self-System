# ADR 0006: Split AI Processing into Skim and Deep Stages

## Status
Accepted

## Date
2026-05-27

## Decision Date
Approx. 2026-04-11

## Context
The system needs fast user feedback on resource ingestion while still providing
richer analysis later. AI calls can be slow and rate-limited, so the processing
pipeline should avoid blocking the UX.

## Decision
Split AI processing into two stages: a fast skim stage for immediate
classification and a queued deep stage for full extraction and enrichment.

## Consequences

Positive:
- Immediate feedback to the user
- Deep processing can be queued and rate-limited
- Graceful degradation if deep processing fails

Negative:
- Added pipeline complexity and eventual consistency
- Requires status tracking between stages

## Alternatives Considered

### Deep-only processing on save
Pros:
- Single pipeline, simpler state

Cons:
- Slower user feedback and higher latency

### Synchronous full processing
Pros:
- Complete data before graph update

Cons:
- Poor UX and fragile under rate limits

## Notes
- This ADR is retrospective; reasoning is reconstructed from plans and current context.
