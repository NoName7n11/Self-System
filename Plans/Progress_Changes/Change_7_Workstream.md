# Change 7 Workstream - AI Intelligence Layer

Date: 2026-06-03
Status: WS1 COMPLETE — WS2 COMPLETE — WS3 COMPLETE — WS4 COMPLETE — WS5 COMPLETE — WS6 COMPLETE
Scope: Replace keyword heuristics with real AI classification, add embedding generation, integrate sqlite-vec for vector storage, wire semantic search, and make deep processing a genuine AI enrichment step.

## Objective

Deliver the full intelligence pipeline: classify resources using AI with confidence scores, generate and store embedding vectors, enable semantic search over the knowledge base, and make deep processing produce real AI-generated summaries and key points instead of metadata annotations.

## Guiding Constraints

- Change 6 (content extraction) must be complete before this workstream begins — AI classification and embeddings require real content.
- Keep keyword classifier as a fallback when AI is unavailable or confidence is too low.
- Confidence threshold for auto-save vs. prompt must be configurable (default 0.85).
- Embedding generation is async — does not block resource creation.
- sqlite-vec must integrate with the existing modernc/sqlite driver already in go.mod.
- All AI calls must go through the existing `ai.Manager` — no direct provider calls from services.
- Cost controls from the security hardening (daily token budget) must cover embedding calls too.

## Coordination with Change 6

- WS1 (AI classification) depends on extracted title + description from Change 6 WS1.
- WS2 (embeddings) depends on full content from Change 6 WS2/WS3.
- WS3 (semantic search) depends on embeddings being stored.
- WS4 (real deep processing) depends on all of the above.

## Workstream 1 — Real AI Classification — COMPLETE (Session 21)

Objective:
Replace the keyword heuristic classifier with real AI calls that return a category suggestion with a confidence score.

Key tasks:
- [x] Add a `ClassifyResource(ctx, content, existingCategories) (CategorySuggestion, error)` method to `ai.Manager`.
- [x] Implement the call: send title + description + existing category names to AI, parse structured response with category name + confidence score.
- [x] Update `classifier.go` to call AI manager first; fall back to keyword heuristics if AI is unavailable or returns confidence < threshold.
- [x] Apply confidence threshold logic: ≥ 0.85 → auto-save; < 0.85 → flag for review (`NeedsReview = true`).
- [x] Store `classification_confidence`, `classification_source`, and `needs_review` in `extracted_data`.
- [x] Fold classification metadata into `ResourceCreated` event payload (one-event invariant preserved).
- [x] Emit `ResourceClassified` event type (reserved for WS5 re-classification).

Deliverables:
- [x] Updated `internal/ai/manager.go` with `ClassifyResource` method + provider name stamping.
- [x] Updated `internal/service/classifier.go` with AI-first + heuristic fallback + source constants.
- [x] Updated `internal/service/resource_service.go` with threshold enforcement + `WithClassificationThreshold` option.
- [x] Updated `internal/eventstore/resource_events.go` — `ExtractedDataJSON` in `ResourceCreatedPayload`.
- [x] Updated `internal/eventstore/resource_projector.go` — projects `extracted_data` column.
- [x] `internal/service/classifier_test.go` — 6 tests for AI path, fallback, and threshold behavior.

Done criteria:
- [x] Resource classification calls AI when available and populates confidence score.
- [x] Fallback to keyword heuristics works when AI is unavailable.
- [x] Confidence threshold is enforced and configurable (default 0.85 via `classification_threshold` in config).

## Workstream 2 — Embedding Generation — COMPLETE (Session 22)

Objective:
Generate and store embedding vectors for resources using AI, enabling vector-based similarity and search.

Key tasks:
- [x] Add `GenerateEmbedding(ctx, text) (Embedding, error)` to `ai.Manager` with provider fan-out.
- [x] Add `LocalEmbeddingProvider` — deterministic feature-hashing, 256-dim, L2-normalised, offline fallback.
- [x] Add `OpenAIEmbeddingProvider` — real embeddings when API key is configured.
- [x] Wire embedding generation into the deep processing worker (`runEmbedding` step, post extraction).
- [x] Store embedding vector in `resource_embeddings` table (resource_id, vector BLOB, model_version, dim).
- [x] Track embedding model version alongside the vector for future re-embedding.
- [x] Emit `ResourceEmbedded` event type (reserved).
- [x] Token budget reservation applied to embedding calls in deep processor.

Deliverables:
- [x] `internal/ai/embedding.go` — `EmbeddingProvider` interface, `Embedding` type, `LocalEmbeddingProvider`, `CosineSimilarity`.
- [x] `internal/ai/openai_embedding_provider.go` — real OpenAI embeddings provider.
- [x] Updated `internal/ai/manager.go` — `embeddingProviders` list + `RegisterEmbedding` + `GenerateEmbedding`.
- [x] Schema migration: `resource_embeddings` table (SQLite in migration.go, Postgres 0004_resource_embeddings.sql).
- [x] `internal/service/embedding_service.go` — orchestrates generation, storage, query embedding, and search.
- [x] `internal/ai/embedding_test.go` — 7 tests for determinism, normalisation, similarity, fallback, edge cases.
- [x] `internal/service/embedding_service_test.go` — 3 tests for store, empty text error, search.

Done criteria:
- [x] Resources processed through deep tier have embedding vectors stored.
- [x] Embedding calls are tracked against the daily token budget.
- [x] Re-embedding on model version change is supported (model_version column + upsert).

## Workstream 3 — Vector Storage and Similarity Search — COMPLETE (Session 22)

> **Architecture decision (2026-06-03):** sqlite-vec is a C loadable extension and is
> incompatible with the pure-Go `modernc.org/sqlite` driver (no CGO, no extension
> loading). Switching to the CGO `mattn/go-sqlite3` driver would break the clean
> pure-Go build that keeps cross-platform Wails packaging simple. **Decision: keep
> pure-Go** — store embeddings as a BLOB in a normal `resource_embeddings` table and
> compute brute-force cosine similarity in Go. At personal-KMS scale (thousands of
> resources) this is <10ms per query and adds zero native dependencies. The
> `SearchSimilar(vector, limit, threshold)` API is identical; only the backend differs.

Objective:
Provide vector storage and similarity search in pure Go, no native dependencies.

Key tasks:
- [x] Store embeddings as a float32 BLOB in the `resource_embeddings` table (delivered in WS2).
- [x] Implement `internal/repository/sqlite/vector_repository.go` with `SearchSimilar(vector, limit, threshold)` using brute-force cosine over same-model-version vectors.
- [x] Implement Postgres parity (`internal/repository/postgres/vector_repository.go`).
- [x] Expose `EmbeddingRepository` interface in `domain/repositories.go`.

Deliverables:
- [x] `internal/domain/repositories.go` — `EmbeddingRepository` interface + `EmbeddingMatch` type.
- [x] `internal/repository/sqlite/vector_repository.go` — float32 BLOB encoding + brute-force cosine search.
- [x] `internal/repository/postgres/vector_repository.go` — BYTEA encoding + same search logic.
- [x] `internal/repository/sqlite/vector_repository_test.go` — 6 tests: upsert/get, overwrite, ordering, threshold, model-version isolation, delete.

Done criteria:
- [x] Vectors can be inserted and queried in pure Go.
- [x] `SearchSimilar` returns nearest neighbours with cosine distance scores.
- [x] Integration test: insert vectors, query nearest, assert correct descending ordering.

## Workstream 4 — Semantic Search Endpoint — COMPLETE (Session 23)

Objective:
Expose semantic search over resources via the existing search API path.

Key tasks:
- [x] Update `ResourceService.SemanticSearch` to use `EmbeddingService.SearchSimilar` (falls back to token scoring when no embeddings stored).
- [x] On query: generate embedding for the query string → vector search → fetch full resources by IDs → return ranked.
- [x] Merge semantic results with keyword results using ranking weights (semantic 0.8, keyword 1.0).
- [x] Wire into `GET /api/v1/resources/search?q=...&mode=semantic|keyword|hybrid`.
- [x] Update OpenAPI spec with `mode` enum parameter + deprecate `/semantic-search`.

Deliverables:
- [x] Updated `internal/service/resource_service.go` with `vectorSearch`, `tokenSearch`, `HybridSearch`, `mergeHybridResults`.
- [x] Updated `internal/http/handler.go` with `mode` query param routing.
- [x] Updated `api/openapi.yaml` — `mode` enum + deprecated `/semantic-search`.
- [x] `internal/service/resource_service_search_test.go` — 5 tests.

Done criteria:
- [x] `?mode=semantic` returns results ranked by vector similarity.
- [x] `?mode=hybrid` merges semantic + keyword results with configured weights.
- [x] Search query embedding is generated on the fly and not stored.

## Workstream 5 — Real Deep Processing (AI Enrichment) — COMPLETE (Session 23)

Objective:
Make deep processing actually call AI to generate a real summary, extract key points, detect entities, and update the resource — replacing the `[deep-processing] route=...` annotation stub.

Key tasks:
- [x] Add `EnrichmentProvider` interface + `OpenAIEnrichmentProvider` to ai package.
- [x] Add `RegisterEnrichment` + `EnrichResource` fan-out method to `ai.Manager`.
- [x] Add `runEnrichment` helper to `DeepProcessor` — calls manager, writes key points + entities to extracted_data, returns summary.
- [x] Update `DeepProcessor.processTask()` to call `runEnrichment` first; fall back to annotation stub when no provider configured.
- [x] Write enrichment results back to the resource: `summary`, `extracted_data.key_points`, `extracted_data.entities`.
- [x] Emit `ResourceEnriched` event type (reserved).

Deliverables:
- [x] `internal/ai/enrichment.go` — `EnrichmentProvider`, `EnrichmentInput`, `EnrichmentResult`, `OpenAIEnrichmentProvider`, `EnrichResource`.
- [x] Updated `internal/ai/manager.go` — `enrichmentProviders` + `RegisterEnrichment`.
- [x] Updated `internal/service/deep_processor.go` — `runEnrichment` replaces direct `buildDeepSummary` call.
- [x] `internal/ai/enrichment_test.go` — 6 tests (success, skip unavailable, no providers, parse valid/wrapped/no-JSON).

Done criteria:
- [x] Deep processing produces a real AI-generated summary for resources with extracted content.
- [x] `[deep-processing]` annotation stub is retained as fallback, not the primary path.
- [x] Token cost of enrichment is counted against daily budget (uses existing `reserveTokenBudget`).

## Workstream 6 — Testing, Gating, and CI — COMPLETE (Session 23)

Objective:
Ensure all WS1–WS5 deliverables are tested end-to-end and enforced in CI.

Key tasks:
- [x] Integration test: create URL resource → verify AI classification fires → verify confidence score stored.
- [x] Integration test: deep processing → verify real summary written → verify embedding stored.
- [x] Integration test: semantic search query → verify ranked results returned.
- [x] Integration test: low-confidence resource flagged NeedsReview.
- [x] Add exported `MockProvider` satisfying all three AI interfaces (no real API calls).
- [x] Export `NormalizeVector` for test helper use.
- [x] CI gate: `ai-pipeline-test` Makefile target + step in `event-sourcing-gates.yml`.

Deliverables:
- [x] `internal/ai/mock_provider.go` — `MockProvider` (ClassifySkim + GenerateEmbedding + Enrich).
- [x] `test/integration/ai_pipeline_integration_test.go` — 4 integration tests.
- [x] `ai-pipeline-test` Makefile target.
- [x] AI pipeline gate step in `.github/workflows/event-sourcing-gates.yml`.

Done criteria:
- [x] All integration tests pass with mock AI provider.
- [x] No real AI API calls made in CI.
- [x] Full `go test ./...` passes.

## Planned Milestones

- [x] Milestone 7A: AI classification live with confidence scores and fallback (WS1 complete).
- [x] Milestone 7B: Embedding generation and pure-Go vector storage working (WS2 + WS3 complete).
- [x] Milestone 7C: Semantic search endpoint live (WS4 complete).
- [x] Milestone 7D: Real deep processing producing genuine AI summaries (WS5 complete).
- [x] Milestone 7E: Full integration test suite and CI gate (WS6 complete).

## Change 7 Definition of Done

- [x] Resource classification uses real AI with configurable confidence threshold and keyword fallback.
- [x] Embedding vectors are generated and stored for all deeply processed resources.
- [x] Semantic search returns vector-ranked results via pure-Go cosine similarity.
- [x] Deep processing writes real AI-generated summaries and key points, not metadata stubs.
- [x] Mock AI provider isolates CI from real API costs.
- [x] `go test ./...` passes with no regressions.
