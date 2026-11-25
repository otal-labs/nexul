package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrustedProxies(t *testing.T) {
	got, err := ParseTrustedProxies(" 127.0.0.1, 172.16.0.0/12 ,,::1")
	require.NoError(t, err)
	assert.Equal(t, "[127.0.0.1/32 172.16.0.0/12 ::1/128]", fmt.Sprint(got))

	_, err = ParseTrustedProxies("127.0.0.1,proxy.example.com")
	require.ErrorContains(t, err, `"proxy.example.com"`)
}

func TestTrustProxies_ClientAddr(t *testing.T) {
	trusted, err := ParseTrustedProxies("127.0.0.1,10.0.0.0/8")
	require.NoError(t, err)
	tests := []struct {
		name      string
		peer      string
		forwarded []string
		want      string
	}{
		{"a trusted peer names the client", "127.0.0.1:5000", []string{"203.0.113.7"}, "203.0.113.7"},
		{"the rightmost untrusted hop wins over one the client wrote", "127.0.0.1:5000", []string{"198.51.100.1, 203.0.113.7"}, "203.0.113.7"},
		{"trusted hops are skipped", "127.0.0.1:5000", []string{"203.0.113.7, 10.1.2.3"}, "203.0.113.7"},
		{"split headers read as one list", "127.0.0.1:5000", []string{"198.51.100.1", "203.0.113.7"}, "203.0.113.7"},
		{"an IPv4-mapped peer is still trusted", "[::ffff:127.0.0.1]:5000", []string{"203.0.113.7"}, "203.0.113.7"},
		{"an untrusted peer's header is ignored", "198.51.100.9:5000", []string{"203.0.113.7"}, "198.51.100.9"},
		{"no header keeps the peer", "127.0.0.1:5000", nil, "127.0.0.1"},
		{"a garbled hop keeps the peer", "127.0.0.1:5000", []string{"203.0.113.7, not-an-ip"}, "127.0.0.1"},
		{"only trusted hops keep the peer", "127.0.0.1:5000", []string{"10.0.0.5"}, "127.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := TrustProxies(trusted, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = ClientAddr(r) }))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.peer
			for _, v := range tt.forwarded {
				req.Header.Add("X-Forwarded-For", v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTrustProxies_NoneConfigured_IgnoresTheHeader(t *testing.T) {
	var got string
	h := TrustProxies(nil, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = ClientAddr(r) }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:5000"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	h.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "127.0.0.1", got)
}
