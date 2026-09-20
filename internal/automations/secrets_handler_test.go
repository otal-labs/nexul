package automations

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secretValueMarker = "sk-live-super-secret-marker-value"

func newTestSecretsHandler(t *testing.T) http.Handler {
	t.Helper()
	svc := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
	return NewSecretsHandler(svc).Routes()
}

func TestSecretsHandler_SetListDelete(t *testing.T) {
	h := newTestSecretsHandler(t)

	rec := serve(t, h, "owner", http.MethodPut, "/api/automation-secrets/API_KEY", `{"value":"`+secretValueMarker+`"}`)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.NotContains(t, rec.Body.String(), secretValueMarker, "the value must never echo back in the set response")

	t.Run("list shows the name, never the value", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodGet, "/api/automation-secrets", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), secretValueMarker, "a secret value must never appear in any response body")
		var list []SecretMeta
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
		assert.Equal(t, "API_KEY", list[0].Name)
	})

	t.Run("delete removes it", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodDelete, "/api/automation-secrets/API_KEY", "")
		require.Equal(t, http.StatusNoContent, rec.Code)
		rec = serve(t, h, "owner", http.MethodGet, "/api/automation-secrets", "")
		var list []SecretMeta
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Empty(t, list)
	})

	t.Run("no actor is 401", func(t *testing.T) {
		rec := serve(t, h, "", http.MethodGet, "/api/automation-secrets", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-privileged actor is 403 on set", func(t *testing.T) {
		rec := serve(t, h, "alice", http.MethodPut, "/api/automation-secrets/API_KEY", `{"value":"x"}`)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid name is 400 and never echoes the attempted value", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPut, "/api/automation-secrets/bad-name", `{"value":"`+secretValueMarker+`"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.NotContains(t, rec.Body.String(), secretValueMarker)
	})

	t.Run("malformed set body is 400 and never echoes the raw body", func(t *testing.T) {
		rec := serve(t, h, "owner", http.MethodPut, "/api/automation-secrets/API_KEY", `{not json`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("non-privileged actor is 403 on delete", func(t *testing.T) {
		rec := serve(t, h, "alice", http.MethodDelete, "/api/automation-secrets/API_KEY", "")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("non-privileged actor is 403 on list", func(t *testing.T) {
		rec := serve(t, h, "alice", http.MethodGet, "/api/automation-secrets", "")
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
