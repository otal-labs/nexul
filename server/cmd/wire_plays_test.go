package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

// TestIntegration_PlaysLinkReader_OneHop reads a bug's origin, its PRs, and its blockers, skipping a doc the starter cannot open.
func TestIntegration_PlaysLinkReader_OneHop(t *testing.T) {
	ctx := context.Background()
	s := storage.New(mentionsTestDB(t), []byte("0123456789abcdef0123456789abcdef"))
	seedMentionsUser(t, s, "u-alice", false)
	aliceCtx := identity.WithActor(ctx, identity.Actor{ID: "u-alice"})
	body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"sessions last a day"}]}]}`
	require.NoError(t, s.Docs.Create(ctx, &docs.Doc{ID: "d-open", Title: "Auth spec", Body: body, Version: 1}))
	require.NoError(t, s.Docs.Create(ctx, &docs.Doc{ID: "d-locked", Title: "Vault", Body: body, Version: 1}))
	require.NoError(t, s.Access.Set(ctx, "doc", "d-open", "u-alice", permissions.SetOf(permissions.DocsRead), nil))

	ticketsSvc := tickets.NewService(s.Tickets, s.Statuses, nil)
	grand, err := ticketsSvc.Create(ctx, "project-general", "older work", "", "", "")
	require.NoError(t, err)
	origin, err := ticketsSvc.Create(ctx, "project-general", "Login page", "logs in", "d-open", "")
	require.NoError(t, err)
	_, err = ticketsSvc.SetFoundIn(ctx, origin.ID, grand.ID, false)
	require.NoError(t, err)
	require.NoError(t, s.Tickets.LinkPR(ctx, origin.ID, tickets.PRRef{Owner: "otal", Repo: "nexul", Number: 12, Title: "Add login"}, tickets.PRStateMerged))
	bug, err := ticketsSvc.Create(ctx, "project-general", "Login 500s", "", "", "", tickets.CreateOptions{OriginID: origin.ID})
	require.NoError(t, err)
	_, err = ticketsSvc.AddBlocker(ctx, bug.ID, grand.ID)
	require.NoError(t, err)
	locked, err := ticketsSvc.Create(ctx, "project-general", "Vault page", "", "d-locked", "")
	require.NoError(t, err)
	lockedBug, err := ticketsSvc.Create(ctx, "project-general", "Vault 500s", "", "", "", tickets.CreateOptions{OriginID: locked.ID})
	require.NoError(t, err)

	reader := playsLinkReader{tickets: ticketsSvc, docs: agentDocReader{svc: docs.NewService(s.Docs, access.NewService(s.Access, accessUsers{users: s.Users}))}}
	got, err := reader.TicketLinks(aliceCtx, bug.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Origin)
	assert.Equal(t, "Login page", got.Origin.Title)
	assert.Equal(t, "logs in", got.Origin.Body)
	assert.Equal(t, "Auth spec", got.Origin.DocTitle)
	assert.Contains(t, got.Origin.DocBody, "sessions last a day")
	require.Len(t, got.Origin.PRs, 1)
	assert.Equal(t, "merged", got.Origin.PRs[0].State)
	require.Len(t, got.Blockers, 1)
	assert.Equal(t, "older work", got.Blockers[0].Title)
	assert.False(t, got.Blockers[0].Done)

	got, err = reader.TicketLinks(aliceCtx, lockedBug.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Origin)
	assert.Empty(t, got.Origin.DocTitle, "a doc the starter cannot read is left out")
}
