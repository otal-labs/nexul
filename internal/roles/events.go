package roles

const TopicUpdated = "role.updated"

func Topics() []string {
	return []string{TopicUpdated}
}

// RoleEvent is the payload of a role's name or permissions changing; every member holding it refetches what they hold.
type RoleEvent struct {
	RoleID      string `json:"role_id"`
	WorkspaceID string `json:"workspace_id"`
	ActorID     string `json:"actor_id,omitempty"`
}
