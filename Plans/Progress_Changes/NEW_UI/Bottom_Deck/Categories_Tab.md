# Bottom Deck — Categories Tab

Source: `dc.html` 534–619 (markup), 1853 (catCards). Default dock tab. Two-pane: ingest (left, flex) + category cards (right, 248px).

## Layout
- `height:100%; display:flex; min-height:0.`

## LEFT — Ingest area
- `flex:1; min-width:0; display:flex; flex-direction:column; justify-content:center; padding:18px 22px; position:relative.`
- `onDragOver/Leave/Drop` → file ingest. Drag overlay (`dragDock`): `position:absolute; inset:10px; border:1.5px dashed #F0703C; background:rgba(240,112,60,0.08); flex-centre` → "DROP FILES TO INGEST" `11px/1px #F0703C`.
- Inner column: `width:100%; max-width:560px; margin:0 auto; display:flex; flex-direction:column; gap:9px`.

### Label
- "INGEST → AUTO-CLASSIFY INTO GRAPH" — `font-size:8px; letter-spacing:1.5px; color:#5C5C66`.

### Command bar
- `display:flex; align-items:center; height:40px; background:#15151A; border:1px solid #25252B` (focus `#34343C`).
  - **Input** `flex:1; height:100%; font-size:12px; letter-spacing:.3px; padding:0 14px; color:#E9E9EC`, placeholder "PASTE A URL OR TYPE A NOTE…". `value=capInput; onKeyDown→onCapKey` (Enter submits).
  - **Attach +** `width:34px; height:100%; border-left:1px solid #25252B; color:#9A9AA0` hover `#F0703C` (icon `plus`); hidden file input.
  - **ADD** `height:100%; padding:0 16px; font-size:10px; letter-spacing:.5px; background:#F0703C; color:#0B0B0D` hover `brightness(1.1)`. `onClick→submitCapture`.
  - **Caret** `height:100%; width:28px; background:#F0703C; color:#0B0B0D; border-left:1px solid rgba(0,0,0,0.18)` → `toggleAddMenu`.

#### Add menu (`addMenuOpen`)
- Scrim `fixed inset:0 z:40`. Menu `position:absolute; top:calc(100%+4px); right:0; z:41; width:260px; background:#16161A; border:1px solid #2E2E36; box-shadow:0 10px 28px; padding:4px; animation:ssfade .12s`.
  - "Add to best-match category" — primary `10.5px #E9E9EC` + sub `8.5px #5C5C66`, hover `#22222A`. → `chooseBestMatch`.
  - divider `1px #26262C; margin:3px 6px`.
  - "+ Add as new category…" — `10.5px #F0703C` + sub, hover `#22222A`. → `startNewCat`.

### New-category mode (`newCatMode`)
- Row `height:36px; background:#101014; border:1px solid #F0703C; animation:ssfade .14s; display:flex`:
  - "NEW CATEGORY" label `8.5px/1px #F0703C; padding:0 12px; border-right:1px solid #2A2A30`.
  - input "NAME (e.g. CLIENTS)" `flex:1; font-size:11px; padding:0 12px`.
  - CANCEL `9px #7A7A84; border-left:1px solid #2A2A30` → `cancelNewCat`.
  - "CREATE & ADD" `9.5px; background:#F0703C; color:#0B0B0D` → `submitCaptureAsNewCat`.

### File chips (`capFileChips`)
- Wrap of chips `height:22px; background:#15151A; border:1px solid #26262C` with colour dot + name + ✕.

### Hint
- "Drop files here, paste a link, or jot a note — it's embedded and routed into the right category automatically." `font-size:9px; letter-spacing:.3px; color:#3C3C44`.

## RIGHT — Category cards pane
- `width:248px; flex:none; border-left:1px solid #1A1A1F; display:flex; flex-direction:column; min-height:0.`
- **Head** `height:30px; padding:0 14px; border-bottom:1px solid #16161A; display:flex; align-items:center`: "CATEGORIES" `9px/1.5px #5C5C66 flex:1` + count `9px #3C3C44`.
- **Body** `flex:1; overflow-y:auto; padding:10px; display:flex; flex-direction:column; gap:7px`.
  - Card (`c.select` → toggle `selectedCat`): `background:#131318; border:1px solid #22222A; padding:9px 11px; cursor:pointer; display:flex; align-items:center; gap:11px`. Hover `border-color:#34343C`.
    - `10×10` swatch `{c.color}`.
    - main `flex:1; min-width:0`:
      - top: name `flex:1; font-size:11px; letter-spacing:.3px; color:#E9E9EC; ellipsis` + count `font-size:15px; font-weight:700; line-height:1; color:#E9E9EC`.
      - **dot-matrix meter** `margin-top:7px; height:6px; background-image:radial-gradient(circle,#202026 .9px,transparent 1.1px); background-size:6px 6px; overflow:hidden` → fill `height:100%; width:{c.fillH}%; background-image:radial-gradient(circle,{c.color} .9px,transparent 1.1px); background-size:6px 6px`. `fillH = round(count/maxCount*100)`.
  - **NEW card**: `height:34px; flex-centre; gap:8px; border:1px dashed #2A2A30; color:#5C5C66; cursor:pointer` hover accent. icon `plus` + "NEW" `10px/.5px`.

## Build notes / current gaps
- Build categories tab is close but: ingest area should be the **large left flex region** (centred, max 560) with cards pinned **right 248px** (build stacks them differently — verify). Missing: caret add-menu, new-category mode, drop-to-ingest overlay, file chips. Card meter is **horizontal width-fill** dot-matrix (build's may be vertical) — match `height:6px` width-fill.
