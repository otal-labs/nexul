package main

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/fstest"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/server/webui"
)

// newLogsMux mirrors buildRoutes' tail: the logs proxy, then the web UI catch-all.
func newLogsMux(t *testing.T, upstream string) *httptest.Server {
	t.Helper()
	var target *url.URL
	if upstream != "" {
		u, err := url.Parse(upstream)
		require.NoError(t, err)
		target = u
	}
	mux := httpx.NewServeMux()
	mountLogsProxy(mux, target, slog.New(slog.DiscardHandler))
	mux.Handle("/", webui.Handler(fstest.MapFS{"index.html": {Data: []byte("<html>spa</html>")}}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestLogsProxy_ForwardsPathAndQueryUnchanged(t *testing.T) {
	var gotURI, gotHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURI, gotHost = r.RequestURI, r.Host
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(upstream.Close)
	srv := newLogsMux(t, upstream.URL)

	resp, err := http.Get(srv.URL + "/openobserve/api/default/_search?type=logs&q=a%20b")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "/openobserve/api/default/_search?type=logs&q=a%20b", gotURI)
	assert.Equal(t, srv.Listener.Addr().String(), gotHost)
}

func TestLogsProxy_StreamsBeforeUpstreamFinishes(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "first\n")
		w.(http.Flusher).Flush()
		<-release
		_, _ = io.WriteString(w, "second\n")
	}))
	t.Cleanup(upstream.Close)
	srv := newLogsMux(t, upstream.URL)

	resp, err := http.Get(srv.URL + "/openobserve/stream")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	body := bufio.NewReader(resp.Body)
	line, err := body.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "first\n", line)

	close(release)
	line, err = body.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "second\n", line)
}

func TestLogsProxy_UpgradesWebSocket(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.CloseNow() }()
		typ, msg, err := c.Read(r.Context())
		if err != nil {
			return
		}
		_ = c.Write(r.Context(), typ, msg)
	}))
	t.Cleanup(upstream.Close)
	srv := newLogsMux(t, upstream.URL)

	c, _, err := websocket.Dial(t.Context(), "ws"+srv.URL[len("http"):]+"/openobserve/ws", nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.CloseNow() })
	require.NoError(t, c.Write(t.Context(), websocket.MessageText, []byte("ping")))
	_, msg, err := c.Read(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "ping", string(msg))
}

func TestLogsProxy_UpstreamDown_Returns502(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	down := upstream.URL
	upstream.Close()
	srv := newLogsMux(t, down)

	resp, err := http.Get(srv.URL + "/openobserve/web/")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestLogsProxy_Unset_NoRouteAndNoSPAFallback(t *testing.T) {
	srv := newLogsMux(t, "")

	spa, err := http.Get(srv.URL + "/tickets")
	require.NoError(t, err)
	_ = spa.Body.Close()
	require.Equal(t, http.StatusOK, spa.StatusCode, "the SPA fallback answers other paths")

	for _, path := range []string{"/openobserve", "/openobserve/web/"} {
		resp, err := http.Get(srv.URL + path)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, path)
	}
}
