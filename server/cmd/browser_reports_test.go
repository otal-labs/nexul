package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserReports_AreTakenWithoutASession(t *testing.T) {
	t.Parallel()
	router, _, _ := newRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/browser-reports", strings.NewReader(`[{"type":"crash","url":"https://example.com/","body":{"reason":"unresponsive"}}]`))
	req.Header.Set("Content-Type", "application/reports+json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}
