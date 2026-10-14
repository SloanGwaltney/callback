package backups

import (
	"context"
	"database/sql"
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

// Agent backups against a real temporary directory and SQLite database, restored with the real
// db.Restorer. See docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

// live is the user's data in a temporary data directory, with the backups service over it.
type live struct {
	ctx     context.Context
	dataDir string
	store   *db.Store
	svc     *Service
	today   time.Time
}

func newLive(t *testing.T) *live {
	t.Helper()
	l := &live{ctx: context.Background(), dataDir: t.TempDir()}
	conn, err := db.Open(l.ctx, l.dataDir, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	l.store = db.NewStore(conn)
	l.svc = New(l.store, db.NewRestorer(conn, migrations.FS), NewDir(l.dataDir), NewAgentDir(l.dataDir), func() time.Time { return l.today })
	l.setToday(2026, time.October, 14)
	return l
}

func (l *live) setToday(year int, month time.Month, day int) {
	l.today = time.Date(year, month, day, 9, 30, 0, 0, time.Local)
}

func (l *live) addJob(t *testing.T, company, status string) int64 {
	t.Helper()
	id, err := l.store.CreateJob(l.ctx, store.CreateJobParams{Company: company, Role: "Software Engineer", Status: status})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (l *live) setStatus(t *testing.T, id int64, status string) {
	t.Helper()
	p := store.UpdateJobPlacementParams{ID: id, Status: status}
	if status == "closed" {
		p.ClosingReason = sql.NullString{String: "withdrew", Valid: true}
	}
	if err := l.store.UpdateJobPlacement(l.ctx, p); err != nil {
		t.Fatal(err)
	}
}

func (l *live) status(t *testing.T, id int64) string {
	t.Helper()
	job, err := l.store.GetJob(l.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return job.Status
}

func (l *live) makeAgent(t *testing.T, name string, days int) AgentBackup {
	t.Helper()
	b, err := l.svc.MakeAgent(l.ctx, name, days)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (l *live) restore(t *testing.T, name string) AgentBackup {
	t.Helper()
	undo, err := l.svc.Restore(l.ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	return undo
}

func (l *live) list(t *testing.T) []AgentBackup {
	t.Helper()
	listed, err := l.svc.ListAgent()
	if err != nil {
		t.Fatal(err)
	}
	return listed
}

// attach stores a copy of a new file named fileName and attaches it to the job, as the jobs
// service does, and returns the stored copy's path.
func (l *live) attach(t *testing.T, jobID int64, fileName string) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), fileName)
	writeFile(t, src, fileName+" contents")
	stored, err := docfiles.New(l.dataDir).Save(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.store.CreateDocument(l.ctx, store.CreateDocumentParams{
		JobID: jobID, Kind: "other", FileName: fileName, StoredPath: stored,
	}); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(l.dataDir, filepath.FromSlash(stored))
}

func (l *live) documentNames(t *testing.T, jobID int64) []string {
	t.Helper()
	docs, err := l.store.ListDocumentsForJob(l.ctx, jobID)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range docs {
		out = append(out, d.FileName)
	}
	return out
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return err == nil
}

// @scenario [#agent-backup]
func TestAgentBackupHoldsTheDataAsItIsNow(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] today is 2026-10-14
	l.setToday(2026, time.October, 14)
	// @and [features/agent-backups.feature] I have a job for "Software Engineer" at "Acme"
	l.addJob(t, "Acme", "saved")

	// @when [features/agent-backups.feature] my agent makes a backup named "before-cleanup" kept for 7 days
	b := l.makeAgent(t, "before-cleanup", 7)

	// @then [features/agent-backups.feature] a backup named "before-cleanup" holds my data as it is now
	path := filepath.Join(l.dataDir, "backups", "agent", "before-cleanup.2026-10-21.db")
	backup, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var company, role string
	if err := backup.QueryRow(`SELECT company, role FROM jobs`).Scan(&company, &role); err != nil || company != "Acme" || role != "Software Engineer" {
		t.Errorf("backup holds %q, %q, %v", company, role, err)
	}
	// @and [features/agent-backups.feature] the "before-cleanup" backup is kept until 2026-10-21
	if b.KeptUntil != "2026-10-21" || !slices.Equal(l.list(t), []AgentBackup{{"before-cleanup", "2026-10-21"}}) {
		t.Errorf("made %+v, listed %v", b, l.list(t))
	}
}

// @scenario [#agent-list-backups]
func TestListingLeavesOutTheDailyBackups(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] an agent made a backup named "old-backup" last week that is still kept
	l.setToday(2026, time.October, 7)
	l.makeAgent(t, "old-backup", 30)
	l.setToday(2026, time.October, 14)
	// @and [features/agent-backups.feature] my agent has just made a backup named "before-cleanup"
	l.makeAgent(t, "before-cleanup", 7)
	// @and [features/agent-backups.feature] a daily backup was taken today
	if taken, err := l.svc.TakeDaily(l.ctx); err != nil || !taken {
		t.Fatalf("taken = %v, %v", taken, err)
	}

	// @when [features/agent-backups.feature] my agent lists the backups
	listed := l.list(t)

	// @then [features/agent-backups.feature] it sees "old-backup" and "before-cleanup", each with the day it's kept until
	// @but [features/agent-backups.feature] it doesn't see the daily backup
	want := []AgentBackup{{"before-cleanup", "2026-10-21"}, {"old-backup", "2026-11-06"}}
	if !slices.Equal(listed, want) {
		t.Errorf("listed %v, want %v", listed, want)
	}
}

// @scenario [#agent-backup-expiry-keeps-daily]
func TestRemovingExpiredBackupsKeepsTheDailyBackups(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] there is a daily backup from 60 days ago
	l.setToday(2026, time.August, 15)
	if _, err := l.svc.TakeDaily(l.ctx); err != nil {
		t.Fatal(err)
	}
	l.makeAgent(t, "expired", 1)
	l.setToday(2026, time.October, 14)

	// @when [features/agent-backups.feature] my agent runs a Callback command
	// Every command calls RemoveExpired (TestEveryCommandRemovesExpiredBackups in internal/cli).
	if err := l.svc.RemoveExpired(); err != nil {
		t.Fatal(err)
	}

	// @then [features/agent-backups.feature] the daily backup from 60 days ago is still kept
	if got := filesIn(t, filepath.Join(l.dataDir, "backups")); !slices.Equal(got, []string{"daily/2026-08-15.db"}) {
		t.Errorf("backups folder holds %v, want only the daily backup", got)
	}
}

// @scenario [#backup-rotation-keeps-agent-backups]
func TestRotatingDailyBackupsKeepsAgentBackups(t *testing.T) {
	l := newLive(t)
	// @given [features/daily-backups.feature] there are 15 daily backups
	for day := 1; day <= Keep; day++ {
		l.setToday(2026, time.October, day)
		if _, err := l.svc.TakeDaily(l.ctx); err != nil {
			t.Fatal(err)
		}
	}
	// @and [features/daily-backups.feature] my agent has made a backup named "before-cleanup"
	b := l.makeAgent(t, "before-cleanup", 30)

	// @when [features/daily-backups.feature] a new daily backup is taken
	l.setToday(2026, time.October, 16)
	if _, err := l.svc.TakeDaily(l.ctx); err != nil {
		t.Fatal(err)
	}

	// @then [features/daily-backups.feature] the "before-cleanup" backup is still kept
	if !slices.Equal(l.list(t), []AgentBackup{b}) {
		t.Errorf("listed %v, want %v", l.list(t), b)
	}
	if dates, _ := NewDir(l.dataDir).Dates(); len(dates) != Keep || dates[0] != "2026-10-02" {
		t.Errorf("daily backups %v, want the oldest one rotated out", dates)
	}
}

// @scenario [#agent-restore]
func TestRestoreBringsBackTheData(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
	id := l.addJob(t, "Acme", "applied")
	l.makeAgent(t, "before-cleanup", 7)
	// @and [features/agent-backups.feature] my agent then moved the job to "Closed"
	l.setStatus(t, id, "closed")

	// @when [features/agent-backups.feature] my agent restores the backup "before-cleanup"
	l.restore(t, "before-cleanup")

	// @then [features/agent-backups.feature] the job at "Acme" is in "Applied" again
	if got := l.status(t, id); got != "applied" {
		t.Errorf("status = %q, want applied", got)
	}
}

// @scenario [#agent-restore-earlier-backup]
func TestRestoreABackupAnotherAgentMadeEarlier(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] an agent made a backup named "last-week" last week, while I had a job at "Acme" in "Applied"
	l.setToday(2026, time.October, 7)
	id := l.addJob(t, "Acme", "applied")
	l.makeAgent(t, "last-week", 14)
	// @and [features/agent-backups.feature] the job at "Acme" has since moved to "Interviewing"
	l.setToday(2026, time.October, 14)
	l.setStatus(t, id, "interviewing")

	// @when [features/agent-backups.feature] my agent restores the backup "last-week"
	l.restore(t, "last-week")

	// @then [features/agent-backups.feature] the job at "Acme" is in "Applied" again
	if got := l.status(t, id); got != "applied" {
		t.Errorf("status = %q, want applied", got)
	}
}

// @scenario [#restore-keeps-backups]
func TestRestoringKeepsTheSameBackups(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] my agent made a backup named "first" and then a backup named "second"
	first := l.makeAgent(t, "first", 10)
	second := l.makeAgent(t, "second", 20)

	// @when [features/agent-backups.feature] my agent restores the backup "first"
	undo := l.restore(t, "first")

	// @then [features/agent-backups.feature] my agent still sees "first" and "second" when it lists the backups
	// @and [features/agent-backups.feature] each is still kept until the same day as before
	want := []AgentBackup{undo, first, second}
	if !slices.Equal(l.list(t), want) {
		t.Errorf("listed %v, want %v", l.list(t), want)
	}
}

// @scenario [#undo-restore]
func TestUndoARestore(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] my agent made a backup named "before-cleanup" while I had a job at "Acme" in "Applied"
	id := l.addJob(t, "Acme", "applied")
	l.makeAgent(t, "before-cleanup", 7)
	// @and [features/agent-backups.feature] my agent then moved the job to "Closed"
	l.setStatus(t, id, "closed")
	// @and [features/agent-backups.feature] my agent restored the backup "before-cleanup"
	undo := l.restore(t, "before-cleanup")

	// @when [features/agent-backups.feature] my agent restores the backup made just before that restore
	l.restore(t, undo.Name)

	// @then [features/agent-backups.feature] the job at "Acme" is in "Closed" again
	if got := l.status(t, id); got != "closed" {
		t.Errorf("status = %q, want closed", got)
	}
}

// Opening a document whose stored copy is gone is TestOpenADocumentRestoredWithoutItsFile in
// internal/jobs.
// @scenario [#restore-missing-document-file]
func TestRestoreListsARemovedDocumentWithoutItsFile(t *testing.T) {
	l := newLive(t)
	// @given [features/agent-backups.feature] my agent made a backup named "before-cleanup" while "resume-2026.pdf" was attached to a job at "Acme"
	id := l.addJob(t, "Acme", "applied")
	stored := l.attach(t, id, "resume-2026.pdf")
	l.makeAgent(t, "before-cleanup", 7)
	// @and [features/agent-backups.feature] I then removed "resume-2026.pdf" from the job
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

	// @when [features/agent-backups.feature] my agent restores the backup "before-cleanup"
	l.restore(t, "before-cleanup")

	// @then [features/agent-backups.feature] the job at "Acme" lists "resume-2026.pdf" again
	if got := l.documentNames(t, id); !slices.Equal(got, []string{"resume-2026.pdf"}) {
		t.Errorf("documents = %v, want resume-2026.pdf", got)
	}
	if exists(t, stored) {
		t.Error("the restore brought back the document's file")
	}
}

// @scenario [#restore-keeps-document-files]
func TestRestoreNeverDeletesADocumentsFile(t *testing.T) {
	l := newLive(t)
	id := l.addJob(t, "Acme", "applied")
	// @given [features/agent-backups.feature] my agent made a backup named "before-cleanup"
	l.makeAgent(t, "before-cleanup", 7)
	// @and [features/agent-backups.feature] I then attached "acme-cover.docx" to a job at "Acme"
	stored := l.attach(t, id, "acme-cover.docx")

	// @when [features/agent-backups.feature] my agent restores the backup "before-cleanup"
	l.restore(t, "before-cleanup")

	// @then [features/agent-backups.feature] the job at "Acme" no longer lists "acme-cover.docx"
	if got := l.documentNames(t, id); len(got) != 0 {
		t.Errorf("documents = %v, want none", got)
	}
	// @but [features/agent-backups.feature] the stored copy of "acme-cover.docx" is still in the app's data directory
	if !exists(t, stored) {
		t.Error("the restore deleted the stored copy")
	}
}

func TestAgentDirIgnoresOtherFiles(t *testing.T) {
	dataDir := t.TempDir()
	d := NewAgentDir(dataDir)
	dir := filepath.Join(dataDir, "backups", "agent")
	for _, name := range []string{
		"before-cleanup.2026-10-21.db",
		"before-cleanup.2026-10-22.db.partial",
		"notes.txt",
		"no-date.db",
		"bad date.2026-10-21.db",
		"before-cleanup.not-a-date.db",
	} {
		writeFile(t, filepath.Join(dir, name), "x")
	}

	got, err := d.Backups()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []AgentBackup{{"before-cleanup", "2026-10-21"}}) {
		t.Errorf("backups = %v", got)
	}
}

func TestAgentDirWithNoFolderYet(t *testing.T) {
	got, err := NewAgentDir(t.TempDir()).Backups()
	if err != nil || len(got) != 0 {
		t.Errorf("backups = %v, %v", got, err)
	}
}

func TestAgentDirRemove(t *testing.T) {
	dataDir := t.TempDir()
	d := NewAgentDir(dataDir)
	b := AgentBackup{Name: "before-cleanup", KeptUntil: "2026-10-21"}
	if err := d.Save(b, func(path string) error { writeFile(t, path, "x"); return nil }); err != nil {
		t.Fatal(err)
	}

	if err := d.Remove(b); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove(b); err != nil {
		t.Errorf("removing it again: %v", err)
	}
	if got := filesIn(t, dataDir); len(got) != 0 {
		t.Errorf("left %v", got)
	}
}
