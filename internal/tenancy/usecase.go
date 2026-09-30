package tenancy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// DefaultWorkspaceID is the id of the seeded single-tenant workspace the Owner Wizard binds new users to.
const DefaultWorkspaceID = "workspace-default"

// Service is the tenancy use-case layer: creating and listing workspaces.
type Service struct {
	repo      Repo
	members   MemberRepo
	invites   InviteRepo
	roles     RoleGate
	perm      PermissionGate
	roleNames RoleNameGate
	wsPerms   WorkspacePermissionGate
	allowlist AllowlistGate
	users     UserLookupGate
	channels  ChannelGate
	plays     PlaysGate
	accounts  AccountGate
	now       func() time.Time
}

// NewService wires the tenancy use-cases over their repos and permission gates.
func NewService(repo Repo, members MemberRepo, invites InviteRepo, roles RoleGate, perm PermissionGate, roleNames RoleNameGate, wsPerms WorkspacePermissionGate, allowlist AllowlistGate, users UserLookupGate, channels ChannelGate, plays PlaysGate, accounts AccountGate) *Service {
	return &Service{repo: repo, members: members, invites: invites, roles: roles, perm: perm, roleNames: roleNames, wsPerms: wsPerms, allowlist: allowlist, users: users, channels: channels, plays: plays, accounts: accounts, now: time.Now}
}

// Create adds a workspace and binds userID as its Owner; requires workspaces:create in any workspace, which makes
// that permission as strong as Owner (ADR 0088).
func (s *Service) Create(ctx context.Context, userID, name string) (*Workspace, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: workspace name is required", apperrs.ErrInvalid)
	}
	ok, err := s.perm.HoldsAnywhere(ctx, userID, permissions.WorkspacesCreate)
	if err != nil {
		return nil, fmt.Errorf("check %s: %w", permissions.WorkspacesCreate, err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.WorkspacesCreate)
	}
	slug, err := s.freeSlug(ctx, Slugify(name))
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	w := &Workspace{ID: ids.New(), Name: name, Slug: slug, MentionChipTemplate: DefaultMentionChipTemplate, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	roleID, err := s.roles.CreateOwnerRole(ctx, w.ID)
	if err != nil {
		return nil, fmt.Errorf("create owner role for workspace %s: %w", w.ID, err)
	}
	// The creator becomes the workspace's first member so it shows up in their own ListForUser call.
	if err := s.members.AddMember(ctx, &Member{UserID: userID, WorkspaceID: w.ID, RoleID: roleID, CreatedAt: now}); err != nil {
		return nil, fmt.Errorf("add workspace creator as member: %w", err)
	}
	// #general gives every new workspace somewhere for chat to land immediately.
	if err := s.channels.CreateGeneralChannel(ctx, w.ID, userID); err != nil {
		return nil, fmt.Errorf("create general channel for workspace %s: %w", w.ID, err)
	}
	if err := s.plays.SeedDefaultPlays(ctx, w.ID); err != nil {
		return nil, fmt.Errorf("seed default plays for workspace %s: %w", w.ID, err)
	}
	return w, nil
}

// freeSlug returns base, or base-2, base-3, and so on: the first that is neither reserved nor taken.
func (s *Service) freeSlug(ctx context.Context, base string) (string, error) {
	for n := 1; ; n++ {
		candidate := base
		if n > 1 {
			suffix := fmt.Sprintf("-%d", n)
			candidate = strings.TrimRight(base[:min(len(base), maxSlugLength-len(suffix))], "-") + suffix
		}
		if reservedSlugs[candidate] {
			continue
		}
		_, err := s.repo.GetBySlug(ctx, candidate)
		if errors.Is(err, apperrs.ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("check slug %s: %w", candidate, err)
		}
	}
}

// Rename requires workspaces:write in the workspace being renamed. The slug changes only when slug is non-nil,
// so links keep working through a rename unless the caller moves them on purpose.
func (s *Service) Rename(ctx context.Context, userID, id, name string, slug *string) (*Workspace, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: workspace name is required", apperrs.ErrInvalid)
	}
	if err := s.requireWorkspacePermission(ctx, userID, id, permissions.WorkspacesWrite); err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *current
	updated.Name = name
	if slug != nil {
		if err := s.checkSlugFor(ctx, id, *slug); err != nil {
			return nil, err
		}
		updated.Slug = *slug
	}
	updated.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("rename workspace %s: %w", id, err)
	}
	return &updated, nil
}

func (s *Service) checkSlugFor(ctx context.Context, id, slug string) error {
	if !ValidSlug(slug) {
		return fmt.Errorf("%w: slug %q must be lowercase letters and digits joined by single dashes, at most %d characters, and not a reserved path such as settings", apperrs.ErrInvalid, slug, maxSlugLength)
	}
	other, err := s.repo.GetBySlug(ctx, slug)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check slug %s: %w", slug, err)
	}
	if other.ID != id {
		return fmt.Errorf("%w: slug %q is taken by another workspace", apperrs.ErrConflict, slug)
	}
	return nil
}

// SetMentionChipTemplate changes how @-mention ticket chips render in workspace id; requires workspaces:write there.
func (s *Service) SetMentionChipTemplate(ctx context.Context, actorID, id, template string) (*Workspace, error) {
	id = strings.TrimSpace(id)
	if err := s.requireWorkspacePermission(ctx, actorID, id, permissions.WorkspacesWrite); err != nil {
		return nil, err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *current
	updated.MentionChipTemplate = template
	updated.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("set mention chip template for workspace %s: %w", id, err)
	}
	return &updated, nil
}

// BindDefaultWorkspaceOwner binds userID as owner of the pre-seeded default workspace without creating a new row.
func (s *Service) BindDefaultWorkspaceOwner(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if _, err := s.repo.Get(ctx, DefaultWorkspaceID); err != nil {
		return fmt.Errorf("get default workspace: %w", err)
	}
	// Idempotent: a retried wizard finish must not mint a second Owner role for an already-bound member.
	if _, err := s.members.RoleIDFor(ctx, DefaultWorkspaceID, userID); err == nil {
		return nil
	}
	roleID, err := s.roles.CreateOwnerRole(ctx, DefaultWorkspaceID)
	if err != nil {
		return fmt.Errorf("create default workspace owner role: %w", err)
	}
	if err := s.members.AddMember(ctx, &Member{UserID: userID, WorkspaceID: DefaultWorkspaceID, RoleID: roleID, CreatedAt: s.now().UTC()}); err != nil {
		return fmt.Errorf("bind %s as default workspace owner: %w", userID, err)
	}
	// The pre-seeded default workspace never goes through Create, so it needs its own #general seed here.
	if err := s.channels.CreateGeneralChannel(ctx, DefaultWorkspaceID, userID); err != nil {
		return fmt.Errorf("create general channel for default workspace: %w", err)
	}
	if err := s.plays.SeedDefaultPlays(ctx, DefaultWorkspaceID); err != nil {
		return fmt.Errorf("seed default plays for default workspace: %w", err)
	}
	return nil
}

// MemberRoleID exposes role lookup for the roles domain's ActionManageRoles gate.
func (s *Service) MemberRoleID(ctx context.Context, workspaceID, userID string) (string, error) {
	roleID, err := s.members.RoleIDFor(ctx, workspaceID, userID)
	if err != nil {
		return "", fmt.Errorf("resolve role for user %s in workspace %s: %w", userID, workspaceID, err)
	}
	return roleID, nil
}

// MemberRoleName resolves the role id via MemberRoleID, then its name via RoleNameGate.
func (s *Service) MemberRoleName(ctx context.Context, workspaceID, userID string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	if workspaceID == "" {
		return "", fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if userID == "" {
		return "", fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	roleID, err := s.MemberRoleID(ctx, workspaceID, userID)
	if err != nil {
		return "", err
	}
	name, err := s.roleNames.RoleName(ctx, workspaceID, roleID)
	if err != nil {
		return "", fmt.Errorf("resolve role name %s: %w", roleID, err)
	}
	return name, nil
}

// MemberPermissions returns an empty list, not an error, for an unresolvable member.
func (s *Service) MemberPermissions(ctx context.Context, workspaceID, userID string) []string {
	return s.wsPerms.WorkspacePermissions(ctx, userID, workspaceID)
}

// requireWorkspacePermission already covers Owner via WorkspacePermissionGate's bypass.
func (s *Service) requireWorkspacePermission(ctx context.Context, actorID, workspaceID string, action permissions.Action) error {
	for _, p := range s.wsPerms.WorkspacePermissions(ctx, actorID, workspaceID) {
		if p == string(action) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s permission required", apperrs.ErrForbidden, action)
}

func (s *Service) requireManageWorkspaceMembers(ctx context.Context, actorID, workspaceID string) error {
	return s.requireWorkspacePermission(ctx, actorID, workspaceID, permissions.MembersWrite)
}

// requireHolds refuses a grant in workspaceID carrying an action actorID does not hold there (ADR 0088).
func (s *Service) requireHolds(ctx context.Context, actorID, workspaceID string, grant permissions.Set) error {
	return permissions.RequireHeld(grant, permissions.SetOfStrings(s.wsPerms.WorkspacePermissions(ctx, actorID, workspaceID)))
}

// requireHoldsRole refuses assigning roleID unless actorID holds every action it grants in workspaceID.
func (s *Service) requireHoldsRole(ctx context.Context, actorID, workspaceID, roleID string) error {
	grant, err := s.roleNames.RolePermissions(ctx, workspaceID, roleID)
	if err != nil {
		return fmt.Errorf("read role %s: %w", roleID, err)
	}
	return s.requireHolds(ctx, actorID, workspaceID, grant)
}

// InviteMember requires login to already be allowlisted; an unknown login is held pending until first sign-in.
func (s *Service) InviteMember(ctx context.Context, actorID, workspaceID, login, roleID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	login = strings.ToLower(strings.TrimSpace(login))
	roleID = strings.TrimSpace(roleID)
	if login == "" {
		return fmt.Errorf("%w: login is required", apperrs.ErrInvalid)
	}
	if roleID == "" {
		return fmt.Errorf("%w: role id is required", apperrs.ErrInvalid)
	}
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	isOwner, err := s.roleNames.IsOwnerRole(ctx, workspaceID, roleID)
	if err != nil {
		return fmt.Errorf("check owner role %s: %w", roleID, err)
	}
	if isOwner {
		return fmt.Errorf("%w: the Owner role cannot be assigned via invite", apperrs.ErrInvalid)
	}
	if err := s.requireHoldsRole(ctx, actorID, workspaceID, roleID); err != nil {
		return err
	}
	allowed, err := s.allowlist.IsAllowlisted(ctx, login)
	if err != nil {
		return fmt.Errorf("check allowlist for %s: %w", login, err)
	}
	if !allowed {
		return fmt.Errorf("%w: %s must be on the instance allowlist before they can be invited to a workspace", apperrs.ErrInvalid, login)
	}
	userID, found, err := s.users.UserIDForLogin(ctx, login)
	if err != nil {
		return fmt.Errorf("resolve user for %s: %w", login, err)
	}
	if found {
		if err := s.members.AddMember(ctx, &Member{UserID: userID, WorkspaceID: workspaceID, RoleID: roleID, CreatedAt: s.now().UTC()}); err != nil {
			return fmt.Errorf("add member %s to workspace %s: %w", userID, workspaceID, err)
		}
		return nil
	}
	if err := s.invites.Upsert(ctx, &Invite{WorkspaceID: workspaceID, Login: login, RoleID: roleID, InvitedBy: actorID, CreatedAt: s.now().UTC()}); err != nil {
		return fmt.Errorf("upsert invite for %s to workspace %s: %w", login, workspaceID, err)
	}
	return nil
}

// CancelInvite withdraws a pending invite before it ever resolves into a membership.
func (s *Service) CancelInvite(ctx context.Context, actorID, workspaceID, login string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	login = strings.ToLower(strings.TrimSpace(login))
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	if err := s.invites.Delete(ctx, workspaceID, login); err != nil {
		return fmt.Errorf("cancel invite for %s in workspace %s: %w", login, workspaceID, err)
	}
	return nil
}

// ListWorkspaceMembers returns the roster enriched with login, plus pending invites.
func (s *Service) ListWorkspaceMembers(ctx context.Context, actorID, workspaceID string) (*MembersList, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return nil, err
	}
	members, err := s.members.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members for workspace %s: %w", workspaceID, err)
	}
	views := make([]MemberView, 0, len(members))
	for _, m := range members {
		login, err := s.users.LoginForUserID(ctx, m.UserID)
		if err != nil {
			return nil, fmt.Errorf("resolve login for user %s: %w", m.UserID, err)
		}
		views = append(views, MemberView{UserID: m.UserID, Login: login, RoleID: m.RoleID})
	}
	invites, err := s.invites.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list invites for workspace %s: %w", workspaceID, err)
	}
	if invites == nil {
		// A nil slice marshals as JSON null, which the SPA .map()s over.
		invites = []*Invite{}
	}
	return &MembersList{Members: views, Invites: invites}, nil
}

// RemoveMember refuses to remove the workspace's Owner.
func (s *Service) RemoveMember(ctx context.Context, actorID, workspaceID, userID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	roleID, err := s.members.RoleIDFor(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("resolve role for %s in workspace %s: %w", userID, workspaceID, err)
	}
	isOwner, err := s.roleNames.IsOwnerRole(ctx, workspaceID, roleID)
	if err != nil {
		return fmt.Errorf("check owner role %s: %w", roleID, err)
	}
	if isOwner {
		return fmt.Errorf("%w: the workspace Owner cannot be removed", apperrs.ErrInvalid)
	}
	if err := s.members.RemoveMember(ctx, workspaceID, userID, memberEvent(TopicWorkspaceMemberRemoved, actorID, workspaceID, userID)); err != nil {
		return fmt.Errorf("remove member %s from workspace %s: %w", userID, workspaceID, err)
	}
	return nil
}

// ChangeMemberRole refuses to demote or promote into the Owner role.
func (s *Service) ChangeMemberRole(ctx context.Context, actorID, workspaceID, userID, roleID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return fmt.Errorf("%w: role id is required", apperrs.ErrInvalid)
	}
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	currentRoleID, err := s.members.RoleIDFor(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("resolve role for %s in workspace %s: %w", userID, workspaceID, err)
	}
	isCurrentlyOwner, err := s.roleNames.IsOwnerRole(ctx, workspaceID, currentRoleID)
	if err != nil {
		return fmt.Errorf("check owner role %s: %w", currentRoleID, err)
	}
	if isCurrentlyOwner {
		return fmt.Errorf("%w: the workspace Owner's role cannot be changed", apperrs.ErrInvalid)
	}
	isNewOwner, err := s.roleNames.IsOwnerRole(ctx, workspaceID, roleID)
	if err != nil {
		return fmt.Errorf("check owner role %s: %w", roleID, err)
	}
	if isNewOwner {
		return fmt.Errorf("%w: the Owner role cannot be assigned via invite", apperrs.ErrInvalid)
	}
	if err := s.requireHoldsRole(ctx, actorID, workspaceID, roleID); err != nil {
		return err
	}
	if err := s.members.SetRole(ctx, workspaceID, userID, roleID, memberEvent(TopicWorkspaceMemberUpdated, actorID, workspaceID, userID)); err != nil {
		return fmt.Errorf("set role for %s in workspace %s: %w", userID, workspaceID, err)
	}
	return nil
}

// ListTeam is every account with its access for a holder of accounts:read; anyone else holding members:write
// somewhere sees only the workspaces they manage and the people in them.
func (s *Service) ListTeam(ctx context.Context, actorID string) (*Team, error) {
	everyone, err := s.perm.HoldsAnywhere(ctx, actorID, permissions.AccountsRead)
	if err != nil {
		return nil, fmt.Errorf("check %s: %w", permissions.AccountsRead, err)
	}
	manageAccounts, err := s.perm.HoldsAnywhere(ctx, actorID, permissions.AccountsWrite)
	if err != nil {
		return nil, fmt.Errorf("check %s: %w", permissions.AccountsWrite, err)
	}
	workspaces, err := s.repo.ListWithRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	// Reading every account is not managing a workspace (ADR 0024): each workspace answers for itself.
	visible := make([]*TeamWorkspace, 0, len(workspaces))
	for _, w := range workspaces {
		w.CanManageMembers = s.requireManageWorkspaceMembers(ctx, actorID, w.ID) == nil
		if everyone || w.CanManageMembers {
			visible = append(visible, w)
		}
	}
	if len(visible) == 0 && !everyone {
		return nil, fmt.Errorf("%w: accounts:read or members:write in a workspace required", apperrs.ErrForbidden)
	}
	accounts, err := s.accounts.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	memberships, err := s.members.ListAllMemberships(ctx)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	online, seen, err := s.accounts.Presence(ctx)
	if err != nil {
		return nil, fmt.Errorf("read presence: %w", err)
	}
	people := teamPeople(accounts, memberships, visible, everyone)
	for _, p := range people {
		p.Online = online[p.ID]
		if at, ok := seen[p.ID]; ok {
			p.LastSeenAt = &at
		}
	}
	slices.SortStableFunc(people, byPresence)
	return &Team{People: people, Workspaces: visible, CanManageAccounts: manageAccounts}, nil
}

// byPresence puts whoever is online first, then the most recently seen; ties keep the accounts' login order.
func byPresence(a, b *TeamPerson) int {
	if a.Online != b.Online {
		if a.Online {
			return -1
		}
		return 1
	}
	return lastSeen(b).Compare(lastSeen(a))
}

func lastSeen(p *TeamPerson) time.Time {
	if p.LastSeenAt == nil {
		return time.Time{}
	}
	return *p.LastSeenAt
}

// teamPeople keeps only the visible workspaces' memberships; unless everyone is shown, a person with none of them is left out.
func teamPeople(accounts []*TeamAccount, memberships []*TeamMembership, visible []*TeamWorkspace, everyone bool) []*TeamPerson {
	shown := make(map[string]bool, len(visible))
	for _, w := range visible {
		shown[w.ID] = true
	}
	byUser := make(map[string][]*TeamMembership, len(accounts))
	for _, m := range memberships {
		if shown[m.WorkspaceID] {
			byUser[m.UserID] = append(byUser[m.UserID], m)
		}
	}
	people := make([]*TeamPerson, 0, len(accounts))
	for _, a := range accounts {
		held := byUser[a.ID]
		if held == nil && !everyone {
			continue
		}
		if held == nil {
			held = []*TeamMembership{}
		}
		person := &TeamPerson{TeamAccount: *a, Workspaces: held}
		person.AvatarURL = effectiveAvatarURL(a)
		people = append(people, person)
	}
	return people
}

// AddMember puts an existing account into workspaceID with a non-Owner role; requires members:write there.
func (s *Service) AddMember(ctx context.Context, actorID, workspaceID, userID, roleID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	roleID = strings.TrimSpace(roleID)
	if userID == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if roleID == "" {
		return fmt.Errorf("%w: role id is required", apperrs.ErrInvalid)
	}
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	isOwner, err := s.roleNames.IsOwnerRole(ctx, workspaceID, roleID)
	if err != nil {
		return fmt.Errorf("check owner role %s: %w", roleID, err)
	}
	if isOwner {
		return fmt.Errorf("%w: the Owner role cannot be assigned", apperrs.ErrInvalid)
	}
	if err := s.requireHoldsRole(ctx, actorID, workspaceID, roleID); err != nil {
		return err
	}
	account, err := s.accounts.Account(ctx, userID)
	if err != nil {
		return fmt.Errorf("get account %s: %w", userID, err)
	}
	if account.Status == AccountStatusRemoved {
		return fmt.Errorf("%w: %s's account is removed; restore it before adding them to a workspace", apperrs.ErrInvalid, account.Login)
	}
	_, err = s.members.RoleIDFor(ctx, workspaceID, userID)
	if err == nil {
		return fmt.Errorf("%w: %s is already a member of this workspace; change their role instead", apperrs.ErrConflict, account.Login)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("check membership of %s in workspace %s: %w", userID, workspaceID, err)
	}
	member := &Member{UserID: userID, WorkspaceID: workspaceID, RoleID: roleID, CreatedAt: s.now().UTC()}
	if err := s.members.AddMember(ctx, member, memberEvent(TopicWorkspaceMemberAdded, actorID, workspaceID, userID)); err != nil {
		return fmt.Errorf("add member %s to workspace %s: %w", userID, workspaceID, err)
	}
	return nil
}

// SetMemberOverrides changes a member's workspace-wide allow and deny sets; a nil set keeps its current value.
func (s *Service) SetMemberOverrides(ctx context.Context, actorID, workspaceID, userID string, allow, deny *permissions.Set) error {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	if allow == nil && deny == nil {
		return fmt.Errorf("%w: allow or deny is required", apperrs.ErrInvalid)
	}
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	roleID, err := s.members.RoleIDFor(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("resolve role for %s in workspace %s: %w", userID, workspaceID, err)
	}
	isOwner, err := s.roleNames.IsOwnerRole(ctx, workspaceID, roleID)
	if err != nil {
		return fmt.Errorf("check owner role %s: %w", roleID, err)
	}
	if isOwner {
		return fmt.Errorf("%w: the workspace Owner bypasses permission overrides, so none can be set", apperrs.ErrInvalid)
	}
	currentAllow, currentDeny, err := s.members.Overrides(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("read overrides for %s in workspace %s: %w", userID, workspaceID, err)
	}
	nextAllow, nextDeny := currentAllow, currentDeny
	if allow != nil {
		nextAllow = permissions.SetOf(*allow...)
	}
	if deny != nil {
		nextDeny = permissions.SetOf(*deny...)
	}
	if err := validateOverrides(nextAllow, nextDeny); err != nil {
		return err
	}
	// Only what the change adds is a grant: a new allow, or a lifted deny, which hands back what the role gives.
	added := permissions.SetOf(slices.Concat(nextAllow.Except(currentAllow), currentDeny.Except(nextDeny))...)
	if err := s.requireHolds(ctx, actorID, workspaceID, added); err != nil {
		return err
	}
	if err := s.members.SetOverrides(ctx, workspaceID, userID, nextAllow, nextDeny, memberEvent(TopicWorkspaceMemberUpdated, actorID, workspaceID, userID)); err != nil {
		return fmt.Errorf("set overrides for %s in workspace %s: %w", userID, workspaceID, err)
	}
	return nil
}

func validateOverrides(allow, deny permissions.Set) error {
	for _, action := range slices.Concat(allow, deny) {
		if _, ok := permissions.ParseAction(string(action)); !ok {
			return fmt.Errorf("%w: %q is not a permission; the permission catalog lists them", apperrs.ErrInvalid, action)
		}
	}
	for _, action := range allow {
		if deny.Has(action) {
			return fmt.Errorf("%w: %s cannot be allowed and denied at once", apperrs.ErrInvalid, action)
		}
	}
	return nil
}

func memberEvent(topic, actorID, workspaceID, userID string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: MemberEvent{UserID: userID, WorkspaceID: workspaceID, ActorID: actorID}}
}

// ResolvePendingInvites binds pending invites to userID so the caller's next request sees the membership.
func (s *Service) ResolvePendingInvites(ctx context.Context, login, userID string) error {
	login = strings.ToLower(strings.TrimSpace(login))
	pending, err := s.invites.ListByLogin(ctx, login)
	if err != nil {
		return fmt.Errorf("list pending invites for %s: %w", login, err)
	}
	for _, inv := range pending {
		if err := s.members.AddMember(ctx, &Member{UserID: userID, WorkspaceID: inv.WorkspaceID, RoleID: inv.RoleID, CreatedAt: s.now().UTC()}); err != nil {
			return fmt.Errorf("resolve invite for %s to workspace %s: %w", login, inv.WorkspaceID, err)
		}
		if err := s.invites.Delete(ctx, inv.WorkspaceID, login); err != nil {
			return fmt.Errorf("delete resolved invite for %s in workspace %s: %w", login, inv.WorkspaceID, err)
		}
	}
	return nil
}

// Get returns a workspace by id.
func (s *Service) Get(ctx context.Context, id string) (*Workspace, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	w, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get workspace %s: %w", id, err)
	}
	return w, nil
}

// ListForUser returns the workspaces userID is a member of.
func (s *Service) ListForUser(ctx context.Context, userID string) ([]*Workspace, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	ws, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces for user %s: %w", userID, err)
	}
	return ws, nil
}
