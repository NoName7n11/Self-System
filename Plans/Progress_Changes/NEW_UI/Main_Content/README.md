# Main_Content — index

Source: `dc.html` 361–510 (markup), 1488–1789 (sim/draw), 1543–1679 (map layout), 1861–1868 (titles/tabs).

Centre column between the rails.
- `<main>`: `flex:1; min-width:0; display:flex; flex-direction:column; background:#0B0B0D.`
- Stack: **Top bar (52px)** → **canvas/view zone (flex:1)** → **Dock** (documented in `Bottom_Deck/`).
- The view zone shows ONE of three views by `state.view`: `graph` | `map` | `progress`. The same `<canvas>` always exists (pointer-events on only in graph); map/progress are absolute overlays on top.

## Component files
| File | Component |
|---|---|
| `Top_Bar.md` | per-view title/subtitle, notifications bell + dropdown, GRAPH/MAP/PROGRESS switch |
| `Graph_View.md` | force-directed canvas sim — nodes/edges, forces, draw, HUD, zoom pill, interaction |
| `Map_View.md` | **TASK MAP** mind-map overlay (category hubs → task children, expandable, SVG edges, pan/zoom) |
| `Progress_View.md` | processing queue + recently completed |

## Per-view title/subtitle (top bar)
| view | topTitle | topSub |
|---|---|---|
| graph | `KNOWLEDGE GRAPH` | `128 RESOURCES · 342 CONNECTIONS` |
| map | `TASK MAP` | `{n} IN PROGRESS · {n} TO DO · {n} DONE` |
| progress | `PROCESSING` | `{n} PROCESSING · {n} COMPLETED` |

## Build notes / current gaps (high level)
- Build top bar title is static "KNOWLEDGE GRAPH" → must change per view.
- Build MAP view is category→resources; design MAP is a **TASK MAP** (tasks by category, expandable, pan/zoom). → rebuild.
- Build PROGRESS view close in spirit; align card/queue styling.
- Graph sim already matches design forces/draw closely (kept from Change 17).
