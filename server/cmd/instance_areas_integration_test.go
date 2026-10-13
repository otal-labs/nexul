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
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// uOtherOwner owns a second workspace: an Owner, so never denied, but not the instance Owner.
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

// TestInstanceAreas_AnOwnerIsNeverDenied: rules denying every area to the Owner role and to each Owner by name take
// nothing from the instance Owner or an Owner of another workspace.
func TestInstanceAreas_AnOwnerIsNeverDenied(t *testing.T) {
	f := newPermFixture(t)
	withOtherWorkspaceOwner(t, f)
	s := f.svc
	for _, area := range s.accessSvc.InstanceAreas() {
		for _, to := range []access.Targets{{UserIDs: []string{uOwner, uOtherOwner}}, {RoleIDs: []string{"role-owner", "role-owner-other"}}} {
			require.NoError(t, s.accessSvc.SetRules(context.Background(), uOwner, "instance_area", []string{area.Key}, to, area.Actions, access.StateDeny))
		}
	}
	for _, owner := range []string{uOwner, uOtherOwner} {
		for _, area := range s.accessSvc.InstanceAreas() {
			for _, action := range area.Actions {
				held, err := s.accessSvc.HoldsArea(context.Background(), owner, area.Key, action)
				require.NoError(t, err)
				assert.True(t, held, "%s holds %s on %s", owner, action, area.Key)
			}
		}
		me, err := s.authSvc.Me(as(owner), owner)
		require.NoError(t, err)
		assert.Len(t, me.InstancePermissions, len(permissions.AllActions()), owner)
	}
	require.NoError(t, s.accessSvc.SetRules(context.Background(), uOwner, "instance_area", []string{"dns"}, access.Targets{UserIDs: []string{uOwner}}, []permissions.Action{permissions.DNSWrite}, access.StateFallback),
		"the instance Owner can always take back a rule")
}
