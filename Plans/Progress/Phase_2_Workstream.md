# Phase 2 Workstream

Date: 2026-04-11
Status: Planned, execution starting now
Scope: Multi-device sync readiness, authentication, deep processing activation, and interactive UI expansion on top of the completed Phase 1 backend.

## Phase 2 Objective

Phase 2 shifts Self Systems from a local-only single-device architecture to a sync-capable platform with real-time updates, server-backed identity, and richer processing depth.

## Guiding Constraints

- Keep local-first behavior for responsiveness and offline tolerance.
- Preserve loose coupling between domain logic and infrastructure adapters.
- Roll out distributed features incrementally with explicit failure handling.
- Prioritize correctness for sync, conflict handling, and queue replay over feature volume.
- Maintain the test pyramid while adding distributed system coverage.

## Workstream 1 - Sync Server Foundation

Objective:
Introduce a dedicated sync service hosted on VPS infrastructure for multi-device state coordination.

Key tasks:
- Define server process boundaries (sync endpoints, websocket broker, health endpoints).
- Add deployment-ready service configuration for VPS runtime.
- Establish environment-specific config for local vs server runtime.

Deliverables:
- Running sync service process with health checks.
- Deployment-ready configuration and startup scripts.
- Architecture notes for sync runtime topology.

Done criteria:
- Sync service starts reliably in VPS-like environment.
- Health and readiness checks report green.

## Workstream 2 - Central Data Layer for Sync

Objective:
Enable a central source of truth for shared cross-device state.

Key tasks:
- Introduce PostgreSQL-backed repository adapters for central storage paths.
- Define migration strategy for central schema evolution.
- Maintain compatible domain contracts across SQLite and PostgreSQL adapters.

Deliverables:
- PostgreSQL schema and migrations.
- Repository implementations for central persistence.
- Adapter switching strategy documented.

Done criteria:
- Core entities (resources, categories, reminders, todos, sync metadata) persist and query from central store.

## Workstream 3 - Realtime Transport and Event Protocol

Objective:
Provide realtime synchronization and processing state propagation via WebSockets.

Key tasks:
- Define websocket event types and payload schemas.
- Implement connection lifecycle handling (connect, reconnect, heartbeat).
- Emit state change events for resource and processing updates.

Deliverables:
- Websocket transport module.
- Event schema contract document.
- Server-side event broadcast and client subscription flows.

Done criteria:
- Resource mutations on one client are visible on another in near realtime.

## Workstream 4 - Conflict Resolution and Offline Queue Replay

Objective:
Provide deterministic behavior for offline edits and concurrent writes.

Key tasks:
- Implement last-write-wins policy for conflicting updates.
- Persist offline queue locally in SQLite.
- Replay queued operations FIFO after reconnect.
- Record conflict history for recovery and troubleshooting.

Deliverables:
- Conflict resolver component.
- Offline queue persistence and replay logic.
- Conflict history model and query path.

Done criteria:
- Offline actions replay correctly after reconnect.
- Conflicts resolve predictably and are inspectable.

## Workstream 5 - Authentication and Session Management

Objective:
Add identity and secure session handling for server-backed sync.

Key tasks:
- Integrate Google OAuth flow.
- Issue and validate JWT session tokens.
- Protect sync endpoints and websocket connections with authentication checks.

Deliverables:
- OAuth integration module.
- JWT session issuance and middleware.
- Auth guards on server sync interfaces.

Done criteria:
- Unauthenticated sync requests are rejected.
- Authenticated users can establish and maintain sync sessions.

## Workstream 6 - Deep Processing Activation

Objective:
Enable Tier 2 deep analysis pipeline on top of existing skim-first workflow.

Key tasks:
- Activate deep queue processing and worker path.
- Add richer extraction and classification enrichment updates.
- Apply model selection strategy for skim vs deep workloads.

Deliverables:
- Deep queue processing flow from enqueue to completion.
- Resource enrichment updates reflected in data model.
- Cost-aware model routing rules.

Done criteria:
- Newly ingested resources transition from skim state to deep-enriched state asynchronously.

## Workstream 7 - API and Contract Expansion

Objective:
Extend contracts for sync and distributed operations while preserving existing behavior.

Key tasks:
- Add sync and auth-related API contracts.
- Define websocket contract references in docs.
- Standardize distributed error responses and retry semantics.

Deliverables:
- Updated OpenAPI and protocol docs.
- Backward-compatible endpoint evolution plan.
- Contract tests for new distributed paths.

Done criteria:
- API consumers can implement sync/auth flows using documented contracts.

## Workstream 8 - Interactive UI Expansion (Phase 2 UI Scope)

Objective:
Move from baseline views to high-interaction user flows.

Key tasks:
- Implement resource add/edit interactions.
- Build graph controls panel.
- Add advanced filtering and search interactions.
- Add basic chat interface layout integration.

Deliverables:
- Interactive forms and controls.
- Graph controls with filter integration.
- UI state updates via dedicated stores/hooks.

Done criteria:
- Users can add/edit/manage resources and filters through complete UI flows.

## Workstream 9 - Infrastructure and Deployment Hardening

Objective:
Prepare repeatable, stable server deployment workflow for Phase 2 runtime.

Key tasks:
- Define VPS Docker Compose stack for sync runtime.
- Add reverse proxy and TLS termination strategy.
- Add production configuration and startup checks.

Deliverables:
- VPS deployment recipe and compose assets.
- TLS/ingress notes and runbook.
- Ops checklist for environment setup.

Done criteria:
- Server stack deploys reproducibly and passes health checks in target environment.

## Workstream 10 - Testing and Quality Gates for Distributed Behavior

Objective:
Expand validation strategy to cover sync and networked failure modes.

Key tasks:
- Add integration tests for sync, conflicts, and replay.
- Add websocket behavior tests (ordering, reconnect, delivery).
- Add regression cases for auth and unauthorized access.

Deliverables:
- New integration test suites for distributed flows.
- CI checks including new sync-related test paths.
- Test data strategy for deterministic conflict cases.

Done criteria:
- Distributed behavior tests pass consistently in CI.

## Workstream 11 - Observability and Operational Controls

Objective:
Make distributed runtime debuggable and monitorable.

Key tasks:
- Add structured logging for sync operations and conflict outcomes.
- Add queue depth and replay metrics.
- Add operational troubleshooting guide.

Deliverables:
- Log conventions for distributed events.
- Basic metrics and operational health signals.
- Troubleshooting runbook section.

Done criteria:
- Operators can identify sync/auth/replay failures quickly from logs and metrics.

## Workstream 12 - Cost and Throughput Controls

Objective:
Control AI and processing cost while increasing processing depth.

Key tasks:
- Introduce model routing by workload complexity.
- Add caching and batching improvements where safe.
- Add queue throttling strategy for deep tasks.

Deliverables:
- Cost control configuration and defaults.
- Throughput policies for deep pipeline.
- Cost-impact validation notes.

Done criteria:
- Deep processing remains stable under load without uncontrolled API spending.

## Planned Phase 2 Milestones

- Milestone 2A: Auth + Sync Backbone
  - OAuth/JWT + websocket sync skeleton + minimal cross-device update loop.
- Milestone 2B: Consistency and Offline Reliability
  - Conflict handling + offline queue persistence and replay.
- Milestone 2C: Deep Processing + Interactive UX
  - Deep tier activation + phase-2 UI interaction scope.
- Milestone 2D: Production Hardening
  - VPS deployment, observability, CI quality gates for distributed behavior.

## Phase 2 Definition of Done

- Two desktop clients can sync resource updates in near realtime.
- Offline actions replay correctly after reconnect.
- Conflict handling is deterministic and recoverable.
- Sync operations require authenticated sessions.
- Deep processing upgrades skimmed resources asynchronously.
- CI includes passing distributed behavior tests.
- VPS deployment runbook and runtime checks are validated.
