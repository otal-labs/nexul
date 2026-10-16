package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestCrashTicketsRepo_SetAndGet_ReplaceTheSubjectsTicket(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.Tickets.Create(ctx, newTestTicket("t-1", "")))
	require.NoError(t, s.Tickets.Create(ctx, newTestTicket("t-2", "")))

	_, err := s.CrashTickets.Get(ctx, "stack-1", "web")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	require.NoError(t, s.CrashTickets.Set(ctx, "stack-1", "web", "t-1"))
	got, err := s.CrashTickets.Get(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.Equal(t, "t-1", got)

	require.NoError(t, s.CrashTickets.Set(ctx, "stack-1", "web", "t-2"))
	got, err = s.CrashTickets.Get(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.Equal(t, "t-2", got, "a service has one linked ticket")
	_, err = s.CrashTickets.Get(ctx, "stack-1", "worker")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestCrashTicketsRepo_Set_ForAnUnknownTicketFails(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedStackForContainers(t, s, "stack-1")
	require.Error(t, s.CrashTickets.Set(context.Background(), "stack-1", "web", "nope"))
}

func TestCrashTicketsRepo_ALinkGoesWithItsTicketOrStack(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	seedStackForContainers(t, s, "stack-2")
	require.NoError(t, s.Tickets.Create(ctx, newTestTicket("t-1", "")))
	require.NoError(t, s.CrashTickets.Set(ctx, "stack-1", "web", "t-1"))
	require.NoError(t, s.CrashTickets.Set(ctx, "stack-2", "web", "t-1"))

	require.NoError(t, s.Stacks.Delete(ctx, "stack-1"))
	_, err := s.CrashTickets.Get(ctx, "stack-1", "web")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a deleted stack takes its links")

	_, err = s.db.ExecContext(ctx, `DELETE FROM tickets WHERE id = ?`, "t-1")
	require.NoError(t, err)
	_, err = s.CrashTickets.Get(ctx, "stack-2", "web")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a deleted ticket takes its links, so the next crash files a new one")
}

func TestStacksRepo_CrashTickets_RoundTripAndOffByDefault(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	stack := newTestStack("svc-1")
	require.NoError(t, s.Stacks.Create(ctx, stack))
	got, err := s.Stacks.GetByID(ctx, "svc-1")
	require.NoError(t, err)
	assert.False(t, got.CrashTickets)

	got.CrashTickets = true
	require.NoError(t, s.Stacks.Update(ctx, got))
	again, err := s.Stacks.GetByID(ctx, "svc-1")
	require.NoError(t, err)
	assert.True(t, again.CrashTickets)
}

func TestUsersRepo_FirstWorkspaceOwner(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, err := s.Users.FirstWorkspaceOwner(ctx, "workspace-default")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	for _, id := range []string{"alice", "bob"} {
		_, _, err := s.Users.UpsertUser(ctx, newTestUser(id, id, id))
		require.NoError(t, err)
	}
	makeOwner(t, s, "bob", "ws-acme")
	got, err := s.Users.FirstWorkspaceOwner(ctx, "workspace-default")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "an Owner elsewhere is no Owner here")
	assert.Empty(t, got)

	makeOwner(t, s, "alice", "workspace-default")
	makeOwner(t, s, "bob", "workspace-default")
	got, err = s.Users.FirstWorkspaceOwner(ctx, "workspace-default")
	require.NoError(t, err)
	assert.Equal(t, "alice", got, "ties on the grant time go to the lower id")

	require.NoError(t, s.Users.SetAccountStatus(ctx, "alice", "disabled"))
	got, err = s.Users.FirstWorkspaceOwner(ctx, "workspace-default")
	require.NoError(t, err)
	assert.Equal(t, "bob", got, "a disabled Owner is skipped")
}
