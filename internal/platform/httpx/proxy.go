package httpx

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// ParseTrustedProxies reads a comma-separated list of IPs and CIDR ranges, the instance's own proxies.
func ParseTrustedProxies(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if p, err := netip.ParsePrefix(part); err == nil {
			out = append(out, p.Masked())
			continue
		}
		addr, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("%q is neither an IP nor a CIDR range", part)
		}
		addr = addr.Unmap()
		out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return out, nil
}

// TrustProxies sets the request's client address from X-Forwarded-For when the peer is one of the trusted proxies,
// reading the header right to left past the trusted hops; any other peer's header is ignored, so it cannot be spoofed.
func TrustProxies(trusted []netip.Prefix, next http.Handler) http.Handler {
	if len(trusted) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client, ok := forwardedClient(trusted, r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		r = r.WithContext(r.Context())
		r.RemoteAddr = net.JoinHostPort(client.String(), "0")
		next.ServeHTTP(w, r)
	})
}

func forwardedClient(trusted []netip.Prefix, r *http.Request) (netip.Addr, bool) {
	isTrusted := func(a netip.Addr) bool {
		return slices.ContainsFunc(trusted, func(p netip.Prefix) bool { return p.Contains(a) })
	}
	peer, err := netip.ParseAddr(ClientAddr(r))
	if err != nil || !isTrusted(peer.Unmap()) {
		return netip.Addr{}, false
	}
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			return netip.Addr{}, false
		}
		hop = hop.Unmap()
		if !isTrusted(hop) {
			return hop, true
		}
	}
	return netip.Addr{}, false
}
