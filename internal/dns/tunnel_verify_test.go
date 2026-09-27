package dns

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// answering is an HTTPS server every hostname resolves to, answering with status.
func answering(t *testing.T, status int) *http.Client {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
	t.Cleanup(srv.Close)
	tr := srv.Client().Transport.(*http.Transport).Clone()
	tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // test server's self-signed certificate
	return &http.Client{Transport: tr}
}

// routedTunnel is a tunnel routed to app.example.com, served by a server answering with status.
func routedTunnel(t *testing.T, status int) (*Service, *fakeProvider, *Tunnel) {
	t.Helper()
	repo := newFakeRepo()
	require.NoError(t, repo.SaveTunnel(t.Context(), Tunnel{ID: "t1", Name: "prod"}))
	s := newTunnelService(repo, newFakeTunnelProvider(), nil)
	s.httpc = answering(t, status)
	routed, err := s.RouteTunnelHostname(t.Context(), RouteTunnelInput{
		TunnelID: "t1", Hostname: "app.example.com", ZoneID: "z1", Zone: "example.com", Service: "http://web:80",
	})
	require.NoError(t, err)
	return s, s.provider.(*fakeProvider), routed
}

func TestVerifyTunnelRoute(t *testing.T) {
	t.Run("a routed, proxied, answering hostname passes every check", func(t *testing.T) {
		s, _, _ := routedTunnel(t, http.StatusOK)
		got := s.verifyAll(t.Context(), "t1")
		assert.Equal(t, map[string]string{
			TunnelCheckIngress:   "ok: app.example.com → http://web:80",
			TunnelCheckRecord:    "ok: Proxied CNAME in example.com",
			TunnelCheckReachable: "ok: Answered HTTP 200 over HTTPS",
		}, got)
	})
	t.Run("a 530 from Cloudflare fails reachable", func(t *testing.T) {
		s, _, _ := routedTunnel(t, 530)
		_, err := s.VerifyTunnelRoute(t.Context(), "t1", TunnelCheckReachable)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.ErrorContains(t, err, "answered HTTP 530")
	})
	t.Run("a hostname that does not answer fails reachable", func(t *testing.T) {
		s, _, _ := routedTunnel(t, http.StatusOK)
		s.httpc = &http.Client{Transport: &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
			return nil, &net.DNSError{Err: "no such host", Name: "app.example.com", IsNotFound: true}
		}}}
		_, err := s.VerifyTunnelRoute(t.Context(), "t1", TunnelCheckReachable)
		assert.ErrorContains(t, err, "does not answer yet")
	})
	t.Run("a DNS-only record fails the record check", func(t *testing.T) {
		s, dnsp, _ := routedTunnel(t, http.StatusOK)
		dnsp.records["z1"][0].Proxied = false
		_, err := s.VerifyTunnelRoute(t.Context(), "t1", TunnelCheckRecord)
		assert.ErrorContains(t, err, "must be proxied")
	})
	t.Run("a record pointing elsewhere fails the record check", func(t *testing.T) {
		s, dnsp, _ := routedTunnel(t, http.StatusOK)
		dnsp.records["z1"][0].Content = "other.cfargotunnel.com"
		_, err := s.VerifyTunnelRoute(t.Context(), "t1", TunnelCheckRecord)
		assert.ErrorContains(t, err, "not this tunnel")
	})
	t.Run("a missing record fails the record check", func(t *testing.T) {
		s, dnsp, _ := routedTunnel(t, http.StatusOK)
		dnsp.records["z1"] = nil
		_, err := s.VerifyTunnelRoute(t.Context(), "t1", TunnelCheckRecord)
		assert.ErrorContains(t, err, "no DNS record")
	})
	t.Run("a missing ingress rule fails the ingress check", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(t.Context(), Tunnel{ID: "t1", Name: "prod", Hostname: "app.example.com"}))
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		_, err := s.VerifyTunnelRoute(t.Context(), "t1", TunnelCheckIngress)
		assert.ErrorContains(t, err, "no ingress rule for app.example.com")
	})
	t.Run("an unrouted tunnel and an unknown check are invalid", func(t *testing.T) {
		repo := newFakeRepo()
		require.NoError(t, repo.SaveTunnel(t.Context(), Tunnel{ID: "t1", Name: "prod"}))
		s := newTunnelService(repo, newFakeTunnelProvider(), nil)
		_, err := s.VerifyTunnelRoute(t.Context(), "t1", TunnelCheckIngress)
		assert.ErrorContains(t, err, "no hostname routed yet")

		s2, _, _ := routedTunnel(t, http.StatusOK)
		_, err = s2.VerifyTunnelRoute(t.Context(), "t1", "bogus")
		assert.ErrorContains(t, err, `unknown tunnel check "bogus"`)
		_, err = s2.VerifyTunnelRoute(t.Context(), "missing", TunnelCheckIngress)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("verifyAll reports failures per check", func(t *testing.T) {
		s, _, _ := routedTunnel(t, 530)
		got := s.verifyAll(t.Context(), "t1")
		assert.Contains(t, got[TunnelCheckReachable], "failed: ")
		assert.Contains(t, got[TunnelCheckIngress], "ok: ")
	})
}

func TestHandler_VerifyTunnelRoute(t *testing.T) {
	s, _, _ := routedTunnel(t, http.StatusOK)
	mux := NewHandler(s).Routes()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/dns/tunnels/t1/verify?check=record", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"detail":"Proxied CNAME in example.com"}`, rec.Body.String())

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/dns/tunnels/t1/verify?check=bogus", nil))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
