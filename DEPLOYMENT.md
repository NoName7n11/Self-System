# Deployment Runbook (Phase 2 Sync Runtime)

This runbook documents a reproducible deployment path for the Phase 2 sync runtime and central PostgreSQL data layer.

## 1. Prerequisites

- Go 1.22+
- Docker and Docker Compose
- GNU Make (optional but recommended)
- Open network ports:
  - `8080` for API
  - `5432` for PostgreSQL (local/private use)
  - `6379` for Redis (local/private use)

## 2. Environment Configuration

1. Copy environment template:
   - PowerShell: `Copy-Item .env.example .env`
   - Bash: `cp .env.example .env`
2. Configure PostgreSQL runtime settings in `.env`:

```dotenv
SS_DATABASE_TYPE=postgres
SS_DATABASE_URL=postgres://selfsystems:selfsystems@127.0.0.1:5432/self_systems?sslmode=disable
SS_AUTH_ENABLED=true
SS_AUTH_JWT_SECRET=<strong-random-secret>
```

3. Keep configuration precedence in mind:
   - `config/config.default.yml`
   - `.env`
   - environment variables (`SS_` prefix)

## 3. Local Deployment (Reproducible Smoke Path)

1. Start infrastructure:

```bash
make docker-up
```

2. Verify PostgreSQL health:

```bash
docker compose ps postgres
```

3. Start API server:

```bash
go run ./cmd/server
```

4. Validate health endpoints from a second terminal:

```bash
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/api/v1/sync/health
```

5. Run distributed quality gates:

```bash
make distributed-test
make distributed-report
make test-postgres
```

6. Generate a local runtime reachability report artifact:

```bash
make verify-sync-runtime SYNC_RUNTIME_BASE_URL=http://127.0.0.1:8080
```

This writes a JSON report to `artifacts/sync-runtime-reachability.json` and returns non-zero if reachability checks fail.

## 4. VPS Deployment (Final Topology)

1. Provision a Linux VPS and install Docker + Compose plugin.
2. Clone repository and create `.env` with production values.
3. Use strong secrets for auth and database credentials.
4. Start runtime stack (PostgreSQL + Redis + Go API + NGINX proxy):

```bash
docker compose -f docker-compose.yml -f docker-compose.vps.yml up -d --build
```

5. Verify service health:

```bash
docker compose -f docker-compose.yml -f docker-compose.vps.yml ps
docker compose -f docker-compose.yml -f docker-compose.vps.yml logs --tail=100 nginx api
```

6. Validate runtime (public proxy endpoint):

```bash
curl http://127.0.0.1/health
curl http://127.0.0.1/api/v1/sync/health
go run ./scripts/verify_sync_runtime -base-url http://127.0.0.1 -report-file artifacts/sync-runtime-reachability.json
```

7. Optionally run the GitHub Actions workflow `.github/workflows/sync-runtime-reachability.yml` for a deployed endpoint.
   - Set workflow input `base_url` to your deployed runtime URL (for example, `https://api.example.com`).
   - For secure non-manual auth, set workflow input `auth_mode=github_oidc` and optionally tune `oidc_audience` to match server-side trust policy.
   - For manual fallback auth, set workflow input `auth_mode=manual_input` and provide `bearer_token`.
   - Download the `sync-runtime-reachability` artifact for JSON verification evidence.

8. If no deployed server exists yet, run `.github/workflows/sync-runtime-local-smoke.yml`.
   - The workflow boots the API inside the job, verifies local reachability, and uploads `sync-runtime-local-smoke` artifacts.
   - Use `artifacts/templates/sync-runtime-reachability.sample.json` as the expected report shape for dashboards/checklists.

## 5. TLS and Ingress Notes

- Final topology uses `docker-compose.vps.yml` and `deploy/nginx/selfsystems.conf` for reverse proxy and websocket upgrades.
- For TLS, terminate HTTPS at NGINX (or upstream load balancer) and forward to `api:8080`.
- Keep `/api/v1/sync/ws` websocket upgrades enabled in proxy configuration.

### 5.1 Optional TLS configuration (NGINX)

If you want NGINX to terminate TLS directly, use the included template at `deploy/nginx/selfsystems-https.conf`.
Use `docker-compose.vps.tls.yml` as an overlay to map 443 and mount certs.

Steps:

1. Place certificates at `deploy/nginx/certs`:

```
deploy/nginx/certs/fullchain.pem
deploy/nginx/certs/privkey.pem
```

2. Use the TLS overlay to:

- Map port 443
- Mount the TLS config and certs

Example:

```yaml
   nginx:
      ports:
         - "${SS_PUBLIC_HTTP_PORT:-80}:80"
         - "${SS_PUBLIC_HTTPS_PORT:-443}:443"
      volumes:
         - ./deploy/nginx/selfsystems-https.conf:/etc/nginx/conf.d/default.conf:ro
         - ./deploy/nginx/certs:/etc/nginx/certs:ro
```

3. Start the TLS-enabled stack:

```
docker compose -f docker-compose.yml -f docker-compose.vps.yml -f docker-compose.vps.tls.yml up -d --build
```

4. Ensure firewall allows port 443.
5. Verify websocket upgrades after TLS:

```
curl https://<host>/health
curl https://<host>/api/v1/sync/health
```

### 5.2 VPS Hardening Checklist

- Restrict PostgreSQL and Redis ports to private networks or localhost.
- Enable firewall rules (allow only 80/443 and SSH from trusted IPs).
- Use strong secrets for `SS_AUTH_JWT_SECRET` and database credentials.
- Back up PostgreSQL regularly (daily or weekly).
- Monitor disk space and container restarts.
- Keep Docker and the host OS updated on a regular cadence.

## 6. Operational Checks

- API health: `GET /health`
- Sync health + counters snapshot: `GET /api/v1/sync/health`
- Auth-protected metrics: `GET /api/v1/sync/metrics` (requires bearer token)
- Postgres integration gate: `make test-postgres`
- Distributed sync/replay gate: `make distributed-test`
- Distributed gate evidence report: `make distributed-report`
- Runtime reachability verification report: `make verify-sync-runtime SYNC_RUNTIME_BASE_URL=<url>`
- VPS stack bring-up: `make vps-up`
- VPS stack shutdown: `make vps-down`
- Ops checklist: `deploy/ops/ops-checklist.md`
- Troubleshooting guide: `deploy/ops/troubleshooting.md`
- Cost-impact validation notes: `deploy/ops/cost-impact.md`

## 7. Rollback Strategy

- Keep previous backend binary available on disk.
- If deployment fails, restart previous binary and keep infrastructure containers running.
- Review logs and re-run gates before re-attempting promotion.
- See `deploy/ops/rollback.md` for a full rollback and Postgres restore playbook.

## 8. Event Sourcing Rollback Procedure

Use this procedure when an event-sourced domain is misbehaving and must be reverted to direct state-table writes.

**Scope:** This affects one or more of the four feature flags — `events_resource_enabled`, `events_category_enabled`, `events_todo_enabled`, `events_reminder_enabled`. Toggle each flag independently.

**Steps:**

1. **Turn the flag off** in config or environment:
   ```
   SS_FEATURES_EVENTS_RESOURCE_ENABLED=false
   ```
   Restart the API server. The service immediately reverts to direct state-table writes for new mutations.

2. **Verify the rollback.** Call a mutation endpoint and confirm that `GET /api/v1/sync/events/health` shows `appends_total` is no longer incrementing for that domain. The existing outbox worker continues draining the events table but will publish no new events for that aggregate type once it catches up.

3. **State-table authority.** The `resources` (or `categories`, `todos`, `reminders`) projection table is now authoritative. Events written prior to rollback are retained in the `events` table for audit but are no longer the source of truth. Do not truncate the events table — it is required for parity checks and potential re-enablement.

4. **Projection drift window.** Any mutations that occurred between the last outbox delivery and the flag flip may have been written to the events table but not yet published. The projection table is always up to date (sync projectors ran in the same transaction), so no projection rebuild is required.

5. **Re-enablement.** To re-enable, set the flag to `true` and restart. Run the parity check (`go run ./cmd/tools parity`) to confirm the projection matches the event log before declaring the domain stable.

**What is NOT affected:**
- The events table itself — append-only, never truncated by this procedure.
- Other domains — flags are independent; rolling back resources does not affect categories.
- The outbox worker — continues running; will simply have no new events to publish for the rolled-back domain.

## 9. Event Sourcing Recovery Procedure (Projection Drift)

Use this procedure when a projection table has drifted from the event log (detected via parity check or data inconsistency report).

**Detect drift:**

```bash
go run ./cmd/tools parity
# Exits 0 = clean, exits 2 = divergences found.
```

The output lists field-level divergences per aggregate ID.

**Recovery steps:**

1. **Identify affected aggregates.** The parity report lists each diverged `aggregate_id` with expected vs actual field values.

2. **Rebuild individual aggregates.** For each affected aggregate, re-apply its event log by calling the relevant service's update/delete path with the values from the events. The parity report shows what the event log says the state should be.

   Automated batch repair is not currently implemented — repair is manual with operator confirmation to avoid unintended overwrites.

3. **Full projection rebuild (nuclear option).** If the number of diverged aggregates is large:
   - Set `events_*_enabled=false` for the affected domain.
   - Truncate the projection table rows that are diverged (do NOT truncate the events table).
   - Re-run the backfill tool to replay all events into the projection table:
     ```bash
     go run ./cmd/tools backfill
     ```
   - Run parity check to confirm clean state.
   - Re-enable the flag.

4. **Monitor after recovery.** Watch `GET /api/v1/sync/events/health` for `occ_retries_total` and `outbox_lag_sequences` to confirm the event system is healthy before declaring recovery complete.

**Event-store health endpoint reference:**

```
GET /api/v1/sync/events/health   (requires bearer token)

{
  "data": {
    "latest_store_sequence":    12345,
    "last_published_sequence":  12340,
    "outbox_lag_sequences":     5,
    "appends_total":            1200,
    "occ_retries_total":        3,
    "projector_apply_count":    1200,
    "projector_avg_latency_ms": 0.4,
    "snapshots_created":        8,
    "redactions_total":         0
  }
}
```

- `outbox_lag_sequences`: how many events the outbox worker has not yet published. Should trend to 0. Sustained high values indicate the outbox worker is stalled.
- `occ_retries_total`: cumulative OCC retries. Occasional retries are normal under concurrent writes; a high rate suggests write contention on a single aggregate.
