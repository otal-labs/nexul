package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// patPrefix marks a personal access token so the gateway can tell it from a session token without verifying both.
const patPrefix = "dep_"

// MintPAT creates a PAT for the calling user; the raw token is returned once and never persisted, only its hash.
func (s *Service) MintPAT(ctx context.Context, userID, name string) (string, *PersonalAccessToken, error) {
	return s.mintPAT(ctx, userID, name, "")
}

// MintComputerPAT mints the token a paired computer's providers connect to MCP with, replacing its active one.
func (s *Service) MintComputerPAT(ctx context.Context, userID, computerID, computerName string) (string, *PersonalAccessToken, error) {
	computerID = strings.TrimSpace(computerID)
	if computerID == "" {
		return "", nil, fmt.Errorf("%w: computer id is required", apperrs.ErrInvalid)
	}
	if err := s.RevokeComputerPAT(ctx, userID, computerID); err != nil {
		return "", nil, err
	}
	return s.mintPAT(ctx, userID, "Nexul MCP on "+strings.TrimSpace(computerName), computerID)
}

func (s *Service) mintPAT(ctx context.Context, userID, name, computerID string) (string, *PersonalAccessToken, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("%w: token name is required", apperrs.ErrInvalid)
	}
	if _, err := s.cfg.Users.GetUserByID(ctx, userID); err != nil {
		return "", nil, fmt.Errorf("get user %s: %w", userID, err)
	}
	raw, err := newPAT()
	if err != nil {
		return "", nil, err
	}
	pat := &PersonalAccessToken{
		ID:         newUserID(),
		UserID:     userID,
		Name:       name,
		Prefix:     raw[len(raw)-6:],
		CreatedAt:  s.cfg.Now(),
		ComputerID: computerID,
		TokenHash:  hashPAT(raw),
	}
	if err := s.cfg.PATs.Create(ctx, pat, tokenEvent(TopicTokenMinted, *pat)); err != nil {
		return "", nil, fmt.Errorf("create pat: %w", err)
	}
	return raw, pat, nil
}

// ListPATs returns the user's tokens, revoked included so the UI can show state; only metadata, never a credential.
func (s *Service) ListPATs(ctx context.Context, userID string) ([]PersonalAccessToken, error) {
	return s.cfg.PATs.ListByUser(ctx, userID)
}

// RevokePAT invalidates a token immediately; later requests fail, since AuthenticatePAT re-reads the store each time.
func (s *Service) RevokePAT(ctx context.Context, userID, id string) error {
	tokens, err := s.cfg.PATs.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list pats: %w", err)
	}
	i := slices.IndexFunc(tokens, func(p PersonalAccessToken) bool { return p.ID == id })
	if i < 0 {
		return fmt.Errorf("revoke pat %s: %w", id, apperrs.ErrNotFound)
	}
	return s.cfg.PATs.Revoke(ctx, id, userID, tokenEvent(TopicTokenRevoked, tokens[i]))
}

// ComputerPAT returns the computer's active token, or ErrNotFound when it has none.
func (s *Service) ComputerPAT(ctx context.Context, userID, computerID string) (*PersonalAccessToken, error) {
	return s.cfg.PATs.GetActiveForComputer(ctx, userID, computerID)
}

// RevokeComputerPAT revokes the computer's active token; a computer without one is already in the wanted state.
func (s *Service) RevokeComputerPAT(ctx context.Context, userID, computerID string) error {
	pat, err := s.cfg.PATs.GetActiveForComputer(ctx, userID, computerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get token for computer %s: %w", computerID, err)
	}
	if err := s.cfg.PATs.Revoke(ctx, pat.ID, userID, tokenEvent(TopicTokenRevoked, *pat)); err != nil {
		return fmt.Errorf("revoke token for computer %s: %w", computerID, err)
	}
	return nil
}

func tokenEvent(topic string, p PersonalAccessToken) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: TokenChangedEvent{
		TokenID: p.ID, UserID: p.UserID, Name: p.Name, ComputerID: p.ComputerID,
	}}
}

// AuthenticatePAT is used instead of Verify when the token carries the dep_ prefix.
func (s *Service) AuthenticatePAT(ctx context.Context, raw string) (*User, error) {
	pat, err := s.cfg.PATs.GetByHash(ctx, hashPAT(raw))
	if err != nil {
		return nil, apperrs.ErrUnauthorized
	}
	if pat.RevokedAt != nil {
		return nil, apperrs.ErrUnauthorized
	}
	user, err := s.cfg.Users.GetUserByID(ctx, pat.UserID)
	if err != nil {
		return nil, apperrs.ErrUnauthorized
	}
	if !accountIsActive(user.AccountStatus) {
		return nil, apperrs.ErrUnauthorized
	}
	// Best-effort last-used stamp; a write failure must not reject an otherwise valid request.
	_ = s.cfg.PATs.TouchLastUsed(ctx, pat.ID)
	return user, nil
}

// newPAT mints a raw token: patPrefix + 32 bytes of randomness, base64url.
func newPAT() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate pat: %w", err)
	}
	return patPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// hashPAT derives the stored hash of a raw token; hashes are long-lived and high-entropy, so a non-keyed hash is safe.
func hashPAT(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// isPAT reports whether a Bearer token is a PAT rather than a session token; session tokens never carry the prefix.
func isPAT(token string) bool {
	return strings.HasPrefix(token, patPrefix)
}
