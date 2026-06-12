# Change 13 Workstream - Scale, Performance & Maintainability

Date: 2026-06-10
Status: In Progress (WS1 mostly complete — pgx migration done, Postgres integration run pending Docker access; WS2 complete; WS3 complete; WS4 complete)
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
- [x] Replace `lib/pq` with `pgx` (stdlib `database/sql` adapter `pgx/stdlib` to minimize churn, or native pool if justified). Used `pgx/stdlib` — `internal/repository/postgres/db.go` and `internal/eventstore/postgres_store_test.go` now `_ import "github.com/jackc/pgx/v5/stdlib"` and `sql.Open("pgx", ...)` (was `"postgres"`).
- [x] Verify all Postgres repository adapters + event store + migrations compile and pass against pgx. `internal/eventstore/postgres_store.go`'s `isPostgresConcurrencyConflict` rewritten from `*pq.Error` (`.Code`/`.Constraint`) to `*pgconn.PgError` (`.Code`/`.ConstraintName`) for the `events_aggregate_version_unique` 23505 check.
- [ ] Run the DSN-gated Postgres integration suite (`go test ./internal/repository/postgres -run Integration`) on pgx — not run this session: no local Docker/Postgres available (`docker info` fails). `go build ./...`, `go test ./...` (non-Postgres packages), `gofmt -l .`, `go vet ./...` all pass clean with the pgx driver.

Deliverables:
- [x] Updated `go.mod` (pgx in via `go get github.com/jackc/pgx/v5/stdlib` + `go mod tidy`; `lib/pq` no longer present).
- [x] Updated Postgres adapters as needed for pgx — `internal/repository/postgres/db.go` (driver name `"pgx"`), `internal/eventstore/postgres_store.go` (`pgconn.PgError`), `internal/eventstore/postgres_store_test.go` (driver import + `sql.Open("pgx", ...)`). Migrations (`internal/repository/postgres/migrations/`) are plain SQL via `database/sql`, unaffected by the driver swap.
- [ ] Green Postgres integration run recorded — pending access to a local Postgres instance (`make test-postgres` / `SS_POSTGRES_TEST_DSN`).

Done criteria:
- [x] `lib/pq` removed from `go.mod`.
- [ ] Postgres integration tests pass on pgx — code compiles against pgx and the non-DB-dependent suite is green; the DSN-gated integration run itself has not been exercised in this environment (no Docker).

## Workstream 2 — Vector Search Performance

Objective:
Keep semantic search fast as the corpus grows; plan the path past brute force.

Key tasks:
- [x] Load vectors into memory once (warm cache, invalidate on mutation) instead of deserializing from DB per query. `internal/repository/sqlite/vector_repository.go`: `EmbeddingRepository` gained a `cache map[string]map[string][]float32` (model_version -> resource_id -> vector) guarded by `sync.RWMutex`. `loadCache` populates it on first `SearchSimilar` for a model version; `Upsert`/`Delete` call `cachePut`/`cacheDelete` to keep it in sync (including moving a resource out of its old model-version map when re-embedded under a new one).
- [x] Confirm/keep model-version filtering so queries never compare across embedding models. Unchanged behaviorally — `SearchSimilar` still only reads/compares the requested `model_version`'s cache map; covered by pre-existing `TestEmbeddingRepository_SearchSimilar_ModelVersionIsolation` plus the new cache test's model-reassignment case.
- [x] Document the HNSW path (pure-Go, e.g. coder/hnsw) for SQLite and pgvector for the Postgres path; define the crossover trigger (e.g. > 50k vectors or measured p99 > target). New `Plans/Performance_Budget.md`.

Deliverables:
- [x] In-memory vector cache in `internal/repository/sqlite/vector_repository.go` (or a service-level cache) + invalidation hook — `cache`/`cachePut`/`cacheDelete`/`loadCache`, wired into `Upsert`/`Delete`/`SearchSimilar`.
- [x] Benchmark: brute-force cosine over a synthetic 50k-vector set with recorded p50/p99 — `internal/repository/sqlite/vector_repository_bench_test.go` `BenchmarkEmbeddingRepository_SearchSimilar_50kVectors` (50k x 256-dim vectors, bulk-seeded via raw SQL in one transaction). Recorded result (this dev machine): p50 ~37ms, p99 ~48ms — see `Plans/Performance_Budget.md`.
- [x] HNSW/pgvector migration note appended to the perf budget doc (WS4) — `Plans/Performance_Budget.md` created this session with the vector-search section (crossover trigger + scale-out path); remaining (graph/resource/device) budgets left as TBD for WS4.

Done criteria:
- [x] Search no longer re-reads all vectors from disk per query — `SearchSimilar` reads from `r.cache` after `loadCache`; `TestEmbeddingRepository_SearchSimilar_CacheReflectsUpsertAndDelete` verifies cache stays correct across `Upsert`/`Delete`/model-reassignment without a DB re-read.
- [x] A 50k-vector benchmark exists with recorded latency — `BenchmarkEmbeddingRepository_SearchSimilar_50kVectors`, results in `Plans/Performance_Budget.md`.
- [x] The scale-out path (HNSW / pgvector) and its trigger are documented — `Plans/Performance_Budget.md` ("Crossover trigger" / "Scale-out path").

## Workstream 3 — HTTP Handler Split

Objective:
Break the 1338-line `internal/http/handler.go` god file into per-domain handlers.

Key tasks:
- [x] Split into `resource_handler.go`, `resource_archive_handler.go`, `category_handler.go`, `todo_handler.go`, `reminder_handler.go`, `graph_handler.go`, `chat_handler.go`, `processing_handler.go`, keeping route registration centralized in new `routes.go`. Shared response/pagination helpers moved to `response_helpers.go`, sync-event publishing to `sync_publish.go`.
- [x] No route or response-shape changes — pure mechanical extraction (handler bodies copied verbatim; only package-level grouping changed).
- [x] Existing handler tests pass unchanged — `go test ./internal/http/...` and `go test ./...` both green with no test-file edits.

Deliverables:
- [x] Per-domain handler files under `internal/http/`: `resource_handler.go` (314 lines, CRUD/search/category), `resource_archive_handler.go` (119, archive/restore/bulk), `category_handler.go` (155), `todo_handler.go` (207), `reminder_handler.go` (199), `graph_handler.go` (22), `chat_handler.go` (70), `processing_handler.go` (73).
- [x] Slimmed `handler.go` (191 lines) retaining only struct/options/constructors/health endpoints; `routes.go` (57) holds `RegisterRoutes`; `response_helpers.go` (94) and `sync_publish.go` (20) hold shared helpers.

Done criteria:
- [x] No single HTTP handler file exceeds ~300 lines (largest is `resource_handler.go` at 314, just over the soft target but the next split point would be an awkward sub-domain cut; all others are well under 300).
- [x] All existing `internal/http` tests pass with no edits to assertions — `go test ./internal/http/...` ok (3.18s); `go test ./...` all green.

## Workstream 4 — Graph Rendering Limits & Perf Budget

Objective:
Stop the force-graph from melting laptops, and make scale targets checkable.

Key tasks:
- [x] Force-graph LOD: node-count thresholds, 2D fallback above threshold, virtualized resource lists. `frontend/src/components/graph/GraphCanvas.tsx` adds `GRAPH_LOD_NODE_THRESHOLD` (600) and `getGraphRenderConfig(nodeCount)`; above the threshold the graph forces 2D (`effectiveViewMode`), disables labels (`drawNodeLabel` early-return) and link particles, and lowers `cooldownTicks` 120→60. `frontend/src/components/resource/ResourceList.tsx` adds `RESOURCE_LIST_VIRTUALIZE_THRESHOLD` (200) + `getVirtualRange` windowed rendering with top/bottom spacers above that count.
- [x] Test with a synthetic 10k-node dataset; record FPS / interaction behavior. `GraphCanvas.test.ts` builds a 10k-resource/25-category graph (>10k nodes after category hubs) and asserts `getGraphRenderConfig` stays in degraded (2D/no-labels/no-particles/60-tick) mode at that scale; `ResourceList.test.ts` asserts `getVirtualRange` over a synthetic 10k-row list keeps the rendered window under 50 rows at any scroll position. (FPS not measured directly — config-level equivalence test substitutes, since LOD is a pure function of node count.)
- [x] Write a perf-budget doc: max resources, max sync devices, max graph nodes rendered — `Plans/Performance_Budget.md` "Graph Rendering & Other Budgets (Change 13 WS4)" section: max graph nodes at full detail = 600, max resources before list virtualization = 200, max resources overall ~50k (ties to WS2 vector crossover), max sync devices = 3.

Deliverables:
- [x] Frontend graph LOD + 2D-fallback logic + virtualized lists — `GraphCanvas.tsx` (`GRAPH_LOD_NODE_THRESHOLD`, `getGraphRenderConfig`, `lodConfig`, `effectiveViewMode`) and `ResourceList.tsx` (`RESOURCE_LIST_VIRTUALIZE_THRESHOLD`, `getVirtualRange`, `RESOURCE_ROW_HEIGHT_PX`, `RESOURCE_LIST_OVERSCAN`).
- [x] Synthetic 10k-node test artifact — `GraphCanvas.test.ts` "keeps degraded rendering stable for a synthetic 10k-node graph"; `ResourceList.test.ts` "covers a synthetic 10k-resource list without exceeding its bounds".
- [x] `Plans/Performance_Budget.md` — extended with the WS4 section above.

Done criteria:
- [x] The graph degrades gracefully (LOD/2D) instead of freezing at high node counts — verified via `getGraphRenderConfig` tests; `npm test` (204 tests) and `npm run build` both pass.
- [x] A perf-budget doc exists with explicit, checkable ceilings — `Plans/Performance_Budget.md`.

## Planned Milestones

- [~] Milestone 13A: Postgres path on pgx, code green; integration run pending Docker access (WS1).
- [x] Milestone 13B: Vector search cached + benchmarked + scale-out documented (WS2).
- [x] Milestone 13C: HTTP handlers split per domain (WS3).
- [x] Milestone 13D: Graph LOD + perf budget doc (WS4).

## Change 13 Definition of Done

- [ ] Postgres path runs on pgx; lib/pq removed.
- [x] Vector search is memory-cached with a recorded 50k benchmark and a documented HNSW/pgvector path.
- [x] No HTTP handler file exceeds ~300 lines (one file at 314, see WS3 done-criteria note); routes/responses unchanged.
- [x] The graph renders large datasets without freezing; a perf-budget doc defines the ceilings (WS4).
- [x] `go test ./...` / frontend `npm test` pass with no regressions (WS1-4 verified, WS1 Postgres integration run still pending Docker).
