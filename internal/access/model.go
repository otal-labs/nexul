package access

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Overwrite is a rule on one entity for a person or a role (Target).
type Overwrite struct {
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Target       string          `json:"target"`
	UserID       string          `json:"user_id"`
	RoleID       string          `json:"role_id,omitempty"`
	Allow        permissions.Set `json:"allow"`
	Deny         permissions.Set `json:"deny"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ProjectHeld is one project someone is in and the project-area actions they hold there.
type ProjectHeld struct {
	ProjectID string
	Actions   permissions.Set
}

// Rules are the rules on one entity that reach one person: the rule naming them, and those naming a role they hold.
type Rules struct {
	Person *Overwrite
	Roles  []*Overwrite
}

// User is mirrored here so access never imports auth (ADR 0017).
type User struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

// RoleInfo is a member's roles in one workspace, mirrored here so access never imports roles (ADR 0017).
type RoleInfo struct {
	// RoleID is the first role given; IsOwnerRole and Permissions answer for every role held.
	RoleID      string
	IsOwnerRole bool
	Permissions permissions.Set
	// Roles are every role held, in the order given; empty means RoleID alone.
	Roles []HeldRole
}

// HeldRole is one role a member holds and what it allows.
type HeldRole struct {
	ID          string
	IsOwner     bool
	Permissions permissions.Set
}

// RolesHeld is the RoleInfo of a member holding roles, in the order given.
func RolesHeld(roles ...HeldRole) RoleInfo {
	info := RoleInfo{Roles: roles}
	var union []permissions.Action
	for _, r := range roles {
		info.IsOwnerRole = info.IsOwnerRole || r.IsOwner
		union = append(union, r.Permissions...)
	}
	info.Permissions = permissions.SetOf(union...)
	if len(roles) > 0 {
		info.RoleID = roles[0].ID
	}
	return info
}

// roleIDs is every role held, in the order given.
func (r RoleInfo) roleIDs() []string {
	if len(r.Roles) == 0 && r.RoleID != "" {
		return []string{r.RoleID}
	}
	out := make([]string, 0, len(r.Roles))
	for _, h := range r.Roles {
		out = append(out, h.ID)
	}
	return out
}

// sourceRole is the role a roles' answer names: the Owner role, else the first allowing action, else the first.
func (r RoleInfo) sourceRole(action permissions.Action) string {
	for _, h := range r.Roles {
		if h.IsOwner || (!r.IsOwnerRole && h.Permissions.Has(action)) {
			return h.ID
		}
	}
	return r.RoleID
}

// The kinds of Source: what decided an answer.
const (
	SourceOwner          = "owner"
	SourceMember         = "member"
	SourceRole           = "role"
	SourceRule           = "rule"
	SourceOutsideProject = "outside_project"
	SourceNotInProject   = "not_in_project"
)

// The states a Source sets, and the targets a rule names.
const (
	StateAllow    = "allow"
	StateFallback = "fallback"
	StateDeny     = "deny"
	TargetRole    = "role"
	TargetPerson  = "person"
)

// Source is what decided an access answer, as ids (ADR 0148): the Owner role, membership, the person's roles, a rule
// on an entity, a project-area action asked outside any project, or a project the person is not in.
type Source struct {
	Kind       string `json:"kind"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	State      string `json:"state"`
}

// NamedSource is a Source with what it names resolved, and the sentence a person reads.
type NamedSource struct {
	Source
	EntityName string `json:"entity_name"`
	TargetName string `json:"target_name"`
	Sentence   string `json:"sentence"`
}

// Verdict is one answer and its source.
type Verdict struct {
	Allowed bool        `json:"allowed"`
	Source  NamedSource `json:"source"`
}

// Answer is a person's answer for one action on an entity, and the one the entity would give with its own rule for
// them at Fallback.
type Answer struct {
	Action permissions.Action `json:"action"`
	Verdict
	Fallback Verdict `json:"fallback"`
}

// Explanation is a person's answer for every action an entity's rules may set.
type Explanation struct {
	ResourceType string   `json:"resource_type"`
	ResourceID   string   `json:"resource_id"`
	UserID       string   `json:"user_id"`
	Effective    []Answer `json:"effective"`
}
