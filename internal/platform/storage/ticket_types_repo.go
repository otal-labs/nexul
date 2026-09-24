package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/workspace"
)

var _ workspace.TicketTypeRepo = (*TicketTypesRepo)(nil)

// TicketTypesRepo implements workspace.TicketTypeRepo; ticket types are project-scoped and ordered.
type TicketTypesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *TicketTypesRepo) Create(ctx context.Context, t *workspace.TicketType, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateTicketType(ctx, sqlcgen.CreateTicketTypeParams{
			ID: t.ID, ProjectID: t.ProjectID, Name: t.Name, Position: int64(t.Position),
			Color: string(t.Color), BodyTemplate: t.BodyTemplate, CreatedAt: t.CreatedAt.Unix(), UpdatedAt: t.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert ticket type %s: %w", t.ID, classifyWriteErr(err))
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *TicketTypesRepo) Get(ctx context.Context, id string) (*workspace.TicketType, error) {
	row, err := r.q.GetTicketType(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get ticket type %s: %w", id, notFoundIfNoRows(err))
	}
	return toTicketType(row), nil
}

func (r *TicketTypesRepo) ListByProject(ctx context.Context, projectID string) ([]*workspace.TicketType, error) {
	rows, err := r.q.ListTicketTypesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list ticket types for project %s: %w", projectID, err)
	}
	return toTicketTypes(rows), nil
}

func (r *TicketTypesRepo) Update(ctx context.Context, t *workspace.TicketType, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateTicketTypeMeta(ctx, sqlcgen.UpdateTicketTypeMetaParams{
			Name: t.Name, Color: string(t.Color), BodyTemplate: t.BodyTemplate, UpdatedAt: t.UpdatedAt.Unix(), ID: t.ID,
		})
		if err != nil {
			return fmt.Errorf("update ticket type %s: %w", t.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("update ticket type %s: %w", t.ID, apperrs.ErrNotFound)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *TicketTypesRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteTicketType(ctx, id)
		if err != nil {
			return fmt.Errorf("delete ticket type %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete ticket type %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *TicketTypesRepo) Reorder(ctx context.Context, projectID string, ids []string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for i, id := range ids {
			n, err := q.ReorderTicketTypePosition(ctx, sqlcgen.ReorderTicketTypePositionParams{
				Position: int64(i), UpdatedAt: time.Now().Unix(), ID: id, ProjectID: projectID,
			})
			if err != nil {
				return fmt.Errorf("reorder ticket type %s: %w", id, err)
			}
			if n == 0 {
				return fmt.Errorf("reorder ticket type %s: %w", id, apperrs.ErrNotFound)
			}
		}
		return nil
	})
}

func (r *TicketTypesRepo) CountTickets(ctx context.Context, typeID string) (int, error) {
	n, err := r.q.CountTicketTypeTickets(ctx, nullString(typeID))
	if err != nil {
		return 0, fmt.Errorf("count tickets for type %s: %w", typeID, err)
	}
	return int(n), nil
}

// BodyTemplate returns a type's body template; satisfies tickets' consumer-side TypeTemplates seam.
func (r *TicketTypesRepo) BodyTemplate(ctx context.Context, typeID string) (string, error) {
	t, err := r.Get(ctx, typeID)
	if err != nil {
		return "", err
	}
	return t.BodyTemplate, nil
}

func toTicketType(row sqlcgen.TicketType) *workspace.TicketType {
	return &workspace.TicketType{
		ID:           row.ID,
		ProjectID:    row.ProjectID,
		Name:         row.Name,
		Position:     int(row.Position),
		Color:        colors.Color(row.Color),
		BodyTemplate: row.BodyTemplate,
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toTicketTypes(rows []sqlcgen.TicketType) []*workspace.TicketType {
	var out []*workspace.TicketType
	for _, row := range rows {
		out = append(out, toTicketType(row))
	}
	return out
}
