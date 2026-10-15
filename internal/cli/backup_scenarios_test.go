package cli

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"

	"callback/internal/backups"
	"callback/internal/db"
)

// The scenarios in features/agent-backups.feature, as the command line carries them out. The fake
// backups service keeps what it's given; the rules for names and days, which backups are kept,
// and what a restore brings back are the backups service's, tested in internal/backups, and the
// restore itself is internal/db's.

// The messages for names and days the rules don't allow are the backups service's
// (TestABackupNameMayOnlyUseSomeCharacters, TestABackupIsKeptForOneToNinetyDays, and
// TestABackupCantReuseTheNameOfOneStillKept in internal/backups). The fake hands them back as the
// service would.
var (
	errNameCharacters = &backups.ValidationError{Msg: "a backup name may only use letters, numbers, hyphens, and underscores"}
	errDaysRange      = &backups.ValidationError{Msg: "use 1 to 90 days for how long a backup is kept"}
	errNameTaken      = &backups.ValidationError{Msg: `the name "before-cleanup" is already taken by a backup kept until 2026-10-20; choose another name`}
)

// wantError fails the test unless r wrote exactly msg as its error.
func (r result) wantError(t *testing.T, msg string) {
	t.Helper()
	if r.stderr != "callback-cli: "+msg+"\n" {
		t.Errorf("stderr = %q, want %q", r.stderr, msg)
	}
}

// What the backup holds is TestAgentBackupHoldsTheDataAsItIsNow in internal/backups.
// @scenario [#agent-backup]
func TestAgentMakesABackup(t *testing.T) {
	h := newHarness()

	// @when [features/agent-backups.feature] my agent makes a backup named "before-cleanup" kept for 7 days
	r := h.run("backup", "make", "before-cleanup", "--days", "7")

	r.wantCode(t, ExitOK)
	// @and [features/agent-backups.feature] the "before-cleanup" backup is kept until 2026-10-21
	if r.stdout != "Made the backup before-cleanup, kept until 2026-10-21.\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
	if !slices.Equal(h.backups.agent, []backups.AgentBackup{{Name: "before-cleanup", KeptUntil: "2026-10-21"}}) {
		t.Errorf("backups = %v", h.backups.agent)
	}
}

func TestMakeABackupAsJSON(t *testing.T) {
	r := newHarness().run("backup", "make", "before-cleanup", "--days", "7", "--json")

	var got backups.AgentBackup
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil || got != (backups.AgentBackup{Name: "before-cleanup", KeptUntil: "2026-10-21"}) {
		t.Errorf("JSON = %+v, %v\n%s", got, err, r.stdout)
	}
}

// @scenario [#agent-backup-needs-name]
func TestABackupNeedsANameFromTheCommandLine(t *testing.T) {
	h := newHarness()

	// @when [features/agent-backups.feature] my agent tries to make a backup with no name, kept for 7 days
	r := h.run("backup", "make", "--days", "7")

	r.wantCode(t, ExitUsage)
	// @then [features/agent-backups.feature] no backup is made
	if len(h.backups.agent) != 0 || h.opened != 0 {
		t.Errorf("backups = %v, opened %d", h.backups.agent, h.opened)
	}
	// @and [features/agent-backups.feature] my agent is told the backup needs a name
	r.wantError(t, "`callback-cli backup make` needs a backup name, like `callback-cli backup make before-cleanup`")
}

// @scenario [#agent-backup-name-characters]
func TestABackupNameTheRulesDontAllow(t *testing.T) {
	h := newHarness()
	h.backups.fail = errNameCharacters

	// @when [features/agent-backups.feature] my agent tries to make a backup named "<name>" kept for 7 days
	r := h.run("backup", "make", "before cleanup", "--days", "7")

	r.wantCode(t, ExitValidation)
	r.wantError(t, errNameCharacters.Msg)
}

// @scenario [#agent-backup-needs-retention]
func TestABackupNeedsANumberOfDays(t *testing.T) {
	h := newHarness()

	// @when [features/agent-backups.feature] my agent tries to make a backup named "before-cleanup" without saying how many days to keep it
	r := h.run("backup", "make", "before-cleanup")

	r.wantCode(t, ExitUsage)
	// @then [features/agent-backups.feature] no backup is made
	if len(h.backups.agent) != 0 {
		t.Errorf("backups = %v", h.backups.agent)
	}
	// @and [features/agent-backups.feature] my agent is told the backup needs a number of days from 1 to 90
	r.wantError(t, "`backup make` needs --days with the number of days to keep the backup, from 1 to 90")
}

func TestDaysMustBeAWholeNumber(t *testing.T) {
	h := newHarness()

	r := h.run("backup", "make", "before-cleanup", "--days", "a week")

	r.wantCode(t, ExitValidation)
	r.wantError(t, `"a week" isn't a number of days; use a whole number from 1 to 90`)
	if len(h.backups.agent) != 0 {
		t.Errorf("backups = %v", h.backups.agent)
	}
}

// @scenario [#agent-backup-retention-range]
func TestDaysOutsideTheRange(t *testing.T) {
	h := newHarness()
	h.backups.fail = errDaysRange

	// @when [features/agent-backups.feature] my agent tries to make a backup named "before-cleanup" kept for <days> days
	r := h.run("backup", "make", "before-cleanup", "--days", "91")

	r.wantCode(t, ExitValidation)
	r.wantError(t, errDaysRange.Msg)
}

// @scenario [#agent-backup-name-in-use]
func TestATakenBackupName(t *testing.T) {
	h := newHarness()
	h.backups.fail = errNameTaken

	// @when [features/agent-backups.feature] my agent tries to make a backup named "before-cleanup" kept for 7 days
	r := h.run("backup", "make", "before-cleanup", "--days", "7")

	r.wantCode(t, ExitValidation)
	// @and [features/agent-backups.feature] my agent is told that name is already taken
	r.wantError(t, errNameTaken.Msg)
}

// Which backups are listed, and in what order, is TestEveryKindOfBackupIsListedNewestFirst in
// internal/backups.
// @scenario [#agent-list-all-backups]
func TestAgentListsEveryBackup(t *testing.T) {
	h := newHarness()
	// The service lists them newest first.
	h.backups.listed = []backups.Backup{
		// @and [features/agent-backups.feature] I restored a backup in the app on 2026-10-14, so a backup was made before restoring it
		{
			Kind: backups.App, Name: "20261014-093000_daily_2026-10-12", KeptUntil: "2026-10-21",
			Restored: &backups.Restored{Kind: backups.Daily, Name: "2026-10-12"},
		},
		// @and [features/agent-backups.feature] an agent made a backup named "before-cleanup" on 2026-10-13
		{Kind: backups.Agent, Name: "before-cleanup", KeptUntil: "2026-10-20"},
		// @given [features/agent-backups.feature] a daily backup was taken on 2026-10-12
		{Kind: backups.Daily, Name: "2026-10-12"},
	}

	// @when [features/agent-backups.feature] my agent lists the backups
	r := h.run("backup", "list")

	r.wantCode(t, ExitOK)
	// @then [features/agent-backups.feature] it sees all 3 backups
	// @and [features/agent-backups.feature] each says whether it's a daily backup, made by an agent, or made by the app before a restore
	// @and [features/agent-backups.feature] the backups made by an agent or before a restore show the day each is kept until
	want := "20261014-093000_daily_2026-10-12 | made by the app before restoring 2026-10-12 | kept until 2026-10-21\n" +
		"before-cleanup | made by an agent | kept until 2026-10-20\n" +
		"2026-10-12 | daily backup\n"
	if r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

func TestListingBackupsAsJSON(t *testing.T) {
	h := newHarness()
	h.backups.listed = []backups.Backup{{Kind: backups.Daily, Name: "2026-10-12", Made: "2026-10-12T09:30:00-04:00"}}

	r := h.run("backup", "list", "--json")

	var got []map[string]any
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"kind": "daily", "name": "2026-10-12", "made": "2026-10-12T09:30:00-04:00"}
	if len(got) != 1 || len(got[0]) != len(want) || got[0]["kind"] != want["kind"] || got[0]["name"] != want["name"] || got[0]["made"] != want["made"] {
		t.Errorf("JSON = %s, want one backup %v", r.stdout, want)
	}
}

func TestABackupMadeBeforeUndoingARestoreSaysSo(t *testing.T) {
	h := newHarness()
	h.backups.listed = []backups.Backup{{
		Kind: backups.App, Name: "20261014-093100_app_20261014-093000", KeptUntil: "2026-10-21",
		Restored: &backups.Restored{Kind: backups.App, Name: "20261014-093000"},
	}}

	r := h.run("backup", "list")

	if want := "20261014-093100_app_20261014-093000 | made by the app before a restore | kept until 2026-10-21\n"; r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

func TestListingNoBackups(t *testing.T) {
	h := newHarness()

	if r := h.run("backup", "list"); r.stdout != "No backups are kept yet.\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
	if r := h.run("backup", "list", "--json"); r.stdout != "[]\n" {
		t.Errorf("JSON = %q", r.stdout)
	}
}

// @scenario [#agent-cant-delete-backup]
func TestAgentHasNoWayToDeleteABackup(t *testing.T) {
	h := newHarness()
	// @given [features/agent-backups.feature] my agent made a backup named "before-cleanup"
	h.backups.agent = []backups.AgentBackup{{Name: "before-cleanup", KeptUntil: "2026-10-21"}}

	// @when [features/agent-backups.feature] my agent looks at what it can do with backups
	help := h.run("--help").stdout

	// @then [features/agent-backups.feature] it can make, list, and restore backups
	var listed []string
	for _, m := range regexp.MustCompile(`(?m)^  backup (\S+)`).FindAllStringSubmatch(help, -1) {
		listed = append(listed, m[1])
	}
	if !slices.Equal(listed, []string{"make", "list", "restore"}) {
		t.Errorf("the help lists backup commands %v, want make, list, and restore:\n%s", listed, help)
	}
	// @but [features/agent-backups.feature] it can't delete one
	for _, word := range []string{"delete", "remove", "rm"} {
		r := h.run("backup", word, "before-cleanup")
		r.wantCode(t, ExitUsage)
	}
	if len(h.backups.agent) != 1 {
		t.Errorf("backups = %v, want before-cleanup kept", h.backups.agent)
	}
}

// What the restore brings back is TestRestoreBringsBackTheData in internal/backups, and the backup
// it makes first is TestRestoringBacksUpTheDataFirst there.
// @scenario [#agent-restore] [#restore-backs-up-first]
func TestAgentRestoresABackup(t *testing.T) {
	h := newHarness()
	h.backups.agent = []backups.AgentBackup{{Name: "before-cleanup", KeptUntil: "2026-10-21"}}
	h.backups.undo = backups.AgentBackup{Name: "before-restore-20261014-093015", KeptUntil: "2026-10-21"}

	// @when [features/agent-backups.feature] my agent restores the backup "before-cleanup"
	r := h.run("backup", "restore", "before-cleanup")

	r.wantCode(t, ExitOK)
	if !slices.Equal(h.backups.restored, []string{"before-cleanup"}) {
		t.Errorf("restored %v", h.backups.restored)
	}
	// @and [features/agent-backups.feature] my agent is told that backup's name
	want := "Restored the backup before-cleanup.\n" +
		"The data from just before is in the backup before-restore-20261014-093015, kept until 2026-10-21. Restore it to undo this restore.\n"
	if r.stdout != want {
		t.Errorf("stdout = %q\nwant     %q", r.stdout, want)
	}
}

func TestRestoreAsJSON(t *testing.T) {
	h := newHarness()
	h.backups.undo = backups.AgentBackup{Name: "before-restore-20261014-093015", KeptUntil: "2026-10-21"}

	r := h.run("backup", "restore", "before-cleanup", "--json")

	var got restoredJSON
	if err := json.Unmarshal([]byte(r.stdout), &got); err != nil || got != (restoredJSON{Restored: "before-cleanup", Undo: h.backups.undo}) {
		t.Errorf("JSON = %+v, %v\n%s", got, err, r.stdout)
	}
	for _, key := range []string{`"restored"`, `"undo"`, `"name"`, `"keptUntil"`} {
		if !strings.Contains(r.stdout, key) {
			t.Errorf("JSON has no %s: %s", key, r.stdout)
		}
	}
}

// @scenario [#agent-restore-daily-backup]
func TestRestoringADailyBackup(t *testing.T) {
	h := newHarness()
	h.backups.fail = &backups.ValidationError{Msg: `"2026-10-14" is a daily backup; you can only restore a backup an agent made`}

	// @when [features/agent-backups.feature] my agent tries to restore the daily backup
	r := h.run("backup", "restore", "2026-10-14")

	r.wantCode(t, ExitValidation)
	// @and [features/agent-backups.feature] my agent is told it can only restore backups an agent made
	r.wantError(t, `"2026-10-14" is a daily backup; you can only restore a backup an agent made`)
}

// Which backups the app made is the backups service's to say (TestAgentCantRestoreABackupTheAppMade
// in internal/backups). The fake hands back its message as the service would.
// @scenario [#agent-restore-app-backup]
func TestRestoringABackupTheAppMade(t *testing.T) {
	h := newHarness()
	msg := `"20261014-093000_daily_2026-10-12" was made by the app before a restore; you can only restore a backup an agent made`
	h.backups.fail = &backups.ValidationError{Msg: msg}

	// @when [features/agent-backups.feature] my agent tries to restore the backup the app made
	r := h.run("backup", "restore", "20261014-093000_daily_2026-10-12")

	// @then [features/agent-backups.feature#agent-restore-app-backup] my data is unchanged
	r.wantCode(t, ExitValidation)
	if len(h.backups.restored) != 0 {
		t.Errorf("restored %v, want nothing", h.backups.restored)
	}
	// @and [features/agent-backups.feature] my agent is told it can only restore backups an agent made
	r.wantError(t, msg)
}

// @scenario [#agent-restore-unknown]
func TestRestoringABackupWithNoSuchName(t *testing.T) {
	h := newHarness()
	h.backups.fail = &backups.NotFoundError{Name: "never-made"}

	// @when [features/agent-backups.feature] my agent tries to restore a backup named "never-made"
	r := h.run("backup", "restore", "never-made")

	r.wantCode(t, ExitNotFound)
	// @and [features/agent-backups.feature] my agent is told there's no backup with that name
	r.wantError(t, "there's no backup named \"never-made\"; run `callback-cli backup list` to see the backups you can restore")
}

// That the data is unchanged is TestRestoreRefusesANewerBackup in internal/db.
// @scenario [#restore-newer-backup]
func TestRestoringANewerBackup(t *testing.T) {
	h := newHarness()
	h.backups.fail = db.ErrNewerBackup

	// @when [features/agent-backups.feature] my agent tries to restore the backup "from-newer"
	r := h.run("backup", "restore", "from-newer")

	// @and [features/agent-backups.feature] my agent is told the backup is from a newer version of Callback
	r.wantError(t, newerBackupMessage)
	if !strings.Contains(newerBackupMessage, "the backup is from a newer version of Callback") {
		t.Errorf("the message doesn't say the backup is from a newer version: %q", newerBackupMessage)
	}
	// @and [features/agent-backups.feature] the command ends with the exit code for a newer version of my data
	r.wantCode(t, ExitNewerData)
}

func TestRestoreNeedsABackupName(t *testing.T) {
	h := newHarness()
	for _, args := range [][]string{{"backup", "restore"}, {"backup", "restore", "a", "b"}} {
		r := h.run(args...)
		r.wantCode(t, ExitUsage)
	}
	if len(h.backups.restored) != 0 || h.opened != 0 {
		t.Errorf("restored %v, opened %d", h.backups.restored, h.opened)
	}
}

// Which backups have expired is the backups service's (TestAnExpiredBackupIsRemoved and
// TestRemovingExpiredBackupsKeepsTheDailyBackups in internal/backups).
// @scenario [#agent-backup-expires] [#agent-backup-expiry-keeps-daily]
func TestEveryCommandRemovesExpiredBackups(t *testing.T) {
	h := newHarness()
	h.acme()

	// @when [features/agent-backups.feature] <run>
	// @when [features/agent-backups.feature] my agent runs a Callback command
	h.run("job", "list")
	h.run("backup", "list")

	if h.backups.removals != 2 {
		t.Errorf("RemoveExpired called %d times, want once per command", h.backups.removals)
	}
}

func TestExpiredBackupsThatCantBeRemovedDontStopTheCommand(t *testing.T) {
	h := newHarness()
	h.backups.removeErr = errUnexpected
	h.acme()

	r := h.run("job", "list")

	r.wantCode(t, ExitOK)
	if !strings.Contains(r.stdout, "Acme") ||
		r.stderr != "callback-cli: warning: removing expired backups failed, and will be tried again next time: disk I/O error\n" {
		t.Errorf("stdout %q, stderr %q", r.stdout, r.stderr)
	}
}

func TestHelpAndUsageErrorsRemoveNoBackups(t *testing.T) {
	h := newHarness()
	for _, args := range [][]string{{"--help"}, {"--overview"}, {"backup", "make", "--help"}, {"backup", "make"}} {
		h.run(args...)
	}
	if h.backups.removals != 0 {
		t.Errorf("RemoveExpired called %d times, want none", h.backups.removals)
	}
}
