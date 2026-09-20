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
	t.Run("empty title is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":""}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing project_id and workspace_id is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories", `{"title":"Title"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories", `{"title":`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("empty project_id with workspace_id creates a workspace-scoped memory", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories", `{"workspace_id":"workspace-1","title":"Team tone"}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		m := decodeMemory(t, rec)
		assert.Empty(t, m.ProjectID)
		assert.Equal(t, "workspace-1", m.WorkspaceID)
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
	t.Run("missing memory is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/nope", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
	t.Run("missing memory is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/memories/nope", `{"title":"B"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("empty title is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/memories/"+created.ID, `{"title":" "}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestMemoriesHandler_Delete(t *testing.T) {
	h, _ := newMemoriesHandler()
	created := decodeMemory(t, serve(t, h, http.MethodPost, "/api/memories", `{"project_id":"project-1","title":"A"}`))

	t.Run("deletes a memory", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/memories/"+created.ID, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("missing memory is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/memories/again", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
	t.Run("non-numeric version is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/"+created.ID+"/versions/nope", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing memory versions is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/nope/versions", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("missing memory version is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/memories/nope/versions/1", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("malformed revert body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/"+created.ID+"/revert", `{"version":`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("reverting an unknown version is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/"+created.ID+"/revert", `{"version":99}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
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
	t.Run("missing destination project id and workspace id is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/"+created.ID+"/clone", `{}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed clone body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/"+created.ID+"/clone", `{"project_id":`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("cloning an unknown memory is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/nope/clone", `{"project_id":"project-2"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("empty project_id with workspace_id clones to the workspace", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/memories/"+created.ID+"/clone", `{"workspace_id":"workspace-1"}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		clone := decodeMemory(t, rec)
		assert.Empty(t, clone.ProjectID)
		assert.Equal(t, "workspace-1", clone.WorkspaceID)
	})
}

func TestMemoriesHandler_Forbidden(t *testing.T) {
	repo := newFakeRepo()
	created, err := newTestService(repo).Create(testCtx(), "project-1", "", "A", "when", "body", false, "")
	require.NoError(t, err)
	h := NewHandler(newDenyService(repo)).Routes()

	rec := serve(t, h, http.MethodGet, "/api/memories/"+created.ID, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
