package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// mainFolderName is the default folder every project is created with.
const mainFolderName = "Main"

func (r *DocsRepo) ListFolders(ctx context.Context, projectID string) ([]*docs.Folder, error) {
	rows, err := r.q.ListDocFoldersByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list folders for project %s: %w", projectID, err)
	}
	out := make([]*docs.Folder, 0, len(rows))
	for _, row := range rows {
		out = append(out, toFolder(row))
	}
	return out, nil
}

func (r *DocsRepo) GetFolder(ctx context.Context, id string) (*docs.Folder, error) {
	row, err := r.q.GetDocFolder(ctx, id)
	if err != nil {
		return nil, notFoundIfNoRows(err)
	}
	return toFolder(row), nil
}

func (r *DocsRepo) DefaultFolder(ctx context.Context, projectID string) (*docs.Folder, error) {
	row, err := r.q.GetDefaultDocFolder(ctx, projectID)
	if err != nil {
		return nil, notFoundIfNoRows(err)
	}
	return toFolder(row), nil
}

func (r *DocsRepo) CreateFolder(ctx context.Context, f *docs.Folder, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := insertDocFolder(ctx, r.q.WithTx(tx), f); err != nil {
			return err
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) RenameFolder(ctx context.Context, f *docs.Folder, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RenameDocFolder(ctx, sqlcgen.RenameDocFolderParams{Name: f.Name, UpdatedAt: f.UpdatedAt.Unix(), ID: f.ID})
		if err != nil {
			return fmt.Errorf("rename folder %s: %w", f.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("rename folder %s: %w", f.ID, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) DeleteFolder(ctx context.Context, id, toFolderID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.MoveDocsToFolder(ctx, sqlcgen.MoveDocsToFolderParams{ToFolderID: toFolderID, FromFolderID: id}); err != nil {
			return fmt.Errorf("move docs out of folder %s: %w", id, err)
		}
		n, err := q.DeleteDocFolder(ctx, id)
		if err != nil {
			return fmt.Errorf("delete folder %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete folder %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

// SetDocFolder moves a doc without bumping its version or its updated time: a move is not an edit.
func (r *DocsRepo) SetDocFolder(ctx context.Context, docID, folderID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetDocFolder(ctx, sqlcgen.SetDocFolderParams{FolderID: folderID, ID: docID})
		if err != nil {
			return fmt.Errorf("move doc %s: %w", docID, err)
		}
		if n == 0 {
			return fmt.Errorf("move doc %s: %w", docID, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func insertDocFolder(ctx context.Context, q *sqlcgen.Queries, f *docs.Folder) error {
	err := q.CreateDocFolder(ctx, sqlcgen.CreateDocFolderParams{
		ID: f.ID, ProjectID: f.ProjectID, Name: f.Name, IsDefault: int64(boolInt(f.IsDefault)),
		CreatedAt: f.CreatedAt.Unix(), UpdatedAt: f.UpdatedAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert folder %s: %w", f.Name, classifyWriteErr(err))
	}
	return nil
}

func toFolder(row sqlcgen.DocFolder) *docs.Folder {
	return &docs.Folder{
		ID: row.ID, ProjectID: row.ProjectID, Name: row.Name, IsDefault: row.IsDefault != 0,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}
