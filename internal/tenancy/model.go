// Package tenancy implements Workspace; named "tenancy" since internal/workspace already claimed that name.
package tenancy

import (
	"regexp"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// DefaultMentionChipTemplate renders a chip as the fixed icon+title+status it showed before templates existed.
const DefaultMentionChipTemplate = "{ticket.Ticket} {ticket.Status}"

// TemplateKind names the mention chip template among the instance templates (ADR 0103).
const TemplateKind = "mention_chip"

// Workspace's Owner implicitly has full access to everything inside it.
type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Slug names the workspace in every URL; unique on the instance and kept when the name changes.
	Slug string `json:"slug"`
	// MentionChipTemplate is the @-mention ticket chip layout with {ticket.Field} tokens; gated on workspaces:write.
	MentionChipTemplate string `json:"mention_chip_template"`
	// MentionChipTemplateEdited is false while the workspace follows the instance's chip template (ADR 0103).
	MentionChipTemplateEdited bool      `json:"mention_chip_template_edited"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

// Member is one person's place in a workspace; Restricted is their Every project row set to None (ADR 0097).
type Member struct {
	UserID      string    `json:"user_id"`
	WorkspaceID string    `json:"workspace_id"`
	RoleID      string    `json:"role_id"`
	CreatedAt   time.Time `json:"created_at"`
	Restricted  bool      `json:"restricted"`
}

// Every project values: From role (the role's project areas on every project) or None (a Restricted member).
const (
	EveryProjectRole = "role"
	EveryProjectNone = "none"
)

// everyProject is the wire value of a membership's Every project row.
func everyProject(restricted bool) string {
	if restricted {
		return EveryProjectNone
	}
	return EveryProjectRole
}

// ProjectAccess is one project's levels for a Restricted member, held as the actions they expand to.
type ProjectAccess struct {
	ProjectID   string          `json:"project_id"`
	ProjectName string          `json:"project_name,omitempty"`
	Allow       permissions.Set `json:"allow"`
	UserID      string          `json:"-"`
	WorkspaceID string          `json:"-"`
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
	// EveryProject is "role" (also when empty) or "none", which lands the person restricted to ProjectAccess.
	EveryProject  string           `json:"every_project,omitempty"`
	ProjectAccess []*ProjectAccess `json:"project_access,omitempty"`
}

// Restricted reports whether the grant admits a Restricted member.
func (g *InvitationGrant) Restricted() bool {
	return g.EveryProject == EveryProjectNone
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
	ID          string    `json:"id"`
	Login       string    `json:"login"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name,omitempty"`
	AvatarURL   string    `json:"avatar_url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	// Providers are the sign-in accounts linked to this account, in the order they were linked.
	Providers []string `json:"providers"`
	// Usernames are the names others know those accounts by, keyed by provider; a provider without one is absent.
	Usernames map[string]string `json:"usernames,omitempty"`
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

// TeamMembership is one person's place in one workspace: its role, their workspace-wide overrides, their Every
// project row, and the projects they hold access to that the viewer may open.
type TeamMembership struct {
	UserID        string           `json:"-"`
	WorkspaceID   string           `json:"workspace_id"`
	WorkspaceName string           `json:"workspace_name"`
	RoleID        string           `json:"role_id"`
	RoleName      string           `json:"role_name"`
	IsOwner       bool             `json:"is_owner"`
	Allow         permissions.Set  `json:"allow"`
	Deny          permissions.Set  `json:"deny"`
	Restricted    bool             `json:"-"`
	EveryProject  string           `json:"every_project"`
	Projects      []*ProjectAccess `json:"projects"`
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

// Team is the people and workspaces the viewer may see; CanManageAccounts is whether they hold accounts:write.
type Team struct {
	People            []*TeamPerson    `json:"people"`
	Workspaces        []*TeamWorkspace `json:"workspaces"`
	CanManageAccounts bool             `json:"can_manage_accounts"`
}

// maxSlugLength keeps a slug short enough to read in a URL; migration 0039 cuts backfilled slugs the same way.
const maxSlugLength = 48

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// reservedSlugs are the top-level paths the server and the web app own; migration 0039 holds the same list.
var reservedSlugs = map[string]bool{
	"api": true, "assets": true, "auth": true, "hooks": true, "invite": true, "login": true, "logout": true,
	"mcp": true, "onboarding": true, "openobserve": true, "settings": true, "setup": true, "static": true,
	"swagger": true, "wizard": true, "ws": true,
}

// Slugify derives a slug from a workspace name exactly as migration 0039 does: ASCII letters and digits kept,
// every other run of characters one dash, "workspace" when nothing is left.
func Slugify(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if len(slug) > maxSlugLength {
		slug = strings.TrimRight(slug[:maxSlugLength], "-")
	}
	if slug == "" {
		return "workspace"
	}
	return slug
}

// ValidSlug reports whether slug is well formed and not a path the app itself owns.
func ValidSlug(slug string) bool {
	return len(slug) <= maxSlugLength && slugPattern.MatchString(slug) && !reservedSlugs[slug]
}
