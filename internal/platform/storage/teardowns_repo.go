package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/runner"
)

var _ runner.TeardownRepo = (*TeardownsRepo)(nil)

// TeardownsRepo persists the stack teardowns waiting for a machine's runner to connect.
type TeardownsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *TeardownsRepo) Save(ctx context.Context, machine string, f runner.Frame) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SavePendingStackTeardown(ctx, sqlcgen.SavePendingStackTeardownParams{
			Machine:   machine,
			Slug:      f.StackSlug,
			Strategy:  f.Strategy,
			CreatedAt: time.Now().UnixNano(),
		})
		if err != nil {
			return fmt.Errorf("save teardown %s on %s: %w", f.StackSlug, machine, err)
		}
		return nil
	})
}

func (r *TeardownsRepo) List(ctx context.Context, machine string) ([]runner.Frame, error) {
	rows, err := r.q.ListPendingStackTeardowns(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("list teardowns on %s: %w", machine, err)
	}
	out := make([]runner.Frame, 0, len(rows))
	for _, row := range rows {
		out = append(out, runner.Frame{Type: runner.FrameStackTeardown, StackSlug: row.Slug, Strategy: row.Strategy})
	}
	return out, nil
}

func (r *TeardownsRepo) Delete(ctx context.Context, machine, slug string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).DeletePendingStackTeardown(ctx, sqlcgen.DeletePendingStackTeardownParams{Machine: machine, Slug: slug})
		if err != nil {
			return fmt.Errorf("delete teardown %s on %s: %w", slug, machine, err)
		}
		return nil
	})
}
