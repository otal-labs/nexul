package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ memories.Repo = (*MemoriesRepo)(nil)

type MemoriesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *MemoriesRepo) Create(ctx context.Context, m *memories.Memory, authorVia string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.CreateMemory(ctx, sqlcgen.CreateMemoryParams{
			ID:             m.ID,
			WorkspaceID:    m.WorkspaceID,
			ProjectID:      sql.NullString{String: m.ProjectID, Valid: m.ProjectID != ""},
			Kind:           m.Kind,
			Title:          m.Title,
			WhenToUse:      m.WhenToUse,
			Body:           m.Body,
			AlwaysIncluded: int64(boolInt(m.AlwaysIncluded)),
			Footer:         int64(boolInt(m.Footer)),
			Version:        int64(m.Version),
			CreatedBy:      m.CreatedBy,
			CreatedAt:      m.CreatedAt.Unix(),
			UpdatedBy:      m.UpdatedBy,
			UpdatedAt:      m.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert memory %s: %w", m.ID, classifyWriteErr(err))
		}
		if err := insertMemoryVersion(ctx, q, m, authorVia); err != nil {
			return err
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) GetByID(ctx context.Context, id string) (*memories.Memory, error) {
	row, err := r.q.GetMemory(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get memory %s: %w", id, notFoundIfNoRows(err))
	}
	return toMemory(row), nil
}

func (r *MemoriesRepo) GetByProjectKind(ctx context.Context, projectID, kind string) (*memories.Memory, error) {
	row, err := r.q.GetMemoryByProjectKind(ctx, sqlcgen.GetMemoryByProjectKindParams{ProjectID: nullString(projectID), Kind: kind})
	if err != nil {
		return nil, fmt.Errorf("get %s memory for project %s: %w", kind, projectID, notFoundIfNoRows(err))
	}
	return toMemory(row), nil
}

func (r *MemoriesRepo) GetInterviewTemplate(ctx context.Context, workspaceID string) (*memories.InterviewTemplate, error) {
	row, err := r.q.GetInterviewTemplate(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get interview template for workspace %s: %w", workspaceID, notFoundIfNoRows(err))
	}
	return &memories.InterviewTemplate{
		WorkspaceID: row.WorkspaceID, Body: row.Body, UpdatedBy: row.UpdatedBy, UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

func (r *MemoriesRepo) SaveInterviewTemplate(ctx context.Context, t *memories.InterviewTemplate, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpsertInterviewTemplate(ctx, sqlcgen.UpsertInterviewTemplateParams{
			WorkspaceID: t.WorkspaceID, Body: t.Body, UpdatedBy: t.UpdatedBy, UpdatedAt: t.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save interview template for workspace %s: %w", t.WorkspaceID, classifyWriteErr(err))
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) DeleteInterviewTemplate(ctx context.Context, workspaceID string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteInterviewTemplate(ctx, workspaceID); err != nil {
			return fmt.Errorf("delete interview template for workspace %s: %w", workspaceID, err)
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]*memories.Memory, error) {
	rows, err := r.q.ListMemoriesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list memories for workspace %s: %w", workspaceID, err)
	}
	return toMemoriesList(rows), nil
}

func (r *MemoriesRepo) ListByProject(ctx context.Context, projectID string) ([]*memories.Memory, error) {
	rows, err := r.q.ListMemoriesByProject(ctx, nullString(projectID))
	if err != nil {
		return nil, fmt.Errorf("list memories for project %s: %w", projectID, err)
	}
	return toMemoriesList(rows), nil
}

// PageByProject reads one window of a project's memories, oldest first, and how many it has.
func (r *MemoriesRepo) PageByProject(ctx context.Context, projectID string, w paging.Window) ([]*memories.Memory, int, error) {
	rows, err := r.q.ListMemoriesPage(ctx, sqlcgen.ListMemoriesPageParams{ProjectID: nullString(projectID), Limit: int64(w.Limit), Offset: int64(w.Offset)})
	if err != nil {
		return nil, 0, fmt.Errorf("page memories for project %s: %w", projectID, err)
	}
	total, err := r.q.CountMemoriesByProject(ctx, nullString(projectID))
	if err != nil {
		return nil, 0, fmt.Errorf("count memories for project %s: %w", projectID, err)
	}
	return toMemoriesList(rows), int(total), nil
}

func (r *MemoriesRepo) Update(ctx context.Context, m *memories.Memory, authorVia string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		// Two saves that read the same version would both claim the next one; numbering inside the write keeps both.
		cur, err := q.GetMemory(ctx, m.ID)
		if err != nil {
			return fmt.Errorf("update memory %s: %w", m.ID, notFoundIfNoRows(err))
		}
		m.Version = int(cur.Version) + 1
		n, err := q.UpdateMemory(ctx, sqlcgen.UpdateMemoryParams{
			Title: m.Title, WhenToUse: m.WhenToUse, Body: m.Body,
			AlwaysIncluded: int64(boolInt(m.AlwaysIncluded)),
			Footer:         int64(boolInt(m.Footer)),
			Version:        int64(m.Version),
			UpdatedBy:      m.UpdatedBy, UpdatedAt: m.UpdatedAt.Unix(), ID: m.ID,
		})
		if err != nil {
			return fmt.Errorf("update memory %s: %w", m.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("update memory %s: %w", m.ID, apperrs.ErrNotFound)
		}
		if err := writeSaveMemoryVersion(ctx, q, m, authorVia); err != nil {
			return err
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) Delete(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteMemory(ctx, id)
		if err != nil {
			return fmt.Errorf("delete memory %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete memory %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) ListVersionSummaries(ctx context.Context, memoryID string) ([]memories.VersionSummary, error) {
	rows, err := r.q.ListMemoryVersionSummaries(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("list memory version summaries %s: %w", memoryID, err)
	}
	out := make([]memories.VersionSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, memories.VersionSummary{
			Version: int(row.Version), CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), Title: row.Title,
			EditorIDs: decodeEditorIDs(row.EditorIds), Via: row.AuthorVia,
			WordsAdded: nullableInt(row.WordsAdded), WordsRemoved: nullableInt(row.WordsRemoved), RestoredFrom: nullableInt(row.RestoredFrom),
		})
	}
	return out, nil
}

func (r *MemoriesRepo) GetVersion(ctx context.Context, memoryID string, version int) (*memories.MemoryVersion, error) {
	row, err := r.q.GetMemoryVersion(ctx, sqlcgen.GetMemoryVersionParams{MemoryID: memoryID, Version: int64(version)})
	if err != nil {
		return nil, fmt.Errorf("get memory %s version %d: %w", memoryID, version, notFoundIfNoRows(err))
	}
	return toMemoryVersion(row), nil
}

func insertMemoryVersion(ctx context.Context, q *sqlcgen.Queries, m *memories.Memory, authorVia string) error {
	size, err := memorySizer(q, m.ID).againstNewest(ctx, docVersionText(m.Title, m.Body))
	if err != nil {
		return fmt.Errorf("size memory version %s v%d: %w", m.ID, m.Version, err)
	}
	err = q.InsertMemoryVersion(ctx, sqlcgen.InsertMemoryVersionParams{
		ID:             ids.New(),
		MemoryID:       m.ID,
		Version:        int64(m.Version),
		Title:          m.Title,
		WhenToUse:      m.WhenToUse,
		Body:           m.Body,
		AlwaysIncluded: int64(boolInt(m.AlwaysIncluded)),
		AuthorID:       m.UpdatedBy,
		AuthorVia:      authorVia,
		EditorIds:      idsJSON(soleEditor(m.UpdatedBy)),
		WordsAdded:     size.added,
		WordsRemoved:   size.removed,
		RestoredFrom:   restoredFromOf(m.RestoredFrom),
		CreatedAt:      m.UpdatedAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert memory version %s v%d: %w", m.ID, m.Version, classifyWriteErr(err))
	}
	return nil
}

// writeSaveMemoryVersion appends the version a save makes, unless the memory already reads as its newest version, then prunes.
func writeSaveMemoryVersion(ctx context.Context, q *sqlcgen.Queries, m *memories.Memory, authorVia string) error {
	same, err := q.MemoryMatchesLatestVersion(ctx, sqlcgen.MemoryMatchesLatestVersionParams{
		ForMemory: m.ID, WantTitle: m.Title, WantWhenToUse: m.WhenToUse, WantBody: m.Body, WantAlwaysIncluded: int64(boolInt(m.AlwaysIncluded)),
	})
	if err != nil {
		return fmt.Errorf("compare memory %s with its newest version: %w", m.ID, err)
	}
	if same {
		return nil
	}
	if err := insertMemoryVersion(ctx, q, m, authorVia); err != nil {
		return err
	}
	err = memorySizer(q, m.ID).pruned(ctx, func() error {
		return q.PruneMemoryVersions(ctx, sqlcgen.PruneMemoryVersionsParams{ForMemory: m.ID, Keep: memories.KeptVersions})
	})
	if err != nil {
		return fmt.Errorf("prune memory %s versions: %w", m.ID, err)
	}
	return nil
}

func toMemoryVersion(row sqlcgen.GetMemoryVersionRow) *memories.MemoryVersion {
	return &memories.MemoryVersion{
		ID:             row.ID,
		MemoryID:       row.MemoryID,
		Version:        int(row.Version),
		Title:          row.Title,
		WhenToUse:      row.WhenToUse,
		Body:           row.Body,
		AlwaysIncluded: row.AlwaysIncluded != 0,
		AuthorID:       row.AuthorID,
		AuthorVia:      row.AuthorVia,
		EditorIDs:      decodeEditorIDs(row.EditorIds),
		CreatedAt:      time.Unix(row.CreatedAt, 0).UTC(),
	}
}

func toMemory(row sqlcgen.Memory) *memories.Memory {
	return &memories.Memory{
		ID:             row.ID,
		WorkspaceID:    row.WorkspaceID,
		ProjectID:      row.ProjectID.String,
		Kind:           row.Kind,
		Title:          row.Title,
		WhenToUse:      row.WhenToUse,
		Body:           row.Body,
		AlwaysIncluded: row.AlwaysIncluded != 0,
		Footer:         row.Footer != 0,
		Version:        int(row.Version),
		CreatedBy:      row.CreatedBy,
		CreatedAt:      time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedBy:      row.UpdatedBy,
		UpdatedAt:      time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toMemoriesList(rows []sqlcgen.Memory) []*memories.Memory {
	out := make([]*memories.Memory, 0, len(rows))
	for _, row := range rows {
		out = append(out, toMemory(row))
	}
	return out
}

func enqueueMemoriesOutbox(ctx context.Context, tx *sql.Tx, evts []eventbus.OutboxEvent) error {
	for _, evt := range evts {
		if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
			return err
		}
	}
	return nil
}
