# ADR 0003: Use SQLite Locally and Postgres for Central Storage

## Status
Accepted

## Date
2026-05-27

## Decision Date
Approx. 2026-04-12

## Context
Phase 1 is local-first and must work without setup friction. Phase 2 introduces
multi-device sync, which requires a central store with durable concurrency
handling. The architecture needs a local database for desktop use and a central
database for sync runtime.

## Decision
Use SQLite as the local default store and PostgreSQL as the central store for
sync in Phase 2+.

## Consequences

Positive:
- SQLite provides zero-configuration local storage
- PostgreSQL is reliable for central, multi-user sync
- Fits the repository adapter strategy for migrations

Negative:
- Dual database support increases migration and testing overhead
- Data parity must be maintained across adapters

## Alternatives Considered

### SQLite everywhere
Pros:
- Single database to support

Cons:
- Less suitable for a multi-device central store

### PostgreSQL everywhere
Pros:
- Single production-grade database

Cons:
- Adds local setup friction and heavier runtime for Phase 1

## Notes
- This ADR is retrospective; reasoning is reconstructed from plans and current context.
