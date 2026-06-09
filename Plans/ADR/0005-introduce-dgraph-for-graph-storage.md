# ADR 0005: Introduce DGraph for Graph Storage

## Status
Superseded

## Date
2026-05-27

## Decision Date
Approx. 2026-04-11

## Context
The knowledge graph relies on category-to-resource relationships and graph
traversal. A graph database was selected to store edges and enable traversal
queries separate from relational storage.

## Decision
Use DGraph to store graph relationships (categories, resources, and related
category edges).

## Consequences

Positive:
- Natural fit for graph traversal and related-category queries
- GraphQL and DQL interfaces align with graph use cases

Negative:
- Adds an extra service to operate and deploy
- Increases data duplication across storage layers

## Alternatives Considered

### Store edges in SQLite or PostgreSQL
Pros:
- Fewer services and simpler operations

Cons:
- More application-level graph computation

### In-memory graph only
Pros:
- No additional storage dependencies

Cons:
- Persistence and sync are harder to maintain

## Notes
- This ADR is retrospective; reasoning is reconstructed from plans and current context.
- Superseded by ADR 0008.
