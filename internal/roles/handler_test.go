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
	t.Run("show_locked_actions is read from the request, and left out it keeps the role's", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")

		var created, saved, kept Role
		rec := do(t, h.Routes(), http.MethodPost, "/api/workspaces/ws-1/roles", `{"name":"Dev","actions":[],"show_locked_actions":true}`, "u-owner")
		require.Equal(t, http.StatusCreated, rec.Code)
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		assert.True(t, created.ShowLockedActions)

		path := "/api/workspaces/ws-1/roles/" + created.ID
		rec = do(t, h.Routes(), http.MethodPatch, path, `{"name":"Dev","actions":[]}`, "u-owner")
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &kept))
		assert.True(t, kept.ShowLockedActions)

		rec = do(t, h.Routes(), http.MethodPatch, path, `{"name":"Dev","actions":[],"show_locked_actions":false}`, "u-owner")
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))
		assert.False(t, saved.ShowLockedActions)
	})
	t.Run("a request without welcome_steps keeps the role's, and one with them replaces them", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		require.NoError(t, repo.Create(context.Background(), &Role{ID: "role-dev", WorkspaceID: "ws-1", Name: "Dev", WelcomeSteps: []string{"github"}}))

		var r Role
		rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/roles/role-dev", `{"name":"Dev","actions":[]}`, "u-owner")
		require.Equal(t, http.StatusOK, rec.Code)
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &r))
		assert.Equal(t, []string{"github"}, r.WelcomeSteps)
		rec = do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/roles/role-dev", `{"name":"Dev","actions":[],"welcome_steps":["phone"]}`, "u-owner")
		require.Equal(t, http.StatusOK, rec.Code)
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &r))
		assert.Equal(t, []string{"phone"}, r.WelcomeSteps)
	})
	t.Run("a request without description keeps the role's, and one over 140 characters is refused", func(t *testing.T) {
		h, repo, members := newTestHandler(t)
		seedOwner(t, repo, members, "ws-1", "u-owner")
		require.NoError(t, repo.Create(context.Background(), &Role{ID: "role-dev", WorkspaceID: "ws-1", Name: "Dev", Description: "Works tickets."}))
		patch := func(body string) (int, Role) {
			rec := do(t, h.Routes(), http.MethodPatch, "/api/workspaces/ws-1/roles/role-dev", body, "u-owner")
			stored, err := repo.Get(context.Background(), "role-dev")
			require.NoError(t, err)
			return rec.Code, *stored
		}

		code, stored := patch(`{"name":"Developer","actions":[]}`)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, "Works tickets.", stored.Description)
		code, stored = patch(`{"name":"Developer","actions":[],"description":"` + strings.Repeat("a", 141) + `"}`)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Equal(t, "Works tickets.", stored.Description, "a refused save changes nothing")
		// The cap counts characters, not bytes, so a description in any script gets the same 140.
		umlauts := strings.Repeat("ü", 140)
		code, stored = patch(`{"name":"Developer","actions":[],"description":"` + umlauts + `"}`)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, umlauts, stored.Description)
		code, stored = patch(`{"name":"Developer","actions":[],"description":""}`)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, stored.Description)
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
}

func TestHandler_Refusals(t *testing.T) {
	const clonePath = "/api/workspaces/ws-1/roles/role-editors/clone"
	seedWorkspace := func(t *testing.T, repo *fakeRepo, members *fakeMemberGate) {
		seedOwner(t, repo, members, "ws-1", "u-owner")
	}
	seedEditors := func(t *testing.T, repo *fakeRepo, members *fakeMemberGate) {
		seedWorkspace(t, repo, members)
		require.NoError(t, repo.Create(context.Background(), &Role{ID: "role-editors", WorkspaceID: "ws-1", Name: "Editors"}))
	}
	tests := []struct {
		name   string
		seed   func(t *testing.T, repo *fakeRepo, members *fakeMemberGate)
		method string
		path   string
		body   string
		user   string
		want   int
	}{
		{"create without a signed-in user", seedWorkspace, http.MethodPost, "/api/workspaces/ws-1/roles", `{"name":"Editors"}`, "", http.StatusBadRequest},
		{"create from a malformed body", nil, http.MethodPost, "/api/workspaces/ws-1/roles", `{`, "u-owner", http.StatusBadRequest},
		{"create without roles:write", func(t *testing.T, repo *fakeRepo, members *fakeMemberGate) {
			seedEditors(t, repo, members)
			members.roleIDs[memberKey("ws-1", "u-plain")] = "role-editors"
		}, http.MethodPost, "/api/workspaces/ws-1/roles", `{"name":"More"}`, "u-plain", http.StatusForbidden},
		{"list as an outsider", seedWorkspace, http.MethodGet, "/api/workspaces/ws-1/roles", "", "u-outsider", http.StatusNotFound},
		{"get a role as an outsider", seedWorkspace, http.MethodGet, "/api/workspaces/ws-1/roles/role-owner", "", "u-outsider", http.StatusNotFound},
		{"get a missing role", nil, http.MethodGet, "/api/workspaces/ws-1/roles/missing", "", "u-owner", http.StatusNotFound},
		{"rename the Owner role", seedWorkspace, http.MethodPatch, "/api/workspaces/ws-1/roles/role-owner", `{"name":"Not Owner"}`, "u-owner", http.StatusBadRequest},
		{"delete the Owner role", seedWorkspace, http.MethodDelete, "/api/workspaces/ws-1/roles/role-owner", "", "u-owner", http.StatusBadRequest},
		{"clone from a malformed body", seedEditors, http.MethodPost, clonePath, `{`, "u-owner", http.StatusBadRequest},
		{"clone into the same workspace", seedEditors, http.MethodPost, clonePath, `{"workspace_id":"ws-1"}`, "u-owner", http.StatusBadRequest},
		{"clone a missing role", seedEditors, http.MethodPost, "/api/workspaces/ws-1/roles/missing/clone", `{"workspace_id":"ws-2"}`, "u-owner", http.StatusNotFound},
		{"clone into a workspace the caller is not in", seedEditors, http.MethodPost, clonePath, `{"workspace_id":"ws-2"}`, "u-owner", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo, members := newTestHandler(t)
			if tt.seed != nil {
				tt.seed(t, repo, members)
			}
			rec := do(t, h.Routes(), tt.method, tt.path, tt.body, tt.user)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}
