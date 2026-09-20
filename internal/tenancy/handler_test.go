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

func TestHandler_Create(t *testing.T) {
	t.Run("creates a workspace", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1")
		assert.Equal(t, http.StatusCreated, rec.Code)
		var w Workspace
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &w))
		assert.Equal(t, "Acme", w.Name)
	})
	t.Run("no user in context is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("bad json is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":" "}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("repo error is 500", func(t *testing.T) {
		h, repo := newTestHandler(t)
		repo.createErr = errors.New("db down")
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces", `{"name":"Acme"}`, "u-1")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
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
	t.Run("no user in context is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces", "", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("repo error is 500", func(t *testing.T) {
		h, repo := newTestHandler(t)
		repo.listErr = errors.New("db down")
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces", "", "u-1")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
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
		svc := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakePlaysGate{})
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
	t.Run("no user in context is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/me", "", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("non-member is not found", func(t *testing.T) {
		h, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/me", "", "u-nobody")
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
}
