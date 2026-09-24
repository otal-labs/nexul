package workspace

import (
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
)

// Project is the organizational grouping inside a workspace; tickets belong to exactly one, ordered by Position.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Prefix is an immutable 2-5 uppercase tag, unique per workspace, rendering a human ticket id (PREFIX-N).
	Prefix   string `json:"prefix"`
	Position int    `json:"position"`
	// WorkspaceID nests this project under a Workspace; immutable once set.
	WorkspaceID string      `json:"workspace_id"`
	Icon        ProjectIcon `json:"icon"`
	// TestsLocation is the project wizard's answer the interview starts from; "" means not answered yet.
	TestsLocation TestsLocation `json:"tests_location"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// TestsLocation says whether a project's tests live in its deployed repository or in a separate tests repository.
type TestsLocation string

const (
	TestsLocationUnset    TestsLocation = ""
	TestsLocationSame     TestsLocation = "same"
	TestsLocationSeparate TestsLocation = "separate"
)

// Valid reports a known answer, unset included so an answer can be withdrawn.
func (l TestsLocation) Valid() bool {
	return l == TestsLocationUnset || l == TestsLocationSame || l == TestsLocationSeparate
}

// RepoRef is a git repository associated with a project (a repository belongs to exactly one project).
type RepoRef struct {
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	ConnectorID string   `json:"connector_id"`
	Role        RepoRole `json:"role"`
}

// RepoRole is what a project's repository is for; a tests repository never builds a stack or deploys.
type RepoRole string

const (
	RepoRoleApp   RepoRole = "app"
	RepoRoleTests RepoRole = "tests"
)

// Valid reports a known role.
func (r RepoRole) Valid() bool {
	return r == RepoRoleApp || r == RepoRoleTests
}

// Category groups tickets inside a project; a ticket belongs to at most one, uncategorized is allowed.
type Category struct {
	ID        string       `json:"id"`
	ProjectID string       `json:"project_id"`
	Name      string       `json:"name"`
	Position  int          `json:"position"`
	Color     colors.Color `json:"color"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// TicketType is referenced by id so renaming a type never rewrites ticket rows.
type TicketType struct {
	ID        string       `json:"id"`
	ProjectID string       `json:"project_id"`
	Name      string       `json:"name"`
	Position  int          `json:"position"`
	Color     colors.Color `json:"color"`
	// BodyTemplate is markdown that pre-fills a new ticket's body; guidance only, never validated or reapplied.
	BodyTemplate string    `json:"body_template"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DefaultTicketTypes are every new project's types in board order; migration 0009 backfills the same templates.
var DefaultTicketTypes = []TicketType{
	{Name: "task", BodyTemplate: "## What needs doing\n\n\n## Acceptance criteria\n\n"},
	{Name: "bug", BodyTemplate: "## Steps to reproduce\n\n\n## Expected result\n\n\n## Actual result\n\n\n## Provide screenshot\n\n"},
	{Name: "feature", BodyTemplate: "## Why\n\n\n## Acceptance criteria\n\n\n## Out of scope\n\n"},
}

// ProjectIcon is purely cosmetic; unset ("") falls back to prefix-only rendering.
type ProjectIcon string

const (
	ProjectIconBox      ProjectIcon = "Box"
	ProjectIconRocket   ProjectIcon = "Rocket"
	ProjectIconServer   ProjectIcon = "Server"
	ProjectIconGlobe    ProjectIcon = "Globe"
	ProjectIconDatabase ProjectIcon = "Database"
	ProjectIconLayers   ProjectIcon = "Layers"
	ProjectIconTerminal ProjectIcon = "Terminal"
	ProjectIconShield   ProjectIcon = "Shield"
	ProjectIconZap      ProjectIcon = "Zap"
	ProjectIconPackage  ProjectIcon = "Package"
	ProjectIconCpu      ProjectIcon = "Cpu"
	ProjectIconCloud    ProjectIcon = "Cloud"
)

var validProjectIcons = map[ProjectIcon]bool{
	ProjectIconBox:      true,
	ProjectIconRocket:   true,
	ProjectIconServer:   true,
	ProjectIconGlobe:    true,
	ProjectIconDatabase: true,
	ProjectIconLayers:   true,
	ProjectIconTerminal: true,
	ProjectIconShield:   true,
	ProjectIconZap:      true,
	ProjectIconPackage:  true,
	ProjectIconCpu:      true,
	ProjectIconCloud:    true,
}

// StatusKind stages are fixed and ordered; only done is terminal, and rules read Kind, not names (ADR 0022).
type StatusKind string

const (
	StatusKindBacklog  StatusKind = "backlog"
	StatusKindProgress StatusKind = "progress"
	StatusKindReview   StatusKind = "review"
	StatusKindTesting  StatusKind = "testing"
	StatusKindDone     StatusKind = "done"
)

// StatusKinds lists the stages in board order.
var StatusKinds = []StatusKind{StatusKindBacklog, StatusKindProgress, StatusKindReview, StatusKindTesting, StatusKindDone}

func (k StatusKind) Valid() bool {
	for _, known := range StatusKinds {
		if k == known {
			return true
		}
	}
	return false
}

// StatusIcon is purely cosmetic and decoupled from Kind, which governs domain behavior.
type StatusIcon string

const (
	StatusIconBacklog    StatusIcon = "CircleDashed"
	StatusIconTodo       StatusIcon = "Circle"
	StatusIconInProgress StatusIcon = "CircleDot"
	StatusIconInReview   StatusIcon = "CircleEllipsis"
	StatusIconDone       StatusIcon = "CircleCheckBig"
	StatusIconCancelled  StatusIcon = "CircleX"
)

var validStatusIcons = map[StatusIcon]bool{
	StatusIconBacklog:    true,
	StatusIconTodo:       true,
	StatusIconInProgress: true,
	StatusIconInReview:   true,
	StatusIconDone:       true,
	StatusIconCancelled:  true,
}

// Status keeps a stable identity across renames; Position is the board order.
type Status struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Position  int        `json:"position"`
	Kind      StatusKind `json:"kind"`
	Icon      StatusIcon `json:"icon"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// DeleteImpact lets the UI warn before a destructive delete, since removal is a one-way door.
type DeleteImpact struct {
	Tickets  int `json:"tickets"`
	Repos    int `json:"repos"`
	Services int `json:"services"`
}

// Kind values are part of the wire contract (ADR 0044) and are additive-only.
type Kind string

const (
	KindTicketAssigned  Kind = "ticket.assigned"
	KindTicketMentioned Kind = "ticket.mentioned"
	KindTicketStatus    Kind = "ticket.status_changed"
	KindDocCreated      Kind = "doc.created"
	KindDocUpdated      Kind = "doc.updated"
	KindMemoryUpdated   Kind = "memory.updated"
	KindPlayRunFinished Kind = "play.run_finished"
	KindPlayRunWaiting  Kind = "play.run_waiting"
)

// SubjectType is what a notification points at, used to build the click-through link in the inbox.
type SubjectType string

const (
	SubjectTicket SubjectType = "ticket"
	SubjectDoc    SubjectType = "doc"
	SubjectMemory SubjectType = "memory"
)

// Notification's SubjectTitle is denormalized so the frontend renders the list without extra fetches.
type Notification struct {
	ID           string      `json:"id"`
	UserID       string      `json:"user_id"`
	Kind         Kind        `json:"kind"`
	SubjectType  SubjectType `json:"subject_type"`
	SubjectID    string      `json:"subject_id"`
	SubjectTitle string      `json:"subject_title"`
	Read         bool        `json:"read"`
	CreatedAt    time.Time   `json:"created_at"`
}
