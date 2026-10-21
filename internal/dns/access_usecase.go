package dns

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// DeleteAccessServiceToken removes the instance's token at Cloudflare and locally; with none stored it is a no-op.
func (s *Service) DeleteAccessServiceToken(ctx context.Context) error {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	stored, err := s.repo.GetAccessServiceToken(ctx)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get access service token: %w", err)
	}
	p, err := s.accessProviderFor(ctx)
	if err != nil {
		return err
	}
	if err := p.DeleteServiceToken(ctx, stored.ID); err != nil {
		return s.classify(err)
	}
	if err := s.repo.DeleteAccessServiceToken(ctx); err != nil {
		return fmt.Errorf("delete access service token: %w", err)
	}
	return nil
}

// DeleteAccessApp removes a hostname's Access app; an already-absent app is a no-op success.
func (s *Service) DeleteAccessApp(ctx context.Context, appID string) error {
	if strings.TrimSpace(appID) == "" {
		return fmt.Errorf("%w: access app id is required", apperrs.ErrInvalid)
	}
	p, err := s.accessProviderFor(ctx)
	if err != nil {
		return err
	}
	if err := p.DeleteAccessApp(ctx, appID); err != nil {
		return s.classify(err)
	}
	return nil
}

func (s *Service) accessProviderFor(ctx context.Context) (AccessProvider, error) {
	if s.access != nil {
		return s.access, nil
	}
	token, err := s.resolveToken(ctx)
	if err != nil {
		return nil, err
	}
	if s.newAccess == nil {
		return nil, fmt.Errorf("%w: dns access provider constructor is not wired", apperrs.ErrInvalid)
	}
	p, err := s.newAccess(ctx, token, s.instanceAccount(ctx))
	if err != nil {
		return nil, fmt.Errorf("build dns access provider: %w", err)
	}
	return p, nil
}

// instanceAccount is the account owning the instance's zone, where its Access apps live; empty when it can't be
// told, which leaves the account to the provider.
func (s *Service) instanceAccount(ctx context.Context) string {
	p, err := s.providerFor(ctx)
	if err != nil {
		return ""
	}
	z, err := s.instanceZone(ctx, p)
	if err != nil {
		return ""
	}
	return z.AccountID
}
