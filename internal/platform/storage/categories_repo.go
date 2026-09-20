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

var _ workspace.CategoryRepo = (*CategoriesRepo)(nil)

// CategoriesRepo implements workspace.CategoryRepo; deleting a category never deletes its tickets.
type CategoriesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *CategoriesRepo) Create(ctx context.Context, c *workspace.Category, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateCategory(ctx, sqlcgen.CreateCategoryParams{
			ID: c.ID, ProjectID: c.ProjectID, Name: c.Name, Position: int64(c.Position),
			Color: string(c.Color), CreatedAt: c.CreatedAt.Unix(), UpdatedAt: c.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert category %s: %w", c.ID, classifyWriteErr(err))
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *CategoriesRepo) Get(ctx context.Context, id string) (*workspace.Category, error) {
	row, err := r.q.GetCategory(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get category %s: %w", id, notFoundIfNoRows(err))
	}
	return toCategory(row), nil
}

func (r *CategoriesRepo) ListByProject(ctx context.Context, projectID string) ([]*workspace.Category, error) {
	rows, err := r.q.ListCategoriesByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list categories for project %s: %w", projectID, err)
	}
	return toCategories(rows), nil
}

func (r *CategoriesRepo) List(ctx context.Context) ([]*workspace.Category, error) {
	rows, err := r.q.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return toCategories(rows), nil
}

func (r *CategoriesRepo) Update(ctx context.Context, c *workspace.Category, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateCategory(ctx, sqlcgen.UpdateCategoryParams{
			Name: c.Name, Color: string(c.Color), UpdatedAt: c.UpdatedAt.Unix(), ID: c.ID,
		})
		if err != nil {
			return fmt.Errorf("update category %s: %w", c.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("update category %s: %w", c.ID, apperrs.ErrNotFound)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

// Delete removes a category. The tickets table's FK is ON DELETE SET NULL, so
// its tickets become uncategorized — they are never deleted.
func (r *CategoriesRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteCategory(ctx, id)
		if err != nil {
			return fmt.Errorf("delete category %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete category %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func (r *CategoriesRepo) Reorder(ctx context.Context, projectID string, ids []string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for i, id := range ids {
			n, err := q.ReorderCategoryPosition(ctx, sqlcgen.ReorderCategoryPositionParams{
				Position: int64(i), UpdatedAt: time.Now().Unix(), ID: id, ProjectID: projectID,
			})
			if err != nil {
				return fmt.Errorf("reorder category %s: %w", id, err)
			}
			if n == 0 {
				return fmt.Errorf("reorder category %s: %w", id, apperrs.ErrNotFound)
			}
		}
		return nil
	})
}

func (r *CategoriesRepo) CountTickets(ctx context.Context, categoryID string) (int, error) {
	n, err := r.q.CountCategoryTickets(ctx, nullString(categoryID))
	if err != nil {
		return 0, fmt.Errorf("count tickets for category %s: %w", categoryID, err)
	}
	return int(n), nil
}

// SetTicketCategory moves a ticket (empty categoryID uncategorizes), appended to the end of its new order.
func (r *CategoriesRepo) SetTicketCategory(ctx context.Context, ticketID, categoryID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := moveTicketCategory(ctx, tx, ticketID, categoryID); err != nil {
			return fmt.Errorf("move ticket %s to category %s: %w", ticketID, categoryID, err)
		}
		return enqueueWorkspaceOutbox(ctx, tx, evts)
	})
}

func toCategory(row sqlcgen.Category) *workspace.Category {
	return &workspace.Category{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Position:  int(row.Position),
		Color:     colors.Color(row.Color),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toCategories(rows []sqlcgen.Category) []*workspace.Category {
	var out []*workspace.Category
	for _, row := range rows {
		out = append(out, toCategory(row))
	}
	return out
}
