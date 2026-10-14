package backups

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
)

// MinDays and MaxDays are how many days an agent backup can be kept.
const (
	MinDays = 1
	MaxDays = 90
)

// UndoDays is how many days the backup a restore makes first is kept.
const UndoDays = 7

// undoPrefix starts the name of the backup a restore makes first.
const undoPrefix = "before-restore-"

// AgentBackup is a backup the user's agent made, under a name, kept through the day KeptUntil.
type AgentBackup struct {
	Name      string `json:"name"`
	KeptUntil string `json:"keptUntil"`
}

// AgentFolder is where the agent backups are kept, apart from the daily backups. *AgentDir
// satisfies it.
type AgentFolder interface {
	// Backups returns every agent backup in the folder, expired ones included, in no set order.
	Backups() ([]AgentBackup, error)
	// Save keeps the file write makes at path as b. If write fails, nothing is left behind.
	Save(b AgentBackup, write func(path string) error) error
	// Remove deletes b.
	Remove(b AgentBackup) error
	// Path returns the file that holds b.
	Path(b AgentBackup) string
}

// Restorer copies a backup into the live data. *db.Restorer satisfies it.
type Restorer interface {
	// RestoreFrom restores the backup at path, bringing a backup from an older version up to date
	// in a copy, and refusing one from a newer version (db.ErrNewerBackup). before runs just
	// before anything is restored; if it fails, nothing is.
	RestoreFrom(ctx context.Context, path string, before func() error) error
}

// ValidationError is an agent backup or restore the rules don't allow. Nothing was changed.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// NotFoundError means no agent backup that's still kept has the name.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("there's no backup named %q", e.Name)
}

var nameChars = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validName reports whether name uses only letters, numbers, hyphens, and underscores. That keeps
// a name from reaching outside the backups' folder, and from holding the dot that comes before
// the date in its file name.
func validName(name string) bool {
	return nameChars.MatchString(name)
}

func (s *Service) today() string {
	return s.now().Format(dateLayout)
}

// kept returns the agent backups that haven't expired, by the day they're kept until and then by
// name. A backup is kept through the last day it's kept until.
func (s *Service) kept() ([]AgentBackup, error) {
	all, err := s.agent.Backups()
	if err != nil {
		return nil, err
	}
	today := s.today()
	kept := slices.DeleteFunc(all, func(b AgentBackup) bool { return b.KeptUntil < today })
	slices.SortFunc(kept, func(a, b AgentBackup) int {
		return cmp.Or(cmp.Compare(a.KeptUntil, b.KeptUntil), cmp.Compare(a.Name, b.Name))
	})
	return kept, nil
}

// MakeAgent backs up the data under name, kept for days days from today. The name must use only
// letters, numbers, hyphens, and underscores, and can't be one a backup that's still kept has.
// An expired backup's name can be used again.
func (s *Service) MakeAgent(ctx context.Context, name string, days int) (AgentBackup, error) {
	switch {
	case name == "":
		return AgentBackup{}, &ValidationError{Msg: "a backup needs a name"}
	case !validName(name):
		return AgentBackup{}, &ValidationError{Msg: "a backup name may only use letters, numbers, hyphens, and underscores"}
	case days < MinDays || days > MaxDays:
		return AgentBackup{}, &ValidationError{Msg: fmt.Sprintf("use %d to %d days for how long a backup is kept", MinDays, MaxDays)}
	}
	kept, err := s.kept()
	if err != nil {
		return AgentBackup{}, err
	}
	if i := slices.IndexFunc(kept, func(b AgentBackup) bool { return b.Name == name }); i >= 0 {
		return AgentBackup{}, &ValidationError{Msg: fmt.Sprintf("the name %q is already taken by a backup kept until %s; choose another name", name, kept[i].KeptUntil)}
	}
	return s.saveAgent(ctx, name, days)
}

func (s *Service) saveAgent(ctx context.Context, name string, days int) (AgentBackup, error) {
	b := AgentBackup{Name: name, KeptUntil: s.now().AddDate(0, 0, days).Format(dateLayout)}
	if err := s.agent.Save(b, func(path string) error { return s.db.SnapshotTo(ctx, path) }); err != nil {
		return AgentBackup{}, err
	}
	return b, nil
}

// ListAgent returns the agent backups that are still kept, by the day they're kept until and then
// by name. The daily backups aren't listed.
func (s *Service) ListAgent() ([]AgentBackup, error) {
	return s.kept()
}

// RemoveExpired removes the agent backups kept until a day before today. One that can't be removed
// is left for a later run, and the rest are still removed. It never removes a daily backup.
func (s *Service) RemoveExpired() error {
	all, err := s.agent.Backups()
	if err != nil {
		return err
	}
	today := s.today()
	var errs []error
	for _, b := range all {
		if b.KeptUntil >= today {
			continue
		}
		if err := s.agent.Remove(b); err != nil {
			errs = append(errs, fmt.Errorf("backups: remove the expired backup %q: %w", b.Name, err))
		}
	}
	return errors.Join(errs...)
}

// Restore copies the agent backup named name into the live data. Just before, it backs up the data
// as it is, kept for UndoDays, and returns that backup, which undoes the restore. It restores only
// agent backups that are still kept, never a daily backup, and leaves which backups are kept as
// it was.
func (s *Service) Restore(ctx context.Context, name string) (AgentBackup, error) {
	kept, err := s.kept()
	if err != nil {
		return AgentBackup{}, err
	}
	i := slices.IndexFunc(kept, func(b AgentBackup) bool { return b.Name == name })
	if i < 0 {
		return AgentBackup{}, s.notAgentBackup(name)
	}

	var undo AgentBackup
	err = s.restorer.RestoreFrom(ctx, s.agent.Path(kept[i]), func() error {
		var err error
		undo, err = s.makeUndo(ctx)
		return err
	})
	if err != nil {
		return AgentBackup{}, err
	}
	return undo, nil
}

// notAgentBackup says why name can't be restored: it's a daily backup's, or no backup's.
func (s *Service) notAgentBackup(name string) error {
	dates, err := s.folder.Dates()
	if err != nil {
		return err
	}
	if slices.Contains(dates, name) {
		return &ValidationError{Msg: fmt.Sprintf("%q is a daily backup; you can only restore a backup an agent made", name)}
	}
	return &NotFoundError{Name: name}
}

// makeUndo backs up the data under a name for the time it's made, like
// before-restore-20261014-093000, kept for UndoDays. A second restore in the same second adds a
// number to the name.
func (s *Service) makeUndo(ctx context.Context) (AgentBackup, error) {
	all, err := s.agent.Backups()
	if err != nil {
		return AgentBackup{}, err
	}
	base := undoPrefix + s.now().Format("20060102-150405")
	name := base
	for n := 2; slices.ContainsFunc(all, func(b AgentBackup) bool { return b.Name == name }); n++ {
		name = fmt.Sprintf("%s-%d", base, n)
	}
	return s.saveAgent(ctx, name, UndoDays)
}
