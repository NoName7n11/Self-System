# Self Systems - Project Workflow Guide

Purpose: Provide a single, practical reference for how we work on this project so any new developer or AI agent can follow the same workflow.
Scope: Consolidates how to build, test, design, and track progress based on Plans/*.md.

## 1. Source Of Truth

Always align work with these documents (in this order):

- Plans/Outline.md (project vision, phases, decisions)
- Plans/Technical_Stack.md (architecture and stack decisions)
- Plans/Development_Workflow.md (branching, releases, CI, and testing flow)
- Plans/UI_Design_Guide.md (UI/UX decisions and dark-mode direction)
- Plans/reminder.md (modularity and deferred features)
- Plans/Future_Considerations.md (explicitly deferred items)
- Plans/Progress/Phase_3_Timeline.md and Plans/Progress/Phase_3_Completion_Checklist.md (progress tracking and evidence)

If any change conflicts with these plans, clarify with the project owner before proceeding.

## 2. Project Overview (Phase 2 Context)

- Phase 1 is local-first: Wails desktop app with React UI and Go backend in one binary via IPC.
- Phase 2 adds multi-device sync via REST + WebSockets; auth becomes mandatory for sync paths.
- Data stores: SQLite (core relational), sqlite-vec (vectors), Redis (Asynq queues).
- Processing: two-tier async (skim pass, deep pass), cloud AI (OpenAI + Anthropic).

## 3. Architecture & Code Structure

Go backend (interface-first, loosely coupled):

- internal/domain: business contracts and interfaces
- internal/service: orchestrates use cases
- internal/repository/*: storage adapters (SQLite/Postgres)

Frontend (React + TypeScript + Zustand):

- Feature logic lives in dedicated Zustand stores and hooks
- UI components remain modular and removable

Key principle: keep features removable with minimal blast radius.

## 4. Development Workflow

Branching (GitHub Flow):

- main is always deployable
- feature/*, bugfix/*, hotfix/*, docs/*, ci/*

Self-review PR workflow:

1. Create feature branch from main
2. Implement change + tests
3. Self-review using PR checklist
4. Run required tests locally
5. Merge after checks pass

Versioning: semantic versioning (MAJOR.MINOR.PATCH). Update CHANGELOG and tag releases.

## 5. Environment & Configuration

Dev container is the preferred setup (VS Code). Default commands:

- make dev-setup
- make dev
- docker compose up -d
- wails dev

Configuration precedence is fixed:

1. config.default.yml
2. .env
3. environment variables

## 6. Testing & Quality Gates

Testing pyramid: 70% unit / 20% integration / 10% E2E

Go tests:

- go test ./...
- go test ./test/integration

Frontend tests:

- npm test (Vitest)
- npx playwright test test/e2e

Lint/format:

- go fmt ./...
- golangci-lint run
- npm run lint

Build:

- wails build
- npm run build

## 7. UI/UX Rules

- Dark-mode-first UI direction
- shadcn/ui + Tailwind baseline
- Use Figma handoff for final visual decisions
- Graph design: functional polish, low-visual-noise, performance-aware

## 8. Modularity And Feature Flags

- Keep features isolated behind interfaces and feature flags
- Use config-driven feature toggles so features can be removed cleanly
- Avoid cross-feature coupling in stores or services

## 9. Progress Tracking (Mandatory)

Every work session must update the progress files.

1) Phase timeline:

- Update Plans/Progress/Phase_3_Timeline.md
- Add a new session heading per work session
- Format each step as:
  - Created <file> ``because <reason>``
  - Updated <file> ``because <reason>``
  - Ran <command> (date, pass/fail) ``because <reason>``

2) Completion checklist:

- Update Plans/Progress/Phase_3_Completion_Checklist.md
- Add evidence entries for new files or coverage
- Update the Validation Snapshot with each test run

If a change touches runbooks or operational guidance, update README.md and DEPLOYMENT.md.

## 10. When In Doubt

- Ask for clarification before diverging from Plans
- Keep API contracts in api/openapi.yaml up to date
- Add tests for new behavior and update progress files
- Avoid assumptions about UI visuals until Figma decisions exist
