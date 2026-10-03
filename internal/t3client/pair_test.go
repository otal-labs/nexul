package t3client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

func TestHarness_Pair_ExchangesThenReadsVersion(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "bearer-xyz", "expires_in": 3600})
	})
	mux.HandleFunc("/.well-known/t3/environment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"serverVersion": "0.0.34"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	h := NewHarness(Options{HTTPClient: srv.Client()})
	result, err := h.Pair(context.Background(), srv.URL, "tok")
	require.NoError(t, err)
	assert.Equal(t, harness.PairResult{BearerToken: "bearer-xyz", ExpiresIn: time.Hour, Version: "0.0.34", Kind: harness.KindT3Code}, result)
}
