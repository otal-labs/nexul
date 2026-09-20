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
		members.roleIDs["u-plain"] = "role-editors"

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
