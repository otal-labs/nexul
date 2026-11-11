package composite

import (
	"context"
	"fmt"

	"github.com/otal-labs/nexul/ee/license"
	"github.com/otal-labs/nexul/ee/sso"
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

// InstanceTools is instance_update, the one tool that changes the instance's own settings (ADR 0185): the enterprise
// license key, and the single sign-on provider that key unlocks.
func InstanceTools(l LicenseKeeper, s SSOSettings) []mcptool.Tool {
	return []mcptool.Tool{instanceUpdateTool(l, s)}
}

type instanceUpdateIn struct {
	LicenseKey *string `json:"license_key,omitempty" jsonschema:"A license key's text; an empty string removes the key."`
	SSO        *ssoIn  `json:"sso,omitempty" jsonschema:"The single sign-on provider; needs a key unlocking sso."`
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
	License license.Status `json:"license"`
	SSO     sso.View       `json:"sso"`
}

func instanceUpdateTool(l LicenseKeeper, s SSOSettings) mcptool.Tool {
	return mcptool.New("instance_update", "Update instance",
		"Changes the instance's own settings: its enterprise license key and its single sign-on provider. license_key replaces "+
			"the key, or removes it when empty (the features stop, what they made stays); a key for another address, with a bad "+
			"signature or expired is refused. sso saves the OIDC provider people sign in through, which needs a key unlocking sso; "+
			"the provider is reached before anything is stored. Returns the license and sso as instance_get shows them.",
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
			if _, err := runSteps(ctx, steps); err != nil {
				return nil, err
			}
			return readBack(ctx, l, s, in, out)
		})
}

// readBack fills what the call did not change, so every answer carries both settings.
func readBack(ctx context.Context, l LicenseKeeper, s SSOSettings, in instanceUpdateIn, out instanceSettings) (any, error) {
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
	return out, nil
}
