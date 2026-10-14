package backups

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// Service tests use hand-written fakes. Dir and the snapshot are tested against a real directory
// and SQLite database in dir_test.go. See
// docs/adr/0010-require-unit-tests-and-test-storage-against-real-sqlite.md.

var errFake = errors.New("fake failure")

// fakeDB records the paths it was asked to snapshot to.
type fakeDB struct {
	snapshots []string
	fail      bool
}

func (f *fakeDB) SnapshotTo(_ context.Context, path string) error {
	if f.fail {
		return errFake
	}
	f.snapshots = append(f.snapshots, path)
	return nil
}

// fakeFolder holds backup dates in memory.
type fakeFolder struct {
	dates   []string
	removed []string
}

func (f *fakeFolder) Dates() ([]string, error) {
	out := slices.Clone(f.dates)
	slices.Sort(out)
	return out, nil
}

func (f *fakeFolder) Save(date string, write func(string) error) error {
	if err := write("backups/daily/" + date + ".db" + partialExt); err != nil {
		return err
	}
	f.dates = append(f.dates, date)
	return nil
}

func (f *fakeFolder) Remove(date string) error {
	f.removed = append(f.removed, date)
	f.dates = slices.DeleteFunc(f.dates, func(d string) bool { return d == date })
	return nil
}

// fakeAgentFolder holds agent backups in memory. failRemove makes the next Remove fail.
type fakeAgentFolder struct {
	backups    []AgentBackup
	failRemove bool
}

func (f *fakeAgentFolder) Backups() ([]AgentBackup, error) {
	return slices.Clone(f.backups), nil
}

func (f *fakeAgentFolder) Save(b AgentBackup, write func(string) error) error {
	if err := write(f.Path(b) + partialExt); err != nil {
		return err
	}
	f.backups = append(f.backups, b)
	return nil
}

func (f *fakeAgentFolder) Remove(b AgentBackup) error {
	if f.failRemove {
		f.failRemove = false
		return errFake
	}
	f.backups = slices.DeleteFunc(f.backups, func(o AgentBackup) bool { return o == b })
	return nil
}

func (f *fakeAgentFolder) Path(b AgentBackup) string {
	return "backups/agent/" + b.Name + "." + b.KeptUntil + ".db"
}

// fakeRestorer records the backups it restored. It runs before as the real one does, and restores
// nothing if before fails or fail is set.
type fakeRestorer struct {
	restored []string
	fail     error
}

func (f *fakeRestorer) RestoreFrom(_ context.Context, path string, before func() error) error {
	if f.fail != nil {
		return f.fail
	}
	if err := before(); err != nil {
		return err
	}
	f.restored = append(f.restored, path)
	return nil
}

type harness struct {
	db       *fakeDB
	restorer *fakeRestorer
	folder   *fakeFolder
	agent    *fakeAgentFolder
	svc      *Service
	today    time.Time
}

func newHarness() *harness {
	h := &harness{db: &fakeDB{}, restorer: &fakeRestorer{}, folder: &fakeFolder{}, agent: &fakeAgentFolder{}}
	h.svc = New(h.db, h.restorer, h.folder, h.agent, func() time.Time { return h.today })
	return h
}

func (h *harness) setToday(year int, month time.Month, day int) {
	h.today = time.Date(year, month, day, 9, 30, 0, 0, time.Local)
}

// seedDays adds a daily backup for each of the n days before the day given.
func (h *harness) seedDays(n int, before time.Time) {
	for i := n; i >= 1; i-- {
		h.folder.dates = append(h.folder.dates, before.AddDate(0, 0, -i).Format(dateLayout))
	}
}

func (h *harness) takeDaily(t *testing.T) bool {
	t.Helper()
	taken, err := h.svc.TakeDaily(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return taken
}

// @scenario [#backup-on-first-start]
func TestFirstStartOfTheDayTakesABackup(t *testing.T) {
	h := newHarness()
	// @given [features/daily-backups.feature] today is 2026-10-14
	h.setToday(2026, time.October, 14)
	// @and [features/daily-backups.feature] no backup has been taken today
	h.folder.dates = []string{"2026-10-13"}

	// @when [features/daily-backups.feature] <start>
	// Opening the app calls TakeDaily (TestStartupTakesTheDailyBackup in the main package).
	taken := h.takeDaily(t)

	// @then [features/daily-backups.feature] a backup of my data from 2026-10-14 is taken
	if !taken || !slices.Equal(h.db.snapshots, []string{"backups/daily/2026-10-14.db.partial"}) {
		t.Errorf("taken = %v, snapshots %v, want one for 2026-10-14", taken, h.db.snapshots)
	}
	if !slices.Equal(h.folder.dates, []string{"2026-10-13", "2026-10-14"}) {
		t.Errorf("backups %v", h.folder.dates)
	}
}

// @scenario [#backup-once-per-day]
func TestStartingAgainTheSameDayTakesNoBackup(t *testing.T) {
	h := newHarness()
	// @given [features/daily-backups.feature] today is 2026-10-14
	h.setToday(2026, time.October, 14)
	// @and [features/daily-backups.feature] a backup was taken earlier today
	h.folder.dates = []string{"2026-10-14"}

	// @when [features/daily-backups.feature] I open the app
	taken := h.takeDaily(t)

	// @then [features/daily-backups.feature] no new backup is taken
	if taken || len(h.db.snapshots) != 0 || !slices.Equal(h.folder.dates, []string{"2026-10-14"}) {
		t.Errorf("taken = %v, snapshots %v, backups %v", taken, h.db.snapshots, h.folder.dates)
	}
}

func TestABackupIsTakenAgainTheNextDay(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	h.takeDaily(t)
	h.takeDaily(t)
	h.setToday(2026, time.October, 15)
	h.takeDaily(t)
	if !slices.Equal(h.folder.dates, []string{"2026-10-14", "2026-10-15"}) {
		t.Errorf("backups %v", h.folder.dates)
	}
}

// @scenario [#backup-no-catch-up]
func TestDaysWhenCallbackDidNotRunGetNoBackup(t *testing.T) {
	h := newHarness()
	// @given [features/daily-backups.feature] the last backup was taken on 2026-10-01
	// @and [features/daily-backups.feature] Callback hasn't run since
	h.folder.dates = []string{"2026-10-01"}
	// @and [features/daily-backups.feature] today is 2026-10-14
	h.setToday(2026, time.October, 14)

	// @when [features/daily-backups.feature] I open the app
	h.takeDaily(t)

	// @then [features/daily-backups.feature] one backup is taken, for 2026-10-14
	if len(h.db.snapshots) != 1 || !slices.Equal(h.folder.dates, []string{"2026-10-01", "2026-10-14"}) {
		t.Errorf("snapshots %v, backups %v", h.db.snapshots, h.folder.dates)
	}
}

// @scenario [#backup-keeps-last-15]
func TestASixteenthBackupRemovesTheOldest(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	// @given [features/daily-backups.feature] there are 15 daily backups
	h.seedDays(15, h.today)
	oldest := h.folder.dates[0]

	// @when [features/daily-backups.feature] a new daily backup is taken
	h.takeDaily(t)

	// @then [features/daily-backups.feature] there are still 15 daily backups
	if len(h.folder.dates) != 15 || !slices.Contains(h.folder.dates, "2026-10-14") {
		t.Errorf("backups %v, want 15 ending with 2026-10-14", h.folder.dates)
	}
	// @and [features/daily-backups.feature] the oldest one has been removed
	if !slices.Equal(h.folder.removed, []string{oldest}) {
		t.Errorf("removed %v, want only the oldest, %s", h.folder.removed, oldest)
	}
}

// @scenario [#backup-count-not-days]
func TestBackupsAreKeptByCountNotAge(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	// @given [features/daily-backups.feature] there are 3 daily backups, the oldest from 60 days ago
	h.folder.dates = []string{"2026-08-15", "2026-09-20", "2026-10-10"}

	// @when [features/daily-backups.feature] a new daily backup is taken
	h.takeDaily(t)

	// @then [features/daily-backups.feature] all 4 daily backups are kept
	if len(h.folder.removed) != 0 || len(h.folder.dates) != 4 {
		t.Errorf("removed %v, backups %v", h.folder.removed, h.folder.dates)
	}
}

func TestAFailedSnapshotIsReportedAndNothingIsRemoved(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	h.seedDays(15, h.today)
	h.db.fail = true

	taken, err := h.svc.TakeDaily(context.Background())
	if taken || !errors.Is(err, errFake) {
		t.Errorf("taken = %v, err = %v, want errFake", taken, err)
	}
	if len(h.folder.removed) != 0 || len(h.folder.dates) != 15 {
		t.Errorf("removed %v, backups %v", h.folder.removed, h.folder.dates)
	}
}
