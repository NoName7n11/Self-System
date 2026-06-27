# Bottom Deck — Chat Tab

Source: `dc.html` 621–676. Shows the **selected conversation** (`dockConvId`, default `cv1`).

## Layout
- `height:100%; display:flex; flex-direction:column.`

### Head
- `height:30px; flex:none; display:flex; align-items:center; gap:8px; padding:0 18px; border-bottom:1px solid #16161A.`
- `7×7` `#F0703C` dot.
- title `flex:1; font-size:10px; letter-spacing:.5px; color:#9A9AA0; ellipsis` = `{dockConvTitle}`.
- "SELECTED CONVERSATION" `font-size:8px; letter-spacing:.5px; color:#3C3C44`.

### Messages (`chatScrollRef`)
- `flex:1; min-height:0; overflow-y:auto; padding:14px 18px; display:flex; flex-direction:column; gap:12px; position:relative.`
- Drag overlay (`dragDock`): dashed `#F0703C` "DROP FILES TO ATTACH".
- Each message: column `align-items:{user?flex-end:flex-start}; animation:ssfade .25s`.
  - bubble `max-width:74%; padding:9px 12px; font-size:12px; line-height:1.5; background:{m.bg}; border:1px solid {m.bd}; color:{m.fg}` (content `white-space:pre-wrap; word-break:break-word`).
    - user: accent-soft bg / accent-bd border / `#E9E9EC`.
    - ai: `#131318` bg / soft border / dim text.
  - attachment chips (if any): `#0E0E11; border:1px solid #2A2A30; font-size:8.5px` + "+N more" hover tooltip.
  - citation chip (if `m.hasCite`): `margin-top:8px; padding:3px 8px; background:#0E0E11; border:1px solid #2A2A30; font-size:9px; #9A9AA0` hover `border/color #F0703C`; `7×7 #5B9CF6` swatch + `{m.cite}`. `onClick → m.citeSelect`.
  - role label below bubble: `font-size:8px; color:#3C3C44; margin-top:4px; letter-spacing:.5px` ("YOU"/"ASSISTANT").

### Composer
- `flex:none; border-top:1px solid #16161A; padding:10px 14px; display:flex; flex-direction:column; gap:7px.`
- pending attach chips row (if any).
- input row `display:flex; align-items:center; gap:10px`:
  - attach `+` `color:#5C5C66` hover `#F0703C` (icon `plus`); hidden file input.
  - text input `flex:1; font-size:12px; height:30px`, placeholder "ASK ABOUT YOUR KNOWLEDGE GRAPH…".
  - send `30×30; background:#F0703C; color:#0B0B0D` (icon `send`). `onClick → sendDockChat`.

## Build notes / current gaps
- Build dock chat thread + composer present; placeholder text matches. Missing: head bar (dot + conv title + "SELECTED CONVERSATION"), role labels under bubbles, citation chips, attach chips/drag-drop. Bubble `max-width:74%` `12px` — verify.
