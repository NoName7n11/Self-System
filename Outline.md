# Self Systems - Project Outline

> **Document Purpose:** This outline captures my understanding of the project based on the initial plan. It serves as a foundation for further discussion and refinement.

---

## 1. Project Vision

A **personal knowledge management system** that intelligently captures, classifies, and organizes diverse information sources into an interactive graph-based structure, enabling users to build a searchable, interconnected "second brain."

---

## 2. Architectural Decisions (Confirmed)

| Decision | Choice | Notes |
|----------|--------|-------|
| **Deployment Model** | Local-First + Sync Server | Data stored locally on each device; lightweight sync server bridges Windows ↔ Android |
| **AI Processing** | Cloud APIs | External LLM APIs (OpenAI, Anthropic, etc.) |
| **Processing Mode** | Background (80-90%) | Async processing; user doesn't wait for AI completion |
| **Sync Strategy** | Real-Time | Immediate sync when online; offline requests queued locally |
| **Classification Mode** | Smart Auto-Confirm | Auto-save for high-confidence; prompt only for new categories |
| **Project Type** | Full Implementation | Not an MVP; all features built progressively |

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
┌─────────────────────────────────────────────────────────────┐
│                  CLASSIFICATION FLOW                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [New Source] ──► [AI Analyzes Content]                     │
│                          │                                  │
│                          ▼                                  │
│              [Match Against Existing Categories]            │
│                          │                                  │
│           ┌──────────────┼──────────────┐                   │
│           ▼              ▼              ▼                   │
│     [HIGH CONF]    [MEDIUM CONF]   [LOW/NO MATCH]           │
│     (≥ threshold)  (uncertain)    (new category?)           │
│           │              │              │                   │
│           ▼              ▼              ▼                   │
│     [AUTO-SAVE]   [AUTO-SAVE +    [PROMPT USER]             │
│                    FLAG FOR        "Create new              │
│                    REVIEW]         category?"               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Examples:**
- "Apple, Banana, Orange" → **Fruits** (high confidence, auto-save)
- "New JavaScript framework" → **Technology** (high confidence, auto-save)  
- "Quantum Computing in Finance" → Could be Tech OR Finance (prompt user)
- "Underwater Basket Weaving" → No existing category (prompt to create)

#### User Behavior Prediction
The system builds a **User Interest Profile** based on:
- Frequency of sources in each category
- Recency of interactions
- Depth of engagement (just saved vs. revisited multiple times)

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
┌─────────────────────────────────────────────────────────────┐
│               DUPLICATE HANDLING FLOW                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [User shares resource]                                     │
│           │                                                 │
│           ▼                                                 │
│  [Check: Exact URL match?]                                  │
│           │                                                 │
│     ┌─────┴─────┐                                           │
│     ▼           ▼                                           │
│   [YES]       [NO]                                          │
│     │           │                                           │
│     ▼           ▼                                           │
│  [INCREMENT   [Check: Similar content                       │
│   COUNTER]     from different URL?]                         │
│     │               │                                       │
│     │         ┌─────┴─────┐                                 │
│     │         ▼           ▼                                 │
│     │       [YES]       [NO]                                │
│     │         │           │                                 │
│     │         ▼           ▼                                 │
│     │    [CREATE NEW   [CREATE NEW                          │
│     │     NODE + LINK   NODE]                               │
│     │     AS "SIMILAR"]                                     │
│     │                                                       │
│     ▼                                                       │
│  [Notify user: "Resource already saved (x times)"]          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
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

### 3.5 User Interaction Layer
- **Chat Interface** - Natural language queries against the knowledge base
- **Search** - Semantic search across all resources
- **Graph Visualization** - Interactive 2D/3D node graph (neural-network style)
- **Tree View Navigation** - Hierarchical browsing panel

---

## 4. System Architecture

### 4.1 Deployment Topology
```
┌─────────────────────────────────────────────────────────────────┐
│                     LOCAL-FIRST ARCHITECTURE                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐                         ┌─────────────┐       │
│   │ WINDOWS PC  │                         │ ANDROID     │       │
│   │             │                         │ DEVICE      │       │
│   │ ┌─────────┐ │                         │ ┌─────────┐ │       │
│   │ │Local DB │ │◄───── Real-Time ───────►│ │Local DB │ │       │
│   │ │(Primary)│ │         Sync            │ │(Primary)│ │       │
│   │ └─────────┘ │                         │ └─────────┘ │       │
│   │      │      │                         │      │      │       │
│   │      ▼      │                         │      ▼      │       │
│   │ ┌─────────┐ │                         │ ┌─────────┐ │       │
│   │ │ Golang  │ │                         │ │ Local   │ │       │
│   │ │ Backend │ │                         │ │ Service │ │       │
│   │ └─────────┘ │                         │ └─────────┘ │       │
│   └──────┬──────┘                         └──────┬──────┘       │
│          │                                       │              │
│          │         ┌─────────────────┐           │              │
│          │         │   SYNC SERVER   │           │              │
│          └────────►│  (Lightweight)  │◄──────────┘              │
│                    │                 │                          │
│                    │ - Conflict Res. │                          │
│                    │ - Queue Mgmt    │                          │
│                    │ - State Sync    │                          │
│                    └────────┬────────┘                          │
│                             │                                   │
└─────────────────────────────┼───────────────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   CLOUD AI APIs   │
                    │  (OpenAI, etc.)   │
                    │                   │
                    │ - Classification  │
                    │ - Extraction      │
                    │ - Embeddings      │
                    └───────────────────┘
```

### 4.2 Sync Architecture

#### Recommended: Raspberry Pi as Sync Server
```
┌──────────────────────────────────────────────────────────────────┐
│                  RASPBERRY PI SYNC SERVER                        │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   [Windows PC]                              [Android Device]     │
│        │                                          │              │
│        │         ┌────────────────────┐           │              │
│        └────────►│   RASPBERRY PI     │◄──────────┘              │
│                  │   (Home Network)   │                          │
│                  │                    │                          │
│                  │  ┌──────────────┐  │                          │
│                  │  │ Sync Service │  │                          │
│                  │  │  (Golang)    │  │                          │
│                  │  └──────────────┘  │                          │
│                  │  ┌──────────────┐  │                          │
│                  │  │ Central DB   │  │                          │
│                  │  │  (SQLite)    │  │                          │
│                  │  └──────────────┘  │                          │
│                  └─────────┬──────────┘                          │
│                            │                                     │
│              ┌─────────────┴─────────────┐                       │
│              ▼                           ▼                       │
│     [Local Network Sync]        [Remote Access via]              │
│     (When at home)              [Tailscale/ZeroTier/Cloudflare]  │
│                                 (When away from home)            │
└──────────────────────────────────────────────────────────────────┘
```

**Why Raspberry Pi works:**
- Always-on, low power (~5W)
- One-time cost (~$35-70)
- Full control over data
- Can use Tailscale (free) for secure remote access without port forwarding

#### Conflict Resolution: Last-Write-Wins
```
┌─────────────────────────────────────────────────────────────┐
│              LAST-WRITE-WINS STRATEGY                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [Device A: Edit at 10:00 AM]    [Device B: Edit at 10:05]  │
│              │                            │                 │
│              └──────────┬─────────────────┘                 │
│                         ▼                                   │
│               [Both come online]                            │
│                         │                                   │
│                         ▼                                   │
│              [Compare timestamps]                           │
│                         │                                   │
│                         ▼                                   │
│              [Device B wins (10:05 > 10:00)]                │
│                         │                                   │
│                         ▼                                   │
│              [Device A receives Device B's version]         │
│                                                             │
│  Note: Overwritten data stored in "conflict history"        │
│        for potential recovery.                              │
└─────────────────────────────────────────────────────────────┘
```

#### Offline Queue System
```
┌─────────────────────────────────────────────┐
│              OFFLINE QUEUE SYSTEM           │
├─────────────────────────────────────────────┤
│                                             │
│  [User adds source while offline]           │
│              │                              │
│              ▼                              │
│  ┌─────────────────────┐                    │
│  │ LOCAL PENDING QUEUE │                    │
│  │ - Source 1 (URL)    │                    │
│  │ - Source 2 (PDF)    │                    │
│  │ - Source 3 (IMG)    │                    │
│  └──────────┬──────────┘                    │
│             │                               │
│             ▼                               │
│  [Connection Restored]                      │
│             │                               │
│             ▼                               │
│  [PROCESS QUEUE]                            │
│  1. Sync to Pi server                       │
│  2. Send to Cloud AI APIs                   │
│  3. Update graph                            │
│  4. Sync back to all devices                │
└─────────────────────────────────────────────┘
```

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
│  primary_category: "Internships"                            │
│  related_categories: ["Technology", "AI"]                   │
│  counter:       2  (times shared)                           │
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
│  color:         "#3498db" (for visualization)               │
│  created_at:    timestamp                                   │
│  resource_count: 47                                         │
│  position_3d:   {x, y, z} (for graph visualization)         │
└─────────────────────────────────────────────────────────────┘
```

---

## 7. Open Questions (Architecture & Logic)

### Classification Logic
- [ ] What is the confidence threshold for auto-save vs. prompt? (e.g., 0.85?)
- [ ] How does the system improve classification over time? (Feedback loop)

### User Behavior Model
- [ ] How quickly should the model adapt to changing interests?
- [ ] Should users be able to manually adjust their interest profile?
- [ ] How is the behavior model stored and synced?

### Edge/Relationship Logic
- [ ] How is "relatedness" between a resource and non-primary categories calculated?
- [ ] Should the system auto-detect relationships, or only when AI explicitly identifies them?

---

*Last Updated: January 5, 2026*
