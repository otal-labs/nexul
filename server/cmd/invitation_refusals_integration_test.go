package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// codeProvider answers each OAuth code with the provider account it stands for.
type codeProvider map[string]*auth.ProviderUser

func (p codeProvider) Exchange(_ context.Context, code string) (string, error) {
	if _, ok := p[code]; !ok {
		return "", apperrs.ErrUnauthorized
	}
	return code, nil
}

func (p codeProvider) FetchUser(_ context.Context, accessToken string) (*auth.ProviderUser, error) {
	return p[accessToken], nil
}

// invitationSignIn runs the browser's provider round trip for a link and returns where the callback sends it.
func invitationSignIn(t *testing.T, h http.Handler, token, code string) *url.URL {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/invitations/oauth", strings.NewReader(`{"provider":"github","token":"`+token+`"}`)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var start struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&start))
	authorize, err := url.Parse(start.URL)
	require.NoError(t, err)
	callback := httptest.NewRequest(http.MethodGet, "/auth/callback?code="+code+"&state="+authorize.Query().Get("state"), nil)
	for _, cookie := range rec.Result().Cookies() {
		callback.AddCookie(cookie)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, callback)
	require.Equal(t, http.StatusFound, rec.Code)
	landed, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	return landed
}

// TestIntegration_Invitation_TurnsExistingAccountsAwayAndStillAdmitsSomeoneNew walks one link through the real store:
// an account signed in and an account coming back through the provider are both refused, and the link then admits the
// new person it was made for.
func TestIntegration_Invitation_TurnsExistingAccountsAwayAndStillAdmitsSomeoneNew(t *testing.T) {
	f := newPermFixture(t)
	ctx := t.Context()
	_, err := f.store.Settings.Set(ctx, "https://nexul.example.com")
	require.NoError(t, err)
	_, err = f.store.Settings.SetGitHubOAuth(ctx, "client", "secret")
	require.NoError(t, err)
	authSvc := auth.NewService(auth.Config{
		Secret: []byte("0123456789abcdef0123456789abcdef"), SPAOrigin: "https://nexul.example.com",
		Users: f.store.Users, OAuthHandoffs: f.store.OAuthHandoffs, Settings: f.store.Settings, Sessions: f.store.Sessions, PATs: f.store.PATs,
		GitHub: codeProvider{
			"writer": {ID: uWriter, Login: uWriter, Name: "Writer"},
			"sam":    {ID: "provider-sam", Login: "sam", Name: "Sam"},
		},
	})
	authSvc.SetInvitationGate(invitationAuthGate{svc: f.svc.invitationSvc})
	h := auth.NewHandler(authSvc).Routes()
	created, err := f.svc.invitationSvc.Create(as(uOwner), uOwner, tenancy.CreateInvitationInput{ExpiresInDays: 1,
		Grants: []*tenancy.InvitationGrant{{WorkspaceID: "workspace-default", RoleID: "role-plain"}}})
	require.NoError(t, err)
	token := created.URL[strings.Index(created.URL, "#")+1:]

	session, err := authSvc.CreateSession(ctx, uWriter)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/invitations/acceptance", strings.NewReader(`{"token":"`+token+`"}`))
	req.Header.Set("Authorization", "Bearer "+session)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"code":"existing_account"`, "signed in")

	landed := invitationSignIn(t, h, token, "writer")
	refusal, err := url.ParseQuery(landed.Fragment)
	require.NoError(t, err)
	assert.Equal(t, "/invite", landed.Path)
	assert.Equal(t, "existing_account", refusal.Get("refused"), "through the provider")
	assert.Equal(t, uWriter, refusal.Get("login"))

	landed = invitationSignIn(t, h, token, "sam")
	acceptance, found := strings.CutPrefix(landed.Fragment, "acceptance-token=")
	require.True(t, found, "the link still works for someone new: %s", landed)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/invitations/redeem", strings.NewReader(`{"acceptance_token":"`+acceptance+`"}`)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	sam, err := f.store.Users.GetUserByProvider(ctx, auth.ProviderGitHub, "provider-sam")
	require.NoError(t, err)
	member, err := f.store.WorkspaceMembers.Member(ctx, "workspace-default", sam.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"role-plain"}, member.RoleIDs)
	writer, err := f.store.WorkspaceMembers.Member(ctx, "workspace-default", uWriter)
	require.NoError(t, err)
	assert.Equal(t, []string{"role-writer"}, writer.RoleIDs, "the refused account gained nothing")
}
