package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
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
