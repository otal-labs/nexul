package tenancy

const (
	TopicInvitationCreated      = "invitation.created"
	TopicInvitationRevoked      = "invitation.revoked"
	TopicInvitationRedeemed     = "invitation.redeemed"
	TopicInvitationDeleted      = "invitation.deleted"
	TopicAccountAdmitted        = "account.admitted"
	TopicWorkspaceMemberAdded   = "workspace.member.added"
	TopicWorkspaceMemberRemoved = "workspace.member.removed"
	TopicWorkspaceMemberUpdated = "workspace.member.updated"
	TopicWorkspaceUpdated       = "workspace.updated"
)

func Topics() []string {
	return []string{
		TopicInvitationCreated,
		TopicInvitationRevoked,
		TopicInvitationRedeemed,
		TopicInvitationDeleted,
		TopicAccountAdmitted,
		TopicWorkspaceMemberAdded,
		TopicWorkspaceMemberRemoved,
		TopicWorkspaceMemberUpdated,
		TopicWorkspaceUpdated,
	}
}

type InvitationEvent struct {
	InvitationID string `json:"invitation_id"`
	ActorID      string `json:"actor_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	WorkspaceID  string `json:"workspace_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// MemberEvent is the payload of a membership change made outside an invitation; updated covers a role or overrides change.
type MemberEvent struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	ActorID     string `json:"actor_id,omitempty"`
}

// WorkspaceEvent is the payload of a rename or slug change; it carries the new name and slug so a client open on the workspace can move its URL.
type WorkspaceEvent struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ActorID     string `json:"actor_id,omitempty"`
}
