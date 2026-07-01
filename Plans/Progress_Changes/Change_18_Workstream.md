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

### GRAPH stability + drag fluidity (fidelity fix 2) ✅
`frontend/src/components/graph/GraphCanvas.tsx` + `frontend/src/stores/useResourceStore.ts`. User report (round 2): nodes "constantly moving in a circle", "flickering as if refreshing", and per-node drag "rough/irregular, not smooth" vs the design's fluid drag. Two distinct root causes:

- **Phantom 12s refresh** — with no backend, `useSyncStore` falls to offline polling (`fallbackPollIntervalMs` 12s) → `loadResources({silent})` → `demoResourcesAsItems()` builds a *fresh array of fresh objects* every poll → store sets a new `resources` reference → filter memo + GraphCanvas `[resources]` rebuild effect fire → `warmRef=false` → nodes snap back to the seed circle and re-explode. Every 12s. Fix in `useResourceStore.loadResources`: added `resourcesSignature()` (id:url:title:cat:summary) and on a **silent** reload, if the signature equals the current one, early-return without `set()` — keeps the same array reference so nothing downstream rebuilds. Applied to both the success and catch (backend-unreachable) branches.
- **Diverged physics → rough drag** — an earlier attempt added d3-style **alpha decay** (tick scaled by a decaying alpha, stop at <0.001, reheat to 0.3 on drag). That diverged from the design, which runs `step()` at **full force every frame, forever** — the system reaches equilibrium (repulsion≈spring+gravity) and 0.9 damping kills residual velocity, so it's visually static at rest yet *immediately* fluid on drag because forces are always at full strength. Reverted alpha entirely; `tick()` now mirrors the design `step()` exactly (incl. coincident-node random nudge, gravity-before-fixed-check ordering, ±14 velocity clamp). Also switched the event model to match the design: `mousedown`/`wheel`/hover bound on the **canvas**, but `mousemove`/`mouseup` bound on the **window** so a drag keeps tracking when the cursor leaves the canvas; coords via `getBoundingClientRect`+clientX (design `evPos`); circular `hypot` hit test (was square AABB).

Verified (Playwright, programmatic, demo fallback): after warm-up the canvas is **pixel-perfect static** (0 changed px across repeated 400ms samples); dragging a node moved 23,843 px (node follows cursor + neighbours react = fluid); after release the sim **re-converges to 0** changed px over ~3-4s (no limit-cycle/orbit). Build clean.

### Left_Rail interaction fixes (fidelity pass) ✅
`frontend/src/components/layout/Sidebar.tsx`, `frontend/src/styles.css`, `frontend/src/stores/useResourceStore.ts`. Four user-reported fixes that mostly conclude the Left_Rail:

- [x] **Library "open in dock" tab** — `RailLibrary` open-in-dock called `openDockTab("categories")`; now `openDockTab("library")` so the dock lands on the LIBRARY tab. Verified: dock active tab reads `LIBRARY`.
- [x] **RECENT hold-to-clear** — added `clearRecents()` to `useResourceStore` and tap/hold handlers on the RECENT toggle (mirrors design `startHoldRecent`: mousedown animates a `.rail-recent-fill` bar over 1.5s → `clearRecents()`; release <400ms → `toggleRecent`; mouseleave cancels). Also removed the `deriveRecents` fallback in the `recents` memo — it re-invented a list from arbitrary resources and masked the clear (cleared list still showed 5). Verified: hold 5→0 ("NO RECENT ITEMS"); quick tap collapses.
- [x] **CATEGORY NODES compaction** — removed the per-row colored type badge (`rail-catnode-badge`); rows are now title-only with tighter padding so long node lists need less scrolling. Verified: 0 badges, row markup is just the title.
- [x] **Uniform "open in dock" icons** — restyled `.nav-affordance` (home CHAT/TASKS/LIBRARY rows) to a 24×24 bordered box matching `.rail-icon-btn` (the LIBRARY-section icon), keeping the hover-reveal. Verified: affordance is 24×24 with a 1px border.

Deferred to UI finishing (user): remaining minor Left_Rail polish items.

### Main_Content fixes (GRAPH / MAP / PROGRESS / Notifications) ✅
`frontend/src/components/graph/GraphCanvas.tsx`, `frontend/src/components/layout/Topbar.tsx`, `frontend/src/components/icons.tsx`, `frontend/src/stores/useLayoutStore.ts`, `frontend/src/styles.css`.

- [x] **GRAPH empty after MAP roundtrip** — the `<canvas>` was conditionally mounted only when `view==='graph'`, so switching to MAP unmounted it; the `ResizeObserver`/sizing effect (deps `[]`) never rebound to the remounted element, leaving its backing store at the default 300×150 → blank graph on return. Fix: keep the canvas always mounted (hidden via `.is-inactive { visibility:hidden }` under the map/progress overlays) and gate the rAF physics/draw with a `viewRef` so it idles off-graph. Verified: 16,383 non-empty px before and after a graph→map→graph roundtrip.
- [x] **MAP rework → interactive mind-map** — replaced the CSS-`scale`-only static list with a faithful port of the design (§11): `buildMapLayout()` builds the ROOT→category→task tree (x=`depth*232`, ROW 44, parents centered on children, cubic-Bézier edges, node widths root 150/cat 184/task 210); rendered as a pannable/zoomable world (`translate(panX,panY) scale(zoom)`) with an SVG edge layer + absolutely-positioned HTML nodes. Added `mapPan`/`setMapPan` to the store, drag-to-pan (window-level move/up, ignores node clicks), wheel-zoom, expand toggles, `EXPAND ALL/COLLAPSE ALL`, zoom pill (clamp 0.45–1.6), and FIT that resets zoom + recenters on the root (auto-centers on entering MAP). Verified: root+3 cats / 3 edges collapsed → 10 nodes expanded; pan changes the world transform.
- [x] **PROGRESS "RECENTLY COMPLETED" column customize + per-row remove** (new feature, not in spec) — a customize button at the section's right edge opens a checkbox menu (DATE · TIME · TYPE · CATEGORY, multi-select, persisted to `localStorage` `ss-doneCols`); selected columns render in each row's meta. Added per-row `×` removal (`removedDone` set) like LIBRARY; the list is sliced-then-filtered so a remove visibly shrinks it instead of backfilling. Verified: meta spans 1→2 on toggling DATE; 4-item menu; row remove works.
- [x] **Notifications fixed** — list moved into the store (`notifList`) from the static `DEMO_NOTIFS`. `CLEAR` now calls `clearNotifs()` (empties the list, keeps the panel open with a `NO NOTIFICATIONS` empty state) instead of merely closing. `MUTE`/`MUTED` toggle reflects in the header (accent when muted). Added per-notification `×` removal (`removeNotif`). Bell now has three states via new `IcoBellAlert` (filled) / `IcoBellOff` (slashed) glyphs: plain bell when the list is empty, alert bell when there are notifications, off bell when muted; the unread dot shows only when there are unseen, unmuted notifs. Verified: per-row remove 4→3; MUTE→"MUTED" + slashed bell; CLEAR→0 rows, panel stays open, plain bell.

### Main_Content polish (notif hover + map sharpness) ✅
`frontend/src/styles.css`.
- [x] **Notification hover resize** — the per-row `×` toggled `display:none→flex`, adding width and reflowing/resizing the row on hover. Made `.notif-x` `position:absolute` (top-right, opacity-revealed) and it now overlays the timestamp (`.notif-time` fades out on hover), so showing it never changes the row's box. Verified: row stays 314×69 px on hover.
- [x] **MAP blurry at rest** — `.map-world` had `will-change:transform; backface-visibility:hidden`, which promoted it to a GPU layer rasterized once at idle → blurry scaled text until an interaction forced a re-raster (sharp). Removed both hints so every paint re-rasterizes crisply. Verified: `will-change:auto`, nodes render sharp at the idle transform.
- [x] **MAP pan trails (zoomed out) vs blur — the coupled tradeoff** — with no compositing layer (the blur fix), a fast pan left repaint "trails" smearing from the root, worst when zoomed out; re-adding a permanent layer brought the blur back. Resolved both by promoting the world **only during interaction**: a `mapBoost` flag sets inline `willChange:'transform'` on mousedown (pan) and on wheel (auto-clears 220ms after the last wheel), and back to `'auto'` on release. Composited while moving (no trails), un-promoted at rest (sharp). Verified via Playwright: `willChange` = auto at rest → transform during drag/wheel → auto after.

### Bottom_Dock fixes (CHAT / TASKS / LIBRARY / CATEGORIES) + MAP trail ✅
`frontend/src/components/chat/ChatDock.tsx`, `frontend/src/stores/useTaskStore.ts`, `frontend/src/stores/useLayoutStore.ts`, `frontend/src/components/graph/GraphCanvas.tsx`, `frontend/src/styles.css`.

- [x] **CHAT attach + role labels** — wired the composer `+` to a hidden `<input type=file>` (opens picker, stages chips with `×` removal); removed the per-message `YOU`/`ASSISTANT` labels (color-coding already conveys author).
- [x] **TASKS uniform cards + AUTO category** — `.dock-task-card` now `min-height:76px` with `align-items:stretch` so cards in a row are equal height yet grow for long titles (verified uniform `[76]`). The create-form CATEGORY picker leads with an **AUTO** chip (default `taskDraft.cat='auto'`, mapped to a real category on create), then the first 3 categories inline, with the rest in a scrollable **MORE ▾** dropdown — scales to many future categories.
- [x] **LIBRARY column customize** — added a MAP-style multi-select menu (TYPE · CATEGORY · DATE, persisted to `localStorage` `ss-libCols`) at the right end of the RECENT header (`libCols`/`toggleLibCol` in the layout store); rows render only the enabled columns.
- [x] **CATEGORIES ingest add-menu + new category** (design §6.2) — added the split-ADD caret that opens the add-options menu (*best-match* / *+ add as new category…*), the accent-bordered NEW CATEGORY row (NAME input + CANCEL + CREATE & ADD, next-unused palette color), and wired the ingest `+` to a real file picker with staged chips. Creating adds a live category card (verified 6→7 with a CLIENTS card) and selects it.
- [x] **MAP pan trail (residual) — final fix** — the interaction-only world promotion (Session 64) still left a faint trail because the first drag frames paint into the un-promoted viewport before React commits the boost, and that div never repaints to clear them. Replaced it by permanently compositing the **viewport** (`.map-viewport { transform: translateZ(0); contain: paint }`) — it is never scaled, so it owns and clears its pixels every frame (no trails) while the un-promoted world re-rasterizes its scaled text crisply (no blur). Removed the `mapBoost` state/handlers. Verified: viewport composited + `contain:paint`, world `will-change:auto`; map sharp at 45% zoom with no smear around the root.

Workspace housekeeping: verification screenshots now live in `Plans/New_UI/References/verification_shots/` (kept, not deleted).

### Dock/Rail polish — ingest center, dropdown anchor, chat gaps, row scroll, uniform scrollbars, MAP trail ✅
`frontend/src/components/chat/ChatDock.tsx`, `frontend/src/styles.css`.

- [x] **CATEGORIES ingest centered + dropdown anchored** (design §6.2, img_6 vs img_7) — wrapped ingest contents in `.ingest-inner` (`max-width:560px; margin:0 auto`) and made `.ingest-bar` `width:100%`; the `.ingest-add-menu` now right-aligns to the ADD/caret and drops 4px below the bar instead of opening far away (verified `menu.right==bar.right`, `menu.top==bar.bottom+4`).
- [x] **CHAT message gaps** (§6.3) — `.chat-thread` gap `6→12px` + padding `14×18`, `.chat-bubble` `max-width 80→74%`, padding `9×12`, font `11→12px`; bubbles no longer vertically cramped or over-stretched.
- [x] **Left_Rail LIBRARY/RECENT constant row size + scroll** (img_7) — added `flex:none` to `.rail-lib-row`/`.rail-recent-row`/`.rail-catnode-row` so they stop flex-shrinking in a small window; rows hold design height and the list scrolls (verified at 900×560: 21 rows @ 34px, list scrolls 734>463).
- [x] **Uniform scrollbars** (§1.7, img_8) — one global `::-webkit-scrollbar` rule (8px, transparent track, square `#26262C`→`#34343C` thumb); removed the six per-section `3px` overrides and the standard `scrollbar-width:thin` line (it overrode webkit in Chromium/WebView2). Verified all scroll zones now 8px.
- [x] **MAP pan trail (root node) — definitive fix** (img_9) — added `will-change: transform` to `.map-node` so every node moves on its own compositor layer and never leaves stale residue in the viewport-layer tiles (the bright root was the only visible offender). Chrome rasters promoted layers at effective screen scale → no blur regression (verified sharp at 160% max upscale and 45%).

### Notifications + chat attachments + Right_Rail (PREVIEW · archive · category badge) ✅
`frontend/src/components/layout/Topbar.tsx`, `frontend/src/components/chat/ChatDock.tsx`, `frontend/src/components/layout/Sidebar.tsx`, `frontend/src/components/resource/ResourceForm.tsx`, `frontend/src/stores/useChatStore.ts`, `frontend/src/stores/useResourceStore.ts`, `frontend/src/hooks/useFilteredResources.ts`, `frontend/src/types.ts`, `frontend/src/styles.css`.

- [x] **Notifications expand in place** (§9.3) — click a notif to toggle `is-expanded`; `.notif-text` clamps to 2 lines by default and un-clamps with `max-height:160px; overflow-y:auto` (title wraps); the `×` stops propagation. Verified line-clamp 2→none, overflow auto.
- [x] **Chat attachments reflect + "+N more" tooltip** (§7/§9.5) — `ChatMessage.atts` + `sendToConversation(atts)`; shared `MsgAtts` renders 2 inline chips + a `+N more` hover tooltip listing all; wired into both dock ChatTab and Left_Rail RailChat (rail got a real file picker too). Verified: 3 files → 2 chips + "+1 more" tooltip with all 3.
- [x] **Right_Rail OPEN→PREVIEW** (§8.2c/§8.3) — links keep OPEN; other types show a PREVIEW toggle (accent `is-active`) that swaps AI SUMMARY for an inline `InlinePreview` mock (PDF/DOC page · NOTE card · IMAGE frame). Verified PDF→toggle swaps section + accent fill.
- [x] **Remove "ASK ABOUT THIS RESOURCE…" composer** — `.inspector-footer` block + `ask` state removed (suggested-Q chips still route into dock chat). Verified footer gone.
- [x] **ARCHIVE + DELETE functional** — `archivedLib`/`archiveResource`/`restoreResource` in the resource store; ARCHIVE clears selection + opens ARCHIVE tab + hides from library/graph; RESTORE un-archives; DELETE uses existing delete. EDIT left as no-op per user. Verified archive (2→3) + restore (3→2).
- [x] **Category-node badge instead of type** — Inspector meta badge shows the category name in the category color (type already in the preview well); duplicate category tag chip dropped. Verified badge=RESEARCH (category blue).

### PREVIEW sticky fix + confirmation notification (Session 68)
- [x] **PREVIEW sticky across nodes** — narrowed the `useEffect` in `ResourceForm.tsx` from resetting on every `selectedId` change to resetting only when the selected type is `link`; PREVIEW now persists as the user clicks through previewable nodes and auto-ends only on a link selection (links show OPEN, never PREVIEW). User re-enables PREVIEW by clicking again.
- [x] **Confirmation notification** — prepended `n0` ("UI changes deployed", long body) to `DEMO_NOTIFS` in `demoData.ts` as a visible deploy confirmation and a functional test of the Session 67 click-to-expand/scroll notification behavior.

## Deferred (post-18, documented)
Resize handles + persistence · conversation rename · task create form + T/P/D segments · drag-drop ingest/attach · add-menu best-match/new-category · hold-to-clear recent · Inspector EDIT action · RESTORE for demo category-archived seed (r20/r21).
