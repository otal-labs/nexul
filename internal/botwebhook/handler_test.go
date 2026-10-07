package botwebhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandler_Routes drives each gateway route as the HTTP client sends it: methods, paths, the deleted query, and
// the status each answer carries.
func TestHandler_Routes(t *testing.T) {
	repo := newFakeRepo()
	routes := NewHandler(newTestService(repo)).Routes()
	serve := func(user, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(as(user), method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	rec := serve(editor, http.MethodPost, "/api/conversations/c-eng/botwebhooks", `{"name":"CI"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created Bot
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Contains(t, created.URL, "/api/botwebhooks/"+created.ID+"/")
	assert.NotContains(t, rec.Body.String(), `"token"`)

	assert.Equal(t, http.StatusBadRequest, serve(editor, http.MethodPost, "/api/conversations/c-eng/botwebhooks", `{`).Code)

	rec = serve(editor, http.MethodPatch, "/api/botwebhooks/"+created.ID, `{"name":"Builds"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"name":"Builds"`)
	assert.Equal(t, http.StatusBadRequest, serve(editor, http.MethodPatch, "/api/botwebhooks/"+created.ID, `[`).Code)
	assert.Equal(t, http.StatusNotFound, serve(editor, http.MethodPatch, "/api/botwebhooks/b-gone", `{}`).Code)

	assert.Equal(t, http.StatusForbidden, serve(editor, http.MethodDelete, "/api/botwebhooks/"+created.ID, "").Code)
	assert.Equal(t, http.StatusNoContent, serve(deleter, http.MethodDelete, "/api/botwebhooks/"+created.ID, "").Code)

	rec = serve(editor, http.MethodGet, "/api/conversations/c-eng/botwebhooks?deleted=true", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var deleted []Bot
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &deleted))
	require.Len(t, deleted, 1)
	assert.Equal(t, created.ID, deleted[0].ID)
	assert.Empty(t, deleted[0].URL)
}
