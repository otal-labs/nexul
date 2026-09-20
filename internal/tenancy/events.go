package tenancy

const (
	TopicInvitationCreated    = "invitation.created"
	TopicInvitationRevoked    = "invitation.revoked"
	TopicInvitationRedeemed   = "invitation.redeemed"
	TopicInvitationDeleted    = "invitation.deleted"
	TopicAccountAdmitted      = "account.admitted"
	TopicWorkspaceMemberAdded = "workspace.member.added"
)

func Topics() []string {
	return []string{
		TopicInvitationCreated,
		TopicInvitationRevoked,
		TopicInvitationRedeemed,
		TopicInvitationDeleted,
		TopicAccountAdmitted,
		TopicWorkspaceMemberAdded,
	}
}

type InvitationEvent struct {
	InvitationID string `json:"invitation_id"`
	ActorID      string `json:"actor_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	WorkspaceID  string `json:"workspace_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
}
