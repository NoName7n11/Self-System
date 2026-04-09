package sqlite

import (
	"database/sql"
	"fmt"
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
`

func migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
