package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

type DeadLettersRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Put satisfies deadletter.Storer directly (no adapter layer): the dead
// letter shape is owned by the deadletter package, this repo just persists it.
func (r *DeadLettersRepo) Put(ctx context.Context, d deadletter.DeadLetter) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).PutDeadLetter(ctx, sqlcgen.PutDeadLetterParams{
			ID:       d.ID,
			Topic:    d.Topic,
			Payload:  d.Payload,
			Error:    d.Error,
			Attempts: int64(d.Attempts),
		})
		if err != nil {
			return fmt.Errorf("put dead letter %s: %w", d.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *DeadLettersRepo) Get(ctx context.Context, id string) (*deadletter.DeadLetter, error) {
	row, err := r.q.GetDeadLetter(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get dead letter %s: %w", id, notFoundIfNoRows(err))
	}
	return toDeadLetter(row), nil
}

func (r *DeadLettersRepo) List(ctx context.Context, limit, offset int) ([]deadletter.DeadLetter, error) {
	rows, err := r.q.ListDeadLetters(ctx, sqlcgen.ListDeadLettersParams{
		Limit: int64(limit), Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list dead letters: %w", err)
	}
	var out []deadletter.DeadLetter
	for _, row := range rows {
		out = append(out, *toDeadLetter(row))
	}
	return out, nil
}

func (r *DeadLettersRepo) Delete(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteDeadLetter(ctx, id)
		if err != nil {
			return fmt.Errorf("delete dead letter %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete dead letter %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toDeadLetter(row sqlcgen.DeadLetter) *deadletter.DeadLetter {
	return &deadletter.DeadLetter{
		ID:        row.ID,
		Topic:     row.Topic,
		Payload:   row.Payload,
		Error:     row.Error,
		Attempts:  int(row.Attempts),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}
