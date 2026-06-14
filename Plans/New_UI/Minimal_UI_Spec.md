# Minimal UI Spec — Desktop Layout

Figma file: **AI-Self-System** (`sCDEP4XDaOvwMTqeP23Ajc`), page **UI**.
This doc is the implementation reference for the "very minimalistic" 3-zone desktop layout designed this session. Use it to map Figma frames → React components when coding.

## 1. Top-level layout — "Desktop - Minimal" (node `658:11`)

3-column horizontal layout, 1440x1024, no gaps/padding:

| Zone | Node | Width | Notes |
|---|---|---|---|
| Left_Rail (collapsed) | `658:12` | 64px | Icon-only nav: logo + Chat/Tasks/Library icons. Toggle expands to "Left_Rail - Expanded". |
| Main_Content | `658:13` | fill | Always the Knowledge Graph canvas. Persistent background regardless of Left_Rail/Right_Rail state. |
| Right_Rail (collapsed) | `658:14` | 64px | Collapse toggle + profile avatar. Expands to "Right_Rail - Expanded" (Inspector). |

**Key UX decision**: Main_Content is *not* a per-nav-item view. It is always the graph canvas. Only Left_Rail and Right_Rail swap their internal content based on navigation/selection.

## 2. Left_Rail states

All states are 280px wide, share the same **Header** (logo "Self Systems" + collapse toggle), built as separate Figma frames (one per state) since this is a swap, not a single component with variants — when coding, implement as one component with a `state` prop.

### 2a. "Left_Rail - Expanded" (`663:11`) — DEFAULT / HOME state
This is what's shown by default (not the 64px collapsed strip — that's the secondary/compact toggle state).

Sections (top to bottom):
- **Header** (`664:*`): Logo + "Self Systems" + collapse toggle button (→ collapses to 64px `Left_Rail` icon strip). Below it: global **Search_Bar** ("Search...").
- **Primary_Nav** (`663:13`): 3 items — **Chat**, **Tasks**, **Library**. (See §4 for why "Search" and "Knowledge Graph" were removed.)
- **Categories** (`663:14`): Up to ~5 category rows (color dot + name + resource count), e.g. Research (24), Projects (12), Reading List (38). Ends with **"View all categories →"** link (accent color) — routes to "Left_Rail - Library" with category filter focused.
- **Recent** (`663:15`): List of 5 most recently *opened/accessed* resources (icon + truncated title). This is a shortcut list, not full history.
- **Footer** (`663:16`): Settings row + Profile row (avatar, name "No_Name", email).

### 2b. "Left_Rail - Search" (`686:11`)
Triggered by tapping the header **Search_Bar** in "Left_Rail - Expanded". This is the *same search field*, now active/expanded.

- **Header**: Logo/toggle row + `← Back` row (label "Search") + the search input itself (now larger, "Search resources, tags, people…").
- **Body**:
  - "RECENT SEARCHES" — list of recent query strings (e.g. "raft consensus", "q3 roadmap").
  - "RESULTS (N)" — result cards, each: title (truncated) + meta line (`Type · Category`).

`← Back` returns to "Left_Rail - Expanded".

### 2c. "Left_Rail - Chat" (`685:11`)
Triggered by **Chat** nav item.

- **Header**: Logo/toggle row + `← Back` row (label "Chat").
- **Body**: Chat thread — alternating message bubbles (user = accent-tinted right-ish, AI = neutral panel bg), spacer pushes content up, **Chat_Input** pinned at bottom ("Message…").

This is the full chat experience — also referenced as the home for "chatting" the user wanted in Left_Rail (no separate page needed).

### 2d. "Left_Rail - Tasks" (`687:11`)
Triggered by **Tasks** nav item.

- **Header**: Logo/toggle row + `← Back` row (label "Tasks") + **"+ New task"** button (accent-tinted chip + label — intentionally styled *differently* from the search bar, see §4).
- **Body**: Grouped checklist —
  - "IN PROGRESS (n)"
  - "TO DO (n)"
  - "DONE (n)" — items shown with filled checkbox + strikethrough text.
  - Each item: checkbox, title (truncated), due-date label (right-aligned).

### 2e. "Left_Rail - Library" (`688:11`)
Triggered by **Library** nav item (and by "View all categories →" from Categories section).

- **Header**: Logo/toggle row + `← Back` row (label "Library") + filter chip row (All / PDF / Link / Note — "All" active by default).
- **Body**:
  - Sort row: "N items" (left) + "Sort: Recent ▾" (right).
  - Full resource list — each row: file-type icon swatch, title (truncated), meta line (`Type · Category · Date`).

> Future: when "View all categories →" is used, this state should also support a category filter (in addition to type chips).

## 3. Main_Content — "Main_Content - Expanded" (`676:11`)

Always the Knowledge Graph canvas (1000x1024 in this mockup, fills remaining space at runtime).

- **Top_Bar** (`679:*`): Title "Knowledge Graph" + subtitle ("128 resources · 342 connections"), right side: "Filter nodes…" field + view switch (Graph / List / Timeline — "Graph" active).
- **Canvas_Area** (`680:*`): Force-directed graph mock —
  - 5 clusters: Research (blue), People (violet), Tasks (yellow), Sources (green), Archive (red). Each cluster = 1 hub node + several satellite nodes, connected by edges; faint gray edges link cluster hubs together.
  - **Legend** (top-left): color dot + cluster name for each of the 5 clusters.
  - **Zoom_Controls** (bottom-right): `−` / `Fit` / `100%` / `+` pill.
- **Command_Bar** (`681:*`): "Ask, search, or add a resource…" input, full width.

## 4. Right_Rail — "Right_Rail - Expanded" (Inspector) (`669:11`)

320px wide. Shown when a graph node/resource is selected; collapses to the 64px `Right_Rail` icon strip otherwise.

- **Header** (`670:*`): "Inspector" title + collapse toggle.
- **Details** (`671:*`): Thumbnail preview, resource title, type chip (e.g. "PDF") + date added, tag chips (Research/Distributed/Raft), source link.
- **Quick_Actions** (`672:*`): Open / Edit / Link / Delete buttons.
- **Connections** (`673:*`): "CONNECTIONS (N)" — list of related resources, each with a relation label (cites / related / referenced by / mentioned in).
- **AI_Panel** (`674:*`): "AI SUMMARY" heading + summary paragraph, spacer, **Chat_Input** ("Ask about this resource…") pinned at bottom.

## 5. Navigation / state-transition rules

- **Left_Rail default state** = "Left_Rail - Expanded". Collapse toggle (top-right of header) shrinks it to the 64px icon strip (`658:12` pattern: logo + icon-only nav). Expand reverses this.
- **Search**: tapping the header Search_Bar transitions Left_Rail → "Left_Rail - Search" (same field, active state). `← Back` returns to default.
- **Chat / Tasks / Library nav items**: tapping swaps Left_Rail body to the corresponding state (`685:11` / `687:11` / `688:11`). `← Back` returns to default ("Left_Rail - Expanded").
- **Main_Content never changes** based on Left_Rail nav — it's always the graph. Selecting a node opens/expands Right_Rail (Inspector).
- **Right_Rail default** = collapsed 64px icon strip; expands to "Right_Rail - Expanded" on node selection or via its own toggle.

## 6. UX decisions log (this session)

1. **Removed "Search" from Primary_Nav** — redundant with the header Search_Bar, which already opens "Left_Rail - Search". One field, two visual states, not two separate search bars.
2. **Removed "Knowledge Graph" from Primary_Nav** — Main_Content always shows the graph, so this nav item only duplicated the `← Back` action (return Left_Rail to default/home state). Cut for the 3-item nav (Chat/Tasks/Library). *Revisit later* if a persistent "home" affordance is needed beyond back-arrow + logo.
3. **Restyled "+ Add task" input** in "Left_Rail - Tasks" — was visually identical to the search-bar pill (same gray pill + placeholder-text styling), causing confusion as a "third search bar". Now an accent-tinted "+ New task" chip+label, visually distinct from search.
4. **Categories scalability** — vertical list breaks at 50+ categories. Capped to ~5 (most-used/recent) + "View all categories →" link routing to "Left_Rail - Library" (category becomes a filter there, alongside type chips).
5. **"RECENT" section** = recently *opened/accessed* resources (quick-access shortcuts), capped at 5 — not a full activity log.

## 7. Open items for future sessions

- Implement category filter chips in "Left_Rail - Library" (currently only type: All/PDF/Link/Note).
- Decide whether a persistent "home" affordance (separate from `← Back`) is needed in Left_Rail.
- Right_Rail default visibility rules (collapsed vs expanded on load) need confirming against real resource-selection flow.
- Main_Content view-switch tabs (List / Timeline) are placeholders — no corresponding designs yet.

## 8. Components & Interactivity (Session 59)

All states converted into Figma **Components** / **Component Sets** (page **UI**), so the prototype is built from reusable, variant-driven pieces. Maps 1:1 to a future `state` prop in React.

| Component Set | Node | Variant property | Variants |
|---|---|---|---|
| **Left_Rail** | `705:16` | `State` | `Default` (705:11), `Search` (705:12), `Chat` (705:13), `Tasks` (705:14), `Library` (705:15), `Collapsed` (707:24) |
| **Right_Rail** | `706:18` | `State` | `Collapsed` (706:16), `Expanded`/Inspector (706:17) |
| **Main_Content** | `707:25` | — (single component, no variants — always the graph canvas) | |

A componentized "Desktop - Minimal" frame (`708:11`) assembles instances: Left_Rail=Default (`708:12`) + Main_Content (`708:80`) + Right_Rail=Collapsed (`708:177`), horizontal auto-layout, 1440x1024.

### Prototype wiring (manual — Figma Plugin API can't script reactions)

Open `708:11` in Figma's **Prototype** tab and drag-connect the following (On Tap → Change to variant, animation: Smart Animate, ~200ms):

- **Left_Rail, Default → Search**: tap `Search_Bar` (in Header) → `Left_Rail.State = Search`.
- **Left_Rail, Default → Chat / Tasks / Library**: tap respective `Primary_Nav` row → `Left_Rail.State = Chat/Tasks/Library`.
- **Left_Rail, Default → Library** (alt entry): tap `View_All_Link` (in Categories) → `Left_Rail.State = Library`.
- **Left_Rail, any non-Default → Default**: tap `Back_Row` (`← Back`) → `Left_Rail.State = Default`.
- **Left_Rail, Default → Collapsed**: tap `Collapse_Toggle` (Header) → `Left_Rail.State = Collapsed`.
- **Left_Rail, Collapsed → Default**: tap logo / any nav icon → `Left_Rail.State = Default`.
- **Right_Rail, Collapsed → Expanded**: tap `Toggle_Panel` icon, or tap a node in `Main_Content`'s `Canvas_Area` → `Right_Rail.State = Expanded`.
- **Right_Rail, Expanded → Collapsed**: tap `Collapse_Toggle` (Inspector header) → `Right_Rail.State = Collapsed`.

`Main_Content` instance stays fixed — no reactions needed on it (per §5, never swaps).