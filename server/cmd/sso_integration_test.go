package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// Without a key that unlocks sso, the settings read back as saved, the button is off for the signed-out page, a write is
// refused for lack of the license, and the sign-in routes lead back to the login page, all through the real router.
func TestSSORoutes_WithoutAUnlockingKey_ThroughTheRouter(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	_, token := signedInOwner(t, store, svc)
	require.NoError(t, store.SSO.Set(t.Context(), sso.Settings{
		IssuerURL: "https://login.example.com", ClientID: "client-1", ClientSecret: "never-in-a-response", DisplayName: "Acme SSO",
		JoinRoleID: "role-member", Only: true, UpdatedAt: time.Now(),
	}))

	rec := callAPI(t, h, http.MethodGet, "/api/instance/sso", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"configured":true`)
	assert.Contains(t, rec.Body.String(), `"unlocked":false`)
	assert.Contains(t, rec.Body.String(), `"active":false`)
	assert.NotContains(t, rec.Body.String(), "never-in-a-response")

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/auth/sso", nil))
	require.Equal(t, http.StatusOK, rec.Code, "the sign-in page asks signed out")
	assert.JSONEq(t, `{"active":false,"name":"","only":false}`, rec.Body.String(), "no button and no switch while unlicensed")

	rec = callAPI(t, h, http.MethodPut, "/api/instance/sso", token, `{"display_name":"Acme"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "needs a license key")
	rec = callAPI(t, h, http.MethodPost, "/api/instance/sso/verify?check=issuer", token, `{}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/oidc", nil))
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "/login?error=not_configured")

	saved, err := store.SSO.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Acme SSO", saved.DisplayName, "the refused write left the saved settings alone")
}

func TestSSORoutes_NeedInstancePermissions(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	_, owner := signedInOwner(t, store, svc)
	member := signedInReader(t, store, svc)

	for _, c := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/instance/sso", ""},
		{http.MethodPut, "/api/instance/sso", `{}`},
		{http.MethodDelete, "/api/instance/sso", ""},
		{http.MethodPost, "/api/instance/sso/verify?check=issuer", `{}`},
	} {
		rec := callAPI(t, h, c.method, c.path, member, c.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s: %s", c.method, c.path, rec.Body.String())
	}

	rec := callAPI(t, h, http.MethodDelete, "/api/instance/sso", owner, "")
	require.Equal(t, http.StatusOK, rec.Code, "removing works without a key: %s", rec.Body.String())
	var audited []string
	for _, row := range auditRows(t, store) {
		audited = append(audited, row.Action)
	}
	assert.Contains(t, audited, "DELETE /api/instance/sso")
}

// signedInReader is bob, a member of the default workspace whose role holds no instance permission.
func signedInReader(t *testing.T, store *storage.Store, svc *coreServices) string {
	t.Helper()
	ctx := t.Context()
	_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-bob", Provider: auth.ProviderGitHub, ProviderUserID: "bob", Login: "bob"})
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, store.Roles.Create(ctx, &roles.Role{ID: "role-member", WorkspaceID: "workspace-default", Name: "Member", Permissions: permissions.SetOf(permissions.DocsRead), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-bob", WorkspaceID: "workspace-default", RoleID: "role-member", CreatedAt: now}))
	raw, _, err := svc.authSvc.MintPAT(ctx, "u-bob", "member")
	require.NoError(t, err)
	return raw
}
