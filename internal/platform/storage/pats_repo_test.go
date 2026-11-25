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

	require.NoError(t, s.PATs.TouchLastUsed(ctx, "p1"))

	got, err = s.PATs.GetByHash(ctx, "hash1")
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
	assert.True(t, got.LastUsedAt.After(got.CreatedAt))
}

func TestPATsRepo_RotateByRefresh_WorksOnceAndNeverOnARevokedOrOtherClientsGrant(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "alice"))
	require.NoError(t, err)
	expires := time.Unix(5000, 0).UTC()
	grant := newPATRecord("u1", "Globex Agent", "access-1", "g1", time.Unix(100, 0).UTC())
	grant.OAuthClientID, grant.RefreshHash, grant.ExpiresAt = "client-1", "refresh-1", &expires
	require.NoError(t, s.PATs.Create(ctx, grant))

	later := time.Unix(9000, 0).UTC()
	next := auth.PersonalAccessToken{TokenHash: "access-2", RefreshHash: "refresh-2", Prefix: "cess-2", ExpiresAt: &later}
	_, err = s.PATs.RotateByRefresh(ctx, "refresh-1", "client-2", time.Unix(0, 0), next)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "another client's refresh")

	got, err := s.PATs.RotateByRefresh(ctx, "refresh-1", "client-1", time.Unix(0, 0), next)
	require.NoError(t, err)
	assert.Equal(t, "g1", got.ID)
	assert.Equal(t, later, *got.ExpiresAt)
	_, err = s.PATs.GetByHash(ctx, "access-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "the old access token is gone")
	_, err = s.PATs.RotateByRefresh(ctx, "refresh-1", "client-1", time.Unix(0, 0), next)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a refresh hash works once")

	require.NoError(t, s.PATs.Revoke(ctx, "g1", "u1"))
	_, err = s.PATs.RotateByRefresh(ctx, "refresh-2", "client-1", time.Unix(0, 0), auth.PersonalAccessToken{TokenHash: "access-3", RefreshHash: "refresh-3", Prefix: "cess-3"})
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a revoked grant")
}

func TestPATsRepo_RotateByRefresh_KeepsThePreviousHashAndRefusesAnIdleGrant(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "alice"))
	require.NoError(t, err)
	expires := time.Unix(5000, 0).UTC()
	grant := newPATRecord("u1", "Globex Agent", "access-1", "g1", time.Unix(100, 0).UTC())
	grant.OAuthClientID, grant.RefreshHash, grant.ExpiresAt = "client-1", "refresh-1", &expires
	require.NoError(t, s.PATs.Create(ctx, grant))

	_, err = s.PATs.GetByPreviousRefresh(ctx, "refresh-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "never rotated")
	later := time.Unix(9000, 0).UTC()
	next := auth.PersonalAccessToken{TokenHash: "access-2", RefreshHash: "refresh-2", Prefix: "cess-2", ExpiresAt: &later}
	_, err = s.PATs.RotateByRefresh(ctx, "refresh-1", "client-1", expires, next)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a grant whose access token expired before the cutoff is idle")
	_, err = s.PATs.RotateByRefresh(ctx, "refresh-1", "client-1", expires.Add(-time.Second), next)
	require.NoError(t, err)

	got, err := s.PATs.GetByPreviousRefresh(ctx, "refresh-1")
	require.NoError(t, err)
	assert.Equal(t, "g1", got.ID)
	require.NoError(t, s.PATs.Revoke(ctx, "g1", "u1"))
	_, err = s.PATs.GetByPreviousRefresh(ctx, "refresh-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a revoked grant")
}
