package tenancy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T) (*Handler, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	return NewHandler(newTestService(repo)), repo
}

func do(t *testing.T, h http.Handler, method, path, body, userID string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if userID != "" {
		r = r.WithContext(WithUserID(r.Context(), userID))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestHandler_WorkspaceRoutes_Refusals(t *testing.T) {
	tests := []struct {
		name   string
		seed   func(repo *fakeRepo)
		method string
		path   string
		body   string
		user   string
		want   int
	}{
		{"create without a signed-in user", nil, http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "", http.StatusBadRequest},
		{"create from a malformed body", nil, http.MethodPost, "/api/workspaces", `{`, "u-1", http.StatusBadRequest},
		{"create without a name", nil, http.MethodPost, "/api/workspaces", `{"name":" "}`, "u-1", http.StatusBadRequest},
		{"create when the repo fails", func(r *fakeRepo) { r.createErr = errors.New("db down") }, http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1", http.StatusInternalServerError},
		{"list without a signed-in user", nil, http.MethodGet, "/api/workspaces", "", "", http.StatusBadRequest},
		{"list when the repo fails", func(r *fakeRepo) { r.listErr = errors.New("db down") }, http.MethodGet, "/api/workspaces", "", "u-1", http.StatusInternalServerError},
		{"read me without a signed-in user", nil, http.MethodGet, "/api/workspaces/ws-1/me", "", "", http.StatusBadRequest},
		{"read me as a non-member", nil, http.MethodGet, "/api/workspaces/ws-1/me", "", "u-nobody", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newTestHandler(t)
			if tt.seed != nil {
				tt.seed(repo)
			}
			rec := do(t, h.Routes(), tt.method, tt.path, tt.body, tt.user)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

func TestHandler_Create(t *testing.T) {
	t.Run("creates a workspace", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1")
		assert.Equal(t, http.StatusCreated, rec.Code)
		var w Workspace
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &w))
		assert.Equal(t, "Acme", w.Name)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("lists the caller's workspaces", func(t *testing.T) {
		h, _ := newTestHandler(t)
		createRec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1")
		require.Equal(t, http.StatusCreated, createRec.Code)

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces", "", "u-1")
		assert.Equal(t, http.StatusOK, rec.Code)
		var workspaces []Workspace
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &workspaces))
		require.Len(t, workspaces, 1)
		assert.Equal(t, "Acme", workspaces[0].Name)
	})
}

func TestHandler_Me(t *testing.T) {
	t.Run("returns the caller's role name in the workspace", func(t *testing.T) {
		h, _ := newTestHandler(t)
		createRec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1")
		require.Equal(t, http.StatusCreated, createRec.Code)
		var w Workspace
		require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &w))

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/"+w.ID+"/me", "", "u-1")
		assert.Equal(t, http.StatusOK, rec.Code)
		var body meResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "role-name-owner-role-"+w.ID, body.RoleName)
	})
	t.Run("returns the caller's resolved permission list", func(t *testing.T) {
		repo := newFakeRepo()
		wsPerms := newFakeWorkspacePermissionGate()
		svc := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakeDefaultsGate{}, newFakeAccountGate())
		h := NewHandler(svc)
		createRec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1")
		require.Equal(t, http.StatusCreated, createRec.Code)
		var w Workspace
		require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &w))
		wsPerms.perms["u-1"] = []string{"projects:write", "members:write"}

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/"+w.ID+"/me", "", "u-1")
		assert.Equal(t, http.StatusOK, rec.Code)
		var body meResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, []string{"projects:write", "members:write"}, body.Permissions)
	})
	t.Run("no permissions is an empty list, not null", func(t *testing.T) {
		h, _ := newTestHandler(t)
		createRec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1")
		require.Equal(t, http.StatusCreated, createRec.Code)
		var w Workspace
		require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &w))

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/"+w.ID+"/me", "", "u-1")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), `"permissions":null`)
	})
}

func TestHandler_InviteMember(t *testing.T) {
	t.Run("invites a login with no User yet, returns 204", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/members", `{"login":"bob","role_id":"role-editor"}`, "actor")
		assert.Equal(t, http.StatusNoContent, rec.Code)

		pending, err := f.invites.ListByWorkspace(context.Background(), "ws-1")
		require.NoError(t, err)
		require.Len(t, pending, 1)
		assert.Equal(t, "bob", pending[0].Login)
	})
	t.Run("actor without members:write is forbidden", func(t *testing.T) {
		f := newInviteFixture()
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/members", `{"login":"bob","role_id":"role-editor"}`, "actor")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("login not allowlisted is a 400", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		f.allowlist.allowed["bob"] = false
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/members", `{"login":"bob","role_id":"role-editor"}`, "actor")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// TestHandler_MemberRoutesTakeRoleIDs: each member route takes role_ids beside the one role_id older clients send.
func TestHandler_MemberRoutesTakeRoleIDs(t *testing.T) {
	heldBy := func(t *testing.T, f *inviteTestFixture, userID string) []string {
		t.Helper()
		m, err := f.svc.Member(context.Background(), "ws-1", userID)
		require.NoError(t, err)
		return m.RoleIDs
	}
	t.Run("patch replaces the set, and an empty one leaves no role", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		h := NewHandler(f.svc).Routes()

		rec := do(t, h, http.MethodPatch, "/api/workspaces/ws-1/members/u-bob", `{"role_ids":["role-editor","role-viewer"]}`, "actor")
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.Equal(t, []string{"role-editor", "role-viewer"}, heldBy(t, f, "u-bob"))

		rec = do(t, h, http.MethodPatch, "/api/workspaces/ws-1/members/u-bob", `{"role_ids":[]}`, "actor")
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.Empty(t, heldBy(t, f, "u-bob"))
	})
	t.Run("put adds an account with several roles", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		f.accounts.accounts["u-bob"] = &TeamAccount{ID: "u-bob", Login: "bob", Status: "active"}

		rec := do(t, NewHandler(f.svc).Routes(), http.MethodPut, "/api/workspaces/ws-1/members/u-bob", `{"role_ids":["role-editor","role-viewer"]}`, "actor")
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.Equal(t, []string{"role-editor", "role-viewer"}, heldBy(t, f, "u-bob"))
	})
	t.Run("post gives a signed-in login several roles, and holds a pending one to a single role", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		f.users.register("bob", "u-bob")
		h := NewHandler(f.svc).Routes()

		rec := do(t, h, http.MethodPost, "/api/workspaces/ws-1/members", `{"login":"bob","role_ids":["role-editor","role-viewer"]}`, "actor")
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.Equal(t, []string{"role-editor", "role-viewer"}, heldBy(t, f, "u-bob"))

		rec = do(t, h, http.MethodPost, "/api/workspaces/ws-1/members", `{"login":"carol","role_ids":["role-editor","role-viewer"]}`, "actor")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "a pending invite gives one role")
	})
}

func TestHandler_ListMembers(t *testing.T) {
	t.Run("returns the roster and pending invites", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		f.users.register("bob", "u-bob")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		require.NoError(t, f.svc.InviteMember(context.Background(), "actor", "ws-1", "carol", "role-editor"))
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/members", "", "actor")
		assert.Equal(t, http.StatusOK, rec.Code)
		var body MembersList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Members, 1)
		assert.Equal(t, "bob", body.Members[0].Login)
		require.Len(t, body.Invites, 1)
		assert.Equal(t, "carol", body.Invites[0].Login)
	})
	t.Run("actor without members:write is forbidden", func(t *testing.T) {
		f := newInviteFixture()
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/members", "", "actor")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_RemoveMember(t *testing.T) {
	t.Run("removes a non-owner member, returns 204", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodDelete, "/api/workspaces/ws-1/members/u-bob", "", "actor")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("removing the Owner is a 400", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-owner", WorkspaceID: "ws-1", RoleID: "owner-role-ws-1"}))
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodDelete, "/api/workspaces/ws-1/members/u-owner", "", "actor")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_ChangeMemberRole(t *testing.T) {
	t.Run("changes a non-owner member's role, returns 204", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/members/u-bob", `{"role_id":"role-admin"}`, "actor")
		assert.Equal(t, http.StatusNoContent, rec.Code)

		roleID, err := f.svc.MemberRoleID(context.Background(), "ws-1", "u-bob")
		require.NoError(t, err)
		assert.Equal(t, "role-admin", roleID)
	})
	t.Run("assigning the Owner role is a 400", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))
		h := NewHandler(f.svc)

		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/members/u-bob", `{"role_id":"owner-role-ws-1"}`, "actor")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("a removed field with a value is refused, naming what replaced it", func(t *testing.T) {
		for _, body := range []string{`{"deny":["docs:delete"]}`, `{"role_id":"role-admin","allow":["docs:read"]}`, `{"every_project":"none"}`,
			`{"project_access":[{"project_id":"p-web","allow":["tickets:read"]}]}`} {
			f := newInviteFixture()
			f.grantManageMembers("actor")
			require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))

			rec := do(t, NewHandler(f.svc).Routes(), http.MethodPatch, "/api/workspaces/ws-1/members/u-bob", body, "actor")

			assert.Equal(t, http.StatusBadRequest, rec.Code, body)
			assert.Regexp(t, "a role|list the person", rec.Body.String(), body)
			roleID, err := f.svc.MemberRoleID(context.Background(), "ws-1", "u-bob")
			require.NoError(t, err)
			assert.Equal(t, "role-editor", roleID, "nothing changed: %s", body)
		}
	})
	t.Run("removed fields sent empty change nothing, as before", func(t *testing.T) {
		f := newInviteFixture()
		f.grantManageMembers("actor")
		require.NoError(t, f.repo.AddMember(context.Background(), &Member{UserID: "u-bob", WorkspaceID: "ws-1", RoleID: "role-editor"}))

		rec := do(t, NewHandler(f.svc).Routes(), http.MethodPatch, "/api/workspaces/ws-1/members/u-bob", `{"allow":[],"deny":[],"project_access":[]}`, "actor")

		assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		assert.Empty(t, f.repo.events)
	})
}

func TestHandler_SetMentionChipTemplate(t *testing.T) {
	newHandler := func(t *testing.T, perms ...string) (*Handler, string) {
		t.Helper()
		repo := newFakeRepo()
		wsPerms := newFakeWorkspacePermissionGate()
		svc := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakeDefaultsGate{}, newFakeAccountGate())
		w, err := svc.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		wsPerms.perms["u-1"] = perms
		return NewHandler(svc), w.ID
	}
	t.Run("bad json is invalid", func(t *testing.T) {
		h, id := newHandler(t, "workspaces:write")
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/"+id+"/mention-chip-template", `{`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("actor without workspaces:write is forbidden", func(t *testing.T) {
		h, id := newHandler(t, "members:write")
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/"+id+"/mention-chip-template", `{"mention_chip_template":"{ticket.Project}"}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("returns the workspace with its new template", func(t *testing.T) {
		h, id := newHandler(t, "workspaces:write")
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/"+id+"/mention-chip-template", `{"mention_chip_template":"{ticket.Project} {ticket.Ticket}"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		var w Workspace
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &w))
		assert.Equal(t, id, w.ID)
		assert.Equal(t, "{ticket.Project} {ticket.Ticket}", w.MentionChipTemplate)

		list := do(t, h.Routes(), http.MethodGet, "/api/workspaces", "", "u-1")
		require.Equal(t, http.StatusOK, list.Code)
		var workspaces []Workspace
		require.NoError(t, json.Unmarshal(list.Body.Bytes(), &workspaces))
		require.Len(t, workspaces, 1)
		assert.Equal(t, "{ticket.Project} {ticket.Ticket}", workspaces[0].MentionChipTemplate, "the list is where the browser reads the template")
	})
}

func TestHandler_Rename(t *testing.T) {
	newHandler := func(t *testing.T) (*Handler, string) {
		t.Helper()
		repo := newFakeRepo()
		wsPerms := newFakeWorkspacePermissionGate()
		svc := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakeDefaultsGate{}, newFakeAccountGate())
		w, err := svc.Create(context.Background(), "u-1", "Acme")
		require.NoError(t, err)
		_, err = svc.Create(context.Background(), "u-1", "Other")
		require.NoError(t, err)
		wsPerms.perms["u-1"] = []string{"workspaces:write"}
		return NewHandler(svc), w.ID
	}
	t.Run("a slug alone keeps the name", func(t *testing.T) {
		h, id := newHandler(t)
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/"+id, `{"slug":"acme-labs"}`, "u-1")
		require.Equal(t, http.StatusOK, rec.Code)
		var w Workspace
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &w))
		assert.Equal(t, "Acme", w.Name)
		assert.Equal(t, "acme-labs", w.Slug)
	})
	t.Run("a slug another workspace holds is a conflict", func(t *testing.T) {
		h, id := newHandler(t)
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/"+id, `{"slug":"other"}`, "u-1")
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
	t.Run("a reserved slug is invalid", func(t *testing.T) {
		h, id := newHandler(t)
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/"+id, `{"slug":"settings"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
