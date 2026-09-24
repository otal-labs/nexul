package pairing

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// MCPToken is a computer's own personal access token, the one its providers connect to Nexul's MCP server with.
type MCPToken struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// MintedMCPToken carries the raw token, returned once at mint time and never readable again.
type MintedMCPToken struct {
	MCPToken
	Token string `json:"token"`
}

// MCPTokens is auth's personal access token slice (ADR 0017); every call is scoped to userID's own tokens.
type MCPTokens interface {
	// MintComputerToken revokes the computer's active token and mints its replacement, named after the computer.
	MintComputerToken(ctx context.Context, userID, computerID, computerName string) (string, *MCPToken, error)
	// ComputerToken returns the computer's active token, or nil when it has none.
	ComputerToken(ctx context.Context, userID, computerID string) (*MCPToken, error)
	// RevokeComputerToken revokes the computer's active token; a computer without one is not an error.
	RevokeComputerToken(ctx context.Context, userID, computerID string) error
}

// MintMCPToken mints the caller's computer its "Nexul MCP on <computer>" token, replacing the one it had.
func (s *Service) MintMCPToken(ctx context.Context, userID, computerID string) (*MintedMCPToken, error) {
	computer, tokens, err := s.tokenComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	raw, token, err := tokens.MintComputerToken(ctx, userID, computer.ID, computer.Name)
	if err != nil {
		return nil, fmt.Errorf("mint mcp token for computer %s: %w", computer.ID, err)
	}
	return &MintedMCPToken{MCPToken: *token, Token: raw}, nil
}

// GetMCPToken returns the caller's computer's active MCP token without its secret, or nil when it has none.
func (s *Service) GetMCPToken(ctx context.Context, userID, computerID string) (*MCPToken, error) {
	computer, tokens, err := s.tokenComputer(ctx, userID, computerID)
	if err != nil {
		return nil, err
	}
	token, err := tokens.ComputerToken(ctx, userID, computer.ID)
	if err != nil {
		return nil, fmt.Errorf("get mcp token for computer %s: %w", computer.ID, err)
	}
	return token, nil
}

// RevokeMCPToken revokes the caller's computer's MCP token; its providers lose Nexul's MCP server until one is minted again.
func (s *Service) RevokeMCPToken(ctx context.Context, userID, computerID string) error {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return err
	}
	return s.revokeMCPToken(ctx, *computer)
}

func (s *Service) revokeMCPToken(ctx context.Context, computer Computer) error {
	tokens, err := s.tokenSeam()
	if err != nil {
		return err
	}
	if err := tokens.RevokeComputerToken(ctx, computer.UserID, computer.ID); err != nil {
		return fmt.Errorf("revoke mcp token for computer %s: %w", computer.ID, err)
	}
	return nil
}

func (s *Service) tokenComputer(ctx context.Context, userID, computerID string) (*Computer, MCPTokens, error) {
	computer, err := s.ownComputer(ctx, userID, computerID)
	if err != nil {
		return nil, nil, err
	}
	tokens, err := s.tokenSeam()
	if err != nil {
		return nil, nil, err
	}
	return computer, tokens, nil
}

func (s *Service) tokenSeam() (MCPTokens, error) {
	if s.tokens == nil {
		return nil, apperrs.Fatal(fmt.Errorf("%w: computer mcp tokens are not wired", apperrs.ErrFatal))
	}
	return s.tokens, nil
}
