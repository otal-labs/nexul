package issuesync

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func handle(t *testing.T, f *fixture, action string, i Issue) {
	t.Helper()
	require.NoError(t, f.svc.HandleIssue(t.Context(), "acme", "app", action, i, "alice-gh"))
}

func TestHandleIssue_ChangesNothing(t *testing.T) {
	off := onSync()
	off.Enabled = false
	tests := []struct {
		name   string
		syncs  []Sync
		sender string
		issue  Issue
	}{
		{"for a repository whose sync is off", []Sync{off}, "alice-gh", newIssue()},
		{"for a repository nobody synced", nil, "alice-gh", newIssue()},
		{"for the App's own bot", []Sync{onSync()}, "Nexul-App[bot]", newIssue()},
		{"for an issue without the label", []Sync{onSync()}, "alice-gh", newIssue(withLabels("bug"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(tt.syncs...)
			require.NoError(t, f.svc.HandleIssue(t.Context(), "acme", "app", "opened", tt.issue, tt.sender))
			assert.Empty(t, f.desk.tickets)
			assert.Empty(t, f.repo.links)
		})
	}
}

func TestHandleIssue_LabelledIssue_FilesOneTicket(t *testing.T) {
	f := newFixture(onSync())
	i := newIssue(func(i *Issue) { i.Assignees = []string{"stranger", "bob-gh"} })

	handle(t, f, "opened", i)
	handle(t, f, "opened", i)
	handle(t, f, "labeled", i)

	require.Len(t, f.desk.tickets, 1, "a redelivery and a second event for the same issue file nothing more")
	got := f.desk.only()
	assert.Equal(t, "Crash on save", got.Fields.Title)
	assert.Equal(t, []string{"bug"}, got.Fields.Labels, "the trigger label stays on GitHub")
	assert.Equal(t, "bob", got.Developer, "the first assignee a member holds")
	assert.Equal(t, "alice-gh", got.Reporter)
	assert.Equal(t, 0, f.desk.applied, "an unchanged issue writes nothing")
	link := f.repo.links[linkKey{"acme", "app", 7}]
	assert.Equal(t, Hash(FieldsFromIssue(i, DefaultLabel)), link.InboundHash)
	assert.Equal(t, f.now, link.LinkedAt)
}

func TestHandleIssue_AllMode_FilesEveryIssue(t *testing.T) {
	all := onSync()
	all.Mode = ModeAll
	f := newFixture(all)
	handle(t, f, "opened", newIssue(withLabels("nexul", "bug")))
	assert.Equal(t, []string{"bug", "nexul"}, f.desk.only().Fields.Labels, "all mode has no trigger label to leave out")
	handle(t, f, "opened", newIssue(func(i *Issue) { i.Number, i.Labels = 8, nil }))
	assert.Len(t, f.desk.tickets, 2)
}

func TestHandleIssue_FieldsFollowTheIssue(t *testing.T) {
	f := newFixture(onSync())
	handle(t, f, "opened", newIssue())

	handle(t, f, "edited", newIssue(withTitle("Crash on save as"), later(time.Minute)))
	assert.Equal(t, "Crash on save as", f.desk.only().Fields.Title)
	assert.Equal(t, 1, f.desk.applied)

	handle(t, f, "edited", newIssue(withTitle("An older title"), later(-time.Minute)))
	assert.Equal(t, "Crash on save as", f.desk.only().Fields.Title, "a delivery older than what was applied changes nothing")

	link := f.repo.links[linkKey{"acme", "app", 7}]
	echoed := newIssue(withTitle("Written from Nexul"), later(2*time.Minute))
	link.OutboundHash = Hash(FieldsFromIssue(echoed, DefaultLabel))
	f.repo.links[linkKey{"acme", "app", 7}] = link
	handle(t, f, "edited", echoed)
	assert.Equal(t, 1, f.desk.applied, "what Nexul last wrote to GitHub coming back changes nothing")
}

func TestHandleIssue_RemovingTheLabel_PausesUntilItIsBack(t *testing.T) {
	f := newFixture(onSync())
	handle(t, f, "opened", newIssue())

	handle(t, f, "unlabeled", newIssue(withLabels("bug"), withTitle("Renamed while paused"), later(time.Minute)))
	assert.True(t, f.repo.links[linkKey{"acme", "app", 7}].Paused)
	assert.Equal(t, "Crash on save", f.desk.only().Fields.Title, "a paused issue changes nothing")

	handle(t, f, "edited", newIssue(withLabels("bug"), withTitle("Still paused"), later(2*time.Minute)))
	assert.Equal(t, "Crash on save", f.desk.only().Fields.Title)

	handle(t, f, "labeled", newIssue(withTitle("Resumed"), later(3*time.Minute)))
	assert.False(t, f.repo.links[linkKey{"acme", "app", 7}].Paused)
	assert.Equal(t, "Resumed", f.desk.only().Fields.Title)
	assert.Len(t, f.desk.tickets, 1, "resuming keeps the link and its ticket")
}

func TestHandleIssue_ResumedUnchanged_OnlyClearsThePause(t *testing.T) {
	f := newFixture(onSync())
	handle(t, f, "opened", newIssue())
	handle(t, f, "unlabeled", newIssue(withLabels("bug"), later(time.Minute)))
	handle(t, f, "labeled", newIssue(later(2*time.Minute)))
	assert.False(t, f.repo.links[linkKey{"acme", "app", 7}].Paused)
	assert.Equal(t, 0, f.desk.applied)
}

func TestHandleIssue_Status(t *testing.T) {
	t.Run("closing moves the ticket to done with the reason on its thread", func(t *testing.T) {
		f := newFixture(onSync())
		handle(t, f, "opened", newIssue())
		handle(t, f, "closed", newIssue(closed("not_planned"), later(time.Minute)))
		assert.Equal(t, StageDone, f.desk.only().Stage)
		require.Len(t, f.desk.notes, 1)
		for _, n := range f.desk.notes {
			assert.Contains(t, n, "closed on GitHub as not planned")
		}
	})
	t.Run("reopening never takes a done ticket out of done", func(t *testing.T) {
		f := newFixture(onSync())
		handle(t, f, "opened", newIssue())
		handle(t, f, "closed", newIssue(closed("completed"), later(time.Minute)))
		handle(t, f, "reopened", newIssue(func(i *Issue) { i.StateReason = "reopened" }, later(2*time.Minute)))
		assert.Equal(t, StageDone, f.desk.only().Stage)
		assert.Len(t, f.desk.notes, 2)
		found := false
		for _, n := range f.desk.notes {
			found = found || strings.Contains(n, "Done tickets stay done")
		}
		assert.True(t, found, "the thread says why it stayed done: %v", f.desk.notes)
	})
	t.Run("reopening a ticket that is not done moves it back to backlog", func(t *testing.T) {
		f := newFixture(onSync())
		handle(t, f, "opened", newIssue())
		f.desk.only().Stage = "progress"
		handle(t, f, "reopened", newIssue(withTitle("Again"), later(time.Minute)))
		assert.Equal(t, StageBacklog, f.desk.only().Stage)
	})
	t.Run("an issue filed already closed lands in done", func(t *testing.T) {
		f := newFixture(onSync())
		handle(t, f, "labeled", newIssue(closed("")))
		assert.Equal(t, StageDone, f.desk.only().Stage)
	})
}

func TestHandleIssue_DeletedOrTransferred_UnlinksAndKeepsTheTicket(t *testing.T) {
	for _, action := range []string{"deleted", "transferred"} {
		t.Run(action, func(t *testing.T) {
			f := newFixture(onSync())
			handle(t, f, "opened", newIssue())
			require.NoError(t, f.repo.PutCommentLink(t.Context(), CommentLink{GitHubCommentID: 1, MessageID: "m-1", TicketID: "t-1"}))

			handle(t, f, action, newIssue(later(time.Minute)))
			assert.Empty(t, f.repo.links)
			assert.Empty(t, f.repo.comments)
			assert.Len(t, f.desk.tickets, 1)
			require.Len(t, f.desk.notes, 1)
			for _, n := range f.desk.notes {
				assert.Contains(t, n, "was "+action+", so it no longer syncs")
			}

			handle(t, f, action, newIssue(later(time.Minute)))
			assert.Len(t, f.desk.notes, 1, "an issue with no link left changes nothing")
		})
	}
}

func TestHandleIssue_DeskFailure_IsReturned(t *testing.T) {
	f := newFixture(onSync())
	f.desk.fail = errors.New("disk full")
	err := f.svc.HandleIssue(t.Context(), "acme", "app", "opened", newIssue(), "alice-gh")
	require.ErrorContains(t, err, "disk full")
	assert.Empty(t, f.repo.links, "no link without its ticket")
}

func TestHandleComment(t *testing.T) {
	comment := Comment{ID: 31, Body: "Seen it too", Author: "carol-gh"}
	setup := func(t *testing.T) *fixture {
		f := newFixture(onSync())
		handle(t, f, "opened", newIssue())
		return f
	}
	post := func(t *testing.T, f *fixture, action string, c Comment) {
		t.Helper()
		require.NoError(t, f.svc.HandleComment(t.Context(), "acme", "app", 7, action, c, c.Author))
	}
	t.Run("a new comment posts once", func(t *testing.T) {
		f := setup(t)
		post(t, f, "created", comment)
		post(t, f, "created", comment)
		assert.Equal(t, map[string]string{"m-31": "Seen it too"}, f.desk.messages)
		assert.Equal(t, CommentLink{GitHubCommentID: 31, MessageID: "m-31", TicketID: "t-1"}, f.repo.comments[31])
	})
	t.Run("an edit follows, and an edit of a comment never seen posts it", func(t *testing.T) {
		f := setup(t)
		post(t, f, "created", comment)
		post(t, f, "edited", Comment{ID: 31, Body: "Seen it twice", Author: "carol-gh"})
		post(t, f, "edited", Comment{ID: 32, Body: "Missed", Author: "carol-gh"})
		assert.Equal(t, map[string]string{"m-31": "Seen it twice", "m-32": "Missed"}, f.desk.messages)
		assert.Equal(t, 1, f.desk.edits)
	})
	t.Run("a delete follows and drops the link", func(t *testing.T) {
		f := setup(t)
		post(t, f, "created", comment)
		post(t, f, "deleted", comment)
		post(t, f, "deleted", comment)
		assert.Equal(t, []string{"m-31"}, f.desk.deleted)
		assert.Empty(t, f.repo.comments)
	})
	t.Run("nothing posts for the bot, an unlinked issue, or a paused one", func(t *testing.T) {
		f := setup(t)
		require.NoError(t, f.svc.HandleComment(t.Context(), "acme", "app", 7, "created", comment, "nexul-app[bot]"))
		require.NoError(t, f.svc.HandleComment(t.Context(), "acme", "app", 99, "created", comment, "carol-gh"))
		handle(t, f, "unlabeled", newIssue(withLabels(), later(time.Minute)))
		post(t, f, "created", comment)
		assert.Empty(t, f.desk.messages)
	})
	t.Run("nothing posts for a repository whose sync is off", func(t *testing.T) {
		f := newFixture()
		post(t, f, "created", comment)
		assert.Empty(t, f.desk.messages)
	})
}

func TestSet(t *testing.T) {
	label, all, on, bad := "triage", ModeAll, true, Mode("some")
	t.Run("refuses a repository the project does not have", func(t *testing.T) {
		f := newFixture()
		for _, repo := range []string{"other", "missing"} {
			_, err := f.svc.Set(t.Context(), "p-1", Change{Owner: "acme", Repo: repo, Enabled: &on})
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), "project_get lists them")
		}
		assert.Empty(t, f.repo.syncs)
	})
	t.Run("refuses a mode it does not know and label mode without a label", func(t *testing.T) {
		f := newFixture()
		_, err := f.svc.Set(t.Context(), "p-1", Change{Owner: "acme", Repo: "app", Mode: &bad})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		empty := " "
		_, err = f.svc.Set(t.Context(), "p-1", Change{Owner: "acme", Repo: "app", Label: &empty})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("turns sync on with the defaults, and a later patch keeps what it omits", func(t *testing.T) {
		f := newFixture()
		got, err := f.svc.Set(t.Context(), "p-1", Change{Owner: "acme", Repo: "app", Enabled: &on})
		require.NoError(t, err)
		assert.Equal(t, Sync{ProjectID: "p-1", Owner: "acme", Repo: "app", Label: DefaultLabel, Mode: ModeLabel, Enabled: true}, got)

		require.NoError(t, f.repo.SetSwept(t.Context(), "acme", "app", f.now))
		got, err = f.svc.Set(t.Context(), "p-1", Change{Owner: "acme", Repo: "app", Label: &label, Mode: &all})
		require.NoError(t, err)
		assert.True(t, got.Enabled)
		assert.Equal(t, ModeAll, got.Mode)
		assert.Nil(t, got.LastSweptAt, "what the sync matches changed, so it backfills again")

		syncs, err := f.svc.Syncs(t.Context(), "p-1")
		require.NoError(t, err)
		assert.Equal(t, []Sync{got}, syncs)
	})
}

func TestHandleEvents_ABadPayloadIsFatal(t *testing.T) {
	f := newFixture(onSync())
	for _, h := range []func(context.Context, eventbus.Event) error{f.svc.HandleIssueEvent, f.svc.HandleCommentEvent} {
		err := h(t.Context(), eventbus.Event{Payload: []byte("{")})
		assert.ErrorIs(t, err, apperrs.ErrFatal)
	}
}

func TestHandleEvents_DecodeThePublishedPayloads(t *testing.T) {
	f := newFixture(onSync())
	issue, err := json.Marshal(map[string]any{"owner": "acme", "repo": "app", "action": "opened", "issue": newIssue(), "sender": map[string]string{"login": "alice-gh", "type": "User"}})
	require.NoError(t, err)
	require.NoError(t, f.svc.HandleIssueEvent(t.Context(), eventbus.Event{Payload: issue}))
	comment, err := json.Marshal(map[string]any{"owner": "acme", "repo": "app", "issue_number": 7, "action": "created", "comment": Comment{ID: 3, Body: "Hi", Author: "carol-gh"}, "sender": map[string]string{"login": "carol-gh"}})
	require.NoError(t, err)
	require.NoError(t, f.svc.HandleCommentEvent(t.Context(), eventbus.Event{Payload: comment}))
	assert.Len(t, f.desk.tickets, 1)
	assert.Equal(t, map[string]string{"m-3": "Hi"}, f.desk.messages)
}
