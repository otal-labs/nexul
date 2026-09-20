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
			Title:          m.Title,
			WhenToUse:      m.WhenToUse,
			Body:           m.Body,
			AlwaysIncluded: int64(boolInt(m.AlwaysIncluded)),
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

func (r *MemoriesRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]*memories.Memory, error) {
	rows, err := r.q.ListMemoriesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list memories for workspace %s: %w", workspaceID, err)
	}
	return toMemoriesList(rows), nil
}

func (r *MemoriesRepo) ListByProject(ctx context.Context, projectID, workspaceID string) ([]*memories.Memory, error) {
	rows, err := r.q.ListMemoriesByProject(ctx, sqlcgen.ListMemoriesByProjectParams{
		ProjectID:   sql.NullString{String: projectID, Valid: true},
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("list memories for project %s: %w", projectID, err)
	}
	return toMemoriesList(rows), nil
}

func (r *MemoriesRepo) ListWorkspaceScoped(ctx context.Context, workspaceID string) ([]*memories.Memory, error) {
	rows, err := r.q.ListWorkspaceScopedMemories(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace-scoped memories for workspace %s: %w", workspaceID, err)
	}
	return toMemoriesList(rows), nil
}

func (r *MemoriesRepo) Update(ctx context.Context, m *memories.Memory, authorVia string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.UpdateMemory(ctx, sqlcgen.UpdateMemoryParams{
			Title: m.Title, WhenToUse: m.WhenToUse, Body: m.Body,
			AlwaysIncluded: int64(boolInt(m.AlwaysIncluded)),
			Version:        int64(m.Version),
			UpdatedBy:      m.UpdatedBy, UpdatedAt: m.UpdatedAt.Unix(), ID: m.ID,
		})
		if err != nil {
			return fmt.Errorf("update memory %s: %w", m.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("update memory %s: %w", m.ID, apperrs.ErrNotFound)
		}
		if err := insertMemoryVersion(ctx, q, m, authorVia); err != nil {
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

func (r *MemoriesRepo) ListVersions(ctx context.Context, memoryID string) ([]*memories.MemoryVersion, error) {
	rows, err := r.q.ListMemoryVersions(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("list memory versions %s: %w", memoryID, err)
	}
	out := make([]*memories.MemoryVersion, len(rows))
	for i, row := range rows {
		out[i] = toMemoryVersion(row)
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
	err := q.InsertMemoryVersion(ctx, sqlcgen.InsertMemoryVersionParams{
		ID:             ids.New(),
		MemoryID:       m.ID,
		Version:        int64(m.Version),
		Title:          m.Title,
		WhenToUse:      m.WhenToUse,
		Body:           m.Body,
		AlwaysIncluded: int64(boolInt(m.AlwaysIncluded)),
		AuthorID:       m.UpdatedBy,
		AuthorVia:      authorVia,
		CreatedAt:      m.UpdatedAt.Unix(),
	})
	if err != nil {
		return fmt.Errorf("insert memory version %s v%d: %w", m.ID, m.Version, classifyWriteErr(err))
	}
	return nil
}

func toMemoryVersion(row sqlcgen.MemoryVersion) *memories.MemoryVersion {
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
		CreatedAt:      time.Unix(row.CreatedAt, 0).UTC(),
	}
}

func toMemory(row sqlcgen.Memory) *memories.Memory {
	return &memories.Memory{
		ID:             row.ID,
		WorkspaceID:    row.WorkspaceID,
		ProjectID:      row.ProjectID.String,
		Title:          row.Title,
		WhenToUse:      row.WhenToUse,
		Body:           row.Body,
		AlwaysIncluded: row.AlwaysIncluded != 0,
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
