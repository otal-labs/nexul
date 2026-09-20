package roles

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Service is the roles use-case layer: the workspace-scoped role catalog and its protected Owner role.
type Service struct {
	repo    Repo
	members MemberGate
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
	r.Name = name
	r.Permissions = perms
	r.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, r); err != nil {
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

// requireManageRoles checks the Owner bypass, then roles:write, gating Create/Update/Delete on custom roles.
func (s *Service) requireManageRoles(ctx context.Context, workspaceID, actorUserID string) error {
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
	if !actorRole.Permissions.Has(permissions.RolesWrite) {
		return fmt.Errorf("%w: roles:write required", apperrs.ErrForbidden)
	}
	return nil
}
