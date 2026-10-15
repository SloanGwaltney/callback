// Package backups takes a backup of the user's data once on each day the app runs, and keeps only
// the most recent ones. It also makes, lists, and restores the backups the user's agent makes
// through the command line (agent.go), each kept for the days the agent chose, and lists and
// restores every backup from the app (restore.go). A backup is a copy of the SQLite database. It
// holds which documents are attached, but not their files, which stay in the documents folder.
package backups

import (
	"context"
	"fmt"
	"slices"
	"time"
)

// Keep is how many daily backups are kept. Older ones are removed by count, never by age, so the
// backups from before a long break are still there after it.
const Keep = 15

// dateLayout is how a daily backup's date is written: YYYY-MM-DD.
const dateLayout = "2006-01-02"

// Snapshotter writes a consistent copy of the database to a new file. *db.Store satisfies it.
type Snapshotter interface {
	SnapshotTo(ctx context.Context, path string) error
}

// Folder is where the daily backups are kept, one per date. *Dir satisfies it.
type Folder interface {
	// Dates returns the dates of the daily backups, oldest first.
	Dates() ([]string, error)
	// Save keeps the file write makes at path as the backup for date. If write fails, nothing is
	// left behind.
	Save(date string, write func(path string) error) error
	// Remove deletes the backup for date.
	Remove(date string) error
	// Path returns the file that holds the backup for date.
	Path(date string) string
	// Made returns when the backup for date was written.
	Made(date string) (time.Time, error)
}

// Service takes the daily backups, makes and restores agent backups, and lists and restores every
// backup for the app.
type Service struct {
	db       Snapshotter
	restorer Restorer
	folder   Folder
	agent    AgentFolder
	// app holds the backups the app makes just before it restores one, apart from the agent's, so
	// an agent can't restore them.
	app AgentFolder
	now func() time.Time
}

// New returns a Service over the daily backups' folder, the agent backups' folder, and the folder
// of backups the app makes before a restore. now supplies the current time, which decides which
// day a backup is for and which agent and app backups have expired.
func New(db Snapshotter, restorer Restorer, folder Folder, agent, app AgentFolder, now func() time.Time) *Service {
	return &Service{db: db, restorer: restorer, folder: folder, agent: agent, app: app, now: now}
}

// TakeDaily backs up the data if no backup has been taken today, then removes the oldest daily
// backups beyond Keep. It reports whether it took one. Days the app didn't run get no backup.
func (s *Service) TakeDaily(ctx context.Context) (bool, error) {
	today := s.now().Format(dateLayout)
	dates, err := s.folder.Dates()
	if err != nil {
		return false, err
	}
	if slices.Contains(dates, today) {
		return false, nil
	}
	if err := s.folder.Save(today, func(path string) error { return s.db.SnapshotTo(ctx, path) }); err != nil {
		return false, err
	}

	dates = append(dates, today)
	slices.Sort(dates)
	for len(dates) > Keep {
		if err := s.folder.Remove(dates[0]); err != nil {
			return true, fmt.Errorf("backups: remove the backup from %s: %w", dates[0], err)
		}
		dates = dates[1:]
	}
	return true, nil
}
