package composite

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/ee/auditlog"
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

// SAMLSettings is the slice of the SAML use-cases instance_update calls.
type SAMLSettings interface {
	SAMLView(ctx context.Context) (sso.SAMLView, error)
	UpdateSAML(ctx context.Context, p sso.SAMLPatch) (sso.SAMLView, error)
	RemoveSAML(ctx context.Context) (sso.SAMLView, error)
	RotateSAMLCertificate(ctx context.Context) (sso.SAMLView, error)
	RetireSAMLCertificate(ctx context.Context) (sso.SAMLView, error)
}

// BrandingSettings is the slice of the branding use-cases instance_update calls.
type BrandingSettings interface {
	Get(ctx context.Context) (branding.Settings, error)
	Update(ctx context.Context, p branding.Patch) (branding.Settings, error)
	SetLogo(ctx context.Context, data []byte) (branding.Settings, error)
	RemoveLogo(ctx context.Context) (branding.Settings, error)
	Clear(ctx context.Context) (branding.Settings, error)
}

// AuditSettings is the slice of the audit log use-cases instance_update calls.
type AuditSettings interface {
	View(ctx context.Context) (auditlog.View, error)
	Update(ctx context.Context, p auditlog.Patch) (auditlog.Saved, error)
	Remove(ctx context.Context) (auditlog.View, error)
	RetryFailed(ctx context.Context) (auditlog.View, error)
}

// InstanceTools is instance_update, the one tool that changes the instance's own settings (ADR 0185): the enterprise
// license key, and the single sign-on provider, OIDC or SAML, branding and audit log settings that key unlocks.
func InstanceTools(l LicenseKeeper, s SSOSettings, m SAMLSettings, b BrandingSettings, a AuditSettings) []mcptool.Tool {
	return []mcptool.Tool{instanceUpdateTool(instanceSettingsOf{l: l, s: s, m: m, b: b, a: a})}
}

// instanceSettingsOf is every settings use-case instance_update reaches.
type instanceSettingsOf struct {
	l LicenseKeeper
	s SSOSettings
	m SAMLSettings
	b BrandingSettings
	a AuditSettings
}

type instanceUpdateIn struct {
	LicenseKey *string     `json:"license_key,omitempty" jsonschema:"A license key's text; an empty string removes the key."`
	SSO        *ssoIn      `json:"sso,omitempty" jsonschema:"The single sign-on provider; needs a key unlocking sso."`
	SAML       *samlIn     `json:"saml,omitempty" jsonschema:"The single sign-on provider when it speaks SAML; needs a key unlocking sso."`
	Branding   *brandingIn `json:"branding,omitempty" jsonschema:"The instance's own look; needs a key unlocking branding."`
	Audit      *auditIn    `json:"audit,omitempty" jsonschema:"How long the audit log is kept and where it streams; needs a key unlocking audit."`
}

type auditIn struct {
	RetentionDays *int    `json:"retention_days,omitempty" jsonschema:"Days audit rows are kept, 45 to 1095."`
	StreamURL     *string `json:"stream_url,omitempty" jsonschema:"The https URL each audit row is posted to, signed; empty removes it."`
	StreamSecret  *string `json:"stream_secret,omitempty" jsonschema:"The signing secret or a secret reference, never returned; left out, one is made that Settings shows on rotating."`
	StreamEnabled *bool   `json:"stream_enabled,omitempty" jsonschema:"True queues each new audit row for the destination."`
	RetryFailed   bool    `json:"retry_failed,omitempty" jsonschema:"True sends the dead deliveries, those that failed 8 times, again."`
	Remove        *bool   `json:"remove,omitempty" jsonschema:"True deletes the settings: rows are kept 45 days and nothing streams."`
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
	Only         *bool   `json:"only,omitempty" jsonschema:"True hides other sign-ins and ends non-SSO sessions but Owners'; an Owner needs a passkey first."`
	Remove       *bool   `json:"remove,omitempty" jsonschema:"True deletes the provider and brings the other sign-ins back."`
}

type samlIn struct {
	MetadataURL    *string `json:"metadata_url,omitempty" jsonschema:"The provider's https metadata URL, read now and daily; resend it to refresh."`
	MetadataXML    *string `json:"metadata_xml,omitempty" jsonschema:"The provider's metadata XML, instead of a URL."`
	EmailAttribute *string `json:"email_attribute,omitempty" jsonschema:"The attribute the email is read from; email when empty."`
	NameAttribute  *string `json:"name_attribute,omitempty" jsonschema:"The attribute the display name is read from; empty reads none."`
	DisplayName    *string `json:"display_name,omitempty" jsonschema:"The name on the button, up to 40 characters."`
	JoinRoleID     *string `json:"join_role_id,omitempty" jsonschema:"The default workspace's role a first sign-in gives, any but Owner."`
	Only           *bool   `json:"only,omitempty" jsonschema:"As sso's only."`
	Remove         *bool   `json:"remove,omitempty" jsonschema:"True deletes the SAML provider."`
	Rotate         bool    `json:"rotate,omitempty" jsonschema:"True makes a new Nexul certificate; the old stays until retire."`
	Retire         bool    `json:"retire,omitempty" jsonschema:"True drops the certificate rotate replaced."`
}

// instanceSettings is instance_update's answer, keyed so later instance settings sit beside these.
type instanceSettings struct {
	License  license.Status    `json:"license"`
	SSO      sso.View          `json:"sso"`
	SAML     sso.SAMLView      `json:"saml"`
	Branding branding.Settings `json:"branding"`
	Audit    auditlog.View     `json:"audit"`
}

func instanceUpdateTool(d instanceSettingsOf) mcptool.Tool {
	return mcptool.New("instance_update", "Update instance",
		"Changes the instance's own settings: its enterprise license key, single sign-on provider and branding. license_key replaces "+
			"the key, or removes it when empty (the features stop, what they made stays); a key for another address, with a bad "+
			"signature or expired is refused. sso saves the OIDC provider people sign in through, which needs a key unlocking sso; "+
			"the provider is reached before anything is stored. saml saves a SAML provider instead; an instance holds one, so "+
			"remove the other first. branding sets the product name, logo, accent, an imported VS Code "+
			"theme and css shown instead of Nexul's, which needs a key unlocking branding; "+
			"instance_get with styling_guide explains the css. audit keeps the audit log longer and streams each row to a signed "+
			"https webhook, which needs a key unlocking audit. Returns them all as instance_get shows them.",
		mcptool.Hints{Idempotent: true, Local: false},
		func(ctx context.Context, in instanceUpdateIn) (any, error) {
			var out instanceSettings
			steps := []step{}
			if in.LicenseKey != nil {
				steps = append(steps, step{field: "license_key", run: func(ctx context.Context) (err error) {
					if *in.LicenseKey == "" {
						out.License, err = d.l.Remove(ctx)
						return err
					}
					out.License, err = d.l.Set(ctx, *in.LicenseKey)
					return err
				}})
			}
			if in.SSO != nil {
				steps = append(steps, step{field: "sso", run: func(ctx context.Context) (err error) {
					if in.SSO.Remove != nil && *in.SSO.Remove {
						out.SSO, err = d.s.Remove(ctx)
						return err
					}
					out.SSO, err = d.s.Update(ctx, sso.Patch{
						IssuerURL: in.SSO.IssuerURL, ClientID: in.SSO.ClientID, ClientSecret: in.SSO.ClientSecret,
						DisplayName: in.SSO.DisplayName, JoinRoleID: in.SSO.JoinRoleID, Only: in.SSO.Only,
					})
					return err
				}})
			}
			if in.SAML != nil {
				steps = append(steps, step{field: "saml", run: func(ctx context.Context) (err error) {
					out.SAML, err = changeSAML(ctx, d.m, in.SAML)
					return err
				}})
			}
			if in.Branding != nil {
				steps = append(steps, step{field: "branding", run: func(ctx context.Context) (err error) {
					out.Branding, err = changeBranding(ctx, d.b, in.Branding)
					return err
				}})
			}
			if in.Audit != nil {
				steps = append(steps, step{field: "audit", run: func(ctx context.Context) (err error) {
					out.Audit, err = changeAudit(ctx, d.a, in.Audit)
					return err
				}})
			}
			if _, err := runSteps(ctx, steps); err != nil {
				return nil, err
			}
			return readBack(ctx, d, in, out)
		})
}

// readBack fills what the call did not change, so every answer carries both settings.
func readBack(ctx context.Context, d instanceSettingsOf, in instanceUpdateIn, out instanceSettings) (any, error) {
	var err error
	if in.LicenseKey == nil {
		if out.License, err = d.l.Status(ctx); err != nil {
			return nil, fmt.Errorf("read the license: %w", err)
		}
	}
	if in.SSO == nil {
		if out.SSO, err = d.s.View(ctx); err != nil {
			return nil, fmt.Errorf("read single sign-on: %w", err)
		}
	}
	if in.SAML == nil {
		if out.SAML, err = d.m.SAMLView(ctx); err != nil {
			return nil, fmt.Errorf("read saml: %w", err)
		}
	}
	if in.Branding == nil {
		if out.Branding, err = d.b.Get(ctx); err != nil {
			return nil, fmt.Errorf("read branding: %w", err)
		}
	}
	if in.Audit == nil {
		if out.Audit, err = d.a.View(ctx); err != nil {
			return nil, fmt.Errorf("read the audit log settings: %w", err)
		}
	}
	return out, nil
}

// changeAudit applies one audit patch; a secret the server made stays out of the answer, since a result never carries
// a secret.
func changeAudit(ctx context.Context, a AuditSettings, in *auditIn) (auditlog.View, error) {
	if in.Remove != nil && *in.Remove {
		return a.Remove(ctx)
	}
	patch := auditlog.Patch{RetentionDays: in.RetentionDays, StreamURL: in.StreamURL, StreamSecret: in.StreamSecret, StreamEnabled: in.StreamEnabled}
	if patch == (auditlog.Patch{}) && in.RetryFailed {
		return a.RetryFailed(ctx)
	}
	saved, err := a.Update(ctx, patch)
	if err != nil || !in.RetryFailed {
		return saved.View, err
	}
	return a.RetryFailed(ctx)
}

// changeSAML removes the SAML provider, rotates or retires Nexul's certificate, or saves the patch.
func changeSAML(ctx context.Context, m SAMLSettings, in *samlIn) (sso.SAMLView, error) {
	if in.Remove != nil && *in.Remove {
		return m.RemoveSAML(ctx)
	}
	if in.Rotate {
		return m.RotateSAMLCertificate(ctx)
	}
	if in.Retire {
		return m.RetireSAMLCertificate(ctx)
	}
	return m.UpdateSAML(ctx, sso.SAMLPatch{
		MetadataURL: in.MetadataURL, MetadataXML: in.MetadataXML, EmailAttribute: in.EmailAttribute, NameAttribute: in.NameAttribute,
		DisplayName: in.DisplayName, JoinRoleID: in.JoinRoleID, Only: in.Only,
	})
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
