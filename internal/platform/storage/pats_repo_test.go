package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newPATRecord(userID, name, hash, id string, createdAt time.Time) *auth.PersonalAccessToken {
	prefix := hash
	if len(hash) > 6 {
		prefix = hash[len(hash)-6:]
	}
	return &auth.PersonalAccessToken{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Prefix:    prefix,
		CreatedAt: createdAt,
		TokenHash: hash,
	}
}

func TestPATsRepo_CreateAndGetByHash(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	ctx := context.Background()

	pat := newPATRecord("u1", "ci", "deadbeef", "p1", time.Unix(100, 0).UTC())
	require.NoError(t, s.PATs.Create(ctx, pat))

	got, err := s.PATs.GetByHash(ctx, "deadbeef")
	require.NoError(t, err)
	assert.Equal(t, "p1", got.ID)
	assert.Equal(t, "ci", got.Name)
	assert.Equal(t, "u1", got.UserID)
	assert.Equal(t, time.Unix(100, 0).UTC(), got.CreatedAt)
	assert.Nil(t, got.LastUsedAt)
	assert.Nil(t, got.RevokedAt)
}

func TestPATsRepo_GetByHash_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.PATs.GetByHash(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPATsRepo_Create_DuplicateHashConflicts(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(context.Background(), newTestUser("u2", "43", "bob"))
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, s.PATs.Create(ctx, newPATRecord("u1", "ci", "samehash", "p1", time.Now())))
	err = s.PATs.Create(ctx, newPATRecord("u2", "ci", "samehash", "p2", time.Now()))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestPATsRepo_ListByUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(context.Background(), newTestUser("other", "43", "bob"))
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("empty", func(t *testing.T) {
		got, err := s.PATs.ListByUser(ctx, "u1")
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("scoped to the user and newest first", func(t *testing.T) {
		require.NoError(t, s.PATs.Create(ctx, newPATRecord("u1", "older", "h1", "p1", time.Unix(10, 0).UTC())))
		require.NoError(t, s.PATs.Create(ctx, newPATRecord("u1", "newer", "h2", "p2", time.Unix(20, 0).UTC())))
		require.NoError(t, s.PATs.Create(ctx, newPATRecord("other", "theirs", "h3", "p3", time.Unix(30, 0).UTC())))

		got, err := s.PATs.ListByUser(ctx, "u1")
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "p2", got[0].ID, "newest first")
		assert.Equal(t, "p1", got[1].ID)
	})
}

func TestPATsRepo_Revoke(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(context.Background(), newTestUser("u2", "43", "bob"))
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, s.PATs.Create(ctx, newPATRecord("u1", "ci", "hash1", "p1", time.Now())))

	t.Run("other user cannot revoke", func(t *testing.T) {
		err := s.PATs.Revoke(ctx, "p1", "u2")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("owner revokes, second revoke not found", func(t *testing.T) {
		require.NoError(t, s.PATs.Revoke(ctx, "p1", "u1"))
		err := s.PATs.Revoke(ctx, "p1", "u1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("revoked token still resolves but flagged", func(t *testing.T) {
		got, err := s.PATs.GetByHash(ctx, "hash1")
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt)
	})
}

func TestPATsRepo_TouchLastUsed(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, _, err := s.Users.UpsertUser(context.Background(), newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, s.PATs.Create(ctx, newPATRecord("u1", "ci", "hash1", "p1", time.Unix(10, 0).UTC())))

	got, err := s.PATs.GetByHash(ctx, "hash1")
	require.NoError(t, err)
	assert.Nil(t, got.LastUsedAt)

	time.Sleep(10 * time.Millisecond)
	require.NoError(t, s.PATs.TouchLastUsed(ctx, "p1"))

	got, err = s.PATs.GetByHash(ctx, "hash1")
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
	assert.True(t, got.LastUsedAt.After(got.CreatedAt))
}
