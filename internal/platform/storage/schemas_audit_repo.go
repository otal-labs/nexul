package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/integrations"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ integrations.SchemaStore = (*EventSchemasRepo)(nil)
var _ integrations.AuditStore = (*AuditRepo)(nil)

// EventSchemasRepo persists the event-schema catalog; Publish leaves an existing topic/version untouched.
type EventSchemasRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *EventSchemasRepo) Publish(ctx context.Context, entry integrations.SchemaEntry) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).PublishSchema(ctx, sqlcgen.PublishSchemaParams{
			Topic: entry.Topic, Version: int64(entry.Version), Schema: entry.Schema, CreatedAt: entry.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("publish schema %s v%d: %w", entry.Topic, entry.Version, err)
		}
		return nil
	})
}

func (r *EventSchemasRepo) Catalog(ctx context.Context) ([]integrations.SchemaEntry, error) {
	rows, err := r.q.ListSchemaCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("list schema catalog: %w", err)
	}
	var out []integrations.SchemaEntry
	for _, row := range rows {
		out = append(out, toSchemaEntry(row))
	}
	return out, nil
}

func toSchemaEntry(row sqlcgen.EventSchema) integrations.SchemaEntry {
	return integrations.SchemaEntry{
		Topic:     row.Topic,
		Version:   int(row.Version),
		Schema:    row.Schema,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}

// AuditRepo persists audit-log rows.
type AuditRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *AuditRepo) Append(ctx context.Context, entry integrations.AuditEntry) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).AppendAudit(ctx, sqlcgen.AppendAuditParams{
			ID:        entry.ID,
			ActorType: entry.ActorType,
			ActorID:   entry.ActorID,
			TokenID:   sql.NullString{String: entry.TokenID, Valid: entry.TokenID != ""},
			Action:    entry.Action,
			CreatedAt: entry.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("append audit: %w", err)
		}
		return nil
	})
}

func (r *AuditRepo) List(ctx context.Context, limit int) ([]integrations.AuditEntry, error) {
	rows, err := r.q.ListAudit(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	var out []integrations.AuditEntry
	for _, row := range rows {
		out = append(out, toAuditEntry(row))
	}
	return out, nil
}

func toAuditEntry(row sqlcgen.AuditLog) integrations.AuditEntry {
	entry := integrations.AuditEntry{
		ID:        row.ID,
		ActorType: row.ActorType,
		ActorID:   row.ActorID,
		Action:    row.Action,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
	if row.TokenID.Valid {
		entry.TokenID = row.TokenID.String
	}
	return entry
}
