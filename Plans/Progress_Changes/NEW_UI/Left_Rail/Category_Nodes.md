# Left Rail — CATEGORY NODES

Source: `dc.html` 122–167. Home view only. Collapsible. Two body states.

## Divider above
- `height:1px; background:#1A1A1F; margin:10px 12px 8px`.

## Section header
- Wrapper: `padding:2px 12px 6px; display:flex; align-items:center; gap:6px`.
- **Toggle span** (`toggleCats`): `display:flex; align-items:center; gap:6px; cursor:pointer; flex:1; min-width:0; color:#5C5C66`. Hover `color:#9A9AA0`.
  - Chevron `catsChev` (`chevDn` open / `chevR` closed), `flex:none`.
  - Label "CATEGORY NODES": `font-size:9px; letter-spacing:1.5px; flex:none`.
  - **Selected-cat pill** (when `hasSelectedCat`): `display:flex; align-items:center; gap:5px; min-width:0; padding-left:6px; border-left:1px solid #1F1F25`.
    - `6×6` swatch `{selCatColor}`.
    - name `font-size:9px; letter-spacing:.5px; color:#9A9AA0; ellipsis` (`selCatName`).
- **CLEAR** (when `hasSelectedCat`): `font-size:9px; color:#7A7A84; cursor:pointer; letter-spacing:.5px`, hover `#E9E9EC`. `onClick → clearSelectedCat`.

## Body (when `catsOpen`)

### State 1 — a category is selected (`hasSelectedCat`)
- **Sub-head**: `padding:2px 12px 4px; display:flex; align-items:center; gap:8px`.
  - `{catNodeCount} NODES` — `font-size:8.5px; letter-spacing:1px; color:#5C5C66; flex:1`.
  - **VIEW ALL →** — `font-size:9px; color:#F0703C; cursor:pointer; letter-spacing:.5px`. `onClick → openCatTab` (opens dock Categories).
- **Node list**: `padding:0 8px 4px; display:flex; flex-direction:column; gap:2px; overflow-y:auto`.
  - Row (`r.select`): `display:flex; align-items:center; gap:8px; padding:6px 6px; border:1px solid transparent; cursor:pointer; background:{r.bg}`. Hover `background:#15151A; border-color:#22222A`.
    - **type badge** `18×18; background:{r.color}; color:#0B0B0D; font-size:7px; font-weight:700; letter-spacing:.5px; flex-centre` — `{r.typeLabel}`.
    - title `flex:1; min-width:0; font-size:10.5px; color:#C9C9CF; ellipsis`.
  - Empty (`catNodesEmpty`): "No nodes in this category yet." `padding:14px 6px; font-size:10px; color:#5C5C66; text-align:center`.

### State 2 — no category selected (`noSelectedCat`, default)
- `padding:14px 14px 4px; display:flex; flex-direction:column; gap:8px`.
- Hint: "Select a category in the graph or in the bottom dock to see its nodes here." `font-size:10px; line-height:1.5; color:#5C5C66`.
- **BROWSE CATEGORIES →** `font-size:9px; color:#F0703C; cursor:pointer; letter-spacing:.5px`. `onClick → openCatTab`.

## Interaction
- `selectedCat` is toggled by: graph hub click, dock category card click, or rail category-node-list rows.
- Selecting a category dims the graph to that cluster and populates State 1 here.

## Build notes / current gaps
- Build renders header + State 2 hint + BROWSE link ✓ and a basic State 1 list. Verify: selected-cat pill in header (swatch + name + CLEAR), `{N} NODES` sub-head + VIEW ALL, node row type badge `18×18` with 7px label (build may use plain swatch), title colour `#C9C9CF` `10.5px`.
- Label text is "CATEGORY NODES" (build currently "CATEGORY NODES" ✓).
