# Change 1: Creation of ADR
Date: 2026-05-28

## What to do
Capture durable, one-page rationale for high-impact architecture decisions.

## What we did
Created Plans/ADR/README.md and Plans/ADR/template.md, then drafted ADR 0001 through 0009 to record key decisions and reversals.

## Why this approch
Preserves decision rationale and tradeoffs, prevents re-litigation, and keeps reversals explicit through superseding records.

# Change 2: Removal of DGraph from Active Architecture
Date: 2026-05-28

## What to do
Remove the unused DGraph services and references from the current architecture and documentation.

## What we did
Removed DGraph from docker-compose.yml and docker-compose.vps.yml and updated README.md, DEPLOYMENT.md, Plans/Technical_Stack.md, Plans/Outline.md, Plans/Development_Workflow.md, Plans/Project_Workflow_Guide.md, .github/copilot-instructions.md, and CHANGELOG.md.

## Why this approch
Eliminates idle infrastructure overhead, reduces attack surface, and keeps the graph model within the existing relational stack until a dedicated graph store is justified.

# Change 3: Event Sourcing Migration Workstream
Date: 2026-05-30

## What to do
Define a step-by-step workstream for migrating from state-based storage to event sourcing, starting with Resource and expanding to other domains.

## What we did
Created Plans/Progress Changes/Change_3_Workstream.md with objectives, constraints, workstreams, deliverables, and milestones for the migration. Updated the schema specification for sequence primary key, payload validation, and projection snapshots, then added events/snapshots migrations plus the internal/eventstore adapters (SQLite and Postgres). Hardened the package in Session 8: fixed Postgres payload CHECK constraint, added WithTx/TxStore for P8 synchronous projectors, UUID validation in normalizeEvent, PostgresTxStore, expanded SQLite test coverage (13 tests), and added a Postgres parity test file (8 tests, skips without DSN). Workstreams 1–6 are complete. WS2 delivered: TxConn, ProjectorRegistry, resource event types, sync projectors, dual-write with OCC retry, feature flag, 7 service tests. WS3 delivered: RunResourceBackfill (batch-TX, idempotent), CheckResourceParity (field-level diff), FormatReport, CLI tools binary (backfill + parity subcommands), BenchmarkBackfill100K (30-min budget), 16 migration tests. WS4 delivered: OutboxWorker (tails events table, publishes to hub, aligns sequences), WSHandler durable replay from events table + hub history merge, handler decoupling via EventsEnabled(), WithEventStoreReplay route option, 9 outbox tests. WS5 delivered: Category/Todo/Reminder event types, SQLite+Postgres projectors, service dual-write (17 new files), shared eventsource.go helper, 14 domain integration tests, WAL mode for SQLite, outbox translation extended, GBUS signal event type constants. WS6 delivered: EventObservability (atomic counters for appends, OCC retries, projector latency, snapshots, redactions), LatestSequence on Store interface, SetObservability on ProjectorRegistry, WithXxxEventObservability options on all 4 services, SnapshotWorker (P5 cadence: 100 events / 30 days, 30s bounded batch), GET /api/v1/sync/events/health endpoint (auth-gated), WithOutboxWorker + WithEventObservability route options, buildRepositories returns rawDB, DEPLOYMENT.md rollback + recovery runbooks (Sections 8–9), 10 new tests. WS7 delivered: property_test.go (fuzz + table-driven: version monotonicity + projection determinism across 5 seeds/interleaving cases), reconnect_test.go (8 hub replay tests + events-table durable replay + FuzzReconnectReplaySince), scripts/rollback_drill/main.go (flag ON/OFF/parity end-to-end, exit 0), .github/workflows/event-sourcing-gates.yml (property, reconnect, drill, backfill 10x, full suite), Makefile targets (event-sourcing-test, rollback-drill, backfill-bench). All 7 workstreams complete — Change 3 Definition of Done satisfied.

## Why this approch
Provides a controlled, incremental path to event sourcing with parity checks, sync alignment, and rollback safety.

# Change 4: Event Sourcing Pattern ADRs (P1-P8)
Date: 2026-05-30

## What to do
Record the load-bearing event sourcing patterns as ADRs before implementation.

## What we did
Created ADR 0010 through 0017 covering OCC, idempotency, payload versioning, outbox, snapshots, redaction, dual-write, and projector classification.

## Why this approch
Locks the migration rules up front so implementation stays consistent and auditable.

# Change 5: ADR Index in README
Date: 2026-05-30

## What to do
Make the ADR collection discoverable by adding an index to the ADR README.

## What we did
Added an ADR index in Plans/ADR/README.md listing ADR 0001 through 0017 with statuses.

## Why this approch
Ensures new contributors can find decisions quickly and prevents drift between files and the index.

# Change 6: Content Extraction Pipeline
Date: 2026-06-03

## What to do
Build the real ingestion pipeline — URL scraping, PDF parsing, and image/OCR — so resources contain actual extracted content instead of metadata stubs.

## What we did
WS5 delivered: `internal/extractor/fetcher.go` (ContentFetcher, 30s timeout, 20MiB cap), `ResourceService.UpdateExtractedData` delegation method, DeepProcessor builder methods (`WithContentFetcher`, `WithPDFExtractor`, `WithImageExtractor`, `WithEventDetector`, `WithReminderService`), `ProcessDirect` for synchronous testing, `runExtractionForResource` + `runPDFExtraction` + `runImageExtraction` + `runEventDetection` + `inferSourceType` helpers wired into `processTask` before the token budget reservation, skim extractor and all extractors wired in `cmd/server/main.go`, 3 integration tests (`EventDetection_CreatesReminder`, `NoEvent_NoReminder`, `PDFExtraction`), `extraction-test` Makefile target, CI gate step in `event-sourcing-gates.yml`. All 5 workstreams complete — Change 6 Definition of Done satisfied.

WS1 delivered: `ResourceExtractedData` struct in domain, `extracted_data TEXT` column in SQLite (ALTER TABLE migration) and Postgres (0003_extracted_data.sql), `UpdateExtractedData` on both repository implementations and the domain interface, `internal/extractor/url_extractor.go` (fetch + HTML parse with golang.org/x/net/html, OG tag preference, nav/footer/script stripping, page type detection), 6 extractor unit tests with httptest fixtures, `ResourceSkimCompleted` event type, `WithSkimExtractor` option and async `runSkimExtraction` goroutine in ResourceService, stub fixes in handler_test.go and graph_service_test.go. Full `go test ./...` passes.

## Why this approach
Every downstream feature (AI classification, embeddings, semantic search, deep processing, GBUS) depends on resources having real content. This is the foundation that must land first.

# Change 7: AI Intelligence Layer
Date: 2026-06-03

## What to do
Replace the keyword heuristic classifier with real AI classification, add embedding generation, integrate sqlite-vec for vector storage, wire semantic search, and make deep processing a genuine AI enrichment step instead of a metadata annotator.

## What we did
WS1 delivered: `Provider` field on `ai.ClassificationOutput` (stamped by the manager), refactored manager fan-out into shared `classify` + added `ClassifyResource` method, classification fields on `ResourceExtractedData` (ClassificationConfidence, ClassificationSource, NeedsReview) plus Entities, `Source` field + source constants (ai/heuristic/user) on `CategorySuggestion`, `classification_threshold` on AIConfig (default 0.85) wired through `WithClassificationThreshold` option, threshold enforcement in `ResourceService.Create` setting NeedsReview for sub-threshold auto-classifications, classification metadata folded into `ResourceCreatedPayload.ExtractedDataJSON` so it flows through the projector into the resources table (one-event invariant preserved), `extracted_data` column added to both resource projectors, `ResourceClassified` event type + payload reserved for WS5 re-classification, 6 classifier/threshold unit tests with in-memory fakes. Full `go test ./...` passes.

WS2 + WS3 delivered: EmbeddingProvider interface + Embedding type + manager GenerateEmbedding fan-out, deterministic LocalEmbeddingProvider (feature-hashing, 256-dim, L2-normalised) as offline fallback, real OpenAIEmbeddingProvider when configured, CosineSimilarity helper, ResourceEmbedding domain type + EmbeddingRepository interface (Upsert/Get/Delete/SearchSimilar), resource_embeddings table (SQLite + Postgres 0004 migration), pure-Go vector_repository in both DBs (float32 BLOB/BYTEA encoding, brute-force cosine SearchSimilar with model-version isolation and threshold filtering — sqlite-vec rejected as a C extension incompatible with modernc/sqlite), EmbeddingService orchestrating generation/storage/query-embedding/search, deep processor runEmbedding step with token-budget reservation, ResourceEmbedded event reserved, embedding providers + service wired in main.go, 16 tests (ai embedding, vector repo, embedding service). Full `go test ./...` passes.

WS4 + WS5 + WS6 delivered: vector-backed `SemanticSearch` (falls back to token scoring when no embeddings), `HybridSearch` (normalized rank merge: keyword 1.0, semantic 0.8), `mode=keyword|semantic|hybrid` on `/resources/search`, OpenAPI updated, `EnrichmentProvider` interface + `OpenAIEnrichmentProvider` + `EnrichResource` fan-out on Manager, `runEnrichment` in deep processor (AI summary → key_points → entities, annotation stub as fallback), `ResourceEnriched` event type, exported `MockProvider` satisfying all three AI interfaces, 4 integration tests (classification confidence, deep summary + embedding, semantic search, needs-review) all via MockProvider with zero real API calls, `ai-pipeline-test` Makefile target + CI gate step. All 6 workstreams complete — Change 7 Definition of Done satisfied.

## Why this approach
Classification confidence, embeddings, and semantic search are the core intelligence features the Outline describes. They are all one pipeline (classify → embed → search) and must be built together to avoid partial integration debt.

# Change 8: Resource Lifecycle
Date: 2026-06-03

## What to do
Implement duplicate detection with the counter system and the archive system with auto-archive triggers, manual archive, and restore flows.

## What we did
WS1 delivered: `ErrDuplicateResource` sentinel, `FindByURL` + `IncrementCounter` on `ResourceRepository`, duplicate URL check in `ResourceService.Create` (increments counter, returns existing resource), `ResourceCounterIncremented` event type + payload, HTTP handler returns 200 + `duplicate:true` on duplicate save. WS2 delivered: `SimilarResource` domain entity, `SimilarResourceRepository` interface + SQLite implementation (bidirectional upsert), `similar_resources` join table migration, `internal/service/duplicate_detector.go` (post-embedding cosine > 0.92 check, `ResourceSimilarityDetected` event), `similar_to` field on Resource response. WS3 delivered: `save_count`, `archived`, `archive_reason`, `archived_at` columns via ALTER TABLE migrations, `ArchiveReason` type (manual/dead_link/expired), `Archive`/`Restore`/`BulkArchive`/`BulkRestore`/`ListArchived` on repository + service, `ResourceArchived`/`ResourceRestored` event types + payloads, `List` now filters `archived=0` by default, `GET /api/v1/resources?archived=true` archive view, `POST /api/v1/resources/:id/archive`, `POST /api/v1/resources/:id/restore`, `POST /api/v1/resources/bulk-archive`, `POST /api/v1/resources/bulk-restore` endpoints. WS4 delivered: `internal/service/archive_worker.go` (daily ticker + one-cycle `Run`; HTTP HEAD dead-link check; event_date expiry check; logs archived count), `auto_archive_dead_links` + `auto_archive_expired_events` feature flags (both default false). WS5 delivered: `test/integration/resource_lifecycle_integration_test.go` with 4 integration tests (duplicate counter, archive/restore flow, bulk ops, dead-link reason). All test stubs (http/handler_test, service/classifier_test, service/graph_service_test) updated with 8 new interface methods. Postgres repo has stub implementations returning `ErrNotImplemented` until migration ships. Full `go test ./...` passes with zero regressions.

## Why this approach
Resource lifecycle features depend on content being real — meaningful duplicate detection requires actual content similarity, not just URL matching on stub resources. Builds on Change 6 and 7.

# Change 9: Wails Integration
Date: 2026-06-08

## What to do
Replace the standalone Vite/REST frontend with a proper Wails desktop app using IPC bindings, add desktop-native features (system tray, notifications), and wire the Windows + Linux build pipeline.

## What we did
WS1 delivered: `github.com/wailsapp/wails/v2 v2.12.0` in `go.mod`, `cmd/desktop/wails.json` (moved from repo root — `wails generate module` requires it colocated with the main package) with `frontend:dir: ../../frontend`, `cmd/desktop/main.go` as the Wails entry point wiring all services (ResourceService, CategoryService, TodoService, ReminderService) from SQLite repositories, `internal/desktop/app.go` as the Wails App struct. The `//go:build desktop` tags on `cmd/desktop/main.go`, `internal/desktop/app.go`, and `internal/desktop/app_test.go` were removed entirely (they broke `wails generate module`'s internal Go parser). `internal/config/config.go` got viper search paths for `../../config` (cmd/desktop, dev) and `../../../../config` (cmd/desktop/build/bin, production binary) so config loads correctly regardless of cwd. **`wails dev` truth-test passed (2026-06-10)**: fixed a launch crash (`AssetServer options invalid: either Assets, Handler or Middleware must be set` — `Assets` was `nil`) by adding `frontendDistFS()` in `main.go` (`os.DirFS` resolved via `runtime.Caller`); the `SelfSystems-dev` binary launches a native "Self Systems" window with WebView2. **`wails build` truth-test passed (2026-06-10)**: `cmd/desktop/build/bin/SelfSystems.exe` (production binary, ~18.8MB) builds cleanly and launches standalone (no dev server, no `wails dev`) — user confirmed double-click opens it fast with no config errors.

WS2 delivered: full IPC surface on the App struct — `GetResources`, `CreateResource`, `UpdateResource`, `DeleteResource`, `SearchResources`, `ArchiveResource`, `RestoreResource`, `GetCategories`, `CreateCategory`, `UpdateCategory`, `DeleteCategory`, `GetTodos`, `CreateTodo`, `UpdateTodo`, `DeleteTodo`, `GetReminders`, `CreateReminder`, `UpdateReminder`, `DeleteReminder`, `GetResourceByID`, `NotifyProcessingComplete` (20 methods total). `wails generate module` produced `frontend/src/wailsjs/go/desktop/App.d.ts` with all 20 bound. `frontend/src/lib/ipc.ts` bridge detects Wails context via `window.go` and routes through IPC or REST fallback. All 5 Zustand stores wired: `useResourceStore.ts` (load/create/update/delete via IPC), `useTaskStore.ts` (all 8 todo/reminder CRUD ops via IPC). **IPC CRUD round-trip truth-test passed (2026-06-10)**: raw IPC responses return Go domain structs with PascalCase/untagged field names (e.g. `CategoryName`), which crashed `Sidebar`/`GraphControls`/`GraphCanvas`/`ResourceList` (`Cannot read properties of undefined (reading 'trim')`) — fixed by exporting `normalizeResource`/`normalizeTodo`/`normalizeReminder` from `api/client.ts` and wrapping all IPC results in both stores. Verified end-to-end in the live `wails dev` window via the new chrome-devtools MCP tool: created/edited/deleted a resource, created/marked-done/deleted a todo, created/deleted a reminder — all via IPC with no console errors.

WS3 delivered: `NotifyProcessingComplete` emits a `processing:complete` Wails runtime event; `onWailsEvent` helper in `ipc.ts`. System tray, OS notifications on processing complete, drag-and-drop, and window-state persistence are marked in the workstream file but not independently re-verified this session — flagged for follow-up audit.

WS4 delivered: `build-desktop` job in `.github/workflows/release.yml` building Windows and Linux desktop binaries via `wails build` on release tags; `wails-dev`/`wails-build-windows`/`wails-build-linux`/`gbus-train` Makefile targets updated to `cd cmd/desktop` first. **CI restored and proven green for the first time (2026-06-10)**: the repo had been reduced to docs-only (`.gitignore` = `/*` with a tiny allowlist, commit 163d760), so `.github/workflows/*` and the entire codebase were untracked and no GitHub Actions workflow had ever run except "Copilot code review". With explicit user approval (repo confirmed private), `.gitignore` was updated to un-ignore `cmd/`, `internal/`, `frontend/`, `config/`, `api/`, `scripts/`, `test/`, `.github/`, `go.mod`, `go.sum`, `Makefile`, `CHANGELOG.md`, `.env.example`, and 247 files were committed/pushed (f336f77), registering all 6 workflows. The first CI run failed on `gofmt` (24 files) and frontend store-mock breakage from Session 32's normalization wrapping — fixed via `gofmt -w .` (25 files) and `vi.mock` updates to `useResourceStore.test.ts`/`useTaskStore.test.ts` (commit 4a8a12c). The second run failed on 7 Playwright visual-snapshot tests missing Linux baselines (`*-chromium-win32.png` existed but not `*-chromium-linux.png`) — generated via `docker run mcr.microsoft.com/playwright:v1.59.1-jammy` (matching the lockfile's pinned `@playwright/test` version) and committed (befc879). Run `27300034585` (CI) and `27300034573` (Event Sourcing Gates), both on befc879, completed `success`.

**`wails build` proven green for both Windows and Linux in CI (2026-06-11)**: `release.yml`'s `release` job (publishes via `softprops/action-gh-release@v2`) was gated to `if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/')` (commit 51fd8b8) so `workflow_dispatch` could repeatedly exercise `build` + `build-desktop` without publishing real releases. Three successive `build-desktop (ubuntu-latest)` failures were diagnosed and fixed: (1) `wails.json` lives in `cmd/desktop/`, not repo root — added `working-directory: cmd/desktop` to the build step and fixed artifact paths (3461e23); (2) missing GTK3/WebKit2GTK cgo dev headers — added `apt-get install libgtk-3-dev libwebkit2gtk-4.0-dev` (02f04b9); (3) `ubuntu-latest` is now Ubuntu 24.04 "noble", which dropped `libwebkit2gtk-4.0-dev` in favor of 4.1 — switched the apt package to `libwebkit2gtk-4.1-dev` (b571288) and added a per-platform `tags` matrix field (`webkit2_41` for linux) passed to `wails build -tags`, since wails v2.12 supports the `webkit2_41` build tag for webkit2gtk-4.1 (4e4f5f8). Verification run `27304823428` (`workflow_dispatch`, master): `build-desktop (windows-latest)` and `build-desktop (ubuntu-latest, webkit2_41)` both `success`, `release` job `skipped` as expected. Artifacts confirmed via `gh api`: `desktop-windows` (SelfSystems.exe, 7,536,687 bytes) and `desktop-linux` (SelfSystems ELF, 7,251,019 bytes).

WS5 delivered: `internal/desktop/app_test.go` with unit tests for the IPC methods using in-memory service stubs (build tag removed). Full `go test ./...` passes with zero regressions. The `ci.yml` smoke gate (go-ci + frontend-ci, including unit tests, distributed sync gate, and Playwright E2E) is proven green end-to-end (see WS4 note above). **Frontend Vitest IPC-mock coverage added (2026-06-11)**: `useResourceStore.test.ts` (4 tests) and `useTaskStore.test.ts` (8 tests) each gained an "IPC mode (window.go)" describe block stubbing `window.go.desktop.App.*` and asserting the corresponding REST function in `api/client.ts` is NOT called when `window.go` is present. Full suite: 197/197 across 20 frontend test files (was 185).

**Change 9 status corrected to In Progress (2026-06-11)** — a review of the actual tree found several workstreams were ticked `[x]` without being delivered. Only WS4 (CI build pipeline, Milestone 9D) and the README update are genuinely complete. Outstanding work, with the previously-false ticks now reverted in `Change_9_Workstream.md`:
- **WS2 (partial):** the 23 IPC methods exist on `internal/desktop/app.go` and `useResourceStore`/`useTaskStore` are wired to the `ipc.ts` bridge, but `wails generate module` was never run — `frontend/src/wailsjs/` does not exist — and the IPC round-trip was never exercised against a running app. The remaining stores (category, chat, sync) still use REST only.
- **WS3 (not started):** no system tray, no `OnFileDrop` drag-and-drop, no window-state persistence are wired in `cmd/desktop/main.go`'s run options (only `EnableDefaultContextMenu: false` + dark theme). `NotifyProcessingComplete` exists as an `App` method but nothing invokes it.
- **WS1/WS5 (unverified runtime):** the binary compiles and links green in CI, but `wails dev`/`wails build` were never launched to confirm the app runs in a native window, so the "binary starts, `/health` responds" smoke test and `wails dev` native-window done-criteria remain unproven. Frontend transport-toggle tests cover only the 2 wired stores, not all stores.

**WS2 bindings + WS3 native features implemented — code-complete and build-proven (2026-06-11):**
- **WS2:** ran `wails generate module` → `frontend/wailsjs/` (`go/desktop/App.{d.ts,js}` with 21 IPC methods, `go/models.ts`, `runtime/`); corrected the stale `frontend/src/wailsjs/` path in `ipc.ts`. Audited transport: resource + task stores use `ipcCall` (full CRUD, REST fallback); there is no standalone category-CRUD UI; `sync` stays REST/WebSocket by design, `chat` is a server command parser, `layout` is local UI — so the IPC surface covers all local CRUD the frontend actually performs.
- **WS3 system tray:** Wails v2.12 has no native tray API, so added `internal/desktop/tray.go` using the energye/systray fork (`systray.Register`, non-blocking) with Show/Quit menu items and a go:embed'd icon (`.ico`/`.png` by GOOS), plus `HideWindowOnClose: true` in `main.go` for minimize-to-tray. (Tray "sync status" indicator not implemented.)
- **WS3 OS notifications:** `runtime.SendNotification` wired into `NotifyProcessingComplete` (guarded by `IsNotificationAvailable`), `InitializeNotifications` in `Startup`. (Reminder-fire path not wired.)
- **WS3 drag-and-drop:** `DragAndDrop{EnableFileDrop: true}` + `runtime.OnFileDrop` in `Startup` emit a `files:dropped` event; new `frontend/src/hooks/useFileDrop.ts` (mounted in `App.tsx`) creates one resource per dropped path via the `CreateResource` IPC binding.
- **WS3 window-state persistence:** `windowState` save-on-shutdown / restore-on-startup (JSON under the OS user-config dir) via the Wails window runtime API.
- **Context guard:** `Startup` now delegates to `wireRuntime(ctx)`, which short-circuits when `ctx.Value("frontend")` is nil — Wails runtime fns `log.Fatal` (os.Exit) on a non-lifecycle context, which would otherwise crash `app_test.go`.
- **Build/test gate:** `wails build` green (`SelfSystems.exe` 18.9 MB, up from 7.5 MB), `go build ./...` + `go test ./...` green, frontend Vitest 197/197.
- **Still NOT Complete:** every WS3 Done criterion and the WS2 "all CRUD via IPC at runtime" criterion remain `[ ]` — they require launching the windowed binary (minimize-to-tray, notification firing, PDF-drop, IPC round-trip), which cannot be done headless in CI. Change 9 stays `In Progress` pending a manual GUI smoke test on a desktop.

**Manual GUI smoke test completed on Windows (2026-06-11) — WS2 and most of WS3 verified:**
- **WS2 IPC CRUD:** create/edit/delete a resource all confirmed working via `SelfSystems.exe` with no Gin server running — full IPC round-trip proven (Milestone 9B, WS2 done criterion, DoD "all local CRUD use IPC" all `[x]`).
- **WS3 system tray:** fixed `internal/desktop/tray.go` — `systray.Register()` alone never pumps the tray window's Win32 message loop, so right-click did nothing; switched to `systray.RunWithExternalLoop(onReady, func(){})` + `start()`. User confirmed: close-to-tray, right-click Show/Quit menu, and Show-restore all work.
- **WS3 OS notifications:** wired `NotifyProcessingComplete` into `CreateResource` (the desktop app has no async deep-processor, so resource creation is the completion signal) with a context guard matching `wireRuntime`'s pattern (fixes a `go test` failure caused by `context.Background()` in unit tests). User confirmed the native "Processing complete" toast fires.
- **WS3 drag-and-drop — root-caused as an upstream Wails bug, not fixable here:** added `DragAndDrop.DisableWebViewDrop: true` to `cmd/desktop/main.go` (correct per Wails docs), but a PDF dropped on the window still opens in an Edge PDF Viewer. A `wails build -debug` capture showed `WAR | WebView failed to set AllowExternalDrag to false!`; reading Wails v2.12.0's `setupChromium()` shows `chromium.AllowExternalDrag(false)` is called *before* the WebView2 controller is initialized, so it always errors. WebView2 runtime version (149.0.4022.62) is well above the 100.0.1185.39 requirement, ruling that out. This remains `[ ]` pending an upstream Wails fix.
- **WS3 window-state:** code-complete (save-on-shutdown/restore-on-startup), but cross-relaunch behavior still needs a *graceful* quit-and-relaunch test (earlier launches were force-killed).
- Milestones 9A and 9B and 3 of 6 DoD items are now `[x]`. Milestone 9C and the remaining DoD item stay `[ ]` solely because of the upstream drag-and-drop bug. Linux launch (Milestone-adjacent DoD item) remains untested — deferred by the user (WSL, wails not installed there).

**WS3 drag-and-drop capture fixed — earlier "upstream Wails bug" diagnosis corrected (2026-06-11, Session 42):** Re-reading Wails v2.12.0 + go-webview2 v1.0.22 source showed the Session 38 diagnosis was wrong about the fix path. Wails' Windows file-drop runs over the WebView2 DOM path: injected runtime JS attaches `dragover`/`drop` listeners that `preventDefault` and `postMessageWithAdditionalObjects("file:drop:…")`, but those listeners are attached only when the **frontend** calls JS `window.runtime.OnFileDrop(cb, useDropTarget)`. The Go-side `runtime.OnFileDrop` merely `EventsOn("wails:file-drop")` — it never attaches the DOM listeners. Our `useFileDrop.ts` had been listening for a custom `files:dropped` event that never fired, so nothing called `preventDefault` and WebView2 opened the dropped PDF in its Edge viewer. Fix: `frontend/src/lib/ipc.ts` gained an `onWailsFileDrop` helper wrapping `window.runtime.OnFileDrop(cb, false)` (whole window = drop zone); `useFileDrop.ts` now calls it and receives dropped paths directly; `cmd/desktop/main.go` dropped the misleading `DisableWebViewDrop: true` (kept `EnableFileDrop: true`); `internal/desktop/app.go` removed the now-dead Go `OnFileDrop`/`files:dropped` emit. `go build`/`go vet`/`tsc --noEmit`/`wails build` all green. **User GUI-verified:** a dropped PDF no longer opens in the Edge viewer — drop is captured and the path delivered. **Still deferred:** converting a dropped local file into a resource needs a local-file ingestion pipeline (read bytes → extract → classify → create node) that does not exist yet — `CreateResource`/extractors are URL-oriented. So WS3 drag-drop is now partial: capture done, end-to-end conversion deferred to a future session; still not a Change 9 closeout blocker.

## Why this approach
The app is described as a local-first desktop app throughout the Outline and ADRs, but the frontend is currently a standalone web app over REST. Wails integration is what makes it actually a desktop application. Done after the backend pipeline is solid to avoid rework.

# Change 10: GBUS — Behavioral Model
Date: 2026-06-08

## What to do
Implement the GBUS behavioral model end-to-end: signal taxonomy and instrumentation, feature store aggregation, training dataset pipeline, baseline model training, inference integration into classification and search, and monitoring + governance.

## What we did
WS1 delivered: `internal/gbus/signals.go` with 10 signal type constants and `SignalWeights` map (manual_classification=1.0, category_correction=1.0, auto_classification=0.5, resource_saved=0.3, resource_deleted=0.1, resource_revisited=0.4, counter_incremented=0.2, search_query=0.2, reminder_dismissed=0.1, deep_process_confirmed=0.3), `GBUSSignalPayload` struct, `internal/gbus/emitter.go` with `SignalEmitter` (fire-and-forget goroutine, writes `aggregate_type="gbus_signal"` events), `internal/gbus/emitter_test.go` (6 tests: disabled no-op, nil store no-op, all 7 core signal types, explicit weight not overridden, async non-blocking). WS2 delivered: `GBUSCategoryFeature` / `GBUSResourceFeature` types and `GBUSFeatureStore` interface in `internal/domain`, `internal/gbus/feature_store.go` aliasing domain types, `internal/gbus/aggregator.go` (reads `ReadBySequence`, filters `gbus_signal` events, upserts category + resource features, daily ticker + startup catch-up, 30s bounded), `internal/repository/sqlite/gbus_repository.go` SQLite implementation with ON CONFLICT upsert and `PruneOlderThan`, `gbus_category_features` and `gbus_resource_features` tables added to SQLite schema (with PRIMARY KEY + indexes), `internal/gbus/aggregator_test.go` (4 tests). WS3 delivered: `scripts/gbus_train/main.go` CLI (reads feature tables, computes time-decay-weighted category affinity scores, normalizes to [0,1], evaluates proxy accuracy against manual_classification signals, saves JSON artifact, `-promote` flag for production promotion), `models/gbus/model_registry.json` with schema, promotion criteria (≥5% lift, ≥50 samples), retraining cadence, and rollback procedure. WS4 delivered: `internal/gbus/inference.go` (`Inference` struct loads JSON model artifact at startup, `CategoryScore`, `BiasClassification` (+10% max boost on sub-threshold classifications), `RerankByInterest` (blend original rank with GBUS affinity at weight 0.5), `Reload`, `ModelVersion`/`ModelStatus` for health reporting, safe no-op when disabled/model missing), `ResourceService` wired with `WithGBUSEmitter` (emits manual/auto_classification on Create, resource_deleted on Delete, counter_incremented on duplicate) and `WithGBUSInference` (biases low-confidence classifications, reranks vectorSearch results). WS5 delivered: `internal/gbus/monitor.go` (`Monitor` with `SignalCount` atomic counter, `LastCheckAt`, `CheckDrift` reads recent 500 signals and compares current accuracy to stored baseline, triggers `Reload` if drift >10%, daily ticker), `GBUSConfig` in `internal/config/config.go` (`enabled`, `inference_enabled`, `retention_days=90`, `model_path`), GBUS defaults in config, `GET /api/v1/gbus/health` endpoint returning model status/signal count/last check, `GBUSMonitor` interface on the handler, full wiring in `cmd/server/main.go` (aggregator + monitor started in `runtimeCtx`), `gbus.enabled: false` and `gbus.inference_enabled: false` defaults in `config.default.yml`. Import cycle (eventstore_test → sqlite → gbus → eventstore) resolved by lifting `GBUSFeatureStore`/feature types to `internal/domain`. Full `go test ./...` passes with zero regressions.

## Why this approach
GBUS learns from signals that only have meaning once the real pipeline exists — classification confidence to correct, content to interact with, search results to click. Doing this last ensures the training data is real and the inference has something meaningful to integrate into.

**Reconciliation note (Change 11, 2026-06-11):** Change 10 status corrected from `Complete` to `Scaffold (model not trained)`. WS1 (signal emission) and WS2 (feature store/aggregation) are genuinely implemented and verified in code. WS3 (training pipeline) has never been run against real data — `models/gbus/model_registry.json` contains only a placeholder entry (version 0.0.0, `validation_accuracy: 0.0`, `baseline_accuracy: 0.0`), and `models/gbus/baseline.json` does not exist. WS4 (inference integration) and WS5 (monitoring) are code-complete but inactive in production: `gbus.inference_enabled` defaults to `false` and there is no model to load, so the safe weighted-scoring fallback is what actually runs. See `Change_10_Workstream.md` "Remaining Work to Reach Complete".

# Change 12: Change-Documenter Skill and Session Tracking Infrastructure
Date: 2026-06-10

## What to do
Automate the mandatory 3-doc end-of-session documentation rule with a persistent background skill that tracks file mutations throughout the session and writes all required progress docs on invocation.

## What we did
Created `.claude/skills/change-documenter/SKILL.md` with two modes: Mode A (Progress_Changes — writes Changes_log.md Session N entry, Changes.md What we did, Change_N_Workstream.md [x] markings) and Mode B (Phase — writes Phase_X_Workstream.md, Phase_X_Completion_Checklist.md, Phase_X_Timeline.md). Wired three project-level hooks in `.claude/settings.local.json`: `change-doc-reset.js` (SessionStart — clears `.claude/change-doc-session.jsonl` scratch log, writes `change-doc-active` flag), `change-doc-tracker.js` (PostToolUse — appends `{ ts, tool, file, op }` entry for every Edit/Write/Bash call), `change-doc-prompt.js` (UserPromptSubmit — reads flag and scratch log, injects documenter-mode context + live file-change summary every turn). Updated `CLAUDE.md` to reference `/change-documenter` and document Phase mode behavior.

## Why this approach
The 3-doc update rule was easy to forget at session end and error-prone to execute manually (wrong session number, missed files, bare `-` instead of `[x]`). A persistent skill with background file tracking makes the rule self-enforcing — the hook log is the authoritative record of what changed, the per-turn context injection keeps the model document-aware throughout the session, and the skill writes all files in one shot.

# Change 11: Correctness Fixes & Finding 3 Verification
Date: 2026-06-07

## What to do
Verify the residual edge case for Finding 3 with an explicit regression test, and neutralize the Store.Snapshot latent comprehension trap with explicit documentation per ADR 0018.

## What we did
Added `TestMergeReplay_SkippedRowInterleaving` in `internal/sync/outbox_worker_test.go` to explicitly test the exact finding 3 edge case (untranslatable event, direct hub event, translatable event) and updated `Store.Snapshot` in `internal/eventstore/store.go` to explicitly forbid its use for projection rebuilds.

## Why this approch
Closes the final correctness loops by ensuring the exact edge cases described are formally tested, and prevents future developers from misusing the `Store.Snapshot` method contrary to ADR 0018.
