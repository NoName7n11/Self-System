# Bottom_Deck (Dock) — index

Source: `dc.html` 512–809 (markup), 1869–1875 (tabs/lib), 1853 (catCards), 1895 (archive chips), 1930–1935 (task segs/draft).

Lives at the bottom of the centre column (below the view zone).

## Container
- `flex:none; border-top:1px solid #1A1A1F; background:#0E0E11; display:flex; flex-direction:column;`
- `height:{dockH}` — open `264px` default (resize `min 160`, `max 50%` of `innerHeight-52`, persist `ss-dockH`); collapsed `42px`.
- `overflow:hidden; position:relative;` transition `height .18s ease` (none while resizing).
- **Resize handle** (open only): absolute `top:0; left:0; height:5px; width:100%; cursor:row-resize; z-index:6`; class `ss-handle` hover/active `#F0703C`; `onMouseDown → startResizeDock`. Drag below `160px` snaps closed.

## Component files
| File | Component |
|---|---|
| `Tab_Strip.md` | 42px strip: CATEGORIES toggle · divider · CHAT/TASKS/LIBRARY/ARCHIVE tabs · spacer · collapse arrow |
| `Categories_Tab.md` | ingest command bar (+ add-menu, new-cat mode, drop-to-ingest) + 248px category cards w/ dot-matrix |
| `Chat_Tab.md` | selected-conversation thread + composer |
| `Tasks_Tab.md` | NEW TASK + create form + horizontal columns + cards w/ T/P/D segments + ⋯ menu |
| `Library_Tab.md` | RECENT·N ITEMS + sort + rows (NO filter chips here) |
| `Archive_Tab.md` | ARCHIVE·N + filter chips + restore rows + empty state |

## Tabs (`dockTabs`)
`CHAT` · `TASKS` · `LIBRARY` · `ARCHIVE` (CATEGORIES is a separate left-side toggle button, not in this array). Active (when `dockOpen && dockTab===id`): `bg:#15151A; fg:#E9E9EC; border-bottom:2px #F0703C`. Inactive: `transparent / #7A7A84 / transparent`.

## Build notes (high level)
- Build has 5 tabs in one row (categories as a tab). Design: CATEGORIES is a **left toggle button** + divider, then the 4 real tabs.
- Categories tab in build = ingest bar + cards (close), but missing add-menu caret, new-category mode, drop-to-ingest, and the ingest area should be the **large left region** with cards pinned **right 248px**.
- Tasks tab missing: create form, T/P/D status segments, ⋯ menu.
- Library tab: build adds filter behaviour from `query`; design dock library has NO chips (filter chips are the *rail* library only).
