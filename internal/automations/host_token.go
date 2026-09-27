package automations

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// hostTokenPrefix marks a host-scoped automation token; '.' never occurs in a minted dat_ token.
const hostTokenPrefix = tokenPrefix + "h."

// hostToken is what a host's worker dials in with: bound to the automation, the host's live credential and the
// automation's own token, so a move, a host removal or a token rotation or revocation each invalidate it.
func hostToken(key []byte, automationID, credentialHash, tokenHash string) string {
	mac := hostTokenMAC(key, automationID, credentialHash, tokenHash)
	return hostTokenPrefix + automationID + "." + base64.RawURLEncoding.EncodeToString(mac)
}

func hostTokenMAC(key []byte, automationID, credentialHash, tokenHash string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte("nexul automations host token\x00" + automationID + "\x00" + credentialHash + "\x00" + tokenHash))
	return m.Sum(nil)
}

func parseHostToken(raw string) (automationID string, mac []byte, ok bool) {
	rest, ok := strings.CutPrefix(raw, hostTokenPrefix)
	if !ok {
		return "", nil, false
	}
	i := strings.LastIndex(rest, ".")
	if i <= 0 {
		return "", nil, false
	}
	mac, err := base64.RawURLEncoding.DecodeString(rest[i+1:])
	if err != nil {
		return "", nil, false
	}
	return rest[:i], mac, true
}
