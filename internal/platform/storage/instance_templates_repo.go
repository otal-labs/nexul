package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/templates"
)

var _ templates.Repo = (*InstanceTemplatesRepo)(nil)

// InstanceTemplatesRepo persists the edited instance templates (ADR 0103).
type InstanceTemplatesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *InstanceTemplatesRepo) Get(ctx context.Context, kind, key string) (*templates.Record, error) {
	row, err := r.q.GetInstanceTemplate(ctx, sqlcgen.GetInstanceTemplateParams{Kind: kind, Key: key})
	if err != nil {
		return nil, fmt.Errorf("get instance template %s %s: %w", kind, key, notFoundIfNoRows(err))
	}
	return toInstanceTemplate(row), nil
}

func (r *InstanceTemplatesRepo) List(ctx context.Context) ([]*templates.Record, error) {
	rows, err := r.q.ListInstanceTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list instance templates: %w", err)
	}
	out := make([]*templates.Record, len(rows))
	for i, row := range rows {
		out[i] = toInstanceTemplate(row)
	}
	return out, nil
}

func (r *InstanceTemplatesRepo) Save(ctx context.Context, t *templates.Record, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpsertInstanceTemplate(ctx, sqlcgen.UpsertInstanceTemplateParams{
			Kind: t.Kind, Key: t.Key, Body: t.Body, UpdatedBy: t.UpdatedBy, UpdatedAt: t.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save instance template %s %s: %w", t.Kind, t.Key, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func (r *InstanceTemplatesRepo) Delete(ctx context.Context, kind, key string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteInstanceTemplate(ctx, sqlcgen.DeleteInstanceTemplateParams{Kind: kind, Key: key}); err != nil {
			return fmt.Errorf("delete instance template %s %s: %w", kind, key, err)
		}
		return insertOutboxRows(ctx, tx, evts)
	})
}

func toInstanceTemplate(row sqlcgen.InstanceTemplate) *templates.Record {
	return &templates.Record{Kind: row.Kind, Key: row.Key, Body: row.Body, UpdatedBy: row.UpdatedBy, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC()}
}
