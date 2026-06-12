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

## Planned Milestones

- [x] Milestone 16A: Signal payload + emitter user/session scoping (WS1).
- [x] Milestone 16B: Domain types + interface updated (WS2).
- [x] Milestone 16C: SQLite migration + repository scoped queries + confidence/evidence (WS3).
- [x] Milestone 16D: Aggregator wiring + test fixes, full test suite green (WS4).
- [x] Milestone 16E: Planning docs updated (WS5).

## Change 16 Definition of Done

- [x] All GBUS signals carry `user_id` and `session_id`.
- [x] `gbus_category_features`/`gbus_resource_features` are scoped by `user_id`.
- [x] `gbus_category_features` rows carry `evidence_count`/`confidence`.
- [x] `go build ./...` and `go test ./...` pass with no regressions.
- [x] Phase 3 spec, Changes.md, and Change_10_Workstream.md updated.

## Deferred (not in scope)

- Multi-axis signal decomposition (interest/intent/utility/fatigue).
- Hourly aggregation cadence (spec section 5 currently daily-only).
- Session-pattern modeling beyond the simple per-process `session_id`.
- Embeddings-based behavioral features.
- A real `(user_id, ...)` composite-key migration — current `user_id` column + index is sufficient while `user_id` is constant `'local'`.
