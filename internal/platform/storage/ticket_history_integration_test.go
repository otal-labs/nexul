package storage_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

type ticketHistoryFixture struct {
	store  *storage.Store
	svc    *tickets.Service
	ticket *tickets.Ticket
	other  *tickets.Ticket
}

// mergedPRs reports every pull request as merged, so linking one finishes its ticket.
type mergedPRs struct{}

func (mergedPRs) PRState(context.Context, string, string, int) (tickets.PRState, error) {
	return tickets.PRStateMerged, nil
}

// newTicketHistoryFixture files two tickets by alice, the first with the body "first".
func newTicketHistoryFixture(t *testing.T) ticketHistoryFixture {
	t.Helper()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	svc := tickets.NewService(s.Tickets, s.Statuses, nil)
	svc.SetGate(allowAll{})
	svc.SetTicketTypes(s.TicketTypes)
	svc.SetPRStates(mergedPRs{})
	ticket, err := svc.Create(actor("alice"), "project-general", "Login times out", "first", "", "")
	require.NoError(t, err)
	other, err := svc.Create(actor("alice"), "project-general", "Sign-in page", "", "", "")
	require.NoError(t, err)
	return ticketHistoryFixture{store: s, svc: svc, ticket: ticket, other: other}
}

func (f ticketHistoryFixture) history(t *testing.T) *tickets.History {
	t.Helper()
	h, err := f.svc.History(actor("alice"), f.ticket.ID)
	require.NoError(t, err)
	return h
}

func (f ticketHistoryFixture) versionNumbers(t *testing.T) []int {
	t.Helper()
	out := []int{}
	for _, v := range f.history(t).Versions {
		out = append(out, v.Version)
	}
	return out
}

func (f ticketHistoryFixture) body(t *testing.T, version int) string {
	t.Helper()
	v, err := f.svc.GetVersion(actor("alice"), f.ticket.ID, version)
	require.NoError(t, err)
	return v.Body
}

func fieldOf(c tickets.Change) [3]string { return [3]string{c.Field, c.From, c.To} }

// TestTicketSave_EachRecordedField_WritesOneChangeWithItsBeforeAndAfter walks one ticket through every recorded field
// and checks each save added exactly one change, newest first, under the person who made it.
func TestTicketSave_EachRecordedField_WritesOneChangeWithItsBeforeAndAfter(t *testing.T) {
	f := newTicketHistoryFixture(t)
	ctx := actor("alice")
	doc, err := docs.NewService(f.store.Docs, allowAll{}, nil).Create(ctx, "project-general", "Spec", "body")
	require.NoError(t, err)
	require.NoError(t, f.store.Categories.Create(ctx, &workspace.Category{ID: "c-1", ProjectID: "project-general", Name: "Sprint 1", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	id, other := f.ticket.ID, f.other.ID

	steps := []struct {
		name string
		do   func() error
		want [3]string
	}{
		{"title", func() error { _, err := f.svc.UpdateTicket(ctx, id, "Login times out badly", "first"); return err }, [3]string{"title", "Login times out", "Login times out badly"}},
		{"status shows column names", func() error { _, err := f.svc.UpdateStatus(ctx, id, "in_progress"); return err }, [3]string{"status", "Open", "In progress"}},
		{"type shows type names", func() error { _, err := f.svc.SetType(ctx, id, "ticket-type-bug"); return err }, [3]string{"type", "task", "bug"}},
		{"developer", func() error { _, err := f.svc.SetPerson(ctx, id, tickets.RoleDeveloper, "sam"); return err }, [3]string{"developer", "", "sam"}},
		{"tester", func() error { _, err := f.svc.SetPerson(ctx, id, tickets.RoleTester, "lena"); return err }, [3]string{"tester", "", "lena"}},
		{"due date", func() error { _, err := f.svc.SetDueOn(ctx, id, "2026-10-30"); return err }, [3]string{"due_on", "", "2026-10-30"}},
		{"source doc", func() error { _, err := f.svc.SetSource(ctx, id, doc.ID); return err }, [3]string{"doc", "", doc.ID}},
		{"label added", func() error { _, err := f.svc.AddLabel(ctx, id, "urgent"); return err }, [3]string{"labels", "", "urgent"}},
		{"label added then removed", func() error { _, err := f.svc.AddLabel(ctx, id, "ui"); return err }, [3]string{"labels", "", "ui"}},
		{"label removed", func() error { _, err := f.svc.RemoveLabel(ctx, id, "ui"); return err }, [3]string{"labels", "ui", ""}},
		{"blocker added", func() error { _, err := f.svc.AddBlocker(ctx, id, other); return err }, [3]string{"blocked_by", "", other}},
		{"blocker removed", func() error { _, err := f.svc.RemoveBlocker(ctx, id, other); return err }, [3]string{"blocked_by", other, ""}},
		{"blocker added again", func() error { _, err := f.svc.AddBlocker(ctx, id, other); return err }, [3]string{"blocked_by", "", other}},
		{"found-in set", func() error { _, err := f.svc.SetFoundIn(ctx, id, other, false); return err }, [3]string{"found_in", "", other}},
		{"found-in removed", func() error { _, err := f.svc.RemoveFoundIn(ctx, id); return err }, [3]string{"found_in", other, ""}},
		{"found-in marked unknown", func() error { _, err := f.svc.SetFoundIn(ctx, id, "", true); return err }, [3]string{"found_in", "", "unknown"}},
		{"category", func() error { return f.store.Categories.SetTicketCategory(ctx, id, "c-1") }, [3]string{"category", "", "Sprint 1"}},
		{"finished", func() error { return f.svc.LinkPR(ctx, id, tickets.PRRef{Owner: "acme", Repo: "app", Number: 1}) }, [3]string{"finished_at", "", time.Now().UTC().Format(time.DateOnly)}},
	}
	for i, step := range steps {
		require.NoError(t, step.do(), step.name)
		changes := f.history(t).Changes
		require.Len(t, changes, i+2, "%s adds exactly one change to the created one", step.name)
		assert.Equal(t, step.want, fieldOf(changes[0]), step.name)
		assert.Equal(t, tickets.Actor{Kind: tickets.ActorKindUser, UserID: "alice"}, changes[0].Actor, step.name)
	}
	assert.Equal(t, "created", f.history(t).Changes[len(steps)].Field, "the created change is the oldest")
}

// TestTicketSave_ChangingNothing_WritesNoChangeAndNoVersion saves every field with the value it already holds, and
// moves a card within its column, which is position and never recorded.
func TestTicketSave_ChangingNothing_WritesNoChangeAndNoVersion(t *testing.T) {
	f := newTicketHistoryFixture(t)
	ctx := actor("alice")
	id, other := f.ticket.ID, f.other.ID
	for _, set := range []func() error{
		func() error { _, err := f.svc.SetPerson(ctx, id, tickets.RoleDeveloper, "sam"); return err },
		func() error { _, err := f.svc.SetDueOn(ctx, id, "2026-10-30"); return err },
		func() error { _, err := f.svc.AddLabel(ctx, id, "urgent"); return err },
		func() error { _, err := f.svc.AddBlocker(ctx, id, other); return err },
		func() error { _, err := f.svc.SetFoundIn(ctx, id, "", true); return err },
		func() error {
			return f.store.Categories.Create(ctx, &workspace.Category{ID: "c-1", ProjectID: "project-general", Name: "Sprint 1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
		},
		func() error { return f.store.Categories.SetTicketCategory(ctx, id, "c-1") },
		func() error { return f.svc.LinkPR(ctx, id, tickets.PRRef{Owner: "acme", Repo: "app", Number: 1}) },
	} {
		require.NoError(t, set())
	}
	before := f.history(t)

	for name, again := range map[string]func() error{
		"the same title and body":      func() error { _, err := f.svc.UpdateTicket(ctx, id, f.ticket.Title, "first"); return err },
		"the same status":              func() error { _, err := f.svc.UpdateStatus(ctx, id, f.ticket.Status); return err },
		"the same type":                func() error { _, err := f.svc.SetType(ctx, id, f.ticket.TypeID); return err },
		"the same developer":           func() error { _, err := f.svc.SetPerson(ctx, id, tickets.RoleDeveloper, "sam"); return err },
		"the same due date":            func() error { _, err := f.svc.SetDueOn(ctx, id, "2026-10-30"); return err },
		"a label it already has":       func() error { _, err := f.svc.AddLabel(ctx, id, "urgent"); return err },
		"a label it never had":         func() error { _, err := f.svc.RemoveLabel(ctx, id, "ghost"); return err },
		"a blocker it already has":     func() error { _, err := f.svc.AddBlocker(ctx, id, other); return err },
		"a blocker it never had":       func() error { _, err := f.svc.RemoveBlocker(ctx, id, "ghost"); return err },
		"the same found-in":            func() error { _, err := f.svc.SetFoundIn(ctx, id, "", true); return err },
		"the same category":            func() error { return f.store.Categories.SetTicketCategory(ctx, id, "c-1") },
		"a second merged pull request": func() error { return f.svc.LinkPR(ctx, id, tickets.PRRef{Owner: "acme", Repo: "app", Number: 2}) },
		"a move within the column":     func() error { _, err := f.svc.SetPosition(ctx, id, 7); return err },
	} {
		require.NoError(t, again(), name)
	}

	after := f.history(t)
	assert.Equal(t, before.Changes, after.Changes)
	assert.Equal(t, before.Versions, after.Versions)
}

// TestTicketCreate_WritesTheCreatedChangeAndVersionOneByWhoFiledIt checks a person's, an automation's and an agent's
// create each leave the one created change and the first description version under their own actor and via.
func TestTicketCreate_WritesTheCreatedChangeAndVersionOneByWhoFiledIt(t *testing.T) {
	f := newTicketHistoryFixture(t)
	automation := identity.WithActor(context.Background(), identity.Actor{ID: "alice", Automation: &identity.AutomationRef{ID: "auto-1", Name: "Nightly sync"}})
	cases := []struct {
		name      string
		ctx       context.Context
		opts      tickets.CreateOptions
		wantActor tickets.Actor
		wantVia   string
		wantEdit  []string
	}{
		{"a person", actor("alice"), tickets.CreateOptions{}, tickets.Actor{Kind: tickets.ActorKindUser, UserID: "alice"}, "", []string{"alice"}},
		{"an agent over MCP", actor("alice"), tickets.CreateOptions{ViaMCP: true}, tickets.Actor{Kind: tickets.ActorKindUserMCP, UserID: "alice"}, "mcp", []string{"alice"}},
		{"an automation", automation, tickets.CreateOptions{}, tickets.Actor{Kind: tickets.ActorKindAutomation, AutomationID: "auto-1", AutomationName: "Nightly sync"}, "Nightly sync", []string{}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			created, err := f.svc.Create(tt.ctx, "project-general", "Filed", "hello", "", "", tt.opts)
			require.NoError(t, err)
			h, err := f.svc.History(actor("alice"), created.ID)
			require.NoError(t, err)
			require.Len(t, h.Changes, 1)
			assert.Equal(t, [3]string{"created", "", ""}, fieldOf(h.Changes[0]))
			assert.Equal(t, tt.wantActor, h.Changes[0].Actor)
			require.Len(t, h.Versions, 1)
			assert.Equal(t, 1, h.Versions[0].Version)
			assert.Equal(t, tt.wantVia, h.Versions[0].Via)
			assert.Equal(t, tt.wantEdit, h.Versions[0].EditorIDs)
		})
	}
}

// TestTicketSave_ByAnAutomationAnAgentAndAPlay_RecordsThatActorOnTheChange shows each kind of caller named on the
// change its save made, including the play a status move arrives from through SetStatusAs.
func TestTicketSave_ByAnAutomationAnAgentAndAPlay_RecordsThatActorOnTheChange(t *testing.T) {
	f := newTicketHistoryFixture(t)
	id := f.ticket.ID
	automation := identity.WithActor(context.Background(), identity.Actor{ID: "alice", Automation: &identity.AutomationRef{ID: "auto-1", Name: "Nightly sync"}})
	play := tickets.Actor{Kind: tickets.ActorKindPlay, PlayLabel: "Implement with AI", TrailID: "trail-1", UserID: "alice"}

	_, err := f.svc.SetDueOn(automation, id, "2026-10-30")
	require.NoError(t, err)
	assert.Equal(t, tickets.Actor{Kind: tickets.ActorKindAutomation, AutomationID: "auto-1", AutomationName: "Nightly sync"}, f.history(t).Changes[0].Actor)

	_, err = f.svc.SetDueOn(tickets.WithViaMCP(actor("alice")), id, "2026-11-02")
	require.NoError(t, err)
	assert.Equal(t, tickets.Actor{Kind: tickets.ActorKindUserMCP, UserID: "alice"}, f.history(t).Changes[0].Actor)

	_, err = f.svc.SetStatusAs(actor("alice"), id, "in_progress", play, "")
	require.NoError(t, err)
	moved := f.history(t).Changes[0]
	assert.Equal(t, [3]string{"status", "Open", "In progress"}, fieldOf(moved))
	assert.Equal(t, play, moved.Actor)
}

// TestTicketUpdate_ChangedBody_WritesTheNextVersionAndSkipsAnEqualBody checks the description version a save writes:
// who it credits and how it came, and that a title-only or unchanged save writes none.
func TestTicketUpdate_ChangedBody_WritesTheNextVersionAndSkipsAnEqualBody(t *testing.T) {
	f := newTicketHistoryFixture(t)
	id := f.ticket.ID
	automation := identity.WithActor(context.Background(), identity.Actor{ID: "alice", Automation: &identity.AutomationRef{ID: "auto-1", Name: "Nightly sync"}})
	require.Equal(t, []int{1}, f.versionNumbers(t), "filing the ticket writes version 1")

	_, err := f.svc.UpdateTicket(actor("alice"), id, f.ticket.Title, "second")
	require.NoError(t, err)
	assert.Equal(t, []int{2, 1}, f.versionNumbers(t))
	hist := f.history(t)
	assert.Equal(t, []string{"alice"}, hist.Versions[0].EditorIDs)
	assert.Empty(t, hist.Versions[0].Via)
	assert.True(t, hist.Versions[0].Current)
	assert.False(t, hist.Versions[1].Current)

	_, err = f.svc.UpdateTicket(actor("alice"), id, f.ticket.Title, "second")
	require.NoError(t, err)
	_, err = f.svc.UpdateTicket(actor("alice"), id, "Renamed", "second")
	require.NoError(t, err)
	assert.Equal(t, []int{2, 1}, f.versionNumbers(t), "an equal body and a title-only save add no version")

	_, err = f.svc.UpdateTicket(automation, id, "Renamed", "by automation")
	require.NoError(t, err)
	_, err = f.svc.UpdateTicket(tickets.WithViaMCP(actor("alice")), id, "Renamed", "by agent")
	require.NoError(t, err)
	hist = f.history(t)
	require.Len(t, hist.Versions, 4)
	for i, want := range []struct {
		via     string
		editors []string
	}{{"mcp", []string{"alice"}}, {"Nightly sync", []string{}}, {"", []string{"alice"}}, {"", []string{"alice"}}} {
		assert.Equal(t, want.via, hist.Versions[i].Via, "version %d", hist.Versions[i].Version)
		assert.Equal(t, want.editors, hist.Versions[i].EditorIDs, "version %d", hist.Versions[i].Version)
	}
}

// TestTicketCommitSitting_WritesOneVersionCreditingEveryEditorAndSkipsAnUnchangedBody mirrors a live room: a flush
// writes the body and a title change but no version, and the sitting that ends writes the one version.
func TestTicketCommitSitting_WritesOneVersionCreditingEveryEditorAndSkipsAnUnchangedBody(t *testing.T) {
	f := newTicketHistoryFixture(t)
	ctx, id := actor("alice"), f.ticket.ID
	require.NoError(t, f.svc.CommitCollab(ctx, id, "Renamed live", "typed in the editor"))
	assert.Equal(t, []int{1}, f.versionNumbers(t), "an editor flush alone writes no version")
	assert.Equal(t, [3]string{"title", "Login times out", "Renamed live"}, fieldOf(f.history(t).Changes[0]), "a flush that renames the ticket is a title change")

	require.NoError(t, f.svc.CommitSitting(ctx, id, []string{"bob", "alice"}))
	assert.Equal(t, []int{2, 1}, f.versionNumbers(t))
	hist := f.history(t)
	assert.Equal(t, []string{"bob", "alice"}, hist.Versions[0].EditorIDs)
	assert.Empty(t, hist.Versions[0].Via)
	assert.Equal(t, "typed in the editor", f.body(t, 2))

	require.NoError(t, f.svc.CommitSitting(ctx, id, []string{"alice"}))
	assert.Equal(t, []int{2, 1}, f.versionNumbers(t), "a sitting that left the text as the newest version has it adds none")

	require.NoError(t, f.svc.CommitCollab(ctx, id, "", "typed again"))
	_, err := f.svc.UpdateTicket(ctx, id, "Renamed live", "typed in the editor")
	require.NoError(t, err)
	assert.Equal(t, []int{2, 1}, f.versionNumbers(t), "a save of the text the newest version holds adds none, though the live text had moved on")
}

// TestTicketUpdate_PastTenDescriptionVersions_PrunesTheOldestAndKeepsNumbering fills the history past its cap.
func TestTicketUpdate_PastTenDescriptionVersions_PrunesTheOldestAndKeepsNumbering(t *testing.T) {
	f := newTicketHistoryFixture(t)
	for i := range 10 {
		_, err := f.svc.UpdateTicket(actor("alice"), f.ticket.ID, f.ticket.Title, fmt.Sprintf("edit %d", i))
		require.NoError(t, err)
	}
	assert.Equal(t, []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}, f.versionNumbers(t), "eleven versions were written; the oldest is gone")
	_, err := f.svc.GetVersion(actor("alice"), f.ticket.ID, 1)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, "edit 9", f.body(t, 11))
}

// TestTicketSave_PastAHundredChanges_PrunesTheOldestAndKeepsTheNewestFirst fills the change list past its cap.
func TestTicketSave_PastAHundredChanges_PrunesTheOldestAndKeepsTheNewestFirst(t *testing.T) {
	f := newTicketHistoryFixture(t)
	newest := ""
	for i := range 101 {
		newest = fmt.Sprintf("2026-10-%02d", i%28+1)
		_, err := f.svc.SetDueOn(actor("alice"), f.ticket.ID, newest)
		require.NoError(t, err)
	}
	changes := f.history(t).Changes
	require.Len(t, changes, 100, "102 changes were written (created and 101 due dates); the oldest two are gone")
	assert.Equal(t, newest, changes[0].To, "the newest is first")
	assert.NotEqual(t, "created", changes[len(changes)-1].Field)
}

// TestTicketRestoreVersion_SavesTheOldBodyAsANewVersionOnTop restores an early version and leaves every other one.
func TestTicketRestoreVersion_SavesTheOldBodyAsANewVersionOnTop(t *testing.T) {
	f := newTicketHistoryFixture(t)
	ctx, id := actor("alice"), f.ticket.ID
	for _, body := range []string{"second", "third"} {
		_, err := f.svc.UpdateTicket(ctx, id, f.ticket.Title, body)
		require.NoError(t, err)
	}

	restored, err := f.svc.RestoreVersion(tickets.WithViaMCP(ctx), id, 1)
	require.NoError(t, err)
	assert.Equal(t, "first", restored.Body)
	got, err := f.svc.Get(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "first", got.Body)
	assert.Equal(t, []int{4, 3, 2, 1}, f.versionNumbers(t), "the restore is a new version on top; nothing is lost")
	assert.Equal(t, "first", f.body(t, 4))
	assert.Equal(t, "third", f.body(t, 3))
	assert.Equal(t, "mcp", f.history(t).Versions[0].Via, "a restore is recorded as the save it is")

	_, err = f.svc.RestoreVersion(ctx, id, 4)
	require.NoError(t, err)
	assert.Equal(t, []int{4, 3, 2, 1}, f.versionNumbers(t), "restoring what the newest version holds adds none")
	_, err = f.svc.RestoreVersion(ctx, id, 99)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
