.PHONY: dev dev-setup run test integration-test distributed-test distributed-report verify-sync-runtime lint build clean docker-up docker-up-postgres docker-down vps-up vps-down vps-logs ci ci-distributed test-postgres

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

docker-up-postgres:
	docker compose up -d postgres

test-postgres:
	SS_POSTGRES_TEST_DSN="$(POSTGRES_TEST_DSN)" go test ./internal/repository/postgres -run Integration

build:
	go build -o bin/self-systems ./cmd/server

clean:
	go clean ./...
