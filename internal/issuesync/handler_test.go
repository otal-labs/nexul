package issuesync

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	f := newFixture()
	routes := NewHandler(f.svc).Routes()
	serve := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))
		return rec
	}

	assert.Equal(t, http.StatusBadRequest, serve(http.MethodPut, "/api/projects/p-1/issue-sync", "{").Code)
	assert.Equal(t, http.StatusBadRequest, serve(http.MethodPut, "/api/projects/p-1/issue-sync", `{"owner":"acme","repo":"other","enabled":true}`).Code)

	rec := serve(http.MethodPut, "/api/projects/p-1/issue-sync", `{"owner":"acme","repo":"app","enabled":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"project_id":"p-1","owner":"acme","repo":"app","label":"nexul","mode":"label","enabled":true}`, rec.Body.String())

	rec = serve(http.MethodGet, "/api/projects/p-1/issue-sync", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `[{"project_id":"p-1","owner":"acme","repo":"app","label":"nexul","mode":"label","enabled":true}]`, rec.Body.String())
}
