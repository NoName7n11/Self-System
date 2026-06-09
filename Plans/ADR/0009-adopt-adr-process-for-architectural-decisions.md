# ADR 0009: Adopt ADR Process for Architectural Decisions

## Status
Accepted

## Date
2026-05-27

## Decision Date
2026-05-27

## Context
The project needs a durable record of architectural decisions to prevent
re-litigation, support onboarding, and make future reversals explicit. Current
planning documents capture outcomes but not always the reasoning behind them.

## Decision
Adopt an ADR process using Plans/ADR with numbered, one-decision-per-file
records, a fixed status set, and explicit superseding rules.

## Consequences

Positive:
- Preserves decision rationale over time
- Speeds onboarding and refactors
- Makes reversals explicit and traceable

Negative:
- Adds lightweight documentation work per major decision

## Alternatives Considered

### Rely on plans only
Pros:
- Less documentation overhead

Cons:
- Reasoning is hard to trace later

### Maintain ad hoc decision notes
Pros:
- Lower process overhead

Cons:
- Inconsistent format and easy to miss

## Notes
- New ADRs should use Plans/ADR/template.md.
