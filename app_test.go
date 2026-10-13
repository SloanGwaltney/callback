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
}

func (f *fakeBackups) TakeDaily(context.Context) (bool, error) {
	f.calls++
	return f.err == nil, f.err
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
