# ADR 0007: Use OpenAPI as the HTTP Contract Source

## Status
Accepted

## Date
2026-05-27

## Decision Date
Approx. 2026-04-11

## Context
The API surface needs a single, durable contract for development, testing, and
integration. The project already maintains a spec in api/openapi.yaml and uses
it as the reference for route behavior.

## Decision
Use OpenAPI as the source of truth for the HTTP contract and keep
api/openapi.yaml up to date with behavior changes.

## Consequences

Positive:
- Clear API contract for onboarding and integration
- Aligns tests and documentation with actual behavior
- Supports future client generation if needed

Negative:
- Requires ongoing maintenance to avoid drift
- Adds documentation work to feature changes

## Alternatives Considered

### Code-first documentation only
Pros:
- Less manual documentation work

Cons:
- Harder to review and diff contract changes

### gRPC or Proto-only contracts
Pros:
- Strongly typed contracts

Cons:
- Higher adoption cost for existing REST clients

## Notes
- This ADR is retrospective; reasoning is reconstructed from plans and current context.
