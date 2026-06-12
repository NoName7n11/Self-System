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

const schemaMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version    INTEGER PRIMARY KEY,
	name       TEXT NOT NULL,
	applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`

// migration is one ordered, versioned schema change. Versions must be
// sequential and never reordered or reused once released.
type migration struct {
	version int
	name    string
	apply   func(*sql.DB) error
}

var migrations = []migration{
	{
		version: 1,
		name:    "base_schema",
		apply: func(db *sql.DB) error {
			if _, err := db.Exec(schema); err != nil {
				return fmt.Errorf("run base schema: %w", err)
			}
			return nil
		},
	},
	{
		version: 2,
		name:    "resource_columns",
		apply: func(db *sql.DB) error {
			for _, stmt := range addColumnMigrations {
				if _, err := db.Exec(stmt); err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return fmt.Errorf("add column migration: %w", err)
					}
				}
			}
			return nil
		},
	},
	{
		version: 3,
		name:    "deep_queue",
		apply: func(db *sql.DB) error {
			if _, err := db.Exec(deepQueueSchema); err != nil {
				return fmt.Errorf("run deep_queue schema: %w", err)
			}
			return nil
		},
	},
	{
		version: 4,
		name:    "gbus_user_scoped_features",
		apply: func(db *sql.DB) error {
			for _, stmt := range gbusUserScopeMigrations {
				if _, err := db.Exec(stmt); err != nil {
					if !strings.Contains(err.Error(), "duplicate column") {
						return fmt.Errorf("gbus user-scope migration: %w", err)
					}
				}
			}
			return nil
		},
	},
}

// gbusUserScopeMigrations adds user_id to GBUS feature tables (future-proofing
// for multi-user/sync, Change 16) and confidence/evidence_count to category
// features so inference/training can treat low-evidence rows conservatively.
// user_id defaults to gbus.DefaultUserID ("local") for all existing rows.
var gbusUserScopeMigrations = []string{
	`ALTER TABLE gbus_category_features ADD COLUMN user_id TEXT NOT NULL DEFAULT 'local'`,
	`ALTER TABLE gbus_category_features ADD COLUMN evidence_count INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE gbus_category_features ADD COLUMN confidence REAL NOT NULL DEFAULT 0`,
	`ALTER TABLE gbus_resource_features ADD COLUMN user_id TEXT NOT NULL DEFAULT 'local'`,
	`CREATE INDEX IF NOT EXISTS idx_gbus_cat_features_user ON gbus_category_features(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_gbus_res_features_user ON gbus_resource_features(user_id)`,
}

// deepQueueSchema is the durable, DB-backed deep-processing queue (Change 12 WS2).
// It replaces the in-memory channel as the source of truth: enqueue writes a
// row here, workers claim/complete/fail rows, and a crash leaves rows in
// 'pending' or 'in_progress' that are requeued on the next startup.
const deepQueueSchema = `
CREATE TABLE IF NOT EXISTS deep_queue (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	resource_id     TEXT NOT NULL,
	status          TEXT NOT NULL CHECK(status IN ('pending','in_progress','done','failed_retryable')),
	attempts        INTEGER NOT NULL DEFAULT 0,
	max_attempts    INTEGER NOT NULL DEFAULT 5,
	last_error      TEXT NOT NULL DEFAULT '',
	next_attempt_at TEXT NOT NULL,
	created_at      TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_deep_queue_claim ON deep_queue(status, next_attempt_at);

-- At most one active (pending or in-progress) row per resource.
CREATE UNIQUE INDEX IF NOT EXISTS idx_deep_queue_active_resource
	ON deep_queue(resource_id)
	WHERE status IN ('pending','in_progress');
`

// migrate brings db up to the latest schema version, recording each applied
// migration in schema_migrations. If the database is already at version > 0
// and has pending migrations, a VACUUM INTO backup is taken first so the
// pre-migration state can be restored if the upgrade goes wrong.
func migrate(db *sql.DB, dbPath string) error {
	if _, err := db.Exec(schemaMigrationsTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var current sql.NullInt64
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	currentVersion := int(current.Int64)

	var pending []migration
	for _, m := range migrations {
		if m.version > currentVersion {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	if currentVersion > 0 && dbPath != "" {
		if _, err := Backup(db, dbPath); err != nil {
			return fmt.Errorf("pre-migration backup: %w", err)
		}
	}

	for _, m := range pending {
		if err := m.apply(db); err != nil {
			return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", m.version, m.name); err != nil {
			return fmt.Errorf("record migration %d (%s): %w", m.version, m.name, err)
		}
	}
	return nil
}
