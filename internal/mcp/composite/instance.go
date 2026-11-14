package composite

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/ee/branding"
	"github.com/otal-labs/nexul/ee/license"
	"github.com/otal-labs/nexul/ee/sso"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// LicenseKeeper is the slice of the license use-cases instance_update calls.
type LicenseKeeper interface {
	Status(ctx context.Context) (license.Status, error)
	Set(ctx context.Context, key string) (license.Status, error)
	Remove(ctx context.Context) (license.Status, error)
}

// SSOSettings is the slice of the single sign-on use-cases instance_update calls.
type SSOSettings interface {
	View(ctx context.Context) (sso.View, error)
	Update(ctx context.Context, p sso.Patch) (sso.View, error)
	Remove(ctx context.Context) (sso.View, error)
}

// BrandingSettings is the slice of the branding use-cases instance_update calls.
type BrandingSettings interface {
	Get(ctx context.Context) (branding.Settings, error)
	Update(ctx context.Context, p branding.Patch) (branding.Settings, error)
	SetLogo(ctx context.Context, data []byte) (branding.Settings, error)
	RemoveLogo(ctx context.Context) (branding.Settings, error)
	Clear(ctx context.Context) (branding.Settings, error)
}

// InstanceTools is instance_update, the one tool that changes the instance's own settings (ADR 0185): the enterprise
// license key, and the single sign-on provider and branding that key unlocks.
func InstanceTools(l LicenseKeeper, s SSOSettings, b BrandingSettings) []mcptool.Tool {
	return []mcptool.Tool{instanceUpdateTool(l, s, b)}
}

type instanceUpdateIn struct {
	LicenseKey *string     `json:"license_key,omitempty" jsonschema:"A license key's text; an empty string removes the key."`
	SSO        *ssoIn      `json:"sso,omitempty" jsonschema:"The single sign-on provider; needs a key unlocking sso."`
	Branding   *brandingIn `json:"branding,omitempty" jsonschema:"The instance's own look; needs a key unlocking branding."`
}

type brandingIn struct {
	ProductName *string `json:"product_name,omitempty" jsonschema:"Shown instead of Nexul, 40 characters at most; empty goes back."`
	Accent      *string `json:"accent,omitempty" jsonschema:"The accent as #rrggbb, moved if needed to stay readable; empty goes back."`
	Logo        *string `json:"logo,omitempty" jsonschema:"A PNG, JPEG, WebP or SVG up to 512 KB as a base64 data URL; empty removes it."`
	ThemeFile   *string `json:"theme_file,omitempty" jsonschema:"A VS Code color theme file's text, replacing the instance theme of its appearance."`
	RemoveTheme string  `json:"remove_theme,omitempty" jsonschema:"light, dark or all: removes that theme."`
	CSS         *string `json:"css,omitempty" jsonschema:"Style rules for every page; empty removes them."`
	Clear       bool    `json:"clear,omitempty" jsonschema:"True removes all branding."`
}

type ssoIn struct {
	IssuerURL    *string `json:"issuer_url,omitempty" jsonschema:"The OIDC issuer URL, such as https://login.example.com."`
	ClientID     *string `json:"client_id,omitempty" jsonschema:"The client id registered for this instance."`
	ClientSecret *string `json:"client_secret,omitempty" jsonschema:"The client secret or a secret reference; never returned."`
	DisplayName  *string `json:"display_name,omitempty" jsonschema:"The name on the button, Sign in with <name>, up to 40 characters."`
	JoinRoleID   *string `json:"join_role_id,omitempty" jsonschema:"The default workspace's role a first sign-in gives, any but Owner."`
	Only         *bool   `json:"only,omitempty" jsonschema:"True hides every other sign-in; the Owner's passkey still works, so an Owner needs one first."`
	Remove       *bool   `json:"remove,omitempty" jsonschema:"True deletes the provider and brings the other sign-ins back."`
}

// instanceSettings is instance_update's answer, keyed so later instance settings sit beside these.
type instanceSettings struct {
	License  license.Status    `json:"license"`
	SSO      sso.View          `json:"sso"`
	Branding branding.Settings `json:"branding"`
}

func instanceUpdateTool(l LicenseKeeper, s SSOSettings, b BrandingSettings) mcptool.Tool {
	return mcptool.New("instance_update", "Update instance",
		"Changes the instance's own settings: its enterprise license key, single sign-on provider and branding. license_key replaces "+
			"the key, or removes it when empty (the features stop, what they made stays); a key for another address, with a bad "+
			"signature or expired is refused. sso saves the OIDC provider people sign in through, which needs a key unlocking sso; "+
			"the provider is reached before anything is stored. branding sets the product name, logo, accent, an imported VS Code "+
			"theme and css shown instead of Nexul's, which needs a key unlocking branding; "+
			"instance_get with styling_guide explains the css. Returns all three as instance_get shows them.",
		mcptool.Hints{Idempotent: true, Local: false},
		func(ctx context.Context, in instanceUpdateIn) (any, error) {
			var out instanceSettings
			steps := []step{}
			if in.LicenseKey != nil {
				steps = append(steps, step{field: "license_key", run: func(ctx context.Context) (err error) {
					if *in.LicenseKey == "" {
						out.License, err = l.Remove(ctx)
						return err
					}
					out.License, err = l.Set(ctx, *in.LicenseKey)
					return err
				}})
			}
			if in.SSO != nil {
				steps = append(steps, step{field: "sso", run: func(ctx context.Context) (err error) {
					if in.SSO.Remove != nil && *in.SSO.Remove {
						out.SSO, err = s.Remove(ctx)
						return err
					}
					out.SSO, err = s.Update(ctx, sso.Patch{
						IssuerURL: in.SSO.IssuerURL, ClientID: in.SSO.ClientID, ClientSecret: in.SSO.ClientSecret,
						DisplayName: in.SSO.DisplayName, JoinRoleID: in.SSO.JoinRoleID, Only: in.SSO.Only,
					})
					return err
				}})
			}
			if in.Branding != nil {
				steps = append(steps, step{field: "branding", run: func(ctx context.Context) (err error) {
					out.Branding, err = changeBranding(ctx, b, in.Branding)
					return err
				}})
			}
			if _, err := runSteps(ctx, steps); err != nil {
				return nil, err
			}
			return readBack(ctx, l, s, b, in, out)
		})
}

// readBack fills what the call did not change, so every answer carries both settings.
func readBack(ctx context.Context, l LicenseKeeper, s SSOSettings, b BrandingSettings, in instanceUpdateIn, out instanceSettings) (any, error) {
	var err error
	if in.LicenseKey == nil {
		if out.License, err = l.Status(ctx); err != nil {
			return nil, fmt.Errorf("read the license: %w", err)
		}
	}
	if in.SSO == nil {
		if out.SSO, err = s.View(ctx); err != nil {
			return nil, fmt.Errorf("read single sign-on: %w", err)
		}
	}
	if in.Branding == nil {
		if out.Branding, err = b.Get(ctx); err != nil {
			return nil, fmt.Errorf("read branding: %w", err)
		}
	}
	return out, nil
}

// changeBranding applies one branding patch: clear alone, else the name, accent and themes, then the logo.
func changeBranding(ctx context.Context, b BrandingSettings, in *brandingIn) (branding.Settings, error) {
	if in.Clear {
		return b.Clear(ctx)
	}
	var out branding.Settings
	patch := branding.Patch{ProductName: in.ProductName, Accent: in.Accent, Theme: in.ThemeFile, RemoveTheme: in.RemoveTheme, CSS: in.CSS}
	if patch != (branding.Patch{}) {
		var err error
		if out, err = b.Update(ctx, patch); err != nil {
			return out, err
		}
	}
	if in.Logo == nil {
		return out, nil
	}
	if *in.Logo == "" {
		return b.RemoveLogo(ctx)
	}
	data, err := decodeDataURL(*in.Logo)
	if err != nil {
		return out, err
	}
	return b.SetLogo(ctx, data)
}

// decodeDataURL reads the bytes of a base64 data URL; the image type is judged from the bytes, never from the header.
func decodeDataURL(s string) ([]byte, error) {
	head, body, ok := strings.Cut(s, ",")
	if !ok || !strings.HasPrefix(head, "data:") || !strings.HasSuffix(head, ";base64") {
		return nil, fmt.Errorf("%w: the logo is a base64 data URL, data:image/png;base64 and then the image", apperrs.ErrInvalid)
	}
	data, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("%w: the logo's base64 text is not valid", apperrs.ErrInvalid)
	}
	return data, nil
}
