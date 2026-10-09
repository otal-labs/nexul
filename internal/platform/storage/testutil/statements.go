package testutil

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"sync/atomic"
)

// Statements counts what reaches the driver, so a guard can pin a query count or how many values a query binds.
type Statements struct {
	n       atomic.Int64
	maxArgs atomic.Int64
	mu      sync.Mutex
	last    string
	args    []any
}

// Count is how many statements ran since the last Reset.
func (s *Statements) Count() int64 { return s.n.Load() }

// MaxArgs is the most values one statement bound since the last Reset.
func (s *Statements) MaxArgs() int64 { return s.maxArgs.Load() }

// Last is the newest statement's text and arguments, ready to run again, such as under EXPLAIN QUERY PLAN.
func (s *Statements) Last() (string, []any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last, s.args
}

// Reset starts the count over.
func (s *Statements) Reset() {
	s.n.Store(0)
	s.maxArgs.Store(0)
}

func (s *Statements) record(query string, args []driver.NamedValue) {
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

// OpenCounted opens a database through inner, recording every statement it runs.
func OpenCounted(inner driver.Connector) (*sql.DB, *Statements) {
	st := &Statements{}
	return sql.OpenDB(countingConnector{inner, st}), st
}

type countingConnector struct {
	inner driver.Connector
	st    *Statements
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
	st *Statements
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
