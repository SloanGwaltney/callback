// Command callback-cli is Callback's command line for AI agents, shipped beside the app and run
// over the same data. Its commands are in internal/cli; this only wires them to the user's data.
// See docs/adr/0023-let-users-drive-agent-tasks-from-their-own-agent-through-a-cli.md.
package main

import (
	"context"
	"errors"
	"os"
	"time"

	"callback/internal/appdata"
	"callback/internal/backups"
	"callback/internal/cli"
	"callback/internal/db"
	"callback/internal/db/migrations"
	"callback/internal/docfiles"
	"callback/internal/jobs"
)

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.Env{
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		ReadFile: os.ReadFile,
		Open:     open,
	}))
}

// open opens the user's data, in the same directory and through the same db.Open as the app, so
// data from an older version of Callback is brought up to date and data from a newer one is left
// unchanged (db.ErrNewerData).
func open(ctx context.Context) (cli.Services, func() error, error) {
	dataDir, err := appdata.Dir()
	if err != nil {
		return cli.Services{}, nil, err
	}
	conn, err := db.Open(ctx, dataDir, migrations.FS)
	if err != nil {
		return cli.Services{}, nil, err
	}
	store := db.NewStore(conn)
	return cli.Services{
		Jobs:    jobs.New(store, docfiles.New(dataDir), noOpener{}, time.Now),
		Backups: backups.New(store, db.NewRestorer(conn, migrations.FS), backups.NewDir(dataDir), backups.NewAgentDir(dataDir), backups.NewAppDir(dataDir), time.Now),
	}, conn.Close, nil
}

// errNoOpening is what opening anything from the command line returns.
var errNoOpening = errors.New("the command line can't open anything on the screen; read the link or the document's stored copy instead")

// noOpener stands in for the app's opener. An agent never opens anything on the user's screen.
type noOpener struct{}

func (noOpener) Open(string) error { return errNoOpening }
