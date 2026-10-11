package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// preparedReads runs each query through a statement prepared once per text and kept, so SQLite parses a hot read once
// per connection instead of on every call. Only static statements go through it, since every text it sees is kept.
type preparedReads struct {
	*sql.DB
	mu    sync.Mutex
	stmts map[string]*sql.Stmt
}

func newPreparedReads(db *sql.DB) *preparedReads {
	return &preparedReads{DB: db, stmts: map[string]*sql.Stmt{}}
}

func (p *preparedReads) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	stmt, err := p.stmt(ctx, query)
	if err != nil {
		return nil, err
	}
	return stmt.QueryContext(ctx, args...)
}

func (p *preparedReads) stmt(ctx context.Context, query string) (*sql.Stmt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if stmt, ok := p.stmts[query]; ok {
		return stmt, nil
	}
	stmt, err := p.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare a read: %w", err)
	}
	p.stmts[query] = stmt
	return stmt, nil
}
