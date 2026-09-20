package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
)

type Serializer struct {
	mu sync.Mutex
}

func (s *Serializer) WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		// Rollback after a successful Commit is a harmless no-op that returns sql.ErrTxDone.
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			err = errors.Join(err, rerr)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
