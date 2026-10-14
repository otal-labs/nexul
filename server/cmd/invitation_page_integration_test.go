package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// TestIntegration_InvitePage_NamesProjectsOnlyAfterSignIn walks one link through the real store: the public preview
// names the instance and what each role means but no project, and after the provider round trip the acceptance names
// the projects whose rules let any of the grant's roles in, in their list order.
func TestIntegration_InvitePage_NamesProjectsOnlyAfterSignIn(t *testing.T) {
	f := newPermFixture(t)
	ctx := t.Context()
	_, err := f.store.Settings.Set(ctx, "https://nexul.example.com")
	require.NoError(t, err)
	_, err = f.store.Settings.SetGitHubOAuth(ctx, "client", "secret")
	require.NoError(t, err)
	_, err = f.db.Exec(`UPDATE roles SET description = 'Reads what is shared.' WHERE id = 'role-reader';
INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at) VALUES
  ('p-web', 'Website relaunch', 2, 'workspace-default', 0, 0),
  ('p-mobile', 'Mobile app', 1, 'workspace-default', 0, 0),
  ('p-payroll', 'Payroll', 3, 'workspace-default', 0, 0);
INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at) VALUES
  ('project', 'p-web', 'role', 'role-reader', '["projects:open"]', '[]', 1, 1),
  ('project', 'p-mobile', 'role', 'role-plain', '["tickets:read"]', '[]', 1, 1),
  ('project', 'p-payroll', 'role', 'role-writer', '["projects:open"]', '[]', 1, 1);`)
	require.NoError(t, err)
	authSvc := auth.NewService(auth.Config{
		Secret: []byte("0123456789abcdef0123456789abcdef"), SPAOrigin: "https://nexul.example.com",
		Users: f.store.Users, OAuthHandoffs: f.store.OAuthHandoffs, Settings: f.store.Settings, Sessions: f.store.Sessions, PATs: f.store.PATs,
		GitHub: codeProvider{"sam": {ID: "provider-sam", Login: "sam", Name: "Sam"}},
	})
	authSvc.SetInvitationGate(invitationAuthGate{svc: f.svc.invitationSvc})
	h := auth.NewHandler(authSvc).Routes()
	created, err := f.svc.invitationSvc.Create(as(uOwner), uOwner, tenancy.CreateInvitationInput{ExpiresInDays: 1,
		Grants: []*tenancy.InvitationGrant{{WorkspaceID: wsDefault, RoleIDs: []string{"role-plain", "role-reader"}}}})
	require.NoError(t, err)
	token := created.URL[strings.Index(created.URL, "#")+1:]

	rec := httptest.NewRecorder()
	tenancy.NewInvitationHandler(f.svc.invitationSvc).PublicRoutes().ServeHTTP(rec,
		httptest.NewRequest(http.MethodPost, "/api/invitations/preview", strings.NewReader(`{"token":"`+token+`"}`)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	for _, project := range []string{"project-general", "p-web", "Website relaunch"} {
		assert.NotContains(t, rec.Body.String(), project, "the preview names no project")
	}
	var preview tenancy.InvitationPreview
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	assert.Equal(t, "nexul.example.com", preview.InstanceName)
	require.Len(t, preview.Grants, 1)
	assert.Equal(t, "Reads what is shared.", preview.Grants[0].Roles[1].Description)

	landed := invitationSignIn(t, h, token, "sam")
	acceptanceToken, found := strings.CutPrefix(landed.Fragment, "acceptance-token=")
	require.True(t, found, landed.String())
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/invitations/acceptance", strings.NewReader(`{"acceptance_token":"`+acceptanceToken+`"}`)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var acceptance auth.InvitationAcceptance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &acceptance))
	require.Len(t, acceptance.Grants, 1)
	assert.Equal(t, []auth.InvitationProject{{ID: "project-general", Name: "General"}, {ID: "p-web", Name: "Website relaunch"}}, acceptance.Grants[0].Projects)
	assert.Equal(t, "Reads what is shared.", acceptance.Grants[0].Roles[1].Description)
}
