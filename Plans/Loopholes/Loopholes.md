 # Loopholes.md — Security Audit: internal/                                                                                                                                                                                                                                   
  ## CRITICAL                                                                                                                            
### 1. API Key Exposed in URL Query Parameter (`internal/ai/gemini_provider.go:50`)                                                     
  **File:** `internal/ai/gemini_provider.go`                                                                                              
  **Line:** 50                                                                                                                            
  ```go                                                                                                                                   
  endpoint := ... + ":generateContent?key=" + url.QueryEscape(p.settings APIKey)
  
  ```                
  Risk: The Gemini API key is appended as a URL query parameter. This means it appears in:                                                
  - HTTP server access logs
  - Reverse proxy logs                      
  - Browser history (if called from a client)
  - Network traces                                                                                                                        
   
  Fix: Use the x-goog-api-key request header instead, same pattern as Anthropic/OpenAI providers.                                         
                                                                                                                                        
  ---
  HIGH

### 2. No Rate Limiting on HTTP Mutation Endpoints (internal/http/handler.go)

  Risk: POST /api/v1/resources, POST /api/v1/todos, POST /api/v1/reminders, POST /api/v1/categories and all PUT/DELETE/PATCH endpoints
  have zero rate limiting. An authenticated client can hammer the API, drain AI token budgets, and cause resource exhaustion.
  Fix: Add a per-IP or per-subject rate limiter middleware before the api group.

  ---
  3. No Rate Limiting on WebSocket Connections (internal/sync/ws_handler.go)

  Risk: Any origin-allowed client can open an unlimited number of WebSocket connections. Each connection creates a goroutine and
  subscribes a buffered channel to the Hub. This is a goroutine-leak / memory exhaustion vector.
  Fix: Track connected clients per IP/subject and enforce a max-connections-per-client limit before upgrading.

  ---
  4. pagination() Has No Upper Bound on limit (internal/http/handler.go:925-934)

  func pagination(c *gin.Context) (int, int) {
      limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
      ...
      return limit, offset
  }
  Risk: limit is accepted from the query string with no maximum cap. A caller can pass limit=10000000 and force the database to return a
  massive result set, causing memory spikes.
  Note: parseBoundedInt (used only for search endpoints) correctly caps at 100, but the general pagination() helper does not.
  Fix: Add if limit > 200 { limit = 200 } (or similar) inside pagination().

  ---
  5. SemanticSearch Always Fetches 500 Records (internal/service/resource_service.go:258)

  candidates, err := s.resources.List(ctx, 500, 0)
  Risk: Every semantic search call unconditionally loads 500 full resource records into memory regardless of the user's requested limit.
  As the dataset grows this becomes a DoS amplification vector — a single GET /resources/semantic-search?q=a&limit=1 loads 500 rows.
  Fix: Either push scoring into the database with a proper FTS or vector index, or at least make the candidate pool configurable and
  bounded.

  ---
  6. isValidationErrorMessage Uses String Matching to Classify Errors (internal/http/handler.go:965-981)

  func isValidationErrorMessage(message string) bool {
      hints := []string{"required", "invalid", "not found", "must be", "empty"}
      for _, hint := range hints {
          if strings.Contains(lower, hint) { return true }
      }
      return false
  }
  Risk: Internal database errors or third-party errors whose text happens to contain "invalid" or "not found" will be classified as 400
  Bad Request instead of 500 Internal Server Error. This can leak internal error messages to clients and masks real bugs. It also produces
   incorrect HTTP semantics.
  Fix: Use typed errors (e.g., a ValidationError interface or sentinel values) instead of string sniffing.

  ---
  7. respondOperationError Leaks Internal Error Messages to Clients (internal/http/handler.go:956)

  respondError(c, http.StatusInternalServerError, message)
  Risk: When a DB error, network error, or AI provider error is not caught by isValidationErrorMessage, its raw .Error() string is sent
  directly to the client in the JSON body. This can expose:
  - Database table/column names
  - Internal service topology
  - File paths (on SQLite errors)

  Fix: Log the full error server-side; return a generic "internal server error" message to the client for 5xx responses.

  ---
  8. executeChatCommand Has No Auth Guard in Handler Registration (internal/http/handler.go:852)

  api.POST("/chat/commands", h.executeChatCommand)
  Risk: chat/commands is registered on the unauthenticated api group — the same group as all other CRUD routes. Whether auth is applied
  depends entirely on whether a JWT middleware was attached to api at the router level (not visible in handler.go). If the middleware is
  ever removed/misconfigured, this endpoint allows arbitrary data mutations (create/delete resources, categories, todos, reminders) with
  no auth. The chat service also exposes a graph command that leaks the entire data graph.
  Fix: Apply auth middleware explicitly at the router group level and document this dependency clearly.

  ---
  9. WebSocket Read Loop Discards All Incoming Messages (internal/sync/ws_handler.go:188)

  for {
      if _, _, err := conn.ReadMessage(); err != nil {
          return
      }
  }
  Risk: The read loop reads and silently discards every message sent by the client. While this is intentional (server-push only), it means
   there is no validation or size check on inbound WS frames beyond the 1024-byte SetReadLimit. A client can send 1024-byte messages at
  full rate indefinitely with no backpressure. Combined with the lack of per-connection rate limiting, this is a low-cost keep-alive abuse
   vector.

  ---
  MEDIUM

  10. BuildEventPayload Allows Caller to Overwrite entity_id / event_source (internal/sync/protocol.go:84)

  enriched := make(map[string]any)
  for key, value := range payload {
      enriched[key] = value   // copies all caller-supplied keys first
  }
  enriched[PayloadKeyEntityID] = ...  // then overwrites
  Risk: The overwrite order is correct for entity_id and event_source, but any other arbitrary key in payload is passed through without
  sanitisation into the broadcast event and stored in the replay queue. Clients submitting to POST /sync/events or POST
  /sync/offline-queue/enqueue can inject arbitrary keys into every subscriber's event stream.
  Fix: Whitelist the keys that are allowed in inbound payloads.

  ---
  11. ReplaySince Trims From the End Instead of the Beginning (internal/sync/hub.go:136)

  if limit > 0 && len(result) > limit {
      result = result[len(result)-limit:]
  }
  Risk: When a reconnecting client requests the last N events via since_sequence, it receives the most recent N events — which means if a
  large burst happened while it was offline, older events in the window are silently dropped. The client has no way to distinguish "no
  events" from "events were dropped". This creates silent data loss on reconnect for slow clients.
  Fix: Either return the oldest N events (slice from front) or return a truncated: true flag so the client knows to do a full resync.

  ---
  12. inferEntityID Falls Back to Arbitrary id Field (internal/sync/offline_replay_manager.go:160)

  value, exists := payload["id"]
  if !exists { return "" }
  return strings.TrimSpace(fmt.Sprint(value))
  Risk: If entity_id is absent, any id field in the payload (e.g., a nested object's id) is used as the entity identity for conflict
  resolution and deduplication. A malicious or buggy client can cause mutations for entity A to be batched and conflict-resolved against
  entity B by crafting a payload with a mismatched id.
  Fix: Remove the id fallback; require entity_id explicitly, which ValidateIncomingEvent already enforces for most event types anyway.

  ---
  13. auth/health Leaks Auth Configuration State (internal/sync/routes.go:54-61)

  c.JSON(http.StatusOK, gin.H{
      "auth_enabled":               cfg.Auth.Enabled,
      "google_oauth_configured":    strings.TrimSpace(cfg.Auth.GoogleClientID) != "" && ...,
      "jwt_signing_key_configured": strings.TrimSpace(cfg.Auth.JWTSecret) != "",
  })
  Risk: This endpoint is unauthenticated and reveals whether auth is enabled, whether OAuth is configured, and whether a JWT signing key
  is set. An attacker can use this to detect misconfigurations (e.g., jwt_signing_key_configured: false) before attempting to exploit
  them.
  Fix: Remove boolean flags from the public response or gate the endpoint behind auth.

  ---
  14. sync/health Leaks Hub Statistics Unauthenticated (internal/sync/routes.go:76-87)

  Risk: Connected client count, history depth, published/dropped totals, and websocket path are all returned to any caller with no auth.
  The websocket path in particular should not be broadcast.
  Fix: Separate an internal health check (for infrastructure monitoring, gated by auth or network policy) from the public liveness probe.

  ---
  15. deepComplexityScore Uses URL Content for Routing Decisions (internal/service/deep_processor.go:443)

  if strings.Contains(urlLower, "github") || strings.Contains(urlLower, "docs") || strings.Contains(urlLower, "research") {
      score += 2
  }
  Risk: A user can submit a URL like https://evil.com/github/docs/research/... to deliberately bump the complexity score and force
  expensive high-cost AI model routing, consuming the daily token budget faster.
  Fix: Route on parsed hostname only, not arbitrary URL substrings.

  ---
  16. Daily Token Budget Uses In-Memory State Only (internal/service/deep_processor.go:371)

  Risk: budgetUsed and budgetDate are in-memory fields. On process restart the budget resets to zero, allowing the daily limit to be
  exceeded by restarting the process (intentionally or via crash loops). With multiple workers or replicas, each instance has an
  independent budget.
  Fix: Persist the daily budget to the database or a shared cache.

  ---
  LOW / INFORMATIONAL

  17. parsePipePayload Accepts Arbitrary Key-Value Injection via |key=value (internal/service/chat_service.go:718)

  Risk: The pipe-delimited chat command format (resource: https://x.com | category=foo | summary=bar) parses all key=value segments into a
   map. There is no whitelist of valid keys. Unknown keys are silently ignored, but a future code change that reads an unvalidated key
  from the map would introduce an injection path without any change to the parsing code.
  Fix: Validate keys against an explicit allowlist per command type.

  18. parseCommandID Takes Only the First Whitespace-Delimited Token

  Risk: get resource abc def will silently use abc as the ID and ignore def, giving no error to the user. This is a UX/logic issue that
  could mask typos in IDs.

  19. No Maximum Payload Size on REST Endpoints

  Risk: Gin's default body limit is 32MB. Endpoints like POST /resources that accept summary fields have no application-level size limit
  on text fields. A large summary (e.g., 10MB of text) is stored in the database and later loaded into memory for semantic scoring.
  Fix: Add a c.Request.Body = http.MaxBytesReader(...) or Gin's body limit middleware.

  20. lastError in DeepProcessor is Overwritten, Not Appended

  Risk: Only the single most recent error is retained. Burst failures (e.g., 50 AI errors in one minute) are collapsed to one message,
  making post-incident debugging harder.
  ```

❯ /btw <If you do find any thing, list them in a markdownfile as "Loopholes.md" 
  ⎿  Branched conversation "btw: &lt;If you do find any thing, list them in a markdownfile as "Loopholes.md"". You are now in the branch.
     Use /resume c7099426-0d25-4e47-b291-f300f5b670ad to return to the original.

● I now have a complete picture of the codebase. Here is the full, line-referenced security audit:
                                                      
  ---
  Security Audit — internal/                                                                                                              
   
  CRITICAL                                                                                                                                
                                                                                                                                        
  ---                                                                                                                                     
  [C1] Unbounded pagination — DoS via memory exhaustion                                                                                   
  - internal/http/handler.go:925-935 — pagination() clamps limit < 0 to 50 but has no upper bound. An unauthenticated caller can send
  ?limit=10000000 to GET /api/v1/resources, GET /api/v1/todos, GET /api/v1/reminders and force the server to attempt loading millions of
  rows.
  - internal/sync/routes.go:200-201 — GET /sync/conflicts reads limit from query params and checks <= 0 but has no ceiling. ?limit=999999
  is passed verbatim to the DB.
  - internal/sync/routes.go:168-184 — POST /sync/offline-queue/replay accepts {"limit": N} with zero validation. Any number is forwarded
  to store.ListPending(ctx, limit) → SQLite LIMIT ?.

  Fix: Add an upper bound (e.g., 200) in all three places, matching the pattern already used in parseBoundedInt.

  ---
  [C2] Raw internal errors surfaced to HTTP clients
  internal/http/handler.go:951-963 — respondOperationError calls err.Error() and returns it in the HTTP body for both 400 and 500
  responses. Database error strings (table names, constraint names, file paths) reach clients in 500 paths.

  // line 956-962
  message := err.Error()        // raw db/internal error
  ...
  respondError(c, http.StatusInternalServerError, message)   // sent to client

  Fix: Log the full error server-side; return a generic "internal error" string to the client.

  ---
  HIGH

  ---
  [H1] Gemini API key exposed in URL
  internal/ai/gemini_provider.go:48-50:
  endpoint := strings.TrimRight(p.settings.BaseURL, "/") +
      "/v1beta/models/" + url.PathEscape(p.settings.Model) +
      ":generateContent?key=" + url.QueryEscape(p.settings.APIKey)
  The API key lands in the URL, which appears in: server access logs, reverse-proxy logs, CDN logs, and OS-level process argument
  inspection. Both Anthropic and OpenAI providers correctly use Authorization: Bearer headers.

  Fix: Use an x-goog-api-key request header instead of a query parameter (Google's API supports this).

  ---
  [H2] Auth middleware is never applied to the main REST routes
  internal/http/handler.go:88-125 — RegisterRoutes creates the api group and registers all CRUD endpoints for resources, categories,
  todos, reminders, graph, chat commands, and deep processing — without attaching any auth middleware. Auth is only wired in the sync
  bootstrap (internal/sync/routes.go:90). Whether the caller applies auth externally is invisible from this file and untestable.

  This means:
  - POST /api/v1/resources, DELETE /api/v1/resources/:id, POST /api/v1/chat/commands, POST /api/v1/processing/deep/reprocess/:id are all
  unauthenticated unless explicitly protected at the router setup level.

  Fix: Accept an optional authMiddleware gin.HandlerFunc in NewHandlerWithOptions (or RegisterRoutes) and apply it to the api group.

  ---
  [H3] Auth-config disclosure on unauthenticated endpoint
  internal/sync/routes.go:54-61 — GET /api/v1/auth/health is open to everyone and reveals:

  {
    "auth_enabled": false,
    "google_oauth_configured": false,
    "jwt_signing_key_configured": true
  }

  An attacker can query this to know exactly what authentication bypass to attempt before hitting protected routes.

  Fix: Remove the boolean flags or require authentication to access this endpoint.

  ---
  [H4] No input length limits on string fields
  No maxLength validation exists anywhere in the handler or service layer for title, summary, message, details, description. An attacker
  can POST a 100 MB summary field which will be stored in the DB and returned in list/get responses, causing storage inflation and
  large-response DoS.

  Fix: Add max-length checks (e.g., title ≤ 512, summary/message ≤ 10 000 chars) in the service layer Create/Update methods.

  ---
  [H5] No rate limiting on any REST endpoint
  The entire API has no rate-limiting middleware. Combined with [H2], unauthenticated callers can:
  - Exhaust AI API quota by spamming POST /api/v1/resources (each triggers AI classification + deep processing).
  - Trigger expensive DB queries via GET /api/v1/resources/semantic-search (fetches 500 candidates every call, resource_service.go:258).
  - Overflow the deep-processing queue by calling POST /api/v1/processing/deep/reprocess/:id in a tight loop.

  ---
  MEDIUM

  ---
  [M1] System metrics exposed on unauthenticated sync health endpoint
  internal/sync/routes.go:76-87 — GET /api/v1/sync/health returns the full hub stats and observability.Snapshot() (auth failure counts,
  drop rates, replay metrics) with no auth. This contradicts GET /api/v1/sync/metrics (line 91) which requires auth.

  Fix: Remove or gate the metrics fields behind auth; keep only { "status": "ok", "sync_enabled": true }.

  ---
  [M2] Daily token budget is in-memory; resets on restart
  internal/service/deep_processor.go:371-395 — budgetDate/budgetUsed are struct fields (not persisted). A service restart resets the daily
   AI token quota. An attacker who can trigger restarts (or crashes) can repeatedly reset the budget and drive up AI API costs.

  Fix: Persist the token budget (and date) to the database so it survives restarts.

  ---
  [M3] LIKE wildcard passthrough in search
  internal/repository/sqlite/resource_repository.go:123:
  pattern := "%" + strings.ToLower(query) + "%"
  The user query is embedded verbatim inside a LIKE pattern. Searching with _ matches any single character; searching % matches
  everything. Users can use ?q=% to retrieve all resources, bypassing the intent of search. This is not SQL injection (parameterized) but
  is a logic bypass.

  Fix: Escape % and _ in the user query before wrapping with wildcards:
  escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(query)
  pattern := "%" + escaped + "%"
  // Use: LIKE ? ESCAPE '\'

  ---
  [M4] reprocess endpoint triggers paid AI calls without auth
  internal/http/handler.go:190-217 — POST /api/v1/processing/deep/reprocess/:id queues an AI processing task for any resource ID. Because
  [H2] means auth is not applied here, an unauthenticated caller can enumerate UUIDs (or guess IDs from the sync.* WebSocket events) and
  trigger unbounded AI API calls draining the token budget.

  ---
  [M5] Operation ID and payload fields have no length bounds in offline queue
  internal/sync/routes.go:128-145 — operation_id, type, and all payload keys/values are stored in SQLite with no length validation. A
  client can enqueue a mutation with a 10 MB operation_id, inflating the sync_offline_queue table.

  ---
  LOW

  ---
  [L1] [deep-processing] marker in summary stripped without notice
  internal/service/deep_processor.go:397-419 — buildDeepSummary silently removes any line containing [deep-processing] from user-provided
  summaries. A user who legitimately writes [deep-processing] in their resource notes will silently lose that content.

  ---
  [L2] Hub subscriber channel buffer (16) can silently drop events
  internal/sync/ws_handler.go:89 — h.hub.Subscribe(16) creates a 16-event buffer per client. If a client falls behind (e.g., slow
  network), events are silently dropped (hub.go:91-96). Clients are never notified of dropped events, so they can miss mutations without
  knowing. The since_sequence reconnect mechanism mitigates this but only within the 1024-event hub history.

  ---
  [L3] acquireThroughputSlot blocks worker goroutines under load
  internal/service/deep_processor.go:336-369 — When the throughput window is saturated, worker goroutines sleep in a timer loop. With
  WorkerCount > 1, all workers can be blocked simultaneously while holding no resources, but the goroutines are not cleaned up if the
  context is cancelled during the sleep path (the timer.Stop() call is there, but the channel draining is correct).

  ---
  Summary Table

  ┌─────┬──────────┬──────────────────────────────────────────────┬─────────────────────────────────────────────────────┐
  │ ID  │ Severity │                     File                     │                        Issue                        │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ C1  │ Critical │ http/handler.go:925, sync/routes.go:184,200  │ No upper bound on pagination/replay/conflict limits │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ C2  │ Critical │ http/handler.go:956-962                      │ Raw err.Error() returned to HTTP clients            │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ H1  │ High     │ ai/gemini_provider.go:48-50                  │ API key in URL query param (log exposure)           │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ H2  │ High     │ http/handler.go:88-125                       │ Auth middleware never applied to REST routes        │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ H3  │ High     │ sync/routes.go:54-61                         │ Auth config disclosed on public endpoint            │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ H4  │ High     │ All handlers/services                        │ No string field length limits                       │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ H5  │ High     │ Entire API                                   │ No rate limiting                                    │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ M1  │ Medium   │ sync/routes.go:76-87                         │ System metrics on unauthenticated endpoint          │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ M2  │ Medium   │ service/deep_processor.go:371-395            │ In-memory token budget resets on restart            │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ M3  │ Medium   │ repository/sqlite/resource_repository.go:123 │ LIKE wildcard passthrough                           │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ M4  │ Medium   │ http/handler.go:190-217                      │ Unauthenticated reprocess triggers AI spend         │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ M5  │ Medium   │ sync/routes.go:128-145                       │ No length limit on enqueued operation payloads      │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ L1  │ Low      │ service/deep_processor.go:405-409            │ User content silently stripped by marker            │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ L2  │ Low      │ sync/ws_handler.go:89                        │ Silent event drop, no client notification           │
  ├─────┼──────────┼──────────────────────────────────────────────┼─────────────────────────────────────────────────────┤
  │ L3  │ Low      │ service/deep_processor.go:336-369            │ Goroutine blocking under throughput saturation      │
  └─────┴──────────┴──────────────────────────────────────────────┴─────────────────────────────────────────────────────┘

  The most impactful items to fix in order: C1 (one-line fixes, immediate DoS protection), C2 (stop leaking internals), H2 (auth must be
  wired in), H1 (Gemini key header migration), H4+H5 (input limits and rate limiting).