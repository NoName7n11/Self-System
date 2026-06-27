# Main_Content — Progress View

Source: `dc.html` 458–499 (markup), 1578–1605 (processing model). Shown when `view==='progress'` (`isProgressView`). topTitle "PROCESSING".

## Overlay
- `position:absolute; inset:0; background:#0B0B0D; overflow-y:auto; padding:16px 20px.`

## Section 1 — PROCESSING QUEUE
- Label: `font-size:9px; letter-spacing:1.5px; color:#5C5C66; margin-bottom:12px` — "PROCESSING QUEUE".
- Cards container `display:flex; flex-direction:column; gap:10px; margin-bottom:26px`.
- Card (`state.processing[]`): `background:#101014; border:1px solid #22222A; padding:13px 14px`.
  - Row 1 `display:flex; align-items:center; gap:10px; margin-bottom:10px`:
    - type chip `20×20; background:{p.color}; color:#0B0B0D; font-size:7px; font-weight:700; flex-centre` = `{p.typeLabel}`.
    - label `flex:1; min-width:0; font-size:12px; color:#E9E9EC; ellipsis`.
    - stage `font-size:9px; letter-spacing:1px; color:#F0703C` = `{p.stage}` (QUEUED→FETCHING→EXTRACTING→EMBEDDING→CLASSIFYING).
    - pct `font-size:10px; color:#7A7A84; width:34px; text-align:right` = `{p.pct}`.
  - Progress bar: `height:6px; background:#1A1A1F; overflow:hidden` → fill `height:100%; width:{p.pct}; background:#F0703C; transition:width .4s`.
  - Row 3 `display:flex; align-items:center; gap:6px; margin-top:8px`: `5×5` `{p.catColor}` dot + "CLASSIFYING → {p.catName}" `font-size:9px; color:#5C5C66`.
- **Empty** (`progressEmpty`, default true): `border:1px dashed #26262C; padding:28px; text-align:center; font-size:10px; letter-spacing:.5px; color:#3C3C44` — "NO ACTIVE PROCESSING · ADD A RESOURCE FROM THE LIBRARY TAB".

## Section 2 — RECENTLY COMPLETED
- Label: same style, `margin-bottom:10px` — "RECENTLY COMPLETED".
- Rows container `display:flex; flex-direction:column; gap:1px`.
- Row (`state.doneProcessing[]`, `d.select`): `display:flex; align-items:center; gap:12px; height:38px; padding:0 10px; cursor:pointer; border:1px solid transparent`. Hover `background:#131318; border-color:#1F1F25`.
  - `14×14` green `#48C78E` `✓` chip.
  - `20×20` type chip `{d.color}` `{d.typeLabel}`.
  - label `flex:1; font-size:12px; color:#E9E9EC; ellipsis`.
  - "{d.catName} · IN GRAPH" `font-size:9px; color:#5C5C66; flex:none`.
  - select → `selectedId`, open inspector, `view:'graph'`.
- **Empty** (`doneEmpty`, default true): `font-size:9px; letter-spacing:.5px; color:#3C3C44; padding:10px` — "Nothing processed yet this session."

## Data flow (live)
- Adding a resource via the dock ingest command bar creates a processing item → `runProcessing` ticks stages every 460ms → on 100% `completeProcessing` adds the resource as a graph node (green new-node ring), pushes a notification, and prepends to `doneProcessing` (cap 6) + `recentIds` (cap 8).
- Default state: both lists empty → both empty states show.

## Build notes / current gaps
- Build PROGRESS shows an empty queue + "recently completed" using recent resources — design's "recently completed" is **only** session-completed items (empty by default). Align: empty-state strings exact; processing cards styled per above; wire to ingest pipeline (deferred until ingest is built).
