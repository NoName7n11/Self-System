# Deployment Runbook

This runbook documents a practical deployment and verification flow for the current master branch runtime.

## 1. Prerequisites

- Go 1.22+
- Docker and Docker Compose
- GNU Make (optional)

## 2. Environment Setup

1. Copy the environment template.
   - PowerShell: Copy-Item .env.example .env
   - Bash: cp .env.example .env
2. Update key overrides as needed.
   - SS_APP_HOST
   - SS_APP_PORT
   - SS_SYNC_ENABLED
   - SS_AUTH_ENABLED

Configuration precedence:

1. config/config.default.yml
2. .env
3. Environment variables (SS_ prefix)

## 3. Local Runtime Bring-Up

Start local infrastructure services:

- make docker-up

Start API server:

- make run

If not using make:

- docker compose up -d
- go run ./cmd/server

## 4. Runtime Health Checks

Validate service endpoints:

- GET /health
- GET /api/v1/sync/health

Examples:

- curl http://127.0.0.1:8080/health
- curl http://127.0.0.1:8080/api/v1/sync/health

## 5. Reachability Verification Script

Run verifier directly against local or deployed runtime:

- go run ./scripts/verify_sync_runtime -base-url http://127.0.0.1:8080 -report-file artifacts/sync-runtime-reachability.json

Optional auth:

- Set SS_SYNC_RUNTIME_BEARER_TOKEN in environment, or
- pass -bearer-token explicitly.

## 6. GitHub Actions Reachability Workflow

Workflow path:

- .github/workflows/sync-runtime-reachability.yml

Dispatch inputs:

- base_url
- websocket_path
- timeout_seconds
- auth_mode: none | manual_input | github_oidc
- oidc_audience (used with github_oidc)
- bearer_token (used with manual_input)

Recommended auth mode:

- github_oidc for secure non-manual token minting in Actions runtime.

Workflow artifacts:

- sync-runtime-reachability/sync-runtime-reachability.json
- sync-runtime-reachability/sync-runtime-reachability-stdout.json

## 7. Validation and Quality Gates

- make test
- make integration-test
- make ci

## 8. Rollback Notes

- Keep a known-good server binary available when deploying manually.
- If deployment fails, restore previous binary and re-run health and reachability checks.
