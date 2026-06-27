# Right Rail — Preview Well

Source: `dc.html` 830–837.

## Outer padding
- `padding:16px 16px 0;` (top/sides 16, bottom 0).

## Well box
- `height:96px; position:relative;`
- `background:#0B0B0D; border:1px solid #22222A;`
- `display:flex; align-items:center; justify-content:center;`
- **Dot-grid texture** (finer than canvas): `background-image:radial-gradient(circle,#18181D 1px,transparent 1.3px); background-size:9px 9px;`

### Centre — type label
- Big type label, `font-size:34px; font-weight:800; letter-spacing:1px;`
- `color:{sel.color}` (type colour: PDF `#F67373`, LINK `#48C78E`, NOTE `#5B9CF6`, DOC `#9B59F6`, IMAGE `#F6739B`).
- Text = `{sel.typeLabel}` (e.g. "PDF").

### Top-left — host
- `position:absolute; top:8px; left:8px;`
- `font-size:8px; letter-spacing:1px; color:#4E4E57;` text `{sel.host}` (e.g. "raft.github.io").

### Top-right — category swatch
- `position:absolute; top:8px; right:8px; width:8px; height:8px; background:{sel.catColor}`.

## Build notes / current gaps
- Build preview well height/structure close, but: dot-grid is the coarse 24px canvas grid — design uses **fine 9px / #18181D**. Type label is `26px` in build → design `34px/800/1px`. Host `9px` → `8px #4E4E57`. Cat swatch `10×10` → `8×8`.
