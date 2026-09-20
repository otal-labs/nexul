package auth

import "time"

// Provider identifies a git/identity provider; the user model stays provider-agnostic for future ones.
type Provider string

const (
	ProviderGitHub Provider = "github"
	// ProviderGoogle is the second sign-in provider (ADR 0040), for a Google account with no GitHub; login is the email.
	ProviderGoogle Provider = "google"
	// ProviderDiscord is the third sign-in provider (ADR 0040), same email-keyed shape as Google.
	ProviderDiscord Provider = "discord"
	// ProviderDev identifies the local-dev session minted by DevLogin, never valid outside a dev-enabled instance.
	ProviderDev Provider = "dev"
)

// AccountStatus controls whether a provider identity may authenticate or use the instance.
type AccountStatus string

const (
	AccountActive   AccountStatus = "active"
	AccountDisabled AccountStatus = "disabled"
	AccountRemoved  AccountStatus = "removed"
)

// User is the persistent identity record, keyed by provider + user ID because a login can be renamed; login/name/avatar sync each sign-in.
type User struct {
	ID             string   `json:"id"`
	Provider       Provider `json:"provider"`
	ProviderUserID string   `json:"provider_user_id"`
	Login          string   `json:"login"`
	Name           string   `json:"name"`
	AvatarURL      string   `json:"avatar_url"`
	// CanCreateWorkspace is the instance-level bit gating workspace creation and, by default, settings/allowlist.
	CanCreateWorkspace bool          `json:"can_create_workspace"`
	FirstLoginDone     bool          `json:"first_login_done"`
	AccountStatus      AccountStatus `json:"account_status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`

	// DisplayName and AvatarOverrideURL are the manual override, untouched by UpsertUser's sync, surviving sign-in.
	DisplayName       *string `json:"display_name,omitempty"`
	AvatarOverrideURL *string `json:"avatar_override_url,omitempty"`
}

// InvitationAcceptance is the non-secret invitation detail shown after preview or OAuth authentication.
type InvitationAcceptance struct {
	InvitationID      string                       `json:"invitation_id"`
	InstanceName      string                       `json:"instance_name"`
	InstanceURL       string                       `json:"instance_url"`
	ExpiresAt         time.Time                    `json:"expires_at"`
	Grants            []InvitationGrant            `json:"grants"`
	AuthenticatedUser *InvitationAuthenticatedUser `json:"authenticated_user,omitempty"`
	AcceptanceToken   string                       `json:"acceptance_token,omitempty"`
}

// InvitationGrant mirrors tenancy grant data without importing tenancy into auth.
type InvitationGrant struct {
	WorkspaceID   string   `json:"workspace_id"`
	WorkspaceName string   `json:"workspace_name"`
	RoleID        string   `json:"role_id"`
	RoleName      string   `json:"role_name"`
	Allow         []string `json:"allow"`
	Deny          []string `json:"deny"`
}

// InvitationAuthenticatedUser is provider identity or an admitted user shown on acceptance.
type InvitationAuthenticatedUser struct {
	ID        string   `json:"id,omitempty"`
	Provider  Provider `json:"provider"`
	Login     string   `json:"login"`
	Name      string   `json:"name"`
	AvatarURL string   `json:"avatar_url"`
}

// ProviderUser is the identity a provider returns: ID is its stable key, Login is what the allowlist matches.
type ProviderUser struct {
	ID        string
	Login     string
	Name      string
	AvatarURL string
}

type OAuthHandoff struct {
	ID             string     `json:"-"`
	InvitationID   string     `json:"-"`
	OAuthStateHash string     `json:"-"`
	AcceptanceHash string     `json:"-"`
	Provider       Provider   `json:"provider"`
	ProviderUserID string     `json:"-"`
	Login          string     `json:"login"`
	Name           string     `json:"name"`
	AvatarURL      string     `json:"avatar_url"`
	ExistingUserID string     `json:"-"`
	AdmittedUserID string     `json:"-"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
}

type OAuthHandoffIdentity struct {
	Provider       Provider
	ProviderUserID string
	Login          string
	Name           string
	AvatarURL      string
	ExistingUserID string
}

// InvitationOAuthStart contains the redirect and state cookie material for an invitation flow.
type InvitationOAuthStart struct {
	URL        string
	State      string
	CookieName string
}

// InvitationIdentity carries the provider identity into atomic invitation redemption.
type InvitationIdentity struct {
	ID             string
	Provider       Provider
	ProviderUserID string
	Login          string
	Name           string
	AvatarURL      string
}

// InvitationAdmission reports the user admitted by an invitation.
type InvitationAdmission struct {
	UserID       string
	Created      bool
	WorkspaceIDs []string
}

// InvitationRedeemResult is the normal session plus the workspaces granted by redemption.
type InvitationRedeemResult struct {
	Token        string   `json:"token"`
	WorkspaceIDs []string `json:"workspace_ids"`
}

// GitHubUser is ProviderUser under its original name.
type GitHubUser = ProviderUser

// Settings is the workspace-level instance configuration; SettingsVersion bumps on instance URL change.
type Settings struct {
	InstanceURL         string
	SettingsVersion     int
	GitHubOAuthClientID string
	// GitHubOAuthClientSecret must never be logged or serialized directly; build response bodies field-by-field instead.
	GitHubOAuthClientSecret string
	// GoogleOAuthClientID/Secret are the optional provider; both empty means no Google button; secret like GitHub's.
	GoogleOAuthClientID      string
	GoogleOAuthClientSecret  string
	DiscordOAuthClientID     string
	DiscordOAuthClientSecret string
	// MentionChipTemplate is the mention chip layout template with {ticket.Field}; doesn't bump SettingsVersion.
	MentionChipTemplate string
}

// Configured reports whether a GitHub OAuth App has been stored via the database bootstrap flow, rather than env vars.
func (s Settings) Configured() bool {
	return s.GitHubOAuthClientID != "" && s.GitHubOAuthClientSecret != ""
}

// OAuthCredentials returns the stored client ID/secret for a sign-in provider; unknown providers yield empty strings.
func (s Settings) OAuthCredentials(p Provider) (clientID, clientSecret string) {
	switch p {
	case ProviderGitHub:
		return s.GitHubOAuthClientID, s.GitHubOAuthClientSecret
	case ProviderGoogle:
		return s.GoogleOAuthClientID, s.GoogleOAuthClientSecret
	case ProviderDiscord:
		return s.DiscordOAuthClientID, s.DiscordOAuthClientSecret
	}
	return "", ""
}

// ProviderConfigured reports whether p's sign-in is offered (ADR 0040); optional providers only gate their login button.
func (s Settings) ProviderConfigured(p Provider) bool {
	id, secret := s.OAuthCredentials(p)
	return id != "" && secret != ""
}

// OnboardingStatus is what GET /api/auth/me returns: the current user plus which wizard, if any, must run first.
type OnboardingStatus struct {
	User                  *User `json:"user"`
	NeedsOwnerWizard      bool  `json:"needs_owner_wizard"`
	NeedsFirstLoginWizard bool  `json:"needs_first_login_wizard"`
}

// ConnectionToken is a signed JWT a client imports to learn how to connect; no identity or credentials, not secret.
type ConnectionToken struct {
	Token           string    `json:"token"`
	InstanceURL     string    `json:"instance_url"`
	MCPURL          string    `json:"mcp_url,omitempty"`
	SettingsVersion int       `json:"settings_version"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// ConnectionTokenClaims is a token's decoded payload; MCPURL points the client at the running MCP server.
type ConnectionTokenClaims struct {
	InstanceURL string `json:"instance_url"`
	MCPURL      string `json:"mcp_url,omitempty"`
	Version     int    `json:"version"`
	Iat         int64  `json:"iat"`
	Exp         int64  `json:"exp"`
}

// PersonalAccessToken only stores its hash; the raw value is shown once.
type PersonalAccessToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	TokenHash  string     `json:"-"`
}

// LoginMatch is one GitHub login suggestion for the allowlist typeahead (ADR 0040); only GitHub exposes a user-search API.
type LoginMatch struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}
