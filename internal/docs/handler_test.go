package docs

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
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: "user-1", CanCreateWorkspace: false}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newDocsHandler() (http.Handler, *fakeRepo) {
	repo := newFakeRepo()
	return NewHandler(newTestService(repo)).Routes(), repo
}

func decodeDoc(t *testing.T, rec *httptest.ResponseRecorder) *Doc {
	t.Helper()
	var d Doc
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &d))
	return &d
}

func TestDocsHandler_Create(t *testing.T) {
	h, _ := newDocsHandler()

	t.Run("creates a doc", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"Spec","body":"body"}`)
		require.Equal(t, http.StatusCreated, rec.Code)
		d := decodeDoc(t, rec)
		assert.Equal(t, "Spec", d.Title)
		assert.Equal(t, "project-1", d.ProjectID)
		assert.Equal(t, 1, d.Version)
		assert.NotEmpty(t, d.ID)
	})
	t.Run("empty title is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"","body":"x"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing project_id is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs", `{"title":"Spec","body":"x"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("malformed body is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs", `{"title":`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDocsHandler_ListAndGet(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))

	t.Run("lists docs", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []DocListItem
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
		assert.True(t, list[0].CanOpen)
	})
	t.Run("gets a doc by id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/"+created.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "A", decodeDoc(t, rec).Title)
	})
	t.Run("missing doc is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/nope", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestDocsHandler_ListByProject(t *testing.T) {
	h, _ := newDocsHandler()
	serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"In project","body":"1"}`)
	serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-2","title":"Other project","body":"1"}`)

	rec := serve(t, h, http.MethodGet, "/api/docs?project_id=project-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var list []DocListItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list, 1)
	assert.Equal(t, "In project", list[0].Title)
}

func TestDocsHandler_Update(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))

	t.Run("updates and bumps version", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/docs/"+created.ID, `{"title":"B","body":"2"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		d := decodeDoc(t, rec)
		assert.Equal(t, "B", d.Title)
		assert.Equal(t, 2, d.Version)
	})
	t.Run("missing doc is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/docs/nope", `{"title":"B","body":"2"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("empty title is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPut, "/api/docs/"+created.ID, `{"title":" ","body":"2"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDocsHandler_Delete(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))

	t.Run("deletes a doc", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/docs/"+created.ID, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})
	t.Run("missing doc is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/docs/again", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestDocsHandler_Search(t *testing.T) {
	h, _ := newDocsHandler()
	serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"Storage Spine","body":"SQLite FTS5"}`)

	t.Run("finds matching docs", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/search?q=sqlite", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var results []SearchResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
		require.Len(t, results, 1)
		assert.Equal(t, "Storage Spine", results[0].Title)
	})
	t.Run("missing query is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/search", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDocsHandler_Versions(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))
	serve(t, h, http.MethodPut, "/api/docs/"+created.ID, `{"title":"B","body":"2"}`)

	t.Run("lists versions newest first", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/"+created.ID+"/versions", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var vs []DocVersion
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &vs))
		require.Len(t, vs, 2)
		assert.Equal(t, 2, vs[0].Version)
	})
	t.Run("gets a specific version", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/"+created.ID+"/versions/1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var v DocVersion
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v))
		assert.Equal(t, 1, v.Version)
		assert.Equal(t, "A", v.Title)
	})
	t.Run("missing version is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/"+created.ID+"/versions/9", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("non-numeric version is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/"+created.ID+"/versions/abc", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDocsHandler_ArchiveRestore(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))

	t.Run("archives a doc", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs/"+created.ID+"/archive", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, decodeDoc(t, rec).Archived)
	})
	t.Run("restores a doc", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs/"+created.ID+"/restore", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.False(t, decodeDoc(t, rec).Archived)
	})
	t.Run("missing doc is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs/nope/archive", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestDocsHandler_Forbidden(t *testing.T) {
	repo := newFakeRepo()
	created, err := newTestService(repo).Create(testCtx(), "project-1", "A", "1")
	require.NoError(t, err)
	h := NewHandler(newDenyService(repo)).Routes()

	rec := serve(t, h, http.MethodGet, "/api/docs/"+created.ID, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDocsHandler_ListDisclosure(t *testing.T) {
	repo := newFakeRepo()
	_, err := newTestService(repo).Create(testCtx(), "project-1", "secret title", "secret body")
	require.NoError(t, err)
	h := NewHandler(newDenyService(repo)).Routes()

	rec := serve(t, h, http.MethodGet, "/api/docs", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var items []DocListItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
	require.Len(t, items, 1)
	assert.False(t, items[0].CanOpen)
	assert.Equal(t, "secret title", items[0].Title)
}

func TestDocsHandler_ImportMarkdown(t *testing.T) {
	h, _ := newDocsHandler()

	rec := serve(t, h, http.MethodPost, "/api/docs/import", `{"project_id":"project-1","title":"Imported","body":"# From markdown"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	d := decodeDoc(t, rec)
	assert.Equal(t, "Imported", d.Title)
	assert.Equal(t, "project-1", d.ProjectID)
	assert.True(t, json.Valid([]byte(d.Body)), "stored body is canonical JSON")

	t.Run("empty title is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodPost, "/api/docs/import", `{"project_id":"project-1","title":"","body":"x"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestDocsHandler_ExportMarkdown(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"Spec","body":"**bold** body"}`))

	t.Run("exports body as markdown", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/"+created.ID+"/export", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Markdown string `json:"markdown"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "**bold** body", body.Markdown)
	})
	t.Run("missing doc is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/docs/nope/export", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
