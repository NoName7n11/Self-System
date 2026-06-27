# Bottom Deck — Library Tab

Source: `dc.html` 757–775 (markup), 1873 (libraryRows). NO filter chips here (those are the *rail* library only).

## Layout
- `height:100%; display:flex; flex-direction:column.`
- Inner: `flex:1; min-height:0; overflow-y:auto; padding:12px 16px.`

### Header
- `display:flex; align-items:center; justify-content:space-between; margin-bottom:8px.`
- "RECENT · {libCount} ITEMS" — `font-size:9px; letter-spacing:1.5px; color:#5C5C66`.
- "SORT: RECENT ▾" — `font-size:9px; letter-spacing:.5px; color:#7A7A84`.

### Rows (`libraryRows` = resources not removed and not archived)
- Row (`ss-libitem`, `r.select`): `display:flex; align-items:center; gap:12px; height:36px; padding:0 8px; cursor:pointer; background:{r.bg}` (selected `#15151A`). Hover `background:#131318`.
  - **type badge** `18×18; background:{r.color}; color:#0B0B0D; font-size:7px; font-weight:700; flex-centre` — `{r.typeLabel}` = `type.toUpperCase().slice(0,4)` (PDF/LINK/NOTE/DOC/IMAG).
  - title `flex:1; min-width:0; font-size:11.5px; ellipsis`.
  - meta (`ss-lmeta`) `font-size:9px; color:#5C5C66; flex:none` = "{catName} · {date}" — hidden on hover.
  - remove (`ss-lx`) `16×16; color:#7A7A84` hover `#E06C75`, icon `cross`. Shown on hover (replaces meta). `onClick → r.remove`.
- `r.select` → `selectedId`, open inspector, `view:'graph'`.

## Build notes / current gaps
- Build dock library shows rows ✓ but filters by `query` — design dock library is **unfiltered** (shows all non-archived; filtering is the rail library's job). Type badge text uses 4-char slice (`IMAG` for image). Add hover swap meta↔remove. Meta format "catName · date".
