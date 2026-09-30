package roles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Service is the roles use-case layer: the workspace-scoped role catalog and its protected Owner role.
type Service struct {
	repo    Repo
	members MemberGate
	perms   PermissionGate
	now     func() time.Time
}

// NewService wires the roles use-cases; members may be nil, wired later via SetMemberGate to break a cycle.
func NewService(repo Repo, members MemberGate) *Service {
	return &Service{repo: repo, members: members, now: time.Now}
}

// SetMemberGate wires the tenancy-domain member lookup after both services exist (see NewService).
func (s *Service) SetMemberGate(g MemberGate) {
	s.members = g
}

// SetPermissionGate wires the access domain's per-workspace grid; unset, a role can carry no permission at all.
func (s *Service) SetPermissionGate(g PermissionGate) {
	s.perms = g
}

// CreateOwnerRole creates workspaceID's protected Owner role; it bypasses every check via IsOwnerRole, not its set.
func (s *Service) CreateOwnerRole(ctx context.Context, workspaceID string) (*Role, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	now := s.now().UTC()
	r := &Role{
		ID:          ids.New(),
		WorkspaceID: workspaceID,
		Name:        "Owner",
		IsOwnerRole: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("create owner role for workspace %s: %w", workspaceID, err)
	}
	return r, nil
}

// Create makes a new custom role in workspaceID; actorUserID must hold roles:write or the Owner bypass.
func (s *Service) Create(ctx context.Context, workspaceID, actorUserID, name string, perms permissions.Set) (*Role, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	name = strings.TrimSpace(name)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: role name is required", apperrs.ErrInvalid)
	}
	if err := s.requireManageRoles(ctx, workspaceID, actorUserID); err != nil {
		return nil, err
	}
	if err := s.requireHolds(ctx, workspaceID, actorUserID, perms); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	r := &Role{
		ID:          ids.New(),
		WorkspaceID: workspaceID,
		Name:        name,
		Permissions: perms,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("create role in workspace %s: %w", workspaceID, err)
	}
	return r, nil
}

// Get returns a role scoped to workspaceID; a role from another workspace is reported not found, never leaked.
func (s *Service) Get(ctx context.Context, workspaceID, roleID string) (*Role, error) {
	r, err := s.getInWorkspace(ctx, workspaceID, roleID)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// List returns every role in workspaceID (Owner included).
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Role, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	rs, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list roles for workspace %s: %w", workspaceID, err)
	}
	return rs, nil
}

// ListForMember lists workspaceID's roles for one of its members; anyone else is told it does not exist.
func (s *Service) ListForMember(ctx context.Context, workspaceID, actorUserID string) ([]*Role, error) {
	if err := s.requireMember(ctx, workspaceID, actorUserID); err != nil {
		return nil, err
	}
	return s.List(ctx, workspaceID)
}

// GetForMember reads one of workspaceID's roles for one of its members; anyone else is told it does not exist.
func (s *Service) GetForMember(ctx context.Context, workspaceID, roleID, actorUserID string) (*Role, error) {
	if err := s.requireMember(ctx, workspaceID, actorUserID); err != nil {
		return nil, err
	}
	return s.Get(ctx, workspaceID, roleID)
}

// requireMember answers not found for a caller outside workspaceID, so its roles never leak to other workspaces.
func (s *Service) requireMember(ctx context.Context, workspaceID, actorUserID string) error {
	if strings.TrimSpace(actorUserID) == "" {
		return fmt.Errorf("%w: actor user id is required", apperrs.ErrInvalid)
	}
	if _, err := s.members.MemberRoleID(ctx, workspaceID, actorUserID); err != nil {
		if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) {
			return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
		}
		return fmt.Errorf("resolve membership in workspace %s: %w", workspaceID, err)
	}
	return nil
}

// Update renames a custom role and/or replaces its permissions; the Owner role can't be renamed or edited.
func (s *Service) Update(ctx context.Context, workspaceID, roleID, actorUserID, name string, perms permissions.Set) (*Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: role name is required", apperrs.ErrInvalid)
	}
	r, err := s.getInWorkspace(ctx, workspaceID, roleID)
	if err != nil {
		return nil, err
	}
	if r.IsOwnerRole {
		return nil, fmt.Errorf("%w: the Owner role can't be renamed or edited", apperrs.ErrInvalid)
	}
	if err := s.requireManageRoles(ctx, workspaceID, actorUserID); err != nil {
		return nil, err
	}
	// Only what the edit adds is a grant; a permission the role already carried may stay without the editor holding it.
	if err := s.requireHolds(ctx, workspaceID, actorUserID, perms.Except(r.Permissions)); err != nil {
		return nil, err
	}
	r.Name = name
	r.Permissions = perms
	r.UpdatedAt = s.now().UTC()
	event := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicUpdated, Payload: RoleEvent{RoleID: r.ID, WorkspaceID: r.WorkspaceID, ActorID: actorUserID}}
	if err := s.repo.Update(ctx, r, event); err != nil {
		return nil, fmt.Errorf("update role %s: %w", roleID, err)
	}
	return r, nil
}

// Delete removes a custom role; the Owner role can't be deleted.
func (s *Service) Delete(ctx context.Context, workspaceID, roleID, actorUserID string) error {
	r, err := s.getInWorkspace(ctx, workspaceID, roleID)
	if err != nil {
		return err
	}
	if r.IsOwnerRole {
		return fmt.Errorf("%w: the Owner role can't be deleted", apperrs.ErrInvalid)
	}
	if err := s.requireManageRoles(ctx, workspaceID, actorUserID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, roleID); err != nil {
		return fmt.Errorf("delete role %s: %w", roleID, err)
	}
	return nil
}

// Clone copies a custom role into another workspace, needing roles:clone in the source ("" = the role's own) and roles:write in the target.
func (s *Service) Clone(ctx context.Context, sourceWorkspaceID, roleID, targetWorkspaceID, actorUserID string) (*Role, error) {
	targetWorkspaceID = strings.TrimSpace(targetWorkspaceID)
	if targetWorkspaceID == "" {
		return nil, fmt.Errorf("%w: target workspace id is required", apperrs.ErrInvalid)
	}
	source, err := s.cloneSource(ctx, sourceWorkspaceID, roleID)
	if err != nil {
		return nil, err
	}
	if err := s.requireAction(ctx, source.WorkspaceID, actorUserID, permissions.RolesClone); err != nil {
		return nil, err
	}
	if source.IsOwnerRole {
		return nil, fmt.Errorf("%w: the Owner role can't be cloned; every workspace already has its own", apperrs.ErrInvalid)
	}
	if targetWorkspaceID == source.WorkspaceID {
		return nil, fmt.Errorf("%w: pick a workspace other than the one the role is in", apperrs.ErrInvalid)
	}
	if err := s.requireAction(ctx, targetWorkspaceID, actorUserID, permissions.RolesWrite); err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return nil, fmt.Errorf("%w: you aren't a member of the target workspace", apperrs.ErrForbidden)
		}
		return nil, err
	}
	if err := s.requireHolds(ctx, targetWorkspaceID, actorUserID, source.Permissions); err != nil {
		return nil, err
	}
	taken, err := s.repo.List(ctx, targetWorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list roles for workspace %s: %w", targetWorkspaceID, err)
	}
	now := s.now().UTC()
	r := &Role{
		ID:          ids.New(),
		WorkspaceID: targetWorkspaceID,
		Name:        freeName(source.Name, taken),
		Permissions: permissions.SetOf(source.Permissions...),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, fmt.Errorf("clone role %s into workspace %s: %w", roleID, targetWorkspaceID, err)
	}
	return r, nil
}

func (s *Service) cloneSource(ctx context.Context, sourceWorkspaceID, roleID string) (*Role, error) {
	if strings.TrimSpace(sourceWorkspaceID) != "" {
		return s.getInWorkspace(ctx, sourceWorkspaceID, roleID)
	}
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return nil, fmt.Errorf("%w: role id is required", apperrs.ErrInvalid)
	}
	r, err := s.repo.Get(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("get role %s: %w", roleID, err)
	}
	return r, nil
}

// freeName returns name, or the first "<name> (copy)", "<name> (copy 2)", ... no role in taken already uses.
func freeName(name string, taken []*Role) string {
	used := make(map[string]bool, len(taken))
	for _, r := range taken {
		used[strings.ToLower(r.Name)] = true
	}
	candidate := name
	for n := 1; used[strings.ToLower(candidate)]; n++ {
		candidate = name + " (copy)"
		if n > 1 {
			candidate = fmt.Sprintf("%s (copy %d)", name, n)
		}
	}
	return candidate
}

func (s *Service) getInWorkspace(ctx context.Context, workspaceID, roleID string) (*Role, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	roleID = strings.TrimSpace(roleID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if roleID == "" {
		return nil, fmt.Errorf("%w: role id is required", apperrs.ErrInvalid)
	}
	r, err := s.repo.Get(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("get role %s: %w", roleID, err)
	}
	if r.WorkspaceID != workspaceID {
		return nil, fmt.Errorf("get role %s: %w", roleID, apperrs.ErrNotFound)
	}
	return r, nil
}

// requireHolds refuses a role carrying an action actorUserID does not hold in workspaceID (ADR 0088).
func (s *Service) requireHolds(ctx context.Context, workspaceID, actorUserID string, grant permissions.Set) error {
	var held permissions.Set
	if s.perms != nil {
		held = permissions.SetOfStrings(s.perms.WorkspacePermissions(ctx, actorUserID, workspaceID))
	}
	return permissions.RequireHeld(grant, held)
}

// requireManageRoles checks the Owner bypass, then roles:write, gating Create/Update/Delete on custom roles.
func (s *Service) requireManageRoles(ctx context.Context, workspaceID, actorUserID string) error {
	return s.requireAction(ctx, workspaceID, actorUserID, permissions.RolesWrite)
}

// requireAction checks the Owner bypass, then action on the actor's role in workspaceID.
func (s *Service) requireAction(ctx context.Context, workspaceID, actorUserID string, action permissions.Action) error {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return fmt.Errorf("%w: actor user id is required", apperrs.ErrInvalid)
	}
	actorRoleID, err := s.members.MemberRoleID(ctx, workspaceID, actorUserID)
	if err != nil {
		return fmt.Errorf("resolve actor role in workspace %s: %w", workspaceID, err)
	}
	actorRole, err := s.repo.Get(ctx, actorRoleID)
	if err != nil {
		return fmt.Errorf("load actor role %s: %w", actorRoleID, err)
	}
	if actorRole.IsOwnerRole {
		return nil
	}
	if !actorRole.Permissions.Has(action) {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
	}
	return nil
}
