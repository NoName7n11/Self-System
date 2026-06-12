# Change 12 Workstream - Production Hardening

Date: 2026-06-10
Status: Complete (all 6 workstreams done; WS3's "CI fails on a known vulnerable dependency" criterion is implemented via `.github/workflows/security.yml` but not yet exercised by a live CI run)

> **Numbering note (resolved 2026-06-12, Session 54):** `Plans/Progress_Changes/Changes.md`
> previously had an unrelated "# Change 12: Change-Documenter Skill and Session Tracking
> Infrastructure" entry (dated 2026-06-10) colliding with this workstream's number. That
> entry has been renumbered to **Change 15** in `Changes.md`, and this workstream now has
> its own "# Change 12: Production Hardening" section there summarizing WS1-6
> (`Changes_log.md` Sessions 44-49).

Scope: Close the production-grade gaps in durability, deep-queue reliability, security, observability, and AI cost control. This is the safety layer that must land before Phase 2 sync is shipped to any real device.

## Objective

The architecture is sound but operationally thin. A crash can strand resources in "processing" forever; SQLite has no backup story and no busy_timeout; the URL extractor is an SSRF surface; API keys sit plaintext on disk; there are no metrics and no dependency scanning. None of these are architectural — they are the difference between a demo and software you can trust with your own data. This change makes the system survivable.

## Guiding Constraints

- No new external service dependencies (stay local-first, single-binary). Prefer stdlib (`expvar`, `slog`) over heavy deps.
- Every durability change must be reversible and covered by a test that simulates the failure it prevents.
- Security fixes take priority within this change — SSRF and key storage before niceties.
- Feature-flag anything that changes runtime behavior so it can be toggled off.
- Must not regress the existing test suite (`go test ./...`).

## Workstream 1 — Data Durability

Objective:
Make the SQLite store survivable: no silent corruption, no SQLITE_BUSY deaths, recoverable backups.

Key tasks:
- [x] Add `PRAGMA busy_timeout` (e.g. 5000ms) to the SQLite open path (`internal/repository/sqlite/db.go`) — WAL is already set, busy_timeout is not. Implemented via `_pragma` DSN params (`journal_mode(WAL)`, `foreign_keys(1)`, `busy_timeout(5000)`) so every pooled connection gets the pragmas, not just the one that ran the original `db.Exec`.
- [x] Periodic `VACUUM INTO` snapshot to a timestamped backup file with a retention policy (keep N most recent).
- [x] Versioned schema migration runner for SQLite (version table + ordered migrations), with backup-before-migrate on startup.

Deliverables:
- [x] Updated `internal/repository/sqlite/db.go` (busy_timeout via DSN pragmas).
- [x] `internal/repository/sqlite/backup.go` — VACUUM INTO snapshot + retention + `StartBackupScheduler`.
- [x] SQLite migration runner + version table (`schema_migrations`, `internal/repository/sqlite/migration.go`). Existing schema + add-column migrations split into versions 1 and 2.
- [x] Tests: `db_test.go` (busy_timeout under 16 concurrent writers), `migration_test.go` (backup round-trip, prune retention, migrate-from-empty, migrate-idempotent, backup-before-pending-migration).

Done criteria:
- [x] Concurrent writers do not error with SQLITE_BUSY.
- [x] A backup snapshot can be restored to a working database.
- [x] Schema migrations run once, in order, with a pre-migration backup (only when upgrading an existing DB with pending migrations — fresh DBs skip the backup since there's nothing to back up).

Additional: `internal/config` gained `database.backup_interval_minutes` (default 60) and `database.backup_retention` (default 7); `cmd/server/main.go` and `cmd/desktop/main.go` start/stop `StartBackupScheduler` alongside the DB connection.

## Workstream 2 — Deep-Queue Reliability

Objective:
The deep-processing queue is an in-memory channel today — lost on crash, leaving resources stuck "processing." Make it durable and self-healing.

Key tasks:
- [x] DB-backed queue table (pending / in-progress / done / failed-retryable) replacing the volatile channel as the source of truth; the channel becomes a runtime dispatch buffer fed from the table (`internal/service/deep_queue_store.go`, fed by `runFeeder` in `deep_processor.go`).
- [x] Resume on restart: `DeepQueueStore.ResumeStuck` requeues in-progress rows back to pending, called once in `Start()` before the feeder/workers start.
- [x] Retry with exponential backoff + max attempts (`deepQueueBackoff`: base 30s, cap 30m, default max_attempts 5).
- [x] Dead-letter state: exhausted jobs land in `failed_retryable` with `last_error` recorded; surfaced via `Metrics().DeadLetterTotal`.

Deliverables:
- [x] Queue table migration (`internal/repository/sqlite/migration.go` v3 `deep_queue`) + `internal/service/deep_queue_store.go`.
- [x] Updated `internal/service/deep_processor.go` (DB-backed enqueue/claim/complete/fail via `WithQueueStore`, resume-on-start, `runFeeder`). Wired into `cmd/server/main.go` (`buildRepositories` now also returns the raw `*sql.DB` for sqlite; postgres returns nil, keeping `DeepQueueStore` a no-op there).
- [x] Tests: `internal/service/deep_queue_store_test.go` (enqueue/claim/complete, dedup, retry/backoff, dead-letter, resume-stuck, nil-store no-op, backoff table) and `internal/service/deep_processor_queue_test.go` (crash-mid-job resume on restart, dead-letter transition end-to-end with metrics).

Done criteria:
- [x] Killing the process mid-job leaves the job re-runnable on restart (not stuck) — `TestDeepProcessor_QueueStore_ResumesInProgressOnRestart`.
- [x] Failed jobs reach a retryable dead-letter state with a recorded reason — `TestDeepProcessor_QueueStore_DeadLettersExhaustedJob`, `TestDeepQueueStore_FailRetriesWithBackoffThenDeadLetters`.

## Workstream 3 — Security Hardening

Objective:
Close the externally-facing risk surfaces.

Key tasks:
- [x] SSRF guard in the URL fetcher: block private/loopback/link-local ranges (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, ::1, plus 100.64.0.0/10, 192.0.0.0/24, 198.18.0.0/15, 0.0.0.0/8), cap redirects, cap response size. Implemented as a shared `safeDialContext` in `internal/extractor/ssrf.go`, applied to both `URLExtractor` and `ContentFetcher` via `http.Transport.DialContext`. Resolves DNS then dials the validated IP directly to close the DNS-rebinding TOCTOU gap. Redirect caps (3 for skim, 5 for deep) and response-size caps (2 MiB skim, 20 MiB deep) were already present and remain unchanged.
- [x] PDF extraction: run per-file with panic recovery + timeout so a malformed PDF cannot kill the worker. `PDFExtractor.Extract` now runs the parse on a goroutine with `recover()` and a 20s timeout (`pdfExtractTimeout`); the original logic moved to unexported `extract`.
- [x] API keys → OS keychain (`zalando/go-keyring`); fall back to env/config only when keychain unavailable or empty, with a warning. `internal/config/keyring.go` adds `resolveAPIKey`, applied to OpenAI/Anthropic/Gemini `APIKey` in `config.Load()`.
- [x] CI dependency scanning: `govulncheck` (Go) + `npm audit` (frontend) + `golangci-lint` (errcheck, staticcheck, govet, unused) as a workflow.

Deliverables:
- [x] New `internal/extractor/ssrf.go` (SSRF guard, `AllowLoopbackForTests` test hook) + `ssrf_test.go` (private-IP/loopback/literal-IP cases). `fetcher.go` and `url_extractor.go` wire `safeDialContext` into their `http.Transport`.
- [x] PDF panic-recovery + timeout wrapper in `internal/extractor/pdf_extractor.go` (`Extract`/`extract` split) + `TestPDFExtractor_TimeoutDoesNotHang` in `pdf_extractor_test.go`.
- [x] `internal/config/keyring.go` (`resolveAPIKey`) + `keyring_test.go` (mocked keychain via `keyring.MockInit()`); wired into `config.Load()`. New dependency `github.com/zalando/go-keyring`.
- [x] `.github/workflows/security.yml` (govulncheck + golangci-lint + npm audit) and `.golangci.yml`.

Done criteria:
- [x] Fetching a private-IP / loopback URL is refused (`TestSafeDialContext_BlocksLoopbackByDefault`, `TestSafeDialContext_BlocksPrivateIPLiteral`); redirect/oversize caps were pre-existing and remain covered by `ContentFetcher`/`URLExtractor` config.
- [x] A malformed PDF fails its one job without crashing the worker — panic recovery + 20s timeout wrapper (`TestPDFExtractor_InvalidBytes`, `TestPDFExtractor_TimeoutDoesNotHang`).
- [x] API keys are not required to sit in plaintext on disk — OS keychain consulted first via `resolveAPIKey`, env/config remains only the fallback.
- [ ] CI fails on a known vulnerable dependency — `security.yml` workflow added (govulncheck/golangci-lint/npm audit); not yet exercised by an actual CI run since this requires a push/PR to `master`.

## Workstream 4 — Observability

Objective:
Make the running system inspectable locally.

Key tasks:
- [x] Counters: queue depth (pre-existing `DeepProcessor.Metrics()`), AI latency + call/error counts (`ai.Manager.Metrics()`, new), extraction failures (new `DeepProcessingMetrics.ExtractionFailuresTotal`), sync lag (pre-existing `ObservabilitySnapshot.ReplayQueueOldestSeconds` / `EventsHealthSnapshot.OutboxLagSequences`).
- [x] Structured `slog` with request ID threaded via `RequestIDMiddleware` + `RequestIDFromContext`, included in `respondInternalError` log lines.
- [x] Health endpoint with component checks: DB writable (`*sql.DB.PingContext`), queue alive (`DeepProcessor.Health()`), AI provider reachable (`Manager.ProviderNames()`).

Deliverables:
- [x] **Deviation from literal spec**: rather than a new `internal/observability/metrics.go` (expvar) package, extended the existing JSON-metrics-endpoint pattern already used by `DeepProcessingMetrics` and `sync.ObservabilitySnapshot` — avoids a parallel/duplicate metrics system. New: `internal/ai/observability.go` (`Manager.Metrics()`, atomic counters for classify/enrich/embed calls/errors/avg latency ms) wired into `internal/ai/manager.go`, `enrichment.go`, `embedding.go` via named-return + `defer recordCall(...)`. `DeepProcessingMetrics.ExtractionFailuresTotal` added to `internal/service/deep_processor.go`, incremented on PDF/image fetch-or-extract failure in `runPDFExtraction`/`runImageExtraction`.
- [x] `internal/http/request_id.go` (new): `RequestIDMiddleware()` + `RequestIDFromContext()`, wired into `cmd/server/main.go` router middleware chain; `respondInternalError` now logs `request_id`.
- [x] `/api/v1/health` (new, unauthenticated, alongside existing root `/health`): `internal/http/handler.go` `healthDetailed` — reports `database`/`deep_queue`/`ai` component status; `WithDB`/`WithAIManager` handler options wired in `cmd/server/main.go`.
- Note: "sync lag" was already covered by pre-existing `internal/sync/observability.go` (`ReplayQueueOldestSeconds`, exposed via `/api/v1/sync/metrics`) and `internal/sync/routes.go` (`OutboxLagSequences`, exposed via `/api/v1/sync/events/health`) from prior workstreams — no new work needed for that sub-item.

Done criteria:
- [x] Queue depth, AI latency/cost, extraction failures, and sync lag are observable at runtime — `ai.Manager.Metrics()`, `DeepProcessor.Metrics().ExtractionFailuresTotal`, sync metrics endpoints above. Tests: `TestManagerMetrics_TracksClassifyCallsAndErrors`, `TestManagerProviderNames` (`internal/ai/manager_test.go`), `TestDeepProcessor_ExtractionFailure_IncrementsCounter` (`test/integration/extraction_integration_test.go`).
- [x] Health endpoint returns per-component status — `TestHealthDetailed_ReportsComponentStatus`, `TestRequestIDMiddleware_SetsResponseHeader` (`internal/http/handler_test.go`). `go test ./...`, `gofmt -l .`, `go vet ./...` all pass clean.

## Workstream 5 — AI Cost Control

Objective:
Stop paying twice and make enrichment re-runnable.

Key tasks:
- [x] Content-hash cache for AI results — a re-shared URL / identical content must not re-bill.
- [x] Record model name + prompt version in `extracted_data` per enrichment, enabling selective re-enrichment when prompts improve.
- [x] Finish the per-day token budget (BudgetStatePath exists) + per-provider circuit breaker on repeated failures.

Deliverables:
- [x] `internal/ai/result_cache.go` — generic `ResultCache[T]` (sha256 `ContentHash`, TTL-based, default 24h), tested via `TestManager_EnrichResource_CachesIdenticalContent` (`enrichment_test.go`) / `TestManager_GenerateEmbedding_CachesIdenticalContent` (`embedding_test.go`). Wired into `Manager.EnrichResource` (key = hash(title,url,content)) and `Manager.GenerateEmbedding` (key = hash(text)); cache hits recorded as `EnrichCacheHitsTotal`/`EmbedCacheHitsTotal` in `ai.Manager.Metrics()`.
- [x] `extracted_data` schema fields for model/prompt version — `internal/domain/entities.go` `ResourceExtractedData.EnrichmentProvider/EnrichmentModel/EnrichmentPromptVersion`. `internal/ai/enrichment.go` adds `EnrichmentResult.Model`/`PromptVersion` + `EnrichmentPromptVersion = "v1"` const; `OpenAIEnrichmentProvider.Enrich` stamps both. `internal/service/deep_processor.go` `runEnrichment` persists all three provenance fields.
- [x] Completed token-budget enforcement (pre-existing `reserveTokenBudget`/`loadBudgetState`/`persistBudgetState` confirmed working, no changes needed) + new `internal/ai/circuit_breaker.go` (`circuitBreaker`: per-provider consecutive-failure counter, opens circuit for 60s after 3 consecutive failures via `circuitBreakerThreshold`/`circuitBreakerCooldown`). Wired into `classify`, `EnrichResource`, and `GenerateEmbedding` provider loops via `Allow`/`RecordResult`. `Manager.OpenCircuitProviders()` exposed via `ai.Manager.Metrics().OpenCircuitProviders` for `/api/v1/health`/observability.

Done criteria:
- [x] Re-processing identical content hits the cache, not the provider — `TestManager_EnrichResource_CachesIdenticalContent`, `TestManager_GenerateEmbedding_CachesIdenticalContent` (provider call count stays at 1 across two identical calls).
- [x] Each enrichment records which model + prompt version produced it — `TestManager_EnrichResource_RecordsModelAndPromptVersion`.
- [x] Daily token budget halts spend (pre-existing, unchanged); a failing provider trips its breaker and falls back — `TestManager_CircuitBreaker_OpensAfterRepeatedFailures` (provider's circuit opens after `circuitBreakerThreshold` consecutive failures, fallback still succeeds). `go test ./...`, `gofmt -l .`, `go vet ./...` all pass clean.

## Workstream 6 — Reliability Glue

Objective:
Graceful lifecycle + idempotent writes.

Key tasks:
- [x] Graceful shutdown on SIGTERM: drain deep-processor queue, flush outbox, close WS connections.
- [x] HTTP-layer idempotency keys for resource creation / sync retries (extend ADR 0011 event idempotency to the API edge).

Deliverables:
- [x] Updated `cmd/server/main.go` shutdown sequence: `server.Shutdown` (stop new HTTP, drain in-flight requests, 10s) → `runtimeCancel()` (stops deep-queue feeder, GBUS monitors, outbox worker from claiming/starting new work) → `deepProcessor.Shutdown(drainCtx)` (30s, waits for any in-flight deep-processing job to finish) → `syncHub.CloseAll()` (unblocks websocket handlers). `cmd/desktop/main.go` runs no HTTP server/deep processor/sync hub, so no shutdown sequence change was needed there.
- [x] `internal/service/deep_processor.go`: new `wg sync.WaitGroup` + `workCtx`/`workCancel` (separate from the `Start` ctx so in-flight `processTask` calls aren't aborted the instant `runtimeCancel()` fires) and exported `Shutdown(ctx context.Context)` that waits on `wg` (with a fallback timeout) then cancels `workCtx`. `internal/sync/hub.go`: new `Hub.CloseAll()` closes and clears all subscriber channels.
- [x] `internal/http/idempotency.go` (new) — `IdempotencyStore` (in-memory, TTL-based, default 24h) + `IdempotencyMiddleware`: for POST/PUT/PATCH requests carrying an `Idempotency-Key` header, caches the first response (status/headers/body) keyed by method+path+key and replays it (with `Idempotent-Replay: true`) on retries instead of re-running the handler. Wired into `cmd/server/main.go`'s middleware chain.
- [x] Tests: `internal/service/deep_processor_test.go` `TestDeepProcessor_Shutdown_DrainsInFlightWork` / `TestDeepProcessor_Shutdown_NeverStartedIsNoOp`; `internal/sync/hub_test.go` `TestHubCloseAllClosesSubscriberChannels`; `internal/http/idempotency_test.go` `TestIdempotencyMiddleware_ReplaysCachedResponse` / `TestIdempotencyMiddleware_DifferentKeysNotCached`.

Done criteria:
- [x] SIGTERM drains in-flight work instead of dropping it — `server.Shutdown` → `runtimeCancel` → `deepProcessor.Shutdown` → `syncHub.CloseAll` sequence in `cmd/server/main.go`; `TestDeepProcessor_Shutdown_DrainsInFlightWork`.
- [x] Repeated create with the same idempotency key does not duplicate — `TestIdempotencyMiddleware_ReplaysCachedResponse` (handler called once across two identical requests, second response replayed from cache). `go test ./...`, `gofmt -l .`, `go vet ./...` all pass clean.

## Planned Milestones

- [x] Milestone 12A: SQLite durable — busy_timeout, backup, versioned migrations (WS1).
- [x] Milestone 12B: Deep queue durable + self-healing (WS2).
- [x] Milestone 12C: Security surfaces closed + CI scanning (WS3).
- [x] Milestone 12D: Metrics, structured logs, health checks live (WS4).
- [x] Milestone 12E: AI cost cache + budget + provenance (WS5).
- [x] Milestone 12F: Graceful shutdown + idempotency (WS6).

## Change 12 Definition of Done

- [x] SQLite has busy_timeout, automated backups, and versioned migrations with pre-migrate backup.
- [x] The deep-processing queue survives a crash and self-heals; no silent limbo.
- [x] SSRF, malformed-PDF, and plaintext-key risks are closed; CI scans dependencies.
- [x] Queue depth, AI cost/latency, extraction failures, and sync lag are observable; health endpoint checks components.
- [x] AI results are content-hash cached, budget-capped, and provenance-stamped.
- [x] Graceful shutdown drains work; resource creation is idempotent.
- [x] `go test ./...` passes with no regressions.
