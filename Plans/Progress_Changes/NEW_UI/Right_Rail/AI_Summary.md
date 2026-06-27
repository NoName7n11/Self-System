# Right Rail — AI Summary

Source: `dc.html` 873–884 (markup), 1956–1960 (suggestedQs).
Shown when `aiSummaryOpen` (default **true**) AND `previewActive` is false.
Replaced by Inline Preview when PREVIEW is toggled on.

## Section
- `padding:14px 16px.` (no bottom border — last block).

### Label
- "AI SUMMARY" — `font-size:9px; letter-spacing:1.5px; color:#5C5C66; margin-bottom:10px.`

### Summary text
- `font-size:11.5px; line-height:1.6; color:#9A9AA0; margin-bottom:14px.`
- Text `{sel.summary}`.

### Suggested-question chips
- Container `display:flex; flex-direction:column; gap:6px.`
- Chip (`q.ask`): `font-size:10.5px; color:#B9B9C0; padding:7px 10px; background:#131318; border:1px solid #1F1F25; cursor:pointer; display:flex; align-items:center; gap:8px.` Hover `border-color:#34343C; color:#E9E9EC`.
  - prefix "›" `color:#F0703C; flex:none`, then `{q.label}`.
- **Fixed 3 questions** (`askInspect` → routes Q into dock chat, opens dock Chat tab):
  1. "How does leader election work here?"
  2. "Key differences from Paxos?"
  3. "Which open questions need follow-up?"

## Footer ask input
> Note: the dc.html prototype does NOT render a separate "ASK ABOUT THIS RESOURCE…" footer input in the inspector — questions are asked via the suggested-Q chips (`askInspect`), which push into the dock Chat thread. The build added a footer input; keep it only if desired, but the canonical design routes asks through the chips + dock chat.

## Build notes / current gaps
- Build suggested-Q text differs ("Key differences from related work?") → use exact "Key differences from Paxos?". Chip bg `#131318` border `#1F1F25` ✓. Summary `11.5px #9A9AA0` ✓.
- Build's inspector footer ask input is non-canonical; design has none — consider removing to match, or leave as an enhancement (decide at implementation).
