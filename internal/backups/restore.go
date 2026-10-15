package backups

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"callback/internal/db"
)

// Kind is who made a backup.
type Kind string

const (
	// Daily is a backup the app or the command line took on the first run of a day.
	Daily Kind = "daily"
	// Agent is a backup the user's agent made under a name.
	Agent Kind = "agent"
	// App is a backup the app made just before it restored one, which undoes that restore.
	App Kind = "app"
)

// Backup is any backup that's still kept, as the app and the agent's list show it.
type Backup struct {
	Kind Kind `json:"kind"`
	// Name identifies the backup within its kind: a daily backup's date, like 2026-10-12, or an agent
	// or app backup's name.
	Name string `json:"name"`
	// Made is when the backup was made, in RFC 3339.
	Made string `json:"made"`
	// KeptUntil is the last day an agent or app backup is kept. A daily backup is kept by count
	// instead, and has none.
	KeptUntil string `json:"keptUntil,omitempty"`
	// Restored, on an app backup, is the backup whose restore it was made before.
	Restored *Restored `json:"restored,omitempty"`
}

// Restored is the backup an app backup was made before restoring. For a daily or agent backup,
// Name is its name. For an app backup, Name is when that backup was made, like 20261014-093000,
// since that's all an app backup's name keeps of it.
type Restored struct {
	Kind Kind   `json:"kind"`
	Name string `json:"name"`
}

// appNameSep separates the parts of an app backup's name: when it was made, then the kind and name
// of the backup restored, like 20261014-093000_daily_2026-10-12. Neither the time nor a kind holds
// one, so the first two are always the separators, and the rest is the restored backup's name.
const appNameSep = "_"

// appName names the backup made now, before restoring the backup of kind named name, with -n after
// the time for a second one in the same second.
func (s *Service) appName(kind Kind, name string) func(n int) string {
	if kind == App {
		name, _, _ = strings.Cut(name, appNameSep)
	}
	return func(n int) string {
		return strings.Join([]string{s.stamp(n), string(kind), name}, appNameSep)
	}
}

// restoredBy reads which backup an app backup was made before restoring from its name. It's nil
// for a name not made by appName.
func restoredBy(name string) *Restored {
	parts := strings.SplitN(name, appNameSep, 3)
	if len(parts) != 3 {
		return nil
	}
	return &Restored{Kind: Kind(parts[1]), Name: parts[2]}
}

// List returns every backup that's still kept, newest first: the daily backups, the agent backups,
// and the backups the app made before a restore.
// @scenario [#app-backups-open] [#app-backups-newest-first] [#app-backups-none] [#app-backups-list-every-kind] [#agent-list-all-backups]
func (s *Service) List() ([]Backup, error) {
	type made struct {
		Backup
		at time.Time
	}
	var all []made

	dates, err := s.folder.Dates()
	if err != nil {
		return nil, err
	}
	for _, date := range dates {
		at, err := s.folder.Made(date)
		if err != nil {
			return nil, err
		}
		all = append(all, made{Backup{Kind: Daily, Name: date}, at})
	}
	for _, kind := range []Kind{Agent, App} {
		folder := s.namedFolder(kind)
		kept, err := s.keptIn(folder)
		if err != nil {
			return nil, err
		}
		for _, b := range kept {
			at, err := folder.Made(b)
			if err != nil {
				return nil, err
			}
			listed := Backup{Kind: kind, Name: b.Name, KeptUntil: b.KeptUntil}
			if kind == App {
				listed.Restored = restoredBy(b.Name)
			}
			all = append(all, made{listed, at})
		}
	}

	slices.SortStableFunc(all, func(a, b made) int {
		return cmp.Or(b.at.Compare(a.at), cmp.Compare(b.Name, a.Name))
	})
	out := make([]Backup, len(all))
	for i, m := range all {
		m.Made = m.at.Format(time.RFC3339)
		out[i] = m.Backup
	}
	return out, nil
}

func (s *Service) namedFolder(kind Kind) AgentFolder {
	if kind == App {
		return s.app
	}
	return s.agent
}

// RestoreError is a restore from the app that didn't happen, with a message the app shows the user
// as is. The data is unchanged.
type RestoreError struct {
	Msg string
	Err error
}

func (e *RestoreError) Error() string { return e.Msg }
func (e *RestoreError) Unwrap() error { return e.Err }

// The messages a RestoreError shows the user.
const (
	goneMessage   = "That backup no longer exists. It may have been removed since you opened your backups. Your data is unchanged."
	newerMessage  = "That backup is from a newer version of Callback, so it can't be restored. Update Callback to restore it. Your data is unchanged."
	failedMessage = "The backup couldn't be restored. Your data is unchanged."
)

// RestoreBackup copies any backup that's still kept into the live data: a daily backup, an agent
// backup, or one the app made before an earlier restore. Just before, it backs up the data as it
// is, kept for UndoDays, in the app's own folder, and returns that backup, which undoes the
// restore. It leaves which backups are kept as it was. A restore that doesn't happen returns a
// RestoreError.
// @scenario [#app-restore-daily-backup] [#app-restore-agent-backup] [#app-restore-keeps-daily-backups] [#app-restore-backs-up-first] [#app-undo-restore] [#app-pre-restore-backup-not-daily] [#app-restore-older-version] [#app-restore-newer-version] [#app-restore-unreadable] [#app-restore-missing] [#app-restore-missing-document-file] [#app-restore-keeps-document-files]
func (s *Service) RestoreBackup(ctx context.Context, kind Kind, name string) (Backup, error) {
	path, err := s.pathOf(kind, name)
	if err != nil {
		return Backup{}, err
	}

	var undo AgentBackup
	err = s.restorer.RestoreFrom(ctx, path, func() error {
		undoName, err := s.unusedName(s.app, s.appName(kind, name))
		if err != nil {
			return err
		}
		undo, err = s.saveIn(ctx, s.app, undoName, UndoDays)
		return err
	})
	switch {
	case errors.Is(err, db.ErrNewerBackup):
		return Backup{}, &RestoreError{Msg: newerMessage, Err: err}
	case err != nil:
		return Backup{}, &RestoreError{Msg: failedMessage, Err: err}
	}
	made, err := s.app.Made(undo)
	if err != nil {
		return Backup{}, err
	}
	return Backup{
		Kind:      App,
		Name:      undo.Name,
		Made:      made.Format(time.RFC3339),
		KeptUntil: undo.KeptUntil,
		Restored:  restoredBy(undo.Name),
	}, nil
}

// pathOf returns the file of the backup of kind named name, if it's still kept.
func (s *Service) pathOf(kind Kind, name string) (string, error) {
	gone := &RestoreError{Msg: goneMessage, Err: &NotFoundError{Name: name}}
	switch kind {
	case Daily:
		dates, err := s.folder.Dates()
		if err != nil {
			return "", err
		}
		if !slices.Contains(dates, name) {
			return "", gone
		}
		return s.folder.Path(name), nil
	case Agent, App:
		folder := s.namedFolder(kind)
		kept, err := s.keptIn(folder)
		if err != nil {
			return "", err
		}
		i := slices.IndexFunc(kept, func(b AgentBackup) bool { return b.Name == name })
		if i < 0 {
			return "", gone
		}
		return folder.Path(kept[i]), nil
	}
	return "", &ValidationError{Msg: fmt.Sprintf("there's no kind of backup called %q", kind)}
}
