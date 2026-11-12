package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Branding reaches the use-case through the real router: the look is public and empty without a key unlocking it, the
// Owner's reads work, and a write is refused until a key arrives and audited like any other write.
func TestBrandingRoutes_ThroughTheRouter(t *testing.T) {
	t.Parallel()
	h, store, svc := newRouter(t)
	_, token := signedInOwner(t, store, svc)

	rec := callAPI(t, h, http.MethodGet, "/api/branding", "", "")
	require.Equal(t, http.StatusOK, rec.Code, "the sign-in page reads it before anyone has signed in: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"enabled":false`)

	rec = callAPI(t, h, http.MethodGet, "/api/branding/logo", "", "")
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = callAPI(t, h, http.MethodGet, "/api/instance/branding", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"unlocked":false`)

	rec = callAPI(t, h, http.MethodPatch, "/api/instance/branding", token, `{"product_name":"Acme Hub"}`)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "license key")

	rec = callAPI(t, h, http.MethodGet, "/api/instance/branding", "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "the Owner's settings need a session")

	var audited []string
	for _, row := range auditRows(t, store) {
		audited = append(audited, row.Action)
	}
	assert.Contains(t, audited, "PATCH /api/instance/branding")
}
