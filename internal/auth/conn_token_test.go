package auth

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// parseConnectionToken checks a minted token the way a verifier holding the secret would: signature, expiry, claims.
func parseConnectionToken(s *Service, token string) (ConnectionTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ConnectionTokenClaims{}, apperrs.ErrUnauthorized
	}
	if !hmac.Equal([]byte(parts[2]), []byte(s.mac(parts[0]+"."+parts[1]))) {
		return ConnectionTokenClaims{}, apperrs.ErrUnauthorized
	}
	dec, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ConnectionTokenClaims{}, apperrs.ErrUnauthorized
	}
	var c ConnectionTokenClaims
	if err := json.Unmarshal(dec, &c); err != nil {
		return ConnectionTokenClaims{}, apperrs.ErrUnauthorized
	}
	if c.InstanceURL == "" || s.cfg.Now().Unix() >= c.Exp {
		return ConnectionTokenClaims{}, apperrs.ErrUnauthorized
	}
	return c, nil
}
