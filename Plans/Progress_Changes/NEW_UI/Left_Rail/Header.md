# Left Rail — Header

Source: `dc.html` lines 61–70 (header), 54–58 (aside + resize handle), 350–357 (collapsed).

## Container (the whole left rail `<aside>`)
- `width: {leftW}` — expanded `264px` default, resizable `220–360px` (persisted `ss-leftW`); collapsed `56px`.
- `flex: none; background: #0E0E11; border-right: 1px solid #1D1D22;`
- `display: flex; flex-direction: column; overflow: hidden; position: relative;`
- `transition:` width `0.18s ease` (disabled while resizing).

### Resize handle (expanded only)
- Absolute, `top:0; right:0; width:5px; height:100%; cursor:col-resize; z-index:6`.
- Class `ss-handle`: `background:transparent; transition:background .12s ease;` → hover/active `background:#F0703C`.
- `onMouseDown → startResizeLeft`. Title "Drag to resize".

## Header bar
- `height: 52px; flex: none;` `display:flex; align-items:center; gap:10px;`
- `padding: 0 14px; border-bottom: 1px solid #1A1A1F;`

### 1. Logo chip
- `24×24px`, `flex:none`, `background:#F0703C`, `color:#0B0B0D`.
- `display:flex; align-items:center; justify-content:center; cursor:pointer.`
- Hover: `filter:brightness(1.12)`.
- Content: `icLogo` pixel mark (7×7 grid) — cells:
  `[1,1][1,2][1,3][1,4][1,5][5,1][5,2][5,3][5,4][5,5][2,3][3,3][4,3]` (two vertical bars + middle bridge = stylised "H"), fill `#0B0B0D`.
- `onClick → onLogo`: when collapsed, expands rail + sets `leftView:'home'`. (No-op when already expanded.)
- Title `{logoTitle}`.

### 2. Wordmark (expanded only)
- Wrapper: `flex:1; min-width:0; display:flex; flex-direction:column; line-height:1.15`.
- Line 1 "SELF SYSTEMS": `font-weight:700; font-size:12px; letter-spacing:.5px;` color `#E9E9EC`.
- Line 2 "LOCAL · v0.1.0": `font-size:9px; color:#5C5C66; letter-spacing:1px`.

### 3. Collapse toggle (expanded only)
- `24×24px; flex:none;` flex-centre. `color:#5C5C66; background:none; border:1px solid transparent; cursor:pointer.`
- Hover: `color:#E9E9EC; border-color:#26262C`.
- Icon `icToggleL` = chevron-left `chevL`: `[4,1][3,2][2,3][3,4][4,5]`.
- `onClick → toggleLeft` (flips `leftCollapsed`). Title "Collapse sidebar".

## Collapsed header (width 56)
- Header keeps the logo chip only, horizontally centred (`justify-content:center; padding:0`).
- Logo `onClick → onLogo` expands the rail.

## Build notes / current gaps
- Logo chip in build is `28×28`; design is `24×24`. → set to 24×24.
- Wordmark name in build is uppercase via CSS but should rely on literal text "SELF SYSTEMS" (already uppercase) at `12px/700/.5px`.
- Resize handle not implemented (deferred, but documented here).
