# Right Rail — Collapsed Strip

Source: `dc.html` 957–964. Shown when `rightOpen` is false (`rightClosed`). Width 56px.

## Strip
- `width:56px; height:100%; display:flex; flex-direction:column; align-items:center; padding-top:10px; gap:4px.`

### 1. Expand button
- `36×36; display:flex; centre; color:#7A7A84; background:none; border:1px solid transparent; cursor:pointer.`
- Hover: `color:#E9E9EC; background:#15151A`.
- Icon `chevL` (points left = "expand toward content/open"). `onClick → toggleRight`.

### 2. Divider
- `width:24px; height:1px; background:#1F1F25; margin:6px 0.`

### 3. Selected-category swatch
- `14×14; background:{sel.catColor}` — colour of the selected resource's category, so the collapsed rail still signals current selection.

## Build notes / current gaps
- Build collapsed strip has expand chevron + cat swatch but: swatch should be `14×14` (build uses 8×8), add the `24×1 #1F1F25` divider above it, expand btn `36×36` hover `#15151A`.
