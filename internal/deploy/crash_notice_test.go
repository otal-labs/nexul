package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type postedNotice struct{ conversation, bot, key, body string }

type fakeNotices struct {
	posts    []postedNotice
	checked  []string
	checkErr error
	postErr  error
	urlErr   error
}

func (f *fakeNotices) Check(_ context.Context, workspaceID, conversationID, botID string) error {
	f.checked = append(f.checked, workspaceID+"/"+conversationID+"/"+botID)
	return f.checkErr
}

func (f *fakeNotices) Post(_ context.Context, conversationID, botID, key, body string) error {
	if f.postErr != nil {
		return f.postErr
	}
	f.posts = append(f.posts, postedNotice{conversationID, botID, key, body})
	return nil
}

func (f *fakeNotices) TicketURL(_ context.Context, ticketID string) (string, error) {
	return "https://nexul.example.com/acme/tickets/" + ticketID, f.urlErr
}

func newNoticeFixture(t *testing.T, mutate ...func(*Stack)) (crashFixture, *fakeNotices) {
	t.Helper()
	on := func(s *Stack) { s.CrashNoticeConversationID, s.CrashNoticeBotID = "c-alerts", "bot-1" }
	f := newCrashFixture(t, append([]func(*Stack){on}, mutate...)...)
	f.s.projects.(*fakeProjects).exists["proj-1"] = true
	notices := &fakeNotices{}
	f.s.SetCrashNotices(notices)
	return f, notices
}

func TestCrashNotices_PostEachMomentWithItsTicket(t *testing.T) {
	tests := []struct {
		name   string
		handle func(crashFixture) error
		want   string
	}{
		{"a crash", func(f crashFixture) error {
			return f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped))
		}, "`web` crashed on `api`: exit 137, out of memory. https://nexul.example.com/acme/tickets/t1"},
		{"a recovery", func(f crashFixture) error {
			return f.s.HandleContainerRecovered(t.Context(), outageEvent(t, TopicContainerRecovered))
		}, "`web` came back on `api` after 3m0s down. It crashed 2 more times meanwhile. https://nexul.example.com/acme/tickets/t1"},
		{"an error-rate crossing", func(f crashFixture) error {
			return f.s.HandleErrorRateCrossed(t.Context(), errorRateEventFor(t, TopicErrorRateCrossed))
		}, "`api` answered 12.3% of its requests with 5xx over the last 5 minutes (62 of 500), above its 5% limit. https://nexul.example.com/acme/tickets/t1"},
		{"an error-rate recovery", func(f crashFixture) error {
			return f.s.HandleErrorRateRecovered(t.Context(), errorRateEventFor(t, TopicErrorRateRecovered))
		}, "5xx on `api` is back under 5%; the worst window reached 12.3%. https://nexul.example.com/acme/tickets/t1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, notices := newNoticeFixture(t)
			f.links.links["svc-1/web"], f.links.links["svc-1/"+errorRateSubject] = "t1", "t1"

			require.NoError(t, tt.handle(f))

			require.Len(t, notices.posts, 1)
			assert.Equal(t, "c-alerts", notices.posts[0].conversation)
			assert.Equal(t, "bot-1", notices.posts[0].bot)
			assert.NotEmpty(t, notices.posts[0].key, "the event's id keys the post, so a redelivery posts nothing new")
			assert.Equal(t, tt.want, notices.posts[0].body)
		})
	}
}

func TestCrashNotices_WithCrashTicketsOff_PostWithoutALink(t *testing.T) {
	f, notices := newNoticeFixture(t, func(s *Stack) { s.CrashTickets = false })

	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped, func(p *OutageEvent) { p.OOM = false })))

	assert.Empty(t, f.desk.filed)
	require.Len(t, notices.posts, 1)
	assert.Equal(t, "`web` crashed on `api`: exit 137.", notices.posts[0].body)
}

func TestCrashNotices_WithNoConversation_PostNothing(t *testing.T) {
	f, notices := newNoticeFixture(t, func(s *Stack) { s.CrashNoticeConversationID, s.CrashNoticeBotID = "", "" })

	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped)))

	assert.Len(t, f.desk.filed, 1, "crash tickets carry on alone")
	assert.Empty(t, notices.posts)
}

func TestCrashNotices_PostFailures(t *testing.T) {
	tests := []struct {
		name    string
		postErr error
		urlErr  error
		wantErr bool
	}{
		{"a conversation or bot gone is dropped", apperrs.ErrNotFound, nil, false},
		{"a bot that may no longer post is dropped", apperrs.ErrForbidden, nil, false},
		{"a write that failed is retried", errors.New("database is locked"), nil, true},
		{"a ticket that can't be linked is retried", nil, errors.New("database is locked"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, notices := newNoticeFixture(t)
			notices.postErr, notices.urlErr = tt.postErr, tt.urlErr

			err := f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped))

			assert.Equal(t, tt.wantErr, err != nil, "error: %v", err)
		})
	}
}

func TestUpdateStack_CrashNoticeTarget(t *testing.T) {
	t.Run("a new target is checked in the stack's workspace", func(t *testing.T) {
		f, notices := newNoticeFixture(t, func(s *Stack) { s.CrashNoticeConversationID, s.CrashNoticeBotID = "", "" })
		stack, err := f.stacks.GetByID(t.Context(), "svc-1")
		require.NoError(t, err)
		stack.CrashNoticeConversationID, stack.CrashNoticeBotID = " c-alerts ", "bot-1"

		updated, err := f.s.UpdateStack(t.Context(), *stack)

		require.NoError(t, err)
		assert.Equal(t, "c-alerts", updated.CrashNoticeConversationID)
		assert.Equal(t, []string{"ws-1/c-alerts/bot-1"}, notices.checked)
	})
	t.Run("a target left as it was is not checked again", func(t *testing.T) {
		f, notices := newNoticeFixture(t)
		notices.checkErr = apperrs.ErrForbidden
		stack, err := f.stacks.GetByID(t.Context(), "svc-1")
		require.NoError(t, err)
		stack.Name = "api-renamed"

		_, err = f.s.UpdateStack(t.Context(), *stack)

		require.NoError(t, err)
		assert.Empty(t, notices.checked)
	})
	t.Run("a refused target is not saved", func(t *testing.T) {
		f, notices := newNoticeFixture(t)
		notices.checkErr = apperrs.ErrForbidden
		stack, err := f.stacks.GetByID(t.Context(), "svc-1")
		require.NoError(t, err)
		stack.CrashNoticeConversationID = "c-other"

		_, err = f.s.UpdateStack(t.Context(), *stack)

		require.ErrorIs(t, err, apperrs.ErrForbidden)
		stored, err := f.stacks.GetByID(t.Context(), "svc-1")
		require.NoError(t, err)
		assert.Equal(t, "c-alerts", stored.CrashNoticeConversationID)
	})
}

func TestStackValidate_CrashNoticeTarget(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Stack)
	}{
		{"a conversation on a stack outside every project", func(s *Stack) { s.ProjectID, s.CrashNoticeConversationID = "", "c-alerts" }},
		{"a bot without its conversation", func(s *Stack) { s.CrashNoticeBotID = "bot-1" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validStack()
			tt.mutate(&s)
			assert.ErrorIs(t, s.Validate(), apperrs.ErrInvalid)
		})
	}
}
