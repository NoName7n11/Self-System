# Main_Content — Graph View (force-directed canvas sim)

Source: `dc.html` 419–509 (markup), 1488–1527 (setup/mount), 1682–1791 (step/draw), 1713–1732 (dimming), 501–508 (zoom controls).

## Canvas zone
- `flex:1; min-height:0; position:relative;`
- `background-color:#0B0B0D; background-image:radial-gradient(circle,#161619 1px,transparent 1.4px); background-size:24px 24px.`
- `<canvas>`: `position:absolute; inset:0; width:100%; height:100%; display:block; cursor:grab; pointer-events:{graph?auto:none}.`
- Renders at `devicePixelRatio`. ResizeObserver → `resize()`.

## Nodes
- **Category hub**: `{ kind:'cat', mass:4, r:13, color:category }`. 6 hubs.
- **Resource**: `{ kind:'res', mass:1, r:5 + counter*0.7, color:typeColor, arch }`. 21.
- Seed: hubs on circle radius **170** (`cos/sin(i/N·2π)`); resources at hub ± **55** random angle.

## Links
| kind | len | k | style |
|---|---|---|---|
| resource → its category | 90 | 0.04 | strong, solid, category colour |
| category ↔ category (`CATLINKS`, 7) | 230 | 0.02 | weak, dashed `#3A3A42` |
| resource ↔ resource (`con[]`, dedup a<b) | 78 | 0.015 | weak, dashed `#2E2E36` |

## Per-tick forces (`step`)
```
repulsion (all pairs): f = charge / d²   charge = 26000 if either node is cat, else 9000
spring (each link):    f = (d − len) · k
gravity (per node):    v += −pos · g      g = 0.004 (cat) | 0.012 (res)
fixed (dragged):       v = 0
integrate:             v *= 0.9; clamp |v| ≤ 14; pos += v
```
On mount: run **260** silent steps, then `doFit()` at **60 / 360 / 900ms**. The rAF loop only steps+draws when `view==='graph'`.

## Draw
- Transform `w2s = world*scale + t`. `setTransform(dpr,...)`.
- **Edges first**: strong `lineWidth 1.1` solid α `0.32`; weak `lineWidth 1` dashed `[2,3]` α `0.6`. Both endpoints dimmed → strong α `0.04`, weak α `0.03`.
- **Category node**: filled pixel square size `max(10, r*scale)` in colour; punch a centred `#0B0B0D` square at `0.4×` size; label below (`+13px`) `700 10px`, colour `#E9E9EC` α `0.92` (dim `0.3`). Label **always** shown.
- **Resource node**: filled square size `max(4, r*scale*0.95)`.
  - selected → accent stroke ring at `(s+4)` `lineWidth 1.5` + accent fill α `0.12`.
  - archived → grey `#5C5C66`.
  - hovered → full alpha.
  - label below (`+11px`) `500 9px` `#C9C9CF` only when selected / hovered / `scale > 1.35`.
  - new node → expanding green `#48C78E` ring animation for 3s.
- **HUD** (top-left, `14,20`): `500 9px` colour `#46464E` — `NODES n · EDGES n · ZOOM n%`.

## Dimming (`matchSet`)
- If `query`: match resources by title / cat / tags, and categories by name/id → non-matches (and resources whose cat isn't matched) drop to α `0.22`.
- Else if `selectedCat`: dim nodes not in that category.

## Interaction
- **Hit-test** world radius + slop → **drag** pins node (`fixed`), follows cursor; release → if not moved, **select** (`selectedId`, open inspector). 
- **Pan**: drag empty space → translate.
- **Zoom**: wheel ×1.1 anchored at cursor; clamp scale `[0.3, 2.6]`.
- Hover sets `hoverId` (brighter + label).

## Zoom controls (graph view only)
- `position:absolute; right:14px; bottom:14px; background:#121215; border:1px solid #26262C; display:flex.`
- Buttons (border-right `#1F1F25` between; hover `color:#fff; background:#17171B`):
  - `−` (`32×30`) → `zoomOut`.
  - `FIT` (`padding:0 10px; height:30px; 9.5px`) → `fit`.
  - `100%` → `reset` (scale 1, recenter).
  - `+` (`32×30`) → `zoomIn`.

## Build notes / current gaps
- Build sim matches these forces/draw closely ✓ (carried from Change 17). Verify: label fonts (`700 10px` cat / `500 9px` res), HUD colour `#46464E`, hub centre-punch `0.4×`, archived grey, selected ring offset `s+4`.
- New-node green ring animation not implemented (low priority).
