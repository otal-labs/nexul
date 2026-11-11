// Package license checks the instance's enterprise license key: a signed file naming its licensee, the instance it is
// for, the features it unlocks and its expiry, verified offline against the public key built into the binary (ADR 0184).
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Feature names one enterprise feature a key can unlock.
type Feature string

const (
	FeatureSSO      Feature = "sso"
	FeatureBranding Feature = "branding"
)

// Features lists every enterprise feature this build knows; a key may name others a newer build added.
var Features = []Feature{FeatureSSO, FeatureBranding}

// NoticeWindow is how long before its expiry the instance Owner is told the key is running out.
const NoticeWindow = 14 * 24 * time.Hour

// keyPrefix opens every key's text, so a pasted string that is not a key is refused by name.
const keyPrefix = "nexul-license-1."

// State is where the instance's key stands right now.
type State string

const (
	StateNone     State = "none"
	StateActive   State = "active"
	StateExpiring State = "expiring"
	StateExpired  State = "expired"
	StateInvalid  State = "invalid"
)

// Claims are what a key's issuer signed.
type Claims struct {
	Licensee    string    `json:"licensee"`
	InstanceURL string    `json:"instance_url"`
	Features    []Feature `json:"features"`
	ExpiresAt   time.Time `json:"expires_at"`
	IssuedAt    time.Time `json:"issued_at"`
}

// Status is the license as the instance Owner and agents see it; Problem says why an invalid key unlocks nothing.
type Status struct {
	State       State      `json:"state"`
	Licensee    string     `json:"licensee"`
	InstanceURL string     `json:"instance_url"`
	Features    []Feature  `json:"features"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Problem     string     `json:"problem,omitempty"`
}

// Unlocks reports whether the status lets feature f run.
func (st Status) Unlocks(f Feature) bool {
	return (st.State == StateActive || st.State == StateExpiring) && slices.Contains(st.Features, f)
}

var (
	errNotAKey      = fmt.Errorf("%w: this is not a Nexul license key", apperrs.ErrInvalid)
	errBadSignature = fmt.Errorf("%w: this key's signature does not check out: it was changed after it was issued, or it was not issued for Nexul", apperrs.ErrInvalid)
)

// Validate refuses claims no key should carry.
func (c Claims) Validate() error {
	if strings.TrimSpace(c.Licensee) == "" {
		return fmt.Errorf("%w: a key names its licensee", apperrs.ErrInvalid)
	}
	if _, err := origin(c.InstanceURL); err != nil {
		return err
	}
	if len(c.Features) == 0 {
		return fmt.Errorf("%w: a key unlocks at least one feature", apperrs.ErrInvalid)
	}
	if c.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: a key has an expiry", apperrs.ErrInvalid)
	}
	return nil
}

// Sign makes a key's text from its claims and the issuer's private key.
func Sign(c Claims, priv ed25519.PrivateKey) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("encode claims: %w", err)
	}
	sig := ed25519.Sign(priv, payload)
	enc := base64.RawURLEncoding
	return keyPrefix + enc.EncodeToString(payload) + "." + enc.EncodeToString(sig), nil
}

// Parse checks a key's signature against pub and returns what it claims; it says nothing yet about this instance.
func Parse(key string, pub ed25519.PublicKey) (Claims, error) {
	body, ok := strings.CutPrefix(strings.TrimSpace(key), keyPrefix)
	if !ok {
		return Claims{}, errNotAKey
	}
	payloadText, sigText, ok := strings.Cut(body, ".")
	if !ok {
		return Claims{}, errNotAKey
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadText)
	if err != nil {
		return Claims{}, errNotAKey
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigText)
	if err != nil {
		return Claims{}, errNotAKey
	}
	if !ed25519.Verify(pub, payload, sig) {
		return Claims{}, errBadSignature
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, errNotAKey
	}
	if err := c.Validate(); err != nil {
		return Claims{}, err
	}
	return c, nil
}

// evaluate is where claims stand on this instance at now.
func evaluate(c Claims, instanceURL string, now time.Time) Status {
	expires := c.ExpiresAt.UTC()
	st := Status{Licensee: c.Licensee, InstanceURL: c.InstanceURL, Features: c.Features, ExpiresAt: &expires}
	if !sameInstance(c.InstanceURL, instanceURL) {
		st.State = StateInvalid
		st.Problem = wrongInstance(c.InstanceURL, instanceURL)
		return st
	}
	if !now.Before(c.ExpiresAt) {
		st.State = StateExpired
		return st
	}
	if c.ExpiresAt.Sub(now) <= NoticeWindow {
		st.State = StateExpiring
		return st
	}
	st.State = StateActive
	return st
}

func wrongInstance(keyURL, instanceURL string) string {
	if instanceURL == "" {
		return fmt.Sprintf("This key is for %s, and this instance has no address set yet.", keyURL)
	}
	return fmt.Sprintf("This key is for %s, but this instance's address is %s.", keyURL, instanceURL)
}

// sameInstance compares two public URLs by scheme, host and path, ignoring case in the host and a trailing slash.
func sameInstance(a, b string) bool {
	oa, errA := origin(a)
	ob, errB := origin(b)
	return errA == nil && errB == nil && oa == ob
}

func origin(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return "", fmt.Errorf("%w: a key names the instance's public URL, such as https://nexul.example.com", apperrs.ErrInvalid)
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + strings.TrimRight(u.Path, "/"), nil
}

// problemOf is the reason a stored key no longer parses, without the sentinel's prefix.
func problemOf(err error) string {
	msg := err.Error()
	if errors.Is(err, apperrs.ErrInvalid) {
		msg = strings.TrimPrefix(msg, apperrs.ErrInvalid.Error()+": ")
	}
	if msg == "" {
		return msg
	}
	return strings.ToUpper(msg[:1]) + msg[1:] + "."
}
