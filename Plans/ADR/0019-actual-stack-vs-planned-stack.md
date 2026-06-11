# ADR 0019: Actual Stack vs. Planned Stack (Reconciliation)

## Status
Accepted

## Date
2026-06-11

## Decision Date
Approx. 2026-05 to 2026-06 (incremental, across Changes 1-10)

## Context

`Plans/Technical_Stack.md` (last fully written ~March 2026) named GORM, Asynq + Redis,
and sqlite-vec as the data/queue/vector layers. None of these ever landed in `go.mod`.
Over Changes 1-10 the implementation diverged onto a simpler, dependency-light stack,
but the planning docs were never updated to match. This created drift: the docs no
longer described the system that exists, and any future decision (human or AI agent)
reading the plans would inherit a wrong architecture.

This ADR records the actual stack as a deliberate, accepted decision so the
reconciliation (Change 11) is not re-litigated.

## Decision

The implemented stack diverges from the original plan in four areas:

1. **Database access: `database/sql` + hand-rolled repository adapters, not GORM.**
   Repositories (`internal/repository/sqlite`, `internal/repository/postgres`) implement
   `internal/domain` interfaces directly over `*sql.DB` / `modernc.org/sqlite`. Schema
   changes are versioned SQL migration files, not GORM auto-migration.

2. **Background processing: in-process goroutine queue, not Asynq + Redis.**
   `internal/service/deep_processor.go` runs as a goroutine inside the same binary
   (server or Wails desktop app) and processes an in-memory queue. GBUS aggregation
   (`internal/gbus/aggregator.go`) runs as a bounded daily in-process job. There is no
   `cmd/worker`, no Redis, no external broker.

3. **Vector search: pure-Go brute-force cosine similarity, not sqlite-vec.**
   `sqlite-vec` (a CGo/C extension) is incompatible with `modernc.org/sqlite` (pure Go,
   no CGo). Embeddings are stored in SQLite columns and ranked via brute-force cosine
   similarity in `internal/repository/sqlite/vector_repository.go`.

4. **AI models are config-driven, not hardcoded.**
   `config/config.default.yml` defines per-task model names (low_cost_model,
   high_cost_model, classification, embeddings) overridable via `SS_AI_*` env vars,
   across OpenAI, Anthropic, and Gemini providers, with a heuristic fallback provider
   for offline/no-key operation.

## Consequences

Positive:
- No CGo dependency — Go cross-compiles natively for Windows/Linux/macOS (supports the
  Wails desktop build matrix in Change 9).
- No Docker/Redis requirement for local single-user operation — fewer moving parts,
  faster `wails dev` / `make run` startup.
- Repository-interface boundary (ADR 0002) means swapping in GORM, Asynq, or Qdrant
  later remains possible without touching service-layer code, if scale ever requires it.
- Planning docs now match `go.mod` / running code, reducing risk of future work being
  designed against a nonexistent architecture.

Negative:
- Brute-force cosine search is O(n) per query; will need replacing (Qdrant, per
  `Technical_Stack.md` Section 10) if resource counts grow large enough to matter.
- In-process queue means background work is lost on crash (no durable broker); event
  sourcing (ADR 0013, 0018) provides the audit trail / replay safety net instead.
- Hand-rolled SQL repositories require more boilerplate per entity than GORM
  auto-migration, but keep query plans explicit and avoid ORM N+1 surprises.

## Alternatives Considered

### Option A: Retrofit the original plan (add GORM, Asynq, Redis, sqlite-vec now)
Pros:
- Matches the original Q31-Q50 design intent.

Cons:
- Large migration effort with no functional benefit at current (single-user, personal)
  scale.
- Reintroduces CGo (sqlite-vec) and Docker/Redis dependencies that the current desktop
  (Wails) packaging avoids.
- No evidence the simpler stack is a bottleneck.

### Option B: Document actual stack, keep planned stack as future-scale option (chosen)
Pros:
- Docs reflect reality now; future-scale options remain documented in
  `Technical_Stack.md` Section 10 (Qdrant) without being load-bearing today.
- Zero runtime risk — docs-only change.

Cons:
- Two "modes" referenced in docs (actual vs. future) requires readers to check which
  applies — mitigated by the superseded-section banner in `Technical_Stack.md`.

## Notes
- This ADR is retrospective: written after the divergence already happened across
  Changes 1-10, reconstructed from `go.mod`, `package.json`, and the current
  `internal/` package layout (Change 11, Workstream 2).
- See `Plans/Technical_Stack.md` for the per-section "ACTUAL:" annotations this ADR
  backs.
