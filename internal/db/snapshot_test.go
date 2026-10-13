package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotToCopiesTheData(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	createJob(t, s, "Acme", "saved", 0)
	path := filepath.Join(t.TempDir(), "snapshot.db")

	if err := s.SnapshotTo(context.Background(), path); err != nil {
		t.Fatal(err)
	}

	snap, err := sql.Open("sqlite3", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	var company string
	if err := snap.QueryRow(`SELECT company FROM jobs`).Scan(&company); err != nil || company != "Acme" {
		t.Errorf("snapshot holds %q, %v, want Acme", company, err)
	}
}

func TestSnapshotToFailsOnAnExistingFile(t *testing.T) {
	s, _ := openStore(t, t.TempDir())
	path := filepath.Join(t.TempDir(), "snapshot.db")
	if err := os.WriteFile(path, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.SnapshotTo(context.Background(), path); err == nil {
		t.Error("snapshot over an existing file succeeded")
	}
	if got, _ := os.ReadFile(path); string(got) != "keep me" {
		t.Errorf("existing file now holds %q", got)
	}
}
