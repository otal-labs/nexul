package docs

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestDocsHandler_ListKeepsCreationTimeAfterAnEdit(t *testing.T) {
	h, repo := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))
	edited := repo.docs[created.ID].UpdatedAt.Add(48 * time.Hour)
	repo.docs[created.ID].UpdatedAt = edited

	rec := serve(t, h, http.MethodGet, "/api/docs?project_id=project-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var list []DocListItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list, 1)
	assert.True(t, list[0].CreatedAt.Equal(created.CreatedAt), "created_at must stay the creation time")
	assert.True(t, list[0].UpdatedAt.Equal(edited))
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
}

func TestDocsHandler_Delete(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))

	t.Run("deletes a doc", func(t *testing.T) {
		rec := serve(t, h, http.MethodDelete, "/api/docs/"+created.ID, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
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
}

func TestDocsHandler_RestoreVersion_AnswersWithTheDocOnTop(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))
	serve(t, h, http.MethodPut, "/api/docs/"+created.ID, `{"title":"B","body":"2"}`)

	rec := serve(t, h, http.MethodPost, "/api/docs/"+created.ID+"/versions/1/restore", "")

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodeDoc(t, rec)
	assert.Equal(t, 3, got.Version)
	assert.Equal(t, "A", got.Title)
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
}

func TestDocsHandler_LockUnlock(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))

	rec := serve(t, h, http.MethodPost, "/api/docs/"+created.ID+"/lock", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, decodeDoc(t, rec).Locked)

	rec = serve(t, h, http.MethodPut, "/api/docs/"+created.ID, `{"title":"B","body":"2"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, "a locked doc refuses an edit")
	assert.Contains(t, rec.Body.String(), "locked")

	rec = serve(t, h, http.MethodPost, "/api/docs/"+created.ID+"/unlock", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, decodeDoc(t, rec).Locked)
}

func TestDocsHandler_WatchUnwatch(t *testing.T) {
	h, _ := newDocsHandler()
	created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))
	watchers := func(rec *httptest.ResponseRecorder) Watchers {
		t.Helper()
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var ws Watchers
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &ws))
		return ws
	}

	got := watchers(serve(t, h, http.MethodPut, "/api/docs/"+created.ID+"/watchers/me", ""))
	assert.True(t, got.Watching)
	assert.Len(t, got.Watchers, 1)
	assert.True(t, watchers(serve(t, h, http.MethodGet, "/api/docs/"+created.ID+"/watchers", "")).Watching)

	got = watchers(serve(t, h, http.MethodDelete, "/api/docs/"+created.ID+"/watchers/me", ""))
	assert.False(t, got.Watching)
	assert.NotNil(t, got.Watchers, "an empty list is [], never null")
	assert.Empty(t, got.Watchers)
}

func TestDocsHandler_UnreadableIsNotFound(t *testing.T) {
	repo := newFakeRepo()
	created, err := newTestService(repo).Create(testCtx(), "project-1", "A", "1")
	require.NoError(t, err)

	rec := serve(t, NewHandler(newDenyService(repo)).Routes(), http.MethodGet, "/api/docs/"+created.ID, "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	rec = serve(t, NewHandler(newReadOnlyService(repo)).Routes(), http.MethodDelete, "/api/docs/"+created.ID, "")
	assert.Equal(t, http.StatusForbidden, rec.Code, "an action refused on a doc they read is forbidden")
}

func TestDocsHandler_ImportMarkdown(t *testing.T) {
	h, _ := newDocsHandler()

	rec := serve(t, h, http.MethodPost, "/api/docs/import", `{"project_id":"project-1","title":"Imported","body":"# From markdown"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	d := decodeDoc(t, rec)
	assert.Equal(t, "Imported", d.Title)
	assert.Equal(t, "project-1", d.ProjectID)
	assert.True(t, json.Valid([]byte(d.Body)), "stored body is canonical JSON")

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
}

func TestDocsHandler_Refusals(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"create without a title", http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"","body":"x"}`, http.StatusBadRequest},
		{"create without a project", http.MethodPost, "/api/docs", `{"title":"Spec","body":"x"}`, http.StatusBadRequest},
		{"create from a malformed body", http.MethodPost, "/api/docs", `{"title":`, http.StatusBadRequest},
		{"get a missing doc", http.MethodGet, "/api/docs/nope", "", http.StatusNotFound},
		{"update a missing doc", http.MethodPut, "/api/docs/nope", `{"title":"B","body":"2"}`, http.StatusNotFound},
		{"update without a title", http.MethodPut, "/api/docs/{id}", `{"title":" ","body":"2"}`, http.StatusBadRequest},
		{"delete a missing doc", http.MethodDelete, "/api/docs/again", "", http.StatusNotFound},
		{"search without a query", http.MethodGet, "/api/docs/search", "", http.StatusBadRequest},
		{"get a missing version", http.MethodGet, "/api/docs/{id}/versions/9", "", http.StatusNotFound},
		{"get a version by a non-number", http.MethodGet, "/api/docs/{id}/versions/abc", "", http.StatusBadRequest},
		{"restore a missing version", http.MethodPost, "/api/docs/{id}/versions/9/restore", "", http.StatusNotFound},
		{"restore a version by a non-number", http.MethodPost, "/api/docs/{id}/versions/abc/restore", "", http.StatusBadRequest},
		{"archive a missing doc", http.MethodPost, "/api/docs/nope/archive", "", http.StatusNotFound},
		{"lock a missing doc", http.MethodPost, "/api/docs/nope/lock", "", http.StatusNotFound},
		{"watch a missing doc", http.MethodPut, "/api/docs/nope/watchers/me", "", http.StatusNotFound},
		{"import without a title", http.MethodPost, "/api/docs/import", `{"project_id":"project-1","title":"","body":"x"}`, http.StatusBadRequest},
		{"export a missing doc", http.MethodGet, "/api/docs/nope/export", "", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newDocsHandler()
			created := decodeDoc(t, serve(t, h, http.MethodPost, "/api/docs", `{"project_id":"project-1","title":"A","body":"1"}`))
			rec := serve(t, h, tt.method, strings.ReplaceAll(tt.path, "{id}", created.ID), tt.body)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}
