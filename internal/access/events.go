package access

import "github.com/otal-labs/nexul/internal/platform/eventbus"

const TopicGrantChanged = "access.grant.changed"

func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicGrantChanged, Payload: GrantEvent{}}}
}

// GrantEvent is the payload of one person's grant on a doc or play changing; it reaches that person, whose open reads refetch.
type GrantEvent struct {
	ResourceType string `json:"resource_type" enum:"doc,play,project"`
	ResourceID   string `json:"resource_id"`
	UserID       string `json:"user_id"`
	ActorID      string `json:"actor_id,omitempty"`
}
