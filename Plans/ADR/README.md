# Architecture Decision Records (ADR)

An ADR is a short, durable record of an important technical decision.
It captures what we decided, why we decided it, and the tradeoffs accepted.

## When to write one

Write an ADR when a decision affects:
- Architecture or runtime topology
- Operations or deployment
- Developer workflow or tooling
- Migration cost or data model
- Security posture
- Long-term extensibility

Do not write ADRs for trivial implementation details.

## Status values

Use exactly these statuses:
- Proposed: under consideration, not yet decided
- Accepted: current decision in force
- Superseded: replaced by a later ADR
- Rejected: considered and explicitly not chosen

If a decision changes later, do not rewrite the old ADR. Add a new one and mark
the old ADR as Superseded.

## Naming and numbering

- Location: Plans/ADR/
- Filenames: 4-digit prefix + kebab-case title
  Example: 0001-use-wails-for-desktop-shell.md
- Numbering: monotonically increasing, never reuse a number

## Index

- 0001: Use Wails for Desktop Shell (Accepted)
- 0002: Use Repository Interfaces for Storage Abstraction (Accepted)
- 0003: Use SQLite Locally and Postgres for Central Storage (Accepted)
- 0004: Adopt Last-Write-Wins for Phase 2 Sync Conflicts (Accepted)
- 0005: Introduce DGraph for Graph Storage (Superseded by 0008)
- 0006: Split AI Processing into Skim and Deep Stages (Accepted)
- 0007: Use OpenAPI as the HTTP Contract Source (Accepted)
- 0008: Remove DGraph from Active Architecture (Accepted)
- 0009: Adopt ADR Process for Architectural Decisions (Accepted)
- 0010: Optimistic Concurrency Control for Events (Accepted)
- 0011: Idempotent Event Appends (Accepted)
- 0012: Event Payload Versioning and Upcasters (Accepted)
- 0013: Events Table as Outbox (Accepted)
- 0014: Snapshot and Compaction Policy (Superseded by 0018)
- 0015: Redaction via Envelope-Preserving Rewrite (Accepted)
- 0016: Dual-Write Transition Policy (Accepted)
- 0017: Projector Classification and Registry (Accepted)
- 0018: Event Sourcing Demoted to Audit Log and Sync Outbox (Accepted) — supersedes rebuild-from-events implication in 0013; supersedes 0014
- 0019: Actual Stack vs. Planned Stack (Accepted) — records database/sql, in-process queue, and brute-force cosine vector search as the implemented stack vs. GORM/Asynq/Redis/sqlite-vec in the original plan

## Superseding rule

A new ADR that replaces a decision must:
- Mark the old ADR as Superseded
- Reference the new ADR in the old ADR notes
- Reference the old ADR in the new ADR notes

## Retrospective ADRs

If an ADR is written after the decision was already implemented, include a note
in the Notes section that it is retrospective and the reasoning was
reconstructed from plans, code, and current context.

## Template

Use Plans/ADR/template.md for new ADRs.
