package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
)

const schema = `
CREATE TABLE IF NOT EXISTS categories (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT NOT NULL DEFAULT '',
	source TEXT NOT NULL CHECK(source IN ('auto','manual')),
	accept_count INTEGER NOT NULL DEFAULT 0,
	override_count INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS resources (
	id TEXT PRIMARY KEY,
	url TEXT NOT NULL UNIQUE,
	host TEXT NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT '',
	category_id TEXT NOT NULL,
	user_override INTEGER NOT NULL DEFAULT 0,
	extracted_data TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY(category_id) REFERENCES categories(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_resources_category_id ON resources(category_id);
CREATE INDEX IF NOT EXISTS idx_resources_created_at ON resources(created_at);

CREATE TABLE IF NOT EXISTS todos (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	details TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL CHECK(status IN ('open','in_progress','done')),
	due_at TEXT,
	resource_id TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY(resource_id) REFERENCES resources(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);

CREATE TABLE IF NOT EXISTS reminders (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	message TEXT NOT NULL DEFAULT '',
	remind_at TEXT NOT NULL,
	status TEXT NOT NULL CHECK(status IN ('scheduled','sent','dismissed')),
	resource_id TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY(resource_id) REFERENCES resources(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reminders_remind_at ON reminders(remind_at);

CREATE TABLE IF NOT EXISTS chat_events (
	id TEXT PRIMARY KEY,
	message TEXT NOT NULL,
	result TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
	sequence INTEGER PRIMARY KEY AUTOINCREMENT,
	event_id TEXT NOT NULL UNIQUE,
	aggregate_id TEXT NOT NULL,
	aggregate_type TEXT NOT NULL,
	event_type TEXT NOT NULL,
	event_version INTEGER NOT NULL,
	payload TEXT NOT NULL CHECK (json_valid(payload)),
	payload_schema_version INTEGER NOT NULL DEFAULT 1,
	occurred_at TEXT NOT NULL,
	recorded_at TEXT NOT NULL DEFAULT (datetime('now')),
	device_id TEXT,
	actor_id TEXT,
	redacted INTEGER NOT NULL DEFAULT 0,
	correlation_id TEXT,
	UNIQUE(aggregate_id, event_version)
);

CREATE INDEX IF NOT EXISTS idx_events_type_time ON events(aggregate_type, event_type, recorded_at);

CREATE TABLE IF NOT EXISTS projection_snapshots (
	aggregate_id TEXT NOT NULL,
	aggregate_type TEXT NOT NULL,
	snapshot_version INTEGER NOT NULL,
	payload TEXT NOT NULL CHECK (json_valid(payload)),
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (aggregate_id, snapshot_version)
);

CREATE INDEX IF NOT EXISTS idx_projection_snapshots_type_version ON projection_snapshots(aggregate_type, snapshot_version);

CREATE TABLE IF NOT EXISTS resource_embeddings (
	resource_id TEXT PRIMARY KEY,
	vector BLOB NOT NULL,
	model_version TEXT NOT NULL,
	dim INTEGER NOT NULL,
	created_at TEXT NOT NULL,
	FOREIGN KEY(resource_id) REFERENCES resources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_resource_embeddings_model ON resource_embeddings(model_version);

CREATE TABLE IF NOT EXISTS similar_resources (
	resource_id      TEXT NOT NULL,
	similar_id       TEXT NOT NULL,
	similarity_score REAL NOT NULL,
	created_at       TEXT NOT NULL,
	PRIMARY KEY (resource_id, similar_id),
	FOREIGN KEY(resource_id) REFERENCES resources(id) ON DELETE CASCADE,
	FOREIGN KEY(similar_id)  REFERENCES resources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_similar_resources_resource ON similar_resources(resource_id);

CREATE TABLE IF NOT EXISTS gbus_category_features (
	category_id    TEXT NOT NULL,
	signal_type    TEXT NOT NULL,
	total_weight   REAL NOT NULL DEFAULT 0,
	signal_count   INTEGER NOT NULL DEFAULT 0,
	last_signal_at TEXT NOT NULL,
	PRIMARY KEY (category_id, signal_type)
);

CREATE INDEX IF NOT EXISTS idx_gbus_cat_features_cat ON gbus_category_features(category_id);

CREATE TABLE IF NOT EXISTS gbus_resource_features (
	resource_id    TEXT NOT NULL,
	signal_type    TEXT NOT NULL,
	total_weight   REAL NOT NULL DEFAULT 0,
	signal_count   INTEGER NOT NULL DEFAULT 0,
	last_signal_at TEXT NOT NULL,
	PRIMARY KEY (resource_id, signal_type)
);

CREATE INDEX IF NOT EXISTS idx_gbus_res_features_res ON gbus_resource_features(resource_id);
`

// addColumnMigrations are ALTER TABLE statements applied after the base schema.
// SQLite does not support IF NOT EXISTS on ALTER TABLE, so we ignore the error
// if the column already exists (indicated by "duplicate column" in the message).
var addColumnMigrations = []string{
	`ALTER TABLE resources ADD COLUMN extracted_data TEXT NOT NULL DEFAULT '{}'`,
	`ALTER TABLE resources ADD COLUMN save_count INTEGER NOT NULL DEFAULT 1`,
	`ALTER TABLE resources ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE resources ADD COLUMN archive_reason TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE resources ADD COLUMN archived_at TEXT`,
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	for _, stmt := range addColumnMigrations {
		if _, err := db.Exec(stmt); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("add column migration: %w", err)
			}
		}
	}
	return nil
}
