package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"time"

	"callback/internal/appdata"
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

	// Create an instance of the app structure
	app := NewApp(jobs.New(db.NewStore(conn), docfiles.New(dataDir), desktop.Opener{}, time.Now))

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "callback",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

// fail reports a startup error and exits. It runs before the window exists, so the error only
// reaches a console.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "callback:", err)
	os.Exit(1)
}
