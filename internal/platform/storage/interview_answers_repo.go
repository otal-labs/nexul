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

func (r *MemoriesRepo) ListAnswers(ctx context.Context, projectID string) ([]*memories.InterviewAnswer, error) {
	rows, err := r.q.ListInterviewAnswers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list interview answers for project %s: %w", projectID, err)
	}
	out := make([]*memories.InterviewAnswer, 0, len(rows))
	for _, row := range rows {
		a, err := toInterviewAnswer(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *MemoriesRepo) UpsertAnswer(ctx context.Context, a *memories.InterviewAnswer, evts ...eventbus.OutboxEvent) (*memories.InterviewAnswer, error) {
	selected, err := json.Marshal(a.Selected)
	if err != nil {
		return nil, fmt.Errorf("encode picked values: %w", err)
	}
	var row sqlcgen.InterviewAnswer
	err = r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		row, err = r.q.WithTx(tx).UpsertInterviewAnswer(ctx, sqlcgen.UpsertInterviewAnswerParams{
			ID: a.ID, WorkspaceID: a.WorkspaceID, ProjectID: a.ProjectID, Round: int64(a.Round), Question: a.Question,
			Selected: string(selected), FreeText: a.Text, Skipped: int64(boolInt(a.Skipped)),
			AnsweredBy: a.AnsweredBy, AnsweredAt: a.AnsweredAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("save interview answer: %w", classifyWriteErr(err))
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
	if err != nil {
		return nil, err
	}
	return toInterviewAnswer(row)
}

func (r *MemoriesRepo) UpdateAnswer(ctx context.Context, a *memories.InterviewAnswer, evts ...eventbus.OutboxEvent) (*memories.InterviewAnswer, error) {
	selected, err := json.Marshal(a.Selected)
	if err != nil {
		return nil, fmt.Errorf("encode picked values: %w", err)
	}
	var row sqlcgen.InterviewAnswer
	err = r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		row, err = r.q.WithTx(tx).UpdateInterviewAnswer(ctx, sqlcgen.UpdateInterviewAnswerParams{
			Selected: string(selected), FreeText: a.Text, Skipped: int64(boolInt(a.Skipped)),
			AnsweredBy: a.AnsweredBy, AnsweredAt: a.AnsweredAt.Unix(),
			ProjectID: a.ProjectID, Round: int64(a.Round), Question: a.Question,
		})
		if err != nil {
			return fmt.Errorf("update interview answer: %w", notFoundIfNoRows(err))
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
	if err != nil {
		return nil, err
	}
	return toInterviewAnswer(row)
}

func (r *MemoriesRepo) DeleteAnswer(ctx context.Context, projectID string, round int, question string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteInterviewAnswer(ctx, sqlcgen.DeleteInterviewAnswerParams{ProjectID: projectID, Round: int64(round), Question: question})
		if err != nil {
			return fmt.Errorf("delete interview answer: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("delete interview answer: %w", apperrs.ErrNotFound)
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func (r *MemoriesRepo) LastRound(ctx context.Context, projectID string) (int, error) {
	v, err := r.q.LastInterviewRound(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("last interview round for project %s: %w", projectID, err)
	}
	n, _, err := optionalInt(v)
	if err != nil {
		return 0, fmt.Errorf("last interview round for project %s: %w", projectID, err)
	}
	return int(n), nil
}

func (r *MemoriesRepo) InsertRound(ctx context.Context, answers []*memories.InterviewAnswer, evts ...eventbus.OutboxEvent) error {
	params := make([]sqlcgen.InsertInterviewAnswerParams, 0, len(answers))
	for _, a := range answers {
		options, err := json.Marshal(a.Options)
		if err != nil {
			return fmt.Errorf("encode follow-up options: %w", err)
		}
		selected, err := json.Marshal(a.Selected)
		if err != nil {
			return fmt.Errorf("encode picked values: %w", err)
		}
		params = append(params, sqlcgen.InsertInterviewAnswerParams{
			ID: a.ID, WorkspaceID: a.WorkspaceID, ProjectID: a.ProjectID, Round: int64(a.Round), Question: a.Question,
			Options: nullString(string(options)), MultiSelect: int64(boolInt(a.MultiSelect)), Why: nullString(a.Why),
			Selected: string(selected), FreeText: a.Text, Skipped: int64(boolInt(a.Skipped)),
			AnsweredBy: a.AnsweredBy, AnsweredAt: a.AnsweredAt.Unix(),
		})
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, p := range params {
			if err := q.InsertInterviewAnswer(ctx, p); err != nil {
				return fmt.Errorf("insert follow-up %q: %w", p.Question, classifyWriteErr(err))
			}
		}
		return enqueueMemoriesOutbox(ctx, tx, evts)
	})
}

func toInterviewAnswer(row sqlcgen.InterviewAnswer) (*memories.InterviewAnswer, error) {
	a := &memories.InterviewAnswer{
		ID: row.ID, WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, Round: int(row.Round), Question: row.Question,
		MultiSelect: row.MultiSelect != 0, Why: row.Why.String, Text: row.FreeText, Skipped: row.Skipped != 0,
		AnsweredBy: row.AnsweredBy, AnsweredAt: time.Unix(row.AnsweredAt, 0).UTC(),
	}
	if err := json.Unmarshal([]byte(row.Selected), &a.Selected); err != nil {
		return nil, fmt.Errorf("decode picked values of interview answer %s: %w", row.ID, err)
	}
	if !row.Options.Valid {
		return a, nil
	}
	if err := json.Unmarshal([]byte(row.Options.String), &a.Options); err != nil {
		return nil, fmt.Errorf("decode options of interview answer %s: %w", row.ID, err)
	}
	return a, nil
}
