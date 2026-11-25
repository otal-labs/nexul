// Package webhooksig signs the bodies Nexul posts to a webhook, so a receiver can check they came from this instance:
// the integrations relay (ADR 0043) and the audit stream (ADR 0190) both send its value as X-Nexul-Signature.
package webhooksig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header is the header the signature travels in.
const Header = "X-Nexul-Signature"

// Sign returns "sha256=<hex>", the HMAC-SHA256 of payload under secret.
func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload) // a hash's Write never fails
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether header is payload's signature under secret, in constant time.
func Verify(secret string, payload []byte, header string) bool {
	return hmac.Equal([]byte(header), []byte(Sign(secret, payload)))
}
