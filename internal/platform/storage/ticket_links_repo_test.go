package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

func TestTicketLinks_Integration_BlockedClearsOnDoneStageAndCascades(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	project := newTestProject("p-links", "Books", 0)
	project.Prefix = "BKS"
	require.NoError(t, s.Projects.Create(ctx, project))
	statuses, err := s.Statuses.ListByProject(ctx, "p-links")
	require.NoError(t, err)
	stage := map[workspace.StatusKind]string{}
	for _, st := range statuses {
		stage[st.Kind] = st.ID
	}
	// Renaming the done column proves the rule reads the stage, not the name.
	done, err := s.Statuses.Get(ctx, stage[workspace.StatusKindDone])
	require.NoError(t, err)
	done.Name = "Shipped"
	require.NoError(t, s.Statuses.Update(ctx, done))

	svc := tickets.NewService(s.Tickets, s.Statuses, nil)
	frontend, err := svc.Create(ctx, "p-links", "frontend /books", "", "", "")
	require.NoError(t, err)
	backend, err := svc.Create(ctx, "p-links", "backend /books", "", "", "")
	require.NoError(t, err)
	bug, err := svc.Create(ctx, "p-links", "books page 500s", "", "", "")
	require.NoError(t, err)
	_, err = svc.UpdateStatus(ctx, backend.ID, tickets.Status(stage[workspace.StatusKindProgress]))
	require.NoError(t, err)

	set, err := svc.AddBlocker(ctx, frontend.ID, backend.ID)
	require.NoError(t, err)
	require.Len(t, set.BlockedBy, 1)
	assert.Equal(t, "BKS", set.BlockedBy[0].Prefix)
	assert.Equal(t, backend.Number, set.BlockedBy[0].Number)
	assert.True(t, set.Blocked)

	_, err = svc.AddBlocker(ctx, backend.ID, frontend.ID)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	err = s.Tickets.PutLink(ctx, tickets.TicketLink{TicketID: frontend.ID, Kind: tickets.LinkBlockedBy, TargetID: backend.ID})
	require.ErrorIs(t, err, apperrs.ErrConflict, "the unique index refuses a duplicate blocker")

	uncleared, err := svc.UnclearedBlockers(ctx)
	require.NoError(t, err)
	require.Len(t, uncleared[frontend.ID], 1)
	assert.Equal(t, "backend /books", uncleared[frontend.ID][0].Title)

	_, err = svc.UpdateStatus(ctx, backend.ID, tickets.Status(stage[workspace.StatusKindDone]))
	require.NoError(t, err)
	set, err = svc.Links(ctx, frontend.ID)
	require.NoError(t, err)
	assert.False(t, set.Blocked)
	assert.True(t, set.BlockedBy[0].Done)
	uncleared, err = svc.UnclearedBlockers(ctx)
	require.NoError(t, err)
	assert.Empty(t, uncleared)

	_, err = svc.SetFoundIn(ctx, bug.ID, "", true)
	require.NoError(t, err)
	set, err = svc.SetFoundIn(ctx, bug.ID, backend.ID, false)
	require.NoError(t, err)
	require.NotNil(t, set.FoundIn)
	assert.False(t, set.OriginUnknown)
	origin, err := svc.Links(ctx, backend.ID)
	require.NoError(t, err)
	assert.Len(t, origin.BugsFound, 1)
	assert.Len(t, origin.Blocks, 1)

	removed, err := s.Tickets.DeleteLink(ctx, tickets.TicketLink{TicketID: frontend.ID, Kind: tickets.LinkBlockedBy, TargetID: "missing"})
	require.NoError(t, err)
	assert.False(t, removed)

	require.NoError(t, svc.Delete(ctx, backend.ID))
	set, err = svc.Links(ctx, frontend.ID)
	require.NoError(t, err)
	assert.Empty(t, set.BlockedBy, "deleting a ticket removes the links to it")
	set, err = svc.Links(ctx, bug.ID)
	require.NoError(t, err)
	assert.Nil(t, set.FoundIn)
}

func TestTicketLinks_Integration_BugFiledWithFoundIn(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Projects.Create(ctx, newTestProject("p-bugs", "Bugs", 0)))
	types, err := s.TicketTypes.ListByProject(ctx, "p-bugs")
	require.NoError(t, err)
	bugType := ""
	for _, tt := range types {
		if tickets.IsBugType(tt.Name) {
			bugType = tt.ID
		}
	}
	require.NotEmpty(t, bugType, "every project is seeded with a bug type")

	svc := tickets.NewService(s.Tickets, s.Statuses, nil)
	svc.SetTicketTypes(s.TicketTypes)
	origin, err := svc.Create(ctx, "p-bugs", "login page", "", "", "")
	require.NoError(t, err)

	_, err = svc.Create(ctx, "p-bugs", "login 500s", "", "", "", tickets.CreateOptions{TypeID: bugType})
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	bug, err := svc.Create(ctx, "p-bugs", "login 500s", "", "", "", tickets.CreateOptions{TypeID: bugType, OriginID: origin.ID})
	require.NoError(t, err)
	set, err := svc.Links(ctx, bug.ID)
	require.NoError(t, err)
	require.NotNil(t, set.FoundIn)
	assert.Equal(t, origin.ID, set.FoundIn.ID)

	unknown, err := svc.Create(ctx, "p-bugs", "random crash", "", "", "", tickets.CreateOptions{TypeID: bugType, OriginUnknown: true})
	require.NoError(t, err)
	set, err = svc.Links(ctx, unknown.ID)
	require.NoError(t, err)
	assert.True(t, set.OriginUnknown)

	_, err = svc.Create(ctx, "p-bugs", "dangling", "", "", "", tickets.CreateOptions{TypeID: bugType, OriginID: "ghost"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	all, err := svc.ListByProject(ctx, "p-bugs")
	require.NoError(t, err)
	assert.Len(t, all, 3, "a refused bug leaves no ticket behind")
}
