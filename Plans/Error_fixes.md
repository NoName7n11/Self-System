# Error Fixes

Log of compile errors found across the Go workspace and how each was resolved.

## Context — why 167 errors appeared

The errors were **not** caused by `/graphify`. Graphify only reads source and writes to `graphify-out/`. Touching many files triggered a VS Code workspace re-scan, which surfaced pre-existing Go compile errors that were already on disk.

**Root cause of the errors themselves:** the `internal/service`, `internal/domain`, and `internal/config` packages on disk are a **stripped-down version**, while callers in `cmd/desktop`, `internal/sync`, `internal/desktop`, and `scripts/` still reference the **full API** that was removed. Result: `undefined`, `no field or method`, and `too many arguments` errors cascading to 167.

All 167 IDE errors trace back to a small set of missing symbols. Fixing each missing symbol clears every error that cascaded from it.

---

## Fixed

### 1. Missing domain interfaces — `internal/domain/repositories.go`

**Error:**
```
undefined: domain.EmbeddingRepository
undefined: domain.EmbeddingMatch
undefined: domain.SimilarResourceRepository
undefined: domain.GBUSFeatureStore
```
Hit in `internal/repository/sqlite/vector_repository.go`, `internal/repository/postgres/vector_repository.go`, `internal/service/embedding_service.go`, `internal/service/duplicate_detector.go`, `internal/gbus/feature_store.go`.

**Cause:** concrete repos + services referenced domain interfaces that were removed from the domain package.

**Fix:** re-added the interfaces + `EmbeddingMatch` struct to `repositories.go`, matching the method sets the existing implementations already satisfy:
- `EmbeddingMatch{ResourceID string; Score float64}`
- `EmbeddingRepository` (Upsert / Get / Delete / SearchSimilar)
- `SimilarResourceRepository` (Upsert)
- `GBUSFeatureStore` (UpsertCategoryFeature / UpsertResourceFeature / PruneOlderThan)

### 2. Missing `ResourceRepository` methods — `internal/domain/repositories.go` + sqlite impl

**Error:**
```
p.resources.UpdateExtractedData undefined (type *ResourceService ...)
w.svc.Archive undefined (type *ResourceService ...)
```
Hit in `internal/service/deep_processor.go`, `internal/service/archive_worker.go`.

**Cause:** `ResourceRepository` interface was missing `UpdateExtractedData` and `Archive`; the sqlite adapter lacked the implementations (postgres already had them).

**Fix:**
- Added `UpdateExtractedData` and `Archive` to the `ResourceRepository` interface.
- Implemented both on `internal/repository/sqlite/resource_repository.go` (JSON-marshal extracted_data; set `archived`/`archive_reason`/`archived_at` — columns already exist in the sqlite migration).

### 3. Missing `ResourceService` passthroughs — `internal/service/resource_service.go`

**Error:** `Archive` / `UpdateExtractedData` undefined on `*ResourceService`.

**Fix:** added thin passthroughs delegating to `s.resources` (the repo field).

### 4. Missing `marshalJSON` helper — `internal/service/eventsource.go`

**Error:** `undefined: marshalJSON` in `internal/service/duplicate_detector.go`.

**Cause:** helper was referenced but not present in the service package.

**Fix:** added `marshalJSON(v any) (json.RawMessage, error)` to `eventsource.go` (alongside the existing `appendWithTx` / `aggregateLatestVersion` event helpers).

---

## Fixed (continued) — Option A, API surface restored

### 5. `NewResourceService` functional options — `internal/service/resource_service.go`

Restored the variadic option API (`ResourceServiceOption`) + backing struct fields so `cmd/desktop` and `scripts/rollback_drill` compile:
- `WithEventSourcing(store, projectors)`, `WithSkimExtractor(ex)`, `WithClassificationThreshold(t)`, `WithResourceEmbeddingService(svc)`.
- `ponytail:` deferral — deps are stored but Create/Update do **not** re-emit events / run skim inline. Repo writes stay authoritative (ADR 0018: event log is audit/outbox, not the read source). Emission wiring left as a marked TODO rather than guessing double-write/projector semantics.

### 6. `CreateResourceInput.ID` + lifecycle methods — service + repos

- `CreateResourceInput.ID` field added; `Create` uses it when non-empty (sync replay cross-device identity), else generates a UUID.
- `ResourceService`: added `Restore`, `ListArchived`, `BulkArchive`, `BulkRestore`, `EventsEnabled` (reflects whether event-sourcing store is wired), `HybridSearch` (keyword ∪ semantic, deduped by ID).
- `domain.ResourceRepository` interface: added `Restore`, `ListArchived`, `BulkArchive`, `BulkRestore`.
- `internal/repository/sqlite/resource_repository.go`: implemented `Restore`, `ListArchived` (WHERE archived=1), `BulkArchive`/`BulkRestore` (per-row loop; `ponytail:` swap for `IN (...)` if bulk sizes grow).
- `domain.ErrDuplicateResource` sentinel added (handler checks `errors.Is`).

### 7. `internal/http` monolith/split duplication — `handler.go` reduced to shell

`handler.go` was the old 999-line monolith duplicating every request type, handler method, and response helper already owned by the split files (`resource_handler.go`, `category_handler.go`, …, `routes.go`, `sync_publish.go`, `response_helpers.go`). Reduced `handler.go` to its unique shell: error-code + page-limit consts, `Handler` struct, options, `NewHandler`/`NewHandlerWithOptions`, `health`, `healthDetailed`. Deleted the duplicated `RegisterRoutes`/`publishSync*`/handlers (owned by split files).

Added the symbols the newer split files reference that were never defined:
- `Handler` fields `gbusMonitor *gbus.Monitor`, `authMiddleware gin.HandlerFunc` + options `WithGBUSMonitor`, `WithAuthMiddleware`.
- `healthDetailed` handler; `defaultPageLimit`/`maxPageLimit` consts.
- `EventsEnabled()` getters on `CategoryService`, `ReminderService`, `TodoService` (return `false` — no event-sourcing wiring; HTTP publishes sync events directly).

### 8. `config` fields stripped — `internal/config/config.go`

Re-added fields `cmd/desktop` + sync require, with defaults:
- `SyncConfig.MaxConnectionsPerClient` (default 5).
- `DatabaseConfig.BackupIntervalMinutes`, `DatabaseConfig.BackupRetention`.
- `AIConfig.ClassificationThreshold`.
- `FeatureConfig.EventsResourceEnabled`.

### 9. Test stub — `internal/http/handler_test.go`

`graphResourceRepoStub` gained no-op impls for the new `ResourceRepository` methods (`UpdateExtractedData`, `Archive`, `Restore`, `ListArchived`, `BulkArchive`, `BulkRestore`).

**Result:** `go build ./...` is **green**. `go vet ./...` passes except one test package (see below).

---

## Pending — needs decision

### 10. `domain_eventsource_test.go` — Category/Todo/Reminder event-sourcing (whole stripped feature)

`go vet` surfaces `undefined: WithCategoryEventSourcing`. This is not an isolated symbol — the file has **14 tests** asserting full event-sourcing behavior (event appended + projector populates read model) on the Category, Todo, and Reminder services. That wiring was stripped from the shipping services (they have no event store; `EventsEnabled()` returns `false`). The event infra (`RegisterCategoryProjectors`, `EventTypeCategoryCreated`, …) still exists.

**Options:**
- **(A) Reconstruct** event-sourcing on all three services (options + emission + projector-populated reads) to pass the 14 tests. Substantial; risks double-write/projector-semantics bugs.
- **(B) Skip/remove the stale tests** — the shipping code deliberately has no Category/Todo/Reminder event-sourcing (docs-only pivot). `t.Skip` with a reason, or delete the file. Clears the last error without inventing a feature the code doesn't have.

**Status:** awaiting decision.

---

## Fixed (continued) — test + runtime layer

### 11. Stale ES test suites excluded — build-tagged out

Two test files assert event-sourcing emission (event append + projector-populated reads, OCC, idempotency) on services whose ES wiring was stripped. Per decision, excluded from the default build via `//go:build stale_eventsourcing` (preserved for future reconstruction, not deleted):
- `internal/service/domain_eventsource_test.go` (Category/Todo/Reminder, 14 tests).
- `internal/service/resource_service_eventsource_test.go` (ResourceService, 7 tests).

`WithEventSourcing` etc. still compile (surface restored); emission remains a documented `ponytail:` deferral. Consequence: the desktop `EventsResourceEnabled` mode is currently a no-op until emission is wired.

### 12. Test stubs + stale HTTP error-message assertions

- `graphResourceRepoStub` (both `internal/http` and `internal/service` copies) gained no-op impls for every new `ResourceRepository` method.
- ~18 handler tests asserted 500 responses **echo the raw internal error** (old monolith behavior). The canonical split handlers return a generic `"internal server error"` via `respondInternalError` (intentional — no internal-detail leak). Updated the assertions to the intended generic message. Status codes unchanged (still 500).

### 13. Classification metadata restored in `ResourceService.Create`

`Create` now writes `ExtractedData.ClassificationSource/ClassificationConfidence/NeedsReview` (manual = user/1.0/false; auto = provider source/score, `NeedsReview` when score < threshold, default 0.85). Fixes the classifier metadata tests.

### 14. sqlite read/write path reconstructed (was gutted)

`internal/repository/sqlite/resource_repository.go` was persisting/scanning only a subset of columns. Restored full round-trip (schema already had every column):
- `scanResource` + all SELECTs (`GetByID`/`List`/`Search`/`ListArchived`/`FindByURL`) now read `extracted_data` (JSON), `save_count`, `archived`, `archive_reason`, `archived_at`.
- `Create` INSERT now persists `extracted_data` + `save_count`.
- `List` filters `archived = 0` (archived resources hidden from the default view; visible via `?archived=true`).
- Added `FindByURL` + `IncrementCounter`; wired URL dedup into `ResourceService.Create` (re-save bumps `save_count`, returns existing + `ErrDuplicateResource` → handler responds 200 `duplicate:true`).
- `domain.ResourceRepository` gained `FindByURL`, `IncrementCounter`.

This fixed the remaining integration tests: `TestDuplicateURLDetection`, `TestArchiveAndRestore`, `TestBulkArchiveAndRestore`, `TestAutoArchiveDeadLink`, `TestDeepProcessor_PDFExtraction`, `TestDeepProcessor_EventDetection_CreatesReminder` (the deep-processor tests passed once `extracted_data` round-trips through the DB).

---

## Final status

- `go build ./...` — **green**
- `go vet ./...` — **green**
- `go test ./...` (unit + integration) — **all pass**

Known deliberate gap: service-layer event-sourcing **emission** is not reconstructed (surface only). Desktop `EventsResourceEnabled` mode is a no-op until wired; ES test suites are tagged out under `stale_eventsourcing`.

---

## Superseded

### `scripts/rollback_drill/main.go:204` — `WithEventSourcing` undefined + too many args

**Error:**
```
too many arguments in call to service.NewResourceService
    have (domain.ResourceRepository, domain.CategoryRepository, nil, *service.CategoryService, unknown type)
    want (domain.ResourceRepository, domain.CategoryRepository, *service.CategoryClassifier, *service.CategoryService)
undefined: service.WithEventSourcing
```

**Cause:** `NewResourceService` on disk is the minimal 4-arg form with **no functional options**. But `scripts/rollback_drill/main.go` and `cmd/desktop/main.go` both call it with functional options:
- `WithEventSourcing(store, registry)`
- `WithSkimExtractor(...)`
- `WithClassificationThreshold(...)`
- `WithResourceEmbeddingService(...)`

None of these options exist anymore — they were stripped along with the event-sourcing / skim / embedding wiring inside `Create`/`Update`/`Delete`.

**Fix options:**
- **(A) Reconstruct the full API** — re-add the functional options + backing struct fields + the event-emission / skim / embedding behavior in `Create`/`Update`/`Delete`. Large; some original behavior would be inferred, not recovered from git (code is untracked in this docs-only repo). Required if the desktop app + rollback drill must run.
- **(B) Treat desktop/scripts as stale** — the primary target `cmd/server` uses the minimal 4-arg form. Trim `scripts/rollback_drill` + `cmd/desktop` to match, or exclude them from the build.

**Resolved:** chose Option A — see items 5–9 above. `WithEventSourcing` and the other options are restored (surface-only, emission deferred), so `rollback_drill:204` and `cmd/desktop` compile.
