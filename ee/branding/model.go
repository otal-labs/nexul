// Package branding lets the instance Owner make the instance carry its own look: a product name and logo that replace
// Nexul's, an accent colour, and a VS Code color theme for the whole instance. It is an enterprise feature (ADR 0184), so
// without a key unlocking it the instance shows the default look and the saved branding stays.
package branding

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

const (
	// MaxNameRunes bounds the product name, which sits in a 220px sidebar header and a browser tab.
	MaxNameRunes = 40
	// MaxLogoBytes bounds an uploaded logo; a mark is a few kilobytes and the favicon is served from it.
	MaxLogoBytes = 512 << 10
)

// logoTypes are the image types a logo may be; SVG is served sandboxed, so it can never run a script.
var logoTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "image/svg+xml": true}

var accentPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Logo describes the stored logo without its bytes; Version changes whenever the bytes do, so a cached copy never goes stale.
type Logo struct {
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Version     string `json:"version"`
}

// Record is everything saved for the instance's branding.
type Record struct {
	ProductName string
	Accent      string
	Light       *Theme
	Dark        *Theme
	// CSS is the Owner's own style rules, applied after the colours; validateCSS keeps it from loading anything.
	CSS       string
	Logo      *Logo
	UpdatedBy string
	UpdatedAt time.Time
}

// Slot returns the saved theme written for mode.
func (r Record) Slot(mode Appearance) *Theme {
	if mode == Dark {
		return r.Dark
	}
	return r.Light
}

// Settings is the saved branding as the instance Owner and agents see it; Unlocked says whether a license key lets it show.
type Settings struct {
	Unlocked    bool    `json:"unlocked"`
	ProductName string  `json:"product_name"`
	Accent      string  `json:"accent"`
	Logo        *Logo   `json:"logo,omitempty"`
	Light       *Themed `json:"light_theme,omitempty"`
	Dark        *Themed `json:"dark_theme,omitempty"`
	CSS         string  `json:"css"`
}

// Themed is an imported theme with the notes on what was moved to keep it readable.
type Themed struct {
	Theme
	Adjusted []string `json:"adjusted,omitempty"`
}

func themed(t *Theme, accent string) *Themed {
	if t == nil {
		return nil
	}
	_, notes := t.Roles(accent)
	return &Themed{Theme: *t, Adjusted: notes}
}

// settings views a record for the Owner.
func (r Record) settings(unlocked bool) Settings {
	return Settings{Unlocked: unlocked, ProductName: r.ProductName, Accent: r.Accent, Logo: r.Logo, Light: themed(r.Light, r.Accent), Dark: themed(r.Dark, r.Accent), CSS: r.CSS}
}

// Look is the branding the web app paints, empty of everything when the instance has no key unlocking it. A mode's
// Roles hold the token values to set; Themed says they replace a person's own palette rather than layering on it.
type Look struct {
	Enabled     bool      `json:"enabled"`
	ProductName string    `json:"product_name"`
	LogoURL     string    `json:"logo_url"`
	Light       ModeRoles `json:"light"`
	Dark        ModeRoles `json:"dark"`
	CSS         string    `json:"css"`
	Version     string    `json:"version"`
}

// ModeRoles are the token values for one appearance.
type ModeRoles struct {
	Themed bool              `json:"themed"`
	Roles  map[string]string `json:"roles"`
}

// look is what the web app paints for r.
func (r Record) look() Look {
	l := Look{Enabled: true, ProductName: r.ProductName, Light: r.modeRoles(Light), Dark: r.modeRoles(Dark), CSS: r.CSS}
	if r.Logo != nil {
		l.LogoURL = "/api/branding/logo?v=" + r.Logo.Version
	}
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%s|%v|%v|%s|%v|%d", r.ProductName, r.Accent, r.Light, r.Dark, r.CSS, r.Logo, r.UpdatedAt.Unix()))
	l.Version = hex.EncodeToString(sum[:6])
	return l
}

func (r Record) modeRoles(mode Appearance) ModeRoles {
	if t := r.Slot(mode); t != nil {
		roles, _ := t.Roles(r.Accent)
		return ModeRoles{Themed: true, Roles: roles}
	}
	if r.Accent == "" {
		return ModeRoles{Roles: map[string]string{}}
	}
	return ModeRoles{Roles: accentRoles(r.Accent, mode)}
}

// Patch is a change to the saved branding; a nil field keeps its value and an empty string removes it.
type Patch struct {
	ProductName *string
	Accent      *string
	// Theme is a VS Code theme file's text; it replaces the saved theme written for the same appearance.
	Theme *string
	// RemoveTheme names appearances to clear, or "all".
	RemoveTheme string
	// CSS replaces the custom style rules; an empty string removes them.
	CSS *string
}

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return "", fmt.Errorf("%w: the product name is at most %d characters", apperrs.ErrInvalid, MaxNameRunes)
	}
	if strings.ContainsAny(name, "<>\n\r\t") {
		return "", fmt.Errorf("%w: the product name is plain text on one line", apperrs.ErrInvalid)
	}
	return name, nil
}

func validateAccent(accent string) (string, error) {
	accent = strings.TrimSpace(accent)
	if accent != "" && !accentPattern.MatchString(accent) {
		return "", fmt.Errorf("%w: the accent is a color written as #rrggbb", apperrs.ErrInvalid)
	}
	return strings.ToLower(accent), nil
}

func validateLogo(contentType string, data []byte) error {
	if !logoTypes[contentType] {
		return fmt.Errorf("%w: a logo is a PNG, JPEG, WebP or SVG image", apperrs.ErrInvalid)
	}
	if len(data) == 0 {
		return fmt.Errorf("%w: the logo file is empty", apperrs.ErrInvalid)
	}
	if len(data) > MaxLogoBytes {
		return fmt.Errorf("%w: a logo is at most %d KB", apperrs.ErrInvalid, MaxLogoBytes>>10)
	}
	return nil
}
