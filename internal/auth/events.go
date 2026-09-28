package auth

const (
	TopicAccountAdmitted    = "account.admitted"
	TopicAccountDisabled    = "account.disabled"
	TopicAccountReactivated = "account.reactivated"
	TopicAccountRemoved     = "account.removed"
	TopicAccountRestored    = "account.restored"
	TopicTokenMinted        = "personal_access_token.minted"
	TopicTokenRevoked       = "personal_access_token.revoked"
	TopicSessionCreated     = "session.created"
	TopicSessionRevoked     = "session.revoked"
)

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
	Client    SessionClient `json:"client"`
	Platform  string        `json:"platform"`
	Label     string        `json:"label"`
}

// Topics lists account, personal access token and session lifecycle events for the event catalog.
func Topics() []string {
	return []string{
		TopicAccountAdmitted, TopicAccountDisabled, TopicAccountReactivated, TopicAccountRemoved, TopicAccountRestored,
		TopicTokenMinted, TopicTokenRevoked, TopicSessionCreated, TopicSessionRevoked,
	}
}
