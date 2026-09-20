package workspace

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T, allowCreate bool) (*Handler, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	owner := &fakeOwner{allowCreate: allowCreate}
	return NewHandler(newTestService(repo, owner)), repo
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

func decodeProject(t *testing.T, rec *httptest.ResponseRecorder) Project {
	t.Helper()
	var p Project
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	return p
}

func TestHandler_List(t *testing.T) {
	t.Run("lists a workspace's projects", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend", Position: 0, WorkspaceID: "ws-1"}
		rec := do(t, h.Routes(), http.MethodGet, "/api/projects?workspace_id=ws-1", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var projects []Project
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &projects))
		require.Len(t, projects, 1)
		assert.Equal(t, "Backend", projects[0].Name)
	})
	t.Run("missing workspace_id is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodGet, "/api/projects", "", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("repo error is 500", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.listErr = errors.New("db down")
		rec := do(t, h.Routes(), http.MethodGet, "/api/projects?workspace_id=ws-1", "", "")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestHandler_Create(t *testing.T) {
	t.Run("owner creates a project", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects", `{"workspace_id":"ws-1","name":"Backend","prefix":"BE"}`, "u-1")
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "Backend", decodeProject(t, rec).Name)
		assert.Equal(t, "BE", decodeProject(t, rec).Prefix)
		assert.Equal(t, "ws-1", decodeProject(t, rec).WorkspaceID)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		h, _ := newTestHandler(t, false)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects", `{"workspace_id":"ws-1","name":"Backend","prefix":"BE"}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("bad json is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects", `{`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing workspace_id is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects", `{"name":"Backend","prefix":"BE"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects", `{"workspace_id":"ws-1","name":" "}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("creates a project with an icon", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects", `{"workspace_id":"ws-1","name":"Backend","prefix":"BE","icon":"Server"}`, "u-1")
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, ProjectIconServer, decodeProject(t, rec).Icon)
	})
	t.Run("invalid icon is 400", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects", `{"workspace_id":"ws-1","name":"Backend","prefix":"BE","icon":"bogus"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_Get(t *testing.T) {
	h, repo := newTestHandler(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "Backend"}
	rec := do(t, h.Routes(), http.MethodGet, "/api/projects/p-1", "", "")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "p-1", decodeProject(t, rec).ID)

	rec = do(t, h.Routes(), http.MethodGet, "/api/projects/nope", "", "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Rename(t *testing.T) {
	t.Run("owner renames", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/projects/p-1", `{"name":"New"}`, "u-1")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "New", decodeProject(t, rec).Name)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		h, _ := newTestHandler(t, false)
		rec := do(t, h.Routes(), http.MethodPatch, "/api/projects/p-1", `{"name":"New"}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("owner renames with an icon", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/projects/p-1", `{"name":"New","icon":"Globe"}`, "u-1")
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, ProjectIconGlobe, decodeProject(t, rec).Icon)
	})
	t.Run("invalid icon is 400", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "Old"}
		rec := do(t, h.Routes(), http.MethodPatch, "/api/projects/p-1", `{"name":"New","icon":"bogus"}`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("empty project deletes", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodDelete, "/api/projects/p-1", "", "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("non-empty project conflicts", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "p-1"
		rec := do(t, h.Routes(), http.MethodDelete, "/api/projects/p-1", "", "u-1")
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestHandler_Impact(t *testing.T) {
	t.Run("returns delete impact", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		repo.ticketPro["t-1"] = "p-1"
		rec := do(t, h.Routes(), http.MethodGet, "/api/projects/p-1/impact", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var impact DeleteImpact
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &impact))
		assert.Equal(t, DeleteImpact{Tickets: 1}, impact)
	})
	t.Run("missing project is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodGet, "/api/projects/nope/impact", "", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_Reorder(t *testing.T) {
	t.Run("owner reorders", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A", WorkspaceID: "ws-1"}
		repo.projects["p-2"] = &Project{ID: "p-2", Name: "B", WorkspaceID: "ws-1"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects/reorder", `{"workspace_id":"ws-1","ids":["p-2","p-1"]}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("non-owner is forbidden", func(t *testing.T) {
		h, _ := newTestHandler(t, false)
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects/reorder", `{"workspace_id":"ws-1","ids":["p-1"]}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_Repos(t *testing.T) {
	t.Run("add then list then remove", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects/p-1/repos", `{"owner":"acme","name":"app"}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)

		rec = do(t, h.Routes(), http.MethodGet, "/api/projects/p-1/repos", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var repos []RepoRef
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &repos))
		require.Len(t, repos, 1)
		assert.Equal(t, "acme/app", repos[0].FullName)
		assert.Equal(t, "github", repos[0].ConnectorID)

		rec = do(t, h.Routes(), http.MethodDelete, "/api/projects/repos/acme/app", "", "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("add with an explicit connector_id persists it", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects/p-1/repos", `{"owner":"acme","name":"app","connector_id":"gitlab-self-hosted"}`, "u-1")
		assert.Equal(t, http.StatusNoContent, rec.Code)

		rec = do(t, h.Routes(), http.MethodGet, "/api/projects/p-1/repos", "", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var repos []RepoRef
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &repos))
		require.Len(t, repos, 1)
		assert.Equal(t, "gitlab-self-hosted", repos[0].ConnectorID)
	})
	t.Run("non-owner cannot add a repo", func(t *testing.T) {
		h, repo := newTestHandler(t, false)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects/p-1/repos", `{"owner":"acme","name":"app"}`, "u-1")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("missing repo is 404", func(t *testing.T) {
		h, _ := newTestHandler(t, true)
		rec := do(t, h.Routes(), http.MethodDelete, "/api/projects/repos/acme/app", "", "u-1")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("bad add-repo json is 400", func(t *testing.T) {
		h, repo := newTestHandler(t, true)
		repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
		rec := do(t, h.Routes(), http.MethodPost, "/api/projects/p-1/repos", `{`, "u-1")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_MoveTicket(t *testing.T) {
	h, repo := newTestHandler(t, true)
	repo.projects["p-1"] = &Project{ID: "p-1", Name: "A"}
	repo.ticketPro["t-1"] = "old"
	rec := do(t, h.Routes(), http.MethodPost, "/api/projects/p-1/tickets/t-1", "", "")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "p-1", repo.ticketPro["t-1"])

	rec = do(t, h.Routes(), http.MethodPost, "/api/projects/nope/tickets/t-1", "", "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
