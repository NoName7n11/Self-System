# Self Systems

Self Systems is a local-first knowledge system backend that captures resources, classifies them, and exposes structured APIs for resources, categories, reminders, todos, graph data, and unified chat commands.

Current repository status: Phase 1 backend and operations closeout are in place (Go + Gin + SQLite, integration tests, CI, and release automation).

## Architecture Snapshot

- Runtime: Go 1.22 service with Gin HTTP routes
- Storage: SQLite (default) with PostgreSQL adapter for Phase 2 central-store runtime
- AI abstraction: heuristic, OpenAI, Anthropic, and Gemini providers behind a manager
- API contract: OpenAPI spec in api/openapi.yaml
- Local infrastructure: PostgreSQL, Redis, and DGraph Docker services for sync, queue, and graph topology
- VPS topology: Dockerized Go API behind NGINX reverse proxy via docker-compose.vps.yml

## Prerequisites

- Go 1.22+
- Docker Desktop (optional, for PostgreSQL, Redis, and DGraph)
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

## Frontend UI Preview

The Workstream 8 UI lives in `frontend/` and can run independently while the Go API is running.

1. Start backend API from repository root:
   - `go run ./cmd/server`
2. In a second terminal, start frontend:
   - `cd frontend`
   - `npm install`
   - `npm run dev`
3. Open the Vite URL shown in terminal (default: `http://127.0.0.1:5173`).

Optional environment overrides for frontend:

- `VITE_API_BASE_URL` (default `http://127.0.0.1:8080`)
- `VITE_SYNC_WS_URL` (full websocket URL override)
- `VITE_SYNC_WS_PATH` (path override, default `/api/v1/sync/ws`)

Current Workstream 8 UI sections:

- `Graph`: graph controls + force graph + resource list/edit/delete
- `Search`: filter-first resource listing + resource edit/delete panel
- `Chat`: command dock + resource context list
- `Tasks`: Todo and Reminder CRUD/status board
- `Settings`: runtime endpoint/sync/status snapshot

## Configuration Precedence

Configuration is loaded in this order:

1. config/config.default.yml
2. .env
3. Environment variables (prefix SS_)

Example override:

- SS_APP_PORT=9090
- SS_DATABASE_PATH=./data/local.db

## Common Commands

Windows note: if `make` is not found but MinGW is installed, use `mingw32-make` with the same targets.

| Task | Command |
|---|---|
| Setup dependencies | make dev-setup |
| Start API + local infra | make dev |
| Start only API | make run |
| Start Docker services | make docker-up |
| Stop Docker services | make docker-down |
| Start VPS runtime topology | make vps-up |
| Stop VPS runtime topology | make vps-down |
| Tail VPS runtime logs | make vps-logs |
| Run all tests | make test |
| Run integration tests | make integration-test |
| Run distributed sync/replay gate | make distributed-test |
| Generate distributed gate evidence report | make distributed-report |
| Verify deployed sync runtime reachability | make verify-sync-runtime SYNC_RUNTIME_BASE_URL=https://api.example.com |
| Format Go files | make lint |
| CI-equivalent local checks | make ci |
| CI-equivalent distributed checks | make ci-distributed |
| Start only PostgreSQL service | make docker-up-postgres |
| Run PostgreSQL central data integration gate | make test-postgres |
| Build server binary | make build |
| Clean Go cache outputs | make clean |

## Testing

- Full test suite: go test ./...
- Integration tests only: go test ./test/integration/...
- Frontend unit tests: cd frontend && npm test
- Frontend unit scope includes task-store + resource-store create/update/delete mutation error handling (including resource silent refresh retention paths) and API client envelope-contract assertions for resource list/create/update/delete plus task list/create/update/delete operations
 - Frontend unit scope includes store-level read-path resilience tests (resource/task stores) covering silent refresh retention vs non-silent list failures, task-store + resource-store create/update/delete mutation error handling (including resource silent refresh retention paths) and API client envelope-contract assertions for resource list/create/update/delete plus task list/create/update/delete operations
- Frontend CI unit gate also emits frontend/test-results/vitest-results.json via npx vitest JSON reporting and publishes pass/fail + failed-assertion summary details for faster contract-test triage
- Frontend E2E tests: cd frontend && npm run test:e2e
- Frontend E2E scope includes positive and negative Resource + Tasks flows (resource create/update/delete success, backend/network/timeout/malformed envelope failure UX assertions, and resource delete malformed/empty success-body tolerance assertions; plus todo/reminder create/update/delete/status success/failure, client-side validation, malformed API envelope handling, and network/timeout mutation failure assertions)
- Frontend E2E first-time browser setup: cd frontend && npx playwright install chromium
- Distributed sync/replay gate: go test ./internal/sync ./test/integration -run "Sync|Offline|Replay"
- Distributed gate evidence report: make distributed-report (writes artifacts/distributed-sync-go-test.json and artifacts/distributed-sync-report.md)
- PostgreSQL central data gate: set SS_POSTGRES_TEST_DSN then run go test ./internal/repository/postgres -run Integration
- Runtime reachability verification: make verify-sync-runtime SYNC_RUNTIME_BASE_URL=https://api.example.com (optional bearer token via SS_SYNC_RUNTIME_BEARER_TOKEN)
- Runtime report template: artifacts/templates/sync-runtime-reachability.sample.json

Testing layout follows:

- Unit tests adjacent to source packages
- Integration tests in test/integration

## CI and Releases

- CI workflow: .github/workflows/ci.yml
   - Trigger: push and pull_request to master and main
   - Checks: Go formatting, distributed behavior gate + evidence artifacts/summary, full Go tests, PostgreSQL integration gate, server build, plus frontend unit/E2E/build gate (Node + Playwright Chromium)
   - Artifacts: validates frontend Vitest JSON report generation/non-empty output, publishes frontend unit-test summary details (including failed assertions), conditionally validates Playwright HTML/JSON + trace artifacts when E2E executes, and uploads frontend test-results/playwright-report artifacts for failure diagnostics
- Sync runtime reachability workflow: .github/workflows/sync-runtime-reachability.yml
   - Trigger: manual workflow dispatch
   - Base URL source: workflow input base_url
   - Auth modes: workflow input auth_mode with options none, manual_input, github_oidc
   - Secure non-manual auth: github_oidc mode (optional workflow input oidc_audience)
   - Manual fallback auth: bearer_token input when auth_mode is manual_input
   - Checks: deployed /health + /api/v1/sync/health plus websocket handshake to /api/v1/sync/ws
   - Output: sync-runtime-reachability artifact with JSON verification report
- Sync runtime local smoke workflow: .github/workflows/sync-runtime-local-smoke.yml
   - Trigger: manual workflow dispatch
   - Use case: no deployed server available yet
   - Checks: boots local API in workflow job, waits for health, then runs reachability verification against localhost
   - Output: sync-runtime-local-smoke artifact with server logs and JSON verification report
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
   - Script now validates token access via GitHub API before applying rules
   - Script applies protection directly and skips only missing branches (HTTP 404)

## Key Paths

- cmd/server/main.go: app entrypoint and dependency wiring
- internal/http: API handlers and route registration
- internal/service: business logic orchestration
- internal/repository/sqlite: persistence adapters and migrations
- internal/config: runtime configuration loading
- api/openapi.yaml: API contract
- scripts/generate_distributed_gate_report/main.go: distributed gate evidence report generator
- scripts/verify_sync_runtime/main.go: deployed sync runtime reachability verifier
- DEPLOYMENT.md: deployment and runtime verification runbook
- artifacts/templates/sync-runtime-reachability.sample.json: sample JSON report template for reachability evidence
- docker-compose.vps.yml: final VPS topology overlay (Go API + NGINX reverse proxy)
- deploy/nginx/selfsystems.conf: websocket-aware proxy config for /api/v1/sync/ws
- Plans/Progress/Phase_1_Timeline.md: implementation timeline

## Near-Term Next Focus

- Add frontend scaffolding (Wails + React) when Phase 2 UI work begins
- Add E2E coverage once frontend routes are available
