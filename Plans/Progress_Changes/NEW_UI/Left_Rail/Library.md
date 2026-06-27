# Left Rail — LIBRARY (nav row + Library rail view)

Source: nav row `dc.html` 86–94; library view 317–344.

## A. Primary-nav row (home view)
Same row spec as Chats §A. This row:
- Icon `library` (3 horizontal bars) cells: `[1,1][2,1][3,1][4,1][5,1][1,3][2,3][3,3][4,3][5,3][1,5][2,5][3,5][4,5][5,5]`. Colour `#7A7A84`.
- Label `LIBRARY`, `flex:1; font-size:11.5px; letter-spacing:.4px`, left-aligned.
- `↗` affordance (hover-only) → `nav.openDock` (note: design wires this to `openCatTab` i.e. opens dock; in build → Library/categories dock tab).
- Row `onClick → nav.open` → `leftView:'library'`.

## B. Library rail view (`leftView==='library'`, `isLibView`)
Body `flex:1; display:flex; flex-direction:column; min-height:0`.

### Head bar (`height:40px; padding:0 10px; border-bottom:1px solid #1A1A1F`)
- Back (chevL) → `backHome`. `24×24 #7A7A84`.
- Title "LIBRARY": `flex:1; font-size:10px; letter-spacing:1px; color:#9A9AA0`.
- Open-in-dock `↗` button `24×24; border:1px solid #26262C; color:#7A7A84` → `openCatTab`.

### Filter chips row (`flex-none; padding:10px 10px 8px; display:flex; flex-wrap:wrap; gap:5px; border-bottom:1px solid #16161A`)
- Chips: `padding:0 9px; height:24px; font-size:9px; letter-spacing:.5px; border:1px solid #25252B`.
  - Active: `background:{ch.bg}` (accent-tinted) `color:{ch.fg}`.
  - Inactive: `background:#15151A` `color:#9A9AA0` (approx).
- Chip set: `ALL` (default active), `PDF`, `LINK`, `NOTE`. `onClick → ch.set` sets `libFilter`.

### Sort row (`flex-none; padding:8px 12px 4px; display:flex; justify-content:space-between`)
- Left: `{libFilterCount} ITEMS` — `font-size:9px; letter-spacing:.5px; color:#5C5C66`.
- Right: `SORT: RECENT ▾` — `font-size:9px; letter-spacing:.5px; color:#7A7A84`.

### List (`flex:1; overflow-y:auto; padding:0 8px 8px`)
- Row (`ss-libitem`): `display:flex; align-items:center; gap:10px; height:34px; padding:0 8px; cursor:pointer; background:{r.bg}`. Hover `background:#131318`.
  - **type badge** `18×18; background:{r.color}; color:#0B0B0D; font-size:7px; font-weight:700; flex-centre` — text `{r.typeLabel}` (PDF/LINK/NOTE/DOC/IMAGE).
  - title `flex:1; min-width:0; font-size:11px; ellipsis`.
  - meta (`ss-lmeta`) date `font-size:8.5px; color:#5C5C66; flex:none` — hidden on hover.
  - remove (`ss-lx`) `16×16; color:#7A7A84` hover `#E06C75`, icon `cross`. Shown on hover (replaces date). `onClick → r.remove`. Title "Remove from library".
- `r.select` → selects resource + opens inspector.

## Implemented (this pass) — Fix 1: per-row remove
- `useResourceStore`: `removedLib: Record<string,boolean>` + `removeFromLibrary(id)` (design `removeLib`). Library lists filter `!removedLib[id]`.
- Rail Library rows (`ss-libitem`): hover hides date (`ss-lmeta`), reveals `ss-lx` ✕ → `removeFromLibrary`. Same applied to dock Library tab.
- CSS: `.ss-libitem:hover .ss-lmeta{display:none}`, `.ss-libitem:hover .ss-lx{display:flex}`, `.ss-lx` hover `#E06C75`.
- Verified live: hover reveals ✕; click removes (21→20) from rail + dock.

## Deferred — Fix 2 (needs GRAPH)
- Design also drops removed items from the **graph** (`removedLib` filtered in graph build, dc.html line ~1368). Wire once the GRAPH is reworked. Type badge already labelled (build uses 4-char type badge ✓).
