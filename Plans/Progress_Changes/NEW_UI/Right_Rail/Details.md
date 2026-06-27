# Right Rail — Details

Source: `dc.html` 838–853.

## Section
- `padding:14px 16px; border-bottom:1px solid #1A1A1F.`

### 1. Title
- `font-size:14px; font-weight:600; line-height:1.35; color:#F4F4F6; margin-bottom:10px.`
- Text `{sel.title}` (e.g. "Raft Consensus Paper").

### 2. Meta row
- `display:flex; align-items:center; gap:8px; margin-bottom:12px.`
- **Type badge**: `font-size:9px; letter-spacing:.5px; padding:2px 6px; background:{sel.color}; color:#0B0B0D; font-weight:700` — `{sel.typeLabel}`.
- **Added date**: `font-size:10px; color:#5C5C66` — "ADDED {sel.date}" (e.g. "ADDED 02 MAR").
- **Spacer** `flex:1`.
- **Counter**: `display:flex; align-items:center; gap:5px; font-size:10px; color:#7A7A84` → `6×6` square `#F0703C` dot + "×{sel.counter}" (e.g. "×5"). Counter = save/share count.

### 3. Tag chips
- `display:flex; flex-wrap:wrap; gap:6px.`
- First chip = **category name** (`{sel.catName}`), then one chip per `{sel.tags}`.
- Chip: `font-size:9px; letter-spacing:.3px; padding:3px 7px; background:#16161A; border:1px solid #26262C; color:#9A9AA0.`
- Example (r1): `RESEARCH` · `Distributed` · `Consensus`.

## Build notes / current gaps
- Build has all parts but: tag chip bg is `--bg-input #15151A`; design `#16161A`. Counter colour `#7A7A84` exact. Type badge already solid type-colour ✓.
