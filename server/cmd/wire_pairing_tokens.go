package main

import (
	"context"
	"errors"
	"time"

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

// MintRunToken is pairing's seam for the token a run on someone else's computer reaches Nexul with (ADR 0210).
func (a pairingMCPTokens) MintRunToken(ctx context.Context, userID, name string, expires time.Time) (string, string, error) {
	raw, pat, err := a.auth.MintRunPAT(ctx, userID, name, expires)
	if err != nil {
		return "", "", err
	}
	return raw, pat.ID, nil
}

// RevokeRunToken is pairing's seam for ending that token.
func (a pairingMCPTokens) RevokeRunToken(ctx context.Context, userID, id string) error {
	return a.auth.RevokePAT(ctx, userID, id)
}
