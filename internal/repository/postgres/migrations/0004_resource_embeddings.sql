-- Change 7 WS2: resource embedding vectors for semantic search (pure-Go brute force).
CREATE TABLE IF NOT EXISTS resource_embeddings (
	resource_id TEXT PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
	vector BYTEA NOT NULL,
	model_version TEXT NOT NULL,
	dim INTEGER NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_resource_embeddings_model ON resource_embeddings(model_version);
