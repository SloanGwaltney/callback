package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"time"

	"callback/internal/appdata"
	"callback/internal/backups"
	"callback/internal/db"
	"callback/internal/db/migrations"
	"callback/internal/desktop"
	"callback/internal/docfiles"
	"callback/internal/jobs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// StartHiddenEnvVar keeps the window hidden when set, so an agent can drive a development run
// through the dev server in a headless browser without a window opening on the user's screen.
// Pair it with appdata.EnvVar so the run doesn't touch the user's real data.
const StartHiddenEnvVar = "CALLBACK_START_HIDDEN"

// TodayEnvVar, set to a date like 2026-10-09, makes the app take that date as today, so an agent
// driving a development run can see what depends on the date, like days in status or next actions
// coming due. Restarting with a later date moves the run forward; the time of day stays real.
const TodayEnvVar = "CALLBACK_TODAY"

func main() {
	dataDir, err := appdata.Dir()
	if err != nil {
		fail(err)
	}
	conn, err := db.Open(context.Background(), dataDir, migrations.FS)
	if err != nil {
		fail(err)
	}
	defer conn.Close()
	now, err := clock(os.Getenv, time.Now)
	if err != nil {
		fail(err)
	}

	// Create an instance of the app structure
	store := db.NewStore(conn)
	app := NewApp(
		jobs.New(store, docfiles.New(dataDir), desktop.Opener{}, now),
		backups.New(store, backups.NewDir(dataDir), now),
	)

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "callback",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		StartHidden:      startHidden(os.Getenv),
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

// startHidden reports whether StartHiddenEnvVar asks for the window to stay hidden.
func startHidden(getenv func(string) string) bool {
	return getenv(StartHiddenEnvVar) != ""
}

// clock returns what the app takes as the current time: now, or with TodayEnvVar set, that date at
// now's local time of day.
func clock(getenv func(string) string, now func() time.Time) (func() time.Time, error) {
	value := getenv(TodayEnvVar)
	if value == "" {
		return now, nil
	}
	day, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, fmt.Errorf("%s must be a date like 2026-10-09, not %q", TodayEnvVar, value)
	}
	return func() time.Time {
		t := now()
		return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
	}, nil
}

// fail reports a startup error and exits. It runs before the window exists, so the error only
// reaches a console.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "callback:", err)
	os.Exit(1)
}
