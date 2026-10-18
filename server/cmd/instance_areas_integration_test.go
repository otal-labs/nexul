package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/dns"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// uOtherOwner owns a second workspace: an Owner there, but not the instance Owner.
const uOtherOwner = "u-other-owner"

func withOtherWorkspaceOwner(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uOtherOwner, Provider: auth.ProviderGitHub, ProviderUserID: uOtherOwner, Login: uOtherOwner})
	require.NoError(t, err)
	require.NoError(t, f.store.Workspaces.Create(ctx, &tenancy.Workspace{ID: "workspace-other", Name: "Other", Slug: "other", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-owner-other", WorkspaceID: "workspace-other", Name: "Owner", IsOwnerRole: true, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uOtherOwner, WorkspaceID: "workspace-other", RoleID: "role-owner-other", CreatedAt: now}))
}

// status sends one request to routes as user and returns its status.
func status(t *testing.T, routes http.Handler, user, method, path, body string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(as(user)))
	return rec.Code
}

// TestInstanceAreas_BobManagesDNSButNotConnectors: the instance Owner hands a member whose roles hold no instance bit
// DNS and gateways over HTTP, and their role read on GitHub over MCP; they manage DNS, read GitHub alone, and
// connect or disconnect nothing.
func TestInstanceAreas_BobManagesDNSButNotConnectors(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	s := f.svc
	permissionRoutes := access.NewHandler(s.accessSvc).Routes()
	update := tool(t, access.MCPTools(s.accessSvc), "permission_overwrite_update")
	list := tool(t, access.MCPTools(s.accessSvc), "permission_overwrite_list")

	require.Equal(t, http.StatusNoContent, status(t, permissionRoutes, uOwner, http.MethodPut, "/api/permissions",
		`{"resource_type":"instance_area","resource_ids":["dns"],"user_ids":["`+uPlain+`"],"actions":["dns:read","dns:write"],"state":"allow"}`))
	_, err := update.Call(as(uOwner), json.RawMessage(`{"resource_type":"instance_area","resource_ids":["connector:github"],"role_ids":["role-plain"],"actions":["connectors:read"],"state":"allow"}`))
	require.NoError(t, err)

	t.Run("over HTTP", func(t *testing.T) {
		dnsRoutes := s.dnsHandler.Routes()
		connectorRoutes := connectors.NewHandler(s.connectorsSvc).Routes()
		assert.Equal(t, http.StatusOK, status(t, dnsRoutes, uPlain, http.MethodGet, "/api/dns/gateways", ""))
		assert.Equal(t, http.StatusBadRequest, status(t, dnsRoutes, uPlain, http.MethodPost, "/api/dns/gateways", `{}`), "past the gate, refused for its input")
		assert.Equal(t, http.StatusForbidden, status(t, dnsRoutes, uPlain, http.MethodDelete, "/api/dns/gateways/g-1", ""), "dns:delete is not handed out")
		assert.Equal(t, http.StatusForbidden, status(t, dnsRoutes, uWriter, http.MethodGet, "/api/dns/gateways", ""), "a rule naming bob is not for anyone else")
		assert.Equal(t, http.StatusForbidden, status(t, connectorRoutes, uPlain, http.MethodPost, "/api/connectors/github/disconnect", ""))

		rec := httptest.NewRecorder()
		connectorRoutes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/connectors", nil).WithContext(as(uPlain)))
		require.Equal(t, http.StatusOK, rec.Code)
		var listed []connectors.ConnectorStatus
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
		require.Len(t, listed, 1, "only the connector their role reads")
		assert.Equal(t, "github", listed[0].Connector.ID)
	})

	t.Run("over MCP", func(t *testing.T) {
		_, err := tool(t, dns.MCPTools(s.dnsSvc), "gateway_list").Call(as(uPlain), json.RawMessage(`{}`))
		require.NoError(t, err)
		_, err = tool(t, dns.MCPTools(s.dnsSvc), "gateway_list").Call(as(uWriter), json.RawMessage(`{}`))
		assert.Equal(t, forbidden, outcome(err))

		explained := func(area string) map[permissions.Action]access.Answer {
			out, err := list.Call(as(uPlain), json.RawMessage(`{"resource_type":"instance_area","resource_id":"`+area+`","user_id":"`+uPlain+`"}`))
			require.NoError(t, err)
			byAction := map[permissions.Action]access.Answer{}
			for _, a := range out.(*access.Explanation).Effective {
				byAction[a.Action] = a
			}
			return byAction
		}
		dnsAnswers := explained("dns")
		assert.True(t, dnsAnswers[permissions.DNSWrite].Allowed)
		assert.Equal(t, "Allowed on DNS and gateways for u-plain", dnsAnswers[permissions.DNSWrite].Source.Sentence)
		github := explained("connector:github")
		assert.True(t, github[permissions.ConnectorsRead].Allowed)
		assert.Equal(t, "Allowed on connector GitHub for role Member", github[permissions.ConnectorsRead].Source.Sentence)
		assert.False(t, github[permissions.ConnectorsWrite].Allowed)
		assert.Equal(t, "Not allowed: role Member doesn't grant it", github[permissions.ConnectorsWrite].Source.Sentence)
	})

	t.Run("instance_permissions on /api/auth/me lists what any area hands them", func(t *testing.T) {
		me, err := s.authSvc.Me(as(uPlain), uPlain)
		require.NoError(t, err)
		assert.Subset(t, me.InstancePermissions, []string{"dns:read", "dns:write", "connectors:read"})
		assert.NotContains(t, me.InstancePermissions, "connectors:write")
		assert.NotContains(t, me.InstancePermissions, "dns:delete")
	})
}

// TestInstanceAreas_OnlyTheInstanceOwnerWrites: holding every instance bit, or owning another workspace, writes no
// instance area's rules, over HTTP or MCP, nor reads as the instance Owner on /api/auth/me; the instance Owner does.
func TestInstanceAreas_OnlyTheInstanceOwnerWrites(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withOtherWorkspaceOwner(t, f)
	s := f.svc
	routes := access.NewHandler(s.accessSvc).Routes()
	update := tool(t, access.MCPTools(s.accessSvc), "permission_overwrite_update")
	list := tool(t, access.MCPTools(s.accessSvc), "permission_overwrite_list")
	body := `{"resource_type":"instance_area","resource_ids":["dns"],"user_ids":["` + uPlain + `"],"actions":["dns:read"],"state":"allow"}`

	for _, user := range []string{uSteward, uOtherOwner, uPlain} {
		assert.Equal(t, http.StatusForbidden, status(t, routes, user, http.MethodPut, "/api/permissions", body), user)
		assert.Equal(t, http.StatusForbidden, status(t, routes, user, http.MethodGet, "/api/permissions?resource_type=instance_area&resource_id=dns", ""), user)
		_, err := update.Call(as(user), json.RawMessage(body))
		assert.Equal(t, forbidden, outcome(err), user)
		_, err = list.Call(as(user), json.RawMessage(`{"resource_type":"instance_area","resource_id":"dns"}`))
		assert.Equal(t, forbidden, outcome(err), user)
		me, err := s.authSvc.Me(as(user), user)
		require.NoError(t, err)
		assert.False(t, me.InstanceOwner, "/api/auth/me hides the access sections from %s", user)
	}
	me, err := s.authSvc.Me(as(uOwner), uOwner)
	require.NoError(t, err)
	assert.True(t, me.InstanceOwner, "/api/auth/me shows the instance Owner the access sections")
	assert.Equal(t, http.StatusNoContent, status(t, routes, uOwner, http.MethodPut, "/api/permissions", body))
	assert.Equal(t, http.StatusOK, status(t, routes, uOwner, http.MethodGet, "/api/permissions?resource_type=instance_area&resource_id=dns", ""))
	assert.Equal(t, http.StatusNotFound, status(t, routes, uOwner, http.MethodPut, "/api/permissions",
		`{"resource_type":"instance_area","resource_ids":["billing"],"user_ids":["`+uPlain+`"],"actions":["dns:read"],"state":"allow"}`))
	assert.Equal(t, http.StatusBadRequest, status(t, routes, uOwner, http.MethodPut, "/api/permissions",
		`{"resource_type":"instance_area","resource_ids":["dns"],"user_ids":["`+uPlain+`"],"actions":["connectors:write"],"state":"allow"}`))

	held, err := s.accessSvc.HoldsAnywhere(context.Background(), uPlain, permissions.DNSRead)
	require.NoError(t, err)
	assert.True(t, held)
}

// TestInstanceAreas_TheInstanceOwnerIsNeverDenied: rules denying every area to the Owner role and to the instance Owner
// by name take nothing from them.
func TestInstanceAreas_TheInstanceOwnerIsNeverDenied(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	s := f.svc
	for _, area := range s.accessSvc.InstanceAreas() {
		for _, to := range []access.Targets{{UserIDs: []string{uOwner}}, {RoleIDs: []string{"role-owner"}}} {
			require.NoError(t, s.accessSvc.SetRules(context.Background(), uOwner, "instance_area", []string{area.Key}, to, area.Actions, access.StateDeny))
		}
	}
	for _, area := range s.accessSvc.InstanceAreas() {
		for _, action := range area.Actions {
			held, err := s.accessSvc.HoldsArea(context.Background(), uOwner, area.Key, action)
			require.NoError(t, err)
			assert.True(t, held, "the instance Owner holds %s on %s", action, area.Key)
		}
	}
	me, err := s.authSvc.Me(as(uOwner), uOwner)
	require.NoError(t, err)
	assert.Len(t, me.InstancePermissions, len(permissions.AllActions()))
	require.NoError(t, s.accessSvc.SetRules(context.Background(), uOwner, "instance_area", []string{"dns"}, access.Targets{UserIDs: []string{uOwner}}, []permissions.Action{permissions.DNSWrite}, access.StateFallback),
		"the instance Owner can always take back a rule")
}

// TestInstanceAreas_AnotherWorkspacesOwnerHoldsNoArea: owning a second workspace hands over no instance area, over HTTP,
// MCP and /api/auth/me, while its workspace actions stay; a rule naming one area gives that area's actions and no other.
func TestInstanceAreas_AnotherWorkspacesOwnerHoldsNoArea(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withOtherWorkspaceOwner(t, f)
	s := f.svc
	dnsRoutes := s.dnsHandler.Routes()
	connectorRoutes := connectors.NewHandler(s.connectorsSvc).Routes()
	gateways := tool(t, dns.MCPTools(s.dnsSvc), "gateway_list")

	t.Run("without a rule", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, status(t, dnsRoutes, uOtherOwner, http.MethodGet, "/api/dns/gateways", ""))
		assert.Equal(t, http.StatusForbidden, status(t, connectorRoutes, uOtherOwner, http.MethodPost, "/api/connectors/github/disconnect", ""))
		_, err := gateways.Call(as(uOtherOwner), json.RawMessage(`{}`))
		assert.Equal(t, forbidden, outcome(err))
		_, err = gateways.Call(as(uOwner), json.RawMessage(`{}`))
		require.NoError(t, err, "the instance Owner still lists them")

		me, err := s.authSvc.Me(as(uOtherOwner), uOtherOwner)
		require.NoError(t, err)
		assert.NotContains(t, me.InstancePermissions, "dns:read")
		assert.NotContains(t, me.InstancePermissions, "instance:write")
		assert.Contains(t, me.InstancePermissions, "projects:create", "the workspace actions of their own workspace stay")
	})

	t.Run("a rule for one area gives that area alone", func(t *testing.T) {
		permissionRoutes := access.NewHandler(s.accessSvc).Routes()
		require.Equal(t, http.StatusNoContent, status(t, permissionRoutes, uOwner, http.MethodPut, "/api/permissions",
			`{"resource_type":"instance_area","resource_ids":["dns"],"user_ids":["`+uOtherOwner+`"],"actions":["dns:read"],"state":"allow"}`))

		assert.Equal(t, http.StatusOK, status(t, dnsRoutes, uOtherOwner, http.MethodGet, "/api/dns/gateways", ""))
		_, err := gateways.Call(as(uOtherOwner), json.RawMessage(`{}`))
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, status(t, dnsRoutes, uOtherOwner, http.MethodDelete, "/api/dns/gateways/g-1", ""), "only the action the rule sets")
		assert.Equal(t, http.StatusForbidden, status(t, connectorRoutes, uOtherOwner, http.MethodPost, "/api/connectors/github/disconnect", ""), "and only the area it sits on")

		me, err := s.authSvc.Me(as(uOtherOwner), uOtherOwner)
		require.NoError(t, err)
		assert.Contains(t, me.InstancePermissions, "dns:read")
		assert.NotContains(t, me.InstancePermissions, "dns:write")
	})
}

// TestRoles_AnotherWorkspacesOwnerCannotHandOutAnInstanceArea: owning a workspace is not holding its instance areas,
// so its Owner can't put dns:write on a role, add it to one, or give themselves a role carrying it; the instance Owner can.
func TestRoles_AnotherWorkspacesOwnerCannotHandOutAnInstanceArea(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withOtherWorkspaceOwner(t, f)
	s := f.svc
	ctx := context.Background()
	now := time.Now()
	for _, r := range []*roles.Role{
		{ID: "role-docs-other", WorkspaceID: "workspace-other", Name: "Docs", Permissions: grant("docs:read")},
		{ID: "role-ops-other", WorkspaceID: "workspace-other", Name: "Ops", Permissions: grant("dns:write")},
	} {
		r.CreatedAt, r.UpdatedAt = now, now
		require.NoError(t, f.store.Roles.Create(ctx, r))
	}
	dnsWrite := grant("dns:write")

	t.Run("another workspace's Owner", func(t *testing.T) {
		_, err := s.rolesSvc.Create(ctx, "workspace-other", uOtherOwner, "Network", "", dnsWrite, nil, false)
		assert.ErrorIs(t, err, apperrs.ErrForbidden, "create a role carrying it")
		_, err = s.rolesSvc.Update(ctx, "workspace-other", "role-docs-other", uOtherOwner, "Docs", nil, grant("docs:read", "dns:write"), nil, nil)
		assert.ErrorIs(t, err, apperrs.ErrForbidden, "add it to a role")
		err = s.tenancySvc.UpdateMemberRoles(ctx, uOtherOwner, "workspace-other", uOtherOwner, []string{"role-ops-other"}, nil)
		assert.ErrorIs(t, err, apperrs.ErrForbidden, "give themselves a role carrying it")
		_, err = s.rolesSvc.Update(ctx, "workspace-other", "role-docs-other", uOtherOwner, "Docs renamed", nil, grant("docs:read"), nil, nil)
		assert.NoError(t, err, "other roles are theirs to edit")
	})

	t.Run("the instance Owner", func(t *testing.T) {
		made, err := s.rolesSvc.Create(ctx, "workspace-default", uOwner, "Network", "", dnsWrite, nil, false)
		require.NoError(t, err)
		_, err = s.rolesSvc.Update(ctx, "workspace-default", "role-editor", uOwner, "Editor", nil, grant("docs:read", "docs:write", "dns:write"), nil, nil)
		require.NoError(t, err)
		require.NoError(t, s.tenancySvc.UpdateMemberRoles(ctx, uOwner, "workspace-default", uPlain, []string{made.ID}, nil))
	})
}

// TestInvitations_AnotherWorkspacesOwnerCannotInviteIntoAnInstanceArea: an invitation giving a role that carries
// dns:write is refused to the Owner of the workspace it lands in, who doesn't hold DNS, and allowed to the instance Owner.
func TestInvitations_AnotherWorkspacesOwnerCannotInviteIntoAnInstanceArea(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withOtherWorkspaceOwner(t, f)
	s := f.svc
	ctx := context.Background()
	now := time.Now()
	_, err := f.store.Settings.Set(ctx, "https://nexul.example.com")
	require.NoError(t, err)
	for _, r := range []*roles.Role{
		{ID: "role-ops-other", WorkspaceID: "workspace-other", Name: "Ops", Permissions: grant("dns:write"), CreatedAt: now, UpdatedAt: now},
		{ID: "role-ops-default", WorkspaceID: "workspace-default", Name: "Ops", Permissions: grant("dns:write"), CreatedAt: now, UpdatedAt: now},
	} {
		require.NoError(t, f.store.Roles.Create(ctx, r))
	}
	invite := func(actor, workspaceID, roleID string) error {
		_, err := s.invitationSvc.Create(as(actor), actor, tenancy.CreateInvitationInput{ExpiresInDays: 1,
			Grants: []*tenancy.InvitationGrant{{WorkspaceID: workspaceID, RoleID: roleID}}})
		return err
	}

	require.ErrorIs(t, invite(uOtherOwner, "workspace-other", "role-ops-other"), apperrs.ErrForbidden)
	require.NoError(t, invite(uOwner, "workspace-default", "role-ops-default"))
}
