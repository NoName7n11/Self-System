# Left Rail — Profile Deck (Update banner + Footer + Collapsed strip)

Source: `dc.html` 169–186 (spacer/update/footer), 350–357 (collapsed strip).

## Spacer
- Between CATEGORY NODES and the update line: `<div style="flex:1"></div>` pushes the update banner + footer to the bottom.

## Update banner
- `flex:none; height:28px; display:flex; align-items:center; gap:8px; padding:0 14px; cursor:pointer; color:#9A9AA0`. Hover `color:#F0703C`.
- `onClick → toggleNotif` (opens the notifications panel — see Main_Content/Top_Bar).
- Children:
  - `6×6` square dot `background:#F0703C`.
  - "UPDATE AVAILABLE · v0.1.1" — `flex:1; font-size:9.5px; letter-spacing:.5px`, left-aligned.
  - chevron `icChevR` `color:#5C5C66; flex:none`.

## Footer (profile row)
- `height:52px; flex:none; border-top:1px solid #1A1A1F; display:flex; align-items:center; gap:10px; padding:0 12px`.
- **Avatar**: `26×26; background:#1B1B20; border:1px solid #2A2A30; flex-centre; font-size:11px; font-weight:700; color:#F0703C` — letter "N".
- **User block**: `flex:1; min-width:0; line-height:1.2`.
  - name "noname" — `font-size:11px; color:#E9E9EC; ellipsis` (lowercase literal).
  - sub "local · single user" — `font-size:9px; color:#5C5C66; ellipsis` (lowercase, NOT uppercase).
- **Settings**: `24×24; color:#5C5C66; background:none; cursor:pointer`, hover `#E9E9EC`. Icon `icCog` (gear): `[3,1][3,5][1,3][5,3][2,2][4,2][2,4][4,4][3,3]`. Title "Settings".

## Collapsed strip (rail width 56)
- Body: `flex:1; display:flex; flex-direction:column; align-items:center; padding-top:8px; gap:2px`.
- **Nav icons** (CHAT/TASKS/LIBRARY): each `36×36; display:flex; centre; color:#7A7A84; background:none; border:1px solid transparent; cursor:pointer`. Hover `color:#E9E9EC; background:#15151A`. `onClick → nav.open` (expands + sets leftView).
- Spacer `flex:1`.
- **Settings gear** pinned bottom: `36×36; margin-bottom:10px; color:#5C5C66`, hover `#E9E9EC`. Icon `icCog`.
- (Header logo chip stays at top, centred — see Header.md.)

## Build notes / current gaps
- Update banner present ✓. Footer: sub-line fixed to lowercase ✓. Verify avatar `26×26` bg `#1B1B20` border `#2A2A30` accent letter; build uses `28×28` `--bg-input` border `--border` → change to design values.
- Collapsed strip: build shows logo + 3 nav icons + gear ✓; verify icon `36×36`, hover `#15151A`, gear `margin-bottom:10px`.
