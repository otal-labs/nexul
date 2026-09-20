package automations

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

func serve(t *testing.T, h http.Handler, actorID, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if actorID != "" {
		req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: actorID}))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_CreateAndList(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	h := NewHandler(svc).Routes()

	rec := serve(t, h, "owner", http.MethodPost, "/api/automations", `{"name":"My automation","scopes":["tickets:read"]}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.NotContains(t, rec.Body.String(), "token_hash", "the stored hash must never reach the wire")
	var created tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.NotEmpty(t, created.Token)
	assert.Equal(t, "My automation", created.Automation.Name)

	t.Run("list includes it", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []Automation
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
	})
	t.Run("no actor is 401", func(t *testing.T) {
		rec := serve(t, h, "", http.MethodGet, "/api/automations", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
	t.Run("non-privileged actor is 403", func(t *testing.T) {
		rec := serve(t, h, "alice", http.MethodGet, "/api/automations", "")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandler_GetUpdateEnableDelete(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	h := NewHandler(svc).Routes()
	rec := serve(t, h, "owner", http.MethodPost, "/api/automations", `{"name":"x","scopes":["tickets:read"]}`)
	var created tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	id := created.Automation.ID

	t.Run("get by id", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automations/"+id, "")
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("update config", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPatch, "/api/automations/"+id+"/config", `{"config_values":{"a":1}}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var a Automation
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &a))
		assert.JSONEq(t, `{"a":1}`, string(a.ConfigValues))
	})
	t.Run("enable", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPatch, "/api/automations/"+id+"/enabled", `{"enabled":true}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var a Automation
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &a))
		assert.True(t, a.Enabled)
	})
	t.Run("delete", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodDelete, "/api/automations/"+id, "")
		assert.Equal(t, http.StatusNoContent, rec.Code)
		rec = serve(t, h, "owner", http.MethodGet, "/api/automations/"+id, "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_TokenMintAndRevoke(t *testing.T) {
	svc := newTestService(newFakeRepo(), allowAll("owner"))
	h := NewHandler(svc).Routes()
	rec := serve(t, h, "owner", http.MethodPost, "/api/automations", `{"name":"x","scopes":["tickets:read"]}`)
	var created tokenResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	id := created.Automation.ID

	t.Run("mint returns a fresh token", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPost, "/api/automations/"+id+"/token", "")
		require.Equal(t, http.StatusCreated, rec.Code)
		var minted tokenResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &minted))
		assert.NotEqual(t, created.Token, minted.Token)
	})
	t.Run("revoke marks it revoked", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodDelete, "/api/automations/"+id+"/token", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var a Automation
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &a))
		assert.NotNil(t, a.TokenRevokedAt)
	})
}
