# Change 9 Workstream - Wails Integration

Date: 2026-06-08
Status: In Progress
Scope: Replace the standalone Vite/REST frontend with a proper Wails desktop app using IPC bindings, add desktop-native features, and wire the Windows + Linux build pipeline.

## Objective

Make Self Systems an actual desktop application. The frontend currently runs as a standalone Vite dev server communicating over REST/WebSocket. Wails integration packages the Go backend and React frontend into a single native binary, enables IPC (no HTTP round-trips for local calls), and unlocks desktop-native capabilities like system tray, OS notifications, and file drag-and-drop.

## Guiding Constraints

- The Go backend (Gin, repositories, services) must remain usable as a standalone HTTP server for the VPS sync path — Wails wraps it, not replaces it.
- IPC bindings should be additive: REST endpoints stay live for sync server compatibility.
- Use Wails v2 (stable) — v3 is not yet production-ready.
- Windows (primary) and Linux must both build from the same codebase.
- The existing React + TypeScript + Zustand frontend is the starting point — no full rewrite.
- Wails IPC calls replace REST calls for local (single-device) operations only; sync paths remain WebSocket/REST.

## Workstream 1 — Wails Scaffold and Go Integration

Objective:
Add Wails to the project and wire the Go backend as the Wails app backend.

Key tasks:
- [x] Add `github.com/wailsapp/wails/v2` to `go.mod`.
- [x] Create `cmd/desktop/main.go` as the Wails entry point (separate from `cmd/server/main.go`).
- [x] Define the Wails `App` struct that exposes backend methods to the frontend via IPC.
- [x] Wire existing services (ResourceService, CategoryService, etc.) into the App struct.
- [x] Verify `wails dev` launches with the existing React frontend served from `frontend/`.
- [x] Verify `wails build` produces a working binary for Windows.

Deliverables:
- [x] `cmd/desktop/main.go` — Wails entry point.
- [x] `internal/desktop/app.go` — Wails App struct with service wiring.
- [x] Updated `go.mod` / `go.sum`.
- [x] `wails.json` config file.

Done criteria:
- [x] `wails dev` launches the app in a native window with the existing frontend visible. (Verified 2026-06-11: built `SelfSystems.exe` launched, native window title "Self Systems" 1280x820, React frontend rendered — sidebar + Phase 2 console screenshotted.)
- [x] `wails build` produces a Windows binary without errors.

## Workstream 2 — IPC Bindings (Replace REST for Local Calls)

Objective:
Expose backend operations as Wails IPC methods so the frontend calls Go directly instead of over HTTP for local operations.

Key tasks:
- [x] Expose IPC methods on the App struct: `GetResources`, `CreateResource`, `UpdateResource`, `DeleteResource`, `GetCategories`, `CreateCategory`, `GetTodos`, `CreateTodo`, `GetReminders`, `CreateReminder`.
- [x] Run `wails generate module` to produce TypeScript bindings in `frontend/wailsjs/` (Wails' canonical location under `frontend:dir`; regenerated on every `wails build`).
- [x] Update Zustand stores to use IPC bindings instead of `fetch()` when running inside Wails (detect via `window.go` flag).
- [x] Keep REST client as fallback for browser/dev mode.
- [ ] Test all CRUD operations via IPC round-trip (requires `wails dev` proven launch — needs manual GUI verification).

Deliverables:
- [x] Updated `internal/desktop/app.go` with all IPC method signatures (resource, category, todo, reminder CRUD).
- [x] Generated `frontend/wailsjs/` (go/desktop/App.{d.ts,js} with 21 IPC methods, go/models.ts, runtime/).
- [x] Updated Zustand stores with IPC/REST toggle (useResourceStore, useTaskStore).
- [x] `frontend/src/lib/ipc.ts` — thin wrapper: uses Wails runtime when available, fetch otherwise.

Done criteria:
- [ ] All CRUD operations work via IPC when running as a desktop app (code path + bindings complete; runtime round-trip needs manual GUI verification).
- [x] Same frontend code works in browser mode (dev) over REST.
- [x] No duplicate state management — same stores, different transport.

## Workstream 3 — Desktop-Native Features

Objective:
Add system tray, OS notifications, and file drag-and-drop — features that make it feel like a real desktop app.

Key tasks:
- [ ] System tray: app minimizes to tray instead of closing; tray icon shows sync status. (Minimize-to-tray + Show/Quit menu implemented via `HideWindowOnClose` + energye/systray; **"tray icon shows sync status" not implemented**; runtime behavior needs manual GUI verification.)
- [ ] OS notifications: trigger native OS notification when deep processing completes or reminder fires. (`SendNotification` wired into `NotifyProcessingComplete`; reminder-fire path not wired; runtime firing needs manual GUI verification.)
- [ ] File drag-and-drop: user can drag a PDF or image file onto the app window to create a resource. (Backend `OnFileDrop` + frontend `useFileDrop` → `CreateResource` implemented; runtime drop needs manual GUI verification.)
- [ ] App window state persistence: remember window size and position across launches. (Save-on-shutdown / restore-on-startup implemented; runtime persistence needs manual GUI verification.)

Deliverables:
- [x] System tray wiring in `cmd/desktop/main.go` (`HideWindowOnClose: true`) and `internal/desktop/tray.go` (energye/systray `Register` with Show/Quit, embedded icon).
- [x] Notification calls in `internal/desktop/app.go` — `runtime.SendNotification` in `NotifyProcessingComplete`, `InitializeNotifications` in `Startup`.
- [x] Drag-and-drop handler in frontend (`frontend/src/hooks/useFileDrop.ts`, mounted in `App.tsx`) wired to the `CreateResource` IPC method via the backend `files:dropped` event.
- [x] Window state persistence — `windowState` save/restore in `internal/desktop/app.go` (JSON under OS user-config dir).

Done criteria (all require manual GUI verification — `wails build` proves the code compiles + links into the binary, but a windowed app cannot be launched headless in CI):
- [ ] App minimizes to system tray and can be restored from tray icon.
- [ ] OS notification fires when a resource finishes deep processing.
- [ ] Dragging a PDF onto the window creates a resource with the file as source.

## Workstream 4 — Build Pipeline (Windows + Linux)

Objective:
Wire the Wails build into CI for both Windows and Linux targets.

Key tasks:
- [x] Add `wails build -platform windows/amd64` to the release workflow.
- [x] Add `wails build -platform linux/amd64` to the release workflow.
- [ ] Verify both binaries launch and connect to SQLite correctly (needs manual GUI verification — not launched in CI).
- [x] Update `DEPLOYMENT.md` with desktop build and distribution instructions.
- [x] Update `README.md` to replace "Add frontend scaffolding (Wails + React)" with current status.

Deliverables:
- [x] Updated `.github/workflows/release.yml` with Wails build steps.
- [x] Updated `DEPLOYMENT.md` desktop section.
- [x] Updated `README.md`.

Done criteria:
- [x] `wails build` succeeds for both Windows and Linux in CI.
- [x] Release artifacts include `.exe` (Windows) and ELF binary (Linux).
- [x] README no longer says "Add frontend scaffolding (Wails + React)".

## Workstream 5 — Testing and Smoke Gate

Objective:
Validate IPC bindings and desktop build correctness with automated checks.

Key tasks:
- [x] Unit tests for IPC method signatures (verify correct service delegation).
- [ ] Smoke test: `wails build` succeeds, binary starts, `/health` responds (for the embedded HTTP server path).
- [x] Frontend Vitest tests: IPC mock → verify stores use IPC path when `window.go` is set.
- [x] Frontend Vitest tests: REST fallback → verify stores use fetch when not in Wails context.

Deliverables:
- [x] `internal/desktop/app_test.go` — IPC method unit tests.
- [x] Updated `frontend/src/stores/*.test.ts` — IPC/REST toggle coverage for task/reminder stores.
- [ ] CI smoke gate proven green.

Done criteria:
- [x] All Go IPC unit tests pass.
- [ ] Frontend store tests cover both IPC and REST transport paths for all stores.
- [x] CI build gate succeeds for Windows and Linux targets.

## Planned Milestones

- [x] Milestone 9A: Wails scaffold running with existing frontend in native window (WS1 complete). (Verified 2026-06-11 via launch + screenshot.)
- [ ] Milestone 9B: All CRUD operations working via IPC bindings (WS2 complete).
- [ ] Milestone 9C: System tray, OS notifications, and drag-and-drop live (WS3 complete).
- [x] Milestone 9D: Windows + Linux CI build pipeline proven green (WS4 complete).
- [ ] Milestone 9E: IPC unit tests and frontend transport toggle coverage (WS5 complete).

## Change 9 Definition of Done

- [ ] Self Systems runs as a native desktop app (not a browser tab) on Windows and Linux.
- [ ] All local CRUD operations use Wails IPC — no HTTP round-trips for local calls.
- [ ] System tray, OS notifications, and file drag-and-drop are functional.
- [x] CI produces Windows and Linux binaries on every release tag.
- [ ] Frontend works identically in browser mode (REST) and desktop mode (IPC).
- [x] README accurately reflects the current state of the frontend integration.

## Manual GUI Smoke Test (run before marking Change 9 Complete)

The WS3 + WS2-runtime done criteria above cannot be verified in CI (a windowed
app cannot launch headless). Run `cmd/desktop/build/bin/SelfSystems.exe` (or
`wails dev` from `cmd/desktop/`) and tick each item once confirmed on a desktop.
Each maps to the `[ ]` runtime criteria above — when all pass, flip those
criteria + Milestones 9A/9B/9C/9E + the Definition of Done items to `[x]` and
set `Status: Complete`.

- [x] App launches in a native window (not a browser tab); frontend renders. → WS1 done criteria, DoD "runs as native desktop app". (Verified 2026-06-11, Windows: `SelfSystems.exe` launched, window "Self Systems" 1280x820, React UI rendered via screenshot. Linux launch still pending.)
- [ ] Create / edit / delete a resource and a todo succeed with **no** local HTTP server running (proves IPC round-trip, not REST). → WS2 "All CRUD via IPC", DoD "all local CRUD use IPC". (Partial 2026-06-11: **read** path confirmed — app rendered clean "No resources/categories yet" empty states with no Gin server running, so `GetResources`/`GetCategories` resolved over IPC, not a failed REST fallback. **Write** path — create/edit/delete clicks — still needs a human.)
- [ ] Close the window → app keeps running, icon stays in the system tray; tray **Show** restores the window; tray **Quit** exits. → WS3 "minimizes to system tray".
- [ ] Drag a PDF (or image) file onto the window → a new resource is created with that file. → WS3 "Dragging a PDF … creates a resource".
- [ ] Trigger deep processing of a resource → a native OS notification fires on completion. → WS3 "OS notification fires".
- [ ] Resize / move the window, quit, relaunch → window reopens at the same size and position. → WS3 window-state task.
- [ ] Repeat the launch + a basic CRUD check on Linux (or confirm via the CI Linux artifact run on a Linux box). → DoD "native desktop app on Windows and Linux".
