# Main_Content — Top Bar

Source: `dc.html` 364–414 (markup), 378–407 (notif dropdown), 1861 (viewTabs), 1865–1868 (titles).

## Bar
- `height:52px; flex:none; display:flex; align-items:center; gap:14px;`
- `padding:0 16px; border-bottom:1px solid #1A1A1F; background:#0E0E11.`

### 1. Title block
- `display:flex; flex-direction:column; line-height:1.2.`
- topTitle: `font-weight:700; font-size:12px; letter-spacing:.5px` color `#E9E9EC`.
- topSub: `font-size:9.5px; color:#5C5C66; letter-spacing:.3px`.
- **Values change per `view`** (see index table): GRAPH→KNOWLEDGE GRAPH/128 RESOURCES · 342 CONNECTIONS; MAP→TASK MAP/task counts; PROGRESS→PROCESSING/processing counts.

### 2. Spacer
- `<div style="flex:1">`.

### 3. Notifications bell
- Button: `width:34px; height:32px; display:flex; centre; color:{muteColor}; background:#15151A; border:1px solid #25252B; position:relative.` Hover `color:#E9E9EC; border-color:#34343C`.
- Icon `bell` (`icBell`).
- **Unseen dot** (when notifications not yet seen): `position:absolute; top:5px; right:6px; width:6px; height:6px; background:#F0703C; border:1px solid #0E0E11`.
- `onClick → toggleNotif`.

#### Dropdown (when `notifOpen`)
- Scrim: `position:fixed; inset:0; z-index:30` (click → `closeNotif`).
- Panel: `position:absolute; top:38px; right:0; z-index:31; width:316px; background:#121215; border:1px solid #2E2E36; box-shadow:0 12px 32px rgba(0,0,0,0.55); animation:ssfade .14s`.
- **Head** `height:36px; padding:0 8px 0 12px; border-bottom:1px solid #1F1F25; display:flex; align-items:center; gap:8px`:
  - bell icon `#9A9AA0`; "NOTIFICATIONS" `flex:1; font-size:10px; letter-spacing:1px; #9A9AA0`; count `9px #5C5C66`.
  - **MUTE** toggle: `height:22px; padding:0 7px; font-size:8.5px; letter-spacing:.5px; color:{muteColor}; border:1px solid #2A2A30`, hover `border #34343C; color #E9E9EC`. Icon + `{muteLabel}` ("MUTE"/"UNMUTE"). `onClick → toggleMute`.
  - **CLEAR**: `height:22px; padding:0 8px; font-size:8.5px; color:#9A9AA0; border:1px solid #2A2A30`, hover `border #E06C75; color #E06C75`. `onClick → clearNotifs`.
- **List** `max-height:300px; overflow-y:auto`:
  - Row (`ss-notif`): `display:flex; gap:10px; padding:11px 12px; border-bottom:1px solid #16161A; cursor:pointer`. Hover `background:#16161A`.
    - icon chip `22×22; background:{n.color}; color:#0B0B0D; flex-centre` (per-notif icon).
    - body `flex:1; min-width:0`:
      - top row: title `flex:1; font-size:11px; #E9E9EC; ellipsis` + time (`ss-ntime`) `9px #4E4E57` + remove ✕ (`ss-nx`, hover `#E06C75`, hidden until row hover, replaces time).
      - body text `font-size:10px; line-height:1.45; #7A7A84; margin-top:2px`.
  - Empty (`notifEmpty`): centered `9.5px #3C3C44`.
- Seed notifs (4): "New resource added" (green, 2m); "App update available" (blue, 1h); "Deadline detected" (accent, 4h); "2 resources archived" (gold, 1d).

### 4. View switch
- Container: `display:flex; background:#15151A; border:1px solid #25252B`.
- Button per view: `padding:0 12px; height:30px; font-size:10px; letter-spacing:.5px; border:none; cursor:pointer`.
  - Active: `background:#22222A; color:#F0703C`.
  - Inactive: `background:transparent; color:#7A7A84`.
- Tabs: `GRAPH` · `MAP` · `PROGRESS`. `onClick → setState({view})`.

## Build notes / current gaps
- Title static → make per-view (index table).
- Notif bell: build bg `--bg-input`/border `--border`; design `#15151A`/`#25252B`. Dropdown built ✓ (verify head MUTE/CLEAR styling, row icon `22×22`, hover-swap time↔✕).
- View switch: build container border `--border`; design `#25252B`; active bg `#22222A` ✓, inactive fg `#7A7A84` ✓.
- Unseen dot logic: design shows dot until opened (`notifSeen`).
