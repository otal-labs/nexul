package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ docs.Repo = (*DocsRepo)(nil)

type DocsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *DocsRepo) Create(ctx context.Context, d *docs.Doc, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.CreateDoc(ctx, sqlcgen.CreateDocParams{
			ID:        d.ID,
			ProjectID: sql.NullString{String: d.ProjectID, Valid: d.ProjectID != ""},
			Title:     d.Title,
			Body:      d.Body,
			BodyMd:    richtext.SearchText(d.Body),
			Version:   int64(d.Version),
			Archived:  int64(boolInt(d.Archived)),
			CreatedAt: d.CreatedAt.Unix(),
			UpdatedAt: d.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert doc %s: %w", d.ID, classifyWriteErr(err))
		}
		if err := insertDocVersion(ctx, q, d); err != nil {
			return err
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) GetByID(ctx context.Context, id string) (*docs.Doc, error) {
	row, err := r.q.GetDoc(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get doc %s: %w", id, notFoundIfNoRows(err))
	}
	return toDoc(row), nil
}

func (r *DocsRepo) List(ctx context.Context) ([]*docs.Doc, error) {
	rows, err := r.q.ListDocs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list docs: %w", err)
	}
	return toDocs(rows), nil
}

// ListByProject returns the docs in a given project, oldest first.
func (r *DocsRepo) ListByProject(ctx context.Context, projectID string) ([]*docs.Doc, error) {
	rows, err := r.q.ListDocsByProject(ctx, nullString(projectID))
	if err != nil {
		return nil, fmt.Errorf("list docs for project %s: %w", projectID, err)
	}
	return toDocs(rows), nil
}

func (r *DocsRepo) Update(ctx context.Context, d *docs.Doc, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.UpdateDoc(ctx, sqlcgen.UpdateDocParams{
			Title: d.Title, Body: d.Body, BodyMd: richtext.SearchText(d.Body),
			Version: int64(d.Version), UpdatedAt: d.UpdatedAt.Unix(), ID: d.ID,
		})
		if err != nil {
			return fmt.Errorf("update doc %s: %w", d.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("update doc %s: %w", d.ID, apperrs.ErrNotFound)
		}
		if err := insertDocVersion(ctx, q, d); err != nil {
			return err
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

// SetArchived flips a doc's archived flag without bumping its version.
func (r *DocsRepo) SetArchived(ctx context.Context, id string, archived bool, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetDocArchived(ctx, sqlcgen.SetDocArchivedParams{
			Archived: int64(boolInt(archived)), UpdatedAt: time.Now().Unix(), ID: id,
		})
		if err != nil {
			return fmt.Errorf("set archived %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set archived %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

// CommitBody writes converged state without a version row; FTS re-indexes on commit, not per keystroke.
func (r *DocsRepo) CommitBody(ctx context.Context, d *docs.Doc, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).CommitDocBody(ctx, sqlcgen.CommitDocBodyParams{
			Title: d.Title, Body: d.Body, BodyMd: richtext.SearchText(d.Body),
			Version: int64(d.Version), UpdatedAt: d.UpdatedAt.Unix(), ID: d.ID,
		})
		if err != nil {
			return fmt.Errorf("commit body doc %s: %w", d.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("commit body doc %s: %w", d.ID, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

// CreateNamedVersion snapshots the doc under a name/author and advances its version counter to a fresh key.
func (r *DocsRepo) CreateNamedVersion(ctx context.Context, v *docs.DocVersion) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.InsertNamedDocVersion(ctx, sqlcgen.InsertNamedDocVersionParams{
			DocID: v.DocID, Version: int64(v.Version), Title: v.Title, Body: v.Body,
			Name: v.Name, AuthorID: v.AuthorID, CreatedAt: v.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert named version %s v%d: %w", v.DocID, v.Version, classifyWriteErr(err))
		}
		n, err := q.BumpDocVersion(ctx, sqlcgen.BumpDocVersionParams{Version: int64(v.Version), ID: v.DocID})
		if err != nil {
			return fmt.Errorf("bump doc %s version: %w", v.DocID, err)
		}
		if n == 0 {
			return fmt.Errorf("bump doc %s version: %w", v.DocID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *DocsRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteDoc(ctx, id)
		if err != nil {
			return fmt.Errorf("delete doc %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete doc %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) ListVersions(ctx context.Context, docID string) ([]*docs.DocVersion, error) {
	rows, err := r.q.ListDocVersions(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list doc versions %s: %w", docID, err)
	}
	var out []*docs.DocVersion
	for _, row := range rows {
		out = append(out, &docs.DocVersion{
			DocID: row.DocID, Version: int(row.Version), Title: row.Title, Body: row.Body,
			Name: row.Name, AuthorID: row.AuthorID, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out, nil
}

func (r *DocsRepo) GetVersion(ctx context.Context, docID string, version int) (*docs.DocVersion, error) {
	row, err := r.q.GetDocVersion(ctx, sqlcgen.GetDocVersionParams{DocID: docID, Version: int64(version)})
	if err != nil {
		return nil, fmt.Errorf("get doc %s version %d: %w", docID, version, notFoundIfNoRows(err))
	}
	return &docs.DocVersion{
		DocID: row.DocID, Version: int(row.Version), Title: row.Title, Body: row.Body,
		Name: row.Name, AuthorID: row.AuthorID, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}, nil
}

func toDoc(row sqlcgen.Doc) *docs.Doc {
	return &docs.Doc{
		ID:        row.ID,
		ProjectID: row.ProjectID.String,
		Title:     row.Title,
		Body:      row.Body,
		Version:   int(row.Version),
		Archived:  row.Archived != 0,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toDocs(rows []sqlcgen.Doc) []*docs.Doc {
	var out []*docs.Doc
	for _, row := range rows {
		out = append(out, toDoc(row))
	}
	return out
}

func insertDocVersion(ctx context.Context, q *sqlcgen.Queries, d *docs.Doc) error {
	err := q.InsertDocVersion(ctx, sqlcgen.InsertDocVersionParams{
		DocID: d.ID, Version: int64(d.Version), Title: d.Title, Body: d.Body, CreatedAt: d.UpdatedAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert doc version %s v%d: %w", d.ID, d.Version, classifyWriteErr(err))
	}
	return nil
}

func enqueueDocsOutbox(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}
