package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The license routes reach the use-case through the real router: a key the built-in issuer did not sign is refused on
// its field, and a removal is audited like any other write.
func TestLicenseRoutes_ThroughTheRouter(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	_, token := signedInOwner(t, store, svc)

	rec := callAPI(t, h, http.MethodGet, "/api/instance/license", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"state":"none","licensee":"","instance_url":"","features":[]}`, rec.Body.String())

	rec = callAPI(t, h, http.MethodPut, "/api/instance/license", token, `{"key":"nexul-license-1.e30.AAAA"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"key":["this key's signature does not check out`)

	rec = callAPI(t, h, http.MethodDelete, "/api/instance/license", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var audited []string
	for _, row := range auditRows(t, store) {
		audited = append(audited, row.Action)
	}
	assert.Contains(t, audited, "DELETE /api/instance/license")
	assert.False(t, svc.licenseSvc.Enabled(t.Context(), "sso"))
}
