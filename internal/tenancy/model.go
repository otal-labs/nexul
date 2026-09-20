// Package tenancy implements Workspace; named "tenancy" since internal/workspace already claimed that name.
package tenancy

import "time"

// Workspace's Owner implicitly has full access to everything inside it.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
