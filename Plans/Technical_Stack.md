# Self Systems - Technical Stack

> **Document Purpose:** Defines all technologies, frameworks, libraries, and tools used across every layer of Self Systems.
> **Status:** Confirmed decisions from Q31-Q50

**Last Updated:** March 9, 2026

> **⚠️ Superseded sections:** Several sections below describe the originally-planned stack
> (GORM, Asynq + Redis, sqlite-vec) which was never adopted. The actual implemented stack
> is recorded in [ADR 0019](ADR/0019-actual-stack-vs-planned-stack.md). Sections below are
> annotated inline with "ACTUAL:" notes where they diverge from what's in `go.mod`/`package.json`.

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
│  │     Standard Go Layout + database/sql + in-process queue    │    │
│  └───┬─────────────────────────────────────────────────────┬──┘    │
│      │                                                     │       │
│      ▼                                                     ▼       │
│  ┌───────────────────────┐                       ┌──────────────┐  │
│  │ SQLite                │                       │ deep_processor│  │
│  │ (local + brute-force  │                       │ (in-process   │  │
│  │  cosine vector search)│                       │  goroutine)   │  │
│  └───────────────────────┘                       └──────────────┘  │
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

> **ACTUAL:** the implemented layout differs from the structure below — see
> [CLAUDE.md](../CLAUDE.md) "Architecture" for the current `internal/` package map
> (`domain`, `service`, `repository/{sqlite,postgres}`, `http`, `eventstore`, `sync`,
> `ai`, `extractor`, `gbus`, `desktop`). No `internal/worker`, `internal/api`, or `pkg/`
> directories exist; there is no `cmd/worker` (background processing runs in-process via
> `deep_processor`).

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
├── docker/              ← Docker configs for Redis
├── docker-compose.yml
├── go.mod
└── go.sum
```

*(structure above is the original Phase-1 plan; superseded — see ADR 0019)*

**Why this structure:**
- Clean separation of concerns
- Interface-based design (loosely coupled — swap DB implementations without touching business logic)
- Testable by design (inject dependencies via interfaces)
- Follows Go community conventions

---

### 1.3 Database Access — `database/sql` + Repository Adapters (ACTUAL — supersedes GORM)

> **Planned:** GORM (`gorm.io/gorm`, `gorm.io/driver/sqlite`).
> **Actual:** plain `database/sql` with hand-written repository adapters, using
> `modernc.org/sqlite` (pure Go driver, no CGo). No ORM, no auto-migration framework —
> migrations are versioned SQL files applied at startup. See ADR 0019.

**Usage:**
- SQLite for all relational data (resources, categories, reminders, todos, events)
- Hand-rolled SQL migrations (`internal/repository/sqlite/migrations/`, `internal/repository/postgres/migrations/`)
- Repository structs implement `internal/domain` interfaces directly over `*sql.DB`

```go
// Repository interface (loosely coupled) — internal/domain
type ResourceRepository interface {
    Create(ctx context.Context, r *domain.Resource) error
    FindByID(ctx context.Context, id string) (*domain.Resource, error)
    FindByCategory(ctx context.Context, categoryID string) ([]*domain.Resource, error)
    Update(ctx context.Context, r *domain.Resource) error
    Delete(ctx context.Context, id string) error
}

// SQLite implementation — swappable without changing business logic
type sqliteResourceRepo struct {
    db *sql.DB
}
```

---

### 1.4 Background Job Processing — In-Process Goroutine Queue (ACTUAL — supersedes Asynq/Redis)

> **Planned:** Asynq (`github.com/hibiken/asynq`) backed by Redis.
> **Actual:** `internal/service/deep_processor.go` — an in-process goroutine worker with
> an in-memory channel/queue. No Redis, no separate worker process. See ADR 0019.

**Job types (actual):**
| Task | Mechanism | Trigger |
|------|-----------|---------|
| Skim processing | async goroutine on resource create | immediate |
| Deep processing (extraction, embedding, enrichment, event detection) | `deep_processor` queue worker | enqueued on resource create |
| GBUS feature aggregation | `internal/gbus/aggregator.go`, bounded daily job | cron-style in-process timer |

```
[User saves resource]
        │
        ▼
[Skim goroutine] ──► writes extracted_data
        │
        ▼
[deep_processor in-memory queue]
        │
        ▼
[Deep pass: extraction, embedding, enrichment, event detection]
```

**Worker runs as a goroutine** inside the same Go binary (server or Wails desktop app) — no separate process, no external broker.

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

The product spans **two distinct apps sharing one Go backend**: the full **desktop app** (Wails) and a lighter **mobile companion app** (separate codebase). Reach = 5 operating systems.

**Desktop app (full — Wails):**

| Platform | Status | Notes |
|----------|--------|-------|
| **Windows** | ✅ Target | Primary target; requires WebView2 runtime |
| **Linux** | ✅ Target | Requires `webkit2gtk` |
| **macOS** | ✅ Target | Wails supports `darwin/amd64` + `darwin/arm64`; add build targets + launch test |

Go cross-compiles natively (pure-Go modernc SQLite, no CGO) — same codebase builds for Windows, Linux, and macOS.

**Mobile companion app (lighter — separate codebase):**

| Platform | Status | Notes |
|----------|--------|-------|
| **Android** | 🔄 Future | Companion instance, not a replica |
| **iOS** | 🔄 Future | Companion instance, not a replica |

The mobile app is **not** a port of the desktop app. It is a companion *instance* (like Claude desktop vs. Claude mobile): a thin client to the VPS sync server showing a feature subset — chat + a simplified graph (list/tree view, not the WebGL 2D/3D network) + search. It runs no local extraction/AI pipeline; all processing stays server-side. **Decision (locked): one shared codebase serves both Android + iOS** — not two native apps. Framework (React Native / Flutter / Capacitor) TBD when reached; the companion UI is simple enough that a single shared codebase covers both phones. Hard dependency: the VPS sync server (Postgres) must be deployed and hardened first.

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

### 3.2 Vector Search — Pure-Go Brute-Force Cosine (ACTUAL — supersedes sqlite-vec)

> **Planned:** `github.com/asg017/sqlite-vec-go-bindings` (sqlite-vec, C extension).
> **Actual:** sqlite-vec was evaluated and rejected — the C extension is incompatible
> with `modernc.org/sqlite` (pure Go, no CGo). Implemented instead as brute-force cosine
> similarity in pure Go: `internal/repository/sqlite/vector_repository.go`. Embeddings
> are stored as serialized vectors in SQLite columns; similarity search scans and ranks
> in Go. See ADR 0019.

**Why brute-force over sqlite-vec:**
- No CGo dependency — keeps cross-compilation simple (Windows/Linux/macOS from one toolchain)
- Sufficient performance for personal-scale data (single-user, low resource counts)
- Repository interface unchanged — same swap-out path to Qdrant if needed at scale (Section 10)

**Future migration path:** If performance becomes an issue at scale, vectors can be migrated to Qdrant without changing the rest of the stack (repository interface stays the same).

---

### 3.3 Job Queue Backend — None (ACTUAL — supersedes Redis/Asynq)

> **Planned:** Redis as the Asynq job queue backend.
> **Actual:** No Redis. Background work runs via the in-process `deep_processor`
> goroutine queue (Section 1.4). See ADR 0019.

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

> Models are **config-driven** (`config/config.default.yml`, override via `SS_AI_*` env vars),
> not hardcoded. Current defaults as of `config.default.yml`:

| Task | Provider | Default Model (config-driven) |
|------|----------|-------|
| Content classification (low cost) | OpenAI | `gpt-4o-mini` |
| Deep content analysis (high cost) | OpenAI | `gpt-4o` |
| Deep content analysis | Anthropic | `claude-3-5-sonnet-latest` |
| Deep content analysis | Gemini | `gemini-1.5-flash` |
| Embedding generation | OpenAI | `text-embedding-3-small` |
| Image OCR/classification | OpenAI | `gpt-4o` (vision) |

A heuristic fallback provider exists for offline/no-API-key operation (`internal/ai`).

**Go clients (actual — verify against `go.mod` before adding new providers):**
```
# OpenAI/Anthropic/Gemini clients are wired through internal/ai provider implementations
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
  ├── PostgreSQL (optional for sync)
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
└── ...                             ├── Redis
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
| **Docker Desktop** | Run PostgreSQL + Redis | docker.com |
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
| macOS desktop target | Wails `darwin/amd64` + `darwin/arm64` build + launch test | desktop |
| Mobile companion app (Android + iOS) | One cross-platform codebase (React Native / Flutter / Capacitor) → VPS sync client | post-sync |
| Custom behavioral ML model | Python service (scikit-learn / PyTorch) | 3+ |
| Server deployment | PostgreSQL + NGINX + VPS | 2+ |
| Scalable vector search | Qdrant (replaces sqlite-vec) | 4+ |

Note: the mobile companion is a separate, lighter app (chat + simplified graph), not a Kotlin port of the desktop app; it depends on the VPS sync server being live.

---

*Last Updated: 2026-06-10*
