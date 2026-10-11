package access

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Overwrite is workspace-wide when ResourceType is "workspace" (ResourceID is the workspace id).
type Overwrite struct {
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	UserID       string          `json:"user_id"`
	Allow        permissions.Set `json:"allow"`
	Deny         permissions.Set `json:"deny"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// User is mirrored here so access never imports auth (ADR 0017).
type User struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

// RoleInfo is mirrored here so access never imports roles (ADR 0017).
type RoleInfo struct {
	RoleID      string
	IsOwnerRole bool
	Permissions permissions.Set
	// Restricted is a Restricted member's membership: project areas answer from Project access alone (ADR 0097).
	Restricted bool
}

// The kinds of Source: what decided an answer.
const (
	SourceOwner        = "owner"
	SourceMember       = "member"
	SourceRole         = "role"
	SourceRule         = "rule"
	SourceRestricted   = "restricted"
	SourceNotInProject = "not_in_project"
)

// The states a Source sets, and the targets a rule names.
const (
	StateAllow    = "allow"
	StateFallback = "fallback"
	StateDeny     = "deny"
	TargetRole    = "role"
	TargetPerson  = "person"
)

// Source is what decided an access answer, as ids (ADR 0148): the Owner role, membership, the person's role, a rule
// on an entity, a Restricted member's Project access, or a project hidden from them.
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
