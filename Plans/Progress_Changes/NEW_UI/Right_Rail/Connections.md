# Right Rail — Connections

Source: `dc.html` 860–872.

## Section
- `padding:14px 16px; border-bottom:1px solid #1A1A1F.`

### Label
- "CONNECTIONS · {sel.connCount}" — `font-size:9px; letter-spacing:1.5px; color:#5C5C66; margin-bottom:10px.`

### List
- `display:flex; flex-direction:column; gap:1px.`
- Row (`c.select`): `display:flex; align-items:center; gap:10px; height:32px; padding:0 6px; cursor:pointer; border:1px solid transparent.` Hover `background:#15151A`.
  - `8×8` swatch `background:{c.color}` (connected resource's category colour).
  - title `flex:1; min-width:0; font-size:11px; color:#B9B9C0; ellipsis` — `{c.title}`.
  - relation label `font-size:8px; letter-spacing:.5px; color:#5C5C66; flex:none` — `{c.rel}` displayed UPPERCASE.
- `onClick → c.select` selects that resource (updates inspector).

## Relation labels
From resource `con[]` data: `CITES`, `CITED BY`, `RELATED`, `REF BY`, `MENTIONED`, `SOURCE OF`, `FOUND VIA`.

Example (r1 Raft Consensus Paper, connCount 3):
- Paxos Made Simple — `CITES`
- RAG Tutorial — `RELATED`
- Advanced RAG Paper — `REF BY`

## Build notes / current gaps
- Build renders this correctly from demo `connections` ✓. Verify swatch `8×8`, row height `32`, rel label `8px #5C5C66`, title `11px #B9B9C0`. Empty state ("No connections yet.") only when a real backend resource has none.
