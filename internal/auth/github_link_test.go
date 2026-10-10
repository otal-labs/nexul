package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeGitHubLinks is an in-memory GitHubLinkStore.
type fakeGitHubLinks struct {
	mu    sync.Mutex
	links map[string]GitHubLink
}

func (f *fakeGitHubLinks) GetGitHubLink(_ context.Context, userID string) (GitHubLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.links[userID]
	if !ok {
		return GitHubLink{}, apperrs.ErrNotFound
	}
	return l, nil
}

func (f *fakeGitHubLinks) SaveGitHubLink(_ context.Context, l GitHubLink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.links[l.UserID] = l
	return nil
}

func (f *fakeGitHubLinks) DeleteGitHubLink(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.links, userID)
	return nil
}

// fakeGranter is a GitHub App's token endpoint: each exchange or refresh hands out its next grant, or refreshErr.
type fakeGranter struct {
	fakeGitHub
	grant      GitHubGrant
	refreshErr error
	refreshes  []string
}

func (f *fakeGranter) ExchangeGrant(context.Context, string) (GitHubGrant, error) {
	return f.grant, nil
}

func (f *fakeGranter) RefreshGrant(_ context.Context, refreshToken string) (GitHubGrant, error) {
	f.refreshes = append(f.refreshes, refreshToken)
	if f.refreshErr != nil {
		return GitHubGrant{}, f.refreshErr
	}
	return GitHubGrant{AccessToken: "ghu_fresh", RefreshToken: "ghr_next", ExpiresIn: 8 * time.Hour}, nil
}

func newGitHubLinkHarness(t *testing.T, gh *fakeGranter) (*Service, *fakeUserStore, *fakeGitHubLinks) {
	t.Helper()
	s, users, _, settings := newTestHarness(gh)
	_, err := settings.Set(t.Context(), "https://deploy.example.com")
	require.NoError(t, err)
	_, err = settings.SetGitHubOAuth(t.Context(), "Iv1.acme", "secret")
	require.NoError(t, err)
	links := &fakeGitHubLinks{links: map[string]GitHubLink{}}
	s.cfg.GitHubLinks = links
	s.cfg.PATs = newFakePATStore()
	return s, users, links
}

func TestLoginWith_GitHub_KeepsThePersonsOwnTokenAndItsRefresh(t *testing.T) {
	gh := &fakeGranter{fakeGitHub: fakeGitHub{user: ghUser("1", "alice")},
		grant: GitHubGrant{AccessToken: "ghu_alice", RefreshToken: "ghr_alice", ExpiresIn: 8 * time.Hour, RefreshExpiresIn: 4380 * time.Hour}}
	s, _, links := newGitHubLinkHarness(t, gh)

	token, err := s.Login(t.Context(), "good-code")
	require.NoError(t, err)
	userID := mustVerify(t, s, token)

	now := s.cfg.Now().UTC()
	assert.Equal(t, GitHubLink{UserID: userID, AccessToken: "ghu_alice", RefreshToken: "ghr_alice",
		ExpiresAt: now.Add(8 * time.Hour), RefreshExpiresAt: now.Add(4380 * time.Hour), ConnectedAt: now}, links.links[userID])
	got, err := s.GitHubToken(t.Context(), userID)
	require.NoError(t, err)
	assert.Equal(t, "ghu_alice", got)
	assert.Empty(t, gh.refreshes, "a live token is used as it is")
}

func TestGitHubToken_RefreshesALapsedTokenAndNeverFallsBack(t *testing.T) {
	lapsed := func(s *Service) GitHubLink {
		return GitHubLink{UserID: "alice", AccessToken: "ghu_old", RefreshToken: "ghr_old", ExpiresAt: s.cfg.Now().Add(-time.Hour), ConnectedAt: s.cfg.Now()}
	}
	tests := []struct {
		name       string
		refreshErr error
		want       string
		wantErr    error
		reconnect  bool
	}{
		{"a lapsed token is refreshed and the new pair kept", nil, "ghu_fresh", nil, false},
		{"GitHub refusing the refresh marks the person to reconnect", apperrs.ErrUnauthorized, "", ErrGitHubReconnect, true},
		{"a network failure is retried later, not a reconnect", apperrs.Retryable(errors.New("timeout")), "", apperrs.ErrRetryable, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gh := &fakeGranter{fakeGitHub: fakeGitHub{user: ghUser("1", "alice")}, refreshErr: tt.refreshErr}
			s, _, links := newGitHubLinkHarness(t, gh)
			links.links["alice"] = lapsed(s)

			got, err := s.GitHubToken(t.Context(), "alice")
			assert.Equal(t, []string{"ghr_old"}, gh.refreshes)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, tt.reconnect, links.links["alice"].NeedsReconnect)
				if !tt.reconnect {
					return
				}
				assert.Empty(t, links.links["alice"].AccessToken, "a refused token is not kept")
				_, err = s.GitHubToken(t.Context(), "alice")
				require.ErrorIs(t, err, ErrGitHubReconnect)
				assert.Len(t, gh.refreshes, 1, "a person marked to reconnect is not refreshed again")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, "ghr_next", links.links["alice"].RefreshToken)
		})
	}

	t.Run("a person with no link is told to connect", func(t *testing.T) {
		s, _, _ := newGitHubLinkHarness(t, &fakeGranter{})
		_, err := s.GitHubToken(t.Context(), "carol")
		require.ErrorIs(t, err, ErrGitHubNotConnected)
		_, err = s.GitHubToken(t.Context(), "")
		require.ErrorIs(t, err, ErrGitHubNotConnected, "the server's own calls have no person to read as")
	})
}

// connectGitHub runs Connect GitHub for userID as the GitHub account gh returns.
func connectGitHub(t *testing.T, s *Service, userID string) error {
	t.Helper()
	_, state, err := s.StartIdentityLink(t.Context(), userID, ProviderGitHub)
	require.NoError(t, err)
	return s.CompleteIdentityLink(t.Context(), ProviderGitHub, state, "good-code")
}

func TestConnectGitHub(t *testing.T) {
	t.Run("a person who signs in with Google links GitHub and keeps its token, on their own account", func(t *testing.T) {
		gh := &fakeGranter{fakeGitHub: fakeGitHub{user: ghUser("7", "alice-gh")}, grant: GitHubGrant{AccessToken: "ghu_alice"}}
		s, users, links := newGitHubLinkHarness(t, gh)
		_, _, err := users.UpsertUser(t.Context(), &Identity{UserID: "alice", Provider: ProviderGoogle, ProviderUserID: "sub-1", Login: "alice@example.com"})
		require.NoError(t, err)

		require.NoError(t, connectGitHub(t, s, "alice"))
		identities, err := s.ListIdentities(t.Context(), "alice")
		require.NoError(t, err)
		assert.Len(t, identities, 2)
		assert.Equal(t, "ghu_alice", links.links["alice"].AccessToken)
		st, err := s.GitHubLinkStatus(t.Context(), "alice")
		require.NoError(t, err)
		assert.Equal(t, GitHubLinkConnected, st.State)
		assert.Equal(t, "alice-gh", st.Login)

		require.NoError(t, s.UnlinkIdentity(t.Context(), "alice", ProviderGitHub))
		assert.NotContains(t, links.links, "alice", "unlinking the GitHub sign-in drops its token too")
	})

	t.Run("reconnecting the GitHub account already linked keeps one identity and a fresh token", func(t *testing.T) {
		gh := &fakeGranter{fakeGitHub: fakeGitHub{user: ghUser("1", "alice")}, grant: GitHubGrant{AccessToken: "ghu_first"}}
		s, _, links := newGitHubLinkHarness(t, gh)
		token, err := s.Login(t.Context(), "good-code")
		require.NoError(t, err)
		userID := mustVerify(t, s, token)
		links.links[userID] = GitHubLink{UserID: userID, NeedsReconnect: true}

		gh.grant = GitHubGrant{AccessToken: "ghu_again"}
		require.NoError(t, connectGitHub(t, s, userID))
		identities, err := s.ListIdentities(t.Context(), userID)
		require.NoError(t, err)
		assert.Len(t, identities, 1)
		assert.Equal(t, GitHubLink{UserID: userID, AccessToken: "ghu_again", ConnectedAt: s.cfg.Now().UTC()}, links.links[userID])
	})

	t.Run("a GitHub account linked to another person is refused and nobody's token changes", func(t *testing.T) {
		gh := &fakeGranter{fakeGitHub: fakeGitHub{user: ghUser("1", "bob")}, grant: GitHubGrant{AccessToken: "ghu_bob_again"}}
		s, users, links := newGitHubLinkHarness(t, gh)
		token, err := s.Login(t.Context(), "good-code")
		require.NoError(t, err)
		bobID := mustVerify(t, s, token)
		_, _, err = users.UpsertUser(t.Context(), &Identity{UserID: "alice", Provider: ProviderGoogle, ProviderUserID: "sub-1", Login: "alice@example.com"})
		require.NoError(t, err)
		bobs := links.links[bobID]

		err = connectGitHub(t, s, "alice")
		require.ErrorIs(t, err, apperrs.ErrConflict)
		assert.NotContains(t, links.links, "alice")
		assert.Equal(t, bobs, links.links[bobID])
	})
}

func TestHandler_DisconnectGitHub_NeedsASignedInDeviceAndIsTheWayBack(t *testing.T) {
	gh := &fakeGranter{fakeGitHub: fakeGitHub{user: ghUser("1", "alice")}, grant: GitHubGrant{AccessToken: "ghu_alice"}}
	s, _, links := newGitHubLinkHarness(t, gh)
	token, err := s.Login(t.Context(), "good-code")
	require.NoError(t, err)
	userID := mustVerify(t, s, token)
	pat, _, err := s.MintPAT(t.Context(), userID, "agent")
	require.NoError(t, err)

	send := func(method, bearer string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/auth/github-link", nil)
		req.Header.Set("Authorization", "Bearer "+bearer)
		s.RequireAuth(NewHandler(s).ProtectedRoutes()).ServeHTTP(rec, req)
		return rec
	}
	assert.JSONEq(t, `{"state":"connected","login":"alice","connected_at":"2023-11-14T22:13:20Z"}`, send(http.MethodGet, token).Body.String())
	assert.Equal(t, http.StatusForbidden, send(http.MethodDelete, pat).Code, "an agent's token never disconnects its person")
	assert.Contains(t, links.links, userID)

	assert.Equal(t, http.StatusNoContent, send(http.MethodDelete, token).Code)
	assert.NotContains(t, links.links, userID)
	assert.JSONEq(t, `{"state":"none"}`, send(http.MethodGet, token).Body.String())
}

func TestHandler_CallbackGET_AnInstallationReturnIsNeverASignIn(t *testing.T) {
	for _, state := range []string{"", "install.ws.1.sig"} {
		gh := &fakeGranter{fakeGitHub: fakeGitHub{user: ghUser("1", "alice")}}
		s, _, links := newGitHubLinkHarness(t, gh)
		rec := httptest.NewRecorder()
		NewHandler(s).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/callback?code=c&installation_id=42&setup_action=install&state="+state, nil))
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "https://deploy.example.com/settings/connectors", rec.Header().Get("Location"))
		assert.Empty(t, rec.Result().Cookies(), "nobody is signed in")
		assert.Empty(t, links.links, "the installer's code is never exchanged")
	}
}
