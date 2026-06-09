# Self Systems — New UI Design Plan

## Overview
This document outlines the planning and specifications for the updated User Interface of Self Systems. The goal is to evolve the current local development UI (`http://127.0.0.1:5173/`) into a more polished, modern, and accessible design based on the core architectural requirements (collapsible sidebars, fluid 3-column layout, dark mode, Shadcn+Tailwind).

All visual wireframes and high-fidelity designs will be iterated on in the **Figma File: "AI Self System"**, specifically under the `UI_Agent` page.

## Current State Analysis (Baseline)
- The existing prototype has functional sidebars configured for smooth Expansion/Retraction (`Left_Bar` and `Right_Bar`).
- The center workspace currently flexes to accommodate the sidebars using a dark placeholder layout.
- The localhost UI includes the left nav items: Graph, Search, Chat, Tasks, Settings, plus Favorites and Recent sections.
- The `UI_Design_Guide.md` directs a responsive 3-column layout favoring a dark theme, using `shadcn/ui` components for consistency.

## Confirmed Direction (From User)
- **Color Mood**: Neutral charcoal/graphite dark theme.
- **Typography**: Compact, IDE-like feel with generic fonts for now.
- **Spacing**: Breathable spacing (not overly dense).
- **Layout**: 3-panel layout; both sidebars are collapsible and resizable.
- **Framing**: Central workspace framed with a top bar and a control dock.
- **Styling**: Subtle rounding + borders + soft shadows.
- **Left Nav Content**: Mirror the localhost UI for initial pass.
- **Right Panel**: File-explorer style layout, but keep content blank for now.
- **Graph Visual Reference**: Use [example.jpg](../../example.jpg) as inspiration for graph layout/structure.
- **Component System**: Use shadcn/ui concepts/components.
- **Process**: Expect review and iterative changes.

## Design Goals & Phases

### 1. Theming & Typography
- **Palette**: Neutral charcoal/graphite base with subtle contrast between panels; keep accents minimal and consistent.
- **Typography**: Generic font stack for now; compact sizing with readable line-height and breathable spacing.
- **Tokens**: Define CSS variables for backgrounds, borders, shadows, and text levels (primary, secondary, muted).

### 2. The Left Sidebar (Navigation & Favorites)
- **Collapsed State**: Icons only; narrow width with tooltip on hover.
- **Expanded State**: Full labels and sections for Favorites and Recent.
- **Content**: Graph, Search, Chat, Tasks, Settings; Favorites and Recent lists from localhost.
- **Interactions**: Hover states, active state highlights, resizable edge handle.

### 3. The Right Sidebar (Details & Metadata)
- **Layout**: File-explorer style structure, but keep content blank initially.
- **Behavior**: Smooth expand/collapse + resizable edge handle.
- **Future**: Slot for node metadata, categories, and contextual actions.

### 4. The Central Workspace (3D Graph & Content)
- **Graph Canvas**: Full height/width within the center panel, inspired by example.jpg.
- **Top Bar**: Framed header containing search, sync status, and primary actions.
- **Control Dock**: A compact dock for graph actions (2D/3D toggle, zoom, reset, filters).
- **Layout**: Supports responsive scaling and maintains alignment under resize.

### 5. Dialogs, Modals, & Toasts
- **Components**: shadcn/ui dialogs, toasts, dropdowns, and tabs where appropriate.
- **Styling**: Minimal, understated, consistent with graphite palette.

## Next Steps / Open Questions
1. Choose an accent color (single accent).
2. Confirm default widths for collapsed/expanded panels (left/right).
3. Control dock placement: bottom-center.
4. Top bar contents: search + sync status + add resource + settings.
5. Resizable handle styling: minimal grab line.

## Decisions (Locked)
- **Accent**: Single accent color (to be set in first Figma pass).
- **Control Dock**: Bottom-center.
- **Top Bar**: Search + sync status + add resource + settings.
- **Resize Handles**: Minimal grab line.
