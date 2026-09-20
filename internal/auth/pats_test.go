package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// newPATHarness wires a service with a real in-memory PAT store.
func newPATHarness() (*Service, *fakeUserStore, *fakePATStore) {
	users := newFakeUserStore()
	s := newTestService(&fakeGitHub{user: ghUser("1", "owner")}, users, newFakeAllowlist(), newFakeSettings())
	pats := newFakePATStore()
	s.cfg.PATs = pats
	return s, users, pats
}

func seedPATUser(t *testing.T, users *fakeUserStore) string {
	t.Helper()
	_, _, err := users.UpsertUser(context.Background(), &User{ID: "u1", Provider: ProviderGitHub, ProviderUserID: "1", Login: "owner"})
	require.NoError(t, err)
	return "u1"
}

func TestMintPAT_ErrorsFirst(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)

	t.Run("empty name is invalid", func(t *testing.T) {
		_, _, err := s.MintPAT(context.Background(), u, "   ")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("unknown user is not found", func(t *testing.T) {
		_, _, err := s.MintPAT(context.Background(), "ghost", "ci")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("no pats yet", func(t *testing.T) {
		toks, err := s.ListPATs(context.Background(), u)
		require.NoError(t, err)
		assert.Empty(t, toks)
	})
}

func TestMintPAT_ReturnsRawOnce_StoresHash(t *testing.T) {
	s, users, pats := newPATHarness()
	u := seedPATUser(t, users)

	raw, pat, err := s.MintPAT(context.Background(), u, "ci agent")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(raw, patPrefix))
	assert.Equal(t, "ci agent", pat.Name)
	assert.Equal(t, u, pat.UserID)
	assert.Equal(t, raw[len(raw)-6:], pat.Prefix)

	t.Run("raw value is never stored", func(t *testing.T) {
		pats.mu.Lock()
		defer pats.mu.Unlock()
		for _, stored := range pats.byID {
			assert.NotEqual(t, raw, stored.TokenHash, "raw token must not be persisted")
			assert.Equal(t, hashPAT(raw), stored.TokenHash)
		}
	})

	t.Run("two tokens differ", func(t *testing.T) {
		raw2, _, err := s.MintPAT(context.Background(), u, "ci agent")
		require.NoError(t, err)
		assert.NotEqual(t, raw, raw2)
	})
}

func TestAuthenticatePAT(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)
	raw, _, err := s.MintPAT(context.Background(), u, "ci")
	require.NoError(t, err)

	t.Run("resolves the token's user", func(t *testing.T) {
		user, err := s.AuthenticatePAT(context.Background(), raw)
		require.NoError(t, err)
		assert.Equal(t, u, user.ID)
		assert.Equal(t, "owner", user.Login)
	})

	t.Run("unknown token is unauthorized", func(t *testing.T) {
		_, err := s.AuthenticatePAT(context.Background(), patPrefix+"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})

	t.Run("wrong shape is unauthorized", func(t *testing.T) {
		_, err := s.AuthenticatePAT(context.Background(), "not-a-pat")
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
}

func TestRevokePAT_TakesEffectImmediately(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)
	raw, pat, err := s.MintPAT(context.Background(), u, "ci")
	require.NoError(t, err)

	_, err = s.AuthenticatePAT(context.Background(), raw)
	require.NoError(t, err)

	require.NoError(t, s.RevokePAT(context.Background(), u, pat.ID))
	_, err = s.AuthenticatePAT(context.Background(), raw)
	require.ErrorIs(t, err, apperrs.ErrUnauthorized, "revoked token must stop authenticating")

	t.Run("second revoke not found", func(t *testing.T) {
		require.ErrorIs(t, s.RevokePAT(context.Background(), u, pat.ID), apperrs.ErrNotFound)
	})

	t.Run("revoking another user's token not found", func(t *testing.T) {
		_, _, err := users.UpsertUser(context.Background(), &User{ID: "u2", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
		require.NoError(t, err)
		require.ErrorIs(t, s.RevokePAT(context.Background(), "u2", pat.ID), apperrs.ErrNotFound)
	})

	t.Run("list still shows the revoked token for the settings UI", func(t *testing.T) {
		toks, err := s.ListPATs(context.Background(), u)
		require.NoError(t, err)
		require.Len(t, toks, 1)
		assert.NotNil(t, toks[0].RevokedAt)
	})
}

func TestMintPAT_LastUsedStamp(t *testing.T) {
	s, users, _ := newPATHarness()
	u := seedPATUser(t, users)
	raw, _, err := s.MintPAT(context.Background(), u, "ci")
	require.NoError(t, err)

	_, err = s.AuthenticatePAT(context.Background(), raw)
	require.NoError(t, err)

	toks, err := s.ListPATs(context.Background(), u)
	require.NoError(t, err)
	require.Len(t, toks, 1)
	require.NotNil(t, toks[0].LastUsedAt)
	assert.True(t, toks[0].LastUsedAt.After(toks[0].CreatedAt))
}

func TestConnectionToken_CarriesMCPURL(t *testing.T) {
	s, _, _, _ := newTestHarness(&fakeGitHub{user: ghUser("1", "owner")})
	token, err := s.Login(context.Background(), "good-code")
	require.NoError(t, err)
	ownerID := mustVerify(t, s, token)
	require.NoError(t, s.CompleteOwnerWizard(context.Background(), ownerID, "https://deploy.example.com"))

	ct, err := s.GenerateConnectionToken(context.Background(), ownerID)
	require.NoError(t, err)
	assert.Equal(t, "https://deploy.example.com/mcp", ct.MCPURL)
	claims, err := s.ParseConnectionToken(ct.Token)
	require.NoError(t, err)
	assert.Equal(t, "https://deploy.example.com/mcp", claims.MCPURL)
}

func TestMCPURLFor(t *testing.T) {
	tests := []struct {
		name        string
		instanceURL string
		want        string
	}{
		{"plain", "https://deploy.example.com", "https://deploy.example.com/mcp"},
		{"path trimmed", "https://deploy.example.com/base", "https://deploy.example.com/mcp"},
		{"port kept", "https://deploy.example.com:8443", "https://deploy.example.com:8443/mcp"},
		{"http", "http://localhost", "http://localhost/mcp"},
		{"no instance url", "", ""},
		{"malformed", "not a url", ""},
		{"ipv6", "https://[::1]:443", "https://[::1]:443/mcp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mcpURLFor(tt.instanceURL))
		})
	}
}

func TestIsPAT(t *testing.T) {
	assert.True(t, isPAT(patPrefix+"xyz"))
	assert.False(t, isPAT("plain.session.token"))
	assert.False(t, isPAT(""))
}
