# Change 13 Workstream - Scale, Performance & Maintainability

Date: 2026-06-10
Status: Planned
Scope: Address scale ceilings and maintainability debt before they bite: Postgres driver migration, vector-search performance, the HTTP god-file split, graph rendering limits, and an explicit performance budget. Lower urgency than Changes 11/12 — no live Postgres deployment yet — but should land before the Postgres/sync path or large datasets solidify.

## Objective

The system works at personal scale today. This change removes the known ceilings so growth does not force a rewrite: migrate off the maintenance-mode `lib/pq` driver, keep vector search fast as data grows, prevent the WebGL graph from melting laptops, and split the 1338-line HTTP handler before it rots. None of this is urgent at current usage — it is the "do it before it's load-bearing" tier.

## Guiding Constraints

- No behavior change for the SQLite (default/local) path — these changes target the Postgres path, performance, and code structure.
- Vector and driver changes must keep the existing repository interfaces unchanged (swap implementations, not contracts — per ADR 0002).
- The handler split is a pure refactor: identical routes, identical responses, no logic change.
- Each change gated behind tests proving equivalence before/after.

## Workstream 1 — Postgres Driver Migration (lib/pq → pgx)

Objective:
Move off the maintenance-mode `lib/pq` to `jackc/pgx` for performance, context support, and active maintenance.

Key tasks:
- [ ] Replace `lib/pq` with `pgx` (stdlib `database/sql` adapter `pgx/stdlib` to minimize churn, or native pool if justified).
- [ ] Verify all Postgres repository adapters + event store + migrations compile and pass against pgx.
- [ ] Run the DSN-gated Postgres integration suite (`go test ./internal/repository/postgres -run Integration`) on pgx.

Deliverables:
- [ ] Updated `go.mod` (pgx in, lib/pq out).
- [ ] Updated Postgres adapters as needed for pgx.
- [ ] Green Postgres integration run recorded.

Done criteria:
- [ ] `lib/pq` removed from `go.mod`.
- [ ] Postgres integration tests pass on pgx.

## Workstream 2 — Vector Search Performance

Objective:
Keep semantic search fast as the corpus grows; plan the path past brute force.

Key tasks:
- [ ] Load vectors into memory once (warm cache, invalidate on mutation) instead of deserializing from DB per query.
- [ ] Confirm/keep model-version filtering so queries never compare across embedding models.
- [ ] Document the HNSW path (pure-Go, e.g. coder/hnsw) for SQLite and pgvector for the Postgres path; define the crossover trigger (e.g. > 50k vectors or measured p99 > target).

Deliverables:
- [ ] In-memory vector cache in `internal/repository/sqlite/vector_repository.go` (or a service-level cache) + invalidation hook.
- [ ] Benchmark: brute-force cosine over a synthetic 50k-vector set with recorded p50/p99.
- [ ] HNSW/pgvector migration note appended to the perf budget doc (WS4).

Done criteria:
- [ ] Search no longer re-reads all vectors from disk per query.
- [ ] A 50k-vector benchmark exists with recorded latency.
- [ ] The scale-out path (HNSW / pgvector) and its trigger are documented.

## Workstream 3 — HTTP Handler Split

Objective:
Break the 1338-line `internal/http/handler.go` god file into per-domain handlers.

Key tasks:
- [ ] Split into `resource_handler.go`, `category_handler.go`, `todo_handler.go`, `reminder_handler.go`, `graph_handler.go`, `chat_handler.go`, `processing_handler.go`, keeping route registration centralized.
- [ ] No route or response-shape changes — pure mechanical extraction.
- [ ] Existing handler tests pass unchanged.

Deliverables:
- [ ] Per-domain handler files under `internal/http/`.
- [ ] Slimmed `handler.go` retaining wiring/registration only.

Done criteria:
- [ ] No single HTTP handler file exceeds ~300 lines.
- [ ] All existing `internal/http` tests pass with no edits to assertions.

## Workstream 4 — Graph Rendering Limits & Perf Budget

Objective:
Stop the force-graph from melting laptops, and make scale targets checkable.

Key tasks:
- [ ] Force-graph LOD: node-count thresholds, 2D fallback above threshold, virtualized resource lists.
- [ ] Test with a synthetic 10k-node dataset; record FPS / interaction behavior.
- [ ] Write a perf-budget doc: max resources (e.g. 50k), max sync devices (3), max graph nodes rendered (2k) — so every future scale decision is checkable against it.

Deliverables:
- [ ] Frontend graph LOD + 2D-fallback logic + virtualized lists.
- [ ] Synthetic 10k-node test artifact.
- [ ] `Plans/Performance_Budget.md`.

Done criteria:
- [ ] The graph degrades gracefully (LOD/2D) instead of freezing at high node counts.
- [ ] A perf-budget doc exists with explicit, checkable ceilings.

## Planned Milestones

- [ ] Milestone 13A: Postgres path on pgx, integration green (WS1).
- [ ] Milestone 13B: Vector search cached + benchmarked + scale-out documented (WS2).
- [ ] Milestone 13C: HTTP handlers split per domain (WS3).
- [ ] Milestone 13D: Graph LOD + perf budget doc (WS4).

## Change 13 Definition of Done

- [ ] Postgres path runs on pgx; lib/pq removed.
- [ ] Vector search is memory-cached with a recorded 50k benchmark and a documented HNSW/pgvector path.
- [ ] No HTTP handler file exceeds ~300 lines; routes/responses unchanged.
- [ ] The graph renders large datasets without freezing; a perf-budget doc defines the ceilings.
- [ ] `go test ./...` passes with no regressions.
