# Left Rail — Search Bar

Source: `dc.html` lines 77–83. Shown only in the **home** left-view (`isHome`).

## Outer padding
- `padding: 12px 12px 8px;` (top 12, sides 12, bottom 8).

## Box
- `display:flex; align-items:center; gap:8px;`
- `height: 34px; padding: 0 10px;`
- `background:#15151A; border:1px solid #25252B; color:#5C5C66;`
- Focus (focus-within): `border-color:#34343C`.

### 1. Search icon
- `color:#5C5C66; flex:none.`
- `icSearch` cells: `[1,1][2,1][3,1][1,2][3,2][1,3][2,3][3,3][4,4][5,5]` (magnifier).

### 2. Input
- `flex:1; font-size:11px; letter-spacing:.3px; color:#E9E9EC;`
- `background:none; border:none; outline:none.`
- Placeholder: `SEARCH RESOURCES, TAGS…` — placeholder color `#4E4E57`.
- `value = {query}` (shared with top-bar filter), `onInput → onSearch`.

### 3. "/" key hint chip
- `font-size:9px; color:#3C3C44; border:1px solid #25252B; padding:1px 4px; flex:none.`
- Literal text `/`.

## Behaviour
- Typing filters the graph (dims non-matches) and the Library/Recent lists.
- Same `query` value is mirrored in the top-bar filter input.

## Build notes / current gaps
- Build search row sits flush (no outer 12/12/8 padding wrapper) and the box border is `--border #26262C`; design uses `#25252B` with the outer padding. → wrap in `padding:12px 12px 8px`, box border `#25252B`.
- Build input placeholder is dimmer; keep `#4E4E57`.
