package auth

const (
	TopicAccountAdmitted    = "account.admitted"
	TopicAccountDisabled    = "account.disabled"
	TopicAccountReactivated = "account.reactivated"
	TopicAccountRemoved     = "account.removed"
	TopicAccountRestored    = "account.restored"
)

// AccountLifecycleEvent is the durable payload for account state changes.
type AccountLifecycleEvent struct {
	AccountID string `json:"account_id"`
	ActorID   string `json:"actor_id,omitempty"`
}

// Topics lists account lifecycle events for the event catalog.
func Topics() []string {
	return []string{TopicAccountAdmitted, TopicAccountDisabled, TopicAccountReactivated, TopicAccountRemoved, TopicAccountRestored}
}
