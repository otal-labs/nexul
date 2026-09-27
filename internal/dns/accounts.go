package dns

import (
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// account is a Cloudflare account the token reaches, known through the zones it owns.
type account struct {
	ID   string
	Name string
}

// zoneAccounts lists the accounts owning the zones the token can see, once each, in zone order.
func (s *Service) zoneAccounts(ctx context.Context) ([]account, error) {
	p, err := s.providerFor(ctx)
	if err != nil {
		return nil, err
	}
	zones, err := p.ListZones(ctx)
	if err != nil {
		return nil, s.classify(err)
	}
	var out []account
	seen := map[string]bool{}
	for _, z := range zones {
		if z.AccountID == "" || seen[z.AccountID] {
			continue
		}
		seen[z.AccountID] = true
		out = append(out, account{ID: z.AccountID, Name: z.AccountName})
	}
	return out, nil
}

// creationAccount picks the account a new tunnel is created in: the one asked for, else the only one the token
// reaches. Several accounts and no choice is refused, because a tunnel in the wrong account can never serve a
// hostname in another account's zone.
func (s *Service) creationAccount(ctx context.Context, requested string) (string, error) {
	accounts, err := s.zoneAccounts(ctx)
	if err != nil {
		return "", err
	}
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested, nil
	}
	if len(accounts) == 0 {
		return "", nil
	}
	if len(accounts) == 1 {
		return accounts[0].ID, nil
	}
	names := make([]string, 0, len(accounts))
	for _, a := range accounts {
		names = append(names, fmt.Sprintf("%s (%s)", a.Name, a.ID))
	}
	return "", fmt.Errorf("%w: the Cloudflare token reaches several accounts, so choose the one that owns the hostname's zone: %s",
		apperrs.ErrInvalid, strings.Join(names, ", "))
}

// tunnelProviderIn builds a tunnel provider pinned to account; empty leaves the account to the provider.
func (s *Service) tunnelProviderIn(ctx context.Context, account string) (TunnelProvider, error) {
	if s.tunnel != nil {
		return s.tunnel, nil
	}
	token, err := s.resolveToken(ctx)
	if err != nil {
		return nil, err
	}
	if s.newTunnel == nil {
		return nil, fmt.Errorf("%w: dns tunnel provider constructor is not wired", apperrs.ErrInvalid)
	}
	p, err := s.newTunnel(ctx, token, account)
	if err != nil {
		return nil, fmt.Errorf("build dns tunnel provider: %w", err)
	}
	return p, nil
}

// tunnelProviderFor builds the provider for the account tunnelID lives in: the tracked row's account, else the
// account that answers for it (a tunnel found running on a machine, or a computer's tunnel).
func (s *Service) tunnelProviderFor(ctx context.Context, tunnelID string) (TunnelProvider, error) {
	if s.tunnel != nil {
		return s.tunnel, nil
	}
	if t, err := s.repo.GetTunnel(ctx, tunnelID); err == nil && t.AccountID != "" {
		return s.tunnelProviderIn(ctx, t.AccountID)
	}
	accounts, err := s.zoneAccounts(ctx)
	if err != nil {
		return nil, err
	}
	if len(accounts) < 2 {
		return s.tunnelProviderIn(ctx, firstAccountID(accounts))
	}
	for _, a := range accounts {
		p, err := s.tunnelProviderIn(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		if _, err := p.GetTunnel(ctx, tunnelID); err == nil {
			return p, nil
		}
	}
	return s.tunnelProviderIn(ctx, accounts[0].ID)
}

func firstAccountID(accounts []account) string {
	if len(accounts) == 0 {
		return ""
	}
	return accounts[0].ID
}

// checkSameAccount refuses to route a zone's hostname into a tunnel of another account: Cloudflare accepts the
// ingress rule and the CNAME, but the hostname never resolves.
func (s *Service) checkSameAccount(ctx context.Context, tunnelID, zoneID string) error {
	t, err := s.repo.GetTunnel(ctx, tunnelID)
	if err != nil || t.AccountID == "" {
		return nil
	}
	p, err := s.providerFor(ctx)
	if err != nil {
		return err
	}
	zones, err := p.ListZones(ctx)
	if err != nil {
		return s.classify(err)
	}
	for _, z := range zones {
		if z.ID == zoneID && z.AccountID != "" && z.AccountID != t.AccountID {
			return fmt.Errorf("%w: %s belongs to the Cloudflare account %s, but tunnel %s was created in another account; "+
				"create a tunnel in %s to serve it", apperrs.ErrInvalid, z.Name, z.AccountName, t.Name, z.AccountName)
		}
	}
	return nil
}
