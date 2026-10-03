package t3client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
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

func TestHarness_PairAndVersion_T3MovedOn_ReadTheDescriptorBeforeSpendingTheToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		protocol int
		check    func(t *testing.T, err error, host string)
	}{
		{"protocol 2 moves the call to the protocol-2 client", 2, func(t *testing.T, err error, _ string) {
			var moved *harness.MovedError
			require.ErrorAs(t, err, &moved)
			assert.Equal(t, harness.KindT3CodeV2, moved.To)
		}},
		{"past protocol 2 is refused", 3, func(t *testing.T, err error, host string) {
			require.ErrorIs(t, err, harness.ErrProtocol)
			assert.EqualError(t, err, "T3 Code on "+host+" needs a newer Nexul.")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := t3rpctest.New(t)
			f.Protocol = tt.protocol
			host, err := url.Parse(f.URL)
			require.NoError(t, err)
			h := NewHarness(Options{HTTPClient: f.Client()})

			_, err = h.Version(t.Context(), f.URL)
			tt.check(t, err, host.Host)
			_, err = h.Pair(t.Context(), f.URL, f.PairToken)
			tt.check(t, err, host.Host)
			assert.Zero(t, f.Exchanges.Load(), "the one-time token stays unspent for the client that can use it")
		})
	}
}
