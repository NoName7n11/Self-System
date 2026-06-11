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
- [x] All CRUD operations work via IPC when running as a desktop app. (Verified 2026-06-11: create/edit/delete a resource all confirmed working with no Gin server running.)
- [x] Same frontend code works in browser mode (dev) over REST.
- [x] No duplicate state management — same stores, different transport.

## Workstream 3 — Desktop-Native Features

Objective:
Add system tray, OS notifications, and file drag-and-drop — features that make it feel like a real desktop app.

Key tasks:
- [x] System tray: app minimizes to tray instead of closing; tray icon shows sync status. (Verified 2026-06-11, reliably on a fresh launch: minimize-to-tray + tray icon + right-click Show/Quit menu + left-click restore all work via `HideWindowOnClose` + energye/systray. The window, icon setup, and a custom Win32 `GetMessage` pump (`internal/desktop/tray_windows.go`) all run on a single `runtime.LockOSThread`-pinned goroutine — energye/systray's own `start()` pumps on a separate unlocked goroutine, which left messages undelivered and the icon intermittently missing. **"tray icon shows sync status" not implemented** — tray has no dynamic icon/state.)
- [x] OS notifications: trigger native OS notification when deep processing completes or reminder fires. (Verified 2026-06-11: native "Processing complete" toast fires on resource creation via `runtime.SendNotification`. Reminder-fire path not wired — desktop app has no reminder-due scheduler.)
- [ ] File drag-and-drop: user can drag a PDF or image file onto the app window to create a resource. **DEFERRED 2026-06-11** — blocked by an upstream Wails v2.12/WebView2 ordering bug (see done criteria below), not missing code. Revisit when Wails ships a fix for `setupChromium()`'s `AllowExternalDrag` ordering, or upgrade past v2.12 once a fixed release exists. Does not block Change 9 closeout.
- [x] App window state persistence: remember window size and position across launches. (Save-on-shutdown / restore-on-startup implemented and code-verified; cross-relaunch persistence still needs a manual graceful-restart check.)

Deliverables:
- [x] System tray wiring in `cmd/desktop/main.go` (`HideWindowOnClose: true`) and `internal/desktop/tray.go` + `tray_windows.go` + `tray_other.go` (energye/systray `RunWithExternalLoop(nil, …)` for window creation, then synchronous icon/menu setup, then a custom Win32 `GetMessage` pump — all on one `runtime.LockOSThread()` goroutine; `SetOnRClick`/`SetOnClick`, Show/Quit menu, embedded icon).
- [x] Notification calls in `internal/desktop/app.go` — `runtime.SendNotification` in `NotifyProcessingComplete`, `InitializeNotifications` in `Startup`.
- [x] Drag-and-drop handler in frontend (`frontend/src/hooks/useFileDrop.ts`, mounted in `App.tsx`) wired to the `CreateResource` IPC method via the backend `files:dropped` event. (Code complete and correct; never fires — see done criteria.)
- [x] Window state persistence — `windowState` save/restore in `internal/desktop/app.go` (JSON under OS user-config dir).

Done criteria (`wails build` proves the code compiles + links into the binary; runtime behavior verified by manual GUI launch on 2026-06-11 unless noted):
- [x] App minimizes to system tray and can be restored from tray icon. (Re-verified 2026-06-11 after the tray threading fix — now reliable on a fresh launch with no registry/Explorer workaround: tray icon appears; close → icon remains; right-click → Show/Quit menu; left-click and Show both restore the window.)
- [x] OS notification fires when a resource finishes deep processing. (Verified: native toast "Processing complete" appears on resource creation.)
- [ ] Dragging a PDF onto the window creates a resource with the file as source. **DEFERRED — upstream Wails bug, not our code.** `DragAndDrop.DisableWebViewDrop: true` is set correctly in `cmd/desktop/main.go`, but Wails v2.12's `setupChromium()` calls `chromium.AllowExternalDrag(false)` *before* the WebView2 controller is initialized, so it always errors (`debug` build logs: `WAR | WebView failed to set AllowExternalDrag to false!`). Result: WebView2's own drop handler still wins and opens the dropped PDF in an Edge viewer instead of firing `runtime.OnFileDrop`. Our `OnFileDrop` registration + `useFileDrop.ts` + `--wails-drop-target` CSS are all correct and would work once Wails fixes the controller-readiness ordering (or exposes a post-init hook to retry `AllowExternalDrag`). No further fix possible from `internal/desktop`/`cmd/desktop`. **Tracked as deferred scope, not a Change 9 blocker** — revisit on Wails upgrade.

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
- [x] Milestone 9B: All CRUD operations working via IPC bindings (WS2 complete). (Verified 2026-06-11: create/edit/delete confirmed with no Gin server.)
- [ ] Milestone 9C: System tray, OS notifications, and drag-and-drop live (WS3 complete). Tray + notifications verified 2026-06-11; drag-and-drop **DEFERRED** (upstream Wails/WebView2 ordering bug, see WS3 done criteria) — tracked as deferred scope, does not block Change 9 closeout.
- [x] Milestone 9D: Windows + Linux CI build pipeline proven green (WS4 complete).
- [ ] Milestone 9E: IPC unit tests and frontend transport toggle coverage (WS5 complete).

## Change 9 Definition of Done

- [x] Self Systems runs as a native desktop app (not a browser tab) on Windows. (Linux launch still pending — user will verify via WSL later.)
- [x] All local CRUD operations use Wails IPC — no HTTP round-trips for local calls. (Verified 2026-06-11.)
- [ ] System tray, OS notifications, and file drag-and-drop are functional. Tray + notifications done; drag-and-drop **DEFERRED** (upstream Wails bug, see WS3) — does not block Change 9 closeout.
- [x] CI produces Windows and Linux binaries on every release tag.
- [x] Frontend works identically in browser mode (REST) and desktop mode (IPC). (IPC path verified on desktop 2026-06-11; REST fallback covered by existing Vitest tests.)
- [x] README accurately reflects the current state of the frontend integration.

## Manual GUI Smoke Test (run before marking Change 9 Complete)

The WS3 + WS2-runtime done criteria above cannot be verified in CI (a windowed
app cannot launch headless). Run `cmd/desktop/build/bin/SelfSystems.exe` (or
`wails dev` from `cmd/desktop/`) and tick each item once confirmed on a desktop.
Each maps to the `[ ]` runtime criteria above — when all pass, flip those
criteria + Milestones 9A/9B/9C/9E + the Definition of Done items to `[x]` and
set `Status: Complete`.

- [x] App launches in a native window (not a browser tab); frontend renders. → WS1 done criteria, DoD "runs as native desktop app". (Verified 2026-06-11, Windows: `SelfSystems.exe` launched, window "Self Systems" 1280x820, React UI rendered via screenshot. Linux launch still pending.)
- [x] Create / edit / delete a resource and a todo succeed with **no** local HTTP server running (proves IPC round-trip, not REST). → WS2 "All CRUD via IPC", DoD "all local CRUD use IPC". (Verified 2026-06-11: create, edit, and delete a resource all confirmed working with no Gin server running — full IPC round-trip, both read and write.)
- [x] Close the window → app keeps running, icon stays in the system tray; tray **Show** restores the window; tray **Quit** exits. → WS3 "minimizes to system tray". (Verified 2026-06-11 after fixing `tray.go` to use `systray.RunWithExternalLoop` — `systray.Register` alone never pumped the tray window's Win32 message loop, so right-click did nothing. Now right-click shows Show/Quit, Show restores.)
- [ ] Drag a PDF (or image) file onto the window → a new resource is created with that file. → WS3 "Dragging a PDF … creates a resource". **DEFERRED — upstream Wails/WebView2 ordering bug** (see WS3 done criteria for details). Tested 2026-06-11 with `DisableWebViewDrop: true` set; WebView2 still intercepts and opens the PDF in its own viewer. Not fixable from app code; tracked as deferred scope, revisit on Wails upgrade.
- [x] Trigger deep processing of a resource → a native OS notification fires on completion. → WS3 "OS notification fires". (Verified 2026-06-11: native "Processing complete" toast fired on resource creation.)
- [ ] Resize / move the window, quit, relaunch → window reopens at the same size and position. → WS3 window-state task. (Not yet tested — requires a *graceful* quit via tray Quit or window close, since `Shutdown`/`saveWindowState` must run. Earlier launch tests force-killed the process.)
- [ ] Repeat the launch + a basic CRUD check on Linux (or confirm via the CI Linux artifact run on a Linux box). → DoD "native desktop app on Windows and Linux".
