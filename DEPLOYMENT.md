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
  - `8081` and `9080` for DGraph (local/private use)

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
4. Start runtime stack (PostgreSQL + Redis + DGraph + Go API + NGINX proxy):

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

## 7. Rollback Strategy

- Keep previous backend binary available on disk.
- If deployment fails, restart previous binary and keep infrastructure containers running.
- Review logs and re-run gates before re-attempting promotion.
