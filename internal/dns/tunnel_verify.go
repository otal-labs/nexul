package dns

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// The checks the tunnel ticker runs once a hostname is routed, one request per row.
const (
	TunnelCheckIngress   = "ingress"
	TunnelCheckRecord    = "record"
	TunnelCheckReachable = "reachable"
)

// VerifyTunnelRoute runs one check on a tunnel's routed hostname and returns what it found, or why it fails.
func (s *Service) VerifyTunnelRoute(ctx context.Context, tunnelID, check string) (string, error) {
	if err := s.require(ctx, permissions.DNSRead); err != nil {
		return "", err
	}
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil {
		return "", fmt.Errorf("get tunnel %s: %w", tunnelID, err)
	}
	if t.Hostname == "" {
		return "", fmt.Errorf("%w: tunnel %s has no hostname routed yet", apperrs.ErrInvalid, t.Name)
	}
	switch check {
	case TunnelCheckIngress:
		return s.checkIngress(ctx, t)
	case TunnelCheckRecord:
		return s.checkTunnelRecord(ctx, t)
	case TunnelCheckReachable:
		return s.checkReachable(ctx, t.Hostname)
	}
	return "", fmt.Errorf("%w: unknown tunnel check %q", apperrs.ErrInvalid, check)
}

func (s *Service) checkIngress(ctx context.Context, t *Tunnel) (string, error) {
	tp, err := s.tunnelProviderFor(ctx, t.ID)
	if err != nil {
		return "", err
	}
	routes, err := tp.ListTunnelHostnames(ctx, t.ID)
	if err != nil {
		return "", s.classify(err)
	}
	for _, r := range routes {
		if strings.EqualFold(r.Hostname, t.Hostname) {
			return r.Hostname + " → " + r.Service, nil
		}
	}
	return "", fmt.Errorf("%w: tunnel %s has no ingress rule for %s", apperrs.ErrInvalid, t.Name, t.Hostname)
}

func (s *Service) checkTunnelRecord(ctx context.Context, t *Tunnel) (string, error) {
	p, err := s.providerFor(ctx)
	if err != nil {
		return "", err
	}
	records, err := p.ListRecords(ctx, t.ZoneID)
	if err != nil {
		return "", s.classify(err)
	}
	target := t.ID + ".cfargotunnel.com"
	for _, r := range records {
		if r.ID != t.RecordID {
			continue
		}
		if r.Type != RecordCNAME || r.Content != target {
			return "", fmt.Errorf("%w: the %s record for %s points at %s, not this tunnel", apperrs.ErrInvalid, r.Type, t.Hostname, r.Content)
		}
		if !r.Proxied {
			return "", fmt.Errorf("%w: the CNAME for %s is DNS-only; a tunnel hostname must be proxied", apperrs.ErrInvalid, t.Hostname)
		}
		return "Proxied CNAME in " + t.Zone, nil
	}
	return "", fmt.Errorf("%w: no DNS record for %s in %s", apperrs.ErrInvalid, t.Hostname, t.Zone)
}

// checkReachable asks the hostname over HTTPS from this server, so the answer comes back through Cloudflare and
// the tunnel. A 5xx is Cloudflare saying the tunnel isn't serving it (530 is a tunnel in another account or offline).
func (s *Service) checkReachable(ctx context.Context, hostname string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+hostname+"/", nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", apperrs.ErrInvalid, err)
	}
	resp, err := s.httpc.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %s does not answer yet: %w", apperrs.ErrInvalid, hostname, err)
	}
	defer func() { _ = resp.Body.Close() }() // nothing is read from the body
	if resp.StatusCode >= http.StatusInternalServerError {
		return "", fmt.Errorf("%w: %s answered HTTP %d, so the tunnel is not serving it yet", apperrs.ErrInvalid, hostname, resp.StatusCode)
	}
	return fmt.Sprintf("Answered HTTP %d over HTTPS", resp.StatusCode), nil
}

// verifyAll runs every hostname check, keyed by check, each holding its finding or its failure.
func (s *Service) verifyAll(ctx context.Context, tunnelID string) map[string]string {
	out := map[string]string{}
	for _, check := range []string{TunnelCheckIngress, TunnelCheckRecord, TunnelCheckReachable} {
		detail, err := s.VerifyTunnelRoute(ctx, tunnelID, check)
		if err != nil {
			out[check] = "failed: " + err.Error()
			continue
		}
		out[check] = "ok: " + detail
	}
	return out
}
