package auth

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// connectionTokenTTL bounds a token's lifetime; not secret, but expiring it forces a re-import eventually.
const connectionTokenTTL = 30 * 24 * time.Hour

// signConnectionToken mints a minimal HS256 JWT carrying the instance URL, MCP endpoint, and settings version.
func (s *Service) signConnectionToken(st Settings) (string, error) {
	now := s.cfg.Now()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(ConnectionTokenClaims{
		InstanceURL: st.InstanceURL,
		MCPURL:      mcpURLFor(st.InstanceURL),
		Version:     st.SettingsVersion,
		Iat:         now.Unix(),
		Exp:         now.Add(connectionTokenTTL).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("sign connection token: %w", err)
	}
	enc := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	return enc + "." + s.mac(enc), nil
}

// ParseConnectionToken validates a token's signature and expiry and returns its claims, mirroring Verify's mechanism.
func (s *Service) ParseConnectionToken(token string) (ConnectionTokenClaims, error) {
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

// mcpURLFor derives the MCP endpoint from the instance URL: the same origin at /mcp, which the main HTTP
// listener serves and every proxy in front of it forwards; "" if the instance URL is unusable.
func mcpURLFor(instanceURL string) string {
	u, err := url.Parse(instanceURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/mcp"
}
