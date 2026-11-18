package issuesync

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// linkedFixture is a ticket filed from issue #7 by the inbound sync, so it starts in step with its issue.
func linkedFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(onSync())
	handle(t, f, "opened", newIssue())
	require.Len(t, f.desk.tickets, 1)
	return f
}

func (f *fixture) push(t *testing.T) time.Duration {
	t.Helper()
	wait, err := f.svc.Push(t.Context())
	require.NoError(t, err)
	return wait
}

func (f *fixture) changed(t *testing.T, added ...string) {
	t.Helper()
	require.NoError(t, f.svc.TicketChanged(t.Context(), "t-1", "p-1", added))
}

func TestPush_ABurstOfEdits_MakesOneWriteOnceQuiet(t *testing.T) {
	f := linkedFixture(t)
	for n := range 20 {
		f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Body = fmt.Sprintf("Steps, take %d", n) })
		f.changed(t)
		f.now = f.now.Add(500 * time.Millisecond)
		f.push(t)
	}
	assert.Zero(t, f.github.writes(), "nothing goes out while the edits keep coming")

	wait := f.push(t)
	assert.Equal(t, Quiet-500*time.Millisecond, wait, "the loop sleeps until the last edit has rested")
	f.now = f.now.Add(wait)
	assert.Equal(t, time.Duration(-1), f.push(t), "nothing waits once it is written")

	require.Len(t, f.github.updates, 1)
	got := f.github.updates[0]
	assert.Equal(t, "Steps, take 19\n\n<!-- nexul:ticket:t-1 -->", got.Body)
	assert.Equal(t, []string{"bug", "nexul"}, got.Labels, "labels in full, plus the trigger label")
	assert.Nil(t, got.Assignees, "a ticket with no developer leaves the issue's assignees alone")
}

func TestPush_WritesNothing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, f *fixture)
	}{
		{"for a change GitHub itself made", func(t *testing.T, f *fixture) {
			handle(t, f, "edited", newIssue(withTitle("Renamed on GitHub"), later(time.Minute)))
			f.changed(t)
		}},
		{"for a ticket whose fields did not change", func(t *testing.T, f *fixture) { f.changed(t) }},
		{"for a paused ticket", func(t *testing.T, f *fixture) {
			handle(t, f, "unlabeled", newIssue(withLabels("bug"), later(time.Minute)))
			f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited in Nexul" })
			f.changed(t)
		}},
		{"for a repository whose sync was turned off", func(t *testing.T, f *fixture) {
			f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited in Nexul" })
			f.changed(t)
			sy := onSync()
			sy.Enabled = false
			f.repo.syncs["acme/app"] = sy
		}},
		{"for an unlinked ticket without the trigger label", func(t *testing.T, f *fixture) {
			f.desk.tickets["t-2"] = &fakeTicket{Fields: TicketFields{Title: "Local"}}
			require.NoError(t, f.svc.TicketChanged(t.Context(), "t-2", "p-1", []string{"bug"}))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := linkedFixture(t)
			tt.setup(t, f)
			f.now = f.now.Add(time.Hour)
			f.push(t)
			assert.Zero(t, f.github.writes())
			assert.Zero(t, f.repo.waiting(), "a push with nothing to write is dropped")
		})
	}
}

func TestPush_FieldMapping(t *testing.T) {
	f := linkedFixture(t)
	f.desk.edit("t-1", func(tk *fakeTicket) {
		tk.Stage, tk.Developer, tk.Fields.Labels = StageDone, "bob", []string{"ui", "nexul", "bug"}
	})
	f.changed(t)
	f.now = f.now.Add(Quiet)
	f.push(t)

	require.Len(t, f.github.updates, 1)
	got := f.github.updates[0]
	assert.True(t, got.Closed, "a done ticket closes its issue")
	assert.Equal(t, []string{"bob-gh"}, got.Assignees, "the developer is the one assignee, by their GitHub login")
	assert.Equal(t, []string{"bug", "ui", "nexul"}, got.Labels, "the trigger label once, however the ticket carries it")

	f.desk.edit("t-1", func(tk *fakeTicket) { tk.Developer = "carol" })
	f.changed(t)
	f.now = f.now.Add(Quiet)
	f.push(t)
	require.Len(t, f.github.updates, 2)
	assert.Nil(t, f.github.updates[1].Assignees, "a developer with no GitHub login leaves the assignees alone")
}

func TestPush_OwnWriteComesBackAsNothing(t *testing.T) {
	f := linkedFixture(t)
	f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited in Nexul" })
	f.changed(t)
	f.now = f.now.Add(Quiet)
	f.push(t)
	require.Len(t, f.github.updates, 1)
	applied := f.desk.applied

	echo := f.github.issue(7, f.github.updates[0])
	echo.UpdatedAt = f.now.Add(time.Second)
	require.NoError(t, f.svc.HandleIssue(t.Context(), "acme", "app", "edited", echo, "alice-gh"))
	assert.Equal(t, applied, f.desk.applied, "the issue as Nexul wrote it matches the outbound hash, so it changes nothing")
}

func TestPush_TriggerLabel_CreatesTheIssueOnceThoughTheFirstTryTimedOut(t *testing.T) {
	f := newFixture(onSync())
	f.desk.tickets["t-1"] = &fakeTicket{Fields: TicketFields{Title: "Made in Nexul", Body: "Body", Labels: []string{"nexul"}}}
	f.changed(t, "nexul")
	f.github.fail, f.github.accept = []error{apperrs.Retryable(errors.New("timeout"))}, true
	f.now = f.now.Add(Quiet)
	_, err := f.svc.Push(t.Context())
	require.NoError(t, err, "a retryable failure waits for its retry")
	require.Len(t, f.github.creates, 1, "GitHub took it though the answer was lost")

	f.now = f.now.Add(time.Hour)
	f.push(t)

	assert.Len(t, f.github.creates, 1, "the retry finds the issue by its marker instead of making another")
	link, err := f.repo.GetLinkByTicket(t.Context(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, 41, link.IssueNumber)
	assert.Equal(t, "Body\n\n<!-- nexul:ticket:t-1 -->", f.github.creates[0].Body)
	assert.Equal(t, []string{"nexul"}, f.github.creates[0].Labels)
	assert.Zero(t, f.repo.waiting())

	require.NoError(t, f.svc.HandleIssue(t.Context(), "acme", "app", "opened", f.issues.pages[1][0], "alice-gh"))
	assert.Len(t, f.desk.tickets, 1, "the new issue's own webhook files no second ticket")
}

func TestHandleIssue_MarkedIssueWithoutLink_ClaimsItsTicket(t *testing.T) {
	f := newFixture(onSync())
	f.desk.tickets["t-1"] = &fakeTicket{Fields: TicketFields{Title: "Made in Nexul"}}
	i := newIssue(func(i *Issue) { i.Body = "Body\n\n<!-- nexul:ticket:t-1 -->" })

	handle(t, f, "opened", i)

	assert.Len(t, f.desk.tickets, 1, "an issue Nexul made links back to its ticket when the link was lost")
	link, err := f.repo.GetLinkByTicket(t.Context(), "t-1")
	require.NoError(t, err)
	assert.Equal(t, 7, link.IssueNumber)
	assert.Equal(t, "Body", FieldsFromIssue(i, DefaultLabel).Body, "the marker never reaches the ticket")
}

func TestPush_DoneTicketGetsItsIssueClosedAfterMakingIt(t *testing.T) {
	f := newFixture(onSync())
	f.desk.tickets["t-1"] = &fakeTicket{Fields: TicketFields{Title: "Already done"}, Stage: StageDone}
	f.changed(t, "nexul")
	f.now = f.now.Add(Quiet)
	assert.Zero(t, f.push(t), "the close is due at once")
	f.now = f.now.Add(writeGap)
	f.push(t)

	require.Len(t, f.github.creates, 1)
	require.Len(t, f.github.updates, 1, "GitHub opens every new issue, so a second write closes it")
	assert.True(t, f.github.updates[0].Closed)
}

func TestPush_IssueGoneFromGitHub_UnlinksWithANote(t *testing.T) {
	f := linkedFixture(t)
	f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited" })
	f.changed(t)
	f.github.fail = []error{apperrs.ErrNotFound}
	f.now = f.now.Add(Quiet)
	f.push(t)

	_, err := f.repo.GetLinkByTicket(t.Context(), "t-1")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, f.desk.notes["github-issue acme/app#7 gone"], "no longer on GitHub")
	assert.Len(t, f.desk.tickets, 1, "the ticket stays")
}

func TestPush_RateLimited_WaitsOutRetryAfter(t *testing.T) {
	f := linkedFixture(t)
	f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited" })
	f.changed(t)
	f.github.fail = []error{&apperrs.RetryAfterError{Wait: 5 * time.Minute, Err: errors.New("secondary rate limit")}}
	f.now = f.now.Add(Quiet)

	assert.Equal(t, 5*time.Minute, f.push(t), "the next try is Retry-After away")
	f.now = f.now.Add(4 * time.Minute)
	f.push(t)
	assert.Empty(t, f.github.updates, "nothing goes out before Retry-After passes")
	assert.True(t, f.svc.pace.wait("acme", f.now, false) > 0, "the owner rests, for the sweep too")

	f.now = f.now.Add(time.Minute)
	f.push(t)
	assert.Len(t, f.github.updates, 1)
}

func TestPush_RefusedWrite_IsDroppedAndReported(t *testing.T) {
	f := linkedFixture(t)
	f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited" })
	f.changed(t)
	f.github.fail = []error{apperrs.ErrUnauthorized}
	f.now = f.now.Add(Quiet)

	_, err := f.svc.Push(t.Context())
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
	assert.Zero(t, f.repo.waiting())
}

func TestPush_RetryableFailures_BackOffThenGiveUp(t *testing.T) {
	f := linkedFixture(t)
	f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited" })
	f.changed(t)
	for range maxPushAttempts {
		f.github.fail = append(f.github.fail, apperrs.Retryable(errors.New("502")))
	}
	f.now = f.now.Add(Quiet)
	var waits []time.Duration
	for range maxPushAttempts - 1 {
		waits = append(waits, f.push(t))
		f.now = f.now.Add(time.Hour)
	}
	assert.Equal(t, []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute}, waits[:3])
	assert.Equal(t, 30*time.Minute, waits[6])
	_, err := f.svc.Push(t.Context())
	assert.ErrorIs(t, err, apperrs.ErrRetryable, "the last failure is reported")
	assert.Zero(t, f.repo.waiting(), "a write GitHub keeps failing is dropped; the next change queues it again")
}

func (f *fixture) said(t *testing.T, id, body string) {
	t.Helper()
	f.desk.mu.Lock()
	f.desk.thread[id] = Mirror{Person: true, Body: body}
	f.desk.mu.Unlock()
	require.NoError(t, f.svc.MessageChanged(t.Context(), "c-1", id))
}

func TestPush_Comment_PostsOnceThoughTheFirstTryTimedOut(t *testing.T) {
	f := linkedFixture(t)
	f.github.fail, f.github.accept = []error{apperrs.Retryable(errors.New("timeout"))}, true
	f.said(t, "m-1", "**Bob** (@bob-gh) wrote in Nexul:\n\nLooks fixed")
	f.push(t)
	f.now = f.now.Add(time.Hour)
	f.push(t)

	require.Len(t, f.github.posted, 1, "the retry finds the comment by its marker")
	assert.Equal(t, "**Bob** (@bob-gh) wrote in Nexul:\n\nLooks fixed\n\n<!-- nexul:message:m-1 -->", f.github.posted[0])
	cl, err := f.repo.GetCommentLinkByMessage(t.Context(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(7001), cl.GitHubCommentID)

	require.NoError(t, f.svc.HandleComment(t.Context(), "acme", "app", 7, "created", f.issues.comments[7][0], "nexul-app[bot]"))
	assert.Empty(t, f.desk.messages, "the comment's own webhook posts nothing back")
	f.issues.mu.Lock()
	f.issues.pages[1] = []Issue{newIssue(later(time.Hour))}
	f.issues.mu.Unlock()
	require.NoError(t, f.svc.Sweep(t.Context(), true))
	assert.Empty(t, f.desk.messages, "nor does the sweep, which skips the App's own comments")
}

func TestPush_Comment_EditAndDeleteFollow(t *testing.T) {
	f := linkedFixture(t)
	f.said(t, "m-1", "First")
	f.push(t)
	f.now = f.now.Add(time.Second)
	f.said(t, "m-1", "Second")
	f.push(t)
	f.now = f.now.Add(time.Second)
	f.desk.mu.Lock()
	f.desk.thread["m-1"] = Mirror{Person: true, Deleted: true}
	f.desk.mu.Unlock()
	require.NoError(t, f.svc.MessageChanged(t.Context(), "c-1", "m-1"))
	f.push(t)

	assert.Len(t, f.github.posted, 1)
	assert.Equal(t, []string{"Second\n\n<!-- nexul:message:m-1 -->"}, f.github.edited)
	assert.Equal(t, []int64{7001}, f.github.removed)
	_, err := f.repo.GetCommentLinkByMessage(t.Context(), "m-1")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPush_Comment_MirrorsOnlyPeoplesMessagesOnLinkedTickets(t *testing.T) {
	f := linkedFixture(t)
	f.desk.thread["m-agent"] = Mirror{Body: "An agent's reply"}
	require.NoError(t, f.svc.MessageChanged(t.Context(), "c-1", "m-agent"))
	require.NoError(t, f.svc.MessageChanged(t.Context(), "c-other", "m-elsewhere"))
	f.push(t)
	assert.Zero(t, f.github.writes())
	assert.Zero(t, f.repo.waiting())
}

func TestPush_PacesEachOwnersWrites(t *testing.T) {
	f := linkedFixture(t)
	f.said(t, "m-1", "One")
	f.said(t, "m-2", "Two")
	assert.Equal(t, writeGap, f.push(t), "the second comment waits a second behind the first")
	assert.Len(t, f.github.posted, 1)
	f.now = f.now.Add(writeGap)
	f.push(t)
	assert.Len(t, f.github.posted, 2)
}

func TestPacer_CapsCreationsAnHour(t *testing.T) {
	p := newPacer()
	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	now := start
	for range createsPerHour {
		require.Zero(t, p.wait("acme", now, true))
		p.took("acme", now, true)
		now = now.Add(writeGap)
	}
	assert.Equal(t, start.Add(time.Hour).Sub(now), p.wait("acme", now, true), "the next creation waits for the hour's first to age out")
	assert.Zero(t, p.wait("acme", now, false), "an edit is no creation")
	assert.Zero(t, p.wait("globex", now, true), "another installation has its own budget")
}

func TestHandleEvents_QueueWhatTheIssueTakes(t *testing.T) {
	f := linkedFixture(t)
	f.desk.tickets["t-2"] = &fakeTicket{}
	f.desk.threads["c-2"] = "t-2"
	events := []struct {
		topic   string
		payload string
	}{
		{TopicTicketUpdated, `{"ticket":{"id":"t-1","project_id":"p-1"}}`},
		{TopicTicketLabelsChanged, `{"ticket":{"id":"t-2","project_id":"p-1"},"added":["NEXUL"],"removed":[]}`},
		{TopicMessageCreated, `{"message":{"id":"m-1","conversation_id":"c-1","author_kind":"user"}}`},
		{TopicMessageCreated, `{"message":{"id":"m-2","conversation_id":"c-1","author_kind":"agent"}}`},
		{TopicMessageUpdated, `{"message":{"id":"m-3","conversation_id":"c-1","author_kind":"external"}}`},
		{TopicMessageDeleted, `{"conversation_id":"c-1","message_id":"m-4"}`},
		{TopicMessageCreated, `{"message":{"id":"m-5","conversation_id":"c-2","author_kind":"user"}}`},
	}
	for _, e := range events {
		ev := event(e.topic, e.payload)
		switch {
		case strings.HasPrefix(e.topic, "ticket."):
			require.NoError(t, f.svc.HandleTicketEvent(t.Context(), ev))
		case e.topic == TopicMessageDeleted:
			require.NoError(t, f.svc.HandleMessageDeletedEvent(t.Context(), ev))
		default:
			require.NoError(t, f.svc.HandleMessageEvent(t.Context(), ev))
		}
	}
	assert.Contains(t, f.repo.ticketPushes, "t-1")
	assert.True(t, f.repo.ticketPushes["t-2"].Create, "the trigger label, in any case, asks for an issue")
	assert.ElementsMatch(t, []string{"m-1", "m-4"}, keys(f.repo.commentPushes), "a person's message, or a delete the push checks, on a linked ticket")

	for _, h := range []func() error{
		func() error { return f.svc.HandleTicketEvent(t.Context(), event(TopicTicketUpdated, "{")) },
		func() error { return f.svc.HandleMessageEvent(t.Context(), event(TopicMessageCreated, "{")) },
		func() error { return f.svc.HandleMessageDeletedEvent(t.Context(), event(TopicMessageDeleted, "{")) },
	} {
		assert.ErrorIs(t, h(), apperrs.ErrFatal)
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func event(topic, payload string) eventbus.Event {
	return eventbus.Event{Topic: topic, Payload: []byte(payload)}
}
