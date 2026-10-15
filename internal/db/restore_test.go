package db

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"callback/internal/db/migrations"
	"callback/internal/store"
)

// Restoring a backup into the live database with the online backup API. internal/backups decides
// which backup to restore, and makes the backup of the data from just before.

// snapshot backs up s to a file named name in a new directory, as an agent backup is made, and
// returns its path.
func snapshot(t *testing.T, s *Store, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".db")
	if err := s.SnapshotTo(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func statusOf(t *testing.T, s *Store, id int64) string {
	t.Helper()
	job, err := s.GetJob(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return job.Status
}

func noBefore() error { return nil }

func TestRestoreFromCopiesTheBackupIn(t *testing.T) {
	s, conn := openStore(t, t.TempDir())
	id := createJob(t, s, "Acme", "applied", 0)
	path := snapshot(t, s, "before-cleanup")
	createJob(t, s, "Globex", "saved", 0)

	ran := false
	if err := NewRestorer(conn, migrations.FS).RestoreFrom(context.Background(), path, func() error { ran = true; return nil }); err != nil {
		t.Fatal(err)
	}

	if !ran {
		t.Error("before didn't run")
	}
	if got := companiesIn(t, s, "saved"); len(got) != 0 {
		t.Errorf("saved = %v, want Globex gone", got)
	}
	if got := statusOf(t, s, id); got != "applied" {
		t.Errorf("status = %q", got)
	}
}

func TestRestoreFromRestoresNothingWhenBeforeFails(t *testing.T) {
	s, conn := openStore(t, t.TempDir())
	path := snapshot(t, s, "before-cleanup")
	createJob(t, s, "Acme", "saved", 0)
	errBefore := errors.New("the undo backup failed")

	err := NewRestorer(conn, migrations.FS).RestoreFrom(context.Background(), path, func() error { return errBefore })

	if !errors.Is(err, errBefore) {
		t.Errorf("err = %v, want before's error", err)
	}
	if got := companiesIn(t, s, "saved"); len(got) != 1 {
		t.Errorf("saved = %v, want Acme still there", got)
	}
}

func TestRestoreFromAMissingBackupChangesNothing(t *testing.T) {
	s, conn := openStore(t, t.TempDir())
	createJob(t, s, "Acme", "saved", 0)

	err := NewRestorer(conn, migrations.FS).RestoreFrom(context.Background(), filepath.Join(t.TempDir(), "gone.db"), noBefore)

	if err == nil {
		t.Error("restoring a missing backup worked")
	}
	if got := companiesIn(t, s, "saved"); len(got) != 1 {
		t.Errorf("saved = %v, want Acme still there", got)
	}
}

// @scenario [#restore-older-backup] [#app-restore-older-version]
func TestRestoreBringsAnOlderBackupUpToDate(t *testing.T) {
	ctx := context.Background()
	// @given [features/agent-backups.feature] my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
	// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
	// The backup is from before the status history (00007) existed.
	backupDir := t.TempDir()
	old := openBefore(t, backupDir, "00007")
	mustExec(t, old, `INSERT INTO jobs (company, role, status, position) VALUES ('Acme', 'Software Engineer', 'applied', 0)`)
	old.Close()
	path := filepath.Join(backupDir, FileName)
	before := readFile(t, path)
	// @and [features/agent-backups.feature] I have since updated to a version of Callback that changes how my data is stored
	// @and [features/restore-backups.feature] I have since updated to a version of Callback that changes how my data is stored
	s, conn := openStore(t, t.TempDir())
	createJob(t, s, "Globex", "saved", 0)

	// @when [features/agent-backups.feature] my agent restores the backup "before-cleanup"
	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	if err := NewRestorer(conn, migrations.FS).RestoreFrom(ctx, path, noBefore); err != nil {
		t.Fatal(err)
	}

	// @then [features/agent-backups.feature] the job at "Acme" is in "Applied" again
	// @then [features/restore-backups.feature] the job at "Acme" is in "Applied" again
	if got := companiesIn(t, s, "applied"); len(got) != 1 || got[0] != "Acme" {
		t.Errorf("applied = %v, want Acme", got)
	}
	// @and [features/agent-backups.feature] my data is stored the way the current version stores it
	// 00007 started each job's status history, which the restored data now has.
	history, err := s.ListStatusChangesForJob(ctx, 1)
	if err != nil || len(history) != 1 || history[0].Status != "applied" {
		t.Errorf("history = %+v, %v, want the job's Applied status", history, err)
	}
	// @and [features/agent-backups.feature] the "before-cleanup" backup itself is unchanged
	// @and [features/restore-backups.feature] the backup from 2026-10-12 itself is unchanged
	if !bytes.Equal(readFile(t, path), before) {
		t.Error("the backup file changed")
	}
}

// @scenario [#restore-newer-backup] [#app-restore-newer-version]
func TestRestoreRefusesANewerBackup(t *testing.T) {
	// @given [features/agent-backups.feature] a backup named "from-newer" was made by a newer version of Callback than the command line
	// @given [features/restore-backups.feature] a daily backup from 2026-10-12 was taken by a newer version of Callback
	backupDir := t.TempDir()
	newer := openWith(t, backupDir, withNewerMigration(t))
	createJob(t, newer, "Initech", "saved", 0)
	path := filepath.Join(backupDir, FileName)
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	createJob(t, s, "Acme", "applied", 0)
	data := readDatabase(t, dataDir)

	// @when [features/agent-backups.feature] my agent tries to restore the backup "from-newer"
	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	ran := false
	err := NewRestorer(conn, migrations.FS).RestoreFrom(context.Background(), path, func() error { ran = true; return nil })

	// @then [features/agent-backups.feature] my data is unchanged
	// @then [features/restore-backups.feature#app-restore-newer-version] my data is unchanged
	if !bytes.Equal(readDatabase(t, dataDir), data) || ran {
		t.Errorf("the data changed, or the backup of the data from before was made (%v)", ran)
	}
	// @and [features/agent-backups.feature] my agent is told the backup is from a newer version of Callback
	// The command line gives ErrNewerBackup its message and exit code (TestRestoringANewerBackup in
	// internal/cli).
	if !errors.Is(err, ErrNewerBackup) {
		t.Errorf("err = %v, want ErrNewerBackup", err)
	}
}

// The app's message for it is TestTheAppIsToldABackupCouldntBeRestored in internal/backups.
// @scenario [#app-restore-unreadable]
func TestRestoreFromADamagedBackupChangesNothing(t *testing.T) {
	// @given [features/restore-backups.feature] the daily backup from 2026-10-12 is damaged
	path := filepath.Join(t.TempDir(), "2026-10-12.db")
	if err := os.WriteFile(path, []byte("not a database, or what's left of one"), 0o600); err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	s, conn := openStore(t, dataDir)
	createJob(t, s, "Acme", "applied", 0)
	data := readDatabase(t, dataDir)

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	ran := false
	err := NewRestorer(conn, migrations.FS).RestoreFrom(context.Background(), path, func() error { ran = true; return nil })

	// @then [features/restore-backups.feature#app-restore-unreadable] my data is unchanged
	if err == nil {
		t.Error("restoring a damaged backup worked")
	}
	if !bytes.Equal(readDatabase(t, dataDir), data) || ran {
		t.Errorf("the data changed, or the backup of the data from before was made (%v)", ran)
	}
}

// @scenario [#restore-app-open-safe]
func TestRestoreWhileTheAppIsOpen(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	// @given [features/agent-backups.feature] the app is open
	app, _ := openStore(t, dataDir)
	id := createJob(t, app, "Acme", "applied", 0)
	// @and [features/agent-backups.feature] my agent made a backup named "before-cleanup"
	path := snapshot(t, app, "before-cleanup")
	if err := app.UpdateJobPlacement(ctx, store.UpdateJobPlacementParams{ID: id, Status: "interviewing"}); err != nil {
		t.Fatal(err)
	}

	// @when [features/agent-backups.feature] my agent restores the backup "before-cleanup"
	// The command line opens the same data on a connection of its own.
	_, cli := openStore(t, dataDir)
	err := NewRestorer(cli, migrations.FS).RestoreFrom(ctx, path, noBefore)

	// @then [features/agent-backups.feature] the restore finishes without the app being closed
	if err != nil {
		t.Fatal(err)
	}
	// @and [features/agent-backups.feature] the app keeps working with the restored data
	if got := statusOf(t, app, id); got != "applied" {
		t.Errorf("the app reads status %q, want the restored applied", got)
	}
	createJob(t, app, "Globex", "saved", 0)
	if got := companiesIn(t, app, "saved"); len(got) != 1 || got[0] != "Globex" {
		t.Errorf("the app's new job: saved = %v", got)
	}
}
