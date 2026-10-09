// Package db opens callback's SQLite database and brings its schema up to date. See
// docs/adr/0003-store-user-data-in-sqlite.md, docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md,
// and docs/adr/0006-run-database-migrations-with-goose.md.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// FileName is the database file's name inside the data directory.
const FileName = "callback.db"

// Open opens the database in dataDir, creating the directory and the database file if they don't
// exist, and applies any pending migrations from migrations.
func Open(ctx context.Context, dataDir string, migrations fs.FS) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("db: create data directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dsn(filepath.Join(dataDir, FileName)))
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	if err := migrate(ctx, conn, migrations); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// dsn builds the connection string. SQLite leaves foreign key enforcement off unless it's turned
// on for each connection, and cascading deletes depend on it. The busy timeout makes a connection
// wait for a lock instead of failing at once.
func dsn(path string) string {
	params := url.Values{}
	params.Set("_foreign_keys", "on")
	params.Set("_busy_timeout", "5000")
	return "file:" + filepath.ToSlash(path) + "?" + params.Encode()
}

func migrate(ctx context.Context, conn *sql.DB, migrations fs.FS) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, migrations)
	if err != nil {
		return fmt.Errorf("db: load migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("db: migrate: %w", err)
	}
	return nil
}
