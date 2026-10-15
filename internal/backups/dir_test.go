package backups

import (
	"bytes"
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"callback/internal/db"
	"callback/internal/db/migrations"
	"callback/internal/docfiles"
	"callback/internal/store"
)

// Storage-tier tests against a real temporary directory and SQLite database. See
// docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

// filesIn lists every file under dir, as slash-separated paths relative to it.
func filesIn(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		out = append(out, filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// @scenario [#backup-holds-data-not-documents]
func TestABackupHoldsTheDataButNotTheDocumentsFiles(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	conn, err := db.Open(ctx, dataDir, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	s := db.NewStore(conn)

	// @given [features/daily-backups.feature] I have a job for "Software Engineer" at "Acme" with "resume-2026.pdf" attached
	jobID, err := s.CreateJob(ctx, store.CreateJobParams{
		Company: "Acme", Role: "Software Engineer", Status: "applied", Notes: "Referred by Sam",
	})
	if err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(t.TempDir(), "resume-2026.pdf")
	writeFile(t, original, "%PDF resume-2026 contents")
	stored, err := docfiles.New(dataDir).Save(original)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateDocument(ctx, store.CreateDocumentParams{
		JobID: jobID, Kind: "resume", FileName: "resume-2026.pdf", StoredPath: stored,
	}); err != nil {
		t.Fatal(err)
	}

	// @when [features/daily-backups.feature] a daily backup is taken
	svc := New(s, db.NewRestorer(conn, migrations.FS), NewDir(dataDir), NewAgentDir(dataDir), func() time.Time { return time.Date(2026, time.October, 14, 9, 0, 0, 0, time.Local) })
	if _, err := svc.TakeDaily(ctx); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(dataDir, "backups", "daily", "2026-10-14.db")
	backup, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(backupPath)+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	// @then [features/daily-backups.feature] the backup holds the job, its details, and that "resume-2026.pdf" is attached
	var company, role, notes, fileName string
	err = backup.QueryRow(`
		SELECT j.company, j.role, j.notes, d.file_name
		FROM jobs j JOIN documents d ON d.job_id = j.id`).Scan(&company, &role, &notes, &fileName)
	if err != nil || company != "Acme" || role != "Software Engineer" || notes != "Referred by Sam" || fileName != "resume-2026.pdf" {
		t.Errorf("backup holds %q, %q, %q, %q, %v", company, role, notes, fileName, err)
	}
	// @but [features/daily-backups.feature] the backup doesn't hold a copy of the "resume-2026.pdf" file
	if got := filesIn(t, filepath.Join(dataDir, "backups")); !slices.Equal(got, []string{"daily/2026-10-14.db"}) {
		t.Errorf("backups folder holds %v, want only the database", got)
	}
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("%PDF resume-2026 contents")) {
		t.Error("the backup holds the file's contents")
	}
}

// @scenario [#backup-failure-still-starts]
func TestAFailedBackupLeavesNothingBehind(t *testing.T) {
	dataDir := t.TempDir()
	d := NewDir(dataDir)
	// @given [features/daily-backups.feature] no backup has been taken today
	if dates, err := d.Dates(); err != nil || len(dates) != 0 {
		t.Fatalf("dates = %v, %v", dates, err)
	}

	// @and [features/daily-backups.feature] the backup can't be written
	// It fails partway, after writing some of the file.
	failing := func(path string) error {
		writeFile(t, path, "half a database")
		return errFake
	}
	if err := d.Save("2026-10-14", failing); err == nil {
		t.Fatal("a failed backup was saved")
	}

	// @and [features/daily-backups.feature] no partial backup is left behind
	if got := filesIn(t, dataDir); len(got) != 0 {
		t.Errorf("left %v behind", got)
	}
}

func TestDatesAreOldestFirstAndIgnoreOtherFiles(t *testing.T) {
	dataDir := t.TempDir()
	daily := filepath.Join(dataDir, "backups", "daily")
	for _, name := range []string{"2026-10-14.db", "2026-09-30.db", "notes.db", "2026-10-13.db.partial", "2026-13-01.db"} {
		writeFile(t, filepath.Join(daily, name), "x")
	}
	if err := os.Mkdir(filepath.Join(daily, "2026-10-01.db"), 0o700); err != nil {
		t.Fatal(err)
	}

	dates, err := NewDir(dataDir).Dates()
	if err != nil || !slices.Equal(dates, []string{"2026-09-30", "2026-10-14"}) {
		t.Errorf("Dates() = %v, %v", dates, err)
	}
}

func TestDatesWithNoFolderYet(t *testing.T) {
	dates, err := NewDir(t.TempDir()).Dates()
	if err != nil || len(dates) != 0 {
		t.Errorf("Dates() = %v, %v", dates, err)
	}
}

func TestSaveReplacesAPartialFileFromAnEarlierCrash(t *testing.T) {
	dataDir := t.TempDir()
	d := NewDir(dataDir)
	writeFile(t, filepath.Join(dataDir, "backups", "daily", "2026-10-14.db.partial"), "half a database")

	err := d.Save("2026-10-14", func(path string) error {
		if _, err := os.Stat(path); err == nil {
			t.Error("the old partial file is still there")
		}
		writeFile(t, path, "whole database")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := filesIn(t, dataDir); !slices.Equal(got, []string{"backups/daily/2026-10-14.db"}) {
		t.Errorf("files %v", got)
	}
}

func TestRemove(t *testing.T) {
	dataDir := t.TempDir()
	d := NewDir(dataDir)
	for _, date := range []string{"2026-10-13", "2026-10-14"} {
		if err := d.Save(date, func(path string) error { writeFile(t, path, date); return nil }); err != nil {
			t.Fatal(err)
		}
	}

	if err := d.Remove("2026-10-13"); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove("2026-10-13"); err != nil {
		t.Errorf("removing a backup that's already gone: %v", err)
	}
	if dates, _ := d.Dates(); !slices.Equal(dates, []string{"2026-10-14"}) {
		t.Errorf("Dates() = %v", dates)
	}
}
