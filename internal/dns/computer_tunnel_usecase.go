package dns

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// ComputerTunnel is a paired computer's own tunnel (ADR 0062); the pairing domain stores it, never dns_tunnels.
type ComputerTunnel struct {
	TunnelID    string
	Hostname    string
	ZoneID      string
	RecordID    string
	AccessAppID string
}

// computerLabelMax leaves room in a 63-byte DNS label for the dash and the eight random characters.
const computerLabelMax = 63 - 9

// CreateComputerTunnel routes <name slug>-<8 random>.<instance zone> to the computer's loopback port, Access app before DNS record.
func (s *Service) CreateComputerTunnel(ctx context.Context, name string, port int) (*ComputerTunnel, error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("%w: local harness port %d is not between 1 and 65535", apperrs.ErrInvalid, port)
	}
	dnsp, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	zone, err := s.instanceZone(ctx, dnsp)
	if err != nil {
		return nil, err
	}
	tp, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	label := computerLabel(name)
	created, err := tp.CreateTunnel(ctx, label)
	if err != nil {
		return nil, s.classify(err)
	}
	ct := &ComputerTunnel{TunnelID: created.ID, Hostname: label + "." + zone.Name, ZoneID: zone.ID}
	if err := s.openComputerTunnel(ctx, tp, dnsp, ct, label, port); err != nil {
		// Nothing was stored yet, so whatever did get created at Cloudflare would otherwise be orphaned.
		return nil, errors.Join(err, s.DeleteComputerTunnel(ctx, *ct))
	}
	return ct, nil
}

// openComputerTunnel fills in ct step by step so a failure leaves exactly the ids a rollback needs.
func (s *Service) openComputerTunnel(ctx context.Context, tp TunnelProvider, dnsp DNSProvider, ct *ComputerTunnel, label string, port int) error {
	appID, err := s.CreateAccessApp(ctx, ct.Hostname)
	if err != nil {
		return err
	}
	ct.AccessAppID = appID
	if err := tp.RouteTunnelHostname(ctx, ct.TunnelID, ct.Hostname, "http://127.0.0.1:"+strconv.Itoa(port)); err != nil {
		return s.classify(err)
	}
	// A CNAME to *.cfargotunnel.com only resolves through Cloudflare's proxy; DNS-only leaves the hostname dead.
	rec, err := dnsp.CreateRecord(ctx, ct.ZoneID, RecordInput{
		Type: RecordCNAME, Name: label, Content: ct.TunnelID + ".cfargotunnel.com", TTL: 1, Proxied: true,
	})
	if err != nil {
		return s.classify(err)
	}
	ct.RecordID = rec.ID
	return nil
}

// DeleteComputerTunnel rotates the token to disconnect the computer, then deletes tunnel, record, and app; missing parts are skipped.
func (s *Service) DeleteComputerTunnel(ctx context.Context, ct ComputerTunnel) error {
	if ct.TunnelID != "" {
		if err := s.deleteTunnelAtProvider(ctx, ct.TunnelID); err != nil {
			return err
		}
	}
	if ct.RecordID != "" {
		dnsp, err := s.providerFor(ctx)
		if err != nil {
			return err
		}
		if err := dnsp.DeleteRecord(ctx, ct.ZoneID, ct.RecordID); err != nil {
			return s.classify(err)
		}
	}
	if ct.AccessAppID != "" {
		return s.DeleteAccessApp(ctx, ct.AccessAppID)
	}
	return nil
}

func (s *Service) deleteTunnelAtProvider(ctx context.Context, tunnelID string) error {
	tp, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return err
	}
	_, err = tp.RotateTunnelCredentials(ctx, tunnelID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return s.classify(err)
	}
	if err := tp.DeleteTunnel(ctx, tunnelID); err != nil {
		return s.classify(err)
	}
	return nil
}

// ComputerTunnelStatus returns Cloudflare's connector status for a computer tunnel.
func (s *Service) ComputerTunnelStatus(ctx context.Context, tunnelID string) (string, error) {
	if strings.TrimSpace(tunnelID) == "" {
		return "", fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	tp, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return "", err
	}
	t, err := tp.GetTunnel(ctx, tunnelID)
	if err != nil {
		return "", s.classify(err)
	}
	return t.Status, nil
}

// ComputerTunnelToken returns the connector token a computer's cloudflared service runs with.
func (s *Service) ComputerTunnelToken(ctx context.Context, tunnelID string) (string, error) {
	if strings.TrimSpace(tunnelID) == "" {
		return "", fmt.Errorf("%w: tunnel id is required", apperrs.ErrInvalid)
	}
	tp, err := s.tunnelProviderFor(ctx)
	if err != nil {
		return "", err
	}
	token, err := tp.TunnelToken(ctx, tunnelID)
	if err != nil {
		return "", s.classify(err)
	}
	return token, nil
}

// instanceZone is the zone holding the instance's host; hostnames sit one label under it, all the free certificate covers.
func (s *Service) instanceZone(ctx context.Context, p DNSProvider) (Zone, error) {
	instanceURL, err := s.instanceURL(ctx)
	if err != nil {
		return Zone{}, err
	}
	host, err := hostFromURL(instanceURL)
	if err != nil {
		return Zone{}, err
	}
	zones, err := p.ListZones(ctx)
	if err != nil {
		return Zone{}, s.classify(err)
	}
	host = strings.ToLower(host)
	var best Zone
	for _, z := range zones {
		name := strings.ToLower(z.Name)
		if (host == name || strings.HasSuffix(host, "."+name)) && len(name) > len(best.Name) {
			best = Zone{ID: z.ID, Name: name, Status: z.Status}
		}
	}
	if best.ID == "" {
		return Zone{}, fmt.Errorf("%w: the instance's host %s is in no Cloudflare zone the connected token can edit", apperrs.ErrInvalid, host)
	}
	return best, nil
}

// computerLabel is the computer's name slugged plus eight random characters, so nobody can guess another computer's hostname.
func computerLabel(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > computerLabelMax {
		slug = strings.Trim(slug[:computerLabelMax], "-")
	}
	if slug == "" {
		slug = "computer"
	}
	return slug + "-" + strings.ToLower(rand.Text()[:8])
}
