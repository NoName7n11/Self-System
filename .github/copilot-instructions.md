# Copilot Instructions for Self Systems

## Current repository state

- This repository currently contains planning documents (mainly under `Plans/` and `Old_Context/`), not the full code scaffold yet.
- Treat these files as the source of truth while implementing:
  - `Plans/Development_Workflow.md` for commands, testing strategy, CI/CD, and branching rules
  - `Plans/Technical_Stack.md` for system architecture and runtime topology
  - `Plans/Outline.md` for phase boundaries and end-to-end flow
  - `Plans/UI_Design_Guide.md` and `Plans/reminder.md` for UI and modularity constraints

## Build, test, and lint commands

Documented project commands (from `Plans/Development_Workflow.md` and `Plans/Technical_Stack.md`):

```bash
# Environment/setup
make dev-setup
make dev
docker compose up -d
wails dev

# Lint/format
go fmt ./...
golangci-lint run
npm run lint

# Build
wails build
npm run build
```

Test commands:

```bash
# Full suites
go test ./...
npm test
npx playwright test test/e2e

# Targeted tests
go test -short ./...
go test -run TestIntegration ./test/integration
npm test --testPathPattern="(?<!integration)\.test\.tsx?$"
```

Single-test patterns to use once test files exist:

```bash
go test ./<package> -run ^TestName$
npm test -- <path-to-test-file> -t "<test name>"
npx playwright test <path-to-spec>
```

## High-level architecture

- **Phase 1 is local-first**: a Wails desktop app runs React UI and Go (Gin) backend in one binary via IPC.
- **Data/storage is polyglot**:
  - SQLite for core relational data
  - `sqlite-vec` for vectors/embeddings
  - DGraph (Docker) for graph relationships
  - Redis (Docker) as Asynq queue backend
- **Processing is two-tier asynchronous**:
  - Skim pass (fast/critical queue) for immediate response
  - Deep pass (default FIFO queue) for richer analysis
  - Worker executes inside the same Go process (goroutine) in Phase 1
- **External AI remains cloud-based** (OpenAI + Anthropic) for extraction, classification, embeddings, and chat.
- **Transport split**:
  - Local desktop interactions can call Go through Wails IPC
  - REST handles CRUD and uploads
  - WebSockets push processing/sync updates
- **Phase 2+** introduces VPS-based multi-device sync over WebSockets; Phase 1 remains single-user local with no auth.

## Key conventions

- **Layered Go structure with interfaces first**:
  - `internal/domain` defines business contracts/interfaces
  - `internal/service` orchestrates use cases
  - `internal/repository/*` implements storage adapters
  - keep business logic independent from concrete DB choices
- **Loose-coupling is a hard requirement** (`Plans/reminder.md`):
  - features should be removable with minimal blast radius
  - isolate optional capabilities behind feature flags
  - keep frontend feature logic in dedicated Zustand stores/hooks
- **Configuration precedence is fixed**:
  1. `config.default.yml` defaults
  2. `.env` overrides
  3. environment variables override both
- **Testing layout is intentional**:
  - unit tests adjacent to source
  - integration tests separate (e.g., `/test/integration`)
  - E2E tests under `test/e2e`
  - expected pyramid is 70% unit / 20% integration / 10% E2E
- **Workflow conventions**:
  - GitHub Flow branch families: `feature/*`, `bugfix/*`, `hotfix/*`, `docs/*`, `ci/*`
  - self-review PR workflow and semantic version tags (`vMAJOR.MINOR.PATCH`)
- **UI implementation constraints**:
  - dark-mode-first UI direction
  - shadcn/ui + Tailwind as component/styling baseline
  - final visual specifics should follow Figma handoff via Figma MCP
