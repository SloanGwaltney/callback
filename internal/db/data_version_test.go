package db

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"callback/internal/db/migrations"
)

// withNewerMigration returns the embedded migrations plus one a newer version of callback might
// add, numbered past any real migration.
func withNewerMigration(t *testing.T) fstest.MapFS {
	t.Helper()
	newer := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := fs.ReadFile(migrations.FS, e.Name())
		if err != nil {
			t.Fatal(err)
		}
		newer[e.Name()] = &fstest.MapFile{Data: data}
	}
	newer["99999_from_a_newer_version.sql"] = &fstest.MapFile{Data: []byte(`
-- +goose Up
CREATE TABLE from_a_newer_version (id INTEGER PRIMARY KEY);

-- +goose Down
DROP TABLE from_a_newer_version;
`)}
	return newer
}

// openWith opens dataDir with migrations and closes it at the end of the test.
func openWith(t *testing.T, dataDir string, migrations fs.FS) *Store {
	t.Helper()
	conn, err := Open(context.Background(), dataDir, migrations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return NewStore(conn)
}

func readDatabase(t *testing.T, dataDir string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// The app and the command line both open the data through Open (cmd/callback-cli), so these tests
// carry their steps for both.

// @scenario [#app-newer-data] [#cli-newer-data]
func TestOpenRefusesDataFromANewerVersion(t *testing.T) {
	dataDir := t.TempDir()
	// @given [features/data-version.feature] my data was last updated by a newer version of Callback than the app
	// @given [features/agent-cli.feature] my data was last updated by a newer version of Callback than the command line
	conn, err := Open(context.Background(), dataDir, withNewerMigration(t))
	if err != nil {
		t.Fatal(err)
	}
	createJob(t, NewStore(conn), "Acme", "applied", 0)
	conn.Close()
	before := readDatabase(t, dataDir)

	// @when [features/data-version.feature] I open the app
	// @when [features/agent-cli.feature] my agent runs any command
	conn, err = Open(context.Background(), dataDir, migrations.FS)

	if conn != nil {
		conn.Close()
	}
	if !errors.Is(err, ErrNewerData) {
		t.Fatalf("got %v, want ErrNewerData", err)
	}
	// @and [features/data-version.feature] my data is unchanged
	// @then [features/agent-cli.feature] my data is unchanged
	if !bytes.Equal(readDatabase(t, dataDir), before) {
		t.Error("opening data from a newer version changed the database file")
	}
}

// @scenario [#app-older-data] [#cli-older-data]
func TestOpenBringsOlderDataUpToDate(t *testing.T) {
	dataDir := t.TempDir()
	// @given [features/data-version.feature] my data was last updated by an older version of Callback
	// @given [features/agent-cli.feature] my data was last updated by an older version of Callback
	older := openBefore(t, dataDir, "00007")
	createJob(t, NewStore(older), "Acme", "applied", 0)
	older.Close()

	// @when [features/data-version.feature] I open the app
	// @when [features/agent-cli.feature] my agent runs any command
	s := openWith(t, dataDir, migrations.FS)

	// @then [features/data-version.feature] my data is brought up to date
	// @then [features/agent-cli.feature] my data is brought up to date first
	// Status changes came in 00007, so its table is only there once the data is up to date.
	if _, err := s.ListStatusChangesForJob(context.Background(), 1); err != nil {
		t.Errorf("the newest migration wasn't applied: %v", err)
	}
	// @and [features/data-version.feature] the board shows my jobs as usual
	if got := companiesIn(t, s, "applied"); len(got) != 1 || got[0] != "Acme" {
		t.Errorf("applied jobs = %v, want [Acme]", got)
	}
}

// @scenario [#app-same-version-data]
func TestOpenOpensDataFromTheSameVersion(t *testing.T) {
	dataDir := t.TempDir()
	// @given [features/data-version.feature] my data was last updated by the same version of Callback as the app
	conn, err := Open(context.Background(), dataDir, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	createJob(t, NewStore(conn), "Acme", "applied", 0)
	conn.Close()

	// @when [features/data-version.feature] I open the app
	s := openWith(t, dataDir, migrations.FS)

	// @then [features/data-version.feature] the board shows my jobs as usual
	if got := companiesIn(t, s, "applied"); len(got) != 1 || got[0] != "Acme" {
		t.Errorf("applied jobs = %v, want [Acme]", got)
	}
}

func TestOpenCreatesANewDatabaseAtTheNewestVersion(t *testing.T) {
	s := openWith(t, t.TempDir(), migrations.FS)
	if _, err := s.ListJobs(context.Background()); err != nil {
		t.Errorf("a new database wasn't migrated: %v", err)
	}
}
