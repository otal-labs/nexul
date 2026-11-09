package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/otal-labs/nexul/internal/platform/wake"
)

type Serializer struct {
	mu        sync.Mutex
	committed wake.Broadcast
	watches   []*tableWatch
	// wrote is the writes the open transaction made, per table; unknown means the connection could not report them.
	wrote   map[string]writeOps
	unknown bool
}

type writeOps uint8

const (
	opInsert writeOps = 1 << iota
	opUpdate
	opDelete
)

// TableWrite is the writes to one table that can give a background loop work.
type TableWrite struct {
	table string
	ops   writeOps
}

// Inserts is a row inserted into table.
func Inserts(table string) TableWrite { return TableWrite{table: table, ops: opInsert} }

// Writes is a row inserted into, updated in or deleted from table.
func Writes(table string) TableWrite {
	return TableWrite{table: table, ops: opInsert | opUpdate | opDelete}
}

type tableWatch struct {
	writes []TableWrite
	b      wake.Broadcast
}

type preUpdateHooker interface {
	RegisterPreUpdateHook(sqlite.PreUpdateHookFn)
}

func (s *Serializer) WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	if err := conn.Raw(s.track); err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { err = errors.Join(err, conn.Raw(untrack)) }()

	tx, err := conn.BeginTx(ctx, nil)
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
	s.notify()
	return nil
}

// track records every row the transaction writes, triggers' included, into s.wrote.
func (s *Serializer) track(dc any) error {
	if s.wrote == nil {
		s.wrote = map[string]writeOps{}
	}
	clear(s.wrote)
	h, ok := dc.(preUpdateHooker)
	s.unknown = !ok
	if !ok {
		return nil
	}
	h.RegisterPreUpdateHook(func(d sqlite.SQLitePreUpdateData) {
		s.wrote[d.TableName] |= opOf(d.Op)
	})
	return nil
}

func untrack(dc any) error {
	if h, ok := dc.(preUpdateHooker); ok {
		h.RegisterPreUpdateHook(nil)
	}
	return nil
}

func opOf(op int32) writeOps {
	switch op {
	case sqlite3.SQLITE_INSERT:
		return opInsert
	case sqlite3.SQLITE_UPDATE:
		return opUpdate
	case sqlite3.SQLITE_DELETE:
		return opDelete
	}
	return opInsert | opUpdate | opDelete
}

func (s *Serializer) notify() {
	s.committed.Notify()
	for _, w := range s.watches {
		if s.unknown || w.wokenBy(s.wrote) {
			w.b.Notify()
		}
	}
}

func (w *tableWatch) wokenBy(wrote map[string]writeOps) bool {
	for _, tw := range w.writes {
		if wrote[tw.table]&tw.ops != 0 {
			return true
		}
	}
	return false
}

// watch returns a broadcast that a commit wakes only when it made one of writes.
func (s *Serializer) watch(writes []TableWrite) *wake.Broadcast {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := &tableWatch{writes: writes}
	s.watches = append(s.watches, w)
	return &w.b
}
