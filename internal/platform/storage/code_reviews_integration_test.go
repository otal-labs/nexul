package storage_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/codereview"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

// newReviewBus creates an in-process bus over the real store, mirroring the
// composition root wiring. Handlers are registered with
// subscribeReviewHandlers once the service exists.
func newReviewBus(t *testing.T, s *storage.Store) *inprocess.Bus {
	bus := inprocess.New(inprocess.Options{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		DedupeStore:     s.ProcessedEvents,
		DeadLetterStore: s.DeadLetters,
	})
	t.Cleanup(func() { require.NoError(t, bus.Close()) })
	return bus
}

// subscribeReviewHandlers registers the codereview git-event consumers.
func subscribeReviewHandlers(t *testing.T, bus *inprocess.Bus, reviewSvc *codereview.Service) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, bus.Subscribe(ctx, gitprovider.TopicPROpened, func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePROpened(ctx, reviewSvc, ev)
	}))
	require.NoError(t, bus.Subscribe(ctx, gitprovider.TopicPRReviewSubmitted, func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePRReviewSubmitted(ctx, reviewSvc, ev)
	}))
	require.NoError(t, bus.Subscribe(ctx, gitprovider.TopicPRMerged, func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePRMerged(ctx, reviewSvc, ev)
	}))
	require.NoError(t, bus.Subscribe(ctx, gitprovider.TopicPRClosed, func(ctx context.Context, ev eventbus.Event) error {
		return codereview.HandlePRClosed(ctx, reviewSvc, ev)
	}))
}

// Acceptance criterion: PR opened -> per-PR review record -> shown on every
// linked ticket via the ticket_pr_links join.
func TestIntegration_PROPened_CreatesPerPRReviewShownOnTickets(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))

	tkSvc := tickets.NewService(s.Tickets, s.Statuses, nil)
	bus := newReviewBus(t, s)
	reviewSvc := codereview.NewService(s.CodeReviews)
	subscribeReviewHandlers(t, bus, reviewSvc)

	tk1, err := tkSvc.Create(ctx, "project-general", "Fix login", "wrap the auth middleware", "", "")
	require.NoError(t, err)
	tk2, err := tkSvc.Create(ctx, "project-general", "Add tests", "", "", "")
	require.NoError(t, err)

	pr := gitprovider.PREvent{Owner: "acme", Repo: "app", PR: gitprovider.PR{Number: 7, LinkedTicketIDs: []string{tk1.ID, tk2.ID}}}
	require.NoError(t, bus.Publish(ctx, gitprovider.TopicPROpened, pr))

	require.Eventually(t, func() bool {
		reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 7)
		return err == nil && len(reviews) == 1
	}, 2*time.Second, 10*time.Millisecond, "one per-PR record created from git.pr_opened")

	reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 7)
	require.NoError(t, err)
	require.Len(t, reviews, 1)
	assert.Equal(t, "acme/app", reviews[0].Repo)
	assert.Equal(t, 7, reviews[0].PRNumber)
	assert.Equal(t, codereview.StatusPending, reviews[0].Status)

	// The ticket join must surface the same record on every linked ticket.
	require.NoError(t, tkSvc.LinkPR(ctx, tk1.ID, tickets.PRRef{Owner: "acme", Repo: "app", Number: 7}))
	require.NoError(t, tkSvc.LinkPR(ctx, tk2.ID, tickets.PRRef{Owner: "acme", Repo: "app", Number: 7}))
	got, err := reviewSvc.ListByTicket(ctx, tk1.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, reviews[0].ID, got[0].ID)
	got, err = reviewSvc.ListByTicket(ctx, tk2.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, reviews[0].ID, got[0].ID)
}

// Acceptance criterion: git.pr_review_submitted is consumed — the aggregate
// status flips on the per-PR record, latest review wins.
func TestIntegration_ReviewSubmitted_FlipsStatus(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))

	bus := newReviewBus(t, s)
	reviewSvc := codereview.NewService(s.CodeReviews)
	subscribeReviewHandlers(t, bus, reviewSvc)

	pr := gitprovider.PREvent{Owner: "acme", Repo: "app", PR: gitprovider.PR{Number: 7}}
	require.NoError(t, bus.Publish(ctx, gitprovider.TopicPROpened, pr))
	require.Eventually(t, func() bool {
		reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 7)
		return err == nil && len(reviews) == 1
	}, 2*time.Second, 10*time.Millisecond)

	review := gitprovider.ReviewSubmittedEvent{Owner: "acme", Repo: "app", PR: gitprovider.ReviewRef{Number: 7, State: "approved", Reviewer: "alice"}}
	require.NoError(t, bus.Publish(ctx, gitprovider.TopicPRReviewSubmitted, review))
	require.Eventually(t, func() bool {
		reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 7)
		return err == nil && len(reviews) == 1 && reviews[0].Status == codereview.StatusApproved
	}, 2*time.Second, 10*time.Millisecond, "approved review flips the status")

	review.PR.State = "changes_requested"
	review.PR.Reviewer = "bob"
	require.NoError(t, bus.Publish(ctx, gitprovider.TopicPRReviewSubmitted, review))
	require.Eventually(t, func() bool {
		reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 7)
		return err == nil && len(reviews) == 1 && reviews[0].Status == codereview.StatusChangesRequested && reviews[0].Reviewer == "bob"
	}, 2*time.Second, 10*time.Millisecond, "latest review wins")
}

// PR merged -> the review record flips to merged; unmerged close -> closed.
func TestIntegration_PRMergedAndClosed_ResolveReview(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))

	bus := newReviewBus(t, s)
	reviewSvc := codereview.NewService(s.CodeReviews)
	subscribeReviewHandlers(t, bus, reviewSvc)

	for _, pr := range []gitprovider.PREvent{
		{Owner: "acme", Repo: "app", PR: gitprovider.PR{Number: 9}},
		{Owner: "acme", Repo: "app", PR: gitprovider.PR{Number: 10}},
	} {
		require.NoError(t, bus.Publish(ctx, gitprovider.TopicPROpened, pr))
	}
	require.Eventually(t, func() bool {
		reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 9)
		return err == nil && len(reviews) == 1
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, bus.Publish(ctx, gitprovider.TopicPRMerged, gitprovider.PREvent{Owner: "acme", Repo: "app", PR: gitprovider.PR{Number: 9}}))
	require.Eventually(t, func() bool {
		reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 9)
		return err == nil && len(reviews) == 1 && reviews[0].Status == codereview.StatusMerged
	}, 2*time.Second, 10*time.Millisecond, "merged event flips the review status")

	require.NoError(t, bus.Publish(ctx, gitprovider.TopicPRClosed, gitprovider.PREvent{Owner: "acme", Repo: "app", PR: gitprovider.PR{Number: 10}}))
	require.Eventually(t, func() bool {
		reviews, err := reviewSvc.ListByPR(ctx, "acme/app", 10)
		return err == nil && len(reviews) == 1 && reviews[0].Status == codereview.StatusClosed
	}, 2*time.Second, 10*time.Millisecond, "unmerged close moves the review to closed")
}
