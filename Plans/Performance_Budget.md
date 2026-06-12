# Performance Budget

This document records explicit, checkable scale ceilings for Self Systems, and
the plan for what changes when a ceiling is approached. Started under Change
13 (Scale, Performance & Maintainability); extended as later workstreams add
their own budgets/benchmarks.

## Vector Search (Change 13 WS2)

### Current implementation

`internal/repository/sqlite/vector_repository.go` does brute-force cosine
similarity in pure Go. As of WS2, vectors for each `model_version` are loaded
into an in-memory cache on first search and kept in sync via `Upsert`/
`Delete` (`cachePut`/`cacheDelete`/`loadCache`), so steady-state search no
longer re-deserializes every row from disk per query.

Model-version filtering is preserved: `SearchSimilar` only ever compares
vectors stored under the requested `model_version`, so a model upgrade can't
silently mix incompatible embedding spaces.

### Benchmark (50k vectors, 256-dim, in-memory cache warm)

`internal/repository/sqlite/vector_repository_bench_test.go`,
`BenchmarkEmbeddingRepository_SearchSimilar_50kVectors`:

```
go test ./internal/repository/sqlite -run NONE -bench SearchSimilar_50k -benchtime=20x
```

Recorded result (2026-06-12, dev machine, 12th Gen Intel i7-1260P):

| Metric | Value |
|---|---|
| Vectors | 50,000 |
| Dimensions | 256 |
| p50 | ~37 ms |
| p99 | ~48 ms |

This is per-query latency for a brute-force scan of the full cached corpus —
acceptable for personal-scale usage (the repo's stated target is well under
50k resources today), but it scales linearly with vector count.

### Crossover trigger

Re-evaluate brute-force search when **either**:

- the embedding corpus exceeds **~50,000 vectors** for a single model
  version, or
- measured p99 search latency exceeds **~100 ms** in real usage.

### Scale-out path

- **SQLite (local-first) path**: pure-Go HNSW (e.g. `coder/hnsw`) layered in
  front of `EmbeddingRepository` — build an approximate index alongside the
  existing brute-force cache, used once the corpus crosses the trigger above.
  The `domain.EmbeddingRepository` interface (ADR 0002) stays unchanged; only
  the SQLite adapter's internals would change.
- **Postgres path**: `pgvector` extension + an ANN index (HNSW/IVFFlat) on
  `resource_embeddings.vector`, queried via SQL `ORDER BY vector <-> $1
  LIMIT $2` instead of loading rows into Go. Requires a Postgres-side
  migration and a `pgvector`-aware implementation of
  `domain.EmbeddingRepository` for `internal/repository/postgres`.

Neither path is needed yet — both are documented here so the decision is
triggered by a measurement, not a guess.

## Graph Rendering & Other Budgets (Change 13 WS4)

### Graph LOD (Level of Detail)

`frontend/src/components/graph/GraphCanvas.tsx` exports `GRAPH_LOD_NODE_THRESHOLD`
(`600`) and `getGraphRenderConfig(nodeCount)`. Above the threshold, the graph:

- forces 2D mode (`forceMode: "2d"`), overriding the user's 3D view selection,
  via `effectiveViewMode = lodConfig.forceMode ?? viewMode`,
- stops drawing node labels (`drawNodeLabel` early-returns),
- disables link directional particles (`linkDirectionalParticles: 0`),
- reduces `cooldownTicks` from 120 to 60 so the force simulation settles
  faster and stops consuming CPU sooner.

A "LOD active (2D, labels off)" indicator is shown in the graph meta bar when
this kicks in. `GraphCanvas.test.ts` covers `getGraphRenderConfig` at the
threshold and above it, including a synthetic 10,000-node graph (built via
`buildGraph` over 10k resources across 25 categories) to confirm the degraded
config stays stable at that scale.

- **Max graph nodes rendered at full detail**: 600 (`GRAPH_LOD_NODE_THRESHOLD`).
  Above this, the graph degrades to 2D/no-labels/reduced-cooldown rather than
  freezing the tab.

### Resource list virtualization

`frontend/src/components/resource/ResourceList.tsx` exports
`RESOURCE_LIST_VIRTUALIZE_THRESHOLD` (`200`). Below this resource count, all
rows render normally. Above it, the list switches to windowed rendering
(`getVirtualRange`): only rows within the visible scroll viewport plus a
6-row overscan are mounted, with top/bottom spacer divs sized by
`RESOURCE_ROW_HEIGHT_PX` (92px) to preserve scrollbar size and position.
`ResourceList.test.ts` covers `getVirtualRange` at list boundaries and for a
synthetic 10,000-row list.

- **Max resources before list virtualization kicks in**: 200
  (`RESOURCE_LIST_VIRTUALIZE_THRESHOLD`); beyond that, list rendering cost is
  bounded by viewport size, not total resource count.

### Other ceilings

- **Max resources (overall)**: ~50,000, matching the vector-search crossover
  trigger above — this is the point at which both vector search and the
  resource list virtualization assumptions should be re-measured.
- **Max sync devices**: 3, per the existing sync/offline-replay design
  (`internal/sync`); not a hard technical limit, but the tested and supported
  ceiling for the WebSocket hub + offline-replay manager.
