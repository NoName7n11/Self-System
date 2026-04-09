.PHONY: dev dev-setup test lint build clean run

dev-setup:
	go mod tidy

dev: run

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	go fmt ./...

build:
	go build -o bin/self-systems ./cmd/server

clean:
	go clean ./...
