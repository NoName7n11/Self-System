# Updated UI — Instrument Redesign

Date: 2026-06-26
Status: Planned — implementation pending
Scope: Full presentation-layer reskin of the Self Systems desktop frontend. Go backend, IPC bindings, Wails shell, Zustand stores, and api/client.ts are unchanged.

---

## Design Authority

Primary spec: `Self Systems.dc.html` + `Self_Systems_UI_Build_Spec.md` (in claude.ai/design project "Redesign from scratch").
Secondary reference: `Plans/New_UI/Minimal_UI_Spec.md` (Figma sessions 59–61).
When the two conflict, `Self_Systems_UI_Build_Spec.md` wins.

---

## Design Intent

"Data-instrument" aesthetic:
- **Font**: JetBrains Mono only — weights 400/500/600/700/800. No other typeface.
- **Accent**: `#F0703C` orange — selection, active states, CTAs.
- **Shape**: `border-radius: 0` everywhere. Hard edges only.
- **Borders**: always `1px solid`. Dashed only for `+ NEW` affordance.
- **Icons**: custom 7×7 pixel SVG grid. 2×2 rect per lit cell. `currentColor`.
- **Texture**: dot-grid canvas background (`radial-gradient` 24×24), dot-matrix fill meters.
- **No gradients. No emoji. No external UI library.**

---

## Layout — 3-Zone Shell

```
┌──────────┬──────────────────────────────────────────┬──────────┐
│ LEFT     │ TOP BAR (52px)                            │ RIGHT    │
│ RAIL     ├──────────────────────────────────────────┤ RAIL     │
│ 264px    │ GRAPH CANVAS (flex:1, dot-grid bg)        │ 336px    │
│ (→56px)  │   custom canvas 2D physics sim            │ (→56px)  │
│          │   zoom controls bottom-right              │          │
│          │   LIST / TIMELINE overlays                │ INSPECTOR│
│          ├──────────────────────────────────────────┤          │
│          │ DOCK (264px ↔ 42px)                       │          │
│          │   Categories · Chat · Tasks · Library     │          │
└──────────┴──────────────────────────────────────────┴──────────┘
```

Shell: `display:flex; height:100vh; width:100vw; overflow:hidden`.
Center column: `flex:1; min-width:0; flex-direction:column`.
Left/Right rails: `flex:none`, animated width (`transition: width 0.18s ease`).

---

## Design Tokens (CSS Custom Properties)

### Color
| Token | Hex | Use |
|---|---|---|
| `--bg-app` | `#0B0B0D` | App background, graph canvas |
| `--bg-panel` | `#0E0E11` | Rails, top bar, dock |
| `--bg-card` | `#131318` | Cards, AI chat bubble |
| `--bg-input` | `#15151A` | Inputs, hover surfaces |
| `--bg-inset` | `#0B0B0D` | Preview wells |
| `--border` | `#26262C` | Default 1px border |
| `--border-soft` | `#1A1A1F` | Section dividers |
| `--border-hover` | `#34343C` | Hover/focus border |
| `--text` | `#E9E9EC` | Primary text |
| `--text-dim` | `#9A9AA0` | Secondary text |
| `--text-mute` | `#5C5C66` | Labels, captions |
| `--text-faint` | `#3C3C44` | Disabled, placeholders |
| `--accent` | `#F0703C` | Selection, active, CTAs |
| `--accent-soft` | `rgba(240,112,60,0.12)` | Active backgrounds |
| `--accent-bd` | `rgba(240,112,60,0.35)` | User chat bubble border |

### Category Colors
| Name | Hex |
|---|---|
| research | `#5B9CF6` |
| ai | `#A98BF5` |
| finance | `#48C78E` |
| people | `#E5B567` |
| sources | `#56B6C2` |
| archive | `#E06C75` |

### Resource Type Colors
| Type | Hex | Badge |
|---|---|---|
| pdf | `#F67373` | PDF |
| link | `#48C78E` | LINK |
| note | `#5B9CF6` | NOTE |
| doc | `#9B59F6` | DOC |
| image | `#F6739B` | IMAGE |

### Type Scale (px)
micro-label 9 · caption 10 · body-sm 11 · body 12 · base 13 · inspector title 14 · metric 26

---

## State Model

Lives in `useLayoutStore`. Clean break from old `sidebarCollapsed`/`activeSection`.

```ts
interface UIState {
  leftCollapsed: boolean;       // false = 264px, true = 56px
  rightOpen: boolean;           // false = 56px, true = 336px
  dockOpen: boolean;            // false = 42px strip, true = 264px
  dockTab: DockTab;             // 'categories' | 'chat' | 'tasks' | 'library'
  view: GraphView;              // 'graph' | 'list' | 'timeline'
  selectedCat: string | null;   // isolate-highlight category id, null = all
}
```

`query` stays in `useResourceStore.filters.query` (shared: left-rail search + top-bar filter bind to same field).
`selectedId` stays in `useResourceStore.selectedResourceId`.

---

## Pixel Icon System

7×7 grid SVGs. Each lit cell = 2×2 `<rect>`. `viewBox="0 0 14 14"`. `fill="currentColor"`. `shape-rendering:crispEdges`.

Helper:
```tsx
function px(cells: [number,number][]): JSX.Element {
  return (
    <svg width={15} height={15} viewBox="0 0 14 14"
      style={{ display: 'block', shapeRendering: 'crispEdges' }}
      fill="currentColor">
      {cells.map(([c, r], i) => (
        <rect key={i} x={c * 2} y={r * 2} width={2} height={2} />
      ))}
    </svg>
  );
}
```

Icons and their cell coordinates:
| Name | Cells |
|---|---|
| logo | [1,1][1,2][1,3][1,4][1,5][5,1][5,2][5,3][5,4][5,5][2,3][3,3][4,3] |
| search | [1,1][2,1][3,1][1,2][3,2][1,3][2,3][3,3][4,4][5,5] |
| chat | [1,1][2,1][3,1][4,1][5,1][1,2][5,2][1,3][2,3][3,3][4,3][5,3][1,4] |
| tasks | [1,3][2,4][3,3][4,2][5,1] |
| library | [1,1][2,1][3,1][4,1][5,1][1,3]…[5,3][1,5]…[5,5] |
| grid | [1,1][2,1][1,2][2,2][4,1][5,1][4,2][5,2][1,4][2,4][1,5][2,5][4,4][5,4][4,5][5,5] |
| gear | [3,1][3,5][1,3][5,3][2,2][4,2][2,4][4,4][3,3] |
| plus | [3,1][3,2][3,3][3,4][3,5][1,3][2,3][4,3][5,3] |
| send | [1,1][1,2][2,2][1,3][2,3][3,3][1,4][2,4][1,5] |
| filter | [1,1][2,1][3,1][4,1][5,1][2,3][3,3][4,3][3,5] |
| trend | [1,5][2,4][3,3][4,2][5,1][3,1][4,1][5,2] |
| chevL | [4,1][3,2][2,3][3,4][4,5] |
| chevR | [2,1][3,2][4,3][3,4][2,5] |
| chevUp | [1,4][2,3][3,2][4,3][5,4] |
| chevDn | [1,2][2,3][3,4][4,3][5,2] |

---

## Components

### Left Rail (`Sidebar.tsx`)

**Expanded (264px) — default:**
- Header (52px): accent logo chip → `SELF SYSTEMS` wordmark → `LOCAL · v0.1.0` → `chevL` collapse toggle.
- Search bar (34px): `search` icon + input (`SEARCH RESOURCES, TAGS…`) + `/` key hint. Binds to `useResourceStore.filters.query`.
- Primary nav (3 rows × 34px): `CHAT` · `TASKS` · `LIBRARY`. Each row: pixel icon + label + right-end `trend` affordance. Click row or affordance → `setDockOpen(true); setDockTab(tab)`. Affordance: `opacity:0`, reveal on hover via CSS `.nav-row:hover .nav-affordance { opacity:1 }`.
- Recent (cap 5): 9px type-color swatch + truncated title + type label. `deriveRecents(resources)` sorted by `createdAt` desc. Click → `selectResource(id); setRightOpen(true)`.
- Footer (52px): avatar chip (`N`) + `noname / local · single user` + inline `gear` icon → opens settings.

**Collapsed (56px):**
Icon-only strip: logo chip (36px) + 3 nav icons (36px each) + gear pinned bottom.

### Top Bar (`Topbar.tsx`)

52px fixed. Contents:
- Left: `KNOWLEDGE GRAPH` (13px, `--text`) · `N RESOURCES` subtitle (11px, `--text-dim`).
- Right: filter input (`filter` icon, placeholder `FILTER NODES…`, binds to `useResourceStore.setQuery`) · segmented control `GRAPH / LIST / TIMELINE` (active = `bg:#22222A; color:var(--accent)`).

### Graph Canvas (`GraphCanvas.tsx`)

Custom canvas 2D. No library. `requestAnimationFrame` loop. All physics state in mutable refs (not React state → no re-renders during sim).

**Node types:**
- Hub (`kind:'cat'`): `mass:4, r:13`. Color from category palette.
- Resource (`kind:'res'`): `mass:1, r: 5 + (saveCount * 0.7)`. Color from type palette.

**Links:**
- Resource → hub: `len:90, k:0.04`, strong (solid, category color).
- Hub ↔ hub: `len:230, k:0.02`, weak (dashed `#3A3A42`).
- Resource ↔ resource: `len:78, k:0.015`, weak (dashed `#2E2E36`) — deferred (no connection data yet).

**Physics per tick:**
```
repulsion (all pairs): F = charge / d²   charge = 26000 (hub involved) else 9000
spring (each link):    F = (d − len) × k
gravity (per node):    v += −pos × g     g = 0.004 (hub), 0.012 (resource)
integrate:             v *= 0.9 (damping); clamp |v| ≤ 14; pos += v
```

**Seed:** hubs on circle r≈170, equally spaced. Resources ±55 near their hub. Run 260 silent steps on mount, then `fit()` at 60ms/360ms/900ms.

**Drawing order:** edges first (strong: solid α.32; weak: dashed [2,3] α.6) → hubs (pixel square + punched center + label always) → resources (filled square; selected → accent stroke + accent-soft fill; label at `scale > 1.35`).

**Dimming:** when `query` or `selectedCat` set — non-matching nodes drop to α.22.

**Interaction:**
- Hit-test in world coords (`radius + 6/scale` slop).
- Drag: pin node (`fixed=true`), follow cursor, unpin on mouseup.
- Click (no drag): `selectResource(id); setRightOpen(true)` / `setSelectedCat(catId)`.
- Pan: drag empty space → translate `tx/ty`.
- Zoom: wheel × 1.1, anchored at cursor. Clamp `scale ∈ [0.3, 2.6]`.
- FIT button: compute bounds + 70px pad, clamp `[0.3, 1.8]`, center.
- 100% button: `scale=1`, recenter.

**Overlays (in same component):**
- HUD top-left: `NODES n · EDGES n · ZOOM n%`.
- Zoom controls bottom-right: `−` / `FIT` / `100%` / `+` pill.
- LIST overlay (`view==='list'`): opaque panel, resource rows.
- TIMELINE overlay (`view==='timeline'`): grouped by month, resource rows.

**Future:** 3D mode is a planned drop-in. The sim node/link data structures will be reused; only the renderer changes. When implementing, extract sim logic into a separate `useSim` hook so 2D/3D renderers can both consume it.

### Dock (`ChatDock.tsx`)

Bottom of center column. Replaces current full-page ChatDock.

**Collapsed (42px):** tab strip only — `grid` Categories toggle · pipe · `CHAT` `TASKS` `LIBRARY` tabs · spacer · `chevDn`/`chevUp` arrow. Active tab: `bg:#15151A; 2px accent bottom-border`.

**Expanded (264px):** tab strip + body.

Tab bodies:
- **Categories**: derive unique categories from `useResourceStore.resources`. Horizontal scroll of 184px cards — swatch + name + big count metric + dot-matrix fill meter. Trailing dashed `+ NEW` card.
- **Chat**: scrollable thread (user bubble right accent-soft, AI bubble left `#131318`) + composer (`plus` + input + accent `send`). Uses `useChatStore`.
- **Tasks**: mini-kanban — `IN PROGRESS` / `TO DO` / `DONE` columns (240px). Cards: checkbox + title + category dot + `DUE …`. Uses `useTaskStore`.
- **Library**: `RECENT · N ITEMS` + `SORT: RECENT ▾`. 36px rows `[type badge] title · category · date`. Uses `useResourceStore.resources` filtered by `filters.query`.

### Inspector / Right Rail (`ResourceForm.tsx`)

Replaces the add/edit form. Read-mostly view of selected resource.

**Collapsed (56px):** `chevR` expand toggle + selected resource's category color swatch.

**Expanded (336px):**
- Header (52px): `INSPECTOR` + `chevR` collapse.
- Preview well (96px): dot-grid inset, large type label in type color, host top-left, category swatch top-right.
- Details: title (14px) · type badge + `ADDED <date>` · tag chips (categoryName + tags).
- Quick actions 2×2: `OPEN` `EDIT` `LINK` `DELETE`.
- Connections (`CONNECTIONS · 0`): empty section — deferred until graph service exposes relation data.
- AI summary: `AI SUMMARY` label + `resource.summary` paragraph + 3 suggested-question chips (`›`). Click chip → fills footer input + sends to chat.
- Footer: `ASK ABOUT THIS RESOURCE…` input + accent `send`. Send → `useChatStore.sendMessage`.

---

## ResourceType — Client-Side Inference

New field added to `ResourceItem`. Inferred in `normalizeResource` from URL since the Go backend does not yet expose a type field.

```ts
function inferResourceType(url: string, userOverride: boolean): ResourceType {
  if (userOverride) return "note";
  const lower = url.toLowerCase();
  if (lower.endsWith(".pdf") || lower.includes("/pdf/")) return "pdf";
  if (lower.match(/\.(png|jpe?g|gif|webp|svg)(\?|$)/)) return "image";
  if (lower.match(/\.docx?(\?|$)/)) return "doc";
  return "link";
}
```

Future: when the Go `Resource` struct exposes a `type` field, wire it through `normalizeResource` and remove the inference fallback.

---

## Files Touched

| File | Change |
|---|---|
| `frontend/index.html` | Add JetBrains Mono Google Fonts link |
| `frontend/src/types.ts` | Add `ResourceType`, `DockTab`, `GraphView` types; add `type: ResourceType` to `ResourceItem` |
| `frontend/src/api/client.ts` | Add `inferResourceType()`; wire into `normalizeResource` |
| `frontend/src/stores/useLayoutStore.ts` | Rewrite — clean UIState replacing old `sidebarCollapsed`/`activeSection` |
| `frontend/src/styles.css` | Full rewrite — CSS tokens, shell layout, all component styles |
| `frontend/src/App.tsx` | Rewrite — 3-zone shell (`<Sidebar/> <center-col> <Inspector/>`) |
| `frontend/src/components/layout/Sidebar.tsx` | Rewrite — Left Rail expanded/collapsed |
| `frontend/src/components/layout/Topbar.tsx` | Rewrite — Top Bar with filter + view switch |
| `frontend/src/components/graph/GraphCanvas.tsx` | Rewrite — custom canvas 2D physics sim |
| `frontend/src/components/chat/ChatDock.tsx` | Rewrite — Dock (tab strip + 4 tab bodies) |
| `frontend/src/components/resource/ResourceForm.tsx` | Rewrite — Inspector right rail |
| `frontend/src/components/icons.tsx` | New — pixel icon helper `px()` + all 15 icon exports |
| `frontend/src/components/layout/Sidebar.test.ts` | Mark `describe.skip` — re-calibrate after design stabilizes |
| `frontend/src/components/layout/Topbar.test.ts` | Mark `describe.skip` |
| `frontend/src/components/graph/GraphCanvas.test.ts` | Mark `describe.skip` |
| `frontend/src/components/graph/GraphControls.test.ts` | Mark `describe.skip` |
| `frontend/src/components/chat/ChatDock.test.ts` | Mark `describe.skip` |
| `frontend/src/components/tasks/TaskBoard.test.ts` | Mark `describe.skip` |
| `frontend/src/components/resource/ResourceForm.test.ts` | Mark `describe.skip` |
| `frontend/src/components/resource/ResourceList.test.ts` | Mark `describe.skip` |
| `frontend/src/components/settings/SettingsPanel.test.ts` | Mark `describe.skip` |

**Not touched:** `api/client.ts` (beyond `inferResourceType`), all Zustand stores (resource/task/chat/sync), Go backend, IPC bindings (`wailsjs/`), `useFileDrop.ts`, `useFilteredResources.ts`, `useSyncStore.ts`.

---

## Architectural Constraints

- This is Wails desktop app — all data flows through `ipc.ts` bridge (IPC when `window.go` present, REST fallback in browser). No change to transport layer.
- `react-force-graph-2d` and `react-force-graph-3d` remain in `package.json` for future 3D mode. GraphCanvas no longer imports them.
- `GraphControls.tsx` is no longer imported. Not deleted — left as stub until test suite is re-calibrated.
- `SettingsPanel.tsx` and `TaskBoard.tsx` remain unchanged — used as-is inside the Dock's tasks tab and gear-icon popover.
- `ResourceList.tsx` remains unchanged — used inside the Dock's library tab.

---

## Workstreams

### WS1 — Foundation (tokens, types, store)
- [ ] `index.html` — JetBrains Mono font
- [ ] `types.ts` — `ResourceType`, `DockTab`, `GraphView`, `type` on `ResourceItem`
- [ ] `api/client.ts` — `inferResourceType` + wire into `normalizeResource`
- [ ] `useLayoutStore.ts` — clean UIState
- [ ] `styles.css` — full CSS rewrite (tokens + shell + all component styles)

### WS2 — Shell + Left Rail
- [ ] `App.tsx` — 3-zone shell
- [ ] `icons.tsx` — pixel icon helper + 15 icon exports
- [ ] `Sidebar.tsx` — Left Rail (expanded + collapsed states)

### WS3 — Center Zone
- [ ] `Topbar.tsx` — Top Bar
- [ ] `GraphCanvas.tsx` — custom canvas 2D physics sim + zoom controls + overlays

### WS4 — Dock + Right Rail
- [ ] `ChatDock.tsx` → Dock (tab strip + Categories/Chat/Tasks/Library bodies)
- [ ] `ResourceForm.tsx` → Inspector (preview + details + connections stub + AI summary)

### WS5 — Test Cleanup
- [ ] Mark all component `.test.ts` files as `describe.skip`
- [ ] Verify `npm run build` TypeScript compiles clean
- [ ] Verify `go test ./...` unaffected (presentation-only change)

---

## Done Criteria

- [ ] `wails dev` launches showing 3-zone layout with JetBrains Mono + orange accent
- [ ] Left rail collapses/expands (264px ↔ 56px)
- [ ] Right rail opens on graph node click (shows inspector)
- [ ] Dock tab strip visible at bottom; expands to 264px on tab click
- [ ] Graph canvas renders category hubs + resource satellites as pixel squares
- [ ] Drag, pan, zoom all functional on graph canvas
- [ ] Filter input in top bar dims non-matching graph nodes
- [ ] GRAPH / LIST / TIMELINE switch changes canvas overlay
- [ ] Chat tab in dock sends/receives messages via `useChatStore`
- [ ] Tasks tab in dock shows todos from `useTaskStore`
- [ ] Library tab in dock lists resources from `useResourceStore`
- [ ] `npm run build` exits clean (TypeScript errors = 0)
- [ ] `go test ./...` exits clean (no regressions)
- [ ] Existing Zustand store tests (`useResourceStore.test.ts`, `useTaskStore.test.ts`) still pass

---

## Open Items (deferred)

- **Inspector connections section**: empty for now — needs graph service to expose resource-to-resource relation data (`con[]` in spec §8).
- **ResourceType on backend**: Go `Resource` struct doesn't expose type field yet. Client infers from URL. Wire when backend adds it.
- **3D graph mode**: sim data structures designed for future 3D renderer drop-in. Extract `useSim` hook when implementing.
- **Dot-matrix meters in left-rail category list**: stub as plain bars initially; implement full meter once categories section is re-added (currently removed per Minimal_UI_Spec §6.8).
- **Global capture/ask input**: Command_Bar removed per spec §3. Re-site in future session.
- **Multi-thread chat history**: `New_Chat_Button` destination not yet designed.
- **Split_Section resize handle**: user-adjustable dock height deferred.
