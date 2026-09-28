package roles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T) (*Handler, *fakeRepo, *fakeMemberGate) {
	t.Helper()
	repo := newFakeRepo()
	members := newFakeMemberGate()
	return NewHandler(newTestService(repo, members)), repo, members
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
	t.Run("owner creates a custom role", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")

		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/roles", `{"name":"Editors","actions":["projects:write"]}`, "u-owner")
		assert.Equal(t, http.StatusCreated, rec.Code)
		var r Role
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &r))
		assert.Equal(t, "Editors", r.Name)
	})
	t.Run("no user in context is invalid", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/roles", `{"name":"Editors"}`, "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("bad json is invalid", func(t *testing.T) {
		h, _, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/roles", `{`, "u-owner")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("actor without roles:write is forbidden", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		editors := &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors"}
		require.NoError(t, repo.Create(context.Background(), editors))
		members.roleIDs[memberKey("ws-1", "u-plain")] = "role-editors"

		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/roles", `{"name":"More"}`, "u-plain")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("lists the workspace's roles", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/roles", "", "u-owner")
		assert.Equal(t, http.StatusOK, rec.Code)
		var got []Role
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Len(t, got, 1)
		assert.Equal(t, "Owner", got[0].Name)
	})
	t.Run("someone outside the workspace gets not found, for the list and for one role", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")

		assert.Equal(t, http.StatusNotFound, do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/roles", "", "u-outsider").Code)
		assert.Equal(t, http.StatusNotFound, do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/roles/role-owner", "", "u-outsider").Code)
	})
}

func TestHandler_Get(t *testing.T) {
	t.Run("gets a role by id", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")

		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/roles/role-owner", "", "u-owner")
		assert.Equal(t, http.StatusOK, rec.Code)
		var r Role
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &r))
		assert.Equal(t, "Owner", r.Name)
	})
	t.Run("missing role is not found", func(t *testing.T) {
		h, _, _ := newTestHandler(t)
		rec := do(t, h.Routes(), http.MethodGet, "/api/workspaces/ws-1/roles/missing", "", "u-owner")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_Update(t *testing.T) {
	t.Run("owner renames a custom role", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		editors := &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors"}
		require.NoError(t, repo.Create(context.Background(), editors))

		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/roles/role-editors", `{"name":"Reviewers","actions":["projects:write"]}`, "u-owner")
		assert.Equal(t, http.StatusOK, rec.Code)
		var r Role
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &r))
		assert.Equal(t, "Reviewers", r.Name)
	})
	t.Run("owner role can't be renamed", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/roles/role-owner", `{"name":"Not Owner"}`, "u-owner")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("owner deletes a custom role", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		editors := &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors"}
		require.NoError(t, repo.Create(context.Background(), editors))

		rec := do(t, h.Routes(), http.MethodDelete, "/api/workspaces/ws-1/roles/role-editors", "", "u-owner")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("owner role can't be deleted", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		rec := do(t, h.Routes(), http.MethodDelete, "/api/workspaces/ws-1/roles/role-owner", "", "u-owner")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_Clone(t *testing.T) {
	const path = "/api/workspaces/ws-1/roles/role-editors/clone"
	seed := func(t *testing.T, targetOwner bool) http.Handler {
		t.Helper()
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		require.NoError(t, repo.Create(context.Background(), &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors", Permissions: setFromActions([]string{"docs:read", "roles:clone"})}))
		if targetOwner {
			require.NoError(t, repo.Create(context.Background(), &Role{ID: "role-owner-2", WorkspaceID: "ws-2", Name: "Owner", IsOwnerRole: true}))
			members.roleIDs[memberKey("ws-2", "u-owner")] = "role-owner-2"
		}
		return h.Routes()
	}
	t.Run("owner of both workspaces gets the clone back", func(t *testing.T) {
		rec := do(t, seed(t, true), http.MethodPost, path, `{"workspace_id":"ws-2"}`, "u-owner")
		assert.Equal(t, http.StatusCreated, rec.Code)
		var r Role
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &r))
		assert.Equal(t, "ws-2", r.WorkspaceID)
		assert.Equal(t, "Editors", r.Name)
		assert.Equal(t, setFromActions([]string{"docs:read", "roles:clone"}), r.Permissions)
	})
	t.Run("bad json is invalid", func(t *testing.T) {
		rec := do(t, seed(t, true), http.MethodPost, path, `{`, "u-owner")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("same workspace is invalid", func(t *testing.T) {
		rec := do(t, seed(t, true), http.MethodPost, path, `{"workspace_id":"ws-1"}`, "u-owner")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing role is not found", func(t *testing.T) {
		rec := do(t, seed(t, true), http.MethodPost, "/api/workspaces/ws-1/roles/missing/clone", `{"workspace_id":"ws-2"}`, "u-owner")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("not a member of the target is forbidden", func(t *testing.T) {
		rec := do(t, seed(t, false), http.MethodPost, path, `{"workspace_id":"ws-2"}`, "u-owner")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
