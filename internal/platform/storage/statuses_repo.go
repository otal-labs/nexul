package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/workspace"
)

var _ workspace.StatusRepo = (*StatusesRepo)(nil)

// StatusesRepo implements workspace.StatusRepo; statuses use a stable id so a rename never rewrites tickets.
type StatusesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *StatusesRepo) Create(ctx context.Context, s *workspace.Status, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateStatus(ctx, sqlcgen.CreateStatusParams{
			ID: s.ID, ProjectID: s.ProjectID, Name: s.Name, Position: int64(s.Position),
			Kind: string(s.Kind), Icon: string(s.Icon), CreatedAt: s.CreatedAt.Unix(), UpdatedAt: s.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert status %s: %w", s.ID, classifyWriteErr(err))
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *StatusesRepo) Get(ctx context.Context, id string) (*workspace.Status, error) {
	row, err := r.q.GetStatus(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get status %s: %w", id, notFoundIfNoRows(err))
	}
	return toStatus(row), nil
}

func (r *StatusesRepo) ListByProject(ctx context.Context, projectID string) ([]*workspace.Status, error) {
	rows, err := r.q.ListStatusesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list statuses for project %s: %w", projectID, err)
	}
	return toStatuses(rows), nil
}

func (r *StatusesRepo) Update(ctx context.Context, s *workspace.Status, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateStatus(ctx, sqlcgen.UpdateStatusParams{
			Name: s.Name, Kind: string(s.Kind), Icon: string(s.Icon), UpdatedAt: s.UpdatedAt.Unix(), ID: s.ID,
		})
		if err != nil {
			return fmt.Errorf("update status %s: %w", s.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("update status %s: %w", s.ID, apperrs.ErrNotFound)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *StatusesRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteStatus(ctx, id)
		if err != nil {
			return fmt.Errorf("delete status %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete status %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *StatusesRepo) Reorder(ctx context.Context, projectID string, ids []string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for i, id := range ids {
			n, err := q.ReorderStatusPosition(ctx, sqlcgen.ReorderStatusPositionParams{
				Position: int64(i), UpdatedAt: time.Now().Unix(), ID: id, ProjectID: projectID,
			})
			if err != nil {
				return fmt.Errorf("reorder status %s: %w", id, err)
			}
			if n == 0 {
				return fmt.Errorf("reorder status %s: %w", id, apperrs.ErrNotFound)
			}
		}
		return nil
	})
}

func (r *StatusesRepo) CountTickets(ctx context.Context, statusID string) (int, error) {
	n, err := r.q.CountStatusTickets(ctx, statusID)
	if err != nil {
		return 0, fmt.Errorf("count tickets in status %s: %w", statusID, err)
	}
	return int(n), nil
}

// Exists reports whether a status id is a configured column; satisfies tickets' consumer-side StatusStore seam.
func (r *StatusesRepo) Exists(ctx context.Context, id string) (bool, error) {
	n, err := r.q.StatusExists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check status %s: %w", id, err)
	}
	return n > 0, nil
}

func toStatus(row sqlcgen.Status) *workspace.Status {
	return &workspace.Status{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Position:  int(row.Position),
		Kind:      workspace.StatusKind(row.Kind),
		Icon:      workspace.StatusIcon(row.Icon),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toStatuses(rows []sqlcgen.Status) []*workspace.Status {
	var out []*workspace.Status
	for _, row := range rows {
		out = append(out, toStatus(row))
	}
	return out
}
