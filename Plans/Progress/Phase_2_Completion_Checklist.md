# Phase 2 Completion Checklist

Date: 2026-04-11
Status: Complete
Completion Date: 2026-05-06
Scope: Phase 2 distributed architecture, sync, auth, and deep processing rollout.

## Exit Criteria

- [x] Sync server runtime is deployed and reachable.
- [x] Central data layer is operational for multi-device writes.
- [x] WebSocket sync events are stable for realtime updates.
- [x] Conflict handling and offline queue replay are implemented and validated.
- [x] OAuth and JWT-based auth gating is active for sync paths.
- [x] Deep processing tier is active and updates resources asynchronously.
- [x] API contracts are updated for sync/auth/event payloads.
- [x] Interactive Phase 2 UI flows are implemented (add/edit, controls, filters, chat layout).
- [x] Deployment runbook for VPS stack is documented and reproducible.
- [x] Distributed behavior tests pass in CI.
- [x] Observability signals exist for sync/auth/replay issues.
- [x] Cost controls for deep processing are configured.

## Evidence Map (To Fill During Execution)

- Sync service and runtime wiring: cmd/server/main.go, internal/sync/routes.go, internal/sync/hub.go, internal/sync/ws_handler.go
- Sync runtime reachability verification artifacts: scripts/verify_sync_runtime/main.go, .github/workflows/sync-runtime-reachability.yml, .github/workflows/sync-runtime-local-smoke.yml, Makefile (verify-sync-runtime target), artifacts/templates/sync-runtime-reachability.sample.json, DEPLOYMENT.md (runtime verification command path)
- Central store adapters and migrations: internal/repository/postgres/db.go, internal/repository/postgres/migration.go, internal/repository/postgres/migrations/0001_initial.sql, internal/repository/postgres/repositories.go (CRUD methods implemented across category/resource/todo/reminder), internal/repository/postgres/repositories_integration_test.go (real PostgreSQL CRUD gate, DSN-gated)
- WebSocket contracts and handlers: internal/sync/ws_handler.go, api/openapi.yaml (/api/v1/sync/ws, /api/v1/sync/events)
- Conflict resolution and replay logic: internal/sync/conflict.go, internal/sync/offline_replay_manager.go, internal/sync/replay_store.go, internal/sync/replay_store_memory.go, internal/sync/replay_store_sqlite.go, internal/sync/service_mutation_applier.go
- OAuth/JWT middleware and auth integration: internal/auth/jwt.go, internal/auth/jwt_test.go, internal/sync/routes.go, internal/config/config.go, config/config.default.yml (Authorization header only), test/integration/sync_integration_test.go (protected sync endpoint gating assertions)
- Deep processing worker and queue flow: internal/service/deep_processor.go, internal/service/deep_processor_test.go, internal/http/handler.go (enqueue hook + deep processing endpoints), internal/http/deep_processing_handler_test.go, cmd/server/main.go (runtime worker startup), test/integration/deep_processing_integration_test.go
- API contract updates: api/openapi.yaml (phase 2 sync/auth routes + bearer auth responses + /api/v1/processing/deep/* endpoints and schemas)
- UI interaction implementation files: frontend/package.json, frontend/tsconfig.json, frontend/vite.config.ts, frontend/vitest.config.ts, frontend/playwright.config.ts, frontend/src/App.tsx, frontend/src/styles.css, frontend/src/components/layout/Sidebar.tsx, frontend/src/components/layout/Topbar.tsx, frontend/src/components/graph/GraphControls.tsx, frontend/src/components/graph/GraphCanvas.tsx, frontend/src/components/resource/ResourceForm.tsx, frontend/src/components/resource/ResourceList.tsx, frontend/src/components/chat/ChatDock.tsx, frontend/src/components/tasks/TaskBoard.tsx, frontend/src/stores/useLayoutStore.ts, frontend/src/stores/useResourceStore.ts, frontend/src/stores/useChatStore.ts, frontend/src/stores/useSyncStore.ts, frontend/src/stores/useTaskStore.ts, frontend/src/stores/useTaskStore.test.ts, frontend/src/hooks/useFilteredResources.ts, frontend/src/api/client.ts, frontend/test/e2e/tasks.spec.ts, frontend/test/e2e/resources.spec.ts
- Frontend integration, UI, and visual tests: frontend/test/integration/store.msw.test.ts, frontend/test/integration/ui/resource-form.ui.test.tsx, frontend/test/integration/ui/resource-list.ui.test.tsx, frontend/test/integration/ui/task-board.ui.test.tsx, frontend/test/e2e/chat.spec.ts, frontend/test/e2e/navigation.spec.ts, frontend/test/e2e/visual.spec.ts, frontend/test/e2e/visual.spec.ts-snapshots
- VPS deployment assets and runbook: DEPLOYMENT.md, docker-compose.yml (PostgreSQL service + healthcheck), docker-compose.vps.yml, docker-compose.vps.tls.yml, deploy/nginx/selfsystems.conf, deploy/nginx/selfsystems-https.conf (optional TLS template), deploy/ops/ops-checklist.md, deploy/ops/rollback.md, .env.example (PostgreSQL runtime defaults), Makefile (docker-up-postgres + test-postgres gate + verify-sync-runtime)
- Distributed test suites: internal/sync/hub_test.go, test/integration/sync_integration_test.go, test/integration/deep_processing_integration_test.go
- CI workflow updates: .github/workflows/ci.yml (explicit distributed sync/replay gate command + generated distributed evidence report artifact + PostgreSQL service-backed central data integration gate + dedicated frontend unit/E2E/build Playwright gate + Playwright report/test-results artifact upload with CI-side report/trace validation for retry/failure diagnostics)
- Observability and metrics hooks: internal/sync/observability.go, internal/sync/logging.go, internal/sync/routes.go (/api/v1/sync/metrics + /api/v1/sync/health metrics snapshot), internal/sync/ws_handler.go, test/integration/sync_integration_test.go
- Operational troubleshooting guide: deploy/ops/troubleshooting.md, DEPLOYMENT.md
- Cost control configuration: config/config.default.yml (processing.deep throughput/token budget defaults), .env.example (SS_PROCESSING_DEEP_* overrides), internal/config/config.go and internal/config/config_test.go (defaults/env override loader coverage)
- Cost-impact validation notes: deploy/ops/cost-impact.md, DEPLOYMENT.md
- Project workflow guide: Plans/Project_Workflow_Guide.md (workflow conventions, testing, progress tracking)

## Validation Snapshot (To Update Iteratively)

Latest validation commands:

- [x] cd frontend ; npx playwright test test/e2e (2026-05-05, pass after visual snapshot stabilization and graph layout masking)
- [x] cd frontend ; npx playwright test test/e2e/visual.spec.ts --update-snapshots (2026-05-05, pass after expanding visual snapshots for graph/chat/error layouts)
- [x] cd frontend ; npx vitest run test/integration (2026-05-04, pass after adding UI integration coverage and jsdom config)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-30, pass after Workstream 8 task-store validation expansion: todo/reminder required-field checks, invalid date handling, and status sanitization coverage)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-30, pass after Workstream 8 sync-store event-shape expansion: missing/blank type handling and invalid sequence retention)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-29, pass after Workstream 8 sync-store constructor-failure expansion: websocket constructor fallback plus reasonless-close handling)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-29, pass after Workstream 8 sync-store reconnect lifecycle expansion: unknown close-code handling, progressive reconnect backoff/max-delay capping, and reconnect-open recovery)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-29, pass after Workstream 8 sync-store lifecycle recovery expansion: connect/reconnect state transition recovery, stop-cancelled reconnect and refresh timers, and shared websocket reuse while connecting/connected)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 sync-store repeated offline start/polling cadence coverage: repeated start while offline reuses fallback polling and maintains reload cadence)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 sync-store duplicate-start protection coverage: repeated start while connected reuses the existing websocket connection)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 sync-store blank-URL startup coverage: empty websocket URL offline handling plus fallback polling assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 sync-store websocket-message/error coverage: malformed payload ignore, non-mutation heartbeat handling, and transport-error surfacing unit assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 sync-store stop-cleanup coverage: stop resets status, closes websocket, and cancels pending refresh/reconnect work)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 sync-store reconnect/fallback mutation coverage: fallback URL failure, reconnect-after-close, and debounced mutation refresh unit assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 chat-store command handling expansion: blank-input skip, mutation refresh trigger, and failure surfacing unit assertions plus full frontend gate re-validation)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-28, pass after Workstream 8 task-store reminder list-read resilience expansion: reminder load failure retention/error assertions plus full frontend gate re-validation)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-27, pass after Workstream 8 resource list-read API contract parity expansion: malformed-success, non-ok envelope/fallback, and non-array success-data unit assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-27, pass after Workstream 8 task list-read API contract parity expansion: todo/reminder list malformed-success, non-ok envelope/fallback, and non-array success-data unit assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-27, pass after Workstream 8 task-create API contract parity expansion: todo/reminder create malformed-success and non-ok envelope/fallback unit assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-27, pass after Workstream 8 dual parity expansion: API client resource create/update malformed-success and non-ok envelope-contract assertions plus resource-store loadResources silent refresh retention/failure unit coverage)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-27, pass after Workstream 8 resource-store unit parity expansion: create/update validation and failure-path assertions plus selected-resource state-preservation checks for resource mutations)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-27, pass after Workstream 8 resource delete success-envelope parity completion: malformed JSON and empty-body delete success tolerance E2E assertions plus deterministic dispatchEvent-based resource-row selection hardening for update/delete mutation tests)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-27, pass after Workstream 8 Session 58: added store-level list-read resilience tests for resource and task stores (silent refresh retention vs surfaced non-silent errors) and full frontend gate passed)
  - Vitest JSON: frontend/test-results/vitest-results.json
  - Playwright report: frontend/playwright-report (HTML artifacts saved when Playwright runs with reporter=html)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-23, pass after Workstream 8 resource failure-class parity completion: resource create malformed-envelope failure and resource update backend-failure E2E UX/state-preservation assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-22, pass after Workstream 8 resource network/timeout parity expansion: resource create network failure, resource update timeout failure, and resource delete timeout failure E2E UX assertions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-22, pass after Workstream 8 resource E2E expansion: create/update/delete success coverage, backend/malformed failure-path UX assertions, and selector-stability hardening against chat-panel pointer interception/strict-locator collisions)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-21, pass after Workstream 8 resource management hardening: resource delete flow wiring + resource update payload contract fix (`category_name`) + new resource store/API client contract tests)
- [x] cd frontend ; npx vitest run --reporter=default --reporter=json --outputFile=test-results/vitest-results.json ; npm run test:e2e ; npm run build (2026-04-20, pass after Workstream 8 E2E failure-toggle helper extraction and frontend CI unit-reporting resilience/triage alignment)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 API client contract-test hardening: malformed/no-data update-envelope assertions and delete no-data edge-envelope assertions)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 malformed-envelope symmetry hardening: todo update and reminder create malformed-success response E2E coverage, plus deterministic sync websocket test harness stabilization)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 mutation-symmetry hardening: reminder create network failure and todo update timeout failure E2E coverage)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 network/timeout + delete hardening: todo create network failure, reminder mark-sent timeout failure, and todo delete failure-path E2E coverage)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 validation/malformed-response E2E additions: todo/reminder client-side validation errors plus todo create and reminder mark-sent invalid API envelope handling)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 status-transition negative-path E2E additions (Mark Done/Mark Sent failures), expanded task-store mutation error unit coverage, and CI Playwright artifact validation hardening)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 negative-path E2E additions for todo/reminder create failure UX)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 negative-path E2E additions for todo update failure and reminder delete failure UX)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 todo status-transition + reminder delete E2E additions and CI Playwright artifact publishing)
- [x] cd frontend ; npm run test:e2e ; npm test ; npm run build (2026-04-19, pass after Workstream 8 reminder create/status-transition E2E scenario and frontend CI gate expansion)
- [x] cd frontend ; npm test ; npm run test:e2e ; npm run build (2026-04-19, pass after Workstream 8 Playwright E2E scaffold, Tasks linked-resource flow spec, and Vitest e2e exclusion config)
- [x] cd frontend ; npm test ; npm run build (2026-04-18, pass after Workstream 8 task-resource link draft wiring and initial frontend Vitest task-store unit scaffold)
- [x] cd frontend ; npm run build (2026-04-18, pass after Workstream 8 section-aware rendering, Todo/Reminder task board CRUD flows, and task sync-refresh/store wiring)
- [x] cd frontend ; npm run build (2026-04-18, pass after Workstream 8 websocket fallback polling mode and UI launch-doc updates)
- [x] cd frontend ; npm run build (2026-04-18, pass after Workstream 8 realtime websocket sync-store wiring and topbar sync-status integration)
- [x] cd frontend ; npm install react-force-graph-2d react-force-graph-3d ; npm run build (2026-04-18, pass for Workstream 8 force-graph 2D/3D integration and lazy-load chunk split validation)
- [x] cd frontend ; npm install ; npm run build (2026-04-18, pass for Workstream 8 interactive UI scaffold compile gate)
- [x] docker compose -f docker-compose.yml -f docker-compose.vps.yml config ; go test ./... (2026-04-15, pass for final VPS-hosted Go runtime topology overlay validation and full regression safety after workflow updates)
- [x] go run ./scripts/verify_sync_runtime -base-url "https://ydxkcghztcncnvigjrhk.supabase.co/functions/v1" -websocket-path "/api/v1/sync/ws" -timeout-seconds "15" -report-file "artifacts/sync-runtime-reachability-supabase.json" (2026-04-15, pass with remote sync-enabled websocket handshake and sync.connected first-event verification)
- [x] go test ./internal/sync -run "Hub|WS|Observability" ; go test ./test/integration -run "Sync|Offline|Replay|Observability" ; go test ./... ; make distributed-test (2026-04-14, pass after websocket reconnect replay (since_sequence), hub history depth telemetry, burst sequence stability integration coverage, and verifier hub telemetry parsing fix)
- [x] go test ./internal/config -run DeepProcessing ; go test ./internal/service -run DeepProcessor ; go test ./internal/http -run DeepProcessing ; go test ./test/integration -run DeepProcessing (2026-04-14, pass for deep-processing config/service/API/integration coverage)
- [x] go test ./... (2026-04-14, full pass after deep-processing activation + CI evidence/reporting + sync runtime verification artifact wiring)
- [x] go test -json ./internal/sync ./test/integration -run "Sync|Offline|Replay" > artifacts/distributed-sync-go-test.json ; go run ./scripts/generate_distributed_gate_report -input artifacts/distributed-sync-go-test.json -output artifacts/distributed-sync-report.md (2026-04-14, pass with generated distributed evidence report)
- [x] go test ./... (2026-04-11, full pass)
- [x] go test ./test/integration -run Sync (2026-04-11, pass)
- [x] go test ./internal/auth ./internal/repository/postgres ./... (2026-04-12, full pass)
- [x] go test ./... (2026-04-14, full pass after domain interface expansion)
- [x] go test ./internal/http ./internal/service (2026-04-13, pass for category/resource CRUD service+API adoption)
- [x] go test ./... (2026-04-13, full pass after category/resource CRUD endpoint rollout)
- [x] go test ./test/integration -run CategoryResourceCRUD (2026-04-14, pass for integration-first gate)
- [x] go test ./internal/http ./internal/service (2026-04-14, pass for todo/reminder CRUD service+API adoption)
- [x] go test ./test/integration -run CRUD (2026-04-14, pass for category/resource + todo/reminder CRUD integration coverage)
- [x] go test ./... (2026-04-14, full pass after todo/reminder CRUD endpoint rollout)
- [x] go test ./internal/service ./internal/sync (2026-04-14, pass after chat CRUD parity + sync protocol validator)
- [x] go test ./test/integration -run "ChatCRUD|SyncEventProtocol" (2026-04-14, pass)
- [x] go test ./... (2026-04-14, full pass after chat CRUD completion and Workstream 3 kickoff)
- [x] go test ./internal/http ./internal/service ./internal/sync (2026-04-14, pass after CRUD mutation event emission wiring)
- [x] go test ./test/integration -run "ChatCRUD|SyncEventProtocol|CRUD" (2026-04-14, pass)
- [x] go test ./... (2026-04-14, full pass after Workstream 3 mutation event emission slice)
- [x] go test ./internal/sync ./internal/http ./test/integration (2026-04-14, pass after sync payload metadata enrichment and websocket CRUD fanout assertions)
- [x] go test ./... (2026-04-14, full pass after Workstream 3 metadata/source tagging + websocket fanout coverage)
- [x] go test ./internal/sync ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after Workstream 4 conflict resolution and replay queue implementation)
- [x] go test ./... (2026-04-14, full pass after Workstream 4 offline replay/conflict integration)
- [x] go test ./internal/sync ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after replay-to-service application wiring + idempotent enqueue + partial-batch replay safety)
- [x] go test ./... (2026-04-14, full pass after replay service-application and safety/idempotency hardening)
- [x] go test ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after extending replay service-application integration coverage for category/todo/reminder + replay-apply failure persistence assertions)
- [x] go test ./... (2026-04-14, full pass after replay integration coverage expansion)
- [x] go test ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after adding replay negative-path integration assertions for invalid todo/reminder payload variants and retry persistence)
- [x] go test ./... (2026-04-14, full pass after replay negative-path integration coverage expansion)
- [x] go test ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after extending replay negative-path integration assertions to invalid resource/category payload variants and retry persistence)
- [x] go test ./... (2026-04-14, full pass after replay resource/category negative-path integration expansion)
- [x] go test ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after adding malformed resource-create replay payload assertion with retry persistence)
- [x] go test ./... (2026-04-14, full pass after malformed resource-create replay negative-path integration expansion)
- [x] go test ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after adding missing-category resource-create replay edge-case assertion with retry persistence)
- [x] go test ./... (2026-04-14, full pass after missing-category resource-create replay negative-path integration expansion)
- [x] go test ./internal/sync ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass for new CI distributed behavior gate command)
- [x] go test ./... (2026-04-14, full pass after CI distributed behavior gate wiring)
- [x] go test ./internal/sync ./test/integration -run "Sync|Offline|Replay|Observability|Unauthorized" (2026-04-14, pass after adding sync observability metrics endpoint and counter assertions)
- [x] go test ./... (2026-04-14, full pass after sync observability instrumentation wiring)
- [x] go test ./internal/sync ./test/integration -run "Sync|Offline|Replay|Observability|Unauthorized|AuthGates" (2026-04-14, pass after adding comprehensive auth gating coverage across protected sync endpoints)
- [x] go test ./... (2026-04-14, full pass after auth gating integration coverage expansion)
- [x] go test ./internal/repository/postgres -run Integration -v (2026-04-14, DSN-gated test confirmed; local run skipped without SS_POSTGRES_TEST_DSN/SS_DATABASE_URL)
- [x] go test ./internal/sync ./test/integration -run "Sync|Offline|Replay" (2026-04-14, pass after adding CI PostgreSQL central-data gate alongside distributed replay gate)
- [x] go test ./... (2026-04-14, full pass after PostgreSQL integration gate + deployment runbook/infrastructure updates)
- [x] frontend unit gate (`cd frontend ; npm test`) is active with initial task store coverage
- [x] frontend E2E gate (`cd frontend ; npm run test:e2e`) is active with initial Tasks linked-resource flow coverage

Result summary:

- JWT auth middleware now gates sync publish and websocket paths; sync integration tests cover delivery, reconnect, and unauthorized access.
- Workstream 2 scaffold initialized with PostgreSQL adapter package, DB switch config, and first migration set.
- Authorization token ingestion is header-only; query token fallback was removed and covered by auth tests.
- PostgreSQL category and resource repository interfaces are now backed by concrete SQL implementations.
- Full CRUD parity rollout is started with PostgreSQL Todo and Reminder create/list implementations added.
- Domain repository contracts were expanded with explicit GetByID/Update/Delete methods and lockstep adapter implementations started in SQLite and PostgreSQL.
- Service and HTTP layers now expose category/resource GET-by-id, update, and delete paths with explicit not-found responses and expanded handler regression coverage.
- OpenAPI contract now documents category/resource by-id GET/PUT/DELETE routes with 404 not-found response semantics.
- Service and HTTP layers now expose todo/reminder GET-by-id, update, and delete paths with consistent validation and explicit not-found responses.
- API integration tests now cover CRUD round-trips for category/resource and todo/reminder by-id endpoints.
- OpenAPI contract now documents todo/reminder by-id GET/PUT/DELETE routes and update request schemas.
- Chat command actions now support get/update/delete CRUD parity across category/resource/todo/reminder, with service and API integration coverage.
- Workstream 3 kickoff: sync event publish path now validates allowed event types and entity-scoped payload requirements before broadcast.
- OpenAPI now documents the sync publish event protocol schema and expanded chat CRUD command examples.
- Workstream 3 mutation path now emits sync events automatically from CRUD HTTP handlers (and chat-command mutation responses) when a sync hub is configured.
- Workstream 3 sync protocol payloads now include `event_version` and `event_source` metadata across both `/sync/events` publishing and internal CRUD/chat mutation emissions.
- Websocket integration coverage now validates that CRUD-originated sync fanout includes event metadata and entity identifiers.
- Workstream 4 scaffold now includes a deterministic last-write-wins conflict resolver and persistent SQLite-backed offline replay queue/conflict history stores.
- Sync routes now expose authenticated offline queue enqueue/replay and conflict listing endpoints with replay fanout emitted over WebSockets.
- Integration tests now validate conflict winner selection, conflict-history recording, and FIFO replay ordering after reconnect-style queue flush.
- Workstream 4 replay now applies winner mutations through resource/category/todo/reminder services before websocket fanout in runtime wiring.
- Offline replay enqueue is now idempotent by operation ID across memory and SQLite stores, including duplicate retry safety after apply.
- Replay execution now has partial-batch safety: each entity batch is applied/marked independently, replay stops on apply failure, and remaining mutations stay queued.
- Integration coverage for service-applied replay mutations now spans category, resource, todo, and reminder entity updates with persisted state assertions.
- Replay apply-failure integration assertions now verify failed service application returns internal error and leaves queued mutations pending for retry.
- Replay negative-path integration coverage now verifies invalid todo status and invalid reminder timestamp replay payloads fail with internal error, keep queued mutations pending across retries, and leave existing entity state unchanged.
- Replay negative-path integration coverage now additionally verifies invalid resource category references and invalid category replay payloads fail with internal error, remain retry-persistent, and preserve existing entity state.
- Replay negative-path integration coverage now also verifies malformed replay resource-create URLs fail with internal error, remain queued across retries, and do not alter existing resource state.
- Replay negative-path integration coverage now also verifies replayed resource-create events with valid URLs but missing category references fail with internal error, remain queued across retries, and do not create unintended resources.
- CI now includes an explicit distributed behavior gate (`go test ./internal/sync ./test/integration -run "Sync|Offline|Replay"`) in addition to full Go test execution.
- Sync runtime now exposes authenticated observability counters at `/api/v1/sync/metrics`, includes metrics snapshots in `/api/v1/sync/health`, and tracks auth failures, websocket lifecycle, sync event publish outcomes, replay outcomes, and conflict-list request outcomes.
- Sync integration coverage now explicitly validates JWT auth gating across all protected sync endpoints (`/sync/events`, offline queue enqueue/replay, conflict listing, metrics, websocket path) with authorized request-path assertions.
- Central PostgreSQL path now has a dedicated CRUD integration gate (`go test ./internal/repository/postgres -run Integration`) that runs against a real database when DSN is provided.
- CI now provisions a PostgreSQL service container and executes the central-data integration gate, adding automated validation for multi-device write-path readiness.
- Deployment/runbook assets now document a reproducible Phase 2 stack path through `DEPLOYMENT.md`, compose PostgreSQL service defaults, and Makefile helper targets.
- Deep processing is now active in runtime wiring with asynchronous queue workers, deep health/metrics/reprocess endpoints, and integration coverage for summary enrichment (`[deep-processing]`).
- Cost controls for deep processing are now configurable and enforced via throughput (`max_tasks_per_minute`) and daily token budget (`max_tokens_per_day`) controls.
- CI distributed behavior gate now emits explicit evidence artifacts (`distributed-sync-go-test.json`, `distributed-sync-report.md`) and publishes report content in workflow summary.
- Runtime reachability verification artifacts now exist via `scripts/verify_sync_runtime`, Makefile target `verify-sync-runtime`, and manual GitHub workflow `.github/workflows/sync-runtime-reachability.yml`.
- Sync hub now assigns monotonic event sequence numbers and exposes publish/drop counters in `/api/v1/sync/health`; runtime reachability reports now include `sync_hub_*` telemetry fields.
- Sync integration coverage now validates `/api/v1/sync/health` hub telemetry (`published_total`, `dropped_total`, `last_sequence`) after event publication.
- Websocket reconnect behavior now supports buffered replay via `since_sequence` and bounded `replay_limit`, with reconnect payload metadata (`replayed_count`, `last_replayed_sequence`) and de-duplication against live subscription fanout.
- Sync integration coverage now validates replay-after-disconnect sequence continuity and burst publish sequence stability with `dropped_total=0` assertions.
- Sync runtime verifier now correctly parses nested sync hub telemetry from `/api/v1/sync/health` and emits `sync_hub_history_depth` alongside published/dropped/sequence fields.
- Remote runtime reachability now has a deployed Supabase-backed endpoint at `https://ydxkcghztcncnvigjrhk.supabase.co/functions/v1`, with verifier evidence in `artifacts/sync-runtime-reachability-supabase.json` showing `sync_enabled=true`, `websocket_connected=true`, and first websocket event `sync.connected`.
- Final VPS-hosted topology assets now exist in-repo via `Dockerfile`, `docker-compose.vps.yml`, and `deploy/nginx/selfsystems.conf` (Go API container + websocket-aware NGINX reverse proxy + existing datastore services).
- Reachability workflow `.github/workflows/sync-runtime-reachability.yml` now supports warning-free auth modes (`none`, `manual_input`, `github_oidc`) with secure non-manual GitHub OIDC token sourcing and manual bearer fallback.
- Workstream 8 kickoff now includes a modular React + TypeScript + Zustand frontend shell with interactive resource add/edit form, graph controls/surface, filter-aware resource list, and chat command dock scaffolding.
- Workstream 8 graph surface now uses real force-directed rendering (`react-force-graph-2d`/`react-force-graph-3d`) with resource-to-category links, node selection focus, category click-to-filter behavior, and lazy-loaded 2D/3D renderers.
- Workstream 8 realtime slice now includes a dedicated websocket sync store (`useSyncStore`) with reconnect handling, mutation-event driven silent resource refresh, and topbar sync runtime status visibility.
- Workstream 8 resilience slice now includes automatic fallback polling when websocket connectivity is unavailable, with visible runtime status (`Polling`) and README runbook guidance for launching/viewing the frontend UI.
- Workstream 8 navigation is now section-aware (`graph`, `search`, `chat`, `tasks`, `settings`) so sidebar state actively drives purpose-specific panel layouts instead of static shared rendering.
- Workstream 8 tasks slice now includes Todo/Reminder API wiring, dedicated task Zustand store, interactive task board CRUD/status flows, and sync/chat-triggered task refresh behavior.
- Workstream 8 task-resource linking is now wired end-to-end in frontend drafts/forms/list views, and frontend unit testing now has an active Vitest gate (`npm test`) with initial task-store coverage.
- Workstream 8 now includes an active Playwright E2E gate with frontend-owned config and an initial Tasks section scenario that verifies todo creation with linked resource selection via mocked API routes.
- Workstream 8 now includes UI integration coverage (ResourceForm/ResourceList/TaskBoard), with jsdom-backed Vitest config and testing-library dependencies for UI-level validation.
- Workstream 8 E2E coverage now includes reminder creation and status-transition (`Mark Sent`) behavior, and CI now enforces frontend unit + Playwright E2E + build gates in addition to existing Go/distributed checks.
- Workstream 8 E2E coverage now also validates todo update/status progression and reminder delete behavior, and CI now uploads Playwright HTML report/test-results artifacts for faster failure triage.
- Workstream 8 E2E coverage now additionally validates negative task mutation UX paths for create/update/delete/status operations: todo/reminder create failures, todo update + mark-done failures, and reminder delete + mark-sent failures surface deterministic backend error messages while preserving expected list state.
- Workstream 8 E2E coverage now also validates client-side task form validation errors and malformed-success API envelope handling (missing `data` payload) for todo create and reminder status transitions, ensuring deterministic error messaging with no unintended list/status mutation.
- Workstream 8 E2E coverage now additionally validates transport and timeout failure classes for task mutations (todo create network failure and reminder mark-sent timeout response), plus todo delete failure parity with deterministic error-surface and state-preservation assertions.
- Workstream 8 E2E coverage now extends transport/timeout mutation symmetry with reminder create network-failure and todo update timeout assertions, preserving deterministic error surfacing and prior entity state.
- Workstream 8 E2E coverage now extends malformed-success envelope symmetry with todo update and reminder create assertions, and the Playwright harness now uses a deterministic connected websocket mock to prevent sync fallback polling from clearing transient error banners during negative-path checks.
- Workstream 8 resource E2E coverage now includes create/update/delete success flows plus backend and malformed-success envelope failures, asserting deterministic error surfacing and state-preservation behavior for resource mutations.
- Workstream 8 resource E2E coverage now includes mutation transport/timeout parity assertions: create network failure plus update/delete timeout failures with deterministic error-surface and state-preservation checks.
- Workstream 8 resource E2E coverage now includes remaining mutation failure-class parity assertions for create malformed-envelope failure and update backend failure, completing deterministic error-surface/state-preservation checks across create/update failure modes.
- Workstream 8 resource E2E coverage now includes delete success-envelope parity assertions for malformed JSON and empty-body success responses, and resource-row selection now uses deterministic dispatchEvent + retry fallback to prevent overlay-driven update/delete flakiness.
- Workstream 8 E2E coverage now includes chat/navigation/graph flows, sidebar and filter validation, and a dev-only graph selection hook for deterministic node selection assertions.
- Workstream 8 visual snapshot coverage now includes search/graph/chat/tasks/settings layouts plus resource-create and chat-error states, with masked graph layout to stabilize pixel diffs.
- Frontend resource-store unit coverage now includes create/update mutation validation and rejected-operation assertions, plus `loadResources` silent-refresh retention checks for selected-resource failure paths and selected-id clearing behavior when refreshed rows drop the selected entity.
- Frontend task-store unit coverage now includes rejected mutation handling for todo/reminder create/update/delete operations, asserting surfaced error messages and no unintended state mutation.
- Frontend API client unit coverage now includes explicit envelope-contract assertions for resource create/update and task update/delete paths: successful mutation responses without `data` fail deterministically, non-ok responses surface envelope/fallback status errors, and delete no-data paths tolerate empty/malformed success bodies.
- Frontend API client unit coverage now additionally includes todo/reminder create malformed-success and non-ok envelope/fallback assertions, completing explicit task create/update/delete error-surface contract parity.
- Frontend API client unit coverage now additionally includes todo/reminder list read-path malformed-success and non-ok envelope/fallback assertions, plus defensive non-array success-data checks for deterministic empty-list handling.
- Frontend API client unit coverage now additionally includes resource list read-path malformed-success and non-ok envelope/fallback assertions, plus defensive non-array success-data checks for deterministic empty-list handling.
- Frontend resource management now supports delete-from-selection flows in the Resource form/store path, and resource update requests now emit `category_name` payload keys so category edits persist correctly through the API contract.
- Frontend CI now validates Playwright HTML/JSON report presence and enforces trace artifact existence when retries/failures occur before publishing Playwright artifacts.
- Workstream 8 Tasks E2E specs now route failure toggles through reusable helper setters for todo/reminder create/update/delete mutation paths, reducing state-toggle duplication while preserving deterministic negative-path behavior.
- Frontend CI unit gating now runs Vitest with JSON report output and non-empty artifact validation, publishes unit pass/fail and failed-assertion details into the GitHub job summary, and skips Playwright artifact validation when E2E execution is skipped so unit-test failures are not masked by secondary artifact errors.
- Workstream 9 deployment hardening now includes a TLS compose overlay, TLS-enabled NGINX template, ops checklist, and rollback playbook referenced in the deployment runbook.
- Workstream 11 observability now includes structured sync logging, replay queue depth metrics, and an operational troubleshooting guide.
- Workstream 12 cost controls now include batching, deduped enqueue protection, min reprocess interval caching, and cost-impact validation notes.

## Notes

This checklist is intentionally initialized with unchecked items. It should be updated continuously as each Phase 2 workstream is implemented and validated.

Local runtime note (2026-04-14): `docker compose up -d postgres` was attempted for an on-machine PostgreSQL execution check but Docker Desktop daemon was unavailable in this environment; CI now carries the real PostgreSQL execution gate via service container.
Local runtime note (2026-04-14): `go run ./scripts/verify_sync_runtime -base-url http://127.0.0.1:8080` was executed to validate reachability artifact behavior and correctly failed with connection-refused errors because no local server instance was running; report output was still generated at `artifacts/sync-runtime-reachability-local.json`.
Remote runtime note (2026-04-15): Supabase Edge Functions (`health`, `api`) were deployed as the reachable remote verification target for `/health`, `/api/v1/sync/health`, and `/api/v1/sync/ws` handshake coverage; this provides remote URL evidence even though it is not yet the final VPS-hosted Go binary topology.
