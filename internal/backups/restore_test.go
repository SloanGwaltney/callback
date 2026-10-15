package backups

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"callback/internal/db"
)

// The rules for listing every backup and restoring any of them from the app, over fakes. What a
// restore brings back is tested against a real database in restore_dir_test.go.

func (h *harness) listAll(t *testing.T) []Backup {
	t.Helper()
	listed, err := h.svc.List()
	if err != nil {
		t.Fatal(err)
	}
	return listed
}

// names returns each backup's kind and name, like "daily 2026-10-12".
func names(listed []Backup) []string {
	var out []string
	for _, b := range listed {
		out = append(out, string(b.Kind)+" "+b.Name)
	}
	return out
}

// at returns 9:30 on a day in October 2026, plus offset.
func at(day int, offset time.Duration) time.Time {
	return time.Date(2026, time.October, day, 9, 30, 0, 0, time.Local).Add(offset)
}

// wantRestoreError fails the test unless err is a RestoreError saying msg.
func wantRestoreError(t *testing.T, err error, msg string) {
	t.Helper()
	var failed *RestoreError
	if !errors.As(err, &failed) || err.Error() != msg {
		t.Errorf("err = %v, want a RestoreError %q", err, msg)
	}
}

// wantNothingRestored fails the test if anything was restored or backed up.
func (h *harness) wantNothingRestored(t *testing.T) {
	t.Helper()
	if len(h.restorer.restored) != 0 || len(h.db.snapshots) != 0 {
		t.Errorf("restored %v, snapshots %v, want nothing", h.restorer.restored, h.db.snapshots)
	}
}

// @scenario [#app-backups-newest-first]
func TestDailyBackupsAreListedNewestFirst(t *testing.T) {
	h := newHarness()
	// @given [features/restore-backups.feature] daily backups were taken on 2026-10-12, 2026-10-14, and 2026-10-13
	h.folder.dates = []string{"2026-10-12", "2026-10-14", "2026-10-13"}

	// @when [features/restore-backups.feature] I open my backups
	// Opening them calls ListBackups (BackupList.test.tsx in the frontend).
	listed := h.listAll(t)

	// @then [features/restore-backups.feature] I see the backups from 2026-10-14, 2026-10-13, and 2026-10-12, in that order
	want := []string{"daily 2026-10-14", "daily 2026-10-13", "daily 2026-10-12"}
	if got := names(listed); !slices.Equal(got, want) {
		t.Errorf("listed %v, want %v", got, want)
	}
}

// @scenario [#app-backups-none]
func TestNoBackupsListsNone(t *testing.T) {
	// @given [features/restore-backups.feature] no backups have been taken
	h := newHarness()

	listed := h.listAll(t)

	// An empty list, not nil, so the frontend gets [] rather than null.
	if listed == nil || len(listed) != 0 {
		t.Errorf("listed %#v, want an empty list", listed)
	}
}

// @scenario [#app-backups-list-every-kind] [#agent-list-all-backups]
func TestEveryKindOfBackupIsListedNewestFirst(t *testing.T) {
	h := newHarness()
	// @given [features/restore-backups.feature] a daily backup was taken on 2026-10-12
	// @given [features/agent-backups.feature] a daily backup was taken on 2026-10-12
	h.today = at(12, 0)
	h.takeDaily(t)
	// @and [features/restore-backups.feature] my agent made a backup named "before-cleanup" on 2026-10-13
	// @and [features/agent-backups.feature] an agent made a backup named "before-cleanup" on 2026-10-13
	h.today = at(13, 0)
	h.makeAgent(t, "before-cleanup", 7)
	// @and [features/restore-backups.feature] I restored a backup on 2026-10-14, so a backup was made before restoring it
	// @and [features/agent-backups.feature] I restored a backup in the app on 2026-10-14, so a backup was made before restoring it
	h.today = at(14, 0)
	if _, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12"); err != nil {
		t.Fatal(err)
	}

	// @when [features/restore-backups.feature] I open my backups
	// @when [features/agent-backups.feature] my agent lists the backups
	listed := h.listAll(t)

	// @then [features/restore-backups.feature] I see all 3 backups, newest first
	// @then [features/agent-backups.feature] it sees all 3 backups
	// @and [features/restore-backups.feature] each says whether it's a daily backup, made by my agent, or made before a restore
	// @and [features/agent-backups.feature] each says whether it's a daily backup, made by an agent, or made by the app before a restore
	// The app and the command line each say so for the kind (BackupList.test.tsx, and
	// TestAgentListsEveryBackup in internal/cli).
	want := []string{"app 20261014-093000_daily_2026-10-12", "agent before-cleanup", "daily 2026-10-12"}
	if got := names(listed); !slices.Equal(got, want) {
		t.Fatalf("listed %v, want %v", got, want)
	}
	// @and [features/restore-backups.feature] the backup from my agent shows its name "before-cleanup"
	if listed[1].Name != "before-cleanup" {
		t.Errorf("agent backup named %q", listed[1].Name)
	}
	// @and [features/restore-backups.feature] the backups made by my agent or before a restore show the day each is kept until
	// @and [features/agent-backups.feature] the backups made by an agent or before a restore show the day each is kept until
	if listed[0].KeptUntil != "2026-10-21" || listed[1].KeptUntil != "2026-10-20" || listed[2].KeptUntil != "" {
		t.Errorf("kept until %q, %q, %q", listed[0].KeptUntil, listed[1].KeptUntil, listed[2].KeptUntil)
	}
	if listed[0].Made != at(14, 0).Format(time.RFC3339) {
		t.Errorf("made %q", listed[0].Made)
	}
}

func TestListLeavesOutExpiredAgentAndAppBackups(t *testing.T) {
	h := newHarness()
	h.today = at(22, 0)
	h.agent.backups = []AgentBackup{{Name: "old", KeptUntil: "2026-10-21"}, {Name: "kept", KeptUntil: "2026-10-22"}}
	h.app.backups = []AgentBackup{{Name: "20261010-093000_daily_2026-10-09", KeptUntil: "2026-10-17"}}

	if got := names(h.listAll(t)); !slices.Equal(got, []string{"agent kept"}) {
		t.Errorf("listed %v, want only the kept agent backup", got)
	}
}

// @scenario [#app-restore-keeps-daily-backups]
func TestRestoringFromTheAppKeepsTheDailyBackups(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)
	// @given [features/restore-backups.feature] daily backups were taken on 2026-10-12, 2026-10-13, and 2026-10-14
	h.folder.dates = []string{"2026-10-12", "2026-10-13", "2026-10-14"}

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	if _, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12"); err != nil {
		t.Fatal(err)
	}

	// @then [features/restore-backups.feature] I still see the backups from 2026-10-14, 2026-10-13, and 2026-10-12
	var daily []string
	for _, b := range h.listAll(t) {
		if b.Kind == Daily {
			daily = append(daily, b.Name)
		}
	}
	if want := []string{"2026-10-14", "2026-10-13", "2026-10-12"}; !slices.Equal(daily, want) {
		t.Errorf("daily backups %v, want %v", daily, want)
	}
	if !slices.Equal(h.restorer.restored, []string{"backups/daily/2026-10-12.db"}) {
		t.Errorf("restored %v", h.restorer.restored)
	}
}

// @scenario [#app-restore-backs-up-first]
func TestRestoringFromTheAppBacksUpTheDataFirst(t *testing.T) {
	h := newHarness()
	// @given [features/restore-backups.feature] today is 2026-10-14
	h.today = at(14, 15*time.Second)
	// @and [features/restore-backups.feature] a daily backup was taken on 2026-10-12
	h.folder.dates = []string{"2026-10-12"}

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	undo, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/restore-backups.feature] a backup of my data from just before the restore is made first
	// The fake restorer runs before, which snapshots the data, before it restores.
	if !slices.Equal(h.db.snapshots, []string{"backups/app/20261014-093015_daily_2026-10-12.2026-10-21.db.partial"}) {
		t.Errorf("snapshots = %v, want the data from just before, in the app's folder", h.db.snapshots)
	}
	if len(h.agent.backups) != 0 {
		t.Errorf("agent backups %v, want none", h.agent.backups)
	}
	// @and [features/restore-backups.feature] that backup is kept until 2026-10-21
	if undo.KeptUntil != "2026-10-21" {
		t.Errorf("kept until %s", undo.KeptUntil)
	}
	// @and [features/restore-backups.feature] it's listed with my backups as made before restoring 2026-10-12
	// The page words it (BackupList.test.tsx).
	listed := h.listAll(t)
	if len(listed) != 2 || listed[0].Name != undo.Name || *listed[0].Restored != (Restored{Kind: Daily, Name: "2026-10-12"}) {
		t.Errorf("listed %+v, want the undo backup first, made before restoring 2026-10-12", listed)
	}
	if undo.Kind != App || *undo.Restored != (Restored{Kind: Daily, Name: "2026-10-12"}) {
		t.Errorf("undo = %+v", undo)
	}
}

func TestUndoingAnUndoNamesTheEarlierUndoByWhenItWasMade(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)
	h.folder.dates = []string{"2026-10-12"}
	first, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}

	h.today = at(14, time.Minute)
	second, err := h.svc.RestoreBackup(context.Background(), App, first.Name)
	if err != nil {
		t.Fatal(err)
	}

	if second.Name != "20261014-093100_app_20261014-093000" || *second.Restored != (Restored{Kind: App, Name: "20261014-093000"}) {
		t.Errorf("second undo = %+v", second)
	}
	if !slices.Equal(h.restorer.restored, []string{"backups/daily/2026-10-12.db", "backups/app/" + first.Name + ".2026-10-21.db"}) {
		t.Errorf("restored %v", h.restorer.restored)
	}
}

func TestTwoRestoresFromTheAppInTheSameSecondGetTheirOwnNames(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)
	h.folder.dates = []string{"2026-10-12"}

	first, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}

	if first.Name != "20261014-093000_daily_2026-10-12" || second.Name != "20261014-093000-2_daily_2026-10-12" {
		t.Errorf("names = %q, %q", first.Name, second.Name)
	}
	if *second.Restored != (Restored{Kind: Daily, Name: "2026-10-12"}) {
		t.Errorf("second restored %+v", second.Restored)
	}
}

// @scenario [#app-pre-restore-backup-not-daily]
func TestTheBackupBeforeARestoreIsntRotatedOut(t *testing.T) {
	h := newHarness()
	h.today = at(15, 0)
	// @given [features/restore-backups.feature] there are 15 daily backups
	h.seedDays(Keep, h.today)
	// @and [features/restore-backups.feature] I restored one of them
	undo, err := h.svc.RestoreBackup(context.Background(), Daily, h.folder.dates[0])
	if err != nil {
		t.Fatal(err)
	}

	// @when [features/restore-backups.feature] a new daily backup is taken
	h.takeDaily(t)

	// @then [features/restore-backups.feature] the backup made before that restore is still kept
	if !slices.ContainsFunc(h.listAll(t), func(b Backup) bool { return b.Kind == App && b.Name == undo.Name }) {
		t.Errorf("listed %v, want %s still kept", names(h.listAll(t)), undo.Name)
	}
	if len(h.folder.dates) != Keep {
		t.Errorf("daily backups %v, want %d", h.folder.dates, Keep)
	}
}

// That the restore checks the backup's version is TestRestoreRefusesANewerBackup in internal/db.
// @scenario [#app-restore-newer-version]
func TestTheAppIsToldABackupIsFromANewerVersion(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)
	h.folder.dates = []string{"2026-10-12"}
	h.restorer.fail = db.ErrNewerBackup

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	_, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")

	// @then [features/restore-backups.feature#app-restore-newer-version] my data is unchanged
	h.wantNothingRestored(t)
	// @and [features/restore-backups.feature] I'm told the backup is from a newer version of Callback, and to update Callback to restore it
	wantRestoreError(t, err, "That backup is from a newer version of Callback, so it can't be restored. Update Callback to restore it. Your data is unchanged.")
	if !errors.Is(err, db.ErrNewerBackup) {
		t.Errorf("err = %v, want it to wrap ErrNewerBackup", err)
	}
}

// That a damaged backup is refused before anything changes is
// TestRestoreFromADamagedBackupChangesNothing in internal/db.
// @scenario [#app-restore-unreadable]
func TestTheAppIsToldABackupCouldntBeRestored(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)
	h.folder.dates = []string{"2026-10-12"}
	h.restorer.fail = errors.New("db: file is not a database")

	// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
	_, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")

	// @then [features/restore-backups.feature#app-restore-unreadable] my data is unchanged
	h.wantNothingRestored(t)
	// @and [features/restore-backups.feature] I'm told the backup couldn't be restored
	wantRestoreError(t, err, "The backup couldn't be restored. Your data is unchanged.")
}

// @scenario [#app-restore-missing]
func TestTheAppIsToldABackupNoLongerExists(t *testing.T) {
	for _, tc := range []struct {
		kind Kind
		name string
	}{
		{Daily, "2026-10-12"},
		{Agent, "before-cleanup"},
		{App, "20261013-093000_daily_2026-10-12"},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			h := newHarness()
			h.today = at(14, 0)
			// @and [features/restore-backups.feature] the backup from 2026-10-12 has since been removed
			h.folder.dates = []string{"2026-10-13"}

			// @when [features/restore-backups.feature] I restore the backup from 2026-10-12 and confirm
			_, err := h.svc.RestoreBackup(context.Background(), tc.kind, tc.name)

			// @then [features/restore-backups.feature#app-restore-missing] my data is unchanged
			h.wantNothingRestored(t)
			// @and [features/restore-backups.feature] I'm told that backup no longer exists
			wantRestoreError(t, err, "That backup no longer exists. It may have been removed since you opened your backups. Your data is unchanged.")
			var missing *NotFoundError
			if !errors.As(err, &missing) {
				t.Errorf("err = %v, want it to wrap a NotFoundError", err)
			}
		})
	}
}

func TestAnExpiredBackupCantBeRestoredFromTheApp(t *testing.T) {
	h := newHarness()
	h.today = at(22, 0)
	h.agent.backups = []AgentBackup{{Name: "before-cleanup", KeptUntil: "2026-10-21"}}

	_, err := h.svc.RestoreBackup(context.Background(), Agent, "before-cleanup")

	h.wantNothingRestored(t)
	var missing *NotFoundError
	if !errors.As(err, &missing) {
		t.Errorf("err = %v, want it to wrap a NotFoundError", err)
	}
}

func TestAnUnknownKindOfBackupCantBeRestored(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)

	_, err := h.svc.RestoreBackup(context.Background(), "weekly", "2026-10-12")

	h.wantNothingRestored(t)
	wantValidation(t, err, `there's no kind of backup called "weekly"`)
}

func TestAnAppRestoreWhoseUndoBackupFailsRestoresNothing(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)
	h.folder.dates = []string{"2026-10-12"}
	h.db.fail = true

	_, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")

	wantRestoreError(t, err, "The backup couldn't be restored. Your data is unchanged.")
	if len(h.restorer.restored) != 0 || len(h.app.backups) != 0 {
		t.Errorf("restored %v, app backups %v, want nothing", h.restorer.restored, h.app.backups)
	}
}

// @scenario [#agent-restore-app-backup]
func TestAgentCantRestoreABackupTheAppMade(t *testing.T) {
	h := newHarness()
	h.today = at(14, 0)
	h.folder.dates = []string{"2026-10-12"}
	// @given [features/agent-backups.feature] I restored a backup in the app, so a backup was made before restoring it
	undo, err := h.svc.RestoreBackup(context.Background(), Daily, "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}
	h.restorer.restored, h.db.snapshots = nil, nil

	// @when [features/agent-backups.feature] my agent tries to restore the backup the app made
	_, err = h.svc.Restore(context.Background(), undo.Name)

	// @then [features/agent-backups.feature#agent-restore-app-backup] my data is unchanged
	h.wantNothingRestored(t)
	// @and [features/agent-backups.feature] my agent is told it can only restore backups an agent made
	wantValidation(t, err, `"20261014-093000_daily_2026-10-12" was made by the app before a restore; you can only restore a backup an agent made`)
}

func TestRemovingExpiredBackupsRemovesExpiredAppBackups(t *testing.T) {
	h := newHarness()
	h.today = at(22, 0)
	kept := AgentBackup{Name: "20261015-093000_daily_2026-10-12", KeptUntil: "2026-10-22"}
	h.app.backups = []AgentBackup{{Name: "20261014-093000_daily_2026-10-12", KeptUntil: "2026-10-21"}, kept}

	if err := h.svc.RemoveExpired(); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(h.app.backups, []AgentBackup{kept}) {
		t.Errorf("app backups %v, want only %v", h.app.backups, kept)
	}
}
