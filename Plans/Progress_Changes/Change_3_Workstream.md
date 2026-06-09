# Change 3 Workstream - Event Sourcing Migration

Date: 2026-05-30
Status: WS1 COMPLETE — WS2 COMPLETE — WS3 COMPLETE — WS4 COMPLETE — WS5 COMPLETE — WS6 COMPLETE — WS7 COMPLETE
Scope: Incremental migration from state-based storage to event sourcing, starting with Resource and expanding to all other domains.

## Objective
Shift the source of truth from mutable state tables to an append-only event log with derived projections, without breaking existing read paths.

## Guiding Constraints

- Keep read performance by serving from projections.
- Keep migrations incremental and reversible.
- Preserve current API behavior and contracts during rollout.
- Maintain strict ordering and idempotency guarantees for events.
- Avoid double-writes that can drift; updates must be transactional.
- Phase 3 GBUS signal capture must land into the same event store, not a parallel one.

## Coordination with Phase 3 GBUS

The Phase 3 GBUS signal capture workstream and Change 3 are starting in the same window. Sequencing matters:

- Change 3 Milestone 3A (event log foundation) lands before Phase 3 Workstream 1 (GBUS signal instrumentation).
- GBUS signal events are written into the same `events` table with `aggregate_type = 'gbus_signal'` and `event_type` values drawn from the signal taxonomy in `Plans/Progress/Phase_3_GBUS_Signals_Feature_Store.md`.
- Aggregation jobs that populate `gbus_feature_*` tables consume the unified `events` table as their input.
- Indexes on `events` must support both per-aggregate replay (for entity sync) and per-event-type range scans (for GBUS aggregation) without separate physical stores.

Decision: unified event store, no parallel `gbus_signal_events` table. The Feature Store spec's logical table name remains for documentation, but it resolves to a view over `events`.

## Schema Specification

The `events` table is the load-bearing artifact for the entire migration. Settle the schema before any other workstream begins.

Columns:
- `sequence` INTEGER/BIGSERIAL PRIMARY KEY — global ordering for sync and outbox readers
- `event_id` TEXT/UUID NOT NULL UNIQUE — client-supplied for idempotency
- `aggregate_id` TEXT NOT NULL — entity identity (resource_id, category_id, etc.)
- `aggregate_type` TEXT NOT NULL — `resource`, `category`, `todo`, `reminder`, `gbus_signal`
- `event_type` TEXT NOT NULL — e.g., `ResourceCreated`, `ResourceSummaryUpdated`, `gbus.resource_opened`
- `event_version` INT NOT NULL — per-aggregate monotonic version for optimistic concurrency
- `payload` TEXT/JSONB NOT NULL — event-type-specific body
- `payload_schema_version` INT NOT NULL DEFAULT 1 — schema version for the payload shape
- `occurred_at` TIMESTAMPTZ NOT NULL — when the change happened on the originating device
- `recorded_at` TIMESTAMPTZ NOT NULL DEFAULT now() — when the server persisted the event
- `device_id` TEXT — originating device for sync attribution
- `actor_id` TEXT — user/subject who caused the event
- `redacted` BOOLEAN NOT NULL DEFAULT false — set true after a redaction event
- `correlation_id` UUID — links related events across aggregates (e.g., import batch)

Indexes:
- UNIQUE (`aggregate_id`, `event_version`) — enforces optimistic concurrency
- UNIQUE (`event_id`) — enforces idempotency
- (`sequence`) — primary sync/outbox read path (primary key)
- (`aggregate_type`, `event_type`, `recorded_at`) — GBUS aggregation read path

Payload notes:
- SQLite stores `payload` as TEXT with JSON1 functions. Do not run hot-path JSON filters in SQLite; materialize fields into projections or aggregate tables.
- Postgres stores `payload` as JSONB for expression indexing when needed.
- Add a payload-validity CHECK (`json_valid(payload)` in SQLite; JSONB parse in Postgres).

Constraints:
- Append-only at the application layer; revoke UPDATE/DELETE grants except for the redaction code path.
- `payload` for redacted events is `{"redacted": true}`; envelope (timestamps, IDs, type) is retained for audit.

Projection snapshots table (created in Workstream 1, worker deferred to Workstream 6):

```
CREATE TABLE projection_snapshots (
	aggregate_id      TEXT NOT NULL,
	aggregate_type    TEXT NOT NULL,
	snapshot_version  INT NOT NULL,
	payload           TEXT/JSONB NOT NULL,
	created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (aggregate_id, snapshot_version)
);
CREATE INDEX ON projection_snapshots (aggregate_type, snapshot_version);
```

## Pattern Decisions

Document each as a dedicated ADR before Workstream 1 starts (ADR 0010 through 0017).

### P1 - Optimistic Concurrency Control
- Per-aggregate `event_version` is monotonic. Writers read current max version, compute next, attempt INSERT with UNIQUE constraint, retry on conflict.
- Lost-update protection lives entirely in the constraint; no advisory locks.

### P2 - Idempotency
- `event_id` UUID supplied by the writer. INSERT uses `ON CONFLICT (event_id) DO NOTHING`.
- Sync replay sends the same `event_id`; duplicates are silently ignored.

### P3 - Event Versioning
- Payload shape is versioned per event type via `payload_schema_version`.
- New shape ships as either `event_type = "ResourceSummaryUpdated"` with `payload_schema_version = 2`, OR a renamed event type when the semantics change.
- Read-time upcasters convert older payload versions to the current shape before applying to the projection. Upcasters are pure functions, unit tested per version pair.

### P4 - Outbox Pattern for Sync Emission
- The `events` table is the outbox. A dedicated outbox worker tails by `sequence`, publishes to the sync hub, and tracks last-published sequence per subscriber.
- Append and emit are decoupled: the writer commits the event transactionally; the outbox worker delivers asynchronously with at-least-once semantics.
- Subscribers dedupe on `event_id` to achieve effective exactly-once.

### P5 - Snapshot and Compaction
- Per-aggregate snapshot every 100 events or every 30 days, whichever comes first.
- Snapshots are stored in a `projection_snapshots` table keyed by `(aggregate_id, snapshot_version)`.
- Rebuild path: load latest snapshot, apply events with `event_version > snapshot_version`.
- Raw events retained 365 days, then archived to cold storage. Snapshots retained indefinitely.

### P6 - Redaction
- Deletion of user data is itself an event: `EventRedacted` records the target `event_id` and reason.
- The redaction worker rewrites the target row's `payload` to `{"redacted": true}` and sets `redacted = true`. The envelope (timestamps, IDs, event_type) is preserved so audit and sequence integrity are not broken.
- Projections are rebuilt for affected aggregates after redaction.

### P7 - Dual-Write Transition
- During rollout, the feature flag controls the WRITE path only. When the flag is ON, writes go through the event store and project to the existing state table within the same transaction.
- When the flag is OFF, writes go directly to the state table (current behavior).
- The state table is always the authoritative read source until Workstream 4 completes. Reads never branch on the flag.
- No backfill or projection rebuild runs while the flag is OFF for an aggregate type.

### P8 - Event Handlers / Projectors
- Each event type has zero or more projectors registered via a `ProjectorRegistry`.
- Projectors are idempotent functions `(event, txn) -> error` invoked inside the append transaction for synchronous projections (current resource state) and outside for asynchronous projections (sync outbox, deep processing enqueue, GBUS aggregation).
- Synchronous vs asynchronous classification is declared per projector at registration time.

## Workstream 1 - Event Log Foundation — COMPLETE (Sessions 6-8)

Objective:
Introduce the core event store schema and write/read contracts.

Key tasks:
- [x] Apply the Schema Specification as SQL migrations for SQLite and PostgreSQL.
- [x] Implement the `EventStore` interface (`Append`, `ReadByAggregate`, `ReadBySequence`, `Snapshot`, `Redact`).
- [x] Add `WithTx` and `TxStore` interface for P8 synchronous projectors.
- [x] UUID validation in `normalizeEvent`; server-set `RecordedAt`.
- [x] Implement optimistic concurrency conflict detection (OCC constraint + `ErrConcurrencyConflict`).
- [x] Write the pattern ADRs (P1-P8) and link them from the workstream.
- [ ] `ProjectorRegistry` and synchronous projector invocation — deferred to WS2 (domain-specific).

Deliverables:
- [x] `internal/repository/sqlite/migration.go` — events + projection_snapshots tables.
- [x] `internal/repository/postgres/migrations/0002_events.sql` — Postgres schema with correct JSONB CHECK.
- [x] `internal/eventstore/store.go` — shared types, `Store`, `TxStore` interfaces.
- [x] `internal/eventstore/sqlite_store.go` — SQLiteStore + SQLiteTxStore.
- [x] `internal/eventstore/postgres_store.go` — PostgresStore + PostgresTxStore.
- [x] `internal/eventstore/sqlite_store_test.go` — 13 unit tests.
- [x] `internal/eventstore/postgres_store_test.go` — 8 parity tests (skip without DSN).
- [x] ADRs 0010 through 0017.

Done criteria:
- [x] Events can be appended and read by aggregate and by sequence.
- [x] Optimistic concurrency conflicts surface as `ErrConcurrencyConflict`.
- [x] Idempotent appends return `Applied=false` for duplicates, same sequence.
- [x] `WithTx` commit and rollback verified in both SQLite and Postgres tests.
- [x] `go test ./...` passes (13 SQLite PASS, 8 Postgres SKIP without DSN).

## Workstream 2 - Resource Event-Sourced Write Path — COMPLETE (Session 10)

Objective:
Make Resource mutations event-first with projection updates.

Key tasks:
- [x] Define Resource event types: `ResourceCreated`, `ResourceUpdated`, `ResourceDeleted`, `ResourceSummaryUpdated`, `ResourceCategoryAssigned`, `ResourceArchived`, `ResourceUnarchived`.
- [x] Implement `ProjectorRegistry` and register a synchronous projector that maintains the `resources` projection table (same shape as today) inside `WithTx`.
- [x] Add `TxConn` interface + `Conn() TxConn` to `TxStore` so sync projectors can write to projection tables.
- [x] SQLite and Postgres sync projectors for Create/Update/Delete/CategoryAssigned.
- [x] OCC retry loop (maxRetries=3) with fresh event_id per retry (P1+P2).
- [x] `latestResourceVersion` helper — reads aggregate event history to determine next version.
- [x] Wire the `events.resource.enabled` feature flag in config; default OFF per P7.
- [x] Implement the dual-write transition per P7 in all four ResourceService mutation methods.
- [x] `buildRepositories` in main.go now returns `eventstore.Store`; server wires flag+registry when enabled.
- [ ] Async projectors for sync outbox, deep processing, GBUS signal — deferred to WS4/WS5 (handler still publishes sync events directly).

Deliverables:
- [x] `internal/eventstore/projector.go` — `TxConn`, `SyncProjector`, `AsyncProjector`, `ProjectorRegistry`.
- [x] `internal/eventstore/resource_events.go` — event type constants + payload structs v1.
- [x] `internal/eventstore/resource_projector.go` — SQLite + Postgres sync projectors; `RegisterResourceProjectors(registry, dialect)`.
- [x] `internal/eventstore/store.go` — `TxConn` interface, `Conn() TxConn` on `TxStore`.
- [x] `internal/eventstore/sqlite_store.go` + `postgres_store.go` — `Conn()` implemented.
- [x] `internal/service/resource_service.go` — dual-write with `WithEventSourcing` option; `appendWithTx` with OCC retry.
- [x] `internal/service/resource_service_eventsource_test.go` — 7 integration tests.
- [x] `internal/config/config.go` — `EventsResourceEnabled bool`.
- [x] `config/config.default.yml` — `events_resource_enabled: false`.
- [x] `cmd/server/main.go` — event store returned from `buildRepositories`; wired when flag ON.

Done criteria:
- [x] Resource mutations write events and update the projection in a single transaction when the flag is ON.
- [x] All existing Resource handler tests pass with the flag both ON and OFF.
- [x] Concurrent updates to the same resource exercise OCC retry behavior (TestResourceServiceOCCRetryOnConcurrentUpdate).
- [x] `go test ./...` passes — all packages green.

Deliverables:
- Resource event definitions, payload schemas, and upcasters (initial v1 only).
- Transactional append + synchronous projection path.
- Async projector registrations for sync, deep processing, GBUS.
- Feature flag wiring with config validation.

Done criteria:
- Resource mutations write events and update the projection in a single transaction when the flag is ON.
- All existing Resource handler tests pass with the flag both ON and OFF.
- Concurrent updates to the same resource exercise OCC retry behavior.

## Workstream 3 - Resource Backfill and Verification — COMPLETE (Session 10)

Objective:
Seed the event log for existing Resource rows and validate parity.

Key tasks:
- [x] Generate synthetic `ResourceImported` events for existing rows, one per resource, with a shared `correlation_id` per batch.
- [x] Implement a parity validation tool that rebuilds the projection from events and diffs against the live projection.
- [x] Define and enforce a backfill performance budget: 100K resources backfill in under 30 minutes (`BenchmarkBackfill100K`, skip in short mode).
- [ ] Repair tooling for divergences with operator confirmation — deferred; parity report identifies divergences for manual remediation.

Deliverables:
- [x] `internal/migration/backfill.go` — `RunResourceBackfill(ctx, db, store, cfg)` with batch-TX, idempotency, progress callback.
- [x] `internal/migration/parity.go` — `CheckResourceParity(ctx, db, store)`, `ParityReport`, `FormatReport`.
- [x] `internal/migration/backfill_test.go` — 7 tests + `BenchmarkBackfill100K` (skips in short mode).
- [x] `internal/migration/parity_test.go` — 9 tests covering clean, divergence, extra-in-projection, extra-in-events, deleted.
- [x] `cmd/tools/main.go` — `tools backfill` and `tools parity` subcommands with flag parsing.
- [x] `internal/eventstore/resource_events.go` — `EventTypeResourceImported` constant.
- [x] `internal/eventstore/resource_projector.go` — `ResourceImported` registered with same projector as `ResourceCreated`.

Done criteria:
- [x] Rebuilt projection matches live projection after backfill (TestParityCleanAfterBackfill).
- [x] Divergences are detected and reported field-by-field (TestParityDetectsDivergence).
- [x] Backfill is idempotent: second run skips already-evented resources (TestBackfillIdempotent).
- [x] Performance budget enforced by BenchmarkBackfill100K (30-minute ceiling; run with -bench, skip in -short).
- [x] `go test ./...` passes — all packages green.

## Workstream 4 - Sync Read Path Switch — COMPLETE (Session 10)

Objective:
Move sync reads from state-based changes to event sequence and migrate in-flight sync state.

Key tasks:
- [x] Outbox worker tails events table by sequence and publishes to sync hub (P4).
- [x] Outbox worker aligns hub sequence numbers with eventstore sequences, enabling durable `since_sequence` reconnect.
- [x] WS handler: when events table is configured, query it for since_sequence replay before hub history; merge and deduplicate with hub history for non-event-sourced types.
- [x] HTTP handler: skip direct `publishSyncEvent` for resources when `EventsEnabled()`; outbox worker delivers within one poll interval.
- [x] In-flight queue: `OfflineReplayManager.Replay()` calls ResourceService methods → when flag=ON creates events → outbox delivers. No data loss.
- [x] Deprecation notice added to `replay_store_sqlite.go`; tables scheduled for removal after WS5 + one release cycle.
- [ ] Chat command path still publishes resource sync events directly — deferred to WS5 cleanup.

Deliverables:
- [x] `internal/sync/outbox_worker.go` — `OutboxWorker` with catch-up loop, steady-state poll (250ms default), `outboxTranslate`, `LastSequence()`, `Published()`.
- [x] `internal/sync/outbox_worker_test.go` — 9 tests: Created, Imported, Updated, Deleted, skip-unknown, sequence alignment, LastSequence, new-event pickup, translate mapping.
- [x] `internal/sync/hub.go` — `sortEventsBySequence` helper for merged replay.
- [x] `internal/sync/protocol.go` — `EventSourceOutboxWorker` constant.
- [x] `internal/sync/routes.go` — `WithEventStoreReplay(store)` route option; event store wired to WSHandler.
- [x] `internal/sync/ws_handler.go` — `SetEventStore`, durable replay from events table merged with hub history.
- [x] `internal/service/resource_service.go` — `EventsEnabled() bool`.
- [x] `internal/http/handler.go` — skip direct publishSyncEvent for Create/Update/Delete/UpdateCategory when `EventsEnabled()`.
- [x] `cmd/server/main.go` — outbox worker started in goroutine; `WithEventStoreReplay` wired.
- [x] `internal/sync/replay_store_sqlite.go` — deprecation comment added.

Done criteria:
- [x] Sync can replay Resource changes using the event log alone (events-table replay path in WSHandler).
- [x] No data loss during in-flight queue migration (OfflineReplayManager flow unchanged; events appended when flag=ON).
- [x] All existing sync + HTTP tests pass — `go test ./...` green.
- [x] Outbox worker correctly skips unknown event types (GBUS signals, etc.) and only maps resource event types to sync event types.

## Workstream 5 - Progressive Domain Expansion — COMPLETE (Session 11)

Objective:
Roll the pattern out to other domains after Resource is stable.

Key tasks:
- [x] Category event sourcing: Created, Updated, Deleted + SQLite/Postgres projectors + service dual-write.
- [x] Todo event sourcing: Created, Updated, Deleted + SQLite/Postgres projectors + service dual-write.
- [x] Reminder event sourcing: Created, Updated, Deleted + SQLite/Postgres projectors + service dual-write.
- [x] GBUS signal aggregate type and event type constants defined (aggregation jobs in Phase 3 GBUS workstream).
- [x] Handler decoupling for all 9 new domain mutation paths.
- [x] Outbox worker updated to translate Category/Todo/Reminder events to sync event types.
- [x] Shared `appendWithTx`/`aggregateLatestVersion` extracted to `eventsource.go` (no duplication).
- [x] WAL mode enabled for SQLite (PRAGMA journal_mode=WAL) for concurrent read/write robustness.
- [ ] GBUS signal projectors (aggregation tables) — deferred to Phase 3 GBUS workstream.

Deliverables:
- [x] `internal/eventstore/category_events.go` + `category_projector.go`.
- [x] `internal/eventstore/todo_events.go` + `todo_projector.go`.
- [x] `internal/eventstore/reminder_events.go` + `reminder_projector.go`.
- [x] `internal/eventstore/gbus_events.go` — aggregate type + signal event type constants.
- [x] `internal/service/eventsource.go` — shared `appendWithTx`, `aggregateLatestVersion`.
- [x] `internal/service/category_service.go` — `WithCategoryEventSourcing`, `EventsEnabled`, dual-write + `EnsureByName`.
- [x] `internal/service/todo_service.go` — `WithTodoEventSourcing`, `EventsEnabled`, dual-write.
- [x] `internal/service/reminder_service.go` — `WithReminderEventSourcing`, `EventsEnabled`, dual-write.
- [x] `internal/service/domain_eventsource_test.go` — 14 tests across all 3 domains (Create/Update/Delete/FlagOff per domain, + EnsureByName for category).
- [x] Config: `EventsCategoryEnabled`, `EventsTodoEnabled`, `EventsReminderEnabled` (all false by default).
- [x] `cmd/server/main.go` — all three services wired with event-sourcing options.
- [x] `internal/http/handler.go` — 9 publishSyncEvent calls gated on `EventsEnabled()`.
- [x] `internal/sync/outbox_worker.go` — Category/Todo/Reminder translation added.
- [x] `internal/repository/sqlite/db.go` — WAL mode enabled.

Done criteria:
- [x] All core domains (Category, Todo, Reminder) event-sourced with stable projections.
- [x] 14 domain integration tests pass (Create/Update/Delete event+projection, flag-OFF compatibility).
- [x] GBUS signal events flow through the unified events table (aggregate_type='gbus_signal') — constants ready for Phase 3.
- [x] `go test ./...` passes — all packages green.

## Workstream 6 - Observability and Governance — COMPLETE (Session 13)

Objective:
Make event sourcing auditable and operable.

Key tasks:
- [x] Add metrics: events appended total, OCC retry rate, outbox lag (sequence delta), projector apply latency (avg ms), snapshot creation count, redaction count.
- [x] Implement automatic snapshot creation per P5 cadence (every 100 events or 30 days) with 30-second bounded batch execution time.
- [x] Document the rollback procedure: feature flag off, state table becomes authoritative, events table retained for audit.
- [x] Document the recovery procedure: detect projection drift, rebuild from snapshot + events, parity check to confirm.
- [x] Add `GET /api/v1/sync/events/health` endpoint exposing outbox lag and OCC retry rate, gated behind auth.
- [x] Add `LatestSequence(ctx) (int64, error)` to Store interface + SQLite and Postgres implementations.

Deliverables:
- [x] `internal/eventstore/observability.go` — `EventObservability` with atomic counters; `EventObservabilitySnapshot` struct; `RecordAppend`, `RecordOCCRetry`, `RecordProjectorLatency`, `RecordSnapshotCreated`, `RecordRedaction` (nil-safe).
- [x] `internal/eventstore/snapshot_worker.go` — `SnapshotWorker` with `Register(aggregateType, SnapshotFunc)`, P5 cadence (100 events / 30 days), bounded batch timeout, polling loop.
- [x] `internal/eventstore/store.go` — `LatestSequence` added to `Store` interface.
- [x] `internal/eventstore/sqlite_store.go` + `postgres_store.go` — `LatestSequence` implemented on both store types and their TxStore variants.
- [x] `internal/eventstore/projector.go` — `obs *EventObservability` field on `ProjectorRegistry`; `SetObservability`; latency measured in `ApplySync` per-invocation.
- [x] `internal/service/eventsource.go` — `appendWithTx` accepts `obs *EventObservability` (nil-safe); records append success and OCC retry per attempt.
- [x] `internal/service/resource_service.go` + `category_service.go` + `todo_service.go` + `reminder_service.go` — `eventObs` field + `WithXxxEventObservability` option added; all `appendWithTx` calls updated.
- [x] `internal/sync/routes.go` — `WithOutboxWorker`, `WithEventObservability` route options; `GET /api/v1/sync/events/health` endpoint; `EventsHealthSnapshot` response type.
- [x] `cmd/server/main.go` — `buildRepositories` returns `rawDB *sql.DB` (9th value); shared `EventObservability` created and wired into all 4 service registries + snapshot worker + sync routes; outbox worker reference captured and wired.
- [x] `DEPLOYMENT.md` — Section 8 (Event Sourcing Rollback) + Section 9 (Recovery / Projection Drift) added with step-by-step operator runbooks and health endpoint reference.
- [x] `internal/eventstore/observability_test.go` — 5 tests: nil-safe counters, counter accumulation, latency avg, no-latency-on-unmatched-type, latency-on-matched-type.
- [x] `internal/eventstore/snapshot_worker_test.go` — 5 tests: below-cadence no-op, at-cadence snapshot, idempotency, unregistered-type skip, aged-out-aggregate trigger.

Done criteria:
- [x] Event system health observable via `GET /api/v1/sync/events/health` (auth-gated).
- [x] Snapshot worker runs on schedule without blocking writes (bounded 30s batch).
- [x] Rollback and recovery procedures documented in DEPLOYMENT.md.
- [x] `go test ./...` passes — all packages green.

## Workstream 7 - Testing and Safety Gates — COMPLETE (Session 14)

Objective:
Protect correctness during migration.

Key tasks:
- [x] Property-based tests: event version monotonicity and projection determinism.
- [x] Sync sequence replay tests: hub ReplaySince at arbitrary offsets, events-table durable replay.
- [x] Rollback drill script: flag ON → flag OFF → parity verification.
- [x] Backfill benchmark wired into CI with fast smoke gate (10x) and full budget test (100K, manual).
- [x] CI workflow: event-sourcing-gates.yml runs all safety gates on push/PR.
- [x] Makefile targets: event-sourcing-test, rollback-drill, backfill-bench.

Note on `pgregory.net/rapid`: offline environment prevented installation. Property tests are
implemented using Go's built-in fuzz testing (`testing.F`) and `math/rand`, which validates
the same invariants. The fuzz seeds run deterministically in `go test`; continuous fuzzing
available via `go test -fuzz`. Files are structured for easy swap to rapid once available.

Deliverables:
- [x] `internal/eventstore/property_test.go` — fuzz tests + deterministic table-driven variants:
      `FuzzEventVersionMonotonicity`, `FuzzProjectionDeterminism`,
      `TestEventVersionMonotonicityVariants`, `TestProjectionDeterminismSeeds` (5 seeds each).
- [x] `internal/sync/reconnect_test.go` — 8 reconnect tests:
      ReplaySinceZero, Midpoint, Latest, BeyondLatest, WithLimit, SequenceContinuity,
      ExplicitSequencePreserved, EventsTableReplay; + `FuzzReconnectReplaySince`.
- [x] `scripts/rollback_drill/main.go` — standalone drill: Phase1 (ON), Phase2 (OFF), Phase3 (parity).
      Exit 0 = passed; validates that post-rollback Phase2 resources appear as extra-in-projection.
- [x] `.github/workflows/event-sourcing-gates.yml` — CI workflow: property tests, reconnect tests,
      rollback drill, backfill benchmark (10x), full test suite.
- [x] `Makefile` — `event-sourcing-test`, `rollback-drill`, `backfill-bench` targets.

Done criteria:
- [x] Tests pass for event append, projection rebuild, sync replay, projection determinism, and rollback.
- [x] Backfill benchmark wired into CI (fast smoke gate 10x; full 30-min budget available manually).
- [x] Rollback drill script runs end-to-end and exits 0.
- [x] `go test ./...` passes — all packages green.

## Planned Milestones

- Milestone 3A: Schema Specification, Pattern ADRs (P1-P8), event log foundation, Resource events and synchronous projection.
- Milestone 3B: Resource backfill, parity validation, performance benchmark.
- Milestone 3C: Sync read path switch via outbox, in-flight queue migration.
- Milestone 3D: Category, Todo, Reminder, and GBUS signal event sourcing rollout.
- Milestone 3E: Observability, snapshots, rollback drill, long-term maintenance.

## Definition of Done

- Events are the source of truth for all core domains, including GBUS signals.
- Projections are rebuildable from snapshots plus events and match the live projection at row-level checksum.
- Sync reads from event sequence through the outbox without drift.
- Optimistic concurrency, idempotency, redaction, and snapshot strategies are implemented and tested.
- Operational guidance and tests cover the event system, including a rehearsed rollback procedure.
- Phase 3 GBUS signal capture and aggregation consume the unified `events` table.