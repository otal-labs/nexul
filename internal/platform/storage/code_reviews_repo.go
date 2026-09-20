package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/codereview"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ codereview.Repo = (*CodeReviewsRepo)(nil)

type CodeReviewsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

// Create inserts a review record. The unique (repo, pr_number) constraint
// makes redelivered git events idempotent: a duplicate PR is a no-op.
func (r *CodeReviewsRepo) Create(ctx context.Context, review *codereview.CodeReview) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateCodeReview(ctx, sqlcgen.CreateCodeReviewParams{
			ID: review.ID, PrNumber: int64(review.PRNumber), Repo: review.Repo,
			Status: string(review.Status), Reviewer: review.Reviewer, CreatedAt: review.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert review %s: %w", review.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *CodeReviewsRepo) Get(ctx context.Context, id string) (*codereview.CodeReview, error) {
	row, err := r.q.GetCodeReview(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get review %s: %w", id, notFoundIfNoRows(err))
	}
	return toCodeReview(row), nil
}

func (r *CodeReviewsRepo) UpdateStatus(ctx context.Context, id string, status codereview.Status, reviewer string, evts ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.UpdateCodeReviewStatus(ctx, sqlcgen.UpdateCodeReviewStatusParams{
			Status: string(status), Reviewer: reviewer, ID: id,
		})
		if err != nil {
			return fmt.Errorf("update review %s status: %w", id, err)
		}
		if n == 0 {
			return fmt.Errorf("update review %s status: %w", id, apperrs.ErrNotFound)
		}
		for _, evt := range evts {
			if err := insertOutboxRow(ctx, tx, evt.ID, evt.Topic, evt.Payload); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListByTicket returns every PR's review linked to the ticket, joined since the association lives in tickets.
func (r *CodeReviewsRepo) ListByTicket(ctx context.Context, ticketID string) ([]*codereview.CodeReview, error) {
	rows, err := r.q.ListCodeReviewsByTicket(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list reviews for ticket %s: %w", ticketID, err)
	}
	return toCodeReviews(rows), nil
}

func (r *CodeReviewsRepo) ListByPR(ctx context.Context, repo string, number int) ([]*codereview.CodeReview, error) {
	rows, err := r.q.ListCodeReviewsByPR(ctx, sqlcgen.ListCodeReviewsByPRParams{Repo: repo, PrNumber: int64(number)})
	if err != nil {
		return nil, fmt.Errorf("list reviews for %s#%d: %w", repo, number, err)
	}
	return toCodeReviews(rows), nil
}

func toCodeReview(row sqlcgen.CodeReview) *codereview.CodeReview {
	return &codereview.CodeReview{
		ID:        row.ID,
		PRNumber:  int(row.PrNumber),
		Repo:      row.Repo,
		Status:    codereview.Status(row.Status),
		Reviewer:  row.Reviewer,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}
}

func toCodeReviews(rows []sqlcgen.CodeReview) []*codereview.CodeReview {
	var out []*codereview.CodeReview
	for _, row := range rows {
		out = append(out, toCodeReview(row))
	}
	return out
}
