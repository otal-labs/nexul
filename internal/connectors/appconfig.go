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
}

// Status derives the safe, secret-free AppConfigStatus view.
func (c AppConfig) Status() AppConfigStatus {
	return AppConfigStatus{Configured: c.Configured(), ClientID: c.ClientID, BaseURL: c.BaseURL, AppSlug: c.AppSlug}
}

// AppConfigStore is CredentialsStore's sibling for CN3a: the instance-wide app-level OAuth registration.
type AppConfigStore interface {
	// GetAppConfig returns the stored app config, or a zero value if never set; "never configured" is not an error here.
	GetAppConfig(ctx context.Context, connectorID string) (AppConfig, error)
	// SetAppConfig upserts the app config, for both first-time setup and rotating an existing secret.
	SetAppConfig(ctx context.Context, c AppConfig) error
}
