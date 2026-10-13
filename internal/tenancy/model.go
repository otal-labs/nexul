// Package tenancy implements Workspace; named "tenancy" since internal/workspace already claimed that name.
package tenancy

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
	MentionChipTemplateEdited bool `json:"mention_chip_template_edited"`
	// AccessSetupFinished is false while an Owner has yet to finish the one-time access setup (ADR 0148).
	AccessSetupFinished bool      `json:"access_setup_finished"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Member is one person's place in a workspace.
type Member struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	// RoleID is the first of RoleIDs, the roles they hold in the order given (ADR 0148).
	RoleID    string    `json:"role_id"`
	RoleIDs   []string  `json:"role_ids"`
	CreatedAt time.Time `json:"created_at"`
}

// Held is the roles the membership holds: RoleIDs, or RoleID alone where a writer gave just that one.
func (m *Member) Held() []string {
	if len(m.RoleIDs) == 0 && m.RoleID != "" {
		return []string{m.RoleID}
	}
	return m.RoleIDs
}

// MyAccess is a member's own access in one workspace, as /api/workspaces/{id}/me answers it.
type MyAccess struct {
	Roles       []*TeamRole
	Permissions []string
	Restricted  bool
	Projects    []MeProject
	// AccessSetupFinished is the workspace's, so a member waiting on the one-time setup sees why (ADR 0148).
	AccessSetupFinished bool
}

// RoleNames is roles' names joined, the one role_name a client from before members held several reads.
func RoleNames(roles []*TeamRole) string {
	names := make([]string, len(roles))
	for i, r := range roles {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}

// MeProject is one project a member is in and the project-area actions they hold there.
type MeProject struct {
	ProjectID string          `json:"project_id"`
	Actions   permissions.Set `json:"actions"`
}

// ProjectAccess is the shape the removed Project access was sent in; old fields still parse and read as empty.
type ProjectAccess struct {
	ProjectID   string          `json:"project_id"`
	ProjectName string          `json:"project_name,omitempty"`
	Allow       permissions.Set `json:"allow"`
}

// RefusePerPersonAccess refuses a non-empty value for a field the wipe removed (ADR 0148), naming what replaced it;
// an empty one still passes, so a client that sends it keeps working.
func RefusePerPersonAccess(allow, deny permissions.Set, everyProject string, projectAccess int) error {
	if len(allow) > 0 || len(deny) > 0 {
		return fmt.Errorf("%w: per-person allow and deny are gone; give the person a role that holds it, or set a rule on the project, doc or play", apperrs.ErrInvalid)
	}
	if everyProject != "" || projectAccess > 0 {
		return fmt.Errorf("%w: every_project and project_access are gone; list the person, or a role they hold, on each project they should be in", apperrs.ErrInvalid)
	}
	return nil
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

// InvitationGrant is one workspace of an invitation: the roles it gives there and nothing per person (ADR 0148).
type InvitationGrant struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name,omitempty"`
	// RoleID and RoleName name the first of RoleIDs, for a client from before an invitation gave several.
	RoleID   string      `json:"role_id"`
	RoleName string      `json:"role_name,omitempty"`
	RoleIDs  []string    `json:"role_ids"`
	Roles    []*TeamRole `json:"roles"`
	// Allow, Deny, EveryProject and ProjectAccess are removed fields that read as empty and refuse a value.
	Allow         permissions.Set  `json:"allow"`
	Deny          permissions.Set  `json:"deny"`
	EveryProject  string           `json:"every_project,omitempty"`
	ProjectAccess []*ProjectAccess `json:"project_access,omitempty"`
}

// Held is the roles the grant gives: RoleIDs, or RoleID alone where a writer gave just that one.
func (g *InvitationGrant) Held() []string {
	if len(g.RoleIDs) == 0 && g.RoleID != "" {
		return []string{g.RoleID}
	}
	return g.RoleIDs
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

// Person is all any member of a workspace sees of another: never a role but Owner, an email, or account status.
type Person struct {
	UserID      string `json:"user_id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	// Owner is set in a workspace's directory, so someone with no role knows who to ask for one (ADR 0148).
	Owner bool `json:"owner,omitempty"`
}

// People is a workspace's people as the directory route answers them.
type People struct {
	People []Person `json:"people"`
}

// ProjectPeople is a project's People and the roles its rules let in.
type ProjectPeople struct {
	People  []Person `json:"people"`
	RoleIDs []string `json:"role_ids"`
}

// TeamMembership is one person's place in one workspace: their roles, RoleID naming the first and RoleName all. Allow,
// Deny, EveryProject and Projects are removed fields that read as empty (ADR 0148).
type TeamMembership struct {
	UserID        string           `json:"-"`
	WorkspaceID   string           `json:"workspace_id"`
	WorkspaceName string           `json:"workspace_name"`
	RoleID        string           `json:"role_id"`
	RoleName      string           `json:"role_name"`
	Roles         []*TeamRole      `json:"roles"`
	IsOwner       bool             `json:"is_owner"`
	Allow         permissions.Set  `json:"allow"`
	Deny          permissions.Set  `json:"deny"`
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
