.PHONY: dev dev-setup run test integration-test distributed-test distributed-report verify-sync-runtime lint build clean docker-up docker-up-postgres docker-down vps-up vps-down vps-logs ci ci-distributed test-postgres event-sourcing-test rollback-drill backfill-bench extraction-test ai-pipeline-test wails-dev wails-build-windows wails-build-linux gbus-train

POSTGRES_TEST_DSN ?= postgres://selfsystems:selfsystems@127.0.0.1:5432/self_systems?sslmode=disable
SYNC_RUNTIME_BASE_URL ?= http://127.0.0.1:8080
SYNC_RUNTIME_WS_PATH ?= /api/v1/sync/ws
SYNC_RUNTIME_TIMEOUT ?= 10
SYNC_RUNTIME_REPORT ?= artifacts/sync-runtime-reachability.json
DISTRIBUTED_GATE_JSON ?= artifacts/distributed-sync-go-test.json
DISTRIBUTED_GATE_REPORT ?= artifacts/distributed-sync-report.md

dev-setup:
	go mod tidy

dev: docker-up run

docker-up:
	docker compose up -d

docker-down:
	docker compose down

vps-up:
	docker compose -f docker-compose.yml -f docker-compose.vps.yml up -d --build

vps-down:
	docker compose -f docker-compose.yml -f docker-compose.vps.yml down

vps-logs:
	docker compose -f docker-compose.yml -f docker-compose.vps.yml logs -f nginx api

run:
	go run ./cmd/server

test:
	go test ./...

integration-test:
	go test ./test/integration/...

distributed-test:
	go test ./internal/sync ./test/integration -run "Sync|Offline|Replay"

distributed-report:
	@mkdir -p artifacts
	@set +e; \
	go test -json ./internal/sync ./test/integration -run "Sync|Offline|Replay" > "$(DISTRIBUTED_GATE_JSON)"; \
	EXIT_CODE=$$?; \
	go run ./scripts/generate_distributed_gate_report -input "$(DISTRIBUTED_GATE_JSON)" -output "$(DISTRIBUTED_GATE_REPORT)"; \
	exit $$EXIT_CODE

verify-sync-runtime:
	@mkdir -p artifacts
	go run ./scripts/verify_sync_runtime -base-url "$(SYNC_RUNTIME_BASE_URL)" -websocket-path "$(SYNC_RUNTIME_WS_PATH)" -timeout-seconds "$(SYNC_RUNTIME_TIMEOUT)" -report-file "$(SYNC_RUNTIME_REPORT)"

lint:
	go fmt ./...

ci: lint test

ci-distributed: lint test distributed-test test-postgres

# ── Event sourcing safety gates (WS7) ────────────────────────────────────────

# Run property-based tests and sync reconnect tests.
event-sourcing-test:
	go test -run 'TestEventVersionMonotonicity|TestProjectionDeterminism|FuzzEventVersion|FuzzProjection|TestReconnect|FuzzReconnect' \
		./internal/eventstore/... ./internal/sync/... -v

# Run the rollback drill: flag ON → flag OFF → parity check.
rollback-drill:
	go run ./scripts/rollback_drill

# Run the backfill benchmark with 10 iterations (fast smoke gate).
# For the full 100K budget test: go test -bench BenchmarkBackfill100K -benchtime=1x ./internal/migration/...
backfill-bench:
	go test -run '^$$' -bench BenchmarkBackfill -benchtime=10x ./internal/migration/... -v

# Run the AI pipeline integration tests (uses mock provider — no real API calls).
ai-pipeline-test:
	go test -run 'TestAIPipeline|TestClassifier|TestEmbedding|TestManager_Enrich|TestParseEnrichment' \
		./internal/ai/... ./internal/service/... ./test/integration/... -v -timeout 5m

# Run the content extraction unit + integration tests.
extraction-test:
	go test ./internal/extractor/... ./test/integration/... -run 'TestURL|TestPDF|TestImage|TestEvent|TestDeepProcessor' -v -timeout 5m

docker-up-postgres:
	docker compose up -d postgres

test-postgres:
	SS_POSTGRES_TEST_DSN="$(POSTGRES_TEST_DSN)" go test ./internal/repository/postgres -run Integration

build:
	go build -o bin/self-systems ./cmd/server

# ── Wails desktop build targets ───────────────────────────────────────────────

# Launch the desktop app in dev mode (requires wails CLI: go install github.com/wailsapp/wails/v2/cmd/wails@latest)
wails-dev:
	cd cmd/desktop && wails dev

# Build Windows desktop binary (requires wails CLI and WebView2 runtime)
wails-build-windows:
	cd cmd/desktop && wails build -platform windows/amd64

# Build Linux desktop binary (requires wails CLI and GTK/WebKit dependencies)
wails-build-linux:
	cd cmd/desktop && wails build -platform linux/amd64

# ── GBUS training ─────────────────────────────────────────────────────────────

gbus-train:
	go run ./scripts/gbus_train -db ./data/self_systems.db -out ./models/gbus/baseline.json

clean:
	go clean ./...
