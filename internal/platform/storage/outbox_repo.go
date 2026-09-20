package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

type OutboxRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Unpublished satisfies outbox.Storer directly; the entry shape is owned by outbox, this repo just persists it.
func (r *OutboxRepo) Unpublished(ctx context.Context, limit int) ([]outbox.Entry, error) {
	rows, err := r.q.ListUnpublishedOutbox(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("list pending outbox: %w", err)
	}
	var out []outbox.Entry
	for _, row := range rows {
		out = append(out, outbox.Entry{
			ID: row.ID, Topic: row.Topic, Payload: row.Payload,
			CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out, nil
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).MarkOutboxPublished(ctx, id)
		if err != nil {
			return fmt.Errorf("mark outbox %s published: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("mark outbox %s published: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

// insertOutboxRow is the shared enqueue every repo's write transaction uses: same tx as the row it announces.
func insertOutboxRow(ctx context.Context, tx *sql.Tx, id, topic string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}
	if err := sqlcgen.New(tx).EnqueueOutbox(ctx, sqlcgen.EnqueueOutboxParams{ID: id, Topic: topic, Payload: b}); err != nil {
		return fmt.Errorf("enqueue outbox %s: %w", id, classifyWriteErr(err))
	}
	return nil
}
