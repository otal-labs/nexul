package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/workspace"
)

// A doc's project has no cascade, so removing a project that still holds one must be refused, never fail mid-delete.
func TestIntegration_ProjectDelete_RefusesAProjectThatHoldsADoc(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	p, err := f.svc.workspaceSvc.Create(as(uOwner), wsDefault, "Notes", "NOTE", "")
	require.NoError(t, err)
	doc, err := f.svc.docsSvc.Create(as(uOwner), p.ID, "Requirements", "")
	require.NoError(t, err)
	routes := workspace.NewHandler(f.svc.workspaceSvc).Routes()
	call := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequest(method, path, nil).WithContext(as(uOwner)))
		return rec
	}

	impact := call(http.MethodGet, "/api/projects/"+p.ID+"/impact")
	require.Equal(t, http.StatusOK, impact.Code)
	assert.Contains(t, impact.Body.String(), `"docs":1`)
	assert.Equal(t, http.StatusConflict, call(http.MethodDelete, "/api/projects/"+p.ID).Code)

	require.NoError(t, f.svc.docsSvc.Delete(as(uOwner), doc.ID))
	require.Less(t, call(http.MethodDelete, "/api/projects/"+p.ID).Code, 300)
}
