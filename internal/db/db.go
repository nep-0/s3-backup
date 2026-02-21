package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS endpoints (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	endpoint TEXT NOT NULL,
	access_key TEXT NOT NULL,
	secret_key TEXT NOT NULL,
	bucket TEXT NOT NULL,
	prefix TEXT NOT NULL,
	region TEXT NOT NULL,
	use_ssl INTEGER NOT NULL,
	path_style INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS watch_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT NOT NULL,
	endpoint_id INTEGER NOT NULL,
	target_path TEXT NOT NULL,
	excludes TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	FOREIGN KEY(endpoint_id) REFERENCES endpoints(id)
);

CREATE TABLE IF NOT EXISTS backups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	watch_item_id INTEGER NOT NULL,
	endpoint_id INTEGER NOT NULL,
	started_at TEXT NOT NULL,
	completed_at TEXT,
	status TEXT NOT NULL,
	total_files INTEGER NOT NULL,
	total_bytes INTEGER NOT NULL,
	error TEXT,
	FOREIGN KEY(watch_item_id) REFERENCES watch_items(id),
	FOREIGN KEY(endpoint_id) REFERENCES endpoints(id)
);

CREATE TABLE IF NOT EXISTS backup_files (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	backup_id INTEGER NOT NULL,
	path TEXT NOT NULL,
	size INTEGER NOT NULL,
	mod_time TEXT NOT NULL,
	hash TEXT NOT NULL,
	object_key TEXT NOT NULL,
	zstd_size INTEGER NOT NULL,
	FOREIGN KEY(backup_id) REFERENCES backups(id)
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &DB{DB: db}, nil
}

func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func (d *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx rollback: %v (orig: %w)", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}
