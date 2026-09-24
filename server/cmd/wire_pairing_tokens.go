package main

import (
	"context"
	"errors"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// pairingMCPTokens adapts auth's per-computer personal access tokens to pairing's MCPTokens seam (ADR 0017).
type pairingMCPTokens struct {
	auth *auth.Service
}

func (a pairingMCPTokens) MintComputerToken(ctx context.Context, userID, computerID, computerName string) (string, *pairing.MCPToken, error) {
	raw, pat, err := a.auth.MintComputerPAT(ctx, userID, computerID, computerName)
	if err != nil {
		return "", nil, err
	}
	return raw, toMCPToken(pat), nil
}

func (a pairingMCPTokens) ComputerToken(ctx context.Context, userID, computerID string) (*pairing.MCPToken, error) {
	pat, err := a.auth.ComputerPAT(ctx, userID, computerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toMCPToken(pat), nil
}

func (a pairingMCPTokens) RevokeComputerToken(ctx context.Context, userID, computerID string) error {
	return a.auth.RevokeComputerPAT(ctx, userID, computerID)
}

func toMCPToken(p *auth.PersonalAccessToken) *pairing.MCPToken {
	return &pairing.MCPToken{ID: p.ID, Name: p.Name, Prefix: p.Prefix, CreatedAt: p.CreatedAt, LastUsedAt: p.LastUsedAt}
}
