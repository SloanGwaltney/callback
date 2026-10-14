package backups

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// DailyFolder is the daily backups' folder, relative to the data directory. Each backup is named for
// its date, like 2026-10-14.db.
const DailyFolder = "backups/daily"

// partialExt marks a backup that's still being written. It's renamed to .db once complete, so a
// backup that failed partway is never mistaken for a whole one.
const partialExt = ".partial"

// Dir keeps the daily backups as files in the data directory.
type Dir struct {
	dir string
}

var _ Folder = (*Dir)(nil)

// NewDir returns the daily backups' folder in dataDir. It's created when the first backup is saved.
func NewDir(dataDir string) *Dir {
	return &Dir{dir: filepath.Join(dataDir, filepath.FromSlash(DailyFolder))}
}

// Dates returns the dates of the daily backups, oldest first. Files not named like a daily
// backup are ignored, and never removed.
func (d *Dir) Dates() ([]string, error) {
	entries, err := os.ReadDir(d.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backups: %w", err)
	}
	var dates []string
	for _, e := range entries {
		date, ok := strings.CutSuffix(e.Name(), ".db")
		if !ok || !e.Type().IsRegular() {
			continue
		}
		if _, err := time.Parse(dateLayout, date); err != nil {
			continue
		}
		dates = append(dates, date)
	}
	slices.Sort(dates)
	return dates, nil
}

// Save has write make the backup at a temporary path, then renames it to the backup for date. If
// write or the rename fails, the temporary file is removed.
func (d *Dir) Save(date string, write func(path string) error) error {
	return saveFile(d.dir, d.path(date), write)
}

// saveFile has write make a file at a temporary path in dir, then renames it to final. If write or
// the rename fails, the temporary file is removed.
func saveFile(dir, final string, write func(path string) error) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("backups: %w", err)
	}
	partial := final + partialExt
	// A partial file from a run that crashed partway would stop the snapshot, which won't
	// overwrite a file.
	if err := os.Remove(partial); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("backups: %w", err)
	}
	if err := write(partial); err != nil {
		return errors.Join(err, removePartial(partial))
	}
	if err := os.Rename(partial, final); err != nil {
		return errors.Join(fmt.Errorf("backups: %w", err), removePartial(partial))
	}
	return nil
}

func removePartial(partial string) error {
	if err := os.Remove(partial); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("backups: remove a partial backup: %w", err)
	}
	return nil
}

// Remove deletes the backup for date. One that's already gone isn't an error.
func (d *Dir) Remove(date string) error {
	if err := os.Remove(d.path(date)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("backups: %w", err)
	}
	return nil
}

func (d *Dir) path(date string) string {
	return filepath.Join(d.dir, date+".db")
}
