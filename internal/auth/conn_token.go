package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
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

// mcpURLFor derives the MCP endpoint from the instance URL: the same origin at /mcp, which the main HTTP
// listener serves and every proxy in front of it forwards; "" if the instance URL is unusable.
func mcpURLFor(instanceURL string) string {
	u, err := url.Parse(instanceURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/mcp"
}
