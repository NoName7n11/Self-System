# Self Systems - Project Outline

> **Document Purpose:** This outline captures my understanding of the project based on the initial plan. It serves as a foundation for further discussion and refinement.

---

## 1. Project Vision

A **personal knowledge management system** that intelligently captures, classifies, and organizes diverse information sources into an interactive graph-based structure, enabling users to build a searchable, interconnected "second brain."

### Four Core Pillars

```
┌──────────────────────────────────────────────────────┐
│                    SELF SYSTEMS                      │
├──────────────────────────────────────────────────────┤
│                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │   SHARED    │  │  REMINDER   │  │  TO-DO LIST │   │
│  │   MEMORY    │  │   SYSTEM    │  │   MANAGER   │   │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘   │
│         │                │                │          │
│         └────────────────┼────────────────┘          │
│                          │                           │
│                   ┌──────▼──────┐                    │
│                   │  ASSISTANT  │                    │
│                   │   (AI)      │                    │
│                   └─────────────┘                    │
│                                                      │
└──────────────────────────────────────────────────────┘
```

1. **Shared Memory:** Graph-based knowledge base of all saved resources
2. **Reminder System:** Calendar integration with intelligent event detection and notifications
3. **To-Do List Manager:** Personalized task lists derived from saved resources and user goals
4. **AI Assistant:** Conversational interface for managing resources, finding similar content, and answering queries

---

## 2. Architectural Decisions (Confirmed)

| Decision | Choice | Notes |
|----------|--------|-------|
| **Deployment Model** | Local-First (Phase 1) → VPS Sync (Phase 2+) | Phase 1: fully local; Phase 2+: cheap VPS (~$4/mo) for multi-device sync |
| **AI Processing** | Cloud APIs | OpenAI (`gpt-4o-mini` for skim, `gpt-4o` for deep) + Anthropic |
| **Processing Mode** | Phase 1: Skim-only (Deep in Phase 2+) | Skim now for immediate classification; Deep queue/GBUS starts in Phase 2 |
| **Sync Strategy** | Real-Time via WebSockets | REST for CRUD; WebSockets for live sync and processing status |
| **Classification Mode** | Smart Auto-Confirm | Auto-save for high-confidence; prompt only for new/ambiguous categories |
| **Project Type** | Full Implementation | Not an MVP; all features built progressively |
| **Desktop Framework** | Wails (Go + React) | Single binary, Go backend reuse; Windows + Linux + macOS support |
| **Backend** | Go + Gin + GORM + Asynq | Standard Go Layout; loosely coupled via repository interfaces |
| **Databases** | SQLite + sqlite-vec + Redis | Relational + Vector + Queue backend |
| **Authentication** | None (Phase 1) → Google OAuth (Phase 2+) | Local single-user needs no auth; OAuth added with server |
| **Cross-Platform** | Desktop: Windows + Linux + macOS · Mobile: Android + iOS | Desktop = full Wails app (5-OS reach). Mobile = separate lighter **companion** app (chat + simplified graph), thin client to the VPS sync server — not a desktop port. Locked: one shared mobile codebase for Android + iOS (framework TBD). Mobile deferred until sync server is live + hardened. |
| **Testing** | Pyramid (Unit 70% + Integration 20% + E2E 10%) | Go testing + Vitest + Playwright |
| **CI/CD** | GitHub Actions + Docker | Build for Windows+Linux; Docker Compose for local infrastructure |
| **Git Workflow** | GitHub Flow | Feature branches → main; automated tests required before merge |
| **Code Review** | Self-Review PR | Solo development; self-review for accountability & documentation |
| **Release Versioning** | Semantic Versioning (MAJOR.MINOR.PATCH) | Industry standard; enables clear version tracking |
| **Dev Environment** | VS Code Dev Container | One-click setup; Go + Node + Docker pre-configured |
| **Configuration** | Hybrid (config.yml + .env + env vars) | Sensible defaults → local overrides → production overrides |
| **Test Organization** | Unit adjacent + Integration separate | Unit tests co-located with source; integration tests in `/test` |
| **Testing Pyramid** | Complete (70% unit, 20% integration, 10% E2E) | ~250 tests total; 80% coverage target |
| **CI/CD Pipeline** | Standard (Lint + Tests + Build) | GitHub Actions; 6-8 min pipeline; required checks on PRs |
| **Automated Releases** | Build + Release on Git Tag | Tag `v1.2.0` → auto-build Windows/Linux → GitHub Release |
| **Code Quality Gates** | Required Checks (tests + lint must pass) | Branch protection: format, lint, tests, build all required |

---

## 3. Core Capabilities

### 3.1 Resource Ingestion
| Source Type | Examples |
|-------------|----------|
| Hyperlinks | Websites, articles, videos, social media posts |
| Documents | PDFs, Word docs, spreadsheets |
| Media | Images, screenshots |
| Text | Plain text snippets, notes |

### 3.2 Intelligent Processing Pipeline
```
[Input Source] → [Content Extraction] → [AI Analysis] → [Classification Engine] → [Graph Storage]
                                              │
                                              ▼
                                    [User Behavior Model]
```

1. **Content Extraction** - Scrape/parse the source to get raw content
2. **AI Analysis** - Understand context, extract key information (dates, topics, entities)
3. **Classification Engine** - Assign to category with confidence scoring
4. **User Behavior Model** - Learn user interests/patterns over time
5. **Actionable Insights** - Generate todos, calendar events, reminders

### 3.3 Classification Logic

#### Confidence-Based Auto-Classification
```
┌───────────────────────────────────────────────────────┐
│                  CLASSIFICATION FLOW                  │
├───────────────────────────────────────────────────────┤
│                                                       │
│  [New Source] ──► [AI Analyzes Content]               │
│                          │                            │
│                          ▼                            │
│              [Match Against Existing Categories]      │
│                          │                            │
│           ┌──────────────┼──────────────┐             │
│           ▼              ▼              ▼             │
│     [HIGH CONF]    [MEDIUM CONF]   [LOW/NO MATCH]     │
│     (≥ threshold)  (uncertain)    (new category?)     │
│           │              │              │             │
│           ▼              ▼              ▼             │
│     [AUTO-SAVE]   [AUTO-SAVE +    [PROMPT USER]       │
│                    FLAG FOR        "Create new        │
│                    REVIEW]         category?"         │
│                                                       │
└───────────────────────────────────────────────────────┘
```

**Examples:**
- "Apple, Banana, Orange" → **Fruits** (high confidence, auto-save)
- "New JavaScript framework" → **Technology** (high confidence, auto-save)  
- "Quantum Computing in Finance" → Could be Tech OR Finance (prompt user)
- "Underwater Basket Weaving" → No existing category (prompt to create)

**Major Category vs Subcategory:**
- **Major Category:** Where the resource is actually saved (one per resource)
- **Subcategory:** Metadata tags indicating related topics (multiple allowed)
- Subcategories only create visual edges if a corresponding categorical node exists

**Example: "AI in Healthcare" Article**
```
AI Analysis:
  ├─ Major Category: "AI" (resource saved here)
  └─ Subcategories: ["Healthcare", "Medicine"]

Visual Graph:
  ├─ [AI] ←──── strong edge ────→ [Resource Node]
  ├─ [Healthcare] ←── weak edge ──→ [Resource Node]  (if Healthcare node exists)
  └─ (No edge for "Medicine" if no Medicine node exists)
```

#### User Behavior Prediction: GBUS (Generalized Behavioral Understanding System)

**🚨 FUTURE INTEGRATION - MUST-HAVE FEATURE (Phase 3+)**

The system uses a **sophisticated behavioral model** that avoids simplistic assumptions. Key principles:

**Not Single-Minded:**
- Saving a Python article → "Highly likely" interested in Python (not 100% certain)
- Deleting a Python article ≠ No interest in Python
  - Could mean: Resource utilized, content consumed, goal achieved
- **Phase 1-2:** Cloud APIs (OpenAI/Anthropic) for classification and pattern detection
- **Phase 3+:** Custom ML model for behavioral learning (see section 6.3)

**Weighted Signal System:**
```
┌──────────────────────────────────────────────────────────────┐
│               WEIGHTED LEARNING MECHANISM                    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Event Type                    Weight    Impact              │
│  ─────────────────────────────────────────────────────────   │
│  User manual classification     1.0      Strongest signal    │
│  User correction/move           1.0      Override system     │
│  System auto-classification     0.5      Weaker signal       │
│  Resource shared (no confirm)   0.3      Mild interest       │
│  Resource deleted               0.1      Ambiguous (maybe    │
│                                          completed/unused)   │
│  Resource revisited            +0.4      Strong interest     │
│  Counter increment             +0.2      Reaffirmed value    │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Example Scenario:**
1. System classifies "Startup Funding Guide" → Finance (weight: 0.5)
2. User moves to Entrepreneurship (weight: 1.0)
3. **Result:** User judgment prioritized; future similar content biased toward Entrepreneurship

**User Interest Profile (Built from Weighted Signals):**
```
User Interest Profile Example:
┌────────────────────────────────────┐
│  Technology     ████████████  78%  │
│  AI/ML          █████████     65%  │
│  Internships    ██████        42%  │
│  Finance        ███           21%  │
│  Media          ██            15%  │
└────────────────────────────────────┘
```

This profile influences:
- Classification confidence (bias toward likely categories)
- Suggested categories when prompting
- Reminder priority (higher interest = more prominent reminders)

### 3.4 Knowledge Graph Structure

#### Design Principles
- **NO ROOT NODE** - Categorical nodes are independent, floating in 3D space
- **FLAT HIERARCHY** - No sub-categories; only Category → Resource relationships
- **SINGLE OWNERSHIP** - Each resource belongs to ONE primary category
- **VISUAL RELATIONSHIPS** - Related categories shown via edges with distance indicating strength

#### Graph Visualization Model
```
    3D SPACE REPRESENTATION (Top-Down View)
    
                    [TECHNOLOGY]
                     /    |    \
                   /      |      \
              [res1]   [res2]   [res3]
                                   .
                                    .
                                     . (weak edge, large distance)
                                      .
    [FINANCE]                          [AI]
     /   |   \                        / |  \
   /     |     \                    /   |    \
[res4] [res5] [res6]           [res7] [res8] [res9]
                                        |
                                   "AI in Healthcare"
                                        |
                                    . (weak edge)
                                   .
                                  .
                            [HEALTHCARE]
                              /    \
                           [res10] [res11]
```

#### Edge Distance Logic
| Relationship | Distance | Meaning |
|--------------|----------|---------|
| Resource → Primary Category | **Short** | Resource belongs to this category |
| Resource → Related Category | **Long** | Resource is related but doesn't belong here |

#### Example: "AI in Healthcare" Article
```
Primary:    [AI] ←──────── short ────────→ [AI in Healthcare Article]
Related:    [Healthcare] ←── long (dotted) ──→ [AI in Healthcare Article]
```

### 3.5 Duplicate & Counter System

#### Duplicate Detection
```
┌─────────────────────────────────────────────────────┐
│               DUPLICATE HANDLING FLOW               │
├─────────────────────────────────────────────────────┤
│                                                     │
│  [User shares resource]                             │
│           │                                         │
│           ▼                                         │
│  [Check: Exact URL match?]                          │
│           │                                         │
│     ┌─────┴─────┐                                   │
│     ▼           ▼                                   │
│   [YES]       [NO]                                  │
│     │           │                                   │
│     ▼           ▼                                   │
│  [INCREMENT   [Check: Similar content               │
│   COUNTER]     from different URL?]                 │
│     │               │                               │
│     │         ┌─────┴─────┐                         │
│     │         ▼           ▼                         │
│     │       [YES]       [NO]                        │
│     │         │           │                         │
│     │         ▼           ▼                         │
│     │    [CREATE NEW   [CREATE NEW                  │
│     │     NODE + LINK   NODE]                       │
│     │     AS "SIMILAR"]                             │
│     │                                               │
│     ▼                                               │
│  [Notify user: "Resource already saved (x times)"]  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

#### Counter System
```
┌─────────────────────────────────────┐
│         RESOURCE NODE               │
├─────────────────────────────────────┤
│  Title: "Intro to RAG"              │
│  URL: youtube.com/...               │
│  Category: AI                       │
│  Counter: 3  ◄── Shared 3 times     │
│  Priority: HIGH (derived)           │
│  Metadata: {...}                    │
└─────────────────────────────────────┘
```

**Counter Impact:**
- Higher counter = Higher priority in search results
- Higher counter = More prominent in graph visualization
- Can be used for "Most Important Resources" view

### 3.6 Archive System for Stale Resources

#### Auto-Archive Logic
```
┌─────────────────────────────────────────────────┐
│              RESOURCE STALENESS DETECTION       │
├─────────────────────────────────────────────────┤
│                                                 │
│  Triggers for Auto-Archive:                     │
│  ┌────────────────────────────────────────┐     │
│  │ 1. Dead Links (404, domain expired)    │     │
│  │ 2. Event dates passed (e.g., old       │     │
│  │    hackathons, expired internships)    │     │
│  │ 3. Content marked as time-sensitive    │     │
│  │    and deadline exceeded               │     │
│  └────────────────────────────────────────┘     │
│                    │                            │
│                    ▼                            │
│        [Archive Resource]                       │
│                    │                            │
│                    ▼                            │
│        [Remove from 3D Graph Visualization]     │
│                    │                            │
│                    ▼                            │
│        [Notify User: "X resources archived"]    │
│                    │                            │
│           ┌────────┴────────┐                   │
│           ▼                 ▼                   │
│      [User: Keep]     [User: Delete]            │
│           │                 │                   │
│           ▼                 ▼                   │
│    [Restore to      [Permanently                │
│     Graph]          Delete]                     │
│                                                 │
└─────────────────────────────────────────────────┘
```

**Archive Section Features:**
- Separate view in UI for archived resources
- Resources remain searchable in archive
- Can be bulk restored or bulk deleted
- Archive is synced across devices

### 3.7 Search & Query System

#### Ranking Strategy
```
┌───────────────────────────────────────────────────────────────┐
│              SEARCH RESULT RANKING LOGIC                      │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  Default Priority Order:                                      │
│                                                               │
│  1. COUNTER (Primary) ─────────► Most shared = Most important │
│     Weight: 1.0                                               │
│                                                               │
│  ─── Optional Filters (Applied on demand) ───                 │
│                                                               │
│  2. Semantic Similarity  ─────► Match to query                │
│     Weight: 0.8                                               │
│                                                               │
│  3. Recency ──────────────────► Newer resources               │
│     Weight: 0.6                                               │
│                                                               │
│  4. User Interest Profile ────► GBUS alignment                │
│     Weight: 0.5                                               │
│                                                               │
│  5. Engagement History ────────► Revisit frequency            │
│     Weight: 0.4                                               │
│                                                               │
│  ─── Example Search ───                                       │
│                                                               │
│  Query: "machine learning"                                    │
│                                                               │
│  Results (Default - by Counter):                              │
│  1. ML Tutorial (Counter: 5)                                  │
│  2. ML Paper Collection (Counter: 3)                          │
│  3. ML Internship (Counter: 2)                                │
│  4. ML Article (Counter: 1)                                   │
│                                                               │
│  User can apply filters:                                      │
│  [Filter: Recency] [Filter: Category] [Filter: Semantic]      │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

### 3.8 AI Assistant Capabilities

#### Capability Matrix
| Level | Feature | Example |
|-------|---------|----------|
| **Basic** | Keyword search | "Show AI resources" |
| **Semantic** | Natural language understanding | "What did I save about neural networks last month?" |
| **Conversational** | Follow-up context | "Show me more like that" |
| **Analytical** | Insights & patterns | "You've saved 20 internship links but haven't applied to any" |
| **Proactive** | Internet search for similar events | "Found 3 similar hackathons happening next month" |

#### Proactive Discovery (Advanced Feature)
```
┌───────────────────────────────────────────────────┐
│           PROACTIVE RESOURCE DISCOVERY            │
├───────────────────────────────────────────────────┤
│                                                   │
│  Trigger: User ASKS for similar resources         │
│          (NOT automatic background)               │
│                    │                              │
│                    ▼                              │
│         [User: "Find similar hackathons"]         │
│                    │                              │
│                    ▼                              │
│         [System searches internet]                │
│                    │                              │
│                    ▼                              │
│         [Finds: ML Competition, AI Summit, etc.]  │
│                    │                              │
│                    ▼                              │
│         [Presents: "Found 3 similar events"]      │
│                    │                              │
│           ┌────────┴────────┐                     │
│           ▼                 ▼                     │
│      [User: Add]      [User: Ignore]              │
│           │                                       │
│           ▼                                       │
│      [Save as resources]                          │
│                                                   │
└───────────────────────────────────────────────────┘
```

### 3.9 To-Do List Integration

#### Auto-Generated Tasks
```
┌──────────────────────────────────────────────────────────────┐
│              TO-DO LIST AUTO-GENERATION                      │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Source: "Hackathon - Apply by Jan 20"                       │
│            │                                                 │
│            ▼                                                 │
│  [AI extracts actionable items]                              │
│            │                                                 │
│            ▼                                                 │
│  Generated Tasks:                                            │
│  ☐ Review hackathon requirements (Due: Jan 12)               │
│  ☐ Prepare project proposal (Due: Jan 18)                    │
│  ☐ Submit application (Due: Jan 20)                          │
│                                                              │
│  ─── User can: ───                                           │
│  • Modify tasks                                              │
│  • Add custom tasks                                          │
│  • Link tasks to calendar                                    │
│  • Set reminders per task                                    │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### 3.10 User Interaction Layer
- **Chat Interface** - Natural language queries against the knowledge base
- **Search** - Semantic search across all resources
- **Graph Visualization** - Interactive 2D/3D node graph (neural-network style)
- **Tree View Navigation** - Hierarchical browsing panel

---

## 4. System Architecture

### 4.1 Deployment Topology

#### Phase 1 — Fully Local (Current)
```
┌─────────────────────────────────────────────────────────────┐
│              LOCAL-FIRST ARCHITECTURE (Phase 1)             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌─────────────────────────────────────────────────────┐   │
│   │          WINDOWS PC / LINUX MACHINE                 │   │
│   │                                                     │   │
│   │   ┌──────────────────────────────────────────────┐  │   │
│   │   │           WAILS DESKTOP APP                  │  │   │
│   │   │   React UI  ◄──IPC──►  Go Backend (Gin)      │  │   │
│   │   └────────────────────────┬─────────────────────┘  │   │
│   │                            │                        │   │
│   │          ┌─────────────────┼──────────────┐         │   │
│   │          ▼                 ▼              ▼         │   │
│   │    ┌──────────┐    ┌──────────┐           │
│   │    │  SQLite  │    │  Redis   │           │
│   │    │ +sqlite- │    │ (Docker) │           │
│   │    │   vec    │    │  Asynq   │           │
│   │    └──────────┘    └──────────┘           │
│   └─────────────────────────────────────────────────────┘   │
│                             │                               │
└─────────────────────────────┼───────────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   CLOUD AI APIs   │
                    │  OpenAI/Anthropic  │
                    │                   │
                    │ - Classification  │
                    │ - Extraction      │
                    │ - Embeddings      │
                    │ - Chat            │
                    └───────────────────┘
```

#### Phase 2+ — With Sync Server (Future, when mobile companion is added)
```
┌──────────────────┐    WebSocket    ┌──────────────────────┐
│  Wails Desktop   │ ◄────────────► │   VPS (~$4/month)    │
│ (Win/Linux/macOS)│                │                      │
│  ├── SQLite      │                │  Go Sync Service     │
│  └── local cache │                │  ├── PostgreSQL      │
└──────────────────┘                │  └── vector search   │
┌──────────────────┐    WebSocket   │      (brute-force /  │
│ Mobile Companion │ ◄────────────► │       Qdrant later)  │
│ (Android + iOS,  │                └──────────────────────┘
│  thin client)    │   Companion = chat + simplified graph; no local pipeline.
└──────────────────┘
```

### 4.2 Sync Architecture

#### 🔄 Phase 2+ — VPS Sync Server (Deferred)

> **Note:** Raspberry Pi was considered but dropped in favor of a VPS for reliability.
> See `Technical_Stack.md` → Section 6.2 for full details.

#### Current (Phase 1): Local-only, no sync needed

> Full sync architecture documented in `Technical_Stack.md` → Section 6.2

#### Conflict Resolution (Phase 2+): Last-Write-Wins
- Device with the later timestamp wins
- Overwritten data stored in "conflict history" for recovery

#### Offline Queue (Phase 2+)
- Actions taken offline are queued locally (FIFO)
- Processed in order when connection is restored
- Queue persisted to SQLite so it survives app restarts

### 4.3 Background Processing Flow
```
┌──────────────────────────────────────────────────────────────┐
│               BACKGROUND PROCESSING (80-90%)                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  [User shares source] ──► [Immediate ACK to user]            │
│                                  │                           │
│                                  ▼                           │
│                     ┌────────────────────────┐               │
│                     │    TASK QUEUE          │               │
│                     │    (Background)        │               │
│                     └───────────┬────────────┘               │
│                                 │                            │
│              ┌──────────────────┼──────────────────┐         │
│              ▼                  ▼                  ▼         │
│       [Extract Content]  [AI Analysis]    [Generate Embeds]  │
│              │                  │                  │         │
│              └──────────────────┼──────────────────┘         │
│                                 ▼                            │
│                     ┌────────────────────────┐               │
│                     │  Classification Ready  │               │
│                     └───────────┬────────────┘               │
│                                 │                            │
│                    ┌────────────┴────────────┐               │
│                    ▼                         ▼               │
│             [High Confidence]         [Needs Review]         │
│                    │                         │               │
│                    ▼                         ▼               │
│             [Auto-save to           [Notify user,            │
│              Graph silently]         await confirmation]     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## 5. Key User Flows

### 5.1 Sharing a Resource (Android)
```
User shares URL → App receives via Implicit Intent → 
Immediate acknowledgment → Background processing queued →
AI classifies → [High conf: auto-save | Low conf: prompt] →
Graph updated → Sync to other devices
```

### 5.2 Querying the Knowledge Base
```
User asks "What internships did I save?" → 
Chat processes query → Graph traversal → 
Returns relevant nodes → Displays results
```

### 5.3 Event Detection & Reminder
```
URL contains event info → AI extracts date/details → 
Calendar event created → Reminder scheduled → 
User notified before deadline
```

---

## 6. Node Metadata Structure

### Resource Node Schema
```
┌─────────────────────────────────────────────────────────────┐
│                    RESOURCE NODE                            │
├─────────────────────────────────────────────────────────────┤
│  id:            UUID                                        │
│  url:           "https://..."                               │
│  title:         "Hackathon 2026"                            │
│  source_type:   "hyperlink" | "pdf" | "image" | "document"  │
│  primary_category: "Internships"  (major category)          │
│  subcategories: ["Technology", "AI"]  (metadata tags)       │
│  counter:       2  (times shared)                           │
│  archived:      false                                       │
│  archive_reason: null | "dead_link" | "expired" | "manual"  │
│                                                             │
│  ─── STATUS FLAGS ───                                       │
│  status:        "new" | "processing" | "active" | "stale"   │
│  is_priority:   false  (user-flagged as priority)           │
│  is_favorite:   false  (user-marked favorite)               │
│  is_read:       false  (read/unread status)                 │
│                                                             │
│  created_at:    timestamp                                   │
│  updated_at:    timestamp                                   │
│                                                             │
│  ─── EXTRACTED METADATA ───                                 │
│  extracted_data: {                                          │
│    "event_date": "2026-01-25",                              │
│    "deadline": "2026-01-20",                                │
│    "topic": "AI Innovation",                                │
│    "location": "Online",                                    │
│    "key_points": ["...", "..."]                             │
│  }                                                          │
│                                                             │
│  ─── CALENDAR INTEGRATION ───                               │
│  calendar_event_id: "cal_xyz" (if user linked to calendar)  │
│  reminder_set: true/false                                   │
│                                                             │
│  ─── AI ANALYSIS ───                                        │
│  embedding_vector: [0.12, 0.45, ...] (for semantic search)  │
│  classification_confidence: 0.92                            │
│  summary: "A hackathon focused on..."                       │
└─────────────────────────────────────────────────────────────┘
```

### Category Node Schema
```
┌─────────────────────────────────────────────────────────────┐
│                    CATEGORY NODE                            │
├─────────────────────────────────────────────────────────────┤
│  id:            UUID                                        │
│  name:          "Technology"                                │
│  color:         "#3498db" (for visualization)             │
│  created_at:    timestamp                                   │
│  resource_count: 47                                         │
│  position_3d:   {x, y, z} (for graph visualization)         │
│  is_favorite:   false  (user-marked favorite category)      │
└─────────────────────────────────────────────────────────────┘
```

---

## 6.1 Two-Tier Processing System (Skim → Deep)

### Rationale
Like the human brain, the system first skims resources to quickly organize them, then performs deep processing later. This allows immediate user feedback while maintaining thorough analysis.

```
┌──────────────────────────────────────────────────────────────┐
│              TWO-TIER PROCESSING PIPELINE                    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  [User shares resource] ──► [Immediate ACK]                  │
│                                  │                           │
│                    ┌─────────────┴────────────┐              │
│                    ▼                          ▼              │
│            ═══ TIER 1: SKIM ═══      [Add to deep queue]    │
│         (Fast, 2-5 seconds)                                  │
│                    │                                         │
│            ┌───────┼───────┐                                 │
│            ▼       ▼       ▼                                 │
│      [Extract   [Quick   [Basic                              │
│       Title]    Category] Metadata]                          │
│            │       │       │                                 │
│            └───────┼───────┘                                 │
│                    ▼                                         │
│         [Create "PROCESSING" node]                           │
│         [Visible in graph immediately]                       │
│                    │                                         │
│                    ▼                                         │
│         [User can see node with badge]                       │
│                                                              │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                              │
│            ═══ TIER 2: DEEP ═══                              │
│         (Thorough, 30-60 seconds)                            │
│         [Processes via FIFO queue]                           │
│                    │                                         │
│            ┌───────┼───────┐                                 │
│            ▼       ▼       ▼                                 │
│      [Full      [AI      [Generate                           │
│       Content  Analysis] Embeddings]                         │
│       Extract]           [GBUS Update]                       │
│            │       │       │                                 │
│            └───────┼───────┘                                 │
│                    ▼                                         │
│         [Update node to "ACTIVE"]                            │
│         [Remove processing badge]                            │
│         [Classification refined]                             │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Benefits:**
- **Immediate feedback:** Users see their resource added within seconds
- **Better UX:** No waiting for full AI processing
- **Queue management:** Deep processing can handle API rate limits
- **Graceful degradation:** If AI fails, skim data is still useful

---

## 6.2 Content Extraction Strategy by Resource Type

### Website/Hyperlink
```
┌──────────────────────────────────────────────────────────────┐
│                  WEBSITE EXTRACTION                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Stage 1 (Skim):                                             │
│  • Page title                                                │
│  • Domain/URL                                                │
│  • Meta description                                          │
│  • Quick category guess                                      │
│                                                              │
│  Stage 2 (Deep):                                             │
│  • Full content scraping                                     │
│  • Detect page type:                                         │
│    - Landing page → Extract business/service info            │
│    - Article → Extract author, date, key points              │
│    - Event page → Extract dates, location, registration      │
│    - Product → Extract price, features                       │
│  • Event detection logic:                                    │
│    - Keywords: "hackathon", "deadline", "apply by"           │
│    - Date patterns: "January 25, 2026"                       │
│    - If detected → Auto-create calendar event + reminder     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Document (PDF, Word, etc.)
```
┌──────────────────────────────────────────────────────────────┐
│                  DOCUMENT EXTRACTION                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Stage 1 (Skim):                                             │
│  • Filename                                                  │
│  • File size                                                 │
│  • Page count (if available)                                 │
│  • Document type                                             │
│                                                              │
│  Stage 2 (Deep):                                             │
│  • Size-based processing:                                    │
│                                                              │
│    Small (< 5 pages / < 2MB):                                │
│    → Full text extraction + AI analysis                      │
│                                                              │
│    Medium (5-50 pages / 2-20MB):                             │
│    → Extract first/last pages + headings                     │
│    → Summarize key sections                                  │
│                                                              │
│    Large (> 50 pages / > 20MB):                              │
│    → Extract table of contents                               │
│    → Analyze abstract/introduction                           │
│    → Index for search, don't process all                     │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Image
```
┌──────────────────────────────────────────────────────────────┐
│                   IMAGE EXTRACTION                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Stage 1 (Skim):                                             │
│  • Filename                                                  │
│  • File type (jpg, png, etc.)                                │
│  • Thumbnail generation                                      │
│                                                              │
│  Stage 2 (Deep):                                             │
│  • OCR (if text visible)                                     │
│  • Image classification (screenshot, diagram, photo, etc.)   │
│  • Object detection (optional)                               │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

---

## 6.3 Behavioral Pattern Modeling System

**🚨 PHASE 3+ FEATURE - MUST-HAVE FOR PRODUCTION**

**Phase 1-2 Approach:** Use cloud API classification confidence + simple weighted scoring for interest tracking

**Phase 3+ Advanced Implementation:**

### Overview
A dedicated machine learning system that infers user interests from holistic behavior patterns, not just direct add/remove actions.

**Key Principle:** Multiple interests coexist; adding "Field Y" resources doesn't mean declining interest in "Field X"

```
┌──────────────────────────────────────────────────────────────┐
│           BEHAVIORAL PATTERN FEATURES                        │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Input Signals (Multi-dimensional):                          │
│                                                              │
│  1. Resource Interaction Patterns                            │
│     • Frequency of saves per category                        │
│     • Time spent viewing resources                           │
│     • Revisit patterns                                       │
│     • Search queries                                         │
│                                                              │
│  2. User Actions                                             │
│     • Manual categorization choices                          │
│     • Favorite/Priority flags                                │
│     • Rename patterns                                        │
│     • Calendar/Todo linking                                  │
│                                                              │
│  3. Temporal Patterns                                        │
│     • Time of day preferences                                │
│     • Seasonality (exam season, job hunting season)          │
│     • Burst vs sustained interest                            │
│                                                              │
│  4. Cross-Category Relationships                             │
│     • Frequently co-occurring topics                         │
│     • Interest clusters (AI + Healthcare often together)     │
│                                                              │
│  5. Meta-Signals                                             │
│     • Favorites percentage per category                      │
│     • Resource quality (counter values)                      │
│     • Deletion context (read → delete vs unread → delete)    │
│                                                              │
│  ═══ Model Output ═══                                        │
│                                                              │
│  Hidden interest profile used for:                           │
│  • Classification suggestions                                │
│  • Search result boosting                                    │
│  • Reminder prioritization                                   │
│  • Proactive discovery targeting                             │
│                                                              │
│  NOT visible to user (backend only)                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Model Architecture Candidates:**
- **Option A:** Gradient Boosting (XGBoost/LightGBM) - Good for tabular features
- **Option B:** LSTM - Captures temporal patterns in user behavior
- **Option C:** Transformer - Attention mechanism for cross-category relationships
- **Option D:** Hybrid ensemble

*Decision pending further research and experimentation*

---

## 6.4 Reminder & Calendar System

### Configuration Options

**User Preferences:**
1. **Reminder Mode:**
   - Manual only (user sets all reminders)
   - System-assisted (system suggests, user approves)
   - Fully automatic (system sets, user can modify)

2. **Notification Channels:**
   - Push notifications only
   - Email only
   - Both push + email

3. **Smart Scheduling:**
   ```
   Event detected: Hackathon on Jan 25
   
   System suggests:
   ☐ 1 week before (Jan 18) - "Start preparing"
   ☐ 3 days before (Jan 22) - "Final review"
   ☐ 1 day before (Jan 24) - "Tomorrow is the event"
   
   User can accept/modify/reject
   ```

4. **Anti-Spam Logic:**
   - Maximum N reminders per day (user configurable)
   - Priority-based: High-priority events get more reminders
   - Consolidation: "You have 3 upcoming events this week"
   - Snooze functionality

---

## 6.5 User Management Capabilities

### Allowed Operations

**Category Management:**
- ✅ Rename categories (updates all linked resources)
- ✅ Delete categories (prompts: "Move resources to?" or "Archive all?")
- ✅ Mark category as favorite (visual indicator in graph)
- ✅ Change category color

**Resource Management:**
- ✅ Rename resources
- ✅ Delete resources (moves to trash, 30-day recovery)
- ✅ Mark as Priority (affects processing queue, search ranking)
- ✅ Mark as Favorite (affects behavioral model)
- ✅ Mark as Read/Unread
- ✅ Move to different category
- ✅ Add/remove subcategories manually

**Bulk Operations:** (See Future Considerations for optimization strategies)

---

## 6.6 Search & Highlight Behavior

When user searches (via chat or graph search):

```
Query: "RAG"

Visual Effect:
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│     [Category: AI]  (full brightness)                        │
│          │                                                   │
│       ┌──┼──┐                                                │
│       │  │  │                                                │
│    ░░RAG Tutorial░░  ← HIGHLIGHTED (pulsing/glow)            │
│       │  │  │                                                │
│    [Other AI]    ← DIMMED (30% opacity)                      │
│    [Resources]                                               │
│                                                              │
│                                                              │
│    [Category: ML]  ← DIMMED (30% opacity)                    │
│       │                                                      │
│    ░░RAG Paper░░   ← HIGHLIGHTED                             │
│       │                                                      │
│    [Other ML]      ← DIMMED                                  │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Highlight Rules:**
- Exact matches: Bright highlight + glow effect
- Semantic matches: Subtle highlight
- Unrelated nodes: 30% opacity (dimmed but still visible for context)
- Clear search: Return all nodes to normal brightness

---

## 7. Decisions Summary & Open Questions

### ✅ Confirmed Decisions (Q17-Q30)

**Resource Lifecycle:**
- [x] Processing badge visible during analysis
- [x] Priority flag available (user-set)
- [x] Read/Unread status tracking
- [x] Favorite flag (affects behavioral model)

**User Management:**
- [x] Can rename categories and resources
- [x] Can delete categories/resources
- [x] Favorites affect behavioral patterns

**GBUS (Behavioral Model):**
- [x] **Phase 1-2:** Cloud APIs only (OpenAI/Anthropic) for classification
- [x] **Phase 3+:** Custom ML model (MUST-HAVE for production)
- [x] ML-based interest profiling (deferred)
- [x] Hidden from users (backend suggestion algorithm)
- [x] Multi-interest aware (not zero-sum)
- [x] Uses holistic behavior patterns
- [x] No direct UI for users to see/adjust profile

**Content Extraction:**
- [x] Two-tier: Skim (fast) → Deep (thorough)
- [x] Website: Basic info + event detection
- [x] Documents: Size-based processing depth
- [x] Images: OCR + classification

**Reminders:**
- [x] Manual, assisted, or automatic modes (user choice)
- [x] Push, email, or both (user preference)
- [x] Anti-spam logic (max per day, consolidation)

**Search & Visualization:**
- [x] Highlight matches, dim unrelated
- [x] No cross-resource relationship links needed (chat handles this)

**Processing:**
- [x] FIFO queue for deep processing
- [x] Two-tier for immediate feedback

**Export:**
- [x] JSON format

**Subcategories:**
- [x] No hard limit, but prompt user if category gets overcrowded

### 🔄 Pending Decisions

**Classification Logic:**
- [ ] What is the confidence threshold for auto-save vs. prompt? (e.g., 0.85?)
- [ ] Multi-category resources: Choose primary, create hybrid, or system decides?

**Behavioral Model:**
- [ ] Which ML architecture? (XGBoost, LSTM, Transformer, Hybrid?)
- [ ] Training data collection strategy?
- [ ] How often to retrain the model?
- [ ] Behavioral pattern calculation methodology (needs dedicated design)

### 📋 Deferred to Future Considerations File

See `Future_Considerations.md` for:
- Graph visualization performance optimization
- Batch operations implementation strategy  
- Security & privacy features
- Archive scheduling strategy
- To-do list integration approach
- Offline semantic search caching

---

*Last Updated: February 15, 2026*
