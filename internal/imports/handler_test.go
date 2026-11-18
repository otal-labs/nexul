package imports

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_RunsAnImportOverTheGateway(t *testing.T) {
	svc, _ := newTestService(fakeSource{})
	routes := NewHandler(svc).Routes()
	call := func(user, method, path, body string) (int, Import) {
		req := httptest.NewRequestWithContext(as(user), method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		var imp Import
		if strings.HasPrefix(rec.Body.String(), "{") {
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &imp))
		}
		return rec.Code, imp
	}

	code, _ := call(testOwner, http.MethodPost, "/api/imports", "{")
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = call("u-writer", http.MethodPost, "/api/imports", `{"source":"fake","project_id":"p-1"}`)
	assert.Equal(t, http.StatusForbidden, code)
	code, started := call(testOwner, http.MethodPost, "/api/imports", `{"source":"fake","project_id":"p-1","credential":"secret"}`)
	require.Equal(t, http.StatusCreated, code)
	code, _ = call(testOwner, http.MethodPost, "/api/imports/"+started.ID+"/confirm", "")
	assert.Equal(t, http.StatusBadRequest, code, "a scanning import is not ready")

	svc.Pass(t.Context())
	code, read := call(testOwner, http.MethodGet, "/api/imports/"+started.ID, "")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, StateReady, read.State)
	code, confirmed := call(testOwner, http.MethodPost, "/api/imports/"+started.ID+"/confirm", "")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, StateImporting, confirmed.State)
	code, cancelled := call(testOwner, http.MethodPost, "/api/imports/"+started.ID+"/cancel", "")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, StateCancelled, cancelled.State)

	req := httptest.NewRequestWithContext(as(testOwner), http.MethodGet, "/api/imports?workspace_id="+testWorkspace, nil)
	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "secret")
	var list []Import
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Len(t, list, 1)
}
