package freshdns

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixed struct {
	addrs []string
	err   error
	calls int
}

func (f *fixed) LookupHost(context.Context, string) ([]string, error) {
	f.calls++
	return f.addrs, f.err
}

func TestResolver_LookupHost(t *testing.T) {
	tests := []struct {
		name         string
		public       *fixed
		system       *fixed
		want         []string
		wantErr      bool
		systemCalled bool
	}{
		{"public answers first", &fixed{addrs: []string{"1.2.3.4"}}, &fixed{addrs: []string{"9.9.9.9"}}, []string{"1.2.3.4"}, false, false},
		{"a private name falls back to the host", &fixed{err: errors.New("no such host")}, &fixed{addrs: []string{"10.0.0.5"}}, []string{"10.0.0.5"}, false, true},
		{"an empty public answer falls back", &fixed{}, &fixed{addrs: []string{"10.0.0.5"}}, []string{"10.0.0.5"}, false, true},
		{"neither knows it", &fixed{err: errors.New("no such host")}, &fixed{err: errors.New("no such host")}, nil, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Resolver{Public: tt.public, System: tt.system}
			got, err := r.LookupHost(t.Context(), "nexul.example.com")
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, tt.systemCalled, tt.system.calls > 0)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.systemCalled, tt.system.calls > 0)
		})
	}
}

func TestResolver_ClientDialsTheResolvedAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))
	t.Cleanup(srv.Close)
	host, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)
	r := &Resolver{Public: &fixed{addrs: []string{host}}, System: &fixed{err: errors.New("unused")}}

	resp, err := r.Client(5 * time.Second).Get("http://nexul.example.com:" + port + "/")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	assert.Equal(t, http.StatusTeapot, resp.StatusCode)
}

func TestResolver_DialContext_Errors(t *testing.T) {
	r := &Resolver{Public: &fixed{err: errors.New("no")}, System: &fixed{err: errors.New("no such host")}}
	_, err := r.DialContext(t.Context(), "tcp", "missing-port")
	require.Error(t, err)
	_, err = r.DialContext(t.Context(), "tcp", "nexul.example.com:443")
	require.ErrorContains(t, err, "no such host")

	unreachable := &Resolver{Public: &fixed{addrs: []string{"127.0.0.1"}}, System: &fixed{}}
	_, err = unreachable.DialContext(t.Context(), "tcp", "nexul.example.com:1")
	require.Error(t, err, "nothing listens on port 1")
}

func TestNew_WiresBothResolvers(t *testing.T) {
	r := New()
	assert.NotNil(t, r.Public)
	assert.Same(t, net.DefaultResolver, r.System)
}
