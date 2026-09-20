package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type ghServer struct {
	token     string
	userID    string
	userLogin string
}

func (g *ghServer) Exchange(_ context.Context, code string) (string, error) {
	if code == "bad" {
		return "", apperrs.ErrUnauthorized
	}
	return g.token, nil
}

func (g *ghServer) FetchUser(_ context.Context, _ string) (*auth.GitHubUser, error) {
	id := g.userID
	if id == "" {
		id = "42"
	}
	login := g.userLogin
	if login == "" {
		login = "onik97"
	}
	return &auth.GitHubUser{ID: id, Login: login, Name: "Name " + login, AvatarURL: "https://avatar/" + login}, nil
}

// fakeDefaultWorkspace is a minimal stand-in for the tenancy domain's
// BindDefaultWorkspaceOwner seam for these auth-domain integration tests,
// which drive the real SQLite-backed auth.Service without wiring up the
// tenancy domain.
type fakeDefaultWorkspace struct{}

func (fakeDefaultWorkspace) BindDefaultWorkspaceOwner(context.Context, string) error { return nil }

// fakePendingInviteResolver is a minimal stand-in for the tenancy domain's
// ResolvePendingInvites seam (Membership invites) for these auth-domain
// integration tests, which drive the real SQLite-backed auth.Service without
// wiring up the tenancy domain.
type fakePendingInviteResolver struct{}

func (fakePendingInviteResolver) ResolvePendingInvites(context.Context, string, string) error {
	return nil
}

// TestAuthIntegration_OwnerBootstrapAndAdmission drives first-user bootstrap and known-user admission against SQLite.
func TestAuthIntegration_OwnerBootstrapAndAdmission(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	gh := &ghServer{token: "at"}
	svc := auth.NewService(auth.Config{
		Secret:           []byte("integration-secret"),
		Users:            store.Users,
		Allowlist:        store.Allowlist,
		Settings:         store.Settings,
		GitHub:           gh,
		DefaultWorkspace: fakeDefaultWorkspace{},
		PendingInvites:   fakePendingInviteResolver{},
		Now:              func() time.Time { return time.Unix(1_700_000_000, 0) },
	})

	t.Run("first sign-in leads to the owner wizard", func(t *testing.T) {
		token, err := svc.Login(ctx, "good")
		require.NoError(t, err)
		userID, err := svc.Verify(token)
		require.NoError(t, err)
		st, err := svc.Me(ctx, userID)
		require.NoError(t, err)
		assert.True(t, st.NeedsOwnerWizard)

		require.NoError(t, svc.CompleteOwnerWizard(ctx, userID, "https://deploy.example.com"))
		st, err = svc.Me(ctx, userID)
		require.NoError(t, err)
		assert.False(t, st.NeedsOwnerWizard)
		assert.True(t, st.User.CanCreateWorkspace)

		conn, err := svc.GenerateConnectionToken(ctx, userID)
		require.NoError(t, err)
		claims, err := svc.ParseConnectionToken(conn.Token)
		require.NoError(t, err)
		assert.Equal(t, "https://deploy.example.com", claims.InstanceURL)
	})

	t.Run("known active user signs in without allowlist", func(t *testing.T) {
		owner := &auth.User{ID: "owner-id", Provider: auth.ProviderGitHub, ProviderUserID: "1", Login: "owner"}
		ownerRec, _, err := store.Users.UpsertUser(ctx, owner)
		require.NoError(t, err)
		require.NoError(t, store.Users.SetCanCreateWorkspace(ctx, ownerRec.ID, true))

		member := &auth.User{ID: "member-id", Provider: auth.ProviderGitHub, ProviderUserID: "2", Login: "member"}
		_, _, err = store.Users.UpsertUser(ctx, member)
		require.NoError(t, err)

		gh.userLogin = "member"
		gh.userID = "2"
		token, err := svc.Login(ctx, "good")
		require.NoError(t, err)
		userID, err := svc.Verify(token)
		require.NoError(t, err)
		st, err := svc.Me(ctx, userID)
		require.NoError(t, err)
		assert.True(t, st.NeedsFirstLoginWizard)
	})
}
func TestAuthIntegration_HTTPGateway(t *testing.T) {
	store := newTestStore(t)

	svc := auth.NewService(auth.Config{
		Secret:           []byte("integration-secret"),
		Users:            store.Users,
		Allowlist:        store.Allowlist,
		Settings:         store.Settings,
		GitHub:           &ghServer{token: "at"},
		DefaultWorkspace: fakeDefaultWorkspace{},
		PendingInvites:   fakePendingInviteResolver{},
	})

	handler := auth.NewHandler(svc)
	mux := http.NewServeMux()
	mux.Handle("/auth", handler.Routes())
	mux.Handle("/auth/", handler.Routes())
	mux.Handle("/api/auth", svc.RequireAuth(handler.ProtectedRoutes()))
	mux.Handle("/api/auth/", svc.RequireAuth(handler.ProtectedRoutes()))

	cb := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=good&state=st8", nil)
	req.AddCookie(&http.Cookie{Name: "nexul_oauth_state", Value: "st8"})
	mux.ServeHTTP(cb, req)
	require.Equal(t, http.StatusFound, cb.Code)
	token := strings.TrimPrefix(cb.Header().Get("Location"), "http://example.com/login?token=")
	require.NotEmpty(t, token)

	// /api/auth/me resolves the user record behind the session token.
	me := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(me, meReq)
	require.Equal(t, http.StatusOK, me.Code)
	assert.Contains(t, me.Body.String(), `"login":"onik97"`)
	assert.Contains(t, me.Body.String(), `"needs_owner_wizard":true`)

	ow := httptest.NewRecorder()
	owReq := httptest.NewRequest(http.MethodPost, "/api/auth/onboarding/owner",
		strings.NewReader(`{"instance_url":"https://deploy.example.com"}`))
	owReq.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(ow, owReq)
	require.Equal(t, http.StatusOK, ow.Code)
	assert.Contains(t, ow.Body.String(), `"can_create_workspace":true`)

	// Members list is owner-only and now reachable.
	mem := httptest.NewRecorder()
	memReq := httptest.NewRequest(http.MethodGet, "/api/auth/members", nil)
	memReq.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(mem, memReq)
	require.Equal(t, http.StatusOK, mem.Code)
}

// TestAuthIntegration_PATLifecycle drives the personal-access-token lifecycle end to end over a real SQLite
// store: mint a personal access token, authenticate requests with it through
// RequireAuth, then revoke it and prove the very next request is rejected.
func TestAuthIntegration_PATLifecycle(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	svc := auth.NewService(auth.Config{
		Secret:           []byte("integration-secret"),
		Users:            store.Users,
		Allowlist:        store.Allowlist,
		Settings:         store.Settings,
		PATs:             store.PATs,
		GitHub:           &ghServer{token: "at"},
		DefaultWorkspace: fakeDefaultWorkspace{},
		PendingInvites:   fakePendingInviteResolver{},
	})

	// First sign-in becomes the owner.
	ownerToken, err := svc.Login(ctx, "good")
	require.NoError(t, err)
	ownerID, err := svc.Verify(ownerToken)
	require.NoError(t, err)
	require.NoError(t, svc.CompleteOwnerWizard(ctx, ownerID, "https://deploy.example.com"))

	raw, pat, err := svc.MintPAT(ctx, ownerID, "ci agent")
	require.NoError(t, err)
	require.NotNil(t, pat)

	// A request guarded by RequireAuth acts as the token's user.
	handler := svc.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := auth.UserFromCtx(r.Context())
		if u == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		httpxEnvelope(w, u.ID)
	}))
	me := httptest.NewRecorder()
	patReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	patReq.Header.Set("Authorization", "Bearer "+raw)
	handler.ServeHTTP(me, patReq)
	require.Equal(t, http.StatusOK, me.Code)
	assert.Equal(t, ownerID, me.Body.String())

	// Revocation is immediate for new requests.
	require.NoError(t, svc.RevokePAT(ctx, ownerID, pat.ID))
	me = httptest.NewRecorder()
	handler.ServeHTTP(me, patReq)
	require.Equal(t, http.StatusUnauthorized, me.Code)
}

func httpxEnvelope(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}
