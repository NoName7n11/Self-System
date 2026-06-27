# Bottom Deck — Archive Tab

Source: `dc.html` 778–805 (markup), 1895 (archiveChips). Holds archived resources/chats/tasks.

## Layout
- `height:100%; display:flex; flex-direction:column.`

### Head
- `flex:none; display:flex; align-items:center; gap:8px; padding:11px 16px 10px; border-bottom:1px solid #16161A.`
- "ARCHIVE · {archiveCount}" — `font-size:9px; letter-spacing:1.5px; color:#5C5C66; flex:none`.
- spacer `flex:1`.
- **Filter chips** segmented `border:1px solid #25252B`: buttons `padding:0 10px; height:24px; font-size:8.5px; letter-spacing:.5px` — `ALL` / `RESOURCES` / `CHATS` / `TASKS`. Active `background:#22222A; color:#F0703C`; inactive `transparent; color:#7A7A84`. `onClick → ch.set` (`archiveFilter`).

### List
- `flex:1; min-height:0; overflow-y:auto; padding:8px 12px.`
- Row (`ss-arch`, `a.open`): `display:flex; align-items:center; gap:12px; height:42px; padding:0 10px; cursor:pointer; border:1px solid transparent`. Hover `background:#131318; border-color:#1F1F25`.
  - **type badge** `22×22; background:{a.color}; color:#0B0B0D; font-size:7px; font-weight:700; flex-centre` = `{a.typeLabel}`.
  - main `flex:1; min-width:0`:
    - title `font-size:11.5px; color:#E9E9EC; ellipsis`.
    - sub `font-size:9px; color:#5C5C66; ellipsis; margin-top:1px` = "{a.kind} · {a.sub}".
  - **RESTORE** button `height:24px; padding:0 10px; font-size:8.5px; letter-spacing:.5px; color:#48C78E; background:none; border:1px solid #2A2A30` hover `border-color:#48C78E; background:rgba(72,199,142,0.1)`. `onClick → a.restore`.
- **Empty** (`archiveEmpty`, default for chats/tasks): `border:1px dashed #26262C; padding:30px; text-align:center; font-size:10px; letter-spacing:.5px; color:#3C3C44; margin-top:8px` — "NOTHING ARCHIVED".

## Default data
- Seed archived resources: `r20 Old Hackathon 2024`, `r21 Expired Internship` (both `arch:true`, category archive). Chats/tasks archives empty unless user archives them.

## Build notes / current gaps
- Build archive tab lists archived-category resources + RESTORE ✓ but missing: filter chips (ALL/RESOURCES/CHATS/TASKS), `22×22` badge, sub-line "kind · sub", row height `42`. Restore is a no-op in build (wire later).
