package auth

import (
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const (
	TopicAccountAdmitted    = "account.admitted"
	TopicAccountDisabled    = "account.disabled"
	TopicAccountReactivated = "account.reactivated"
	TopicAccountRemoved     = "account.removed"
	TopicAccountRestored    = "account.restored"
	TopicProfileUpdated     = "account.profile_updated"
	TopicTokenMinted        = "personal_access_token.minted"
	TopicTokenRevoked       = "personal_access_token.revoked"
	TopicSessionCreated     = "session.created"
	TopicSessionRevoked     = "session.revoked"
	TopicIdentityLinked     = "identity.linked"
	TopicIdentityUnlinked   = "identity.unlinked"
	TopicGitHubLinkChanged  = "github_link.changed"
)

// GitHubLinkEvent is a person's GitHub link being made, needing a reconnect, or forgotten; it never carries a token.
type GitHubLinkEvent struct {
	UserID string          `json:"user_id"`
	State  GitHubLinkState `json:"state" enum:"connected,reconnect,none" jsonschema:"What the link is now: connected lists the person's repositories, reconnect needs them to connect GitHub again, none is no link."`
}

func githubLinkEvent(userID string, state GitHubLinkState) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicGitHubLinkChanged, Payload: GitHubLinkEvent{UserID: userID, State: state}}
}

// IdentityChangedEvent is the payload for both identity topics; it never carries the provider's own user id.
type IdentityChangedEvent struct {
	UserID   string   `json:"user_id"`
	Provider Provider `json:"provider"`
	Login    string   `json:"login"`
}

// AccountLifecycleEvent is the durable payload for account state changes.
type AccountLifecycleEvent struct {
	AccountID string `json:"account_id"`
	ActorID   string `json:"actor_id,omitempty"`
}

// TokenChangedEvent is the payload for both token topics; it never carries the token or its hash.
type TokenChangedEvent struct {
	TokenID    string `json:"token_id"`
	UserID     string `json:"user_id"`
	Name       string `json:"name"`
	ComputerID string `json:"computer_id,omitempty"`
}

// SessionChangedEvent is the payload for both session topics; it never carries the token or its hash.
type SessionChangedEvent struct {
	SessionID string        `json:"session_id"`
	UserID    string        `json:"user_id"`
	Client    SessionClient `json:"client" enum:"browser,desktop,phone"`
	Platform  string        `json:"platform"`
	Label     string        `json:"label"`
}

// Topics lists account, personal access token, session and identity lifecycle events for the event catalog.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicAccountAdmitted, Payload: AccountLifecycleEvent{}},
		{Name: TopicAccountDisabled, Payload: AccountLifecycleEvent{}},
		{Name: TopicAccountReactivated, Payload: AccountLifecycleEvent{}},
		{Name: TopicAccountRemoved, Payload: AccountLifecycleEvent{}},
		{Name: TopicAccountRestored, Payload: AccountLifecycleEvent{}},
		{Name: TopicProfileUpdated, Payload: AccountLifecycleEvent{}},
		{Name: TopicTokenMinted, Payload: TokenChangedEvent{}},
		{Name: TopicTokenRevoked, Payload: TokenChangedEvent{}},
		{Name: TopicSessionCreated, Payload: SessionChangedEvent{}},
		{Name: TopicSessionRevoked, Payload: SessionChangedEvent{}},
		{Name: TopicIdentityLinked, Payload: IdentityChangedEvent{}},
		{Name: TopicIdentityUnlinked, Payload: IdentityChangedEvent{}},
		{Name: TopicGitHubLinkChanged, Payload: GitHubLinkEvent{}},
	}
}
