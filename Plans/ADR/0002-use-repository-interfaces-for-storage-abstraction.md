# ADR 0002: Use Repository Interfaces for Storage Abstraction

## Status
Accepted

## Date
2026-05-27

## Decision Date
Approx. 2026-04-11

## Context
The system must stay modular and loosely coupled so features can be removed
without cascading changes. Storage backends include local SQLite and a future
central PostgreSQL store. We need clear contracts between domain logic and
persistence adapters.

## Decision
Define repository interfaces in the domain layer and implement storage adapters
in the repository layer.

## Consequences

Positive:
- Storage adapters are swappable without changing business logic
- Easier testing with mocks and fakes
- Supports gradual migrations (SQLite to PostgreSQL)

Negative:
- Additional interface and adapter boilerplate
- Risk of interface drift if contracts are not kept in sync

## Alternatives Considered

### Direct DB access in services
Pros:
- Less boilerplate

Cons:
- Tight coupling to storage and harder migrations

### Generic data access layer only
Pros:
- Fewer interfaces to maintain

Cons:
- Less explicit contracts and weaker separation of concerns

## Notes
- This ADR is retrospective; reasoning is reconstructed from plans and current context.
