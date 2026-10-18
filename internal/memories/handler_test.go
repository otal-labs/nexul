package memories

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

func serve(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: "user-1"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newMemoriesHandler() (http.Handler, *fakeRepo) {
	repo := newFakeRepo()
	return NewHandler(newTestService(repo)).Routes(), repo
}

func decodeMemory(t *testing.T, rec *httptest.ResponseRecorder) *Memory {
	t.Helper()
	var m Memory
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m))
	return &m
}

func TestMemoriesHandler_Create(t *testing.T) {
	h, _ := newMemoriesHandler()

	t.Run("creates a memory", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"Deploy quirks","when_to_use":"when deploying","body":"body","always_included":true}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		m := decodeMemory(t, rec)
		assert.Equal(t, "Deploy quirks", m.Title)
		assert.Equal(t, "project-1", m.ProjectID)
		assert.Equal(t, "workspace-1", m.WorkspaceID)
		assert.True(t, m.AlwaysIncluded)
		assert.NotEmpty(t, m.ID)
	})
}

func TestMemoriesHandler_ListByProjectAndWorkspace(t *testing.T) {
	h, _ := newMemoriesHandler()
	serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"In project"}`)

	t.Run("lists by project_id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories?project_id=project-1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*Memory
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
		assert.Equal(t, "In project", list[0].Title)
	})
	t.Run("lists by workspace_id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories?workspace_id=workspace-1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*Memory
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
	})
}

func TestMemoriesHandler_Get(t *testing.T) {
	h, _ := newMemoriesHandler()
	created := decodeMemory(t, serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"A"}`))

	t.Run("gets a memory by id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/"+created.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "A", decodeMemory(t, rec).Title)
	})
}

func TestMemoriesHandler_Update(t *testing.T) {
	h, _ := newMemoriesHandler()
	created := decodeMemory(t, serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"A"}`))

	t.Run("updates fields", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/memories/"+created.ID, `{"title":"B","when_to_use":"use it","body":"new body","always_included":true}`)
		require.Equal(t, http.StatusOK, rec.Code)
		m := decodeMemory(t, rec)
		assert.Equal(t, "B", m.Title)
		assert.Equal(t, "use it", m.WhenToUse)
		assert.True(t, m.AlwaysIncluded)
	})
	t.Run("sets the footer, and a save without it keeps it", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/memories/"+created.ID, `{"title":"B","footer":true}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, decodeMemory(t, rec).Footer)
		rec = serve(t, h, http.MethodPut, "/api/memories/"+created.ID, `{"title":"C"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, decodeMemory(t, rec).Footer)
	})
}

func TestMemoriesHandler_Delete(t *testing.T) {
	h, _ := newMemoriesHandler()
	created := decodeMemory(t, serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"A"}`))

	t.Run("deletes a memory", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/memories/"+created.ID, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestMemoriesHandler_VersionsAndRevert(t *testing.T) {
	h, _ := newMemoriesHandler()
	created := decodeMemory(t, serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"A","body":"v1"}`))
	serve(t, h, http.MethodPut, "/api/memories/"+created.ID, `{"title":"A","body":"v2"}`)

	t.Run("lists versions newest first", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/"+created.ID+"/versions", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var versions []*MemoryVersion
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &versions))
		require.Len(t, versions, 2)
		assert.Equal(t, 2, versions[0].Version)
	})
	t.Run("gets one version", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/"+created.ID+"/versions/1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var v MemoryVersion
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
		assert.Contains(t, v.Body, "v1")
	})
	t.Run("reverts to a prior version", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/"+created.ID+"/revert", `{"version":1}`)
		require.Equal(t, http.StatusOK, rec.Code)
		m := decodeMemory(t, rec)
		assert.Equal(t, 3, m.Version)
		assert.Contains(t, m.Body, "v1")
	})
}

func TestMemoriesHandler_Clone(t *testing.T) {
	h, _ := newMemoriesHandler()
	created := decodeMemory(t, serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"A","body":"body"}`))

	t.Run("clones to another project", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/"+created.ID+"/clone", `{"project_id":"project-2"}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		clone := decodeMemory(t, rec)
		assert.NotEqual(t, created.ID, clone.ID)
		assert.Equal(t, "project-2", clone.ProjectID)
		assert.Equal(t, 1, clone.Version)
	})
}

func TestMemoriesHandler_InterviewTemplate(t *testing.T) {
	h, _ := newMemoriesHandler()

	t.Run("reads a project's template with project_id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/interview-template?project_id=project-1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"workspace_id":"workspace-1"`)
	})
}

func TestMemoriesHandler_Refusals(t *testing.T) {
	tests := []struct {
		name   string
		denied bool
		method string
		path   string
		body   string
		want   int
	}{
		{"create without memories:write", true, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"Title"}`, http.StatusForbidden},
		{"create for a workspace with no project", false, http.MethodPost, "/api/memories", `{"workspace_id":"workspace-1","title":"Team tone"}`, http.StatusBadRequest},
		{"create without a title", false, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":""}`, http.StatusBadRequest},
		{"create from a malformed body", false, http.MethodPost, "/api/memories", `{"title":`, http.StatusBadRequest},
		{"get a missing memory", false, http.MethodGet, "/api/memories/nope", "", http.StatusNotFound},
		{"get without memories:read", true, http.MethodGet, "/api/memories/{id}", "", http.StatusForbidden},
		{"update a missing memory", false, http.MethodPut, "/api/memories/nope", `{"title":"B"}`, http.StatusNotFound},
		{"update without a title", false, http.MethodPut, "/api/memories/{id}", `{"title":" "}`, http.StatusBadRequest},
		{"delete a missing memory", false, http.MethodDelete, "/api/memories/again", "", http.StatusNotFound},
		{"get a version by a non-number", false, http.MethodGet, "/api/memories/{id}/versions/nope", "", http.StatusBadRequest},
		{"list the versions of a missing memory", false, http.MethodGet, "/api/memories/nope/versions", "", http.StatusNotFound},
		{"get a version of a missing memory", false, http.MethodGet, "/api/memories/nope/versions/1", "", http.StatusNotFound},
		{"revert from a malformed body", false, http.MethodPost, "/api/memories/{id}/revert", `{"version":`, http.StatusBadRequest},
		{"revert to an unknown version", false, http.MethodPost, "/api/memories/{id}/revert", `{"version":99}`, http.StatusNotFound},
		{"clone without memories:clone", true, http.MethodPost, "/api/memories/{id}/clone", `{"project_id":"project-2"}`, http.StatusForbidden},
		{"clone into a workspace with no project", false, http.MethodPost, "/api/memories/{id}/clone", `{"workspace_id":"workspace-1"}`, http.StatusBadRequest},
		{"clone from a malformed body", false, http.MethodPost, "/api/memories/{id}/clone", `{"project_id":`, http.StatusBadRequest},
		{"clone an unknown memory", false, http.MethodPost, "/api/memories/nope/clone", `{"project_id":"project-2"}`, http.StatusNotFound},
		{"read the template of an unknown project", false, http.MethodGet, "/api/memories/interview-template?project_id=project-x", "", http.StatusNotFound},
		{"read the template with no project or workspace", false, http.MethodGet, "/api/memories/interview-template", "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newMemoriesHandler()
			created := decodeMemory(t, serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"A","body":"body"}`))
			if tt.denied {
				h = NewHandler(newDenyService(repo)).Routes()
			}
			rec := serve(t, h, tt.method, strings.ReplaceAll(tt.path, "{id}", created.ID), tt.body)
			assert.Equal(t, tt.want, rec.Code)
			assert.Len(t, repo.memories, 1, "a refused request writes nothing")
		})
	}
}
