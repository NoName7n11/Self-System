# Self Systems

Self Systems is a local-first knowledge system backend that captures resources, classifies them, and exposes structured APIs for resources, categories, reminders, todos, graph data, and unified chat commands.

Current repository status: Phase 1 backend and operations closeout are in place (Go + Gin + SQLite, integration tests, CI, and release automation).

## Architecture Snapshot

- Runtime: Go 1.22 service with Gin HTTP routes
- Storage: SQLite for relational data
- AI abstraction: heuristic, OpenAI, Anthropic, and Gemini providers behind a manager
- API contract: OpenAPI spec in api/openapi.yaml
- Local infrastructure: Redis and DGraph Docker services for planned queue and graph topology

## Prerequisites

- Go 1.22+
- Docker Desktop (optional, for Redis and DGraph)
- GNU Make (optional, convenience commands)

## Quick Start

1. Install dependencies:
   - go mod tidy
2. Create local environment file:
   - PowerShell: Copy-Item .env.example .env
   - Bash: cp .env.example .env
3. Start development environment:
   - make dev
4. API starts on http://127.0.0.1:8080 by default.

If you do not use Make, run:

- docker compose up -d
- go run ./cmd/server

## Configuration Precedence

Configuration is loaded in this order:

1. config/config.default.yml
2. .env
3. Environment variables (prefix SS_)

Example override:

- SS_APP_PORT=9090
- SS_DATABASE_PATH=./data/local.db

## Common Commands

| Task | Command |
|---|---|
| Setup dependencies | make dev-setup |
| Start API + local infra | make dev |
| Start only API | make run |
| Start Docker services | make docker-up |
| Stop Docker services | make docker-down |
| Run all tests | make test |
| Run integration tests | make integration-test |
| Format Go files | make lint |
| CI-equivalent local checks | make ci |
| Build server binary | make build |
| Clean Go cache outputs | make clean |

## Testing

- Full test suite: go test ./...
- Integration tests only: go test ./test/integration/...

Testing layout follows:

- Unit tests adjacent to source packages
- Integration tests in test/integration

## CI and Releases

- CI workflow: .github/workflows/ci.yml
   - Trigger: push and pull_request to master and main
  - Checks: formatting, tests, build
- Release workflow: .github/workflows/release.yml
  - Trigger: git tags that match vMAJOR.MINOR.PATCH
  - Output: Linux and Windows binaries plus SHA256 checksums in GitHub Releases
- Release checklist workflow: .github/workflows/release-checklist.yml
   - Trigger: manual workflow dispatch with version input
   - Checks: semantic version format, CHANGELOG entry, optional test run

Tag release example:

1. git tag v0.1.0
2. git push origin v0.1.0

Automated release tag command (safe checks included):

- PowerShell: ./scripts/create-release-tag.ps1 -Version v0.1.0
   - Requires: clean git working tree, optional tests pass (unless -SkipTests is used)

Branch protection setup (requires GITHUB_TOKEN with admin rights):

- PowerShell: ./scripts/apply-branch-protection.ps1 -Owner NoName7n11 -Repo Self-System
   - Requires: GITHUB_TOKEN environment variable set with repo administration scope

## Key Paths

- cmd/server/main.go: app entrypoint and dependency wiring
- internal/http: API handlers and route registration
- internal/service: business logic orchestration
- internal/repository/sqlite: persistence adapters and migrations
- internal/config: runtime configuration loading
- api/openapi.yaml: API contract
- Plans/Progress/Phase_1_Timeline.md: implementation timeline

## Near-Term Next Focus

- Add frontend scaffolding (Wails + React) when Phase 2 UI work begins
- Add E2E coverage once frontend routes are available
