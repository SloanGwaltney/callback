package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// Storage-tier tests run against a real SQLite file in a temporary directory. See
// docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

var testMigrations = fstest.MapFS{
	"00001_parents.sql": {Data: []byte(`
-- +goose Up
CREATE TABLE parents (id INTEGER PRIMARY KEY);
CREATE TABLE children (
    id INTEGER PRIMARY KEY,
    parent_id INTEGER NOT NULL REFERENCES parents (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE children;
DROP TABLE parents;
`)},
}

func TestOpenCreatesDataDirAndDatabase(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "nested", "callback")

	conn, err := Open(context.Background(), dataDir, testMigrations)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := os.Stat(filepath.Join(dataDir, FileName)); err != nil {
		t.Errorf("database file not created: %v", err)
	}
}

func TestOpenAppliesMigrations(t *testing.T) {
	conn, err := Open(context.Background(), t.TempDir(), testMigrations)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`INSERT INTO parents (id) VALUES (1)`); err != nil {
		t.Errorf("migrated table missing: %v", err)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dataDir := t.TempDir()
	for i := range 2 {
		conn, err := Open(context.Background(), dataDir, testMigrations)
		if err != nil {
			t.Fatalf("open %d: %v", i+1, err)
		}
		conn.Close()
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	conn, err := Open(context.Background(), t.TempDir(), testMigrations)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Pin one connection so the PRAGMA-dependent statements below share it.
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`INSERT INTO children (id, parent_id) VALUES (1, 99)`); err == nil {
		t.Error("inserting a child with no parent succeeded; foreign keys are off")
	}

	if _, err := conn.Exec(`INSERT INTO parents (id) VALUES (1); INSERT INTO children (id, parent_id) VALUES (1, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`DELETE FROM parents WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM children`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("cascade delete left %d children", n)
	}
}

func TestOpenFailsWithoutMigrations(t *testing.T) {
	if _, err := Open(context.Background(), t.TempDir(), fstest.MapFS{}); err == nil {
		t.Error("expected an error when there are no migrations")
	}
}

func TestDSNHandlesWindowsPaths(t *testing.T) {
	got := dsn(`C:\Users\sam\AppData\Local\callback\callback.db`)
	want := "file:C:/Users/sam/AppData/Local/callback/callback.db?_busy_timeout=5000&_foreign_keys=on"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
