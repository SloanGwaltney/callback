package backups

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// Restoring from the app against a real temporary directory and SQLite database, restored with the
// real db.Restorer. See docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

// takeDaily takes the day's backup, which must not have been taken yet.
func (l *live) takeDaily(t *testing.T) {
	t.Helper()
	if taken, err := l.svc.TakeDaily(l.ctx); err != nil || !taken {
		t.Fatalf("taken = %v, %v", taken, err)
	}
}

func (l *live) restoreBackup(t *testing.T, kind Kind, name string) Backup {
	t.Helper()
	undo, err := l.svc.RestoreBackup(l.ctx, kind, name)
	if err != nil {
		t.Fatal(err)
	}
	return undo
}

func (l *live) listAll(t *testing.T) []Backup {
	t.Helper()
	listed, err := l.svc.List()
	if err != nil {
		t.Fatal(err)
	}
	return listed
}

// setMade sets when the backup file at rel, under the backups folder, was made.
func (l *live) setMade(t *testing.T, rel string, made time.Time) {
	t.Helper()
	if err := os.Chtimes(filepath.Join(l.dataDir, "backups", filepath.FromSlash(rel)), made, made); err != nil {
		t.Fatal(err)
	}
}

// @scenario [#app-restore-daily-backup]
func TestRestoreADailyBackupFromTheApp(t *testing.T) {
	l := newLive(t)
	// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
	l.setToday(2026, time.October, 12)
	id := l.addJob(t, "Acme", "applied")
	l.takeDaily(t)
	// @and [features/restore-backups.feature] I have since moved the job to "Closed"
	l.setToday(2026, time.October, 14)
	l.setStatus(t, id, "closed")

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	l.restoreBackup(t, Daily, "2026-10-12")

	// @then [features/restore-backups.feature] the job at "Acme" is in "Applied" again
	if got := l.status(t, id); got != "applied" {
		t.Errorf("status = %q, want applied", got)
	}
}

// @scenario [#app-restore-agent-backup]
func TestRestoreAnAgentBackupFromTheApp(t *testing.T) {
	l := newLive(t)
	// @given [features/restore-backups.feature] my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
	id := l.addJob(t, "Acme", "applied")
	l.makeAgent(t, "before-cleanup", 7)
	// @and [features/restore-backups.feature] my agent then moved the job to "Closed"
	l.setStatus(t, id, "closed")

	// @when [features/restore-backups.feature] I restore the backup "before-cleanup" and confirm
	l.restoreBackup(t, Agent, "before-cleanup")

	// @then [features/restore-backups.feature] the job at "Acme" is in "Applied" again
	if got := l.status(t, id); got != "applied" {
		t.Errorf("status = %q, want applied", got)
	}
	// The agent's backup is still kept, and still the agent's.
	if got := l.list(t); len(got) != 1 || got[0].Name != "before-cleanup" {
		t.Errorf("agent backups %v", got)
	}
}

// @scenario [#app-undo-restore]
func TestUndoARestoreFromTheApp(t *testing.T) {
	l := newLive(t)
	// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12 while I had a job at "Acme" in "Applied"
	l.setToday(2026, time.October, 12)
	id := l.addJob(t, "Acme", "applied")
	l.takeDaily(t)
	// @and [features/restore-backups.feature] I have since moved the job to "Closed"
	l.setToday(2026, time.October, 14)
	l.setStatus(t, id, "closed")
	// @and [features/restore-backups.feature] I restored the backup from 2026-10-12
	undo := l.restoreBackup(t, Daily, "2026-10-12")

	// @when [features/restore-backups.feature] I restore the backup made before that restore and confirm
	l.restoreBackup(t, undo.Kind, undo.Name)

	// @then [features/restore-backups.feature] the job at "Acme" is in "Closed" again
	if got := l.status(t, id); got != "closed" {
		t.Errorf("status = %q, want closed", got)
	}
}

func TestTheAppsBackupsAreKeptApartFromTheAgents(t *testing.T) {
	l := newLive(t)
	l.setToday(2026, time.October, 12)
	l.takeDaily(t)
	l.setToday(2026, time.October, 14)

	undo := l.restoreBackup(t, Daily, "2026-10-12")

	want := []string{"app/" + undo.Name + ".2026-10-21.db", "daily/2026-10-12.db"}
	if got := filesIn(t, filepath.Join(l.dataDir, "backups")); !slices.Equal(got, want) {
		t.Errorf("backups folder holds %v, want %v", got, want)
	}
	if got := l.list(t); len(got) != 0 {
		t.Errorf("agent backups %v, want none", got)
	}
}

func TestListShowsEveryKindNewestFirstByWhenItWasMade(t *testing.T) {
	l := newLive(t)
	l.setToday(2026, time.October, 12)
	l.takeDaily(t)
	l.setToday(2026, time.October, 14)
	l.makeAgent(t, "before-cleanup", 7)
	undo := l.restoreBackup(t, Daily, "2026-10-12")
	l.setMade(t, "daily/2026-10-12.db", at(12, 0))
	l.setMade(t, "agent/before-cleanup.2026-10-21.db", at(13, 0))
	l.setMade(t, "app/"+undo.Name+".2026-10-21.db", at(14, 0))

	listed := l.listAll(t)

	want := []string{"app " + undo.Name, "agent before-cleanup", "daily 2026-10-12"}
	if got := names(listed); !slices.Equal(got, want) {
		t.Errorf("listed %v, want %v", got, want)
	}
	if listed[2].Made != at(12, 0).Format(time.RFC3339) {
		t.Errorf("daily backup made %q", listed[2].Made)
	}
}

// Opening a document whose stored copy is gone is TestOpenADocumentRestoredWithoutItsFile in
// internal/jobs.
// @scenario [#app-restore-missing-document-file]
func TestRestoreFromTheAppListsARemovedDocumentWithoutItsFile(t *testing.T) {
	l := newLive(t)
	// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12 while "resume-2026.pdf" was attached to a job at "Acme"
	l.setToday(2026, time.October, 12)
	id := l.addJob(t, "Acme", "applied")
	stored := l.attach(t, id, "resume-2026.pdf")
	l.takeDaily(t)
	// @and [features/restore-backups.feature] I then removed "resume-2026.pdf" from the job
	l.setToday(2026, time.October, 14)
	docs, err := l.store.ListDocumentsForJob(l.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.store.DeleteDocument(l.ctx, docs[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(stored); err != nil {
		t.Fatal(err)
	}

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	l.restoreBackup(t, Daily, "2026-10-12")

	// @then [features/restore-backups.feature] the job at "Acme" lists "resume-2026.pdf" again
	if got := l.documentNames(t, id); !slices.Equal(got, []string{"resume-2026.pdf"}) {
		t.Errorf("documents = %v, want resume-2026.pdf", got)
	}
	if exists(t, stored) {
		t.Error("the restore brought back the document's file")
	}
}

// @scenario [#app-restore-keeps-document-files]
func TestRestoreFromTheAppNeverDeletesADocumentsFile(t *testing.T) {
	l := newLive(t)
	// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12
	l.setToday(2026, time.October, 12)
	id := l.addJob(t, "Acme", "applied")
	l.takeDaily(t)
	// @and [features/restore-backups.feature] I then attached "acme-cover.docx" to a job at "Acme"
	l.setToday(2026, time.October, 14)
	stored := l.attach(t, id, "acme-cover.docx")

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	l.restoreBackup(t, Daily, "2026-10-12")

	// @then [features/restore-backups.feature] the job at "Acme" no longer lists "acme-cover.docx"
	if got := l.documentNames(t, id); len(got) != 0 {
		t.Errorf("documents = %v, want none", got)
	}
	// @but [features/restore-backups.feature] the stored copy of "acme-cover.docx" is still in the app's data directory
	if !exists(t, stored) {
		t.Error("the restore deleted the stored copy")
	}
}
