package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestOAuthClientsRepo_DeleteUngranted_KeepsGrantedAndYoungClients(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "alice"))
	require.NoError(t, err)
	cutoff := time.Unix(1_000_000, 0).UTC()
	for id, created := range map[string]time.Time{
		"old-ungranted": cutoff.Add(-time.Hour), "old-granted": cutoff.Add(-time.Hour), "old-revoked": cutoff.Add(-time.Hour), "young": cutoff.Add(time.Minute),
	} {
		require.NoError(t, s.OAuthClients.CreateOAuthClient(ctx, auth.OAuthClient{ID: id, Name: id, RedirectURIs: []string{"https://example.com/cb"}, CreatedAt: created}))
	}
	granted := newPATRecord("u1", "grant", "hash-granted", "p1", cutoff)
	granted.OAuthClientID = "old-granted"
	require.NoError(t, s.PATs.Create(ctx, granted))
	revoked := newPATRecord("u1", "grant", "hash-revoked", "p2", cutoff)
	revoked.OAuthClientID = "old-revoked"
	require.NoError(t, s.PATs.Create(ctx, revoked))
	require.NoError(t, s.PATs.Revoke(ctx, "p2", "u1"))

	require.NoError(t, s.OAuthClients.DeleteUngrantedOAuthClients(ctx, cutoff))

	_, err = s.OAuthClients.GetOAuthClient(ctx, "old-ungranted")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a day-old client that never got a grant is gone")
	for _, id := range []string{"old-granted", "old-revoked", "young"} {
		_, err = s.OAuthClients.GetOAuthClient(ctx, id)
		assert.NoError(t, err, "%s is kept", id)
	}
}
