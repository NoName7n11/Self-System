# Main_Content — Map View (TASK MAP)

Source: `dc.html` 423–456 (markup), 1543–1570 (mapLayout), 1538 (centerMap on enter).
**This is a TASK MAP** — a horizontal mind-map of *tasks* grouped by category, NOT a resource graph. Shown when `view==='map'` (`isMapView`). topTitle "TASK MAP".

## Tree model (`mapLayout`)
3 depths:
- **root** (depth 0): label "TASKS", color `#F0703C`, kind `root`, always expanded.
- **category** (depth 1): one per category that has tasks; label = category name, color = category colour, `count` = #tasks, kind `cat`, expandable.
- **task** (depth 2): label = task title, color = **status colour** (`done #48C78E` / `in_progress #F0703C` / open `#7A7A84`), `due`, kind `task`.

Layout constants: `COL = 232` (x = depth·COL), `ROW = 44` (vertical step). Node widths `NW = { root:150, cat:184, task:210 }`. Expanded parent's y = midpoint of its children. Output `{ nodes, edges, w:maxX+60, h:maxY+60, minY }`.

## Viewport
- Outer: `position:absolute; inset:0; overflow:hidden; cursor:grab;` same dot-grid bg (`#161619 1px / 24px`).
- `onMouseDown → mapPanStart`, `onWheel → mapWheel`. Pan = `state.mapPan {x,y}`, zoom = `state.mapZoom`.
- Inner transform layer: `width:{mapW}; height:{mapH}; transform:{mapTransform}; transform-origin:0 0; will-change:transform`.
- On entering map view → `centerMap()` (rAF).

## Edges (SVG)
- `<svg>` absolute, `overflow:visible; pointer-events:none`.
- Each edge: cubic bezier from parent right edge `(a.x+NW, a.y)` to child left `(b.x, b.y)`, control points at midpoint-x:
  `M{ax},{ay} C{mx},{ay} {mx},{by} {bx},{by}`. `stroke:{color}; stroke-width:1.4; stroke-opacity:0.5`.

## Nodes
- Each: `position:absolute; left:{n.left}; top:{n.top}; width:{n.w}px; height:32px; margin-top:-16px; box-sizing:border-box; display:flex; align-items:center; gap:8px; padding:0 10px; background:{n.bg}; border:1px solid {n.bd}; cursor:{n.cursor}.` Hover `border-color:#F0703C`.
  - `8×8` colour dot `{n.color}`.
  - label `flex:1; min-width:0; font-size:11px; letter-spacing:.2px; color:{n.fg}; text-decoration:{n.strike}; ellipsis` (strike when task done).
  - count (if `n.count`): `font-size:9px; color:#5C5C66`.
  - expand chevron (if `n.expandable`): `display:flex; color:#9A9AA0` — `{n.chev}` (toggles `mapExpanded[id]`).
- Task node click → select / open the task (per build choice).

## Floating controls (bottom-right)
- Column, `right:14px; bottom:14px; gap:8px; align-items:flex-end`.
- **EXPAND ALL** toggle: `display:flex; gap:7px; height:30px; padding:0 12px; color:#F0703C; background:#121215; border:1px solid #2E2E36`, hover `border #F0703C`. icon `grid` + `{mapAllLabel}` ("EXPAND ALL"/"COLLAPSE ALL"). `onClick → mapToggleAll`.
- **Zoom pill**: `background:#121215; border:1px solid #26262C; display:flex`. Buttons `−` (`32×30`) / `FIT` / `{mapZoomPct}` (label) / `+`; border-right `#1F1F25`; hover `color:#fff; background:#17171B`. → `mapZoomOut/mapFit/mapZoomIn`.

## Build notes / current gaps
- **Build MAP is wrong** — it shows category→resources. Must rebuild as the **task tree** (root→cat→task), horizontal bezier edges, expandable nodes, pan/zoom viewport, EXPAND ALL + zoom pill. topTitle "TASK MAP" + task-count subtitle.
