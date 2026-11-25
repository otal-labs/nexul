// Package auditlog keeps the audit log longer than the free 45 days and streams each row to one signed HTTPS webhook,
// an enterprise feature unlocked by the license key (ADR 0184, ADR 0190). The log itself stays integrations'.
package auditlog

import (
	"fmt"
	"net/url"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const (
	// MinDays is the free log's 45 days (ADR 0138): the default, and the floor of the setting.
	MinDays = 45
	// MaxDays is the longest the setting keeps rows.
	MaxDays = 1095
)

// Settings is the instance's one audit log setting; the stream secret is as the Owner typed it, a secret reference
// included, or as the server made it.
type Settings struct {
	// RetentionDays is 0 while unset, which keeps MinDays.
	RetentionDays int
	StreamURL     string
	StreamSecret  string
	StreamEnabled bool
	UpdatedBy     string
	UpdatedAt     time.Time
}

// Days is how long rows are kept under these settings.
func (s Settings) Days() int {
	return max(s.RetentionDays, MinDays)
}

// View is the settings as the Owner and MCP clients read them; the stream secret never leaves the server.
type View struct {
	// Unlocked is whether the license key unlocks audit right now; without it the saved days still hold and nothing streams.
	Unlocked      bool   `json:"unlocked"`
	RetentionDays int    `json:"retention_days"`
	StreamURL     string `json:"stream_url"`
	SecretSet     bool   `json:"stream_secret_set"`
	StreamEnabled bool   `json:"stream_enabled"`
	// Streaming is Unlocked and StreamEnabled: new audit rows are queued for the destination.
	Streaming bool `json:"streaming"`
}

// Saved is what a save answers the browser: the view, and the signing secret the server made in that call, shown once.
type Saved struct {
	View
	NewSecret string `json:"new_secret,omitempty"`
}

// Patch changes the settings; an omitted field keeps its value, the secret is write-only, and an empty stream URL
// removes the destination.
type Patch struct {
	RetentionDays *int    `json:"retention_days"`
	StreamURL     *string `json:"stream_url"`
	StreamSecret  *string `json:"stream_secret"`
	StreamEnabled *bool   `json:"stream_enabled"`
}

var errNeedsLicense = fmt.Errorf("%w: keeping the audit log longer and streaming it need a license key that unlocks audit; add one in Settings, Instance", apperrs.ErrForbidden)

func checkDays(days int) error {
	if days < MinDays || days > MaxDays {
		return fmt.Errorf("%w: retention_days is a whole number of days from %d to %d", apperrs.ErrInvalid, MinDays, MaxDays)
	}
	return nil
}

// checkStreamURL takes https destinations, and http only on a loopback address for trying a receiver out.
func checkStreamURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("%w: stream_url must be an absolute https URL with no user or password in it, such as https://logs.example.com/nexul", apperrs.ErrInvalid)
	}
	if u.Scheme == "https" {
		return nil
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		if u.Scheme == "http" {
			return nil
		}
	}
	return fmt.Errorf("%w: stream_url must use https", apperrs.ErrInvalid)
}
