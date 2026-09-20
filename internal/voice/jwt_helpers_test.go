package voice

// Test-only HS256 JWT helpers mirroring internal/livekit's own signJWT/
// parseJWT (unexported there, so voice's tests — a different package — need
// their own copy to build signed test fixtures and inspect minted tokens).

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

var testJWTHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

func signTestJWT(t *testing.T, claims map[string]any, secret string) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal test jwt claims: %v", err)
	}
	unsigned := testJWTHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func parseTestJWT(token, secret string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token: want 3 segments, got %d", len(parts))
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	if !hmac.Equal(gotSig, mac.Sum(nil)) {
		return nil, fmt.Errorf("signature mismatch")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}
	return claims, nil
}

// signTestWebhook builds the Authorization header LiveKit's server would
// send for body: an HS256 JWT whose sha256 claim is body's base64-std-encoded
// SHA-256 hash (docs.livekit.io/home/server/webhooks), mirroring
// internal/livekit's own webhook_test.go helper.
func signTestWebhook(t *testing.T, secret string, body []byte, ttl time.Duration) string {
	t.Helper()
	hash := sha256.Sum256(body)
	now := time.Now().UTC()
	claims := map[string]any{
		"iss":    "key1",
		"nbf":    now.Unix(),
		"exp":    now.Add(ttl).Unix(),
		"sha256": base64.StdEncoding.EncodeToString(hash[:]),
	}
	return "Bearer " + signTestJWT(t, claims, secret)
}
