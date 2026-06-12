# Change 16 Workstream - GBUS Schema Alignment: User-Scoped Signals & Confidence Tracking

Date: 2026-06-13
Status: Complete

Scope: Close the gap between the Phase 3 GBUS spec (`Plans/Progress/Phase_3_GBUS_Signals_Feature_Store.md`) and the `internal/gbus` scaffold implementation: add `user_id`/`session_id` to signals and feature rows (spec required these from the start), and add `evidence_count`/`confidence` to category features.

## Objective

Make GBUS signals and feature rows user-scoped (future-proofing for Phase 2+ multi-user/sync without a painful migration), and give inference/training a cheap, built-in way to discount category affinities backed by little evidence — without taking on the larger deferred items (multi-axis signal decomposition, hourly aggregation, session-pattern modeling, embeddings).

## Workstream 1 — Signal Payload & Emitter

Key tasks:
- [x] Add `UserID`/`SessionID` fields to `GBUSSignalPayload` (`internal/gbus/signals.go`).
- [x] Add `DefaultUserID = "local"`, `ExplicitIntentWeightThreshold = 0.5`, `ConfidenceEvidenceThreshold = 10` constants.
- [x] `SignalEmitter` generates a per-process `sessionID` (UUID) in `NewSignalEmitter`.
- [x] `Emit()` defaults `payload.UserID`/`payload.SessionID` when empty.

Deliverables:
- [x] `internal/gbus/signals.go`, `internal/gbus/emitter.go` updated.

## Workstream 2 — Domain Types & Feature Store Interface

Key tasks:
- [x] Add `UserID` to `domain.GBUSCategoryFeature` / `domain.GBUSResourceFeature`.
- [x] Add `EvidenceCount`/`Confidence` to `domain.GBUSCategoryFeature`.
- [x] Move `ExplicitIntentWeightThreshold`/`ConfidenceEvidenceThreshold` constants to `internal/domain` (aliased from `internal/gbus`) to avoid an import cycle (sqlite repo -> gbus -> eventstore -> sqlite repo).
- [x] Update `domain.GBUSFeatureStore` interface: `UpsertCategoryFeature`, `GetCategoryFeatures`, `UpsertResourceFeature`, `GetResourceFeatures` all take a leading `userID string`.

Deliverables:
- [x] `internal/domain/entities.go`, `internal/domain/repositories.go` updated.

## Workstream 3 — SQLite Repository & Migration

Key tasks:
- [x] Migration v4 (`gbus_user_scoped_features`): add `user_id` (default `'local'`) to both feature tables, `evidence_count`/`confidence` to `gbus_category_features`, plus non-unique indexes on `user_id`. Existing `(category_id|resource_id, signal_type)` primary keys kept as-is.
- [x] `gbus_repository.go`: `UpsertCategoryFeature` increments `evidence_count` when `weight >= ExplicitIntentWeightThreshold` and recomputes `confidence = MIN(1.0, evidence_count / ConfidenceEvidenceThreshold)` in the SQL upsert. All four methods scoped by `user_id`.

Deliverables:
- [x] `internal/repository/sqlite/migration.go`, `internal/repository/sqlite/gbus_repository.go` updated.

## Workstream 4 — Aggregator & Tests

Key tasks:
- [x] `aggregator.go`: default `userID` from `payload.UserID` (fallback `DefaultUserID`), pass to both upsert calls.
- [x] Fix `inMemoryFeatureStore` mock in `aggregator_test.go` to match new interface signatures (leading `userID`, filter by `UserID`).
- [x] `go build ./...` and `go test ./...` pass with no regressions.

Deliverables:
- [x] `internal/gbus/aggregator.go`, `internal/gbus/aggregator_test.go` updated.

## Workstream 5 — Documentation

Key tasks:
- [x] Add "Implementation Gap Note" (section 9) and confidence/evidence decision (section 8) to `Plans/Progress/Phase_3_GBUS_Signals_Feature_Store.md`.
- [x] Add Change 16 section to `Plans/Progress_Changes/Changes.md`.
- [x] Cross-reference Change 16 from `Plans/Progress_Changes/Change_10_Workstream.md`.

## Workstream 6 — Make Confidence Behavioral (Training Consumption)

Added in response to review: `confidence`/`evidence_count` were captured but had no
consumer (runtime inference reads the trained artifact's `CategoryWeights`, not the
feature tables; the training script ignored the new columns). This wires the
evidence signal into the one place it belongs — the training pipeline that produces
those weights.

Key tasks:
- [x] `scripts/gbus_train/main.go` `listAll`: SELECT/scan `evidence_count` and `confidence`.
- [x] `computeCategoryWeights`: aggregate `evidence_count` per category and discount the
      category's raw score by `MIN(1, evidence / ConfidenceEvidenceThreshold)` before
      normalization. Discount applied at category level (not per row) because passive
      signal types structurally carry per-row confidence 0.
- [x] `go build ./...` / `go vet ./scripts/gbus_train/` clean.

Known limitation:
- [ ] Effect is latent — not exercised end-to-end, since no real training run has
      occurred yet (Change 10 WS3 remains pending real signal data + Docker/DB).

## Planned Milestones

- [x] Milestone 16A: Signal payload + emitter user/session scoping (WS1).
- [x] Milestone 16B: Domain types + interface updated (WS2).
- [x] Milestone 16C: SQLite migration + repository scoped queries + confidence/evidence (WS3).
- [x] Milestone 16D: Aggregator wiring + test fixes, full test suite green (WS4).
- [x] Milestone 16E: Planning docs updated (WS5).
- [x] Milestone 16F: Confidence wired into training-time category weighting (WS6).

## Change 16 Definition of Done

- [x] All GBUS signals carry `user_id` and `session_id`.
- [x] `gbus_category_features`/`gbus_resource_features` are scoped by `user_id`.
- [x] `gbus_category_features` rows carry `evidence_count`/`confidence`.
- [x] `go build ./...` and `go test ./...` pass with no regressions.
- [x] Phase 3 spec, Changes.md, and Change_10_Workstream.md updated.

## Explicit Caveats (review-confirmed scope limits)

- **Feature tables are NOT yet multi-user isolation-safe.** `user_id` plumbing exists end-to-end (signal payload → emitter → aggregator → repository columns), but the SQLite primary keys remain `(category_id, signal_type)` and `(resource_id, signal_type)`, and upserts use `ON CONFLICT(category_id, signal_type)` / `ON CONFLICT(resource_id, signal_type)` then overwrite `user_id = excluded.user_id`. Consequence: if two different users ever write the same `category+signal` (or `resource+signal`), the second write **merges into the first user's row** rather than creating an isolated row — last-writer-wins on `user_id`. This is safe today only because `user_id` is constant `'local'`. True isolation requires the composite-key rebuild below.
- **`confidence`/`evidence_count` are training-time inputs only, not runtime inference inputs.** `computeCategoryWeights` (WS6) bakes the confidence discount into the trained `CategoryWeights`; `internal/gbus/inference.go` reads those weights at request time and never reads `confidence` itself. By design — there is no per-request confidence path.
- **Training still produces a single global (cross-user) category model.** `scripts/gbus_train/main.go`'s `listAll` does not select `user_id`, and `computeCategoryWeights` aggregates by `CategoryID` only — so even after storage isolation (composite-key rebuild) is fixed, the first real multi-user rollout would still train one shared cross-user profile, conflicting with the Phase 3 target of *user-category* affinity. Closing this requires `listAll` to carry `user_id` and `computeCategoryWeights`/`GBUSModel.CategoryWeights` to key by `(user_id, category_id)`. Deferred alongside the storage rebuild; harmless today since `user_id` is constant `'local'`.

## Deferred (not in scope)

- Multi-axis signal decomposition (interest/intent/utility/fatigue).
- Hourly aggregation cadence (spec section 5 currently daily-only).
- Session-pattern modeling beyond the simple per-process `session_id`.
- Embeddings-based behavioral features.
- A real `(user_id, ...)` composite-key migration (PK rebuild) — required before multi-user rollout for actual row isolation; current `user_id` column + index is sufficient only while `user_id` is constant `'local'` (see caveat above).
