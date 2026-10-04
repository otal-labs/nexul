package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

func (r *DocsRepo) ListClarificationRounds(ctx context.Context, docID string) ([]*docs.ClarificationRound, error) {
	rows, err := r.q.ListDocClarificationRounds(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list clarification rounds of doc %s: %w", docID, err)
	}
	out := make([]*docs.ClarificationRound, 0, len(rows))
	for _, row := range rows {
		out = append(out, &docs.ClarificationRound{
			DocID: row.DocID, Round: int(row.Round), StartedBy: row.StartedBy, TrailID: row.TrailID,
			StartedAt: time.Unix(row.StartedAt, 0).UTC(), Running: row.Running != 0, TookLock: row.TookLock != 0,
			AnythingElse: row.AnythingElse, AnythingElseBy: row.AnythingElseBy, AnythingElseAt: optionalTime(row.AnythingElseAt),
			AnythingElseReply: row.AnythingElseReply, NoGapsAt: optionalTime(row.NoGapsAt),
			ClosedBy: row.ClosedBy, ClosedAt: optionalTime(row.ClosedAt),
		})
	}
	return out, nil
}

func (r *DocsRepo) ListClarificationQuestions(ctx context.Context, docID string) ([]*docs.ClarificationQuestion, error) {
	rows, err := r.q.ListDocClarificationQuestions(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list clarification questions of doc %s: %w", docID, err)
	}
	out := make([]*docs.ClarificationQuestion, 0, len(rows))
	for _, row := range rows {
		q, err := toClarificationQuestion(row)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func (r *DocsRepo) GetClarificationQuestion(ctx context.Context, id string) (*docs.ClarificationQuestion, error) {
	row, err := r.q.GetDocClarificationQuestion(ctx, id)
	if err != nil {
		return nil, notFoundIfNoRows(err)
	}
	return toClarificationQuestion(row)
}

func (r *DocsRepo) CreateClarificationRound(ctx context.Context, round *docs.ClarificationRound, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).InsertDocClarificationRound(ctx, sqlcgen.InsertDocClarificationRoundParams{
			DocID: round.DocID, Round: int64(round.Round), StartedBy: round.StartedBy, TrailID: round.TrailID,
			StartedAt: round.StartedAt.Unix(), Running: int64(boolInt(round.Running)), TookLock: int64(boolInt(round.TookLock)),
		})
		if err != nil {
			return fmt.Errorf("insert round %d of doc %s: %w", round.Round, round.DocID, classifyWriteErr(err))
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) SaveClarification(ctx context.Context, rounds []*docs.ClarificationRound, questions []*docs.ClarificationQuestion, evts ...eventbus.OutboxEvent) error {
	inserts := make([]sqlcgen.InsertDocClarificationQuestionParams, 0, len(questions))
	for _, q := range questions {
		options, err := json.Marshal(q.Options)
		if err != nil {
			return fmt.Errorf("encode options of question %s: %w", q.ID, err)
		}
		inserts = append(inserts, sqlcgen.InsertDocClarificationQuestionParams{
			ID: q.ID, DocID: q.DocID, Round: int64(q.Round), Position: int64(q.Position), Question: q.Question, Why: q.Why,
			Options: string(options), MultiSelect: int64(boolInt(q.MultiSelect)),
		})
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, round := range rounds {
			n, err := q.UpdateDocClarificationRound(ctx, sqlcgen.UpdateDocClarificationRoundParams{
				Running: int64(boolInt(round.Running)), AnythingElse: round.AnythingElse, AnythingElseBy: round.AnythingElseBy,
				AnythingElseAt: unixOrZero(round.AnythingElseAt), AnythingElseReply: round.AnythingElseReply,
				NoGapsAt: unixOrZero(round.NoGapsAt), ClosedBy: round.ClosedBy, ClosedAt: unixOrZero(round.ClosedAt),
				DocID: round.DocID, Round: int64(round.Round),
			})
			if err != nil {
				return fmt.Errorf("update round %d of doc %s: %w", round.Round, round.DocID, err)
			}
			if n == 0 {
				return fmt.Errorf("update round %d of doc %s: %w", round.Round, round.DocID, apperrs.ErrNotFound)
			}
		}
		for _, p := range inserts {
			if err := q.InsertDocClarificationQuestion(ctx, p); err != nil {
				return fmt.Errorf("insert question %q: %w", p.Question, classifyWriteErr(err))
			}
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) DeleteClarificationRound(ctx context.Context, docID string, round int, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteDocClarificationRound(ctx, sqlcgen.DeleteDocClarificationRoundParams{DocID: docID, Round: int64(round)})
		if err != nil {
			return fmt.Errorf("delete round %d of doc %s: %w", round, docID, err)
		}
		if n == 0 {
			return fmt.Errorf("delete round %d of doc %s: %w", round, docID, apperrs.ErrNotFound)
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func (r *DocsRepo) SaveClarificationAnswer(ctx context.Context, question *docs.ClarificationQuestion, roundAnswered *eventbus.OutboxEvent, evts ...eventbus.OutboxEvent) error {
	selected, err := json.Marshal(append([]string{}, question.Selected...))
	if err != nil {
		return fmt.Errorf("encode picked values: %w", err)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.UpdateDocClarificationAnswer(ctx, sqlcgen.UpdateDocClarificationAnswerParams{
			Selected: string(selected), FreeText: question.Text, Skipped: int64(boolInt(question.Skipped)),
			AnsweredBy: question.AnsweredBy, AnsweredAt: unixOrZero(question.AnsweredAt), ID: question.ID,
		})
		if err != nil {
			return fmt.Errorf("save answer to question %s: %w", question.ID, err)
		}
		if n == 0 {
			return fmt.Errorf("save answer to question %s: %w", question.ID, apperrs.ErrNotFound)
		}
		if roundAnswered != nil {
			pending, err := q.CountPendingDocClarificationQuestions(ctx, sqlcgen.CountPendingDocClarificationQuestionsParams{DocID: question.DocID, Round: int64(question.Round)})
			if err != nil {
				return fmt.Errorf("count pending questions of doc %s round %d: %w", question.DocID, question.Round, err)
			}
			if pending == 0 {
				evts = append(evts, *roundAnswered)
			}
		}
		return enqueueDocsOutbox(ctx, tx, evts)
	})
}

func toClarificationQuestion(row sqlcgen.DocClarificationQuestion) (*docs.ClarificationQuestion, error) {
	q := &docs.ClarificationQuestion{
		ID: row.ID, DocID: row.DocID, Round: int(row.Round), Position: int(row.Position), Question: row.Question, Why: row.Why,
		MultiSelect: row.MultiSelect != 0, Text: row.FreeText, Skipped: row.Skipped != 0,
		AnsweredBy: row.AnsweredBy, AnsweredAt: optionalTime(row.AnsweredAt),
	}
	if err := json.Unmarshal([]byte(row.Options), &q.Options); err != nil {
		return nil, fmt.Errorf("decode options of question %s: %w", row.ID, err)
	}
	if err := json.Unmarshal([]byte(row.Selected), &q.Selected); err != nil {
		return nil, fmt.Errorf("decode picked values of question %s: %w", row.ID, err)
	}
	return q, nil
}

// optionalTime reads a unix-seconds column where 0 means never.
func optionalTime(unix int64) *time.Time {
	if unix == 0 {
		return nil
	}
	t := time.Unix(unix, 0).UTC()
	return &t
}

func unixOrZero(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}
