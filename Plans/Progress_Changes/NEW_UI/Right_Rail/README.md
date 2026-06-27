# Right_Rail (Inspector) — index

Source: `dc.html` 813–965 (markup), 2023–2030 (quickActions), 1956–1960 (suggestedQs), 1170–1198 (resize), 1946–1960 (preview model).

Shows the **selected resource** (`selectedId`, default `r1`). When nothing is
selected the design keeps the last selection (always one selected in the seed).

## Container (`<aside>`)
- `width: {rightW}` — open `336px` default, resize `300–460px` (persist `ss-rightW`); closed `56px`.
- `flex:none; background:#0E0E11; border-left:1px solid #1D1D22;`
- `display:flex; flex-direction:column; overflow:hidden; position:relative;`
- transition `width .18s ease` (none while resizing).
- **Resize handle** (open only): absolute `top:0; left:0; width:5px; height:100%; cursor:col-resize; z-index:6`; class `ss-handle` hover/active `background:#F0703C`; `onMouseDown → startResizeRight`. Dragging below `300px` snaps closed (`rightOpen:false`, width reset to 336).

## Component files
| File | Component |
|---|---|
| `Header.md` | INSPECTOR title + collapse chevron |
| `Preview_Well.md` | 96px dot-grid inset, 34px type label, host, cat swatch |
| `Details.md` | title, type badge, ADDED date, ×counter, cat + tag chips |
| `Quick_Actions.md` | PREVIEW · EDIT · ARCHIVE · DELETE (2×2) |
| `Connections.md` | CONNECTIONS·N rows (swatch + title + relation) |
| `AI_Summary.md` | summary text + 3 suggested-question chips |
| `Inline_Preview.md` | paged / link-frame / note / image variants (replaces AI summary) |
| `Collapsed_Strip.md` | 56px: expand chevron + selected-cat swatch |

## Body scroll wrapper
- `flex:1; min-height:0; overflow-y:auto` — holds Preview → Details → Quick actions → Connections → (AI summary OR Inline preview).
