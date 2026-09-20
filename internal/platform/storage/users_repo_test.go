package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
