package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/paging"
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
			FolderID:  d.FolderID,
			Title:     d.Title,
			Body:      d.Body,
			BodyMd:    richtext.SearchText(d.Body),
			Version:   int64(d.Version),
			Archived:  int64(boolInt(d.Archived)),
			CreatedBy: d.CreatedBy,
			CreatedAt: d.CreatedAt.Unix(),
			UpdatedAt: d.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert doc %s: %w", d.ID, classifyWriteErr(err))
		}
		if err := insertDocVersion(ctx, q, d); err != nil {
			return err
		}
		if err := addAutoWatcher(ctx, q, d.ID, d.CreatedBy, d.CreatedAt); err != nil {
			return err
		}
		if err := openSettle(ctx, q, d, true); err != nil {
			return err
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) GetByID(ctx context.Context, id string) (*docs.Doc, error) {
	row, err := r.q.GetDoc(ctx, id)
	if err != nil {
		return nil, notFoundIfNoRows(err)
	}
	return toDoc(row), nil
}

// ScopesOf returns the project and workspace of each known doc in ids, in one read; a doc in no project has neither.
func (r *DocsRepo) ScopesOf(ctx context.Context, ids []string) (map[string]access.DocScope, error) {
	out := make(map[string]access.DocScope, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.q.ListDocScopes(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("scopes of docs: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = access.DocScope{WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID}
	}
	return out, nil
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

func (r *DocsRepo) Update(ctx context.Context, d *docs.Doc, editorID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		// Two saves that read the same version would both claim the next one; numbering inside the write keeps both.
		cur, err := q.GetDoc(ctx, d.ID)
		if err != nil {
			return fmt.Errorf("update doc %s: %w", d.ID, notFoundIfNoRows(err))
		}
		d.Version = int(cur.Version) + 1
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
		if err := addAutoWatcher(ctx, q, d.ID, editorID, d.UpdatedAt); err != nil {
			return err
		}
		if err := openSettle(ctx, q, d, false); err != nil {
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

// SetLocked flips a doc's locked flag without bumping its version or its updated time.
func (r *DocsRepo) SetLocked(ctx context.Context, id string, locked bool, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetDocLocked(ctx, sqlcgen.SetDocLockedParams{Locked: int64(boolInt(locked)), ID: id})
		if err != nil {
			return fmt.Errorf("set locked %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("set locked %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

// CommitBody writes converged state without a version row; FTS re-indexes on commit, not per keystroke.
func (r *DocsRepo) CommitBody(ctx context.Context, d *docs.Doc, editorID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.CommitDocBody(ctx, sqlcgen.CommitDocBodyParams{
			Title: d.Title, Body: d.Body, BodyMd: richtext.SearchText(d.Body),
			Version: int64(d.Version), UpdatedAt: d.UpdatedAt.Unix(), ID: d.ID,
		})
		if err != nil {
			return fmt.Errorf("commit body doc %s: %w", d.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("commit body doc %s: %w", d.ID, apperrs.ErrNotFound)
		}
		if err := addAutoWatcher(ctx, q, d.ID, editorID, d.UpdatedAt); err != nil {
			return err
		}
		if err := openSettle(ctx, q, d, false); err != nil {
			return err
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
			return fmt.Errorf("delete doc %s: %w", id, classifyWriteErr(err))
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
		return nil, notFoundIfNoRows(err)
	}
	return &docs.DocVersion{
		DocID: row.DocID, Version: int(row.Version), Title: row.Title, Body: row.Body,
		Name: row.Name, AuthorID: row.AuthorID, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}, nil
}

// Page reads one window of the docs f and scope keep, and how many they keep in all: oldest first, or by relevance when
// f.Query searches them, which an archived doc never matches.
func (r *DocsRepo) Page(ctx context.Context, f docs.DocFilter, scope docs.DocScope, w paging.Window) ([]*docs.Doc, int, error) {
	q := pageQuery{from: "docs d", id: "d.id", order: "d.created_at, d.id"}
	if f.Query != "" {
		// hand-written: sqlc cannot express an FTS5 table as the operand of MATCH or bm25
		q = pageQuery{from: "docs_fts JOIN docs d ON d.rowid = docs_fts.rowid", id: "d.id", order: "bm25(docs_fts), d.id"}
		q.where("docs_fts MATCH ?", ftsQuery(f.Query))
	}
	if f.ProjectID != "" {
		q.where("d.project_id = ?", f.ProjectID)
	}
	if f.FolderID != "" {
		q.where("d.folder_id = ?", f.FolderID)
	}
	if f.Query != "" || !f.IncludeArchived {
		q.where("d.archived = 0")
	}
	if !scope.All {
		q.where("d.project_id IN (SELECT value FROM json_each(?))", idsJSON(scope.ProjectIDs))
	}
	if read := scope.Read; read != nil {
		q.where(`((d.project_id IN (SELECT value FROM json_each(?)) AND d.id NOT IN (SELECT value FROM json_each(?)))
			OR d.id IN (SELECT value FROM json_each(?)))`, idsJSON(read.ProjectIDs), idsJSON(read.Denied), idsJSON(read.Allowed))
	}
	ids, total, err := q.page(ctx, r.db, w)
	if err != nil {
		return nil, 0, fmt.Errorf("page docs: %w", err)
	}
	if len(ids) == 0 {
		return []*docs.Doc{}, total, nil
	}
	rows, err := r.q.ListDocsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("page docs: %w", err)
	}
	return inOrder(ids, toDocs(rows), func(d *docs.Doc) string { return d.ID }), total, nil
}

func toDoc(row sqlcgen.Doc) *docs.Doc {
	return &docs.Doc{
		ID:        row.ID,
		ProjectID: row.ProjectID.String,
		FolderID:  row.FolderID,
		Title:     row.Title,
		Body:      row.Body,
		Version:   int(row.Version),
		Archived:  row.Archived != 0,
		Locked:    row.Locked != 0,
		CreatedBy: row.CreatedBy,
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

// addAutoWatcher makes the person behind a save a watcher of the doc, unless they stopped watching it (ADR 0101).
func addAutoWatcher(ctx context.Context, q *sqlcgen.Queries, docID, userID string, at time.Time) error {
	if userID == "" {
		return nil
	}
	if err := q.AddAutoDocWatcher(ctx, sqlcgen.AddAutoDocWatcherParams{DocID: docID, UserID: userID, At: at.Unix()}); err != nil {
		return fmt.Errorf("add watcher %s to doc %s: %w", userID, docID, err)
	}
	return nil
}

func (r *DocsRepo) ListWatchers(ctx context.Context, docID string) ([]*docs.Watcher, error) {
	rows, err := r.q.ListDocWatchers(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list watchers of doc %s: %w", docID, err)
	}
	out := make([]*docs.Watcher, 0, len(rows))
	for _, row := range rows {
		out = append(out, &docs.Watcher{UserID: row.UserID, Source: docs.WatcherSource(row.Source), CreatedAt: time.Unix(row.CreatedAt, 0).UTC()})
	}
	return out, nil
}

func (r *DocsRepo) SetWatching(ctx context.Context, docID, userID string, watching bool, at time.Time, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetDocWatching(ctx, sqlcgen.SetDocWatchingParams{
			DocID: docID, UserID: userID, Watching: int64(boolInt(watching)), At: at.Unix(),
		})
		if err != nil {
			return fmt.Errorf("set %s watching doc %s: %w", userID, docID, classifyWriteErr(err))
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

// openSettle opens or moves d's settle window for its settler; first sticks once set, so a create stays a create.
func openSettle(ctx context.Context, q *sqlcgen.Queries, d *docs.Doc, first bool) error {
	if d.Settler == "" {
		return nil
	}
	err := q.UpsertDocSettle(ctx, sqlcgen.UpsertDocSettleParams{
		DocID: d.ID, DueAt: d.UpdatedAt.Add(docs.SettleWindow).Unix(), ActorID: d.Settler, First: int64(boolInt(first)),
	})
	if err != nil {
		return fmt.Errorf("open settle window for doc %s: %w", d.ID, err)
	}
	return nil
}

// NextSettleDue is when the earliest open settle window closes; false when none is open.
func (r *DocsRepo) NextSettleDue(ctx context.Context) (time.Time, bool, error) {
	v, err := r.q.NextDocSettleDue(ctx)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("next doc settle: %w", err)
	}
	due, ok, err := optionalInt(v)
	if err != nil || !ok {
		return time.Time{}, false, err
	}
	return time.Unix(due, 0), true, nil
}

// SettleDue closes every window due by now and writes build's events for those whose doc is not archived.
func (r *DocsRepo) SettleDue(ctx context.Context, now time.Time, build func([]docs.SettledEvent) []eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		rows, err := q.ListDueDocSettles(ctx, now.Unix())
		if err != nil {
			return fmt.Errorf("list due doc settles: %w", err)
		}
		if err := q.DeleteDueDocSettles(ctx, now.Unix()); err != nil {
			return fmt.Errorf("delete due doc settles: %w", err)
		}
		settled := make([]docs.SettledEvent, 0, len(rows))
		for _, row := range rows {
			settled = append(settled, docs.SettledEvent{
				Doc:   docs.WatchedDoc{ID: row.DocID, ProjectID: row.ProjectID.String, Title: row.Title},
				First: row.First == 1, ActorID: row.ActorID,
			})
		}
		return enqueueDocsOutbox(ctx, tx, build(settled))
	})
}
