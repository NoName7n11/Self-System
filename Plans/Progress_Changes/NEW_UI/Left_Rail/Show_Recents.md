# Left Rail — SHOW RECENTS

Source: `dc.html` 96–122. Home view only. Collapsible. Hold-to-clear-all.

## Divider above
- `height:1px; background:#1A1A1F; margin:8px 12px` (between nav and RECENT).

## Section header (toggle + hold-to-clear)
- Wrapper: `position:relative; padding:2px 12px 4px; display:flex; align-items:center; gap:6px; cursor:pointer; color:#5C5C66; user-select:none; overflow:hidden`. Hover `color:#9A9AA0`.
- Handlers: `onMouseDown → startHoldRecent`, `onMouseUp → endHoldRecent`, `onMouseLeave → cancelHoldRecent`. Title "Tap to collapse · hold to clear all".
  - **Tap** = toggle `recentOpen`. **Hold** = animate fill then clear all recents.
- **Hold fill bar** (`recentFillRef`): `position:absolute; top:0; left:0; bottom:0; width:0%→100%; background:rgba(240,112,60,0.38); pointer-events:none` (grows during hold).
- Chevron (`recentChev`): `chevDn` when open, `chevR` when closed. `display:flex; flex:none; position:relative`.
- Label "RECENT": `font-size:9px; letter-spacing:1.5px; flex:1; position:relative`.
- Count (`recentCount`): `font-size:9px; color:#3C3C44; position:relative` (e.g. "5").

## Items list (when `recentOpen`)
- Container: `padding:0 8px; display:flex; flex-direction:column; gap:1px`.
- Row (`ss-recent`): `display:flex; align-items:center; gap:10px; height:30px; padding:0 8px; cursor:pointer; border:1px solid transparent`. Hover `background:#15151A`.
  - **swatch** `9×9; background:{r.color}` (resource **type** colour: pdf #F67373, link #48C78E, note #5B9CF6, doc #9B59F6, image #F6739B).
  - **title** `flex:1; min-width:0; font-size:11px; color:#B9B9C0; ellipsis`, left-aligned.
  - **type label** (`ss-rtype`) `font-size:8px; letter-spacing:.5px; color:#4E4E57; flex:none` — e.g. LINK/PDF/NOTE/DOC. **Hidden on row hover.**
  - **remove** (`ss-rx`) `16×16; color:#7A7A84` hover `#E06C75`, icon `cross`. **Shown on hover** (replaces type label). `onClick → r.remove`. Title "Remove from recent".
- Empty (`recentEmpty`): "NO RECENT ITEMS" `font-size:9px; letter-spacing:.5px; color:#3C3C44; padding:8px; text-align:center`.

## Default data
- `recentIds = ['r2','r1','r8','r19','r11']` → RAG Tutorial (LINK), Raft Consensus Paper (PDF), GBUS Design Notes (NOTE), Hackathon 2026 (LINK), Q3 Roadmap (DOC). Cap **5**.
- These are recently *opened/accessed* resources (shortcut list, not full history).

## Interaction
- Row click → `r.select` → select resource + open inspector + `view:'graph'`.

## Implemented (this pass) — per-row remove + cap/scroll
- `useResourceStore`: `recentIds` (seed `DEMO_RECENT_IDS`) + `removeRecent(id)` + `pushRecent(id)` (cap **10**, `RECENT_CAP`).
- Rows (`ss-recent`): hover hides type label (`ss-rtype`), reveals `ss-rx` ✕ → `removeRecent`. CSS swap like library.
- **Cap behaviour:** list (`.rail-recent-list`) `max-height:154px` (~5 rows × 30px + gaps) + `overflow-y:auto` → **≤5 static, >5 scrolls**. Holds up to 10 ids.
- Verified live: remove 5→4; max-height 154px; static at 5, scrolls above 5.

## Deferred (documented): hold-to-clear-all (press-and-hold fill bar) — design `startHoldRecent`/`clearRecent`.
