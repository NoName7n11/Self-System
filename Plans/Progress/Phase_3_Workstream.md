# Phase 3 Workstream

Date: 2026-05-10
Status: Planned, execution starting now
Scope: GBUS behavioral model, Android client + sync refinements, security and encryption, external task integrations, and Phase 3 UI polish.

## Phase 3 Objective

Phase 3 delivers the GBUS behavioral model (custom ML interest profiling) and expands multi-device capability with an Android client, hardened security, and external task integrations, while polishing the UI.

## Guiding Constraints

- Keep local-first behavior for responsiveness and offline tolerance.
- Preserve loose coupling between domain logic and infrastructure adapters.
- Ship security and integrations as opt-in features to minimize blast radius.
- Maintain configuration precedence (config defaults, .env overrides, env overrides).
- Maintain the test pyramid while adding ML and mobile quality gates.
- Prefer safety and correctness over feature volume.

## Important additions to teh project

- Added an ADR decision log under Plans/ADR with a README and shared template.
- Drafted initial ADRs 0001-0009 to capture high-impact decisions; 0001-0007 are retrospective.
- ADR 0008 supersedes ADR 0005, and ADR 0009 establishes the ADR process.

## Workstream 1 - GBUS Signal Instrumentation and Feature Store

Objective:
Capture behavioral signals and aggregate features needed for model training and inference.

Key tasks:
- Define the signal taxonomy (interactions, user actions, temporal patterns, cross-category, meta signals).
- Add event schema and emission points in services and UI interactions.
- Store raw signals and compute daily/weekly aggregates in the central store.
- Add retention, backfill, and privacy redaction controls.

Deliverables:
- Signal schema and documentation.
- Raw signal storage and aggregate feature tables.
- Signal emitters and aggregation jobs.

Done criteria:
- Core signals are captured and aggregated and can be queried for training data.

## Workstream 2 - GBUS Training Dataset and Baseline Model

Objective:
Create a reproducible training pipeline and baseline GBUS model.

Key tasks:
- Define labeling strategy (manual corrections, interaction strength, recency).
- Build dataset extraction and feature assembly pipeline.
- Train a baseline model (start with gradient boosting) and evaluate.
- Document metrics, thresholds, and comparison against weighted scoring.

Deliverables:
- Dataset builder and training pipeline.
- Baseline model artifact and evaluation report.

Done criteria:
- Baseline model is reproducible and beats the weighted scoring baseline on offline metrics.

## Workstream 3 - GBUS Inference and Product Integration

Objective:
Serve GBUS scores in runtime and integrate them into product behavior.

Key tasks:
- Decide inference runtime (Go-embedded model or Python service).
- Implement feature computation for inference (real-time or cached).
- Integrate GBUS scores into classification suggestions, search ranking, and reminder priority.
- Add safe fallback to weighted scoring and feature flags.

Deliverables:
- Inference module/service and runtime wiring.
- Integration hooks in classification/search/reminder flows.

Done criteria:
- GBUS scores influence product flows with safe fallback behavior when the model is unavailable.

## Workstream 4 - GBUS Monitoring and Governance

Objective:
Make GBUS reliable, observable, and reversible.

Key tasks:
- Add model versioning and metadata registry.
- Track drift and performance metrics over time.
- Log model decisions for audit and debugging.
- Define retraining cadence and rollback procedure.

Deliverables:
- Model registry metadata and monitoring metrics.
- Retraining and rollback runbook.

Done criteria:
- Model versions can be rolled back and performance metrics are visible.

## Workstream 5 - Android Client Foundation

Objective:
Establish the Android app with core data and auth flows.

Key tasks:
- Decide Android stack (Kotlin + Compose preferred) and scaffold the app.
- Implement auth, API client, and local cache.
- Build basic navigation and resource list view.
- Add build and run documentation.

Deliverables:
- Android app skeleton and build pipeline.
- Minimal feature parity for listing and viewing resources.

Done criteria:
- Android app can authenticate, sync, and display resources using local cache.

## Workstream 6 - Android Sync and Offline Refinements

Objective:
Harden mobile sync behavior for latency and offline use.

Key tasks:
- Implement websocket sync with reconnect behavior.
- Persist offline queue and replay on reconnect.
- Add background sync scheduling and battery-aware constraints.
- Align conflict handling and replay rules with server behavior.

Deliverables:
- Stable mobile sync module.
- Offline replay coverage and mobile integration tests.

Done criteria:
- Offline actions replay reliably and conflicts resolve predictably on mobile.

## Workstream 7 - Security and Encryption

Objective:
Deliver optional security protections and encryption.

Key tasks:
- Finalize threat model and key management strategy.
- Add app lock (PIN/biometric).
- Add encryption at rest for local databases.
- Add optional end-to-end encryption for sync payloads.
- Harden token storage for desktop and mobile clients.

Deliverables:
- Security modules and configuration options.
- Security documentation and recovery procedures.

Done criteria:
- Security features are functional and opt-in, with encrypted storage validated.

## Workstream 8 - External Task Integrations

Objective:
Integrate Self Systems tasks with external providers.

Key tasks:
- Select the first provider (Google Tasks or Todoist).
- Implement OAuth and token storage.
- Map internal tasks to external task schemas.
- Build background sync with conflict rules and error handling.

Deliverables:
- External task integration module.
- Settings UI for opt-in sync.

Done criteria:
- First provider sync works with opt-in toggle and robust error handling.

## Workstream 9 - UI Polish and Notifications

Objective:
Deliver Phase 3 polish and accessibility improvements.

Key tasks:
- Add animations and transitions.
- Implement notification system (desktop and mobile as applicable).
- Apply responsive refinements across layouts.
- Run accessibility audit and fix issues.

Deliverables:
- UI polish updates and accessibility report.

Done criteria:
- UI polish is complete and accessibility checks pass.

## Workstream 10 - Testing and Quality Gates

Objective:
Expand tests to cover ML, mobile, security, and integrations.

Key tasks:
- Add ML data pipeline and evaluation tests.
- Add Android unit and integration tests.
- Add security regression tests for encryption and auth.
- Update CI gates for new Phase 3 coverage.

Deliverables:
- New test suites and CI updates.

Done criteria:
- CI passes with Phase 3 gates and new test coverage.

## Planned Phase 3 Milestones

- Milestone 3A: GBUS data foundation (signals + feature store + baseline model).
- Milestone 3B: GBUS inference integration and monitoring.
- Milestone 3C: Android client alpha (sync + offline basics).
- Milestone 3D: Security, external task integrations, and UI polish.
- Milestone 3E: Phase 3 quality gates and CI stabilization.

## Phase 3 Definition of Done

- GBUS signals are captured, feature store is live, and baseline model is validated.
- GBUS inference is integrated into classification, search, and reminders with fallback.
- Android client sync is stable with offline replay and conflict handling.
- Security features (app lock, encryption at rest, optional E2E) are delivered.
- First external task integration is live with opt-in controls.
- UI polish and accessibility audits are complete.
- Phase 3 tests and CI gates pass consistently.
