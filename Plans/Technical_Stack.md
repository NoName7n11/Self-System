# Self Systems - Technical Stack

> **Document Purpose:** Defines all technologies, frameworks, libraries, and tools used across every layer of Self Systems.
> **Status:** Confirmed decisions from Q31-Q50

**Last Updated:** March 9, 2026

---

## Stack Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SELF SYSTEMS STACK                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    DESKTOP APP (Wails)                      │    │
│  │         React + TypeScript + Zustand + react-force-graph    │    │
│  └──────────────────────────────┬──────────────────────────────┘    │
│                                 │ IPC Bridge (Wails)                │
│  ┌──────────────────────────────▼──────────────────────────────┐    │
│  │                   GO BACKEND (Gin)                          │    │
│  │         Standard Go Layout + GORM + Asynq                   │    │
│  └───┬───────────────┬──────────────────┬───────────────────┬──┘    │
│      │               │                  │                   │       │
│      ▼               ▼                  ▼                   ▼       │
│  ┌───────┐     ┌──────────┐     ┌────────────┐     ┌──────────────┐ │
│  │SQLite │     │  DGraph  │     │ sqlite-vec │     │    Redis     │ │
│  │(local)│     │  (graph) │     │ (vectors)  │     │   (Asynq)    │ │
│  └───────┘     └──────────┘     └────────────┘     └──────────────┘ │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                     REST + WebSocket API
                                  │
              ┌───────────────────▼────────────────────┐
              │          CLOUD AI APIs                  │
              │  OpenAI / Anthropic                     │
              │  - Classification, Extraction           │
              │  - Embeddings, Chat                     │
              └────────────────────────────────────────┘
```

---

## 1. Backend — Golang

### 1.1 Web Framework
| Setting | Choice |
|---------|--------|
| **Framework** | **Gin** |
| **Reason** | Feature-rich, excellent middleware support (auth, logging, rate limiting, CORS), large community, fastest performance of opinionated frameworks |

```
github.com/gin-gonic/gin
```

**Middleware stack:**
- `gin-contrib/cors` — Cross-origin requests (WebSocket/frontend)
- `gin-contrib/requestid` — Request tracing
- Custom logger middleware
- Rate limiting middleware (production)

---

### 1.2 Project Structure — Standard Go Layout

```
self-systems/
├── cmd/
│   ├── server/          ← Main entry point (Wails app)
│   └── worker/          ← Background job worker (Asynq)
│
├── internal/
│   ├── api/             ← Gin route handlers, middleware
│   │   ├── handlers/
│   │   └── middleware/
│   ├── domain/          ← Core business logic, interfaces
│   │   ├── resource/
│   │   ├── category/
│   │   ├── reminder/
│   │   └── todo/
│   ├── service/         ← Use cases, orchestration
│   ├── repository/      ← Database access layer (implements domain interfaces)
│   │   ├── sqlite/
│   │   └── dgraph/
│   ├── worker/          ← Asynq task handlers
│   │   ├── tasks/
│   │   └── handlers/
│   └── ai/              ← AI provider clients (OpenAI, Anthropic)
│
├── pkg/
│   ├── config/          ← App configuration
│   ├── logger/          ← Structured logging
│   └── utils/           ← Shared utilities
│
├── frontend/            ← React app (Wails frontend)
│   ├── src/
│   └── public/
│
├── migrations/          ← Database migration files
├── docker/              ← Docker configs for DGraph, Redis
├── docker-compose.yml
├── go.mod
└── go.sum
```

**Why this structure:**
- Clean separation of concerns
- Interface-based design (loosely coupled — swap DB implementations without touching business logic)
- Testable by design (inject dependencies via interfaces)
- Follows Go community conventions

---

### 1.3 ORM — GORM

```
gorm.io/gorm
gorm.io/driver/sqlite
```

**Usage:**
- SQLite for all relational data (resources, categories, reminders, todos)
- Auto-migration support
- Soft deletes (30-day recovery for deleted resources)
- Hooks for timestamps (`created_at`, `updated_at`)

**Note:** DGraph is accessed via its own Go client (not GORM), as it is a graph database.

```go
// Example — Repository interface (loosely coupled)
type ResourceRepository interface {
    Create(ctx context.Context, r *domain.Resource) error
    FindByID(ctx context.Context, id string) (*domain.Resource, error)
    FindByCategory(ctx context.Context, categoryID string) ([]*domain.Resource, error)
    Update(ctx context.Context, r *domain.Resource) error
    Delete(ctx context.Context, id string) error
}

// SQLite implementation — swappable without changing business logic
type sqliteResourceRepo struct {
    db *gorm.DB
}
```

---

### 1.4 Background Job Processing — Asynq

```
github.com/hibiken/asynq
```

**Depends on:** Redis (runs as local Docker container)

**Job types:**
| Task | Queue | Priority |
|------|-------|----------|
| Skim processing (Tier 1) | `critical` | Immediate |
| Deep processing (Tier 2) | `default` | FIFO |
| Embedding generation | `default` | FIFO |
| Archive scan | `low` | Scheduled (weekly) |
| Link health check | `low` | Scheduled |

```
[User saves resource]
        │
        ▼
[Asynq: enqueue skim task] ──► Redis queue
        │
        ▼
[Worker picks up task]
        │
   ┌────┴────┐
   ▼         ▼
[Skim]   [Enqueue deep task]
```

**Worker runs as a separate goroutine** inside the same Go binary (Wails app), no separate process needed in Phase 1.

---

### 1.5 Authentication

**Phase 1:** No authentication required
- Single-user, local app
- No network exposure (all local)

**Phase 2+ (Future Integration — Must Have):**
- **Google OAuth 2.0** (`golang.org/x/oauth2`)
- "Login with Google" for user identification
- JWT tokens for session management after OAuth
- Required when adding multi-device sync with a server

---

### 1.6 Cross-Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| **Windows** | ✅ Phase 1 | Primary target |
| **Linux** | ✅ Phase 1 | Go + Wails fully supported, requires `webkit2gtk` |
| **macOS** | ⚠️ Possible | Not a target, but Wails supports it |
| **Android** | 🔄 Phase 3+ | Deferred, requires separate app |

Go cross-compiles natively — same codebase builds for Windows and Linux.

---

## 2. Desktop Application

### 2.1 Framework — Wails

```
github.com/wailsapp/wails/v2
```

**Why Wails over alternatives:**
| Framework | Language | App Size | Fit |
|-----------|----------|----------|-----|
| **Wails** | Go + Web | ~10-20MB | ✅ Perfect — Go backend reuse |
| Electron | JS/TS | ~150MB | ❌ Heavy, separate backend |
| Tauri | Rust + Web | ~5-10MB | ❌ Rust learning curve |
| Qt | C++/Go | Native | ❌ Complex UI development |

**How Wails works:**
```
┌─────────────────────────────────────────┐
│           WAILS APPLICATION             │
│                                         │
│  ┌─────────────┐    IPC     ┌─────────┐ │
│  │  React UI   │ ◄────────► │  Go     │ │
│  │ (WebView2)  │            │ Backend │ │
│  └─────────────┘            └─────────┘ │
│                                         │
│  Single binary, no browser needed       │
└─────────────────────────────────────────┘
```

Frontend calls Go functions directly — no HTTP calls needed for local operations.

---

### 2.2 Frontend Framework — React + TypeScript

```json
{
  "react": "^18.x",
  "typescript": "^5.x",
  "vite": "^5.x"
}
```

**Why React:** Largest ecosystem, best library support for complex UIs (3D graph, real-time updates), excellent TypeScript support.

**Build tool:** Vite (fast HMR, instant builds)

---

### 2.3 3D Graph Visualization — react-force-graph

```
npm install react-force-graph
```

**Features used:**
- `ForceGraph3D` — 3D node graph rendering (WebGL via Three.js)
- `ForceGraph2D` — Fallback for lower-end machines
- Custom node rendering (category colors, resource status badges)
- Custom link rendering (strong/weak edges, different thickness)
- Physics simulation (nodes naturally space themselves out)

```typescript
// Example usage
import ForceGraph3D from 'react-force-graph';

<ForceGraph3D
  graphData={graphData}
  nodeLabel={node => node.title}
  nodeColor={node => node.category.color}
  linkWidth={link => link.isPrimary ? 3 : 1}
  linkOpacity={link => link.isPrimary ? 0.9 : 0.3}
  onNodeClick={handleNodeClick}
/>
```

**Performance thresholds:** (see `Future_Considerations.md` → Section 1)

---

### 2.4 State Management — Zustand

```
npm install zustand
```

**Why Zustand over Redux:** Lightweight, no boilerplate, simple API, supports async actions natively.

**Stores:**
```typescript
useGraphStore        // graph data, node positions, selected nodes
useResourceStore     // resource list, filters, pagination
useCategoryStore     // categories, color management
useSearchStore       // search query, results, highlight state
useUIStore           // sidebar state, active view, modals
useSyncStore         // sync status, offline queue indicator
```

---

### 2.5 UI Component Library

**Recommendation (to be confirmed in UI/UX round):**
- **shadcn/ui** — Headless, fully customizable, Tailwind-based
- **Tailwind CSS** — Utility-first styling

---

## 3. Databases

### 3.1 Local Relational Storage — SQLite

**Used for:** All structured data (resources, categories, todos, reminders, user settings)

```
github.com/mattn/go-sqlite3       ← CGo driver
modernc.org/sqlite                ← Pure Go driver (recommended, no CGo)
```

**Accessed via:** GORM with SQLite driver

**Schema location:** `migrations/` (versioned SQL files)

**Why SQLite:**
- Zero configuration, single file
- ACID compliant
- Sufficient for single-user local application
- Works identically on Windows and Linux

---

### 3.2 Graph Database — DGraph

**Used for:** Category→Resource relationships, graph traversal, edge storage

```
github.com/dgraph-io/dgo/v230     ← Official Go client
```

**Deployment:** Docker container (local)

```yaml
# docker-compose.yml
services:
  dgraph-zero:
    image: dgraph/dgraph:latest
    ports: ["5080:5080", "6080:6080"]

  dgraph-alpha:
    image: dgraph/dgraph:latest
    ports: ["8080:8080", "9080:9080"]
```

**Why DGraph:**
- Written in Go (fits the stack)
- GraphQL + DQL query interface
- Designed for graph traversal (category↔resource relationships)
- Open source, self-hosted

**What DGraph stores:**
```graphql
type Category {
    id: ID!
    name: String! @index(exact)
    color: String
    resources: [Resource] @hasInverse(field: category)
}

type Resource {
    id: ID!
    title: String! @index(fulltext)
    category: Category
    relatedCategories: [Category]   # weak edges
    edgeStrength: Float             # primary=1.0, related=0.3
}
```

---

### 3.3 Vector Search — sqlite-vec

**Used for:** Semantic search via embedding vectors

```
github.com/asg017/sqlite-vec-go-bindings
```

**Why sqlite-vec over dedicated vector DB:**
- Embedded directly in SQLite — no extra service/Docker container
- Zero additional infrastructure for Phase 1
- Sufficient performance for personal-scale data
- Vectors stored alongside metadata in same database

**Usage:**
```go
// Store embedding alongside resource
db.Exec(`INSERT INTO vec_resources(resource_id, embedding) VALUES (?, ?)`,
    resourceID, serializeVector(embedding))

// Semantic search
rows := db.Query(`
    SELECT resource_id, distance
    FROM vec_resources
    WHERE embedding MATCH ?
    ORDER BY distance LIMIT 10
`, queryEmbedding)
```

**Future migration path:** If performance becomes an issue at scale, vectors can be migrated to Qdrant without changing the rest of the stack (repository interface stays the same).

---

### 3.4 Job Queue Backend — Redis

**Used for:** Asynq job queue backend only

```yaml
# docker-compose.yml
services:
  redis:
    image: redis:alpine
    ports: ["6379:6379"]
```

**Not used for:** Caching, session storage, or anything else in Phase 1

---

## 4. API & Communication

### 4.1 API Style — REST + WebSockets

| Purpose | Protocol |
|---------|----------|
| CRUD operations (resources, categories, todos) | REST (Gin routes) |
| Real-time sync updates | WebSockets (`gorilla/websocket`) |
| AI processing status updates | WebSockets (pushed to client) |
| File upload (PDFs, images) | REST multipart |

```
github.com/gorilla/websocket
```

**WebSocket events:**
```json
{ "type": "resource.processing_complete", "payload": { "id": "...", "status": "active" } }
{ "type": "resource.classification_needed", "payload": { "id": "...", "suggestions": [...] } }
{ "type": "sync.update", "payload": { "changes": [...] } }
```

---

## 5. Cloud AI Integration

### 5.1 AI Providers

| Task | Provider | Model |
|------|----------|-------|
| Content classification | OpenAI | `gpt-4o-mini` (fast, cheap) |
| Deep content analysis | OpenAI / Anthropic | `gpt-4o` / `claude-3-5-sonnet` |
| Embedding generation | OpenAI | `text-embedding-3-small` |
| Chat assistant | OpenAI / Anthropic | `gpt-4o` / `claude-3-5-sonnet` |
| Image OCR/classification | OpenAI | `gpt-4o` (vision) |

**Go clients:**
```
github.com/sashabaranov/go-openai
github.com/anthropics/anthropic-sdk-go
```

**Cost control:**
- Use `gpt-4o-mini` for skim processing (cheap, fast)
- Use full models only for deep processing
- Cache classification results to avoid re-processing identical content

---

## 6. Deployment & Infrastructure

### 6.1 Phase 1 — Fully Local

```
[User's Machine — Windows or Linux]
│
├── Wails Desktop App (single .exe / binary)
│    └── Go Backend (Gin, embedded)
│
└── Docker Compose services:
     ├── DGraph Zero + Alpha
     └── Redis
```

**Running the app:**
```bash
# Start infrastructure
docker compose up -d

# Run Wails app (development)
wails dev

# Build for production
wails build
```

**Data locations:**
```
~/.self-systems/
├── data.db         ← SQLite database (resources, categories, etc.)
└── files/          ← Uploaded documents, images
```

---

### 6.2 Phase 2+ — With Sync Server (Future)

**When:** Android app is added or multi-device sync is required

**Recommended:** Cheap VPS (Hetzner CX22 ~$4/month)

```
[User's Machine]                    [VPS Server — Hetzner]
├── Wails App      ◄─ WebSocket ──► Go Sync Service (Gin)
├── SQLite (local)                  ├── PostgreSQL (central DB)
└── ...                             ├── DGraph
                                    ├── Redis
                                    └── sqlite-vec / Qdrant

[Android App]      ◄─ WebSocket ──► (same VPS)
```

**Why VPS over Raspberry Pi for production:**
- Always-on regardless of home internet/power
- Fixed public IP (no DDNS)
- Managed data center reliability
- Same or lower cost (~$4/month vs Pi accessories + electricity)
- Easy Docker Compose deployment

**VPS Docker Compose will add:**
- PostgreSQL (replaces SQLite for multi-device central store)
- NGINX (reverse proxy, SSL termination)
- Certbot (Let's Encrypt SSL)

---

### 6.3 CI/CD — Docker + GitHub Actions

**Repository:** GitHub (private)

**GitHub Actions workflows:**

```yaml
# .github/workflows/
├── build.yml          ← Build Wails app for Windows + Linux on every PR
├── test.yml           ← Run Go tests + React tests
└── release.yml        ← Build + attach binaries to GitHub Release on tag push
```

**Build matrix:**
| OS | Architecture | Output |
|----|-------------|--------|
| Windows | amd64 | `self-systems-windows-amd64.exe` |
| Linux | amd64 | `self-systems-linux-amd64` |

**Wails cross-compilation:** Uses `wails build` with GOOS/GOARCH flags

---

## 7. Development Tools

### 7.1 Required Tools

| Tool | Purpose | Install |
|------|---------|---------|
| **Go 1.22+** | Backend language | golang.org |
| **Node.js 20+ + npm** | React frontend | nodejs.org |
| **Wails CLI** | Desktop app build | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| **Docker Desktop** | Run DGraph + Redis | docker.com |
| **Git** | Version control | git-scm.com |
| **VS Code** | Primary editor | Already installed ✅ |

### 7.2 VS Code Extensions (Recommended)

| Extension | Purpose |
|-----------|---------|
| `golang.go` | Go language support |
| `esbenp.prettier-vscode` | Code formatting |
| `dbaeumer.vscode-eslint` | JS/TS linting |
| `ms-azuretools.vscode-docker` | Docker management |
| `bradlc.vscode-tailwindcss` | Tailwind CSS intellisense |
| `DgraphLabs.vscode-dgraph` | DQL syntax support |

### 7.3 Design Tools

| Tool | Purpose |
|------|---------|
| **Figma** | UI/UX mockups and wireframes |
| **draw.io** | Architecture and flow diagrams |

### 7.4 API Testing

| Tool | Purpose |
|------|---------|
| **Postman** | REST API testing + documentation |
| **Hoppscotch** | Lightweight alternative (browser-based) |

### 7.5 Database Management

| Tool | Purpose |
|------|---------|
| **DBeaver** | SQLite database viewer/editor |
| **Ratel (DGraph UI)** | Graph database browser (built into DGraph) |
| **RedisInsight** | Redis queue monitoring |

---

## 8. Testing Strategy

### 8.1 Test Pyramid

```
        ┌─────────────┐
        │    E2E      │  10% — Full user flows
        │  (Playwright│
        └──────┬──────┘
        ┌──────┴──────────┐
        │  Integration    │  20% — API endpoints, DB operations
        │  (Go testing)   │
        └──────┬──────────┘
        ┌──────┴──────────────────┐
        │      Unit Tests         │  70% — Business logic, utils
        │   (Go testing + Vitest) │
        └─────────────────────────┘
```

### 8.2 Test Tools

| Layer | Tool | Language |
|-------|------|----------|
| Go unit + integration | `testing` (stdlib) + `testify` | Go |
| React component tests | `Vitest` + `React Testing Library` | TypeScript |
| E2E desktop tests | `Playwright` | TypeScript |

---

## 9. Pending Decisions (To Be Decided in UI/UX Round)

- [ ] UI component library (shadcn/ui vs others)
- [ ] Color theme / design language
- [ ] Navigation layout (sidebar, top bar, etc.)
- [ ] Graph control panel design
- [ ] Mobile-first vs desktop-first for UI design

---

## 10. Future Stack Additions (Deferred)

| Feature | Addition Needed | Phase |
|---------|----------------|-------|
| Google OAuth login | `golang.org/x/oauth2` | 2+ |
| Android app | Kotlin + Jetpack Compose + Room | 3+ |
| Custom behavioral ML model | Python service (scikit-learn / PyTorch) | 3+ |
| Server deployment | PostgreSQL + NGINX + VPS | 2+ |
| Scalable vector search | Qdrant (replaces sqlite-vec) | 4+ |

---

*Last Updated: March 9, 2026*
