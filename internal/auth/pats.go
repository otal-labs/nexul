package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// patPrefix marks a personal access token so the gateway can tell it from a session token without verifying both.
const patPrefix = "dep_"

// MintPAT creates a PAT for the calling user; the raw token is returned once and never persisted, only its hash.
func (s *Service) MintPAT(ctx context.Context, userID, name string) (string, *PersonalAccessToken, error) {
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
		ID:        newUserID(),
		UserID:    userID,
		Name:      name,
		Prefix:    raw[len(raw)-6:],
		CreatedAt: s.cfg.Now(),
		TokenHash: hashPAT(raw),
	}
	if err := s.cfg.PATs.Create(ctx, pat); err != nil {
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
	return s.cfg.PATs.Revoke(ctx, id, userID)
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
