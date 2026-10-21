package t3rpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

func sessionServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/auth/session", r.URL.Path)
		assert.Equal(t, "Bearer bearer-xyz", r.Header.Get("Authorization"))
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCanClone_ReadsTheScopesT3GrantedTheSession(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"the full list names it", 200, `{"authenticated":true,"permissions":["orchestration:operate","source-control:write"],"scopes":["orchestration:operate"]}`, true},
		{"the full list lacks it", 200, `{"authenticated":true,"permissions":["orchestration:operate","review:write"]}`, false},
		{"an empty grant holds nothing", 200, `{"authenticated":true,"permissions":[]}`, false},
		{"the legacy names alone say nothing", 200, `{"scopes":["orchestration:operate"]}`, true},
		{"neither field is unknown, not lacking", 200, `{"authenticated":true}`, true},
		{"a T3 without the endpoint is unknown", 404, `not found`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := sessionServer(t, tt.status, tt.body)
			got, err := CanClone(t.Context(), srv.Client(), srv.URL, "bearer-xyz")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCanClone_FailuresAreErrorsNotAnAnswer(t *testing.T) {
	t.Parallel()
	srv := sessionServer(t, http.StatusUnauthorized, "revoked")
	_, err := CanClone(t.Context(), srv.Client(), srv.URL, "bearer-xyz")
	require.ErrorIs(t, err, harness.ErrSessionRejected)

	srv = sessionServer(t, http.StatusInternalServerError, "boom")
	_, err = CanClone(t.Context(), srv.Client(), srv.URL, "bearer-xyz")
	require.ErrorContains(t, err, "500")

	srv = sessionServer(t, http.StatusOK, "<html>")
	_, err = CanClone(t.Context(), srv.Client(), srv.URL, "bearer-xyz")
	require.ErrorContains(t, err, "decode T3 session")
}
