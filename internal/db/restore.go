package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mattn/go-sqlite3"
)

// ErrNewerBackup is returned by RestoreFrom when a newer version of callback made the backup. The
// data is left unchanged.
var ErrNewerBackup = errors.New("db: the backup is from a newer version of callback")

// Restorer copies backups into the live database. See
// docs/adr/0025-restore-backups-with-sqlites-online-backup-api-and-check-data-versions-first.md.
type Restorer struct {
	conn       *sql.DB
	migrations fs.FS
}

// NewRestorer returns a Restorer into the open database conn, bringing backups up to date with
// migrations.
func NewRestorer(conn *sql.DB, migrations fs.FS) *Restorer {
	return &Restorer{conn: conn, migrations: migrations}
}

// RestoreFrom copies the backup at path into the live database with SQLite's online backup API,
// so connections that already have the database open, like the app's, see the restored data on
// their next read without being closed. The backup is first copied to a temporary file and that
// copy brought up to date, so a backup from an older version is restored in the current schema
// and the backup itself is never changed. A backup from a newer version returns ErrNewerBackup.
// before runs once the copy is ready and before anything is restored; if it fails, nothing is.
func (r *Restorer) RestoreFrom(ctx context.Context, path string, before func() error) error {
	tmp, err := os.MkdirTemp("", "callback-restore-")
	if err != nil {
		return fmt.Errorf("db: restore: %w", err)
	}
	defer os.RemoveAll(tmp)

	copyPath := filepath.Join(tmp, FileName)
	if err := copyFile(path, copyPath); err != nil {
		return err
	}
	src, err := sql.Open("sqlite3", dsn(copyPath))
	if err != nil {
		return fmt.Errorf("db: restore: %w", err)
	}
	defer src.Close()
	if err := migrate(ctx, src, r.migrations); err != nil {
		if errors.Is(err, ErrNewerData) {
			return ErrNewerBackup
		}
		return err
	}

	if err := before(); err != nil {
		return err
	}
	return copyInto(ctx, r.conn, src)
}

// copyFile copies the file at from to a new file at to.
func copyFile(from, to string) (err error) {
	in, err := os.Open(from)
	if err != nil {
		return fmt.Errorf("db: restore: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("db: restore: %w", err)
	}
	defer func() {
		if cerr := out.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("db: restore: %w", cerr)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("db: restore: %w", err)
	}
	return nil
}

// copyInto copies every page of src into dst with the online backup API, under SQLite's own
// locking: it waits for a write another connection is in the middle of, within dst's busy timeout.
func copyInto(ctx context.Context, dst, src *sql.DB) error {
	dstConn, err := dst.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db: restore: %w", err)
	}
	defer dstConn.Close()
	srcConn, err := src.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db: restore: %w", err)
	}
	defer srcConn.Close()

	return dstConn.Raw(func(d any) error {
		return srcConn.Raw(func(s any) error {
			backup, err := d.(*sqlite3.SQLiteConn).Backup("main", s.(*sqlite3.SQLiteConn), "main")
			if err != nil {
				return fmt.Errorf("db: restore: %w", err)
			}
			done, stepErr := backup.Step(-1)
			if err := errors.Join(stepErr, backup.Finish()); err != nil {
				return fmt.Errorf("db: restore: %w", err)
			}
			if !done {
				return errors.New("db: restore: the backup didn't copy every page")
			}
			return nil
		})
	})
}
