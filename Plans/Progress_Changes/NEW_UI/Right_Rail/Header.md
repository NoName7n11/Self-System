# Right Rail — Header

Source: `dc.html` 822–827.

## Bar
- `height:52px; flex:none; display:flex; align-items:center; gap:8px;`
- `padding:0 14px; border-bottom:1px solid #1A1A1F.`

### 1. Title
- Text "INSPECTOR" — `font-weight:700; font-size:12px; letter-spacing:.5px;` color `#E9E9EC`.

### 2. Spacer
- `<div style="flex:1">`.

### 3. Collapse toggle
- `24×24; display:flex; centre; color:#5C5C66; background:none; border:1px solid transparent; cursor:pointer.`
- Hover: `color:#E9E9EC; border-color:#26262C`.
- Icon `chevR`: `[2,1][3,2][4,3][3,4][2,5]`.
- `onClick → toggleRight` (collapses to 56px strip).

## Build notes / current gaps
- Build header label is `10px/1.5px letter-spacing dim`; design is `12px/700/.5px #E9E9EC`. → match design (bigger, brighter, tighter tracking).
