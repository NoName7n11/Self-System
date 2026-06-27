# Left Rail — CHATS (nav row + Chat rail view)

Source: nav row `dc.html` 86–94; chat view 190–278.

## A. Primary-nav row (home view)
Nav container: `padding:4px 8px; display:flex; flex-direction:column; gap:1px`. Three rows: CHAT, TASKS, LIBRARY (same row spec; this file = CHAT).

Row (class `ss-nav`):
- `display:flex; align-items:center; gap:10px; height:34px; padding:0 8px;`
- `cursor:pointer; color:#B9B9C0; border:1px solid transparent.`
- Hover: `background:#15151A; color:#E9E9EC`.
- `onClick → nav.open` → sets `leftView:'chat'` (swaps rail body; also `leftCollapsed:false`, `railConvId:null`).

Row children:
1. **Icon** — span `width:16px; display:flex; justify-content:center; color:#7A7A84; flex:none`. `chat` icon cells: `[1,1][2,1][3,1][4,1][5,1][1,2][5,2][1,3][2,3][3,3][4,3][5,3][1,4]` (speech bubble).
2. **Label** — `flex:1; font-size:11.5px; letter-spacing:.4px;` text `CHAT`, left-aligned.
3. **Open-in-dock affordance** (`ss-dull ss-navopen`) — `color:#7A7A84; flex:none; padding:3px; margin:-3px`. Icon `trend` (`↗`): `[1,5][2,4][3,3][4,2][5,1][3,1][4,1][5,2]`. **opacity 0 → 1 only on row hover** (`.ss-nav:hover .ss-navopen{opacity:1}`, `transition:opacity .12s`). `onClick → nav.openDock` (stopPropagation) → opens bottom dock to the Chat tab (does NOT change leftView).

## B. Chat rail view (`leftView==='chat'`, `isChatView`)
Body: `flex:1; display:flex; flex-direction:column; min-height:0`. Two sub-states.

### B1. Conversation list (`showConvList`, when no conv selected)
- **Head bar**: `height:40px; padding:0 10px; border-bottom:1px solid #1A1A1F; display:flex; align-items:center; gap:8px`.
  - Back button → `backHome` (returns to home). `24×24`, color `#7A7A84`, border 1px transparent, hover `color:#E9E9EC; border-color:#26262C`. Icon `chevL`.
  - Title "CHATS": `flex:1; font-size:10px; letter-spacing:1px; color:#9A9AA0`.
  - **NEW CHAT** button: `height:24px; padding:0 8px; display:flex; gap:5px; align-items:center; color:#F0703C; background:rgba(240,112,60,0.1); border:1px solid rgba(240,112,60,0.3); font-size:9px; letter-spacing:.5px`. Hover bg `rgba(240,112,60,0.18)`. Icon `plus` + text "NEW CHAT". `onClick → newChat` (prepends a new conversation, opens it).
- **List**: `flex:1; overflow-y:auto; padding:8px; display:flex; flex-direction:column; gap:2px`.
  - Row (`ss-conv`): `display:flex; align-items:center; gap:10px; padding:9px 8px; border:1px solid transparent; cursor:pointer`. Hover `background:#15151A`.
    - dot `7×7` `background:{cv.dot}` (category/topic colour).
    - main `flex:1; min-width:0`:
      - title `font-size:11px; color:#D6D6DB; ellipsis`.
      - preview (last message) `font-size:9px; color:#5C5C66; ellipsis; margin-top:2px`.
    - **⋯ options** (`ss-openicon ss-dots`): `22×22; color:#9A9AA0; background:#0E0E11; border:1px solid #26262C`. Hidden until row hover (`.ss-conv:hover .ss-openicon{opacity:1}`). `onClick → cv.openMenu` → context menu (Edit name / Open in dock / Archive / Delete — see Overlays).
  - Rename inline: when `cv.renaming`, title becomes an input `border:1px solid #F0703C; background:#0B0B0D; padding:3px 5px`.
- Seed conversations: `cv1 "Raft consensus — Q&A"`, `cv2 "RAG pipeline design"`, `cv3 "GBUS weighted signals"`.

### B2. Conversation thread (`showConvThread`, a conv selected)
- **Head**: `height:40px; padding:0 10px; border-bottom:1px solid #1A1A1F`. Back → `backConvList` (chevL). Title = conv title `flex:1; font-size:10px; letter-spacing:.5px; color:#9A9AA0; ellipsis`. Right: open-in-dock `↗` button `24×24 border:1px #26262C` → `openRailInDock`.
- **Messages** (`railChatScrollRef`): `flex:1; overflow-y:auto; padding:12px; display:flex; flex-direction:column; gap:10px`. Drag-over shows dashed drop zone "DROP FILES TO ATTACH".
  - Each msg: column, `align-items:{m.align}` (user→flex-end, ai→flex-start), `animation:ssfade .25s`.
    - bubble: `max-width:90%; padding:8px 10px; font-size:11px; line-height:1.5; background:{m.bg}; border:1px solid {m.bd}; color:{m.fg}`.
      - user: bg `rgba(240,112,60,0.12)`, border `rgba(240,112,60,0.35)`, color `#E9E9EC`.
      - ai: bg `#131318`, border `#1F1F25` (approx), color `#9A9AA0`/`#B9B9C0`.
    - **citation chip** (if `m.hasCite`): `margin-top:7px; inline-flex; gap:5px; padding:3px 7px; background:#0E0E11; border:1px solid #2A2A30; font-size:8px; color:#9A9AA0; cursor:pointer`. Hover `border-color:#F0703C; color:#F0703C`. blue `6×6 #5B9CF6` swatch + `{m.cite}` (e.g. "Raft Consensus Paper §5.2"). `onClick → m.citeSelect` selects that resource.
    - **attachment chips** (if attached): small chips `background:#0E0E11; border:1px solid #2A2A30; font-size:8px` + "+N more" hover tooltip.
- **Composer**: `border-top:1px solid #16161A; padding:9px 10px; flex column gap 7px`.
  - attach chips row (if any pending): chip `height:22px; background:#15151A; border:1px solid #26262C` with colour dot + name + ✕.
  - input row: `display:flex; align-items:center; gap:8px`.
    - attach `+` button `color:#5C5C66` hover `#F0703C` (icon `plus`); hidden `<input type=file multiple>`.
    - text input `flex:1; font-size:11px; height:28px;` placeholder `MESSAGE…`.
    - send button `28×28; background:#F0703C; color:#0B0B0D;` icon `send` (`[1,1][1,2][2,2][1,3][2,3][3,3][1,4][2,4][1,5]`). `onClick → sendRailChat`.

## Verified differences (screenshot diff, this pass)
1. **Conversation dot = "open in dock" indicator.** The `7×7` dot is **accent `#F0703C` only for the conversation currently open in the bottom dock** (`dockConvId`, default `cv1`); all other rows use a **grey dot `#3C3C44`**. Build had every dot blue `#5B9CF6` → fix to dock-driven accent/grey. Requires shared `dockConvId` (add to `useLayoutStore`, default `'cv1'`).
2. **Composer send icon wrong.** Build used `IcoTrend` (↗) for the thread composer send button — design uses the `send` triangle (`IcoSend`, `[1,1][1,2][2,2][1,3][2,3][3,3][1,4][2,4][1,5]`). Same fix needed in the **dock** Chat composer. (↗ `trend` is only the "open in dock" affordance, never send.)
3. **Conversation rows miss the ⋯ options button** (`ss-dots`, hover-reveal) → its menu is how a conversation gets opened in the dock (sets `dockConvId`), which gives the dot (#1) its meaning. Add a hover ⋯ that opens the dock to that conversation.
4. Dock Chat title should reflect `dockConvId` (build hardcodes "Raft consensus — Q&A").
5. NEW CHAT "+" renders chunkier than design — same icon set; acceptable, low priority.

## Interactions (from dc.html handlers — must all work)
- **NEW CHAT** (`newChat`): prepends `{title:'New conversation', messages:[]}` and **jumps into its thread** (`railConvId=id`).
- **⋯ menu** (`openConvMenu` → popup at row's right/bottom): 4 items —
  - **Edit name** (`startRename`): row title becomes an inline input (`border:1px #F0703C; bg #0B0B0D`); Enter commits (`commitRename`), Esc cancels.
  - **Open in dock** (`menuOpenDock`/`openConvInDock`): `dockConvId=id; dockOpen; dockTab='chat'` → row dot turns accent.
  - **Archive** (`archiveConv`): sets `archived:true`, drops from list (restorable in Archive tab).
  - **Delete** (`deleteConv`): removes; if it was the dock conv, dockConvId falls back to first non-archived.
- **Send** (`doSendRailChat`/`doSendDockChat`): appends user msg + a canned AI reply (with `Raft Consensus Paper`/`r1` citation). Enter or send button. Clears input, autoscrolls.
- Conversations are **mutable state** → must live in a store (seed from `DEMO_CONVERSATIONS`), shared by rail list/thread + dock chat + inspector ask.

## Implemented (this pass)
- `useChatStore` rewritten to be conversation-based: `conversations[]` (seed demo) + `newConversation / renameConversation / archiveConversation / deleteConversation / sendToConversation`.
- RailChat: NEW CHAT creates+enters thread; ⋯ opens a context menu (Edit name inline / Open in dock / Archive / Delete); thread send works.
- Dock ChatTab + Inspector ask route through `sendToConversation(dockConvId, …)`.

## Deferred (documented): attach chips, drag-drop, citation-chip → resource select, conversation menu "Open in dock" smart-scroll.
