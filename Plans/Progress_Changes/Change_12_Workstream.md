# Change 12 Workstream - Production Hardening

Date: 2026-06-10
Status: Planned
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
- [ ] Add `PRAGMA busy_timeout` (e.g. 5000ms) to the SQLite open path (`internal/repository/sqlite/db.go`) — WAL is already set, busy_timeout is not.
- [ ] Periodic `VACUUM INTO` snapshot to a timestamped backup file with a retention policy (keep N most recent).
- [ ] Versioned schema migration runner for SQLite (version table + ordered migrations), with backup-before-migrate on startup.

Deliverables:
- [ ] Updated `internal/repository/sqlite/db.go` (busy_timeout).
- [ ] `internal/repository/sqlite/backup.go` — VACUUM INTO snapshot + retention.
- [ ] SQLite migration runner + version table.
- [ ] Tests: busy_timeout under concurrent writers, backup round-trip, migrate-from-empty and migrate-idempotent.

Done criteria:
- [ ] Concurrent writers do not error with SQLITE_BUSY.
- [ ] A backup snapshot can be restored to a working database.
- [ ] Schema migrations run once, in order, with a pre-migration backup.

## Workstream 2 — Deep-Queue Reliability

Objective:
The deep-processing queue is an in-memory channel today — lost on crash, leaving resources stuck "processing." Make it durable and self-healing.

Key tasks:
- [ ] DB-backed queue table (pending / in-progress / done / failed-retryable) replacing the volatile channel as the source of truth; the channel becomes a runtime dispatch buffer fed from the table.
- [ ] Resume on restart: requeue in-progress/pending rows.
- [ ] Retry with exponential backoff + max attempts.
- [ ] Dead-letter state: exhausted jobs land in `failed, retryable`, never silent limbo. Surface count in metrics.

Deliverables:
- [ ] Queue table migration + `internal/service/deep_queue_store.go`.
- [ ] Updated `internal/service/deep_processor.go` (DB-backed enqueue/claim/complete/fail, resume-on-start).
- [ ] Tests: crash-mid-job resume, retry/backoff, dead-letter transition.

Done criteria:
- [ ] Killing the process mid-job leaves the job re-runnable on restart (not stuck).
- [ ] Failed jobs reach a retryable dead-letter state with a recorded reason.

## Workstream 3 — Security Hardening

Objective:
Close the externally-facing risk surfaces.

Key tasks:
- [ ] SSRF guard in the URL fetcher: block private/loopback/link-local ranges (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, ::1), cap redirects, cap response size.
- [ ] PDF extraction: run per-file with panic recovery + timeout so a malformed PDF cannot kill the worker.
- [ ] API keys → OS keychain (`zalando/go-keyring`); fall back to env only when keychain unavailable, with a warning.
- [ ] CI dependency scanning: `govulncheck` (Go) + `npm audit` (frontend) + `golangci-lint` (errcheck, staticcheck) as a workflow.

Deliverables:
- [ ] Updated `internal/extractor/fetcher.go` (SSRF guard) + tests with private-IP/redirect/oversize cases.
- [ ] PDF panic-recovery + timeout wrapper + malformed-PDF test fixture.
- [ ] `internal/config` keychain integration + fallback.
- [ ] `.github/workflows/security.yml` (govulncheck + npm audit) and `.golangci.yml`.

Done criteria:
- [ ] Fetching a private-IP / oversized / redirect-looping URL is refused.
- [ ] A malformed PDF fails its one job without crashing the worker.
- [ ] API keys are not required to sit in plaintext on disk.
- [ ] CI fails on a known vulnerable dependency.

## Workstream 4 — Observability

Objective:
Make the running system inspectable locally.

Key tasks:
- [ ] Counters via stdlib `expvar` (or prometheus/client_golang if preferred): queue depth, AI latency + estimated cost, extraction failures, sync lag.
- [ ] Structured `slog` with request ID + resource ID threaded through the pipeline log lines.
- [ ] Health endpoint with component checks: DB writable, queue alive, AI provider reachable.

Deliverables:
- [ ] `internal/observability/metrics.go` (expvar counters) wired into deep_processor, AI manager, sync hub.
- [ ] Updated logging to include correlation IDs.
- [ ] Updated/added `/health` (or `/api/v1/health`) component-check handler.

Done criteria:
- [ ] Queue depth, AI latency/cost, extraction failures, and sync lag are observable at runtime.
- [ ] Health endpoint returns per-component status.

## Workstream 5 — AI Cost Control

Objective:
Stop paying twice and make enrichment re-runnable.

Key tasks:
- [ ] Content-hash cache for AI results — a re-shared URL / identical content must not re-bill.
- [ ] Record model name + prompt version in `extracted_data` per enrichment, enabling selective re-enrichment when prompts improve.
- [ ] Finish the per-day token budget (BudgetStatePath exists) + per-provider circuit breaker on repeated failures.

Deliverables:
- [ ] `internal/ai/result_cache.go` (content-hash keyed) + tests.
- [ ] `extracted_data` schema fields for model/prompt version.
- [ ] Completed token-budget enforcement + provider circuit breaker.

Done criteria:
- [ ] Re-processing identical content hits the cache, not the provider.
- [ ] Each enrichment records which model + prompt version produced it.
- [ ] Daily token budget halts spend; a failing provider trips its breaker and falls back.

## Workstream 6 — Reliability Glue

Objective:
Graceful lifecycle + idempotent writes.

Key tasks:
- [ ] Graceful shutdown on SIGTERM: drain deep-processor queue, flush outbox, close WS connections.
- [ ] HTTP-layer idempotency keys for resource creation / sync retries (extend ADR 0011 event idempotency to the API edge).

Deliverables:
- [ ] Updated `cmd/server/main.go` (and `cmd/desktop/main.go`) shutdown sequence.
- [ ] Idempotency-key middleware + store.
- [ ] Tests: shutdown drains without loss; duplicate create with same key is a no-op.

Done criteria:
- [ ] SIGTERM drains in-flight work instead of dropping it.
- [ ] Repeated create with the same idempotency key does not duplicate.

## Planned Milestones

- [ ] Milestone 12A: SQLite durable — busy_timeout, backup, versioned migrations (WS1).
- [ ] Milestone 12B: Deep queue durable + self-healing (WS2).
- [ ] Milestone 12C: Security surfaces closed + CI scanning (WS3).
- [ ] Milestone 12D: Metrics, structured logs, health checks live (WS4).
- [ ] Milestone 12E: AI cost cache + budget + provenance (WS5).
- [ ] Milestone 12F: Graceful shutdown + idempotency (WS6).

## Change 12 Definition of Done

- [ ] SQLite has busy_timeout, automated backups, and versioned migrations with pre-migrate backup.
- [ ] The deep-processing queue survives a crash and self-heals; no silent limbo.
- [ ] SSRF, malformed-PDF, and plaintext-key risks are closed; CI scans dependencies.
- [ ] Queue depth, AI cost/latency, extraction failures, and sync lag are observable; health endpoint checks components.
- [ ] AI results are content-hash cached, budget-capped, and provenance-stamped.
- [ ] Graceful shutdown drains work; resource creation is idempotent.
- [ ] `go test ./...` passes with no regressions.
