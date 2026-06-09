# ADR 0008: Remove DGraph from Active Architecture

## Status
Accepted

## Date
2026-05-27

## Decision Date
2026-05-27

## Context
DGraph was introduced for graph storage, but it adds an additional service and
operational overhead. For Phase 1 and Phase 2, graph relationships can be
represented in relational storage and computed in the application layer without
a dedicated graph database.

## Decision
Remove DGraph from the active architecture. Store graph relationships in the
relational store (SQLite locally and PostgreSQL centrally) and compute graph
views in application logic. Revisit a dedicated graph store only if traversal
requirements exceed what relational storage can handle.

## Consequences

Positive:
- Fewer services to run and maintain
- Simplifies local-first setup and sync deployment
- Reduces data duplication across stores

Negative:
- More application-level work to compute graph views
- Potentially less efficient traversal for large graphs

## Alternatives Considered

### Keep DGraph
Pros:
- Native graph traversal features

Cons:
- Additional infrastructure and data complexity

### Defer the decision
Pros:
- Preserves optionality

Cons:
- Keeps operational overhead without clear near-term value

## Notes
- Supersedes ADR 0005.
