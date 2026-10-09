package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"
)

// statements counts what reaches the driver, so a guard can pin a query count or how many values a query binds.
type statements struct {
	n       atomic.Int64
	maxArgs atomic.Int64
	mu      sync.Mutex
	last    string
	args    []any
}

func (s *statements) record(query string, args []driver.NamedValue) {
	s.n.Add(1)
	s.mu.Lock()
	s.last, s.args = query, make([]any, len(args))
	for i, a := range args {
		s.args[i] = a.Value
	}
	s.mu.Unlock()
	for {
		cur := s.maxArgs.Load()
		if int64(len(args)) <= cur || s.maxArgs.CompareAndSwap(cur, int64(len(args))) {
			return
		}
	}
}

// lastStatement is the newest statement's text and arguments, ready to run again, such as under EXPLAIN QUERY PLAN.
func (s *statements) lastStatement() (string, []any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last, s.args
}

func (s *statements) reset() {
	s.n.Store(0)
	s.maxArgs.Store(0)
}

// newCountedStore is newTestStore over a driver connection that records every statement.
func newCountedStore(t *testing.T) (*Store, *statements) {
	t.Helper()
	data, err := os.ReadFile(migratedTemplate(t))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	inner, err := sqlite.NewConnector(path + "?" + pragmas)
	require.NoError(t, err)
	st := &statements{}
	db := sql.OpenDB(countingConnector{inner, st})
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return New(db, testEncKey), st
}

type countingConnector struct {
	inner driver.Connector
	st    *statements
}

func (c countingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return countingConn{conn, c.st}, nil
}

func (c countingConnector) Driver() driver.Driver { return c.inner.Driver() }

type countingConn struct {
	driver.Conn
	st *statements
}

func (c countingConn) QueryContext(ctx context.Context, q string, a []driver.NamedValue) (driver.Rows, error) {
	c.st.record(q, a)
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, a)
}

func (c countingConn) ExecContext(ctx context.Context, q string, a []driver.NamedValue) (driver.Result, error) {
	c.st.record(q, a)
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, q, a)
}

func (c countingConn) BeginTx(ctx context.Context, o driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, o)
}
