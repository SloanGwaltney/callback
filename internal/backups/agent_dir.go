package backups

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentFolderName is the agent backups' folder, relative to the data directory. Each backup is
// named for its name and the day it's kept until, like before-cleanup.2026-10-21.db, so that
// restoring a backup, which replaces the database, never changes which backups are kept.
const AgentFolderName = "backups/agent"

// AgentDir keeps the agent backups as files in the data directory, apart from the daily backups.
type AgentDir struct {
	dir string
}

var _ AgentFolder = (*AgentDir)(nil)

// NewAgentDir returns the agent backups' folder in dataDir. It's created when the first agent
// backup is saved.
func NewAgentDir(dataDir string) *AgentDir {
	return &AgentDir{dir: filepath.Join(dataDir, filepath.FromSlash(AgentFolderName))}
}

// Backups returns every agent backup in the folder, expired ones included, in no set order.
// Files not named like an agent backup are ignored, and never removed.
func (d *AgentDir) Backups() ([]AgentBackup, error) {
	entries, err := os.ReadDir(d.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backups: %w", err)
	}
	var out []AgentBackup
	for _, e := range entries {
		if b, ok := parseAgentFile(e.Name()); ok && e.Type().IsRegular() {
			out = append(out, b)
		}
	}
	return out, nil
}

// parseAgentFile reads an agent backup's name and the day it's kept until from its file name.
// Names can't hold a dot, so the first dot is the one before the date.
func parseAgentFile(fileName string) (AgentBackup, bool) {
	base, ok := strings.CutSuffix(fileName, ".db")
	if !ok {
		return AgentBackup{}, false
	}
	name, until, ok := strings.Cut(base, ".")
	if !ok || !validName(name) {
		return AgentBackup{}, false
	}
	if _, err := time.Parse(dateLayout, until); err != nil {
		return AgentBackup{}, false
	}
	return AgentBackup{Name: name, KeptUntil: until}, true
}

// Save has write make the backup at a temporary path, then renames it to b's file. If write or the
// rename fails, the temporary file is removed.
func (d *AgentDir) Save(b AgentBackup, write func(path string) error) error {
	return saveFile(d.dir, d.Path(b), write)
}

// Remove deletes b. One that's already gone isn't an error.
func (d *AgentDir) Remove(b AgentBackup) error {
	if err := os.Remove(d.Path(b)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("backups: %w", err)
	}
	return nil
}

// Path returns the file that holds b.
func (d *AgentDir) Path(b AgentBackup) string {
	return filepath.Join(d.dir, b.Name+"."+b.KeptUntil+".db")
}
