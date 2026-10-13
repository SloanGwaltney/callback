package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"callback/internal/store"
)

// Store is the storage layer the services depend on: the sqlc queries plus transactions. See
// docs/adr/0016-put-business-rules-in-services-over-a-transactional-store-interface.md.
type Store struct {
	*store.Queries
	conn *sql.DB
}

// NewStore wraps an open database.
func NewStore(conn *sql.DB) *Store {
	return &Store{Queries: store.New(conn), conn: conn}
}

// InTx runs fn in a single transaction. It commits if fn returns nil and rolls back otherwise.
func (s *Store) InTx(ctx context.Context, fn func(store.Querier) error) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	if err := fn(s.Queries.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(err, fmt.Errorf("db: rollback: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}

// SnapshotTo writes a consistent copy of the whole database to a new file at path, with VACUUM
// INTO. The database stays open and usable while it runs. It fails if a file is already at path.
func (s *Store) SnapshotTo(ctx context.Context, path string) error {
	if _, err := s.conn.ExecContext(ctx, `VACUUM INTO ?`, path); err != nil {
		return fmt.Errorf("db: snapshot: %w", err)
	}
	return nil
}
