// Package db opens callback's SQLite database and brings its schema up to date. See
// docs/adr/0003-store-user-data-in-sqlite.md, docs/adr/0004-use-mattn-go-sqlite3-as-the-sqlite-driver.md,
// and docs/adr/0006-run-database-migrations-with-goose.md.
package db

import (
	"context"
	"database/sql"
	"errors"
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

// ErrNewerData is returned by Open when a newer version of callback has migrated the database past
// the newest migration this version embeds. Open leaves the data unchanged.
var ErrNewerData = errors.New("db: the data is from a newer version of callback")

// Open opens the database in dataDir, creating the directory and the database file if they don't
// exist, and applies any pending migrations from migrations. If the database is newer than
// migrations, it returns ErrNewerData without changing anything. See
// docs/adr/0025-restore-backups-with-sqlites-online-backup-api-and-check-data-versions-first.md.
// @scenario [features/data-version.feature#app-newer-data] The app refuses data a newer Callback has updated
// @scenario [features/data-version.feature#app-older-data] The app brings older data up to date
// @scenario [features/data-version.feature#app-same-version-data] The app opens data from the same version as usual
func Open(ctx context.Context, dataDir string, migrations fs.FS) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("db: create data directory: %w", err)
	}

	// An existing database file is opened as is, so everything saved in an earlier session is
	// still there.
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

// migrate checks the database's version, then brings it up to date. The check comes first, since
// migrating data a newer version left would run against a schema this version doesn't know.
func migrate(ctx context.Context, conn *sql.DB, migrations fs.FS) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, migrations)
	if err != nil {
		return fmt.Errorf("db: load migrations: %w", err)
	}
	current, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("db: read the data's version: %w", err)
	}
	sources := provider.ListSources()
	if newest := sources[len(sources)-1].Version; current > newest {
		return fmt.Errorf("%w (data at version %d, this version knows up to %d)", ErrNewerData, current, newest)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("db: migrate: %w", err)
	}
	return nil
}
