.PHONY: dev dev-setup run test integration-test lint build clean docker-up docker-down ci

dev-setup:
	go mod tidy

dev: docker-up run

docker-up:
	docker compose up -d

docker-down:
	docker compose down

run:
	go run ./cmd/server

test:
	go test ./...

integration-test:
	go test ./test/integration/...

lint:
	go fmt ./...

ci: lint test

build:
	go build -o bin/self-systems ./cmd/server

clean:
	go clean ./...
