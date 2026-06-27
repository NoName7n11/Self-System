# Right Rail — Quick Actions

Source: `dc.html` 854–859 (markup), 2023–2030 (definitions).

## Grid
- `padding:12px 16px; border-bottom:1px solid #1A1A1F;`
- `display:grid; grid-template-columns:1fr 1fr; gap:6px.` (2×2)

## Button (shared)
- `display:flex; align-items:center; justify-content:center; gap:6px; height:32px;`
- `font-size:10px; letter-spacing:.5px; cursor:pointer; border:1px solid {bd}; background:{bg}; color:{fg}.`
- Hover: `filter:brightness(1.08)`.

## The 4 actions (order: PREVIEW, EDIT, ARCHIVE, DELETE)

1. **PREVIEW** — toggles `previewActive` (swaps AI Summary ↔ Inline Preview).
   - Inactive: `bg:#15151A; bd:#26262C; fg:#B9B9C0`.
   - Active (preview on): `bg:#F0703C; bd:#F0703C; fg:#0B0B0D` (accent-filled).
2. **EDIT** — `bg:#15151A; bd:#26262C; fg:#B9B9C0`. (No-op in prototype.)
3. **ARCHIVE** — `bg:#15151A; bd:#26262C; fg:#B9B9C0`. Action: archive resource → opens dock Archive tab, `previewActive:false`.
4. **DELETE** — `bg:#15151A; bd:#26262C; fg:#E06C75` (red text). Action: mark removed from library, `previewActive:false`.

> Note: design uses **PREVIEW** (not OPEN) as the first action; it toggles the inline preview. Links/notes all use the same 4 actions.

## Build notes / current gaps
- Build uses **OPEN/EDIT/ARCHIVE/DELETE** — design first action is **PREVIEW** (toggle inline preview), accent-filled when active. → rename OPEN→PREVIEW and wire `previewActive` toggle. DELETE text colour `#E06C75` ✓.
