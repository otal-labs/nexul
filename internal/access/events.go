package access

const TopicGrantChanged = "access.grant.changed"

func Topics() []string {
	return []string{TopicGrantChanged}
}

// GrantEvent is the payload of one person's grant on a doc or play changing; it reaches that person, whose open reads refetch.
type GrantEvent struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	UserID       string `json:"user_id"`
	ActorID      string `json:"actor_id,omitempty"`
}
