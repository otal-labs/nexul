// Package automations implements first-party event-driven code, not a no-code rule engine.
package automations

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Kind distinguishes shipped code from owner-written code; both run through the same dial-in machinery.
type Kind string

const (
	KindDefault Kind = "default"
	KindCustom  Kind = "custom"
)

func (k Kind) valid() bool {
	return k == KindDefault || k == KindCustom
}

// Automation is one first-party event-driven program; code overwrites its declared fields at each dial-in.
type Automation struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Kind           Kind            `json:"kind"`
	Enabled        bool            `json:"enabled"`
	Subscriptions  []string        `json:"subscriptions"`
	ConfigSchema   json.RawMessage `json:"config_schema"`
	ConfigValues   json.RawMessage `json:"config_values"`
	Scopes         []string        `json:"scopes"`
	CreatedBy      string          `json:"created_by"`
	TokenPrefix    string          `json:"token_prefix,omitempty"`
	TokenHash      string          `json:"-"`
	TokenRevokedAt *time.Time      `json:"token_revoked_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Validate checks the fields every automation must carry regardless of how it was constructed.
func (a *Automation) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("%w: automation name is required", apperrs.ErrInvalid)
	}
	if !a.Kind.valid() {
		return fmt.Errorf("%w: automation kind must be default or custom", apperrs.ErrInvalid)
	}
	return nil
}

// normalizeScopes trims, drops empties, and dedupes preserving order; at least one scope is required.
func normalizeScopes(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: at least one scope is required", apperrs.ErrInvalid)
	}
	return out, nil
}
