package license

import (
	"crypto/ed25519"
	"encoding/base64"
)

// issuerKey checks every license key; its private half never enters the repository.
const issuerKey = "fa+a1+khnTTSvk37CnTpqqdbNhtYO3kGRxtKr8+/4I8="

// PublicKey returns the issuer's public key built into this binary.
func PublicKey() ed25519.PublicKey {
	b, err := base64.StdEncoding.DecodeString(issuerKey)
	if err != nil || len(b) != ed25519.PublicKeySize {
		panic("license: the built-in issuer key is not an ed25519 public key")
	}
	return b
}
