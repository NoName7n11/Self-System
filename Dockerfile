# syntax=docker/dockerfile:1.7

FROM golang:1.22-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/self-systems ./cmd/server

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S app && adduser -S app -G app

COPY --from=builder /out/self-systems /app/self-systems
COPY config /app/config
COPY data /app/data

ENV SS_APP_HOST=0.0.0.0 \
    SS_APP_PORT=8080 \
    SS_DATABASE_TYPE=postgres

EXPOSE 8080

USER app
ENTRYPOINT ["/app/self-systems"]
