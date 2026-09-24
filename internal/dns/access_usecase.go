package dns

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const serviceTokenName = "Nexul server"

// EnsureAccessServiceToken returns the instance's Access service token in the clear, creating and storing it on first use.
func (s *Service) EnsureAccessServiceToken(ctx context.Context) (*ServiceToken, error) {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	return s.ensureServiceToken(ctx)
}

func (s *Service) ensureServiceToken(ctx context.Context) (*ServiceToken, error) {
	stored, err := s.repo.GetAccessServiceToken(ctx)
	if err == nil {
		return s.openServiceToken(stored)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get access service token: %w", err)
	}
	p, err := s.accessProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	tok, err := p.CreateServiceToken(ctx, serviceTokenName)
	if err != nil {
		return nil, s.classify(err)
	}
	tok.CreatedAt = s.now().UTC()
	if err := s.saveServiceToken(ctx, *tok); err != nil {
		// An unstored token can never be used or cleaned up later, and the account holds only 50.
		return nil, errors.Join(err, p.DeleteServiceToken(ctx, tok.ID))
	}
	return tok, nil
}

// RotateAccessServiceToken issues a new secret for the instance's token; every Access app keeps admitting it by id.
func (s *Service) RotateAccessServiceToken(ctx context.Context) (*ServiceToken, error) {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	stored, err := s.repo.GetAccessServiceToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access service token: %w", err)
	}
	p, err := s.accessProviderFor(ctx)
	if err != nil {
		return nil, err
	}
	tok, err := p.RotateServiceToken(ctx, stored.ID)
	if err != nil {
		return nil, s.classify(err)
	}
	tok.CreatedAt = stored.CreatedAt
	if err := s.saveServiceToken(ctx, *tok); err != nil {
		return nil, err
	}
	return tok, nil
}

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

// CreateAccessApp closes hostname to everything but a request carrying the instance's service token, returning the app id.
func (s *Service) CreateAccessApp(ctx context.Context, hostname string) (string, error) {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return "", fmt.Errorf("%w: hostname is required", apperrs.ErrInvalid)
	}
	tok, err := s.EnsureAccessServiceToken(ctx)
	if err != nil {
		return "", err
	}
	p, err := s.accessProviderFor(ctx)
	if err != nil {
		return "", err
	}
	id, err := p.CreateAccessApp(ctx, hostname, tok.ID)
	if err != nil {
		return "", s.classify(err)
	}
	return id, nil
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
	p, err := s.newAccess(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("build dns access provider: %w", err)
	}
	return p, nil
}

// saveServiceToken stores tok with its secret sealed, leaving the caller's copy in the clear.
func (s *Service) saveServiceToken(ctx context.Context, tok ServiceToken) error {
	enc, err := s.encryptTunnelToken(tok.ClientSecret)
	if err != nil {
		return err
	}
	tok.ClientSecret = enc
	tok.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveAccessServiceToken(ctx, tok); err != nil {
		return fmt.Errorf("save access service token: %w", err)
	}
	return nil
}

func (s *Service) openServiceToken(stored *ServiceToken) (*ServiceToken, error) {
	secret, err := s.decryptTunnelToken(stored.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("open access service token: %w", err)
	}
	out := *stored
	out.ClientSecret = secret
	return &out, nil
}
