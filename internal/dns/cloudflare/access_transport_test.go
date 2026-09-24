package cloudflare

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headerSink records the Access headers each path received.
type headerSink struct {
	mu   sync.Mutex
	seen map[string]http.Header
}

func (h *headerSink) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	h.seen[r.URL.Path] = http.Header{
		"Id":     r.Header.Values("CF-Access-Client-Id"),
		"Secret": r.Header.Values("CF-Access-Client-Secret"),
	}
	h.mu.Unlock()
	if r.URL.Path == "/redirect" {
		http.Redirect(w, r, r.URL.Query().Get("to"), http.StatusFound)
		return
	}
	if r.URL.Path != "/ws" {
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	_ = conn.Close(websocket.StatusNormalClosure, "") // the handshake is all the test needs
}

func (h *headerSink) got(path string) http.Header {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seen[path]
}

// tunnelOnly treats 127.0.0.1 as the one computer tunnel hostname; localhost reaches the same server as a URL-paired machine.
func tunnelOnly(_ context.Context, host string) (string, string, bool, error) {
	if host != "127.0.0.1" {
		return "", "", false, nil
	}
	return "cid.access", "s3cret", true, nil
}

func newSink(t *testing.T) (*headerSink, *url.URL, *http.Client) {
	t.Helper()
	sink := &headerSink{seen: map[string]http.Header{}}
	srv := httptest.NewServer(sink)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return sink, u, &http.Client{Transport: &AccessTransport{Credentials: tunnelOnly}}
}

func get(t *testing.T, client *http.Client, rawURL string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Empty(t, req.Header, "the caller's request is never modified")
}

func TestAccessTransport_HeadersReachOnlyTunnelHostnames(t *testing.T) {
	sink, u, client := newSink(t)
	other := "http://localhost:" + u.Port()

	get(t, client, "http://127.0.0.1:"+u.Port()+"/tunnel")
	get(t, client, other+"/url-paired")
	get(t, client, "http://127.0.0.1:"+u.Port()+"/redirect?to="+url.QueryEscape(other+"/landing"))

	assert.Equal(t, []string{"cid.access"}, sink.got("/tunnel")["Id"])
	assert.Equal(t, []string{"s3cret"}, sink.got("/tunnel")["Secret"])
	assert.Empty(t, sink.got("/url-paired")["Id"])
	assert.Empty(t, sink.got("/url-paired")["Secret"])
	assert.Equal(t, []string{"cid.access"}, sink.got("/redirect")["Id"])
	assert.Empty(t, sink.got("/landing")["Secret"], "a redirect off the tunnel hostname drops the secret")
}

func TestAccessTransport_WebSocketDialsCarryHeadersOnlyToTunnelHostnames(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		wantID []string
	}{
		{"computer tunnel", "127.0.0.1", []string{"cid.access"}},
		{"url-paired machine", "localhost", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink, u, client := newSink(t)
			conn, _, err := websocket.Dial(t.Context(), "ws://"+tt.host+":"+u.Port()+"/ws", &websocket.DialOptions{HTTPClient: client})
			require.NoError(t, err)
			_ = conn.CloseNow() // closing is not what this test checks
			assert.Equal(t, tt.wantID, sink.got("/ws")["Id"])
		})
	}
}

type closeTracker struct {
	io.Reader
	closed bool
}

func (c *closeTracker) Close() error {
	c.closed = true
	return nil
}

func TestAccessTransport_CredentialFailure_SendsNothingAndClosesTheBody(t *testing.T) {
	sink, u, _ := newSink(t)
	boom := errors.New("token store unreachable")
	transport := &AccessTransport{Credentials: func(context.Context, string) (string, string, bool, error) {
		return "", "", false, boom
	}}
	body := &closeTracker{Reader: strings.NewReader("{}")}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, u.String()+"/tunnel", body)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.ErrorIs(t, err, boom)
	assert.Nil(t, resp)
	assert.True(t, body.closed)
	assert.Nil(t, sink.got("/tunnel"), "the request never left")
}
