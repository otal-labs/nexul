package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
)

// TestOAuthRegister_BehindTrustedProxy_ThrottlesEachClientNotTheProxy drives the real router: the 60-an-hour
// registration limit counts the client a trusted proxy names, and an untrusted peer's header changes nothing.
func TestOAuthRegister_BehindTrustedProxy_ThrottlesEachClientNotTheProxy(t *testing.T) {
	t.Parallel()
	h, _, _ := routerWith(t, migrated.DB(t), func(cfg *config.Config) {
		var err error
		cfg.TrustedProxies, err = httpx.ParseTrustedProxies("172.17.0.0/16")
		require.NoError(t, err)
	})
	register := func(peer, forwarded string) int {
		req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(`{"redirect_uris":["http://localhost:4000/cb"]}`))
		req.RemoteAddr = peer + ":40000"
		req.Header.Set("X-Forwarded-For", forwarded)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	for range 60 {
		require.Equal(t, http.StatusCreated, register("172.17.0.2", "203.0.113.7"))
	}
	assert.Equal(t, http.StatusTooManyRequests, register("172.17.0.2", "203.0.113.7"), "one client behind the proxy runs out")
	assert.Equal(t, http.StatusCreated, register("172.17.0.2", "203.0.113.8"), "another client behind the same proxy does not")

	for range 60 {
		require.Equal(t, http.StatusCreated, register("198.51.100.9", "203.0.113.50"))
	}
	assert.Equal(t, http.StatusTooManyRequests, register("198.51.100.9", "203.0.113.51"), "an untrusted peer's spoofed header does not get it a fresh bucket")
}
