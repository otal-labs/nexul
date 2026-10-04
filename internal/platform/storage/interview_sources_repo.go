package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

func (r *MemoriesRepo) ListSources(ctx context.Context, projectID string) ([]*memories.InterviewSource, error) {
	rows, err := r.q.ListInterviewSources(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list interview sources for project %s: %w", projectID, err)
	}
	out := make([]*memories.InterviewSource, 0, len(rows))
	for _, row := range rows {
		out = append(out, toInterviewSource(row))
	}
	return out, nil
}

func (r *MemoriesRepo) GetSource(ctx context.Context, id string) (*memories.InterviewSource, error) {
	row, err := r.q.GetInterviewSource(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get interview source %s: %w", id, notFoundIfNoRows(err))
	}
	return toInterviewSource(row), nil
}

func (r *MemoriesRepo) CountSources(ctx context.Context, projectID string) (int, error) {
	n, err := r.q.CountInterviewSources(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("count interview sources for project %s: %w", projectID, err)
	}
	return int(n), nil
}

func (r *MemoriesRepo) InsertSource(ctx context.Context, src *memories.InterviewSource, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).InsertInterviewSource(ctx, sqlcgen.InsertInterviewSourceParams{
			ID: src.ID, WorkspaceID: src.WorkspaceID, ProjectID: src.ProjectID, Kind: src.Kind, Ref: src.Ref,
			Label: src.Label, Body: src.Body, Stance: src.Stance, AddedBy: src.AddedBy,
			AddedAt: src.AddedAt.Unix(), UpdatedAt: src.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert interview source: %w", classifyWriteErr(err))
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) UpdateSource(ctx context.Context, src *memories.InterviewSource, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateInterviewSource(ctx, sqlcgen.UpdateInterviewSourceParams{
			Stance: src.Stance, Label: src.Label, UpdatedAt: src.UpdatedAt.Unix(), ID: src.ID,
		})
		if err != nil {
			return fmt.Errorf("update interview source %s: %w", src.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("update interview source %s: %w", src.ID, apperrs.ErrNotFound)
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) DeleteSource(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteInterviewSource(ctx, id)
		if err != nil {
			return fmt.Errorf("delete interview source %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete interview source %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) ListDrafts(ctx context.Context, projectID string) ([]*memories.InterviewDraft, error) {
	rows, err := r.q.ListInterviewDrafts(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list interview drafts for project %s: %w", projectID, err)
	}
	out := make([]*memories.InterviewDraft, 0, len(rows))
	for _, row := range rows {
		d, err := toInterviewDraft(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *MemoriesRepo) GetDraft(ctx context.Context, id string) (*memories.InterviewDraft, error) {
	row, err := r.q.GetInterviewDraft(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get interview draft %s: %w", id, notFoundIfNoRows(err))
	}
	return toInterviewDraft(row)
}

func (r *MemoriesRepo) SaveDrafts(ctx context.Context, drafts []*memories.InterviewDraft, evts ...eventbus.OutboxEvent) error {
	params := make([]sqlcgen.UpsertInterviewDraftParams, 0, len(drafts))
	for _, d := range drafts {
		selected, err := json.Marshal(d.Selected)
		if err != nil {
			return fmt.Errorf("encode picked values: %w", err)
		}
		sourceIDs, err := json.Marshal(d.SourceIDs)
		if err != nil {
			return fmt.Errorf("encode source ids: %w", err)
		}
		params = append(params, sqlcgen.UpsertInterviewDraftParams{
			ID: d.ID, WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, Question: d.Question,
			Selected: string(selected), FreeText: d.Text, SourceIds: string(sourceIDs), WhereLine: d.Where,
			TrailID: d.TrailID, DraftedBy: d.DraftedBy, DraftedAt: d.DraftedAt.Unix(),
		})
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, p := range params {
			if err := q.UpsertInterviewDraft(ctx, p); err != nil {
				return fmt.Errorf("save draft %q: %w", p.Question, classifyWriteErr(err))
			}
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) DeleteDraft(ctx context.Context, id string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteInterviewDraft(ctx, id)
		if err != nil {
			return fmt.Errorf("delete interview draft %s: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("delete interview draft %s: %w", id, apperrs.ErrNotFound)
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func toInterviewSource(row sqlcgen.InterviewSource) *memories.InterviewSource {
	return &memories.InterviewSource{
		ID: row.ID, WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, Kind: row.Kind, Ref: row.Ref,
		Label: row.Label, Body: row.Body, Stance: row.Stance, AddedBy: row.AddedBy,
		AddedAt: time.Unix(row.AddedAt, 0).UTC(), UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

func toInterviewDraft(row sqlcgen.InterviewDraft) (*memories.InterviewDraft, error) {
	d := &memories.InterviewDraft{
		ID: row.ID, WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, Question: row.Question, Text: row.FreeText,
		Where: row.WhereLine, TrailID: row.TrailID, DraftedBy: row.DraftedBy, DraftedAt: time.Unix(row.DraftedAt, 0).UTC(),
	}
	if err := json.Unmarshal([]byte(row.Selected), &d.Selected); err != nil {
		return nil, fmt.Errorf("decode picked values of interview draft %s: %w", row.ID, err)
	}
	if err := json.Unmarshal([]byte(row.SourceIds), &d.SourceIDs); err != nil {
		return nil, fmt.Errorf("decode source ids of interview draft %s: %w", row.ID, err)
	}
	return d, nil
}
