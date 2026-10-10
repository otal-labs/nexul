package githubapp

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// ParsePrivateKey reads the .pem GitHub generates for an App: PKCS#1, or PKCS#8 holding an RSA key.
func ParsePrivateKey(pemText string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemText)))
	if block == nil {
		return nil, fmt.Errorf("%w: the private key is not a PEM file; paste the whole .pem, BEGIN and END lines included", apperrs.ErrInvalid)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: the private key could not be read: %v", apperrs.ErrInvalid, err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: the private key is not an RSA key", apperrs.ErrInvalid)
	}
	return key, nil
}

// SignJWT signs the short-lived RS256 token GitHub accepts as the App itself; the client ID is the issuer.
func SignJWT(clientID string, key *rsa.PrivateKey, now time.Time) (string, error) {
	enc := base64.RawURLEncoding
	claims, err := json.Marshal(map[string]any{
		// Backdated a minute and kept under GitHub's ten-minute ceiling, so a skewed clock on either side still passes.
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": clientID,
	})
	if err != nil {
		return "", err
	}
	signing := enc.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`)) + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("sign app jwt: %w", err)
	}
	return signing + "." + enc.EncodeToString(sig), nil
}

// VerifyKey confirms GitHub accepts a JWT signed with pemText as the App with clientID, through GET /app.
func VerifyKey(ctx context.Context, hc *http.Client, apiBase, clientID, pemText string) (err error) {
	if apiBase == "" {
		apiBase = DefaultAPIBase
	}
	key, err := ParsePrivateKey(pemText)
	if err != nil {
		return err
	}
	token, err := SignJWT(clientID, key, time.Now())
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/app", nil)
	if err != nil {
		return fmt.Errorf("build app request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("reach github: %w", err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: GitHub rejected the private key for client ID %s; generate it on the same App", apperrs.ErrInvalid, clientID)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github app lookup: status %d", resp.StatusCode)
	}
	var app struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return fmt.Errorf("decode github app: %w", err)
	}
	if app.ClientID != "" && app.ClientID != clientID {
		return fmt.Errorf("%w: the private key belongs to the App with client ID %s, not %s", apperrs.ErrInvalid, app.ClientID, clientID)
	}
	return nil
}
