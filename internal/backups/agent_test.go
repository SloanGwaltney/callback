package backups

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"callback/internal/db"
)

// The rules for agent backups, over fakes. What a backup holds and what a restore brings back are
// tested against a real database in agent_dir_test.go.

// makeAgent makes a backup and fails the test if it can't.
func (h *harness) makeAgent(t *testing.T, name string, days int) AgentBackup {
	t.Helper()
	b, err := h.svc.MakeAgent(context.Background(), name, days)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// wantValidation fails the test unless err is a ValidationError saying msg.
func wantValidation(t *testing.T, err error, msg string) {
	t.Helper()
	var invalid *ValidationError
	if !errors.As(err, &invalid) || err.Error() != msg {
		t.Errorf("err = %v, want a ValidationError %q", err, msg)
	}
}

// wantNoBackupMade fails the test if anything was snapshotted.
func (h *harness) wantNoBackupMade(t *testing.T) {
	t.Helper()
	if len(h.db.snapshots) != 0 {
		t.Errorf("snapshots = %v, want none", h.db.snapshots)
	}
}

func (h *harness) list(t *testing.T) []AgentBackup {
	t.Helper()
	listed, err := h.svc.ListAgent()
	if err != nil {
		t.Fatal(err)
	}
	return listed
}

func TestMakeAgentSnapshotsTheData(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)

	b := h.makeAgent(t, "before-cleanup", 7)

	if b != (AgentBackup{Name: "before-cleanup", KeptUntil: "2026-10-21"}) {
		t.Errorf("made %+v", b)
	}
	if !slices.Equal(h.db.snapshots, []string{"backups/agent/before-cleanup.2026-10-21.db.partial"}) {
		t.Errorf("snapshots = %v", h.db.snapshots)
	}
}

// @scenario [#agent-backup-needs-name]
func TestABackupNeedsAName(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)

	// @when [features/agent-backups.feature] my agent tries to make a backup with no name, kept for 7 days
	_, err := h.svc.MakeAgent(context.Background(), "", 7)

	// @then [features/agent-backups.feature] no backup is made
	h.wantNoBackupMade(t)
	// @and [features/agent-backups.feature] my agent is told the backup needs a name
	wantValidation(t, err, "a backup needs a name")
}

// @scenario [#agent-backup-name-characters]
func TestABackupNameMayOnlyUseSomeCharacters(t *testing.T) {
	for _, name := range []string{"../escape", "before cleanup", "notes.db"} {
		t.Run(name, func(t *testing.T) {
			h := newHarness()
			h.setToday(2026, time.October, 14)

			// @when [features/agent-backups.feature] my agent tries to make a backup named "<name>" kept for 7 days
			_, err := h.svc.MakeAgent(context.Background(), name, 7)

			// @then [features/agent-backups.feature] no backup is made
			h.wantNoBackupMade(t)
			// @and [features/agent-backups.feature] my agent is told which characters a backup name may use
			wantValidation(t, err, "a backup name may only use letters, numbers, hyphens, and underscores")
		})
	}
}

func TestABackupNameMayUseLettersNumbersHyphensAndUnderscores(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	if b := h.makeAgent(t, "Before_Cleanup-2", 7); b.Name != "Before_Cleanup-2" {
		t.Errorf("made %+v", b)
	}
}

// @scenario [#agent-backup-retention-range]
func TestABackupIsKeptForOneToNinetyDays(t *testing.T) {
	for _, tc := range []struct {
		days      int
		keptUntil string
	}{
		{1, "2026-10-15"},
		{90, "2027-01-12"},
		{0, ""},
		{91, ""},
	} {
		t.Run(fmt.Sprint(tc.days), func(t *testing.T) {
			h := newHarness()
			// @given [features/agent-backups.feature] today is 2026-10-14
			h.setToday(2026, time.October, 14)

			// @when [features/agent-backups.feature] my agent tries to make a backup named "before-cleanup" kept for <days> days
			b, err := h.svc.MakeAgent(context.Background(), "before-cleanup", tc.days)

			// @then [features/agent-backups.feature] <outcome>
			if tc.keptUntil != "" {
				if err != nil || b.KeptUntil != tc.keptUntil {
					t.Errorf("made %+v, %v, want it kept until %s", b, err, tc.keptUntil)
				}
				return
			}
			h.wantNoBackupMade(t)
			wantValidation(t, err, "use 1 to 90 days for how long a backup is kept")
		})
	}
}

// @scenario [#agent-backup-name-in-use]
func TestABackupCantReuseTheNameOfOneStillKept(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	// @given [features/agent-backups.feature] another agent made a backup named "before-cleanup" that is still kept
	existing := AgentBackup{Name: "before-cleanup", KeptUntil: "2026-10-20"}
	h.agent.backups = []AgentBackup{existing}

	// @when [features/agent-backups.feature] my agent tries to make a backup named "before-cleanup" kept for 7 days
	_, err := h.svc.MakeAgent(context.Background(), "before-cleanup", 7)

	// @then [features/agent-backups.feature] no backup is made
	h.wantNoBackupMade(t)
	// @and [features/agent-backups.feature] the existing "before-cleanup" backup is unchanged
	if !slices.Equal(h.agent.backups, []AgentBackup{existing}) {
		t.Errorf("backups = %v, want only the existing one", h.agent.backups)
	}
	// @and [features/agent-backups.feature] my agent is told that name is already taken
	wantValidation(t, err, `the name "before-cleanup" is already taken by a backup kept until 2026-10-20; choose another name`)
}

// The data the backup holds is checked against a real database in
// TestAgentBackupHoldsTheDataAsItIsNow.
// @scenario [#agent-backup-name-freed]
func TestAnExpiredBackupsNameCanBeUsedAgain(t *testing.T) {
	h := newHarness()
	// @given [features/agent-backups.feature] today is 2026-10-22
	h.setToday(2026, time.October, 22)
	// @and [features/agent-backups.feature] my agent made a backup named "before-cleanup" that was kept until 2026-10-21
	h.agent.backups = []AgentBackup{{Name: "before-cleanup", KeptUntil: "2026-10-21"}}

	// @when [features/agent-backups.feature] my agent makes a backup named "before-cleanup" kept for 7 days
	b, err := h.svc.MakeAgent(context.Background(), "before-cleanup", 7)

	// @then [features/agent-backups.feature] a backup named "before-cleanup" holds my data as it is now
	if err != nil || !slices.Equal(h.db.snapshots, []string{"backups/agent/before-cleanup.2026-10-29.db.partial"}) {
		t.Errorf("snapshots = %v, %v, want one of the data now", h.db.snapshots, err)
	}
	// @and [features/agent-backups.feature] the "before-cleanup" backup is kept until 2026-10-29
	if b.KeptUntil != "2026-10-29" {
		t.Errorf("kept until %s", b.KeptUntil)
	}
}

func TestListAgentIsByKeptUntilThenNameAndLeavesOutExpired(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	h.agent.backups = []AgentBackup{
		{Name: "zeta", KeptUntil: "2026-10-15"},
		{Name: "expired", KeptUntil: "2026-10-13"},
		{Name: "beta", KeptUntil: "2026-10-20"},
		{Name: "alpha", KeptUntil: "2026-10-15"},
	}

	got := h.list(t)

	want := []AgentBackup{{"alpha", "2026-10-15"}, {"zeta", "2026-10-15"}, {"beta", "2026-10-20"}}
	if !slices.Equal(got, want) {
		t.Errorf("listed %v, want %v", got, want)
	}
}

// @scenario [#agent-backup-kept-through-last-day]
func TestABackupIsKeptThroughItsLastDay(t *testing.T) {
	h := newHarness()
	// @given [features/agent-backups.feature] today is 2026-10-21
	h.setToday(2026, time.October, 21)
	// @and [features/agent-backups.feature] my agent made a backup named "before-cleanup" kept until 2026-10-21
	b := AgentBackup{Name: "before-cleanup", KeptUntil: "2026-10-21"}
	h.agent.backups = []AgentBackup{b}

	// @when [features/agent-backups.feature] I open the app
	// Opening the app calls RemoveExpired (TestStartupRemovesExpiredAgentBackups in the main package).
	if err := h.svc.RemoveExpired(); err != nil {
		t.Fatal(err)
	}

	// @then [features/agent-backups.feature] the "before-cleanup" backup is still kept
	if !slices.Equal(h.agent.backups, []AgentBackup{b}) || !slices.Equal(h.list(t), []AgentBackup{b}) {
		t.Errorf("backups = %v, listed %v, want before-cleanup kept", h.agent.backups, h.list(t))
	}
}

// @scenario [#agent-backup-expires]
func TestAnExpiredBackupIsRemoved(t *testing.T) {
	h := newHarness()
	// @given [features/agent-backups.feature] today is 2026-10-22
	h.setToday(2026, time.October, 22)
	// @and [features/agent-backups.feature] my agent made a backup named "before-cleanup" kept until 2026-10-21
	kept := AgentBackup{Name: "other", KeptUntil: "2026-10-22"}
	h.agent.backups = []AgentBackup{{Name: "before-cleanup", KeptUntil: "2026-10-21"}, kept}

	// @when [features/agent-backups.feature] <run>
	// Opening the app and running a command both call RemoveExpired
	// (TestStartupRemovesExpiredAgentBackups in the main package, and
	// TestEveryCommandRemovesExpiredBackups in internal/cli).
	if err := h.svc.RemoveExpired(); err != nil {
		t.Fatal(err)
	}

	// @then [features/agent-backups.feature] there is no backup named "before-cleanup"
	if !slices.Equal(h.agent.backups, []AgentBackup{kept}) {
		t.Errorf("backups = %v, want only %v", h.agent.backups, kept)
	}
}

// The app opening as usual is TestStartupStillOpensWhenExpiredBackupsCantBeRemoved in the main
// package.
// @scenario [#agent-backup-expiry-retries]
func TestAnExpiredBackupThatCantBeRemovedIsRemovedLater(t *testing.T) {
	h := newHarness()
	// @given [features/agent-backups.feature] today is 2026-10-22
	h.setToday(2026, time.October, 22)
	// @and [features/agent-backups.feature] my agent made a backup named "before-cleanup" kept until 2026-10-21
	expired := AgentBackup{Name: "before-cleanup", KeptUntil: "2026-10-21"}
	other := AgentBackup{Name: "also-expired", KeptUntil: "2026-10-20"}
	h.agent.backups = []AgentBackup{expired, other}
	// @and [features/agent-backups.feature] the "before-cleanup" backup can't be removed right now
	h.agent.failRemove = true

	// @when [features/agent-backups.feature] I open the app
	err := h.svc.RemoveExpired()

	if !errors.Is(err, errFake) {
		t.Errorf("err = %v, want the failed removal", err)
	}
	if !slices.Equal(h.agent.backups, []AgentBackup{expired}) {
		t.Errorf("backups = %v, want the other expired one removed anyway", h.agent.backups)
	}
	// @and [features/agent-backups.feature] the "before-cleanup" backup is removed the next time Callback runs and can remove it
	if err := h.svc.RemoveExpired(); err != nil {
		t.Fatal(err)
	}
	if len(h.agent.backups) != 0 {
		t.Errorf("backups = %v, want none", h.agent.backups)
	}
}

// @scenario [#agent-restore-daily-backup]
func TestAgentCantRestoreADailyBackup(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	// @given [features/agent-backups.feature] a daily backup was taken today
	h.takeDaily(t)
	h.db.snapshots = nil

	// @when [features/agent-backups.feature] my agent tries to restore the daily backup
	_, err := h.svc.Restore(context.Background(), "2026-10-14")

	// @then [features/agent-backups.feature] my data is unchanged
	if len(h.restorer.restored) != 0 || len(h.db.snapshots) != 0 {
		t.Errorf("restored %v, snapshots %v, want nothing", h.restorer.restored, h.db.snapshots)
	}
	// @and [features/agent-backups.feature] my agent is told it can only restore backups an agent made
	wantValidation(t, err, `"2026-10-14" is a daily backup; you can only restore a backup an agent made`)
}

func TestAnAgentBackupNamedLikeADateCanBeRestored(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	h.takeDaily(t)
	b := h.makeAgent(t, "2026-10-14", 7)

	if _, err := h.svc.Restore(context.Background(), "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.restorer.restored, []string{h.agent.Path(b)}) {
		t.Errorf("restored %v, want the agent backup", h.restorer.restored)
	}
}

// @scenario [#agent-restore-unknown]
func TestRestoringABackupThatDoesntExist(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	h.makeAgent(t, "before-cleanup", 7)
	h.db.snapshots = nil

	// @when [features/agent-backups.feature] my agent tries to restore a backup named "never-made"
	_, err := h.svc.Restore(context.Background(), "never-made")

	// @then [features/agent-backups.feature] my data is unchanged
	if len(h.restorer.restored) != 0 || len(h.db.snapshots) != 0 {
		t.Errorf("restored %v, snapshots %v, want nothing", h.restorer.restored, h.db.snapshots)
	}
	// @and [features/agent-backups.feature] my agent is told there's no backup with that name
	var missing *NotFoundError
	if !errors.As(err, &missing) || err.Error() != `there's no backup named "never-made"` {
		t.Errorf("err = %v, want a NotFoundError", err)
	}
}

func TestAnExpiredBackupCantBeRestored(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 22)
	h.agent.backups = []AgentBackup{{Name: "before-cleanup", KeptUntil: "2026-10-21"}}

	_, err := h.svc.Restore(context.Background(), "before-cleanup")

	var missing *NotFoundError
	if !errors.As(err, &missing) || len(h.restorer.restored) != 0 {
		t.Errorf("err = %v, restored %v, want a NotFoundError and nothing restored", err, h.restorer.restored)
	}
}

// What the restore brings back is TestRestoreBringsBackTheData, against a real database.
// @scenario [#restore-backs-up-first]
func TestRestoringBacksUpTheDataFirst(t *testing.T) {
	h := newHarness()
	// @given [features/agent-backups.feature] today is 2026-10-14
	h.today = time.Date(2026, time.October, 14, 9, 30, 15, 0, time.Local)
	// @and [features/agent-backups.feature] my agent made a backup named "before-cleanup"
	b := h.makeAgent(t, "before-cleanup", 3)
	h.db.snapshots = nil

	// @when [features/agent-backups.feature] my agent restores the backup "before-cleanup"
	undo, err := h.svc.Restore(context.Background(), "before-cleanup")
	if err != nil {
		t.Fatal(err)
	}

	// @then [features/agent-backups.feature] a backup of my data from just before the restore is made first
	// The fake restorer runs before, which snapshots the data, before it restores.
	if !slices.Equal(h.db.snapshots, []string{"backups/agent/before-restore-20261014-093015.2026-10-21.db.partial"}) {
		t.Errorf("snapshots = %v, want the data from just before", h.db.snapshots)
	}
	if !slices.Equal(h.restorer.restored, []string{h.agent.Path(b)}) {
		t.Errorf("restored %v", h.restorer.restored)
	}
	// @and [features/agent-backups.feature] that backup is kept until 2026-10-21
	if undo.KeptUntil != "2026-10-21" {
		t.Errorf("kept until %s", undo.KeptUntil)
	}
	// @and [features/agent-backups.feature] it's listed with the other backups agents made
	if want := []AgentBackup{b, undo}; !slices.Equal(h.list(t), want) {
		t.Errorf("listed %v, want %v", h.list(t), want)
	}
	// @and [features/agent-backups.feature] my agent is told that backup's name
	// The command line prints the name it's given (TestAgentRestoresABackup in internal/cli).
	if undo.Name != "before-restore-20261014-093015" {
		t.Errorf("name = %q", undo.Name)
	}
}

func TestASecondRestoreInTheSameSecondGetsItsOwnName(t *testing.T) {
	h := newHarness()
	h.today = time.Date(2026, time.October, 14, 9, 30, 15, 0, time.Local)
	h.makeAgent(t, "before-cleanup", 7)

	first, err := h.svc.Restore(context.Background(), "before-cleanup")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.svc.Restore(context.Background(), "before-cleanup")
	if err != nil {
		t.Fatal(err)
	}

	if first.Name != "before-restore-20261014-093015" || second.Name != "before-restore-20261014-093015-2" {
		t.Errorf("names = %q, %q", first.Name, second.Name)
	}
}

func TestARestoreThatFailsMakesNoUndoBackup(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	h.makeAgent(t, "from-newer", 7)
	h.db.snapshots = nil
	h.restorer.fail = db.ErrNewerBackup

	_, err := h.svc.Restore(context.Background(), "from-newer")

	if !errors.Is(err, db.ErrNewerBackup) {
		t.Errorf("err = %v, want ErrNewerBackup", err)
	}
	if len(h.db.snapshots) != 0 || len(h.list(t)) != 1 {
		t.Errorf("snapshots %v, listed %v, want no undo backup", h.db.snapshots, h.list(t))
	}
}

func TestARestoreWhoseUndoBackupFailsRestoresNothing(t *testing.T) {
	h := newHarness()
	h.setToday(2026, time.October, 14)
	h.makeAgent(t, "before-cleanup", 7)
	h.db.fail = true

	if _, err := h.svc.Restore(context.Background(), "before-cleanup"); !errors.Is(err, errFake) {
		t.Errorf("err = %v, want the failed snapshot", err)
	}
	if len(h.restorer.restored) != 0 {
		t.Errorf("restored %v, want nothing", h.restorer.restored)
	}
}
