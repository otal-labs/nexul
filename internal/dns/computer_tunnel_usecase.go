package dns

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// ComputerTunnel is a computer tunnel left from pairing through one (ADR 0062); the pairing domain stores it, never dns_tunnels.
type ComputerTunnel struct {
	TunnelID    string
	Hostname    string
	ZoneID      string
	RecordID    string
	AccessAppID string
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
	tp, err := s.tunnelProviderFor(ctx, tunnelID)
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
			best = Zone{ID: z.ID, Name: name, Status: z.Status, AccountID: z.AccountID, AccountName: z.AccountName}
		}
	}
	if best.ID == "" {
		return Zone{}, fmt.Errorf("%w: the instance's host %s is in no Cloudflare zone the connected token can edit", apperrs.ErrInvalid, host)
	}
	return best, nil
}
