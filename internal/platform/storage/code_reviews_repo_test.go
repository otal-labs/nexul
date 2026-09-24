package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/codereview"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
)

var fixedNow = time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

func mustCreateTicket(t *testing.T, s *Store, id string) string {
	t.Helper()
	ctx := context.Background()
	svc := tickets.NewService(s.Tickets, s.Statuses, nil)
	tk, err := svc.Create(ctx, "project-general", "ticket "+id, "", "", "")
	require.NoError(t, err)
	return tk.ID
}

func validReview(id string, number int) *codereview.CodeReview {
	return &codereview.CodeReview{
		ID:        id,
		PRNumber:  number,
		Repo:      "acme/app",
		Status:    codereview.StatusPending,
		CreatedAt: fixedNow,
	}
}

func TestCodeReviewsRepo_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	t.Run("round trips a review", func(t *testing.T) {
		rv := validReview("r-1", 42)
		require.NoError(t, s.CodeReviews.Create(ctx, rv))

		got, err := s.CodeReviews.Get(ctx, "r-1")
		require.NoError(t, err)
		assert.Equal(t, "r-1", got.ID)
		assert.Equal(t, 42, got.PRNumber)
		assert.Equal(t, "acme/app", got.Repo)
		assert.Equal(t, codereview.StatusPending, got.Status)
		assert.Equal(t, fixedNow, got.CreatedAt)
	})

	t.Run("duplicate pr is a no-op", func(t *testing.T) {
		require.NoError(t, s.CodeReviews.Create(ctx, validReview("r-2", 43)))
		dup := validReview("r-2b", 43)
		require.NoError(t, s.CodeReviews.Create(ctx, dup))

		var n int
		require.NoError(t, s.db.QueryRow(`SELECT COUNT(*) FROM code_reviews WHERE repo = 'acme/app' AND pr_number = 43`).Scan(&n))
		assert.Equal(t, 1, n, "redelivered create must stay idempotent")
	})
}

func TestCodeReviewsRepo_Get_Missing(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	_, err := s.CodeReviews.Get(ctx, "nope")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestCodeReviewsRepo_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	require.NoError(t, s.CodeReviews.Create(ctx, validReview("r-1", 42)))

	t.Run("persists status and reviewer", func(t *testing.T) {
		require.NoError(t, s.CodeReviews.UpdateStatus(ctx, "r-1", codereview.StatusApproved, "alice"))
		got, err := s.CodeReviews.Get(ctx, "r-1")
		require.NoError(t, err)
		assert.Equal(t, codereview.StatusApproved, got.Status)
		assert.Equal(t, "alice", got.Reviewer)
	})

	t.Run("missing review is not found", func(t *testing.T) {
		err := s.CodeReviews.UpdateStatus(ctx, "nope", codereview.StatusMerged, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestCodeReviewsRepo_List(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	t1 := mustCreateTicket(t, s, "1")
	t2 := mustCreateTicket(t, s, "2")

	require.NoError(t, s.CodeReviews.Create(ctx, validReview("r-1", 42)))
	require.NoError(t, s.CodeReviews.Create(ctx, validReview("r-2", 43)))
	require.NoError(t, s.CodeReviews.UpdateStatus(ctx, "r-2", codereview.StatusMerged, ""))

	t.Run("lists by ticket through the pr links", func(t *testing.T) {
		require.NoError(t, s.Tickets.LinkPR(ctx, t1, tickets.PRRef{Owner: "acme", Repo: "app", Number: 42}, tickets.PRStateOpen))
		require.NoError(t, s.Tickets.LinkPR(ctx, t2, tickets.PRRef{Owner: "acme", Repo: "app", Number: 42}, tickets.PRStateOpen))
		rs, err := s.CodeReviews.ListByTicket(ctx, t1)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, "r-1", rs[0].ID)
	})

	t.Run("lists by pr", func(t *testing.T) {
		rs, err := s.CodeReviews.ListByPR(ctx, "acme/app", 42)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, "r-1", rs[0].ID)
	})

	t.Run("empty for unknown ticket or pr", func(t *testing.T) {
		rs, err := s.CodeReviews.ListByTicket(ctx, "none")
		require.NoError(t, err)
		assert.Empty(t, rs)
		rs, err = s.CodeReviews.ListByPR(ctx, "acme/app", 999)
		require.NoError(t, err)
		assert.Empty(t, rs)
	})
}
