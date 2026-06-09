# Loopholes_2.md — Security Audit (Round 2)

Scope: New findings discovered after Check 1 closure, evaluated against the NIST Cybersecurity Framework and MASVS categories (CODE, NETWORK, AUTH, STORAGE, CRYPTO). All `Loopholes.md` items are already closed and verified in `Plans/Security_Check/Check_1_Timeline.md` — this document tracks the next batch.

Date: 2026-05-08
Source: Codebase audit of `internal/`, `cmd/server/`, `migrations/`, `config/`, `frontend/src/`.

---

## CRITICAL

### [N1] Unsafe Type Assertion in Rate Limiter — MASVS-CODE
- **File:** `internal/http/rate_limit.go:38`
- **Code:** `subject.(string)` — unchecked type assertion against `auth.subject` context value.
- **Risk:** If any future middleware ever sets `auth.subject` to a non-string value (e.g., a struct or numeric ID), this assertion panics and the request goroutine crashes. With Gin's default recovery missing on the limiter path, this can be a denial-of-service trigger.
- **Fix:** Use the comma-ok form:
  ```go
  if s, ok := subject.(string); ok {
      trimmed := strings.TrimSpace(s)
      if trimmed != "" {
          key = "sub:" + trimmed
      }
  }
  ```

### [N2] CORS Defaults to Allow-All When Origins Are Empty — MASVS-NETWORK
- **File:** `internal/http/cors.go:12`
- **Risk:** When `allowedOrigins` is empty (the default if config omits it), `allowAll := true` is set and `Access-Control-Allow-Origin: *` is sent for every cross-origin request. This silently exposes the entire authenticated REST surface to any origin a browser can reach.
- **Composition risk:** Combined with [N7] (no CSRF), this enables cross-site state mutation by any malicious origin once a user has a valid session.
- **Fix:** Fail closed. If the origins list is empty, deny all cross-origin requests, OR add a config validation step that requires at least one explicit origin when running in non-development mode.

### [N3] X-Forwarded-For Spoofing Bypasses Per-IP Rate Limits — MASVS-NETWORK
- **File:** `internal/http/rate_limit.go:29-31`
- **Risk:** The limiter reads `X-Forwarded-For` directly without checking whether the request actually came from a trusted reverse proxy. A direct caller can send `X-Forwarded-For: 1.2.3.4` and have the limiter key off the spoofed value, defeating per-IP throttling for [H5] and [H6].
- **Fix:** Only honor `X-Forwarded-For` if `c.RemoteIP()` matches a configured trusted-proxy CIDR. Otherwise use `c.ClientIP()` with Gin's `(*Engine).SetTrustedProxies` to enforce proxy trust at the framework level.

---

## HIGH

### [N4] Missing HTTP Security Headers — MASVS-NETWORK
- **File:** `internal/http/handler.go` (entire HTTP layer)
- **Risk:** No middleware sets `Strict-Transport-Security`, `X-Frame-Options`, `Content-Security-Policy`, or `X-Content-Type-Options`. Browsers consuming the API (or any embedded UI) are exposed to clickjacking, MIME-sniffing, and HSTS-downgrade attacks.
- **Fix:** Add a security-headers middleware applied at the router root:
  ```go
  c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
  c.Header("X-Frame-Options", "DENY")
  c.Header("X-Content-Type-Options", "nosniff")
  c.Header("Content-Security-Policy", "default-src 'self'")
  c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
  ```

### [N5] WebSocket Origin Validation Compares Lowercased Strings — MASVS-NETWORK
- **File:** `internal/sync/ws_handler.go:40-59`
- **Risk:** Origin allowlisting normalizes via lowercase string compare rather than parsing the URL. Edge cases — scheme case-mismatch (`HTTP://...`), default-port equivalence (`https://x` vs `https://x:443`), or trailing slashes — can permit unexpected origins or reject legitimate ones.
- **Fix:** Parse each configured origin and the inbound `Origin` header with `url.Parse`, then compare scheme + hostname + explicit port components. Reject anything that fails to parse.

### [N6] JWT Secret Presence Validated Per-Request, Not At Startup — MASVS-AUTH / MASVS-CRYPTO
- **File:** `internal/auth/jwt.go:65-66`
- **Risk:** When auth is enabled but `JWTSecret` is empty, the service starts cleanly and only fails on the first authenticated request with `errors.New("missing jwt secret")`. Operators get no startup-time signal. A misconfigured deploy is silently broken until traffic arrives, and any path that sidesteps secret validation accepts effectively unsigned tokens.
- **Fix:** In `NewJWTService` (or config validation), refuse to start when `cfg.Auth.Enabled == true && strings.TrimSpace(cfg.Auth.JWTSecret) == ""`. Return a fatal startup error.

### [N7] No CSRF Protection on State-Changing Endpoints — MASVS-AUTH
- **File:** `internal/http/handler.go` (all `POST/PUT/PATCH/DELETE` routes)
- **Risk:** If any browser client uses cookie-based session auth, every mutation endpoint is exposed to cross-site request forgery. Combined with [N2]'s permissive CORS default, a malicious page can issue authenticated mutations from any origin.
- **Fix:** For browser flows, require a CSRF token (double-submit cookie or synchronizer pattern) on unsafe verbs. For pure bearer-token clients, document the requirement and ensure the auth middleware rejects credentials sent via cookies on mutation routes.

---

## MEDIUM

### [N8] Sync Hub History Grows Unbounded — MASVS-CODE
- **File:** `internal/sync/hub.go`
- **Risk:** The hub's event history slice has no upper bound visible in the structure or append paths. On a long-running server, memory consumption grows linearly with event volume, eventually leading to OOM.
- **Fix:** Introduce `historyLimit` (e.g., 1024 events, matching the existing replay window) with FIFO eviction or a ring buffer. Document the trade-off with replay coverage.

### [N9] No Write Deadline on WebSocket Writes — MASVS-CODE
- **File:** `internal/sync/ws_handler.go` (writeLoop, ~lines 130-180)
- **Risk:** `conn.WriteMessage` has no `SetWriteDeadline`. A client whose TCP read side hangs (slow network, malicious slowloris-style consumer) blocks the write goroutine indefinitely, leaking goroutines and pinning the per-connection buffer.
- **Fix:** Before each write call, `conn.SetWriteDeadline(time.Now().Add(5 * time.Second))`. On timeout, close the connection.

### [N10] Silent Failure on Budget State Persistence — MASVS-STORAGE
- **File:** `internal/service/deep_processor.go:681-694` (`saveBudgetState`)
- **Risk:** Errors writing the budget state file are swallowed. If the disk fills, permissions break, or the path is unwritable, the in-memory budget continues to advance while the on-disk record stagnates — defeating the durability goal of [M6]. A subsequent restart can reload a stale snapshot and reset the daily quota silently.
- **Fix:** Log the error with `slog.Error` and increment a `budget_persist_failures_total` metric. Optionally fail closed (skip further deep processing) until persistence recovers.

### [N13] Content-Type Not Validated Before JSON Bind — MASVS-CODE
- **File:** `internal/http/handler.go` (all `ShouldBindJSON` call sites)
- **Risk:** Gin parses JSON bodies regardless of `Content-Type`. A client sending `Content-Type: text/plain` with a JSON body still succeeds, which can confuse intermediate proxies/CDNs about cacheability and create content-confusion conditions.
- **Fix:** Reject early when `c.Request.Header.Get("Content-Type")` does not start with `application/json` for routes that bind JSON.

---

## LOW

### [N11] JWT Issuer/Audience Not Enforced End-to-End — MASVS-AUTH
- **File:** `internal/auth/jwt.go:88-109` (`IssueToken`)
- **Risk:** If multiple services share the same JWT secret with different intended audiences, tokens issued for service A are accepted by service B because audience validation is not enforced. Cross-service token reuse becomes possible.
- **Fix:** Require and validate an `aud` claim in `IssueToken` and `ValidateToken`. Reject tokens whose audience does not match this service's identifier.

### [N12] WebSocket Inbound Rate-Limit Closures Are Not Logged — MASVS-CODE
- **File:** `internal/sync/ws_handler.go:256-270` (readLoop error handling)
- **Risk:** When [H7]'s inbound rate limit triggers, the connection is closed but no structured log is emitted. Incident response loses the ability to attribute abuse or detect coordinated probing.
- **Fix:** Before closing, log with `slog.Warn("sync websocket inbound rate limit exceeded", "client_ip", ip, "subject", subject)`.

---

## Severity × MASVS Distribution

| Severity | CODE | NETWORK | AUTH | STORAGE | CRYPTO |
|----------|------|---------|------|---------|--------|
| Critical | 1 (N1) | 2 (N2, N3) | — | — | — |
| High     | — | 2 (N4, N5) | 2 (N6, N7) | — | 1 (N6 overlap) |
| Medium   | 3 (N8, N9, N13) | — | — | 1 (N10) | — |
| Low      | 1 (N12) | — | 1 (N11) | — | — |

Total: 13 findings.

---

## Recommended Remediation Order

1. **[N3] X-Forwarded-For trust** — directly weakens an already-shipped control (H5).
2. **[N2] CORS fail-closed** — expands the surface that H2/H4 were meant to protect.
3. **[N1] Type-assertion crash** — one-line fix, removes a panic vector.
4. **[N6] JWT secret startup validation** — operational hardening for the auth wiring landed in H2.
5. **[N7] CSRF** — coupled with N2; tackle together.
6. **[N4] Security headers** — broad win, low blast radius.
7. **[N5] WS origin parsing** — compose with N2.
8. **[N8]–[N13]** — schedule into a Check 2 closeout milestone.

## Composition With Check 1

Several Check 1 fixes are weakened or made conditional by these findings:

- [H5] HTTP rate limiting is bypassable via [N3] header spoofing.
- [H6] WS connection cap is bypassable via [N3] header spoofing on the same key derivation.
- [H2] auth wiring assumes a non-empty JWT secret; [N6] removes that guarantee at startup.
- [M6] durable budget persistence is silently defeatable by [N10].

These compositions argue for treating N1-N7 as part of the Check 1 release gate rather than a separate phase.