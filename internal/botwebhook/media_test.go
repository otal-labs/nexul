package botwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// pngBytes is enough of a PNG for http.DetectContentType to name it.
var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)

// mediaHost stands in for a sender's public image host and records what each request carried.
type mediaHost struct {
	*httptest.Server
	mu       sync.Mutex
	requests []*http.Request
}

func newMediaHost(t *testing.T) *mediaHost {
	t.Helper()
	h := &mediaHost{}
	mux := http.NewServeMux()
	mux.HandleFunc("/chart.png", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(pngBytes) })
	mux.HandleFunc("/moved.png", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/chart.png", http.StatusFound) })
	mux.HandleFunc("/hop", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n == 0 {
			_, _ = w.Write(pngBytes)
			return
		}
		http.Redirect(w, r, "/hop?n="+strconv.Itoa(n-1), http.StatusFound)
	})
	mux.HandleFunc("/to", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.Query().Get("u"), http.StatusFound)
	})
	mux.HandleFunc("/page.html", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<!DOCTYPE html><p>hi</p>")) })
	mux.HandleFunc("/logo.svg", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`))
	})
	mux.HandleFunc("/huge.png", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(append(pngBytes, make([]byte, mediaMaxBytes)...))
	})
	h.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.requests = append(h.requests, r.Clone(context.Background()))
		h.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(h.Close)
	return h
}

func (h *mediaHost) hits() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.requests)
}

func (h *mediaHost) last() *http.Request {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.requests[len(h.requests)-1]
}

// testClient is the production guard plus the media host's own loopback address, standing in for a public one, and
// a fake DNS: hosts named in names dial the address mapped to them, as a resolver answering that would.
func (h *mediaHost) testClient(names map[string]string) *http.Client {
	public := netip.MustParseAddrPort(h.Listener.Addr().String())
	d := guardedDialer(func(ap netip.AddrPort) bool { return ap == public || publicAddr(ap) })
	return newMediaClient(func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if ip, ok := names[host]; ok {
			addr = net.JoinHostPort(ip, port)
		}
		return d.DialContext(ctx, network, addr)
	})
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

// TestFetchMedia_Guard refuses every non-public destination at the dial, whether named directly, by a hostname
// resolving to one, or behind a redirect from a public host, while the same client still fetches the public host.
func TestFetchMedia_Guard(t *testing.T) {
	host := newMediaHost(t)
	names := map[string]string{"media.example.com": "10.1.2.3", "internal.example.com": "192.168.1.10"}
	client := host.testClient(names)
	production := newMediaClient(guardedDialer(publicAddr).DialContext)

	_, data, err := fetchMedia(t.Context(), client, mustURL(t, host.URL+"/chart.png"))
	require.NoError(t, err, "the public host itself is reachable")
	assert.Equal(t, pngBytes, data)

	for name, tc := range map[string]struct {
		client *http.Client
		url    string
	}{
		"loopback with a live image": {production, host.URL + "/chart.png"},
		"RFC 1918":                   {production, "http://10.0.0.1/a.png"},
		"CGNAT":                      {production, "http://100.64.0.1/a.png"},
		"cloud metadata":             {production, "http://169.254.169.254/latest/meta-data/"},
		"IPv6 loopback":              {production, "http://[::1]:" + strconv.Itoa(host.Listener.Addr().(*net.TCPAddr).Port) + "/chart.png"},
		"IPv6 unique local":          {production, "http://[fc00::1]/a.png"},
		"IPv4-mapped IPv6":           {production, "http://[::ffff:10.0.0.1]/a.png"},
		"hostname to a private IP":   {client, "http://media.example.com/a.png"},
		"redirect to metadata":       {client, host.URL + "/to?u=" + url.QueryEscape("http://169.254.169.254/latest/meta-data/")},
		"redirect to a private name": {client, host.URL + "/to?u=" + url.QueryEscape("http://internal.example.com/a.png")},
	} {
		before := host.hits()
		_, _, err := fetchMedia(t.Context(), tc.client, mustURL(t, tc.url))
		require.ErrorIs(t, err, errMediaRefused, name)
		assert.Contains(t, err.Error(), "is not a public address", name)
		assert.NotContains(t, err.Error(), "meta-data", "%s: the refusal never repeats the URL", name)
		if name == "loopback with a live image" {
			assert.Equal(t, before, host.hits(), "a refused dial never reaches the host")
		}
	}
}

// TestFetchMedia_Response serves only a raster image of at most 8 MiB, three redirects deep at most, and sends
// neither a Referer nor a cookie, under the proxy's own User-Agent.
func TestFetchMedia_Response(t *testing.T) {
	host := newMediaHost(t)
	client := host.testClient(nil)

	contentType, data, err := fetchMedia(t.Context(), client, mustURL(t, host.URL+"/moved.png"))
	require.NoError(t, err)
	assert.Equal(t, "image/png", contentType)
	assert.Equal(t, pngBytes, data)
	hop := host.last()
	assert.Equal(t, "/chart.png", hop.URL.Path)
	assert.Empty(t, hop.Header.Get("Referer"), "the redirect hop carries no Referer")
	assert.Empty(t, hop.Header.Get("Cookie"))
	assert.Equal(t, "Nexul-Media-Proxy", hop.Header.Get("User-Agent"))

	_, _, err = fetchMedia(t.Context(), client, mustURL(t, host.URL+"/hop?n=3"))
	require.NoError(t, err, "three redirects are followed")

	for name, path := range map[string]string{
		"four redirects": "/hop?n=4",
		"HTML":           "/page.html",
		"SVG":            "/logo.svg",
		"over 8 MiB":     "/huge.png",
		"missing":        "/nothing.png",
	} {
		_, _, err := fetchMedia(t.Context(), client, mustURL(t, host.URL+path))
		require.ErrorIs(t, err, errMediaRefused, name)
	}
}

// fakeMessages is chat's stored messages, read ungated.
type fakeMessages map[string]*PostedMessage

func (f fakeMessages) BotMessage(_ context.Context, id string) (*PostedMessage, error) {
	m, ok := f[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return m, nil
}

// readGate is chat's read rule: outsider reads no conversation.
type readGate struct {
	fakeConversations
	outsider string
}

func (g readGate) Conversation(ctx context.Context, id string) (*Conversation, error) {
	if actor, _ := identity.ActorFromCtx(ctx); actor.ID == g.outsider {
		return nil, apperrs.ErrNotFound
	}
	return g.fakeConversations.Conversation(ctx, id)
}

func embedsJSON(t *testing.T, embeds ...Embed) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(embeds)
	require.NoError(t, err)
	return raw
}

// TestHandler_Media proxies only an image the named message shows to a caller who reads its conversation, with
// the safety headers on the bytes, and answers a refused fetch with 502.
func TestHandler_Media(t *testing.T) {
	host := newMediaHost(t)
	chart, avatar, icon := host.URL+"/chart.png", host.URL+"/moved.png", host.URL+"/hop?n=1"
	s := newTestService(newFakeRepo())
	s.conversations = readGate{fakeConversations: fakeConversations{"c-eng": {ID: "c-eng", WorkspaceID: "w-acme"}}, outsider: everyone}
	s.messages = fakeMessages{
		"m-1": {ConversationID: "c-eng", AvatarURL: avatar, Embeds: embedsJSON(t,
			Embed{Title: "Build", Image: &EmbedImage{URL: chart}, Footer: &EmbedFooter{Text: "CI", IconURL: icon}},
			Embed{Title: "Deploy", Thumbnail: &EmbedImage{URL: host.URL + "/page.html"}, Author: &EmbedAuthor{Name: "alice", IconURL: host.URL + "/logo.svg"}},
		)},
		"m-gone": {ConversationID: "c-eng", Embeds: embedsJSON(t, Embed{Image: &EmbedImage{URL: chart}}), Deleted: true},
	}
	routes := (&Handler{svc: s, mediaClient: host.testClient(nil)}).Routes()
	get := func(user, message, raw string) *httptest.ResponseRecorder {
		target := "/api/botwebhooks/media?message=" + url.QueryEscape(message) + "&url=" + url.QueryEscape(raw)
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequestWithContext(as(user), http.MethodGet, target, nil))
		return rec
	}

	for _, raw := range []string{chart, avatar, icon} {
		rec := get(reader, "m-1", raw)
		require.Equal(t, http.StatusOK, rec.Code, raw)
		assert.Equal(t, pngBytes, rec.Body.Bytes())
		assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "default-src 'none'", rec.Header().Get("Content-Security-Policy"))
		assert.Equal(t, "private, max-age=3600", rec.Header().Get("Cache-Control"))
	}

	hits := host.hits()
	assert.Equal(t, http.StatusNotFound, get(reader, "m-1", host.URL+"/other.png").Code, "a URL the message does not show")
	assert.Equal(t, http.StatusNotFound, get(reader, "m-1", "").Code)
	assert.Equal(t, http.StatusNotFound, get(everyone, "m-1", chart).Code, "a caller who cannot read the conversation")
	assert.Equal(t, http.StatusNotFound, get(reader, "m-gone", chart).Code, "a deleted message")
	assert.Equal(t, http.StatusNotFound, get(reader, "m-missing", chart).Code)
	assert.Equal(t, hits, host.hits(), "no refused request reaches the sender's host")

	assert.Equal(t, http.StatusBadGateway, get(reader, "m-1", host.URL+"/page.html").Code, "a thumbnail that is not an image")
	assert.Equal(t, http.StatusBadGateway, get(reader, "m-1", host.URL+"/logo.svg").Code, "an SVG author icon")
}
