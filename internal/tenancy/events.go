package tenancy

import "github.com/otal-labs/nexul/internal/platform/eventbus"

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

// Topics declares each topic with its payload; workspace.member.added carries an InvitationEvent when an
// invitation admits the member, a MemberEvent otherwise.
func Topics() []eventbus.Topic {
	return []eventbus.Topic{
		{Name: TopicInvitationCreated, Payload: InvitationEvent{}},
		{Name: TopicInvitationRevoked, Payload: InvitationEvent{}},
		{Name: TopicInvitationRedeemed, Payload: InvitationEvent{}},
		{Name: TopicInvitationDeleted, Payload: InvitationEvent{}},
		{Name: TopicAccountAdmitted, Payload: InvitationEvent{}},
		{Name: TopicWorkspaceMemberAdded, Payload: InvitationEvent{}},
		{Name: TopicWorkspaceMemberAdded, Payload: MemberEvent{}},
		{Name: TopicWorkspaceMemberRemoved, Payload: MemberEvent{}},
		{Name: TopicWorkspaceMemberUpdated, Payload: MemberEvent{}},
		{Name: TopicWorkspaceUpdated, Payload: WorkspaceEvent{}},
	}
}

type InvitationEvent struct {
	InvitationID string   `json:"invitation_id"`
	ActorID      string   `json:"actor_id,omitempty"`
	UserID       string   `json:"user_id,omitempty"`
	WorkspaceID  string   `json:"workspace_id,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	RoleIDsAdded []string `json:"role_ids_added,omitempty" jsonschema:"The roles the person was just given in the workspace, on workspace.member.added and workspace.member.updated."`
}

// MemberEvent is the payload of a membership change made outside an invitation; updated covers the roles a member holds.
type MemberEvent struct {
	UserID       string   `json:"user_id"`
	WorkspaceID  string   `json:"workspace_id"`
	ActorID      string   `json:"actor_id,omitempty"`
	ProjectIDs   []string `json:"project_ids,omitempty" jsonschema:"On workspace.member.updated, the workspace's projects, whose access the change can move."`
	RoleIDsAdded []string `json:"role_ids_added,omitempty" jsonschema:"The roles the person was just given in the workspace, on workspace.member.added and workspace.member.updated."`
}

// WorkspaceEvent is the payload of a rename or slug change; it carries the new name and slug so a client open on the workspace can move its URL.
type WorkspaceEvent struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ActorID     string `json:"actor_id,omitempty"`
}
