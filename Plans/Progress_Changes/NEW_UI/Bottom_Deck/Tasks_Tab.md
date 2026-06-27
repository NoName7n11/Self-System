# Bottom Deck — Tasks Tab

Source: `dc.html` 678–753 (markup), 1930 (T/P/D segs), 1935 (draft status chips).

## Layout
- `height:100%; display:flex; flex-direction:column.`

### Head
- `height:38px; flex:none; display:flex; align-items:center; gap:8px; padding:0 16px; border-bottom:1px solid #16161A.`
- spacer `flex:1`.
- **NEW TASK** chip: `height:26px; padding:0 10px; gap:6px; color:#F0703C; background:rgba(240,112,60,0.1); border:1px solid rgba(240,112,60,0.3); font-size:9.5px; letter-spacing:.5px` hover `rgba(...0.18)`. icon `plus` + "NEW TASK". `onClick → openTaskCreate`.

### Create form (`taskCreating`)
- `flex:none; padding:13px 16px; border-bottom:1px solid #1A1A1F; background:#101014; animation:ssfade .16s.`
  - Title input `width:100%; height:32px; font-size:12px; background:#15151A; border:1px solid #25252B; padding:0 10px; margin-bottom:10px` (focus `#34343C`), placeholder "TASK TITLE…".
  - Row `display:flex; gap:16px; flex-wrap:wrap`:
    - **CATEGORY**: label `8px/1px #5C5C66` + chips (per category): `height:22px; padding:0 7px; background:#15151A; border:1px solid {on}` with `8×8` colour dot + name `8.5px #9A9AA0`. `onClick → c.set`.
    - **STATUS**: label + segmented `border:1px solid #25252B`: chips `padding:0 9px; height:22px; font-size:8.5px` — `[TO DO, IN PROGRESS, DONE]`; active `background:{statusColor}; color:#0B0B0D`, inactive `transparent; #9A9AA0`.
    - **DUE**: label + input `width:96px; height:22px; font-size:10px; background:#15151A; border:1px solid #25252B`, placeholder "e.g. 20 JUL".
  - Buttons `margin-top:12px; gap:8px`: **CREATE TASK** `height:28px; padding:0 14px; background:#F0703C; color:#0B0B0D` → `createTask`; **CANCEL** `border:1px solid #26262C; color:#9A9AA0` → `cancelTaskCreate`.

### Columns body
- `flex:1; overflow-y:auto; overflow-x:hidden; padding:14px 16px; display:flex; flex-direction:column; gap:22px.`
- Per column (`taskCols`: IN PROGRESS / TO DO / DONE):
  - Head: `display:flex; align-items:center; gap:8px; margin-bottom:10px; padding-bottom:8px; border-bottom:1px solid #1A1A1F`. `7×7` dot `{col.color}` + label `9px/1px #9A9AA0` + count `9px #4E4E57`.
  - **Cards row** (`display:flex; gap:10px; overflow-x:auto; padding-bottom:6px; align-items:flex-start`) — horizontal scroll.
    - Card (`ss-task`): `width:240px; flex:none; background:#131318; border:1px solid #22222A; padding:9px 10px; position:relative` hover `border-color:#34343C`.
      - top `display:flex; align-items:flex-start; gap:8px`:
        - **checkbox** `13×13; margin-top:1px; border:1px solid {boxBd}; background:{boxBg}; color:#0B0B0D; font-size:9px` → `t.toggle`.
        - title `flex:1; font-size:11px; line-height:1.4; color:{fg}` (strike when done).
        - **⋯ menu** (`ss-taskdots`): `18×18; color:#7A7A84; opacity:0` (shown on card hover) → `t.openMenu` (Archive / Delete context menu).
      - meta `display:flex; align-items:center; gap:8px; margin-top:8px; padding-left:21px`:
        - `6×6` category dot `{t.catColor}`.
        - due `flex:1; font-size:9px; color:#5C5C66` = `{t.due}`.
        - **T/P/D status segments** `display:flex; border:1px solid #26262C`: 3 buttons `18×16; font-size:8px; font-weight:700` — `T`(open) / `P`(in_progress) / `D`(done). Active = `background:{statusColor}; color:#0B0B0D`; inactive `transparent; color:#5C5C66`. `onClick → setTaskStatus(id, status)` (stopPropagation).

## Build notes / current gaps
- Build dock tasks = vertical columns w/ simple cards. Design: **horizontal-scroll** card rows per group, card `width:240`, plus **T/P/D segments**, ⋯ menu, and the **create form**. Add NEW TASK form + segments + horizontal scroll.
