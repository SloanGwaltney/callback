package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeBackups stands in for the backups service, whose rules are tested in internal/backups.
type fakeBackups struct {
	calls int
	err   error
	// removals counts the calls to RemoveExpired, which returns removeErr.
	removals  int
	removeErr error
}

func (f *fakeBackups) TakeDaily(context.Context) (bool, error) {
	f.calls++
	return f.err == nil, f.err
}

func (f *fakeBackups) RemoveExpired() error {
	f.removals++
	return f.removeErr
}

// newTestApp returns an App with fake backups whose log is collected in logged.
func newTestApp(b *fakeBackups) (*App, *[]string) {
	var logged []string
	a := NewApp(nil, b)
	a.logf = func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }
	return a, &logged
}

// @scenario [features/daily-backups.feature#backup-on-first-start] The first start of the day takes a backup
func TestStartupTakesTheDailyBackup(t *testing.T) {
	b := &fakeBackups{}
	a, logged := newTestApp(b)

	// @when [features/daily-backups.feature] <start>
	// @when [features/daily-backups.feature] I open the app
	a.startup(context.Background())

	// Which day it's for, and whether one was already taken, is decided by the backups service
	// (TestFirstStartOfTheDayTakesABackup in internal/backups).
	if b.calls != 1 || len(*logged) != 0 {
		t.Errorf("TakeDaily called %d times, logged %v", b.calls, *logged)
	}
}

// @scenario [features/daily-backups.feature#backup-failure-still-starts] Callback still starts when the daily backup fails
func TestStartupStillStartsWhenTheBackupFails(t *testing.T) {
	// @given [features/daily-backups.feature] no backup has been taken today
	// @and [features/daily-backups.feature] the backup can't be written
	b := &fakeBackups{err: errors.New("disk full")}
	a, logged := newTestApp(b)
	ctx := context.WithValue(context.Background(), struct{}{}, "wails")

	// @when [features/daily-backups.feature] I open the app
	a.startup(ctx)

	// @then [features/daily-backups.feature] the board opens as usual
	// startup returns without failing, and keeps the context every binding, GetBoard included,
	// runs with.
	if a.ctx != ctx {
		t.Error("startup didn't keep the context the bindings use")
	}
	if len(*logged) != 1 || !strings.Contains((*logged)[0], "disk full") {
		t.Errorf("logged %v, want the backup's error", *logged)
	}
	// No partial backup is left behind (TestAFailedBackupLeavesNothingBehind in internal/backups).
}

// @scenario [features/data-version.feature#app-newer-data-no-backup] The app takes no daily backup of data from a newer Callback
func TestStartupTakesNoBackupOfNewerData(t *testing.T) {
	// @given [features/data-version.feature] my data was last updated by a newer version of Callback than the app
	// db.Open reports it (TestOpenRefusesDataFromANewerVersion in internal/db), and main then makes
	// a NewNewerDataApp, which has no backups service; this fake would record any call.
	a := NewNewerDataApp()
	// @and [features/data-version.feature] no backup has been taken today
	b := &fakeBackups{}
	a.backups = b

	// @when [features/data-version.feature] I open the app
	a.startup(context.Background())

	// @then [features/data-version.feature] no daily backup is taken
	// @and [features/data-version.feature] no backup is removed
	// TakeDaily is what takes the day's backup and then removes the oldest ones, and
	// RemoveExpired removes expired agent backups.
	if b.calls != 0 || b.removals != 0 {
		t.Errorf("TakeDaily called %d times and RemoveExpired %d, want none", b.calls, b.removals)
	}
}

// Which backups have expired is decided by the backups service (TestAnExpiredBackupIsRemoved and
// TestABackupIsKeptThroughItsLastDay in internal/backups).
// @scenario [features/agent-backups.feature#agent-backup-expires] An expired backup is removed when Callback runs
// @scenario [features/agent-backups.feature#agent-backup-kept-through-last-day] A backup is still kept on the last day it's kept until
func TestStartupRemovesExpiredAgentBackups(t *testing.T) {
	b := &fakeBackups{}
	a, logged := newTestApp(b)

	// @when [features/agent-backups.feature] <run>
	// @when [features/agent-backups.feature] I open the app
	a.startup(context.Background())

	if b.removals != 1 || len(*logged) != 0 {
		t.Errorf("RemoveExpired called %d times, logged %v", b.removals, *logged)
	}
}

// That it's removed on a later run is TestAnExpiredBackupThatCantBeRemovedIsRemovedLater in
// internal/backups.
// @scenario [features/agent-backups.feature#agent-backup-expiry-retries] An expired backup that can't be removed yet is removed on a later run
func TestStartupStillOpensWhenExpiredBackupsCantBeRemoved(t *testing.T) {
	// @and [features/agent-backups.feature] the "before-cleanup" backup can't be removed right now
	b := &fakeBackups{removeErr: errors.New("the file is in use")}
	a, logged := newTestApp(b)
	ctx := context.WithValue(context.Background(), struct{}{}, "wails")

	// @when [features/agent-backups.feature] I open the app
	a.startup(ctx)

	// @then [features/agent-backups.feature] the app opens as usual
	if a.ctx != ctx || b.calls != 1 {
		t.Errorf("startup didn't keep the context, or took %d daily backups", b.calls)
	}
	if len(*logged) != 1 || !strings.Contains((*logged)[0], "the file is in use") {
		t.Errorf("logged %v, want the removal's error", *logged)
	}
}

// @scenario [features/data-version.feature#app-newer-data] The app refuses data a newer Callback has updated
func TestDataIsNewer(t *testing.T) {
	if !NewNewerDataApp().DataIsNewer() {
		t.Error("an App for newer data reports the data isn't newer")
	}
	if NewApp(nil, &fakeBackups{}).DataIsNewer() {
		t.Error("an App with services reports the data is newer")
	}
}
