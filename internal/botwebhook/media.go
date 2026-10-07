package botwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Messages reads one chat message with no gate of its own; the media proxy checks its conversation as the caller.
type Messages interface {
	BotMessage(ctx context.Context, id string) (*PostedMessage, error)
}

// PostedMessage is what the media proxy needs of a stored message: where it lives and the images it may load.
type PostedMessage struct {
	ConversationID string
	AvatarURL      string
	Embeds         json.RawMessage
	Deleted        bool
}

// mediaURLs are the external images the message shows: its avatar override and every embed's pictures.
func (m *PostedMessage) mediaURLs() ([]string, error) {
	var embeds []Embed
	if len(m.Embeds) > 0 {
		if err := json.Unmarshal(m.Embeds, &embeds); err != nil {
			return nil, fmt.Errorf("decode stored embeds: %w", err)
		}
	}
	out := []string{m.AvatarURL}
	for _, e := range embeds {
		if e.Image != nil {
			out = append(out, e.Image.URL)
		}
		if e.Thumbnail != nil {
			out = append(out, e.Thumbnail.URL)
		}
		if e.Author != nil {
			out = append(out, e.Author.IconURL)
		}
		if e.Footer != nil {
			out = append(out, e.Footer.IconURL)
		}
	}
	return out, nil
}

// MediaURL returns raw for the proxy to fetch only when the caller reads messageID's conversation and the message
// shows raw as one of its images, so the proxy never fetches a URL a reader did not already see.
func (s *Service) MediaURL(ctx context.Context, messageID, raw string) (*url.URL, error) {
	m, err := s.messages.BotMessage(ctx, strings.TrimSpace(messageID))
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", messageID, err)
	}
	if _, err := s.conversations.Conversation(ctx, m.ConversationID); err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", m.ConversationID, err)
	}
	urls, err := m.mediaURLs()
	if err != nil {
		return nil, err
	}
	if m.Deleted || raw == "" || !slices.Contains(urls, raw) {
		return nil, fmt.Errorf("%w: message %s shows no such image", apperrs.ErrNotFound, messageID)
	}
	if err := httpURL("url", raw); err != nil {
		return nil, err
	}
	return url.Parse(raw)
}

const (
	mediaTimeout   = 10 * time.Second
	mediaMaxBytes  = 8 << 20
	mediaRedirects = 3
)

var errMediaRefused = errors.New("media refused")

// nonPublic are the ranges netip's own checks leave open: CGNAT, the documentation and benchmark nets, reserved
// space, and IPv6 prefixes that embed an IPv4 address (NAT64, Teredo, 6to4).
var nonPublic = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("fec0::/10"),
}

// publicAddr is false for loopback, private (RFC 1918, ULA), link-local (cloud metadata), multicast, and nonPublic.
func publicAddr(ap netip.AddrPort) bool {
	ip := ap.Addr().Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	return !slices.ContainsFunc(nonPublic, func(p netip.Prefix) bool { return p.Contains(ip) })
}

// guardedDialer checks the address it is about to connect to, after DNS, so neither a redirect nor a rebinding
// answer reaches a host allow refuses.
func guardedDialer(allow func(netip.AddrPort) bool) *net.Dialer {
	return &net.Dialer{Timeout: mediaTimeout, Control: func(_, address string, _ syscall.RawConn) error {
		ap, err := netip.ParseAddrPort(address)
		if err != nil || !allow(ap) {
			return fmt.Errorf("%w: %s is not a public address", errMediaRefused, address)
		}
		return nil
	}}
}

// newMediaClient sends no cookies, no Referer, and no proxy hop, which would put the proxy's address past the guard.
func newMediaClient(dial func(ctx context.Context, network, addr string) (net.Conn, error)) *http.Client {
	return &http.Client{
		Timeout:   mediaTimeout,
		Transport: &http.Transport{DialContext: dial, Proxy: nil, TLSHandshakeTimeout: mediaTimeout, ResponseHeaderTimeout: mediaTimeout},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > mediaRedirects {
				return fmt.Errorf("%w: more than %d redirects", errMediaRefused, mediaRedirects)
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("%w: a redirect to %s", errMediaRefused, req.URL.Scheme)
			}
			req.Header.Del("Referer")
			return nil
		},
	}
}

// fetchMedia returns u's bytes and their sniffed type when they are a raster image of at most mediaMaxBytes.
func fetchMedia(ctx context.Context, client *http.Client, u *url.URL) (string, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", errMediaRefused, err)
	}
	req.Header.Set("User-Agent", "Nexul-Media-Proxy")
	req.Header.Set("Accept", "image/*")
	resp, err := client.Do(req)
	// The url.Error around a failure repeats the whole URL, which may carry the sender's own signed query.
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", errMediaRefused, err)
	}
	defer func() { _ = resp.Body.Close() }() // a read-only body; nothing is lost if closing it fails
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("%w: the host answered %d", errMediaRefused, resp.StatusCode)
	}
	if resp.ContentLength > mediaMaxBytes {
		return "", nil, fmt.Errorf("%w: %d bytes, over the %d MiB cap", errMediaRefused, resp.ContentLength, mediaMaxBytes>>20)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, mediaMaxBytes+1))
	if err != nil {
		return "", nil, fmt.Errorf("%w: %w", errMediaRefused, err)
	}
	if len(data) > mediaMaxBytes {
		return "", nil, fmt.Errorf("%w: over the %d MiB cap", errMediaRefused, mediaMaxBytes>>20)
	}
	// The bytes decide, never the sender's header; DetectContentType never names SVG, so an SVG is refused as text.
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return "", nil, fmt.Errorf("%w: %s is not a raster image", errMediaRefused, contentType)
	}
	return contentType, data, nil
}

// media proxies an image a bot message shows, so its sender never sees who reads the message or when.
func (h *Handler) media(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	u, err := h.svc.MediaURL(r.Context(), q.Get("message"), q.Get("url"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), mediaTimeout)
	defer cancel()
	contentType, data, err := fetchMedia(ctx, h.mediaClient, u)
	if err != nil {
		logging.FromCtx(r.Context()).Debug("bot media refused", "message_id", q.Get("message"), "host", u.Hostname(), "reason", err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // the status is already sent; a failed write only means the client went away
}
