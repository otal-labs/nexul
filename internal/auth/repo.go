package auth

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// IdentityStore persists the provider accounts a user signs in with; every write is user-scoped.
type IdentityStore interface {
	// UpsertUser syncs a known identity (and the user's profile when it is their first) or creates the user with it.
	UpsertUser(ctx context.Context, id *Identity) (*User, bool, error)
	CreateFirstUser(ctx context.Context, id *Identity, events ...eventbus.OutboxEvent) (*User, error)
	GetUserByProvider(ctx context.Context, provider Provider, providerUserID string) (*User, error)
	ListIdentities(ctx context.Context, userID string) ([]Identity, error)
	// LinkIdentity attaches one more provider account; ErrConflict when it belongs to anyone already.
	LinkIdentity(ctx context.Context, id *Identity, events ...eventbus.OutboxEvent) error
	// UnlinkIdentity detaches a provider account; ErrConflict for the user's last one, so nobody locks themselves out.
	UnlinkIdentity(ctx context.Context, userID string, provider Provider, events ...eventbus.OutboxEvent) error
}

// UserStore persists user records.
type UserStore interface {
	IdentityStore
	GetUserByID(ctx context.Context, id string) (*User, error)
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
	SetProfileOverride(ctx context.Context, id string, displayName, avatarOverrideURL *string, events ...eventbus.OutboxEvent) error
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
}

// SetupCodeStore keeps the setup code's hash; Replace leaves exactly the one code, Clear leaves none.
type SetupCodeStore interface {
	ReplaceSetupCode(ctx context.Context, hash string, createdAt, expiresAt time.Time) error
	SetupCodeValid(ctx context.Context, hash string, now time.Time) (bool, error)
	ClearSetupCodes(ctx context.Context) error
}

// ConnectCodeStore keeps connect code hashes; Replace leaves the user exactly one code, Consume deletes it and names its user.
type ConnectCodeStore interface {
	ReplaceConnectCode(ctx context.Context, userID, hash string, createdAt, expiresAt time.Time) error
	// ConsumeConnectCode returns the code's user and deletes it; a missing or expired code is ErrNotFound.
	ConsumeConnectCode(ctx context.Context, hash string, now time.Time) (string, error)
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

// SessionStore persists sessions; only the hash reaches the store; deletes are user-scoped so nobody signs out another's device.
type SessionStore interface {
	CreateSession(ctx context.Context, s *Session, evts ...eventbus.OutboxEvent) error
	GetSessionByHash(ctx context.Context, hash string) (*Session, error)
	ListSessionsByUser(ctx context.Context, userID string) ([]Session, error)
	DeleteSession(ctx context.Context, id, userID string, evts ...eventbus.OutboxEvent) error
	DeleteOtherSessions(ctx context.Context, userID, keepID string, evts ...eventbus.OutboxEvent) error
	TouchSession(ctx context.Context, id string, lastActive time.Time, ip string, expiresAt time.Time) error
	DeleteExpiredSessions(ctx context.Context, userID string, now time.Time) error
	SetSessionPushToken(ctx context.Context, id, userID, token string) error
	// LastActiveByUser is each user's latest last-active time across their sessions.
	LastActiveByUser(ctx context.Context) (map[string]time.Time, error)
}
