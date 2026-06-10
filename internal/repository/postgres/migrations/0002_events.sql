CREATE TABLE IF NOT EXISTS events (
    sequence BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    aggregate_id TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_version INT NOT NULL,
    payload JSONB NOT NULL,
    payload_schema_version INT NOT NULL DEFAULT 1,
    occurred_at TIMESTAMPTZ NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    device_id TEXT,
    actor_id TEXT,
    redacted BOOLEAN NOT NULL DEFAULT FALSE,
    correlation_id UUID,
    CONSTRAINT events_aggregate_version_unique UNIQUE (aggregate_id, event_version),
    CONSTRAINT events_payload_valid CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_events_type_time ON events(aggregate_type, event_type, recorded_at);

CREATE TABLE IF NOT EXISTS projection_snapshots (
    aggregate_id TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    snapshot_version INT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aggregate_id, snapshot_version),
    CONSTRAINT projection_snapshots_payload_valid CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_projection_snapshots_type_version ON projection_snapshots(aggregate_type, snapshot_version);
