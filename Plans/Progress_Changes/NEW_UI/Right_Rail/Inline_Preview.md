# Right Rail — Inline Preview

Source: `dc.html` 886–951 (markup), 1946–1955 (model).
Shown when `previewActive` (toggled by the PREVIEW quick action). **Replaces** the
AI Summary block.

## Header row
- `display:flex; align-items:center; gap:8px; margin-bottom:10px.`
- "PREVIEW" `font-size:9px; letter-spacing:1.5px; color:#F0703C`.
- "·" `font-size:8px; color:#3C3C44`.
- "{prev.kindLabel} · {prev.host}" `font-size:8px; letter-spacing:1.5px; color:#5C5C66; flex:1; ellipsis`.

`prev` model: `{ title, host, color, typeLabel, summary, date, catName, catColor, url, lines:[{w}], isPaged, isLink, isNote, isImage }`. `lines` widths: `['100%','94%','97%','88%','100%','72%','96%','83%']` (faux text lines).

## Variant 1 — Paged (PDF / DOC) — `prev.isPaged`
- White page: `background:#fff; color:#15151A; padding:18px 20px; box-shadow:0 4px 14px rgba(0,0,0,0.45).`
  - kicker `font-size:7.5px; letter-spacing:1.5px; color:#9A9AA0; margin-bottom:12px` = "{kindLabel} · {host}".
  - title `font-size:14px; font-weight:700; line-height:1.25; color:#111; margin-bottom:10px`.
  - summary `font-size:10.5px; line-height:1.65; color:#33333A; margin-bottom:14px`.
  - faux lines: each `height:5px; background:#E6E6EA; width:{w}` (5 lines).

## Variant 2 — Link (browser frame) — `prev.isLink`
- Card `background:#fff; box-shadow:0 4px 14px rgba(0,0,0,0.45).`
  - **Browser chrome bar** `height:26px; background:#ECECEF; border-bottom:1px solid #D8D8DC; display:flex; align-items:center; gap:5px; padding:0 8px`:
    - 3 traffic dots `7×7; border-radius:50%` colours `#F66`, `#FB5`, `#4C5`.
    - URL pill `flex:1; height:16px; margin-left:4px; background:#fff; border:1px solid #D2D2D8; border-radius:8px; padding:0 8px; font-size:8px; color:#6A6A72; ellipsis` = `{prev.url}`.
  - Body `padding:18px 20px; color:#15151A`: title `14px/700`; summary `10.5px #33333A`; hero block `height:78px; background:linear-gradient(135deg,#EEF0F5,#DfE3EC); border:1px solid #E0E0E6; margin-bottom:12px`; 4 faux lines.

## Variant 3 — Note — `prev.isNote`
- Dark card `background:#131318; border:1px solid #26262C; padding:16px 18px`:
  - kicker `display:flex; align-items:center; gap:7px; font-size:7.5px; letter-spacing:1.5px; color:#5C5C66; margin-bottom:12px` → `6×6` `{catColor}` swatch + "NOTE · {catName} · {date}".
  - title `font-size:13px; font-weight:600; color:#E9E9EC; margin-bottom:10px`.
  - body `font-size:11px; line-height:1.7; color:#B9B9C0; white-space:pre-wrap` = `{summary}`.

## Variant 4 — Image — `prev.isImage`
- Checkerboard `aspect-ratio:4/3; background:repeating-linear-gradient(45deg,#14141A,#14141A 10px,#101016 10px,#101016 20px); border:1px solid #2A2A30; display:flex; centre; margin-bottom:8px`:
  - camera glyph `46×46; border:2px solid {color}; position:relative` with a `10×10` circle (lens) + triangle (shutter) in `{color}`.
- caption `font-size:10px; color:#9A9AA0; text-align:center` = `{title}`.

## Build notes / current gaps
- **Not implemented** in build (deferred). Document complete; implement when inspector preview is prioritised. Wire to the PREVIEW quick action toggling `previewActive`.
