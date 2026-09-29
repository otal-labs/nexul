// Package tenancy implements Workspace; named "tenancy" since internal/workspace already claimed that name.
package tenancy

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// DefaultMentionChipTemplate renders a chip as the fixed icon+title+status it showed before templates existed.
const DefaultMentionChipTemplate = "{ticket.Ticket} {ticket.Status}"

// Workspace's Owner implicitly has full access to everything inside it.
type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// MentionChipTemplate is the @-mention ticket chip layout with {ticket.Field} tokens; gated on workspaces:write.
	MentionChipTemplate string    `json:"mention_chip_template"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Member's RoleID is currently always the workspace's Owner role, assigned via Service.Create.
type Member struct {
	UserID      string    `json:"user_id"`
	WorkspaceID string    `json:"workspace_id"`
	RoleID      string    `json:"role_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Invite resolves into a real Member the moment that login's first User row is created.
type Invite struct {
	WorkspaceID string    `json:"workspace_id"`
	Login       string    `json:"login"`
	RoleID      string    `json:"role_id"`
	InvitedBy   string    `json:"invited_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type Invitation struct {
	ID         string             `json:"id"`
	InvitedBy  string             `json:"invited_by"`
	CreatedAt  time.Time          `json:"created_at"`
	ExpiresAt  time.Time          `json:"expires_at"`
	RedeemedAt *time.Time         `json:"-"`
	RedeemedBy string             `json:"-"`
	Grants     []*InvitationGrant `json:"grants"`
}

type InvitationGrant struct {
	WorkspaceID   string          `json:"workspace_id"`
	WorkspaceName string          `json:"workspace_name,omitempty"`
	RoleID        string          `json:"role_id"`
	RoleName      string          `json:"role_name,omitempty"`
	Allow         permissions.Set `json:"allow"`
	Deny          permissions.Set `json:"deny"`
}

type InvitationIdentity struct {
	ID             string
	Provider       string
	ProviderUserID string
	Login          string
	Name           string
	AvatarURL      string
}

type InvitationAdmission struct {
	UserID  string
	Created bool
}

// MemberView adds login for the roster, since Member alone only carries the UserID.
type MemberView struct {
	UserID string `json:"user_id"`
	Login  string `json:"login"`
	RoleID string `json:"role_id"`
}

// MembersList is a workspace's current roster plus pending invites (Membership invites' Members page).
type MembersList struct {
	Members []MemberView `json:"members"`
	Invites []*Invite    `json:"invites"`
}

// AccountStatusRemoved is the Account status of an account whose access was taken away; it holds no memberships.
const AccountStatusRemoved = "removed"

// TeamAccount is a registered account as the Team page shows it, resolved through AccountGate.
type TeamAccount struct {
	ID                 string    `json:"id"`
	Login              string    `json:"login"`
	Name               string    `json:"name"`
	DisplayName        string    `json:"display_name,omitempty"`
	AvatarURL          string    `json:"avatar_url"`
	Status             string    `json:"status"`
	CanCreateWorkspace bool      `json:"can_create_workspace"`
	CreatedAt          time.Time `json:"created_at"`
	// Online is a live browser socket open right now; LastSeenAt is their latest session activity, nil once signed out everywhere.
	Online     bool       `json:"online"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	// AvatarOverride is the uploaded picture as a data URI; it leaves the server only through the avatar route.
	AvatarOverride string `json:"-"`
}

// Person is all any member of a workspace sees of another: never a role, an override, an email, or account status.
type Person struct {
	UserID      string `json:"user_id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

// People is a workspace's people as the directory route answers them.
type People struct {
	People []Person `json:"people"`
}

// TeamMembership is one person's place in one workspace: its role and their workspace-wide overrides.
type TeamMembership struct {
	UserID        string          `json:"-"`
	WorkspaceID   string          `json:"workspace_id"`
	WorkspaceName string          `json:"workspace_name"`
	RoleID        string          `json:"role_id"`
	RoleName      string          `json:"role_name"`
	IsOwner       bool            `json:"is_owner"`
	Allow         permissions.Set `json:"allow"`
	Deny          permissions.Set `json:"deny"`
}

// TeamPerson is one registered account with every workspace membership it holds.
type TeamPerson struct {
	TeamAccount
	Workspaces []*TeamMembership `json:"workspaces"`
}

// TeamRole is a role a Team member can be given, or the Owner role they may already hold.
type TeamRole struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IsOwner bool   `json:"is_owner"`
}

// TeamWorkspace is one workspace on the instance, its roles, and whether the viewer holds members:write there.
type TeamWorkspace struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	CanManageMembers bool        `json:"can_manage_members"`
	Roles            []*TeamRole `json:"roles"`
}

// Team is the people and workspaces the viewer may see; CanManageAccounts is whether they may change account status.
type Team struct {
	People            []*TeamPerson    `json:"people"`
	Workspaces        []*TeamWorkspace `json:"workspaces"`
	CanManageAccounts bool             `json:"can_manage_accounts"`
}
