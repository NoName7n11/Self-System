# Updated UI — Instrument Redesign (v2, authoritative)

Date: 2026-06-27
Status: In Progress — full rebuild (Change 18)
Supersedes: v1 (Change 17), which was built from the reduced `Minimal_UI_Spec.md` and is visually incomplete.

**Authority:** `Redesign from scratch_7/Self Systems.dc.html` (rendered prototype) + `Self_Systems_UI_Build_Spec.md`. When in doubt, the dc.html behavior wins — it is richer than the spec text.

Scope: presentation-layer only. Go backend, Wails IPC, Zustand store *shapes*, and `api/client.ts` contract unchanged. React/TypeScript/.tsx.

---

## Why the rebuild

Change 17 followed `Minimal_UI_Spec.md` (a reduced Figma cut). The real design (`_7/dc.html`) is a far more complete app. Two root problems with v1:
1. **Empty.** v1 is backend-driven → no data → barren. The design ships a seed dataset and always looks full.
2. **Missing features.** v1 dropped the command bar, left-rail categories, multi-view top bar, notifications, inline previews, and the rail state machine.

**Decision (this change):** frontend fallback seed (`src/lib/demoData.ts`) shown when backend empty + re-baseline every component on the dc.html.

---

## Design tokens

Unchanged from v1 (already correct in `styles.css`):
- Font: JetBrains Mono only (400–800). Body `letter-spacing:-0.1px`; labels uppercase, `+0.5–1.5px`.
- Accent `#F0703C`. `border-radius:0` everywhere. Borders `1px solid` (`1px dashed` only for `+ NEW`).
- Backgrounds: app `#0B0B0D`, panel `#0E0E11`, card `#131318`, input `#15151A`.
- Category palette: research `#5B9CF6`, ai `#A98BF5`, finance `#48C78E`, people `#E5B567`, sources `#56B6C2`, archive `#E06C75`.
- Type palette: pdf `#F67373`, link `#48C78E`, note `#5B9CF6`, doc `#9B59F6`, image `#F6739B`.
- Texture: dot-grid canvas (`radial-gradient #161619 1px / 24px`), dot-matrix meters (`#202026 / 6–7px` track + category-color fill).
- Pixel icons: 7×7 grid SVG, 2×2 rect/cell (existing `icons.tsx`). **Add:** cross, dots(⋯), pencil, trash, archive, bell, gear2.

---

## Layout — 3-zone shell (unchanged skeleton)

```
┌── LEFT RAIL ──┬──────── CENTER ────────┬── RIGHT RAIL ──┐
│ 264 (⇄56)     │ TOP BAR (52)           │ 336 (⇄56)      │
│ STATE MACHINE │ GRAPH / MAP / PROGRESS  │ INSPECTOR      │
│ home/chat/    │ (canvas zone, flex)     │                │
│ tasks/library ├────────────────────────┤                │
│               │ DOCK (264 ⇄ 42)         │                │
│               │ cats·chat·tasks·lib·arch│                │
└───────────────┴────────────────────────┴────────────────┘
```
All three panels resizable via drag handles (defer to later phase). Widths persisted to localStorage in the design (defer).

---

## State model (full)

Extend `useLayoutStore`:

```ts
leftCollapsed: boolean        // false
rightOpen: boolean            // true (design default open w/ r1 selected)
dockOpen: boolean             // true
dockTab: 'categories'|'chat'|'tasks'|'library'|'archive'  // 'categories'
view: 'graph'|'map'|'progress'   // 'graph'   (NOT list/timeline)
leftView: 'home'|'chat'|'tasks'|'library'   // 'home'  ← rail state machine
selectedCat: string|null
recentOpen, catsOpen: boolean
libFilter: 'all'|'pdf'|'link'|'note'|'doc'
archiveFilter: string
notifOpen, notifSeen, notifMuted: boolean
```
`selectedId` + `query` stay in `useResourceStore` (shared filter). `conversations` + active conv ids — new lightweight chat state (or extend `useChatStore`).

---

## Component specs (re-baselined on dc.html)

### Left rail — STATE MACHINE (was: static nav)
The rail body swaps by `leftView`. Nav row click → sets `leftView`; the `↗` (trend) affordance → opens the **dock** tab instead.

**home:** search · primary nav (CHAT/TASKS/LIBRARY rows, each w/ hover `↗`) · **RECENT** (collapsible, 5 items, hold-to-clear) · **CATEGORY NODES** (collapsible; when a cat selected shows its nodes + `VIEW ALL →`, else hint) · spacer · **UPDATE AVAILABLE · v0.1.1** banner · footer (avatar/name/gear).
**chat:** conversation list (NEW CHAT, dot+title+preview, ⋯ menu) → thread (back, title, ↗-to-dock, messages w/ cite chips + attach chips, composer w/ attach).
**tasks:** back/TASKS/NEW · vertical grouped columns (IN PROGRESS/TO DO/DONE) · cards: checkbox + title(strike) + cat dot + due.
**library:** back/LIBRARY/↗ · type filter chips (ALL/PDF/LINK/NOTE) · `N ITEMS` + SORT · rows (18px type badge + title + date, hover ×remove).
**collapsed (56px):** logo + 3 nav icons + gear.

### Top bar (52px)
Title `KNOWLEDGE GRAPH` + `128 RESOURCES · 342 CONNECTIONS` · spacer · **notifications bell** (dot when unseen; dropdown: list, mute, clear) · **view switch GRAPH / MAP / PROGRESS** (active bg `#15151A`/`#22222A`, fg accent).

### Center canvas zone (by `view`)
- **graph:** custom canvas 2D sim (already built in v1 — keep). Hubs = pixel squares + punched center + label; resources = filled squares; selected = accent ring; dim non-matches α.22. Zoom pill `− / FIT / 100% / +`.
- **map:** mind-map overlay — category nodes as labelled boxes, expandable to children, SVG edges, own pan/zoom + `EXPAND ALL`/zoom pill. (Phase 5)
- **progress:** PROCESSING QUEUE (cards w/ progress bar + stage + `CLASSIFYING → cat`) + RECENTLY COMPLETED rows. Empty states. (Phase 5)

### Dock (264 ⇄ 42)
Tab strip: `▦ CATEGORIES` toggle · divider · CHAT/TASKS/LIBRARY tabs (active 2px accent underline) · spacer · expand/collapse arrow. (Archive reachable via tab set too.)
- **categories:** LEFT = ingest area (`INGEST → AUTO-CLASSIFY`, command bar `PASTE A URL OR TYPE A NOTE…` + attach `+` + orange `ADD` + caret menu [best-match / new category]; drop-to-ingest; hint line). RIGHT (248px) = vertical category cards (swatch + name + big count + dot-matrix meter) + dashed `NEW`.
- **chat:** selected-conversation thread (title bar + messages w/ cite/attach + composer `ASK ABOUT YOUR KNOWLEDGE GRAPH…`).
- **tasks:** NEW TASK (+ create form: title/category chips/status/due) · horizontal-scroll columns (IN PROGRESS/TO DO/DONE) · cards w/ checkbox + title + cat dot + due + T/P/D status segments + ⋯ menu.
- **library:** `RECENT · N ITEMS` + SORT · rows (type badge + title + cat·date + hover ×).
- **archive:** `ARCHIVE · N` + filter chips · rows (type badge + title + kind·sub + RESTORE) · empty state.

### Right rail — Inspector (336 ⇄ 56)
Header `INSPECTOR` + collapse. Body: **preview well** (96px, 34px type label in type color, host TL, cat swatch TR) · **details** (title 14px · type badge + `ADDED date` + `×counter` accent dot · cat chip + tag chips) · **quick actions 2×2** (PREVIEW/OPEN/EDIT or ARCHIVE/DELETE — design uses PREVIEW·EDIT·ARCHIVE·DELETE for files, OPEN for links) · **CONNECTIONS · n** (swatch + title + rel label CITES/RELATED/REF BY/…) · **AI SUMMARY** (text + 3 suggested-Q chips `›`). **Inline preview** replaces AI summary when PREVIEW active (paged/link-frame/note/image variants — Phase 4/5). Collapsed strip: chevron + selected cat swatch.

### Overlays (Phase 5+, document now)
- Conversation context menu (rename/open-in-dock/archive/delete).
- Task context menu (archive/delete).
- AI clarification popup (bottom 30% — clarify title/body + similar-resource matches).
- Notifications dropdown.

---

## Seed dataset (implemented: `src/lib/demoData.ts`)
6 categories, 21 resources (r1–r21, 2 archived), 6 tasks, 3 conversations, 4 notifs, recent = [r2,r1,r8,r19,r11], default selected = r1. Category-category links for graph hub edges. Adapters map demo shapes → store shapes (`demoResourcesAsItems`, `demoTasksAsTodos`, `demoChatMessages`).

**Wiring:** stores load backend first; if backend returns empty/errors → fall back to demo data. Real data always wins when present.

---

## Files

| File | Change |
|---|---|
| `src/lib/demoData.ts` | ✅ NEW — seed dataset + adapters |
| `src/types.ts` | DockTab += `archive`; GraphView = `graph/map/progress`; add `LeftView` |
| `src/stores/useLayoutStore.ts` | full state model above |
| `src/stores/useResourceStore.ts` | fallback to demo when backend empty |
| `src/stores/useTaskStore.ts` | fallback to demo when backend empty |
| `src/stores/useChatStore.ts` | conversations + fallback |
| `src/components/icons.tsx` | + cross, dots, pencil, trash, archive, bell, gear2 |
| `src/components/layout/Sidebar.tsx` | rail state machine (home/chat/tasks/library + collapsed) |
| `src/components/layout/Topbar.tsx` | GRAPH/MAP/PROGRESS + notifications |
| `src/components/graph/GraphCanvas.tsx` | keep sim; add MAP + PROGRESS overlays |
| `src/components/chat/ChatDock.tsx` | dock: ingest command bar + cat cards + 5 tabs |
| `src/components/resource/ResourceForm.tsx` | inspector richness + inline preview |

---

## Phase plan → see `Plans/Progress_Changes/Change_18_Workstream.md`

P1 state + seed wiring · P2 left-rail state machine + icons · P3 top-bar views + notifications + dock ingest/archive · P4 inspector richness · P5 MAP + PROGRESS + overlays.

## Deferred (documented, lower priority)
Resize drag handles + localStorage persistence · conversation rename/menu · task create form + ⋯ menu + T/P/D segments · drag-drop file attach/ingest · AI clarification popup · inline preview per-type rendering · add-menu (best-match/new category) · hold-to-clear recent.
