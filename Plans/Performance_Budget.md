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

## Other budgets (Change 13 WS4, pending)

- Max resources: TBD
- Max sync devices: TBD
- Max graph nodes rendered (force-graph LOD threshold): TBD
