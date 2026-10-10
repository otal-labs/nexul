package connectors

import "context"

// AppConfig is the instance-wide, per-connector OAuth registration (CN3a), distinct from Credentials' user token.
type AppConfig struct {
	ConnectorID  string
	ClientID     string
	ClientSecret string
	// BaseURL is the API root for a self-hosted/enterprise provider; empty means the public default.
	BaseURL string
	// AppSlug is the provider-assigned GitHub App slug (T13b), needed to build the install+authorize URL.
	AppSlug string
	// PrivateKey is the GitHub App's PEM key, which Nexul signs as the App with (ADR 0144); write-only, never sent back.
	PrivateKey string `json:"-"`
}

// Configured reports whether both halves of the app credential are set.
func (c AppConfig) Configured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// AppConfigStatus is the safe, secret-free view of an AppConfig API responses return, never AppConfig directly.
type AppConfigStatus struct {
	Configured bool   `json:"configured"`
	ClientID   string `json:"client_id,omitempty"`
	BaseURL    string `json:"base_url,omitempty"`
	AppSlug    string `json:"app_slug,omitempty"`
	// PrivateKeySet says Nexul reads GitHub as the App, not as the connected account.
	PrivateKeySet bool `json:"private_key_set"`
}

// Status derives the safe, secret-free AppConfigStatus view.
func (c AppConfig) Status() AppConfigStatus {
	return AppConfigStatus{Configured: c.Configured(), ClientID: c.ClientID, BaseURL: c.BaseURL, AppSlug: c.AppSlug, PrivateKeySet: c.PrivateKey != ""}
}

// AppConfigStore is CredentialsStore's sibling for CN3a: the instance-wide app-level OAuth registration.
type AppConfigStore interface {
	// GetAppConfig returns the stored app config, or a zero value if never set; "never configured" is not an error here.
	GetAppConfig(ctx context.Context, connectorID string) (AppConfig, error)
	// SetAppConfig upserts the app config for setup or a rotated secret, leaving the private key as it is.
	SetAppConfig(ctx context.Context, c AppConfig) error
	// SetPrivateKey stores or, with "", removes a registered app's private key; ErrNotFound when none is registered.
	SetPrivateKey(ctx context.Context, connectorID, privateKey string) error
}
