package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// Move sign-ins answers whoever holds sign_ins:move, through a role or a rule on its instance area, and nobody else: the
// instance Owner by right, a role given it, a person given it on the area; a member without it and instance:write alone
// are refused, and a write without a key that unlocks sso is refused for the license.
func TestSSOMoves_AnswerHoldersOfMoveSignInsThroughTheRouter(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	h, store, svc := routerOver(t, db)
	ctx := t.Context()
	_, owner := signedInOwner(t, store, svc)
	member := signedInReader(t, store, svc)
	now := time.Now()
	require.NoError(t, store.Roles.Create(ctx, &roles.Role{ID: "role-mover", WorkspaceID: "workspace-default", Name: "Mover", Permissions: permissions.SetOf(permissions.SignInsMove), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.Roles.Create(ctx, &roles.Role{ID: "role-admin", WorkspaceID: "workspace-default", Name: "Admin", Permissions: permissions.SetOf(permissions.InstanceRead, permissions.InstanceWrite), CreatedAt: now, UpdatedAt: now}))
	token := func(user, role string) string {
		_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user[2:]})
		require.NoError(t, err)
		require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role, CreatedAt: now}))
		raw, _, err := svc.authSvc.MintPAT(ctx, user, "moves test")
		require.NoError(t, err)
		return raw
	}
	mover, admin, ruled := token("u-mover", "role-mover"), token("u-admin", "role-admin"), token("u-ruled", "role-member")
	_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES ('instance_area', 'sign_ins', 'person', 'u-ruled', '["sign_ins:move"]', '[]', 1, 1)`)
	require.NoError(t, err)
	require.NoError(t, store.SSO.Set(ctx, sso.Settings{
		IssuerURL: "https://login.example.com", ClientID: "client-1", ClientSecret: "s", DisplayName: "Acme SSO", JoinRoleID: "role-member", UpdatedAt: now,
	}))
	_, _, err = store.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-dana", Provider: auth.ProviderGitHub, ProviderUserID: "dana", Login: "dana"})
	require.NoError(t, err)
	require.NoError(t, store.SSO.Link(ctx, "u-dana", sso.Identity{Provider: sso.ProtocolSAML, Key: "https://idp.example.com#dana", Email: "dana@example.com"}))
	require.NoError(t, store.SSO.Create(ctx, "u-dup", sso.Identity{Provider: sso.ProtocolOIDC, Key: "https://login.example.com#dana", Email: "dana@example.com"}, "role-member", now))

	for name, tok := range map[string]string{"the instance Owner": owner, "a role holding it": mover, "a rule on its area": ruled} {
		rec := callAPI(t, h, http.MethodGet, "/api/instance/sso/moves", tok, "")
		require.Equal(t, http.StatusOK, rec.Code, "%s: %s", name, rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"protocol":"oidc"`, name)
		assert.Contains(t, rec.Body.String(), `"match":{"id":"u-dup"`, name)
	}
	for name, tok := range map[string]string{"a member": member, "instance:write alone": admin} {
		assert.Equal(t, http.StatusForbidden, callAPI(t, h, http.MethodGet, "/api/instance/sso/moves", tok, "").Code, name)
		assert.Equal(t, http.StatusForbidden, callAPI(t, h, http.MethodPost, "/api/instance/sso/move", tok, `{"from_account_id":"u-dup","to_account_id":"u-dana"}`).Code, name)
	}

	rec := callAPI(t, h, http.MethodPost, "/api/instance/sso/moves", mover, `{"pairs":[{"from_account_id":"u-dup","to_account_id":"u-dana"}]}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "needs a license key")
	rec = callAPI(t, h, http.MethodPut, "/api/instance/sso/expected/u-dana", mover, `{"email":"dana@example.com"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, http.StatusNotFound, callAPI(t, h, http.MethodDelete, "/api/instance/sso/expected/u-dana", mover, "").Code, "removing needs no key")
}
