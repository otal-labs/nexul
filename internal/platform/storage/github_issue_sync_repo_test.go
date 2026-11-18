package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/issuesync"
)

func TestIssueSyncRepo_Pushes_KeepALaterChangeAndTheFirstTry(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(ctx, newTestTicket("t-1", "doc-1")))
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

	_, ok, err := s.IssueSync.NextPushDue(ctx)
	require.NoError(t, err)
	assert.False(t, ok, "nothing waits")

	first := issuesync.TicketPush{TicketID: "t-1", Owner: "acme", Repo: "app", Create: true, Due: now}
	require.NoError(t, s.IssueSync.QueueTicketPush(ctx, first))
	require.NoError(t, s.IssueSync.QueueCommentPush(ctx, issuesync.CommentPush{MessageID: "m-1", TicketID: "t-1", Due: now.Add(-time.Second)}))
	next, ok, err := s.IssueSync.NextPushDue(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, now.Add(-time.Second), next, "the earliest of either kind")

	due, err := s.IssueSync.DueTicketPushes(ctx, now, 10)
	require.NoError(t, err)
	require.Equal(t, []issuesync.TicketPush{first}, due)

	tried := due[0]
	tried.Tried, tried.Attempts = now, 1
	require.NoError(t, s.IssueSync.DeferTicketPush(ctx, tried, tried.Due))
	later := now.Add(10 * time.Second)
	require.NoError(t, s.IssueSync.QueueTicketPush(ctx, issuesync.TicketPush{TicketID: "t-1", Owner: "acme", Repo: "app", Due: later}))
	require.NoError(t, s.IssueSync.DoneTicketPush(ctx, tried))

	left, err := s.IssueSync.DueTicketPushes(ctx, later, 10)
	require.NoError(t, err)
	require.Len(t, left, 1, "a push done after the ticket changed again stays for that change")
	assert.Equal(t, later, left[0].Due)
	assert.True(t, left[0].Create, "a create asked for stays asked for")
	assert.Equal(t, now, left[0].Tried, "the first try is kept, so the retry looks for what it made")

	backed := left[0]
	backed.Due, backed.Attempts = later.Add(time.Minute), 2
	require.NoError(t, s.IssueSync.DeferTicketPush(ctx, backed, later))
	none, err := s.IssueSync.DueTicketPushes(ctx, later, 10)
	require.NoError(t, err)
	assert.Empty(t, none, "a deferred push waits for its new due")

	comments, err := s.IssueSync.DueCommentPushes(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	require.NoError(t, s.IssueSync.DoneCommentPush(ctx, comments[0]))
	require.NoError(t, s.Tickets.Delete(ctx, "t-1", nil))
	_, ok, err = s.IssueSync.NextPushDue(ctx)
	require.NoError(t, err)
	assert.False(t, ok, "a deleted ticket takes its pushes")
}
