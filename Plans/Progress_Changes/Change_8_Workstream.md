# Change 8 Workstream - Resource Lifecycle

Date: 2026-06-03
Status: Complete
Scope: Implement duplicate detection with counter system and the archive system with auto-archive triggers, manual archive, and restore flows.

## Objective

Deliver the full resource lifecycle management: detect duplicate resources (exact URL match and content similarity), maintain a counter for re-saves, and implement the archive system (auto-archive for dead links / expired events, manual archive, restore, and bulk operations).

## Guiding Constraints

- Change 6 (content extraction) must be complete before content-similarity duplicate detection is meaningful.
- Change 7 (embeddings) must be complete before semantic duplicate detection can work — URL-exact matching can land earlier as a fast path.
- Archive system must never permanently delete — archive is a soft state transition (30-day trash then hard delete).
- Auto-archive triggers must be configurable and opt-in (not forced on users).
- Counter increments must go through the event store (append `ResourceCounterIncremented` event).
- All lifecycle state changes must be reflected in the graph visualization.

## Workstream 1 — Duplicate Detection (Exact URL Match)

Objective:
Detect exact URL duplicates at resource creation time and increment the counter instead of creating a new record.

Key tasks:
- [x] Add `FindByURL(ctx, url) (*Resource, error)` to `ResourceRepository`.
- [x] On `ResourceService.Create`: check for existing resource with same URL before insert.
- [x] If duplicate found: increment counter, notify caller with `ErrDuplicateResource` + existing resource ID.
- [x] HTTP handler: return 200 with existing resource + `duplicate: true` flag in response body.
- [x] Emit `ResourceCounterIncremented` event on duplicate detection.

Deliverables:
- [x] Updated `internal/domain/repositories.go` with `FindByURL`, `IncrementCounter`, `ErrDuplicateResource`.
- [x] Updated `internal/repository/sqlite/resource_repository.go` with `FindByURL` + `IncrementCounter` implementation.
- [x] Updated `internal/service/resource_service.go` with duplicate check logic and `emitCounterIncrementedEvent`.
- [x] Updated `internal/http/handler.go` with duplicate response handling (200 + `duplicate:true`).
- [x] Tests for duplicate detection and counter increment (`TestDuplicateURLDetection`).

Done criteria:
- [x] Saving the same URL twice returns the existing resource with counter = 2.
- [x] Counter is stored and returned in resource reads.
- [x] `ResourceCounterIncremented` event is emitted.

## Workstream 2 — Content Similarity Duplicate Detection

Objective:
Detect near-duplicate resources (same content, different URL) using embedding similarity.

Key tasks:
- [x] After embedding generation (Change 7 WS2), run similarity check against existing vectors.
- [x] If cosine similarity > 0.92 (configurable): flag as potential duplicate.
- [x] Create a `similar_resources` link record (not a merge — user decides).
- [x] Notify user: "This resource looks similar to [existing resource]".
- [x] Emit `ResourceSimilarityDetected` event with similarity score and linked resource ID.

Deliverables:
- [x] `internal/service/duplicate_detector.go` — similarity threshold check post-embedding.
- [x] `similar_resources` join table migration.
- [x] Updated resource read response to include `similar_to` field when applicable.
- [x] Tests with fixture vectors at varying similarity scores.

Done criteria:
- [x] Near-duplicate detection fires after embedding generation.
- [x] Similarity links are stored and returned in resource reads.
- [x] No automatic merge — user sees the suggestion only.

## Workstream 3 — Archive System

Objective:
Implement the archive state for resources: manual archive, auto-archive triggers, restore, and bulk operations.

Key tasks:
- [x] Add `archived` boolean + `archive_reason` (dead_link / expired / manual) + `archived_at` timestamp to resource schema (migration).
- [x] `ResourceService.Archive(ctx, id, reason)` — soft archive, emits `ResourceArchived` event.
- [x] `ResourceService.Restore(ctx, id)` — unarchive, emits `ResourceRestored` event.
- [x] Filter archived resources out of default graph/list views.
- [x] Add `GET /api/v1/resources?archived=true` for the archive view.
- [x] Bulk archive: `POST /api/v1/resources/bulk-archive` with list of IDs.
- [x] Bulk restore: `POST /api/v1/resources/bulk-restore`.

Deliverables:
- [x] Schema migration for archive fields (`save_count`, `archived`, `archive_reason`, `archived_at`).
- [x] Updated `internal/service/resource_service.go` with `Archive` / `Restore` / `BulkArchive` / `BulkRestore` / `ListArchived`.
- [x] Updated `internal/http/handler.go` with archive endpoints and `?archived=true` query support.
- [x] Updated `api/openapi.yaml`.
- [x] Tests for archive, restore, and filter behavior (`TestArchiveAndRestore`, `TestBulkArchiveAndRestore`).

Done criteria:
- [x] Archived resources are excluded from default list/graph views.
- [x] Archive and restore are soft operations with event store entries.
- [x] Bulk operations work for lists of up to 100 IDs.

## Workstream 4 — Auto-Archive Triggers

Objective:
Automatically archive resources that match staleness criteria: dead links, expired event dates, or time-sensitive content past deadline.

Key tasks:
- [x] Implement `internal/service/archive_worker.go` — periodic background job (daily run).
- [x] Dead link check: HTTP HEAD request to resource URL; 404 / connection refused → auto-archive with reason `dead_link`.
- [x] Expired event check: if `extracted_data.event_date` < now → auto-archive with reason `expired`.
- [x] Notify user: "X resources archived today" (via notification system when available; log for now).
- [x] Configurable: auto-archive enabled/disabled per trigger type in config.

Deliverables:
- [x] `internal/service/archive_worker.go` — background staleness checker with `Start` (daily ticker) and `Run` (one cycle).
- [x] Config flags: `features.auto_archive_dead_links`, `features.auto_archive_expired_events` (both default false).
- [x] Tests with mocked HTTP client for dead link simulation (`TestAutoArchiveDeadLink`).

Done criteria:
- [x] Dead link resources are auto-archived within one daily cycle.
- [x] Expired event resources are auto-archived based on extracted event date.
- [x] Auto-archive is fully opt-in via config flags.

## Workstream 5 — Testing, Gating, and CI

Objective:
End-to-end tests for all lifecycle paths and CI enforcement.

Key tasks:
- [x] Integration test: create duplicate URL → verify counter increment → verify single resource in list.
- [x] Integration test: archive resource → verify excluded from default list → verify visible in archive view → restore → verify back in default list.
- [x] Integration test: bulk archive/restore 3 resources.
- [x] Integration test: auto-archive dead link (mocked) → verify archived reason = dead_link.
- [x] CI gate: `go test ./internal/service/... ./internal/repository/sqlite/...`.

Deliverables:
- [x] `test/integration/resource_lifecycle_integration_test.go` — 4 integration tests.
- [x] CI workflow: existing `go test ./...` gate covers all new paths.

Done criteria:
- [x] All lifecycle integration tests pass.
- [x] Full `go test ./...` passes with no regressions.

## Planned Milestones

- [x] Milestone 8A: Exact URL duplicate detection and counter live (WS1 complete).
- [x] Milestone 8B: Content similarity duplicate detection via embeddings (WS2 complete).
- [x] Milestone 8C: Manual archive and restore flows live (WS3 complete).
- [x] Milestone 8D: Auto-archive triggers running as background job (WS4 complete).
- [x] Milestone 8E: Full lifecycle integration test suite and CI gate (WS5 complete).

## Change 8 Definition of Done

- [x] Duplicate URL resources increment a counter instead of creating a new record.
- [x] Near-duplicate content is detected via embedding similarity and flagged (not auto-merged).
- [x] Resources can be manually archived and restored.
- [x] Auto-archive fires for dead links and expired events when configured.
- [x] All lifecycle state changes emit events and are covered by integration tests.
- [x] `go test ./...` passes with no regressions.
