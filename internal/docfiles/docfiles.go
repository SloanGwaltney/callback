// Package docfiles keeps the app's own copies of attached documents in a folder inside the data
// directory. See docs/adr/0008-store-attached-documents-as-files-in-the-app-data-directory.md.
package docfiles

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Folder is the documents folder's name inside the data directory.
const Folder = "documents"

// Dir stores document copies under a data directory.
type Dir struct {
	dataDir string
}

// New returns the document store for dataDir. The documents folder is created on first save.
func New(dataDir string) *Dir {
	return &Dir{dataDir: dataDir}
}

// Size returns the size in bytes of the file at src.
func (d *Dir) Size(src string) (int64, error) {
	info, err := os.Stat(src)
	if err != nil {
		return 0, fmt.Errorf("docfiles: %w", err)
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("docfiles: %s is not a regular file", src)
	}
	return info.Size(), nil
}

// Save copies the file at src into the documents folder under a generated name that keeps src's
// extension, and returns the copy's path relative to the data directory, with forward slashes.
// Every call makes a new copy, even of the same file.
func (d *Dir) Save(src string) (stored string, err error) {
	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("docfiles: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Join(d.dataDir, Folder), 0o700); err != nil {
		return "", fmt.Errorf("docfiles: create documents folder: %w", err)
	}

	id, err := newID()
	if err != nil {
		return "", err
	}
	stored = path.Join(Folder, id+strings.ToLower(filepath.Ext(src)))
	dst := d.Path(stored)

	// O_EXCL so a generated name can never overwrite an existing copy.
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("docfiles: %w", err)
	}
	defer func() {
		if err != nil {
			os.Remove(dst)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", fmt.Errorf("docfiles: copy %s: %w", src, err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("docfiles: %w", err)
	}
	return stored, nil
}

// Remove deletes a stored copy. A copy that's already gone isn't an error.
func (d *Dir) Remove(stored string) error {
	if err := os.Remove(d.Path(stored)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("docfiles: %w", err)
	}
	return nil
}

// Path returns the absolute path of a stored copy.
func (d *Dir) Path(stored string) string {
	return filepath.Join(d.dataDir, filepath.FromSlash(stored))
}

// newID returns 32 random hex characters, enough that generated names don't collide.
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("docfiles: generate name: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
