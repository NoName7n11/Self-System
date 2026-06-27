# Bottom Deck — Tab Strip

Source: `dc.html` 517–528.

## Strip
- `height:42px; flex:none; display:flex; align-items:center; gap:2px;`
- `padding:0 8px; border-bottom:1px solid #16161A.`

### 1. CATEGORIES toggle (left, separate)
- Button: `display:flex; align-items:center; gap:7px; height:28px; padding:0 10px; color:#9A9AA0; background:none; border:1px solid transparent; cursor:pointer.`
- Hover: `color:#E9E9EC; border-color:#26262C`.
- Icon `grid` (`icGrid`) + label "CATEGORIES" `font-size:10px; letter-spacing:.5px`.
- `onClick → openCatTab` (`dockOpen:true; dockTab:'categories'`).
- NOT styled with the active underline — it's a plain button that switches to the categories body.

### 2. Divider
- `width:1px; height:18px; background:#1F1F25; margin:0 4px.`

### 3. Tabs (CHAT / TASKS / LIBRARY / ARCHIVE)
- Each: `display:flex; align-items:center; gap:8px; height:28px; padding:0 10px; border:none; border-bottom:2px solid {bb}; cursor:pointer.`
- Label `font-size:10px; letter-spacing:.5px`.
- Active (`dockOpen && dockTab===id`): `background:#15151A; color:#E9E9EC; border-bottom-color:#F0703C`.
- Inactive: `background:transparent; color:#7A7A84; border-bottom-color:transparent`.
- `onClick → {dockOpen:true; dockTab:id}`.

### 4. Spacer
- `<div style="flex:1">`.

### 5. Collapse / expand arrow
- `28×28; display:flex; centre; color:#7A7A84; background:none; border:1px solid transparent; cursor:pointer.`
- Hover: `color:#E9E9EC; border-color:#26262C`.
- Icon `dockArrow` = `chevDn` when open / `chevUp` when collapsed.
- `onClick → toggleDock`.

## Build notes / current gaps
- Build merges CATEGORIES into the tab array (5 tabs). → make CATEGORIES a left **toggle button** + `1×18 #1F1F25` divider, then the 4 tabs.
- Collapsed strip still shows the whole 42px tab row (only the body hides). ✓ in build.
