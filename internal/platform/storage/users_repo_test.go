package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestUsersRepo_GetUserByLogin(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	t.Run("matches case-insensitively", func(t *testing.T) {
		u, err := s.Users.GetUserByLogin(context.Background(), "Onik97")
		require.NoError(t, err)
		assert.Equal(t, "u1", u.ID)
		assert.Equal(t, "onik97", u.Login)
	})
	t.Run("not found for unknown login", func(t *testing.T) {
		_, err := s.Users.GetUserByLogin(context.Background(), "ghost")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestUsersRepo_ListUsers(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(context.Background(), newTestUser("u2", "43", "alice"))
	require.NoError(t, err)

	users, err := s.Users.ListUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.ElementsMatch(t, []string{"u1", "u2"}, []string{users[0].ID, users[1].ID})
}

func TestUsersRepo_Identities_LinkSignInAndUnlink(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(ctx, newTestUser("u2", "43", "other"))
	require.NoError(t, err)
	google := &auth.Identity{UserID: "u1", Provider: auth.ProviderGoogle, ProviderUserID: "sub-1", Login: "onik@example.com", Name: "Google Name", CreatedAt: time.Unix(1_700_000_000, 0)}

	linked := eventbus.OutboxEvent{ID: "evt-linked", Topic: auth.TopicIdentityLinked, Payload: auth.IdentityChangedEvent{UserID: "u1"}}
	require.NoError(t, s.Users.LinkIdentity(ctx, google, linked))
	assert.Equal(t, 1, outboxCount(t, s, "evt-linked"))

	t.Run("the linked account signs in as the same user without renaming them", func(t *testing.T) {
		u, created, err := s.Users.UpsertUser(ctx, &auth.Identity{Provider: auth.ProviderGoogle, ProviderUserID: "sub-1", Login: "renamed@example.com", Name: "Renamed"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Equal(t, "u1", u.ID)
		assert.Equal(t, "onik97", u.Login, "the handle follows the first identity")
		ids, err := s.Users.ListIdentities(ctx, "u1")
		require.NoError(t, err)
		require.Len(t, ids, 2)
		assert.Equal(t, auth.ProviderGitHub, ids[0].Provider)
		assert.Equal(t, "renamed@example.com", ids[1].Login, "the identity row itself syncs")
	})

	t.Run("an account already attached to anyone is refused", func(t *testing.T) {
		err := s.Users.LinkIdentity(ctx, &auth.Identity{UserID: "u2", Provider: auth.ProviderGoogle, ProviderUserID: "sub-1", Login: "x"})
		require.ErrorIs(t, err, apperrs.ErrConflict)
		assert.Contains(t, err.Error(), "another user")
		err = s.Users.LinkIdentity(ctx, &auth.Identity{UserID: "u1", Provider: auth.ProviderGoogle, ProviderUserID: "sub-1", Login: "x"})
		require.ErrorIs(t, err, apperrs.ErrConflict)
		assert.Contains(t, err.Error(), "linked to you")
		err = s.Users.LinkIdentity(ctx, &auth.Identity{UserID: "u1", Provider: auth.ProviderGoogle, ProviderUserID: "sub-2", Login: "x"})
		require.ErrorIs(t, err, apperrs.ErrConflict)
		assert.Contains(t, err.Error(), "unlink it first")
	})

	t.Run("unlink keeps the last identity and reports a missing one", func(t *testing.T) {
		err := s.Users.UnlinkIdentity(ctx, "u2", auth.ProviderGitHub)
		require.ErrorIs(t, err, apperrs.ErrConflict)
		err = s.Users.UnlinkIdentity(ctx, "u1", auth.ProviderDiscord)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		unlinked := eventbus.OutboxEvent{ID: "evt-unlinked", Topic: auth.TopicIdentityUnlinked, Payload: auth.IdentityChangedEvent{UserID: "u1"}}
		require.NoError(t, s.Users.UnlinkIdentity(ctx, "u1", auth.ProviderGitHub, unlinked))
		assert.Equal(t, 1, outboxCount(t, s, "evt-unlinked"))
		_, err = s.Users.GetUserByProvider(ctx, auth.ProviderGitHub, "42")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		u, err := s.Users.GetUserByProvider(ctx, auth.ProviderGoogle, "sub-1")
		require.NoError(t, err)
		assert.Equal(t, "u1", u.ID)
	})
}
