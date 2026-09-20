package tenancy

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
	now       func() time.Time
}

// NewService wires the tenancy use-cases over their repos and permission gates.
func NewService(repo Repo, members MemberRepo, invites InviteRepo, roles RoleGate, perm PermissionGate, roleNames RoleNameGate, wsPerms WorkspacePermissionGate, allowlist AllowlistGate, users UserLookupGate, channels ChannelGate, plays PlaysGate) *Service {
	return &Service{repo: repo, members: members, invites: invites, roles: roles, perm: perm, roleNames: roleNames, wsPerms: wsPerms, allowlist: allowlist, users: users, channels: channels, plays: plays, now: time.Now}
}

// Create adds a workspace and binds userID as its first member; requires the can_create_workspace bit.
func (s *Service) Create(ctx context.Context, userID, name string) (*Workspace, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: workspace name is required", apperrs.ErrInvalid)
	}
	ok, err := s.perm.CanCreateWorkspace(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check create-workspace permission: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: can_create_workspace permission required", apperrs.ErrForbidden)
	}
	now := s.now().UTC()
	w := &Workspace{ID: ids.New(), Name: name, CreatedAt: now, UpdatedAt: now}
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

// Rename requires the same can_create_workspace bit as Create since it's an instance-admin action.
func (s *Service) Rename(ctx context.Context, userID, id, name string) (*Workspace, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: workspace name is required", apperrs.ErrInvalid)
	}
	ok, err := s.perm.CanCreateWorkspace(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("check create-workspace permission: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: can_create_workspace permission required", apperrs.ErrForbidden)
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := *current
	updated.Name = name
	updated.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("rename workspace %s: %w", id, err)
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

// requireManageWorkspaceMembers already covers Owner via WorkspacePermissionGate's bypass.
func (s *Service) requireManageWorkspaceMembers(ctx context.Context, actorID, workspaceID string) error {
	for _, p := range s.wsPerms.WorkspacePermissions(ctx, actorID, workspaceID) {
		if p == string(permissions.MembersWrite) {
			return nil
		}
	}
	return fmt.Errorf("%w: members:write permission required", apperrs.ErrForbidden)
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
	if err := s.members.RemoveMember(ctx, workspaceID, userID); err != nil {
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
	if err := s.members.SetRole(ctx, workspaceID, userID, roleID); err != nil {
		return fmt.Errorf("set role for %s in workspace %s: %w", userID, workspaceID, err)
	}
	return nil
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
