package auth

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// UserStore persists user records; UpsertUser returns the record, permission/sync columns included, plus created.
type UserStore interface {
	UpsertUser(ctx context.Context, u *User) (*User, bool, error)
	CreateFirstUser(ctx context.Context, u *User, events ...eventbus.OutboxEvent) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByProvider(ctx context.Context, provider Provider, providerUserID string) (*User, error)
	// GetUserByLogin looks up a user by login (ErrNotFound if none); tenancy checks if a login already has a User.
	GetUserByLogin(ctx context.Context, login string) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)
	CanCreateWorkspaceExists(ctx context.Context) (bool, error)
	SetCanCreateWorkspace(ctx context.Context, id string, can bool) error
	SetAccountStatus(ctx context.Context, id string, status AccountStatus, events ...eventbus.OutboxEvent) error
	CountUsers(ctx context.Context) (int, error)
	CountActiveAdmins(ctx context.Context) (int, error)
	MarkFirstLoginDone(ctx context.Context, id string) error
	// SetProfileOverride sets the caller's profile override; nil clears to provider-sourced; UpsertUser never calls this.
	SetProfileOverride(ctx context.Context, id string, displayName, avatarOverrideURL *string) error
}

type OAuthHandoffStore interface {
	StartOAuthHandoff(ctx context.Context, handoff *OAuthHandoff) error
	CompleteOAuthCallback(ctx context.Context, oauthStateHash, acceptanceHash string, identity OAuthHandoffIdentity, expiresAt, now time.Time) (*OAuthHandoff, error)
	GetOAuthHandoffByAcceptanceHash(ctx context.Context, acceptanceHash string, now time.Time) (*OAuthHandoff, error)
	CompleteOAuthRedemption(ctx context.Context, acceptanceHash, admittedUserID string, now time.Time) error
}

// InvitationGate is the auth-side slice of invitation storage. Raw credentials are hashed before they cross this seam.
type InvitationGate interface {
	GetInvitationByToken(ctx context.Context, rawToken string, now time.Time) (*InvitationAcceptance, error)
	GetInvitationByAcceptance(ctx context.Context, acceptanceHash string, now time.Time) (*InvitationAcceptance, error)
	RedeemInvitation(ctx context.Context, acceptanceHash string, identity InvitationIdentity, now time.Time, events ...eventbus.OutboxEvent) (InvitationAdmission, error)
}

// DefaultWorkspaceBinder is tenancy's slice the Owner Wizard needs (ADR 0017) to bind its user to the default workspace.
type DefaultWorkspaceBinder interface {
	BindDefaultWorkspaceOwner(ctx context.Context, userID string) error
}

// ConnectorAppSeeder is connectors' app-config write (ADR 0017); Bootstrap seeds github's registration before Settings.
type ConnectorAppSeeder interface {
	SeedGitHubApp(ctx context.Context, clientID, clientSecret, appSlug string) error
}

// PendingInviteResolver is tenancy's pending-invite resolution (ADR 0017), called the moment a new User row is created.
type PendingInviteResolver interface {
	ResolvePendingInvites(ctx context.Context, login, userID string) error
}

// AllowlistStore persists the sign-in allowlist (ADR 0040): usernames or emails, lowercased by the caller.
type AllowlistStore interface {
	Add(ctx context.Context, login string) error
	Remove(ctx context.Context, login string) error
	Contains(ctx context.Context, login string) (bool, error)
	List(ctx context.Context) ([]string, error)
}

// SettingsStore persists instance settings; Set bumps the version so tokens regenerate; SetGitHubOAuth doesn't.
type SettingsStore interface {
	Get(ctx context.Context) (Settings, error)
	Set(ctx context.Context, instanceURL string) (Settings, error)
	SetGitHubOAuth(ctx context.Context, clientID, clientSecret string) (Settings, error)
	// SetProviderOAuth persists sign-in credentials; both empty disables it; same no-version-bump rule as SetGitHubOAuth.
	SetProviderOAuth(ctx context.Context, provider Provider, clientID, clientSecret string) (Settings, error)
	// SetMentionChipTemplate persists the chip template; doesn't bump SettingsVersion (see Settings.MentionChipTemplate).
	SetMentionChipTemplate(ctx context.Context, template string) (Settings, error)
}

// MentionLayoutGate is access's HasPermission check (ADR 0017), gating SetMentionChipTemplate on workspaces:write.
type MentionLayoutGate interface {
	CanManageMentionLayout(ctx context.Context, userID string) bool
}

// PATStore persists PATs; only the hash reaches the store; Revoke is user-scoped, a repeat is ErrNotFound.
type PATStore interface {
	Create(ctx context.Context, pat *PersonalAccessToken, evts ...eventbus.OutboxEvent) error
	GetByHash(ctx context.Context, hash string) (*PersonalAccessToken, error)
	ListByUser(ctx context.Context, userID string) ([]PersonalAccessToken, error)
	Revoke(ctx context.Context, id, userID string, evts ...eventbus.OutboxEvent) error
	TouchLastUsed(ctx context.Context, id string) error
	// GetActiveForComputer returns the computer's unrevoked token, or ErrNotFound.
	GetActiveForComputer(ctx context.Context, userID, computerID string) (*PersonalAccessToken, error)
}
