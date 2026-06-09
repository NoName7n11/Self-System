# Check 1 Security Patchs Planing

Date: 2026-05-07
Status: Planned, execution starting now
Scope: Security hardening and patch implementation based on findings documented in `Loopholes.md`.

## Check 1 Objective

Check 1 addresses security gaps identified in `Loopholes.md` and converts them into verified patches with tests, safer defaults, and operational controls.

## Guiding Constraints

- Prioritize exploitability reduction first (Critical, then High, then Medium, then Low).
- Land small, reviewable patches that keep behavior predictable.
- Require regression tests for every security-sensitive code path changed.
- Do not weaken existing API contracts unless required for security.
- Prefer explicit middleware and typed error handling over implicit behavior.

## Workstream 1 - Immediate DoS Surface Reduction (Critical)

Objective:
Close high-impact unbounded input paths that can exhaust memory/CPU.

Key tasks:
- Clamp pagination limits in REST handlers.
- Clamp replay/conflict list limits in sync routes.
- Add bounded defaults for all list-style endpoints.

Deliverables:
- Upper-bound checks in `internal/http/handler.go` and `internal/sync/routes.go`.
- Tests covering oversized limits and fallback behavior.

Done criteria:
- Oversized limits are consistently capped.
- No unbounded list/read path remains in public handlers.

## Workstream 2 - Safe Error Handling and Response Hygiene (Critical)

Objective:
Prevent internal implementation details from leaking to clients.

Key tasks:
- Replace string-sniffed validation classification with typed/sentinel errors.
- Return generic messages for 5xx responses.
- Log detailed internal errors server-side only.

Deliverables:
- Updated error mapping logic in HTTP layer.
- New typed error helpers in service/domain boundary.
- Tests ensuring no raw DB/provider errors are returned in 5xx responses.

Done criteria:
- 5xx responses do not include stack/internal/provider/DB details.
- 4xx and 5xx semantics remain correct and deterministic.

## Workstream 3 - Auth Boundary Hardening (High)

Objective:
Ensure all mutation and sensitive read paths are explicitly protected.

Key tasks:
- Apply auth middleware explicitly to REST API groups.
- Protect or sanitize `/api/v1/auth/health` and `/api/v1/sync/health` outputs.
- Re-validate chat mutation and deep reprocess routes under auth.

Deliverables:
- Route-group middleware wiring updates.
- Health endpoint split between public liveness and protected diagnostics.
- Unauthorized integration test coverage.

Done criteria:
- Sensitive endpoints reject unauthenticated requests.
- Public health endpoints expose minimal non-sensitive data.

## Workstream 4 - Secret Handling and Provider Security (High)

Objective:
Eliminate credential exposure via URLs and logs.

Key tasks:
- Move Gemini API key from query string to `x-goog-api-key` header.
- Confirm provider URL construction contains no embedded secrets.

Deliverables:
- Updated Gemini provider request construction.
- Unit tests asserting no key material appears in URL.

Done criteria:
- All AI provider authentication uses headers only.

## Workstream 5 - Rate Limiting and Abuse Controls (High)

Objective:
Mitigate request/connection flooding and spend-amplification paths.

Key tasks:
- Add per-IP/per-subject rate limiter middleware for HTTP mutations.
- Add websocket connection limit per client identity.
- Add inbound WS message-frequency guardrails where applicable.

Deliverables:
- HTTP limiter middleware with configurable thresholds.
- WS connection tracking and rejection policy.
- 429/close-code behavior tests.

Done criteria:
- Burst traffic beyond policy is throttled/rejected predictably.
- Resource/AI-cost amplification paths are bounded.

## Workstream 6 - Input and Payload Validation Hardening (High/Medium)

Objective:
Reduce storage amplification and malformed payload risk.

Key tasks:
- Add max-length constraints for titles, summaries, details, messages, IDs.
- Add request body size limits on REST mutation paths.
- Validate offline-queue operation_id/type/payload size.
- Escape wildcard special characters for SQL LIKE search.

Deliverables:
- Centralized validation constants and helpers.
- Handler/service input checks and deterministic error responses.
- Repository search escape behavior update.

Done criteria:
- Oversized payloads are rejected early.
- Search wildcard bypass behavior is removed.

## Workstream 7 - Sync Protocol Integrity Hardening (Medium)

Objective:
Improve correctness and safety of event replay/stream semantics.

Key tasks:
- Add payload key allowlists for inbound sync events.
- Remove `id` fallback in replay entity inference; require `entity_id`.
- Add replay truncation signaling or oldest-window semantics.

Deliverables:
- Protocol validation updates.
- Replay manager/hub semantics update with tests.

Done criteria:
- Event identity and replay behavior are explicit and non-ambiguous.

## Workstream 8 - Deep Processing Cost-Control Hardening (Medium)

Objective:
Prevent budget bypass and manipulation of expensive routing.

Key tasks:
- Base complexity routing on hostname/domain, not arbitrary URL substrings.
- Persist daily token budget in durable storage.
- Improve error retention/aggregation for incident diagnostics.

Deliverables:
- Deep processor scoring update.
- Persistent budget state model and integration.
- Improved metrics/error visibility.

Done criteria:
- Restart does not reset daily token budget.
- Cost routing cannot be trivially gamed by path string injection.

## Workstream 9 - Verification, Contracts, and Release Gate

Objective:
Ship security patches with enforceable quality gates and documentation.

Key tasks:
- Add/expand unit + integration tests for all patched paths.
- Update OpenAPI where response/auth behavior changed.
- Update deployment/ops docs for new security controls.

Deliverables:
- Passing security-focused test slices in CI.
- Updated docs and runbook notes.
- Security patch completion checklist.

Done criteria:
- All `Loopholes.md` items are either fixed or explicitly risk-accepted with rationale.

## Planned Check 1 Milestones

- Milestone 3A: Critical Surface Closure
  - Complete DoS bound fixes and internal-error leak removal.
- Milestone 3B: Auth + Secret Hardening
  - Complete auth boundary fixes and Gemini key handling patch.
- Milestone 3C: Abuse Controls and Validation
  - Complete rate limiting, payload bounds, and queue/search hardening.
- Milestone 3D: Sync/Deep Integrity and Final Gate
  - Complete sync semantics hardening, deep-budget persistence, and full regression gate.

## Check 1 Definition of Done

- Critical and High findings from `Loopholes.md` are fixed and tested.
- Medium findings are fixed or documented with explicit risk acceptance.
- No known endpoint leaks internal 5xx implementation details.
- Auth boundaries are explicit and covered by tests.
- Rate limiting and payload limits are active for abuse-prone paths.
- Security-focused CI checks pass and documentation is updated.