# Self Systems — UI/UX Design Guide

> **Document Purpose:** Consolidates all UI/UX design decisions (Q61-Q70). To be enhanced with Figma wireframes and visual specifications as the user designs them.

**Last Updated:** April 3, 2026

---

## 1. Visual Design Direction

### 1.1 Color Theme

**Decision:** Dark Mode (user to decide specific color palette)

**Rationale:**
- ✅ Reduces eye strain
- ✅ Better for 3D graph visualization (nodes pop on dark background)
- ✅ Modern standard for desktop apps

**Implementation:**
- Use Tailwind CSS with dark mode support
- Color palette to be defined once user determines preferences
- CSS variables for easy theme switching in future

**To be provided by user:**
- Primary dark background color (e.g., #1a1a1a, #0f0f0f, #1e1e1e)
- Accent color (for buttons, highlights, active states)
- Secondary colors (warnings, success, errors)
- Text colors for readability on dark background

---

### 1.2 Component Library

**Decision:** shadcn/ui + shadcn MCP integration

**Integration Plan:**
- Use shadcn/ui for all base components (buttons, inputs, dropdowns, modals, etc.)
- Leverage shadcn MCP for enhanced component generation
- Customize components via Tailwind CSS as needed
- Ensure all components follow accessibility guidelines

**Component Categories to Build:**
| Category | Examples |
|---|---|
| Navigation | Sidebar, Top Bar, Breadcrumbs |
| Forms | Text Input, Dropdown, Multi-select, Date Picker |
| Feedback | Toast, Modal, Alert, Badge |
| Display | Card, List, Table, Grid |
| Graphs | Force Graph 3D/2D, Node details panel |

---

### 1.3 3D Graph Visual Style

**Decision:** Functional + Subtle Polish (Option D)

**Visual Characteristics:**
```
Base Elements:
├── Nodes
│   ├── Colored spheres (by category)
│   ├── Labels (visible when zoomed in > 50%)
│   ├── Processing badges ("PROCESSING", "NEW")
│   ├── Interactive states (hover: glow, selected: highlight)
│   └── Size variation (by counter/importance)
│
├── Links/Edges
│   ├── Thin lines for weak edges (related categories)
│   ├── Thick lines for strong edges (primary category)
│   ├── Opacity variation by strength
│   └── Color gradient matching connected nodes
│
└── Environment
    ├── Soft shadows on nodes
    ├── Soft focus blur on distant nodes
    ├── Smooth transitions on interactions
    ├── Clear visual feedback on hover
    └── No excessive animations (performance-first)
```

**Performance Consideration:**
- LOD (Level of Detail) rendering when > 300 nodes
- Simplified geometry at distance
- Use frustum culling to skip off-screen nodes

**Technology:** react-force-graph + Three.js (via force-graph)

---

## 2. Navigation & Layout

### 2.1 Main Navigation Structure

**Decision:** Collapsible Left Sidebar (Option C)

**Layout:**
```
┌─────────────────────────────────────────────────────┐
│ ≡ [Logo] Self Systems    [🔍] [🔔] [👤]            │ ← Top Bar
├─────────────────────────────────────────────────────┤
│ │                                                   │
│ │ Sidebar (Collapsible)  │   Main Content Area     │
│ │    ├─ Graph           │                           │
│ │    ├─ Search          │   [3D Graph View]        │
│ │    ├─ Chat            │   [Search Results]       │
│ │    ├─ Tasks           │   [Chat Interface]       │
│ │    └─ Settings        │   etc.                   │
│ │                       │                           │
│ │   [≡ Collapse]        │                           │
└─────────────────────────────────────────────────────┘
```

**Behavior:**
- **Desktop (> 1024px):** Always expanded, left sidebar fixed
- **Tablet (768px-1024px):** Collapsed to icons, click to expand
- **Mobile (<768px):** Hamburger menu, modal on click

---

### 2.2 Sidebar Content Organization

**Decision:** Navigation + Favorites + Recent (Option D)

**Sidebar Layout:**
```
┌────────────────────────────┐
│ [🏠] HOME                   │ ← Quick navigation
│ [📊] GRAPH                  │
│ [🔍] SEARCH                 │
│ [💬] CHAT                   │
│ [✓] TASKS                   │
│ [⚙] SETTINGS                │
│                             │
├─ FAVORITES                  │ ← Quick access
│ ❤ AI (12)                   │
│ ❤ Healthcare (4)            │
│ ❤ Opportunities (8)         │
│ [+ Add]                     │
│                             │
├─ RECENT                     │ ← Recently viewed
│ 📌 RAG Tutorial             │
│ 📌 Hackathon 2026           │
│ 📌 ML Research Paper        │
│ [View all]                  │
│                             │
└─ [Settings] [Logout]       │ ← Footer actions
└────────────────────────────┘
```

**Features:**
- Favorites are user-editable (drag to reorder)
- Recent shows last 5 viewed resources
- Search bar for quick category filtering
- Smooth collapse/expand animation

---

### 2.3 Graph Panel Controls

**Decision:** Hybrid (Toolbar + Context Menu) (Option D)

**Toolbar (Top of Graph View):**
```
[−] [+] [↺ Reset] [🔍 Search] [⊕ Filter] [2D/3D ⇅] [Extra ▼]
```

**Floating Toolbar Buttons:**
| Button | Action | Icon |
|---|---|---|
| Zoom Out | Decrease zoom level | − |
| Zoom In | Increase zoom level | + |
| Reset | Return to default view | ↺ |
| Search | Focus search input (Cmd+F) | 🔍 |
| Filter | Show filter options | ⊕ |
| Toggle 2D/3D | Switch between views | ⇅ |
| More Options | Layout, export, etc. | ▼ |

**Right-Click Context Menu (on graph):**
```
┌─────────────────────────────────
│ View details                     │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━      │
│ Pin position                     │
│ Center on view                   │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━      │
│ Edit                             │
│ Move to category                 │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━      │
│ Add to favorites                 │
│ Mark as priority                 │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━      │
│ Delete                           │
└─────────────────────────────────┘
```

**Right-Click on Node:**
```
Same as above + 
│ View related resources
│ Merge with another resource
└────────────────────────────
```

**Design Note:** Context menu to be designed in Figma per user's requirement

---

## 3. Key User Workflows

### 3.1 Adding a New Resource

**User Decision:** To be designed in Figma (Q67)

**Workflow Options Under Consideration:**
- Quick input for links (minimal interaction)
- Detailed form for complex scenarios (subcategories, notes, etc.)
- Drag-and-drop for files

**To be confirmed from Figma design:**
- Exact UI/UX flow
- Form field organization
- Preview before saving
- Success feedback

---

### 3.2 Search & Filter Interaction

**Decision:** Combo A + C (Simple search bar + Chat-based search)

**Implementation:**

#### Option A: Simple Search Bar (Primary)
```
[🔍 Search resources...]  [⊕ Filters ▼]

As user types:
- Instant search results appear below
- 0.5s debounce to avoid excessive queries
- Show: Category badge, counter, date
- Highlight matching text in results
```

**Results Display:**
```
┌─────────────────────────────┐
│ [AI] RAG Tutorial   ⭐⭐⭐   │
│ 5 saves, Added 2d ago       │
├─────────────────────────────┤
│ [AI] Advanced RAG Paper     │
│ 2 saves, Added 10d ago      │
├─────────────────────────────┤
│ [Finance] RAG in Trading    │
│ 1 save, Added 1mo ago       │
└─────────────────────────────┘
```

#### Option C: Chat-Based Search (Secondary)
```
[💬 Go to Chat tab]

User: "Show me AI papers from last month"
System: "Found 12 papers tagged AI, created in March"
         [RAG Tutorial] [ML Survey] [Transformers]...

User: "Only the ones I favorited"
System: "Filtered to 2 papers: [RAG Tutorial] [ML Survey]"
```

**Design Notes:**
- Both search methods act on the same backend query engine
- Results are instantaneous in simple search
- Chat search provides extra context and follow-up capability

---

### 3.3 Resource Interaction & Viewing

**To be designed in Figma (Q66, Q67)**

**Expected Features:**
- Detail panel showing resource metadata
- Action buttons (favorite, priority, archive, delete)
- Related resources section
- Notes/annotations (if added)
- Calendar events linked to this resource

---

## 4. Notification System

### 4.1 Notification Architecture (Q69 — Pending Figma Design)

**Decision:** Hybrid (Toast + Panel + Modal)

**Types of Notifications:**

| Type | When | Show As | Auto-dismiss |
|---|---|---|---|
| Success | Resource added, saved | Toast | 3s |
| Error | Processing failed, network error | Toast + Modal | No |
| Info | Background task complete | Toast | 5s |
| Action Required | Needs categorization | Modal | No |
| Warning | Resource expiring soon | Toast | 5s |
| Persistent | Multiple changes | Panel | No |

**Visual Hierarchy:**
```
Modal (blocks interaction)
  ↓ (high importance/action needed)
Toast (bottom-right, auto-dismiss)
  ↓ (medium importance/quick feedback)
Panel (persistent, low importance/informational)
  ↓ (historical, low importance)
```

**Status:** ⏳ Awaiting Figma design from user (see `reminder.md`)

**Implementation Notes:**
- Build as standalone Zustand store + React component
- QueUe system prevents notification overflow
- Max 3 toasts at once, older ones fade out
- Bell icon [🔔 N] shows unread notification count

---

## 5. Responsive Design

### 5.1 Responsiveness Strategy

**Decision:** Fluid Layout + Collapsing Sidebar + Responsive Panels (B + D combo)

**Breakpoints:**

| Screen Size | Layout | Sidebar | Behavior |
|---|---|---|---|
| > 1920px | 3-column (Sidebar + Graph + Details) | Fixed expanded | All controls visible |
| 1366-1920px | 2-column (Sidebar + Graph) | Fixed expanded | Details slide in |
| 1024-1366px | 2-column | Collapsed (icons) | Click to expand |
| 768-1024px | Stack vertical | Sidebar modal | Toggle menu |
| < 768px | Single column | Full-screen drawer | Hamburger menu |

**Graph Responsiveness:**
```
Large screens (>1920):
  ┌──────────────────────────────┐
  │ [Node details]   [3D Graph]  │
  │                   [Controls] │
  └──────────────────────────────┘

Medium screens (1024-1920):
  ┌──────────────────┐
  │     3D Graph     │ ← Details slide in on click
  │   [Controls]     │
  └──────────────────┘

Small screens (<1024):
  ┌────────────┐
  │  3D Graph  │ ← Simplified controls, pinch zoom
  │ (Simplified)
  └────────────┘
```

**Fluid Scaling:**
- Grid layout uses CSS Grid with `auto-fit` and `minmax()`
- All fonts scale with viewport using `clamp()`
- Components resize proportionally, no abrupt jumps

**Example CSS:**
```css
/* Sidebar width fluid between 200px and 300px */
aside {
    width: clamp(200px, 20vw, 300px);
}

/* Font size scales from 14px to 18px */
body {
    font-size: clamp(14px, 2vw, 18px);
}
```

---

## 6. Accessibility Requirements

**WCAG 2.1 AA Compliance:**
- ✅ Color contrast ratios (4.5:1 for normal text)
- ✅ Keyboard navigation (Tab, arrow keys, Enter)
- ✅ Screen reader support (semantic HTML, ARIA labels)
- ✅ Focus indicators (visible on dark theme)
- ✅ Reduced motion support (prefers-reduced-motion)

**shadcn/ui Benefits:**
- Built on Radix UI (accessible primitives)
- Keyboard navigation baked in
- ARIA attributes pre-configured

---

## 7. Figma Handoff

**To be provided by user via Figma MCP:**

- [ ] Dark mode color palette specifications
- [ ] Sidebar detailed layout with typography
- [ ] 3D graph controls visualization
- [ ] Add resource form/modal design
- [ ] Search results presentation
- [ ] Notification designs (toast, modal, panel)
- [ ] Responsive breakpoint mockups
- [ ] Component library (buttons, inputs, cards, etc.)
- [ ] Animation/transition specifications
- [ ] Detailed interaction flows

**How this will work:**
1. User designs in Figma
2. Figma MCP reads design specs
3. Dev team implements based on specs
4. Iterative refinement as needed

---

## 8. Implementation Roadmap (Phases)

### Phase 1: Core UI
- Sidebar navigation
- Graph 3D viewer (basic)
- Search bar
- Resource list view
- Dark theme implementation

### Phase 2: Interactive Features
- Resource add/edit flows
- Graph controls panel
- Filters and advanced search
- Chat interface basic layout

### Phase 3: Polish
- Animations and transitions
- Notification system
- Responsive refinements
- Accessibility audits

---

*This document will be updated as Figma designs are completed and integrated.*
