package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// allowTickets lets every ticket and chat call through, for tests about what the adapters do past the permission check.
type allowTickets struct{}

func (allowTickets) RequireProject(context.Context, string, permissions.Action) error { return nil }

func (allowTickets) Require(context.Context, string, permissions.Action) error { return nil }

// TestIntegration_PlaysLinkReader_OneHop names a bug's origin by its key, never the origin's own origin, and its blockers.
func TestIntegration_PlaysLinkReader_OneHop(t *testing.T) {
	ctx := context.Background()
	db := mentionsTestDB(t)
	_, err := db.Exec(`UPDATE projects SET prefix = 'GEN' WHERE id = 'project-general'`)
	require.NoError(t, err)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	ticketsSvc := tickets.NewService(s.Tickets, s.Statuses, nil)
	ticketsSvc.SetGate(allowTickets{})
	grand, err := ticketsSvc.Create(ctx, "project-general", "older work", "", "", "")
	require.NoError(t, err)
	origin, err := ticketsSvc.Create(ctx, "project-general", "Login page", "logs in", "", "")
	require.NoError(t, err)
	_, err = ticketsSvc.SetFoundIn(ctx, origin.ID, grand.ID, false)
	require.NoError(t, err)
	bug, err := ticketsSvc.Create(ctx, "project-general", "Login 500s", "", "", "", tickets.CreateOptions{OriginID: origin.ID})
	require.NoError(t, err)
	_, err = ticketsSvc.AddBlocker(ctx, bug.ID, grand.ID)
	require.NoError(t, err)

	got, err := playsLinkReader{tickets: ticketsSvc}.TicketLinks(ctx, bug.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Origin)
	assert.Equal(t, "Login page", got.Origin.Title)
	assert.Equal(t, "GEN-2", got.Origin.Key)
	require.Len(t, got.Blockers, 1)
	assert.Equal(t, "older work", got.Blockers[0].Title)
	assert.False(t, got.Blockers[0].Done)
}

func TestPlaysInterviewAnswers_RoundsLandInOrderAsStoredAnswers(t *testing.T) {
	svc, store := newWired(t)
	ctx := t.Context()
	require.NoError(t, store.Projects.Create(ctx, &workspace.Project{ID: "p-web", WorkspaceID: "workspace-default", Name: "Web", Prefix: "WEB"}))
	a := playsInterviewAnswers{svc: svc.memoriesSvc}

	require.NoError(t, a.RecordRound(ctx, "p-web", "u-1", []plays.FollowUp{
		{Question: "Which test runner?", Why: "ci.yml runs both", Options: []harness.QuestionOption{{Label: "Vitest", Description: "web/", Value: "vitest"}, {Label: "Bun"}}, Selected: []string{"Vitest"}},
		{Question: "What coverage floor?", Selected: []string{}},
	}))
	require.NoError(t, a.RecordRound(ctx, "p-web", "u-1", []plays.FollowUp{{Question: "Keep the 80 floor?", Text: "Raise it to 85"}}))

	got, err := store.Memories.ListAnswers(ctx, "p-web")
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []int{1, 1, 2}, []int{got[0].Round, got[1].Round, got[2].Round})
	assert.Equal(t, []memories.AnswerOption{{Label: "Vitest", Description: "web/"}, {Label: "Bun"}}, got[0].Options)
	assert.Equal(t, "ci.yml runs both", got[0].Why)
	assert.Equal(t, []string{"Vitest"}, got[0].Selected)
	assert.True(t, got[1].Skipped, "an empty answer is stored as skipped")
	assert.Equal(t, "Raise it to 85", got[2].Text)
	assert.Equal(t, "u-1", got[2].AnsweredBy)
}

// TestProjectTargets_ListsWhatTheCallerReads: the board's batch reads by project list the project's tickets or docs
// through their own use-cases, so a caller who cannot read them is refused rather than handed the ids.
func TestProjectTargets_ListsWhatTheCallerReads(t *testing.T) {
	f := newPermFixture(t)
	p := projectTargets{tickets: f.svc.ticketsSvc, docs: f.svc.docsSvc}

	ticketIDs, err := p.TargetIDs(as(uReader), plays.TargetTicket, "project-general")
	require.NoError(t, err)
	assert.Contains(t, ticketIDs, f.ticket.ID)
	docIDs, err := p.TargetIDs(as(uReader), plays.TargetDoc, "project-general")
	require.NoError(t, err)
	assert.Contains(t, docIDs, f.doc)

	_, err = p.TicketIDs(as(uPlain), "project-general")
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = p.TargetIDs(as(uReader), plays.TargetInterview, "project-general")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
