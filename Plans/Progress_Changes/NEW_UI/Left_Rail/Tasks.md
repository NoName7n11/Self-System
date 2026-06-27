# Left Rail — TASKS (nav row + Tasks rail view)

Source: nav row `dc.html` 86–94; tasks view 281–314.

## A. Primary-nav row (home view)
Same row spec as Chats §A. This row:
- Icon `tasks` (checkmark) cells: `[1,3][2,4][3,3][4,2][5,1]`. Colour `#7A7A84`.
- Label `TASKS`, `flex:1; font-size:11.5px; letter-spacing:.4px`, left-aligned.
- `↗` affordance (hover-only) → `nav.openDock` opens bottom dock Tasks tab.
- Row `onClick → nav.open` → `leftView:'tasks'`.

## B. Tasks rail view (`leftView==='tasks'`, `isTasksView`)
Body `flex:1; display:flex; flex-direction:column; min-height:0`.

### Head bar (`height:40px; padding:0 10px; border-bottom:1px solid #1A1A1F; flex; align-items:center; gap:8px`)
- Back button (chevL) → `backHome`. `24×24 color:#7A7A84` hover `#E9E9EC`/border `#26262C`.
- Title "TASKS": `flex:1; font-size:10px; letter-spacing:1px; color:#9A9AA0`.
- **+ NEW** chip: `height:24px; padding:0 8px; gap:5px; color:#F0703C; background:rgba(240,112,60,0.1); border:1px solid rgba(240,112,60,0.3); font-size:9px; letter-spacing:.5px`. Hover `rgba(240,112,60,0.18)`. Icon `plus` + "NEW". `onClick → newTask`.

### Body (`flex:1; overflow-y:auto; padding:12px; flex column; gap:14px`)
Grouped vertical columns (NOT horizontal). One group per status, in order:
1. **IN PROGRESS** (dot `#F0703C`)
2. **TO DO** (dot `#5B9CF6`) — status `open`
3. **DONE** (dot `#48C78E`)

Each group:
- Group head: `display:flex; align-items:center; gap:8px; margin-bottom:8px`.
  - `7×7` square dot `{col.color}`.
  - label `font-size:9px; letter-spacing:1px; color:#9A9AA0; flex:1` (e.g. "IN PROGRESS").
  - count `font-size:9px; color:#4E4E57`.
- Items: `display:flex; flex-direction:column; gap:6px`.
  - Card: `background:#131318; border:1px solid #22222A; padding:8px 9px`. Hover `border-color:#34343C`.
    - top row: `display:flex; align-items:flex-start; gap:8px`.
      - **checkbox** `13×13; margin-top:1px; border:1px solid {boxBd}; background:{boxBg}; color:#0B0B0D; font-size:9px`. Checked (done) → filled accent/green with ✓. `onClick → t.toggle` (done ↔ open).
      - **title** `flex:1; font-size:11px; line-height:1.4; color:{fg}`. When done: `text-decoration:line-through` + dim colour.
    - meta row: `display:flex; align-items:center; gap:6px; margin-top:6px; padding-left:21px`.
      - `6×6` category dot `{t.catColor}`.
      - due `font-size:9px; color:#5C5C66` — e.g. "DUE 18 JAN" (or raw "18 JAN").

Seed tasks (6): Review hackathon requirements (done, sources, 12 JAN); Prepare project proposal (in_progress, sources, 18 JAN); Submit application (open, sources, 20 JAN); Read Raft paper §5 (open, research, 15 MAR); Summarize RAG tutorial (done, research, 10 MAR); Draft GBUS weighted-signal spec (in_progress, ai, 14 MAR).

## Interactions (from dc.html — must work)
- **+ NEW** (`newTask`): prepends `{title:'New task', due:'—', status:'open', cat:'research'}`.
- **Checkbox** (`toggleTask`): toggles `done ↔ open`. Done → green `#48C78E` box with `✓`, title strike + `#5C5C66`.
- **Back** → home.
- (Rail has NO ⋯ menu and NO T/P/D segments — those are dock-only.)
- Card due renders as `DUE {due}` (prefix included). Category dot = category colour.

## Implemented (this pass)
- `useTaskStore`: added local `toggleTodo(id)` (done↔open, optimistic) and `quickAddTask(cat?)` (prepend) — work in demo mode without backend, mirroring the design's pure-local task state.
- `TodoItem` gained optional `cat`; `demoTasksAsTodos` carries it → category-coloured dots.
- RailTasks: NEW wired to `quickAddTask`; checkbox is a real button → `toggleTodo`; dot uses `CAT_COLOR[t.cat]`; due shows `DUE …`.
- Dock TasksTab checkbox + NEW TASK wired to the same actions.
- `.task-checkbox` → `13×13`, border `#3A3A42`; `.is-done` → green `#48C78E` (was accent orange).
- Verified live: NEW adds a card (6→7); done checkbox green + strikethrough.

## Deferred (documented): dock create-form, T/P/D status segments, ⋯ archive/delete menu (Bottom_Deck pass).
