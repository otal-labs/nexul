package issuesync

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type fixture struct {
	svc      *Service
	repo     *fakeRepo
	tickets  *fakeTickets
	thread   *fakeThread
	comments *fakeComments
	host     *fakeHost
	now      time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		repo: newFakeRepo(), tickets: newFakeTickets(), thread: newFakeThread(), comments: newFakeComments(),
		host: &fakeHost{bot: "nexul-app[bot]"}, now: time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC),
	}
	f.svc = NewService(Deps{
		Repo: f.repo, Tickets: f.tickets, Fields: f.tickets, Thread: f.thread, Comments: f.comments,
		Members: fakeMembers{"alice-gh": "alice"}, Host: f.host, Gate: fakeGate{}, Repos: fakeRepos{"p1:acme/app": true},
	})
	f.svc.now = func() time.Time { return f.now }
	require.NoError(t, f.repo.SaveSync(t.Context(), &Sync{ProjectID: "p1", Owner: "acme", Repo: "app", Label: DefaultLabel, Mode: ModeLabel, Enabled: true}))
	return f
}

var issueAt = time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)

func labelled(opts ...func(*Issue)) Issue {
	i := Issue{ID: 900, Number: 7, Title: "Crash on save", Body: "Steps", State: "open", Labels: []string{"nexul", "bug"}, Author: "bob-gh", UpdatedAt: issueAt}
	for _, o := range opts {
		o(&i)
	}
	return i
}

func later(d time.Duration) func(*Issue) { return func(i *Issue) { i.UpdatedAt = issueAt.Add(d) } }

func issueEv(t *testing.T, action, sender string, i Issue) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"owner": "acme", "repo": "app", "action": action, "issue": i, "sender": map[string]string{"login": sender, "type": "User"}})
	require.NoError(t, err)
	return eventbus.Event{ID: "ev", Topic: "git.issue", Payload: raw}
}

func (f *fixture) deliver(t *testing.T, action string, i Issue) {
	t.Helper()
	require.NoError(t, f.svc.HandleIssue(t.Context(), issueEv(t, action, "bob-gh", i)))
}

func (f *fixture) ticket(t *testing.T) Ticket {
	t.Helper()
	l, err := f.repo.GetLink(t.Context(), "acme", "app", 7)
	require.NoError(t, err)
	tk, err := f.tickets.Get(t.Context(), l.TicketID)
	require.NoError(t, err)
	return tk
}

func TestHandleIssue_BadPayload_IsFatal(t *testing.T) {
	f := newFixture(t)
	err := f.svc.HandleIssue(t.Context(), eventbus.Event{Topic: "git.issue", Payload: []byte("{")})
	assert.ErrorIs(t, err, apperrs.ErrFatal)
	err = f.svc.HandleIssue(t.Context(), issueEv(t, "opened", "bob-gh", labelled(func(i *Issue) { i.Number = 0 })))
	assert.ErrorIs(t, err, apperrs.ErrFatal)
}

func TestHandleIssue_StoreFails_IsRetryable(t *testing.T) {
	f := newFixture(t)
	f.repo.err = errors.New("disk")
	assert.ErrorIs(t, f.svc.HandleIssue(t.Context(), issueEv(t, "opened", "bob-gh", labelled())), apperrs.ErrRetryable)
}

func TestHandleIssue_CreateFails_LinksNothing(t *testing.T) {
	f := newFixture(t)
	f.tickets.createErr = errors.New("busy")
	assert.ErrorIs(t, f.svc.HandleIssue(t.Context(), issueEv(t, "opened", "bob-gh", labelled())), apperrs.ErrRetryable)
	assert.Empty(t, f.repo.links)
}

func TestHandleIssue_ChangesNothing(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*fixture)
		sender string
		issue  Issue
	}{
		{"sync off", func(f *fixture) { f.repo.syncs["acme/app"].Enabled = false }, "bob-gh", labelled()},
		{"no sync for the repository", func(f *fixture) { delete(f.repo.syncs, "acme/app") }, "bob-gh", labelled()},
		{"sent by the App's own bot", func(*fixture) {}, "Nexul-App[bot]", labelled()},
		{"no trigger label", func(*fixture) {}, "bob-gh", labelled(func(i *Issue) { i.Labels = []string{"bug"} })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)
			require.NoError(t, f.svc.HandleIssue(t.Context(), issueEv(t, "opened", tt.sender, tt.issue)))
			assert.Zero(t, f.tickets.count())
			assert.Empty(t, f.repo.links)
		})
	}
}

func TestHandleIssue_LabelledIssue_FilesOneTicketWithItsFields(t *testing.T) {
	f := newFixture(t)
	issue := labelled(func(i *Issue) { i.Assignees = []string{"carol-gh", "alice-gh"} })
	f.deliver(t, "opened", issue)
	f.deliver(t, "opened", issue)
	f.deliver(t, "labeled", issue)

	require.Equal(t, 1, f.tickets.count(), "a redelivery files nothing more")
	tk := f.ticket(t)
	assert.Equal(t, "Crash on save", tk.Title)
	assert.Equal(t, []string{"bug"}, tk.Labels, "every label but the trigger")
	assert.Equal(t, "alice", tk.Developer, "the first assignee who is a member; carol-gh maps to nobody")
	assert.Equal(t, "bob-gh", f.tickets.reporters[tk.ID])
	assert.Equal(t, 2, f.tickets.writes, "labels and developer once, the text came with the ticket, and nothing for the repeats")
}

func TestHandleIssue_AllMode_FilesUnlabelledIssues(t *testing.T) {
	f := newFixture(t)
	f.repo.syncs["acme/app"].Mode = ModeAll
	f.deliver(t, "opened", labelled(func(i *Issue) { i.Labels = []string{"nexul", "bug"} }))
	assert.Equal(t, []string{"nexul", "bug"}, f.ticket(t).Labels, "all mode has no trigger label to drop")
}

func TestHandleIssue_TitleEdit_UpdatesTheTicket(t *testing.T) {
	f := newFixture(t)
	f.deliver(t, "opened", labelled())
	f.deliver(t, "edited", labelled(later(time.Minute), func(i *Issue) { i.Title = "Crash on save as" }))
	assert.Equal(t, "Crash on save as", f.ticket(t).Title)
}

func TestHandleIssue_StaleDelivery_IsSkipped(t *testing.T) {
	f := newFixture(t)
	f.deliver(t, "opened", labelled(later(time.Hour), func(i *Issue) { i.Title = "Newer" }))
	f.deliver(t, "edited", labelled(func(i *Issue) { i.Title = "Older" }))
	assert.Equal(t, "Newer", f.ticket(t).Title)
}

func TestHandleIssue_OutboundEcho_WritesNothing(t *testing.T) {
	f := newFixture(t)
	f.deliver(t, "opened", labelled())
	echo := labelled(later(time.Minute), func(i *Issue) { i.Title = "Written by Nexul" })
	l, err := f.repo.GetLink(t.Context(), "acme", "app", 7)
	require.NoError(t, err)
	l.OutboundHash = echo.Hash()
	require.NoError(t, f.repo.SaveLink(t.Context(), l))
	writes := f.tickets.writes
	f.deliver(t, "edited", echo)
	assert.Equal(t, writes, f.tickets.writes)
}

func TestHandleIssue_CloseAndReopen_NeverLeavesDone(t *testing.T) {
	f := newFixture(t)
	f.deliver(t, "opened", labelled())
	f.deliver(t, "closed", labelled(later(time.Minute), func(i *Issue) { i.State, i.StateReason = "closed", "not_planned" }))
	assert.True(t, f.ticket(t).Done)
	f.deliver(t, "reopened", labelled(later(2*time.Minute), func(i *Issue) { i.StateReason = "reopened" }))

	assert.True(t, f.ticket(t).Done, "a done ticket stays done")
	assert.Equal(t, []Stage{StageDone}, f.tickets.moves)
	assert.Equal(t, []string{"Closed on GitHub as not planned.", "Reopened on GitHub. Done tickets stay done; report a bug to follow up."}, f.thread.order)
}

func TestHandleIssue_ReopenOfATicketNotDone_MovesItToBacklog(t *testing.T) {
	f := newFixture(t)
	f.deliver(t, "opened", labelled())
	f.deliver(t, "reopened", labelled(later(time.Minute), func(i *Issue) { i.Body = "again" }))
	assert.Equal(t, []Stage{StageBacklog}, f.tickets.moves)
}

func TestHandleIssue_LabelRemovedThenAdded_PausesAndResumes(t *testing.T) {
	f := newFixture(t)
	f.deliver(t, "opened", labelled())
	f.deliver(t, "unlabeled", labelled(later(time.Minute), func(i *Issue) { i.Labels = []string{"bug"} }))
	f.deliver(t, "edited", labelled(later(2*time.Minute), func(i *Issue) { i.Labels, i.Title = []string{"bug"}, "While paused" }))
	l, err := f.repo.GetLink(t.Context(), "acme", "app", 7)
	require.NoError(t, err)
	assert.True(t, l.Paused)
	assert.Equal(t, "Crash on save", f.ticket(t).Title, "a paused link writes nothing")

	f.deliver(t, "labeled", labelled(later(3*time.Minute), func(i *Issue) { i.Title = "While paused" }))
	l, err = f.repo.GetLink(t.Context(), "acme", "app", 7)
	require.NoError(t, err)
	assert.False(t, l.Paused)
	assert.Equal(t, "While paused", f.ticket(t).Title)
	assert.Equal(t, 1, f.tickets.count())
}

func TestHandleIssue_DeletedOrTransferred_UnlinksAndKeepsTheTicket(t *testing.T) {
	for _, action := range []string{"deleted", "transferred"} {
		t.Run(action, func(t *testing.T) {
			f := newFixture(t)
			f.deliver(t, "opened", labelled())
			f.deliver(t, action, labelled(later(time.Minute)))
			assert.Empty(t, f.repo.links)
			assert.Equal(t, 1, f.tickets.count())
			require.Len(t, f.thread.order, 1)
			assert.Contains(t, f.thread.order[0], "no longer follows it")
		})
	}
}

func commentEv(t *testing.T, action, sender string, c Comment) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"owner": "acme", "repo": "app", "issue_number": 7, "action": action, "comment": c, "sender": map[string]string{"login": sender}})
	require.NoError(t, err)
	return eventbus.Event{ID: "ev", Topic: "git.issue_comment", Payload: raw}
}

func TestHandleIssueComment_FollowsCreateEditDelete(t *testing.T) {
	f := newFixture(t)
	f.deliver(t, "opened", labelled())
	c := Comment{ID: 55, Body: "Seen it too", Author: "bob-gh"}
	require.NoError(t, f.svc.HandleIssueComment(t.Context(), commentEv(t, "created", "bob-gh", c)))
	require.NoError(t, f.svc.HandleIssueComment(t.Context(), commentEv(t, "created", "bob-gh", c)))
	assert.Equal(t, map[string]string{"m55": "Seen it too"}, f.comments.messages)

	c.Body = "Seen it twice"
	require.NoError(t, f.svc.HandleIssueComment(t.Context(), commentEv(t, "edited", "bob-gh", c)))
	assert.Equal(t, "Seen it twice", f.comments.messages["m55"])

	require.NoError(t, f.svc.HandleIssueComment(t.Context(), commentEv(t, "deleted", "bob-gh", c)))
	assert.Empty(t, f.comments.messages)
	assert.Empty(t, f.repo.comments)
}

func TestHandleIssueComment_ChangesNothing(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*testing.T, *fixture)
		sender string
	}{
		{"issue not linked", func(*testing.T, *fixture) {}, "bob-gh"},
		{"sent by the App's own bot", func(t *testing.T, f *fixture) { f.deliver(t, "opened", labelled()) }, "nexul-app[bot]"},
		{"link paused", func(t *testing.T, f *fixture) {
			f.deliver(t, "opened", labelled())
			f.deliver(t, "unlabeled", labelled(later(time.Minute), func(i *Issue) { i.Labels = nil }))
		}, "bob-gh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(t, f)
			require.NoError(t, f.svc.HandleIssueComment(t.Context(), commentEv(t, "created", tt.sender, Comment{ID: 1, Body: "hi", Author: "bob-gh"})))
			assert.Empty(t, f.comments.messages)
		})
	}
}

func TestHandleIssueComment_BadPayload_IsFatal(t *testing.T) {
	f := newFixture(t)
	assert.ErrorIs(t, f.svc.HandleIssueComment(t.Context(), eventbus.Event{Payload: []byte("[")}), apperrs.ErrFatal)
	assert.ErrorIs(t, f.svc.HandleIssueComment(t.Context(), commentEv(t, "created", "bob-gh", Comment{})), apperrs.ErrFatal)
}

func TestSet_Refusals(t *testing.T) {
	bad := "sideways"
	empty := " "
	tests := []struct {
		name string
		gate error
		in   Input
		want error
	}{
		{"not an owner", apperrs.ErrForbidden, Input{Owner: "acme", Repo: "app"}, apperrs.ErrForbidden},
		{"no repository", nil, Input{}, apperrs.ErrInvalid},
		{"a repository the project doesn't have", nil, Input{Owner: "globex", Repo: "site"}, apperrs.ErrInvalid},
		{"an empty label", nil, Input{Owner: "acme", Repo: "app", Label: &empty}, apperrs.ErrInvalid},
		{"an unknown mode", nil, Input{Owner: "acme", Repo: "app", Mode: &bad}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.svc.Gate = fakeGate{err: tt.gate}
			_, err := f.svc.Set(t.Context(), "p1", tt.in)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestSet_NewSyncTakesDefaults_AndAPatchKeepsTheRest(t *testing.T) {
	f := newFixture(t)
	delete(f.repo.syncs, "acme/app")
	on := true
	got, err := f.svc.Set(t.Context(), "p1", Input{Owner: "acme", Repo: "app", Enabled: &on})
	require.NoError(t, err)
	assert.Equal(t, Sync{ProjectID: "p1", Owner: "acme", Repo: "app", Label: DefaultLabel, Mode: ModeLabel, Enabled: true}, *got)
	assert.Len(t, f.svc.kick, 1, "turning it on wakes the sweep")

	label := "triage"
	got, err = f.svc.Set(t.Context(), "p1", Input{Owner: "acme", Repo: "app", Label: &label})
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, "triage", got.Label)

	listed, err := f.svc.Syncs(t.Context(), "p1")
	require.NoError(t, err)
	assert.Equal(t, []*Sync{got}, listed)
}

func page(next, remaining int, reset time.Time, issues ...Issue) IssuePage {
	return IssuePage{Issues: issues, NextPage: next, RateRemaining: remaining, RateReset: reset}
}

func TestSweep_FirstLinksOpenLabelledIssues_ThenListsSince(t *testing.T) {
	f := newFixture(t)
	reset := f.now.Add(time.Hour)
	two := labelled(func(i *Issue) { i.Number, i.ID = 8, 901 })
	f.host.pages = []IssuePage{page(2, 4000, reset, labelled()), page(0, 4000, reset, two)}
	require.NoError(t, f.svc.Sweep(t.Context()))
	assert.Equal(t, 2, f.tickets.count())
	assert.Equal(t, IssueQuery{State: "open", Labels: []string{"nexul"}, Page: 1}, f.host.queries[0])
	swept := f.repo.syncs["acme/app"].LastSweptAt
	require.NotNil(t, swept)

	f.host.queries = nil
	require.NoError(t, f.svc.Sweep(t.Context()))
	assert.Empty(t, f.host.queries, "not due again yet")

	f.now = f.now.Add(SweepInterval)
	require.NoError(t, f.svc.Sweep(t.Context()))
	assert.Equal(t, IssueQuery{State: "all", Since: *swept, Page: 1}, f.host.queries[0])
	assert.Equal(t, 2, f.tickets.count(), "the sweep and the webhooks file each issue once")
}

func TestSweep_LowRateLimit_PausesUntilTheReset(t *testing.T) {
	f := newFixture(t)
	reset := f.now.Add(10 * time.Minute)
	f.host.pages = []IssuePage{page(2, 150, reset, labelled()), page(0, 4000, reset)}
	require.NoError(t, f.svc.Sweep(t.Context()))
	assert.Len(t, f.host.queries, 1, "stops before the next page")
	assert.Nil(t, f.repo.syncs["acme/app"].LastSweptAt, "an unfinished sweep runs again")

	require.NoError(t, f.svc.Sweep(t.Context()))
	assert.Len(t, f.host.queries, 1, "paused until the reset")

	f.now = reset
	f.host.pages[0].RateRemaining = 4000
	require.NoError(t, f.svc.Sweep(t.Context()))
	assert.Len(t, f.host.queries, 3)
	assert.NotNil(t, f.repo.syncs["acme/app"].LastSweptAt)
}

func TestSweep_RateLimitedError_Pauses(t *testing.T) {
	f := newFixture(t)
	f.host.err = &apperrs.RetryAfterError{Wait: time.Minute, Err: errors.New("403")}
	require.NoError(t, f.svc.Sweep(t.Context()))
	f.host.err = nil
	require.NoError(t, f.svc.Sweep(t.Context()))
	assert.Len(t, f.host.queries, 1)
}

func TestSweep_HostFails_ReportsIt(t *testing.T) {
	f := newFixture(t)
	f.host.err = errors.New("boom")
	assert.ErrorContains(t, f.svc.Sweep(t.Context()), "sweep acme/app")
	f.repo.err = errors.New("disk")
	assert.ErrorContains(t, f.svc.Sweep(t.Context()), "disk")
}
