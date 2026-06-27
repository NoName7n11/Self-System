# NEW_UI — Component-by-Component Spec

Authoritative pixel/colour spec for the instrument redesign, extracted from the
canonical prototype `Redesign from scratch_7/Self Systems.dc.html`
(claude.ai/design project `2491c7b4-4385-4401-8bac-0ce2c6a5146c`).

Purpose: close the remaining gaps between the build and the actual design by
documenting **every** component — major to minor — before implementing. Each
`.md` is the single source of truth for that component's structure, dimensions,
colours, typography, icons, states, and interactions.

## Sections

| Dir | Zone | Width/Height |
|---|---|---|
| `Left_Rail/` | left sidebar | 264px (resize 220–360) ⇄ 56px collapsed |
| `Main_Content/` | centre (top bar + graph + views) | flex |
| `Right_Rail/` | inspector | 336px (resize 300–460) ⇄ 56px collapsed |
| `Bottom_Deck/` | dock (categories/chat/tasks/library/archive) | 264px (resize) ⇄ 42px |

## Left_Rail components (this pass)

| File | Component |
|---|---|
| `Left_Rail/Header.md` | logo chip + wordmark + collapse toggle + resize handle |
| `Left_Rail/Search_Bar.md` | search input box + `/` hint |
| `Left_Rail/Chats.md` | CHAT nav row + Chat rail view (conv list → thread) |
| `Left_Rail/Tasks.md` | TASKS nav row + Tasks rail view (grouped columns) |
| `Left_Rail/Library.md` | LIBRARY nav row + Library rail view (filter chips + rows) |
| `Left_Rail/Show_Recents.md` | RECENT collapsible section + hold-to-clear |
| `Left_Rail/Category_Nodes.md` | CATEGORY NODES collapsible + selected-cat node list |
| `Left_Rail/Profile_Deck.md` | UPDATE banner + footer profile row + collapsed strip |

## Global tokens (apply everywhere)

Colours: `--bg-app #0B0B0D` · `--bg-panel #0E0E11` · `--bg-card #131318` ·
`--bg-input #15151A` · `--border #26262C` · `--border-soft #1A1A1F` ·
`--border-softer #1D1D22` · `--border-hover #34343C` · `--text #E9E9EC` ·
`--text-bright #F4F4F6` · `--text-dim #9A9AA0` · `--text-mute #5C5C66` ·
`--text-faint #3C3C44 / #4E4E57` · `--accent #F0703C` ·
`--accent-soft rgba(240,112,60,.12)` · `--accent-bd rgba(240,112,60,.35)`.

Font: `JetBrains Mono` only (400/500/600/700/800). Body `letter-spacing:-0.1px`;
labels uppercase `+0.5–1.5px`. Hard edges (`border-radius:0`). Borders `1px solid`
(dashed only for `+ NEW`). Panel width/height transitions `0.18s ease`.

Status: **documentation phase** — implement only after a section's docs are done
and reviewed.
