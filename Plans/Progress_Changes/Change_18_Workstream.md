# Change 18 Workstream — Instrument UI Full Rebuild

Date: 2026-06-27
Status: In Progress
Scope: Re-baseline the entire frontend UI on `Redesign from scratch_7/Self Systems.dc.html` (the authoritative design), replacing the Change 17 build that followed the reduced `Minimal_UI_Spec.md`. Presentation-layer only — Go backend, Wails IPC, store shapes, and `api/client.ts` contract unchanged.

Spec: `Plans/Progress/Updated_UI.md` (v2). Supersedes Change 17.

---

## Why

Change 17 looked sparse and was missing major features because it followed the wrong (reduced) spec and had no data. This change rebuilds against the real prototype and ships a frontend fallback seed dataset so the UI always looks populated.

## Guiding constraints
- Wails desktop app — all data via `ipc.ts` bridge (IPC when `window.go`, REST fallback).
- Store *shapes* unchanged; only add fallback-to-demo behavior.
- Custom canvas 2D graph sim from Change 17 is kept (it was correct).
- `react-force-graph-*` stays in package.json for future 3D; not imported.

---

## Phase 1 — State + Seed (foundation) ✅
- [x] `src/lib/demoData.ts` — seed dataset (21 res, 6 cat, 6 task, 3 conv, 4 notif) + adapters
- [x] `src/types.ts` — `DockTab` += `archive`; `GraphView` = `graph/map/progress`; add `LeftView`
- [x] `src/stores/useLayoutStore.ts` — full state model (leftView, dockTab+archive, view, selectedCat, recentOpen, catsOpen, libFilter, notif*)
- [x] `src/stores/useResourceStore.ts` — fallback to demo resources when backend empty
- [x] `src/stores/useTaskStore.ts` — fallback to demo todos when backend empty
- [x] `src/stores/useChatStore.ts` — seeded with demo conversation
- [x] Build clean after P1

Done: opening with no backend shows 21 resources / 6 cats / 6 tasks, r1 selected.

## Phase 2 — Left rail state machine ✅
- [x] `src/components/icons.tsx` — add cross, dots, pencil, trash, archive, bell
- [x] `Sidebar.tsx` — home view (search, nav w/ ↗, RECENT collapsible, CATEGORY NODES collapsible, UPDATE banner, footer)
- [x] `Sidebar.tsx` — chat view (conv list → thread)
- [x] `Sidebar.tsx` — tasks view (grouped columns)
- [x] `Sidebar.tsx` — library view (filter chips + rows)
- [x] `Sidebar.tsx` — collapsed strip
- [x] nav row → `leftView`; ↗ affordance → dock tab

Done: clicking CHAT/TASKS/LIBRARY swaps rail body; back returns home; categories + recent render from seed.

## Phase 3 — Top bar views + notifications + dock ingest/archive ✅
- [x] `Topbar.tsx` — GRAPH/MAP/PROGRESS switch + notifications bell
- [x] Notifications dropdown (list, mute, clear, unseen dot)
- [x] `ChatDock.tsx` — categories tab: ingest command bar (`PASTE A URL…` + `+` + ADD) + right category cards w/ dot-matrix
- [x] `ChatDock.tsx` — 5th archive tab (restore rows)
- [x] `ChatDock.tsx` — chat/tasks/library tabs re-baselined

Done: command bar present; categories cards show counts + meters; archive tab works; notifications open.

## Phase 4 — Inspector richness ✅
- [x] preview well (type label, host, cat swatch)
- [x] details (counter ×n, type badge, cat + tag chips)
- [x] quick actions (OPEN/EDIT/ARCHIVE/DELETE)
- [x] connections list (relation labels) from seed `connections`
- [x] AI summary + 3 suggested-Q chips → push to dock chat

Done: selecting r1 shows full populated inspector matching design.

## Phase 5 — MAP + PROGRESS + overlays
- [x] MAP view (category hubs + child nodes)
- [x] PROGRESS view (processing queue empty state + recently completed)
- [ ] inline preview variants in inspector (paged/link/note/image) — deferred
- [ ] context menus (conversation, task), AI clarify popup — deferred

Done: all three top-bar views render. Inline preview + context menus deferred to a follow-up.

---

## Milestones
- [x] 18A — P1 state + seed, build clean, UI populated
- [x] 18B — P2 rail state machine
- [x] 18C — P3 top-bar views + notifications + dock ingest/archive
- [x] 18D — P4 inspector
- [x] 18E — P5 map/progress (overlays/inline-preview deferred)

## Definition of Done
- [ ] UI visually matches `_7` screenshots with seed data (full, not barren)
- [ ] Left rail swaps home/chat/tasks/library; collapse works
- [ ] Command bar present in categories dock tab
- [ ] GRAPH/MAP/PROGRESS all render
- [ ] Inspector fully populated (counter, tags, connections, summary, suggested Qs)
- [ ] Notifications dropdown works
- [ ] `npm run build` clean; `go test ./...` unaffected
- [ ] Real backend data overrides seed when present

## Fidelity pass (Change 18.1) — user-guided, component by component

### Left rail ✅
Full per-component spec docs in `NEW_UI/Left_Rail/`. Fixes applied in `styles.css` (two passes):

Pass 1 (alignment/colour):
- Nav labels + RECENT/catnode/library titles were centered (button default) → `text-align:left`.
- Footer sub-line uppercase → lowercase `local · single user`, `--text-mute`.
- Nav row/icon colour → `#B9B9C0` / `#7A7A84`.

Pass 2 (spec values from NEW_UI docs):
- Header: logo chip `28→24px` + hover brightness; header gap `8→10`, padding `0 12→0 14`.
- Search: boxed (margin `12 12 8`, all-side border `#25252B`, bg `--bg-input`) instead of border-bottom; placeholder `#4E4E57`; input `letter-spacing .3px`.
- Nav: container `padding:4px 8px; gap:1px`; rows `padding 0 8; gap 10; border 1px transparent`.
- RECENT rows: `height 30; gap 10; padding 0 8`; title `#B9B9C0`; type label `8px #4E4E57`.
- Footer avatar: `28→26px`, bg `#1B1B20`, border `#2A2A30`, accent letter.
- Collapsed strip: gear `margin-bottom:10px`.

Deferred (documented in NEW_UI): resize handle, hold-to-clear recent, hover swap (type↔remove ✕), category-node selected-state polish.

### Panel resize + snap-collapse ✅ (all three panels)
`frontend/src/hooks/useResize.ts` + px sizes in `useLayoutStore` (`leftWpx/rightWpx/dockHpx/resizing`). Drag handles: left rail right-edge, right rail left-edge, dock top-edge (`.resize-handle*`, hover accent). On release, snaps to collapsed only when dragged **below `collapseAt`** — set well under each default so it isn't twitchy:
- Left: floor 120, **collapseAt 150** (def 264) — low sensitivity.
- Right: floor 150, **collapseAt 185** (def 336) — low sensitivity.
- Dock: floor 90, **collapseAt 160** (def 264) — design feel.
Panels render inline `width/height` from store; `transition:none` while resizing. Verified: moderate drag resizes without collapse; aggressive drag snaps collapsed.

**Rework (fidelity fix):** Left rail + dock appeared to have *no* resize option and handles looked misplaced. Root cause — `.left-rail`, `.dock`, `.right-rail` lacked `position: relative`, so the absolutely-positioned `.resize-handle*` (top/right/left: 0) anchored to the viewport, not the panel: the left handle landed over the right rail, the dock handle at the page top, etc. Fix: added `position: relative` to all three panel containers in `styles.css`. Verified via Playwright — handles now sit exactly on each panel edge (left@258, dock-top@776, right@1798) and drag resizes correctly (left 264→344 on +80px). Matches design (`Self Systems.dc.html` wraps each panel relative with a 5px `ss-handle`).

### GRAPH rework (fidelity fix) ✅
`frontend/src/components/graph/GraphCanvas.tsx`, ref `Redesign_from_scratch_8/Self Systems.dc.html` §10 + `Self_Systems_UI_Build_Spec.md`. User report: graph "very glitchy" + nodes "too rigidly symmetrical". Root causes + fixes:

- **Rigid symmetry** — hub↔hub links were *all-pairs* (15 edges for 6 cats), forcing a perfect symmetric polygon; resource→hub were the only other links so clusters were perfectly radial. Replaced with the design's **sparse 7-edge `CATLINKS`** + added the missing **resource↔resource `con` links** (from `connections`, deduped `a<b`, len 78/k 0.015). Node radii now vary `5 + counter*0.7` (were fixed 5). Result: organic, asymmetric layout.
- **Glitch (jump)** — `fitNodes` ran on *every* ResizeObserver tick, so the graph re-framed/jumped whenever any panel was dragged or collapsed. ResizeObserver now only updates the DPR backing-store size + centers once; refit happens solely on warm-up settle and the FIT/100% buttons.
- **Blur/jitter** — added DPR handling (`canvas.width = cssW*dpr`, `ctx.setTransform(dpr,…)`) and switched the renderer to **screen-space projection** (world→screen per point, constant-px line widths and fonts) instead of `ctx.scale()` — crisp at all zooms, matches design `w2s`.
- **Cross-zone bug** — graph hub id was `cat:<name>` but the dock cat-cards and left-rail CATEGORY NODES key off `categoryId`. Switched hub id to `categoryId`, so a hub click and a dock card now drive the *same* `selectedCat` → graph isolation + left-rail node list + inspector stay in sync.
- **HUD** now drawn on canvas every frame (live `NODES·EDGES·ZOOM%`); the stale React HUD div was removed. Added hover highlight + cursor feedback. Zoom buttons use the design's 1.2 factor.

Verified (Playwright, demo fallback): NODES 27 · EDGES 37 (21 res→cat + 7 CATLINKS + 9 con); dock RESEARCH card → left rail shows RESEARCH + 9 node rows + graph isolates the research cluster; dock resize 264→344 with no graph jump; DPR backing store correct. Build clean.

## Deferred (post-18, documented)
Resize handles + persistence · conversation rename · task create form + T/P/D segments · drag-drop ingest/attach · per-type inline preview rendering · add-menu best-match/new-category · hold-to-clear recent.
