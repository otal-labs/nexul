// Package pairing implements the user-level half of @Agent: paired computers, their harness sessions, and defaults.
package pairing

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Computer has no upstream refresh flow, so TokenExpiresAt governs expiry.
type Computer struct {
	ID             string       `json:"id"`
	UserID         string       `json:"-"`
	Kind           harness.Kind `json:"kind"`
	Name           string       `json:"name"`
	ServerURL      string       `json:"server_url"`
	TokenExpiresAt time.Time    `json:"token_expires_at"`
	HarnessVersion string       `json:"harness_version"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	// BearerToken is encrypted at rest; the `-` tag keeps it off every HTTP response.
	BearerToken string `json:"-"`
}

// Session is the harness-facing view of a computer; only call it on a decrypted copy.
func (c Computer) Session() harness.Session {
	return harness.Session{Name: c.Name, ServerURL: c.ServerURL, BearerToken: c.BearerToken}
}

// Defaults are a user's pairing-settings defaults for non-project chat contexts; every field is optional.
type Defaults struct {
	UserID            string `json:"-"`
	DefaultComputerID string `json:"default_computer_id,omitempty"`
	FallbackProjectID string `json:"fallback_project_id,omitempty"`
	Provider          string `json:"provider,omitempty"`
	Model             string `json:"model,omitempty"`
}

// ProjectLink is the project-level pairing config; a zero value means unlinked, falling through to defaults.
type ProjectLink struct {
	ProjectID        string    `json:"project_id"`
	ComputerID       string    `json:"computer_id,omitempty"`
	HarnessProjectID string    `json:"harness_project_id,omitempty"`
	Provider         string    `json:"provider,omitempty"`
	Model            string    `json:"model,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

// NotConfiguredReason distinguishes why ResolveTarget failed, for a specific reply, not one generic message.
type NotConfiguredReason string

const (
	// ReasonUnpaired: no computer resolves at all.
	ReasonUnpaired NotConfiguredReason = "unpaired"
	// ReasonExpiredToken: a computer resolved, but its bearer session expired; the fix is re-pairing.
	ReasonExpiredToken NotConfiguredReason = "expired_token"
	// ReasonNoDefault: a valid computer resolved, but no harness project is configured for it.
	ReasonNoDefault NotConfiguredReason = "no_default"
	// ReasonNoDefaultComputer: several paired computers, none picked as default; the fix is choosing one.
	ReasonNoDefaultComputer NotConfiguredReason = "no_default_computer"
)

// NotConfiguredError unwraps to ErrInvalid; Reason carries the specific fix.
type NotConfiguredError struct {
	Reason NotConfiguredReason
}

func (e *NotConfiguredError) Error() string {
	return fmt.Sprintf("pairing not configured: %s", e.Reason)
}

func (e *NotConfiguredError) Unwrap() error { return apperrs.ErrInvalid }

// validateHarnessProjectID rejects a blank harness-side project id.
func validateHarnessProjectID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: harness project id is required", apperrs.ErrInvalid)
	}
	return id, nil
}

// validateName rejects a blank computer name.
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: computer name is required", apperrs.ErrInvalid)
	}
	return name, nil
}

// validateServerURL rejects non-http(s) origins and trims a trailing slash so clients can concatenate paths.
func validateServerURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%w: server URL is required", apperrs.ErrInvalid)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("%w: server URL %q is not a valid http(s) URL", apperrs.ErrInvalid, raw)
	}
	return strings.TrimRight(raw, "/"), nil
}
