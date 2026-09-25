// Package access implements permission overwrites, the HasPermission check, and doc-permission use-cases.
package access

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// resourceTypeDoc is the resource_type value documents use in permission_overwrites.
const resourceTypeDoc = "doc"

// resourceTypePlay is the resource_type value plays use for per-user exclusion (ticket 21): a deny-only
// overwrite of plays:run, managed from the play's settings page with the doc sharing dialog generalised.
const resourceTypePlay = "play"

// resourceTypeWorkspace keys a workspace-wide override by resource_id = the workspace id.
const resourceTypeWorkspace = "workspace"

// Service is the access use-case layer (ADR 0019): permission overwrites and the HasPermission/Can checks.
type Service struct {
	repo          Repo
	users         Users
	roles         RoleResolver
	docWorkspace  DocWorkspaceResolver
	playWorkspace PlayWorkspaceResolver
	now           func() time.Time
}

// NewService wires the access use-cases over the given repo and user store.
func NewService(repo Repo, users Users) *Service {
	return &Service{repo: repo, users: users, now: time.Now}
}

// SetRoles wires the roles/tenancy domains' RoleResolver after both exist (see RoleResolver's doc comment).
func (s *Service) SetRoles(r RoleResolver) {
	s.roles = r
}

// SetDocWorkspaces wires the docs+workspace domains' DocWorkspaceResolver.
func (s *Service) SetDocWorkspaces(r DocWorkspaceResolver) {
	s.docWorkspace = r
}

// SetPlayWorkspaces wires the plays domain's PlayWorkspaceResolver.
func (s *Service) SetPlayWorkspaces(r PlayWorkspaceResolver) {
	s.playWorkspace = r
}

// HasPermission checks Owner bypass, then role set, then workspace overwrite, then resource overwrite, in order.
func (s *Service) HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action, resourceType, resourceID string) bool {
	if userID == "" {
		return false
	}
	ws := s.workspaceLayers(ctx, userID, workspaceID)
	if ws.owner {
		return true
	}
	allowed := ws.has(action)
	if resourceType != "" && resourceID != "" {
		if ow, err := s.repo.Get(ctx, resourceType, resourceID, userID); err == nil {
			allowed = applyOverwrite(allowed, ow, action)
		}
	}
	return allowed
}

// workspaceLayers is the role set and workspace-wide overwrite for one user, fetched once so a
// whole-grid answer (WorkspacePermissions) costs the same lookups as a single check.
type workspaceLayers struct {
	owner     bool
	role      permissions.Set
	overwrite *Overwrite
}

func (s *Service) workspaceLayers(ctx context.Context, userID, workspaceID string) workspaceLayers {
	var ws workspaceLayers
	if workspaceID == "" {
		return ws
	}
	if s.roles != nil {
		if info, err := s.roles.MemberRole(ctx, workspaceID, userID); err == nil {
			ws.owner = info.IsOwnerRole
			ws.role = info.Permissions
		}
	}
	if ow, err := s.repo.Get(ctx, resourceTypeWorkspace, workspaceID, userID); err == nil {
		ws.overwrite = ow
	}
	return ws
}

func (ws workspaceLayers) has(action permissions.Action) bool {
	allowed := ws.role.Has(action)
	if ws.overwrite == nil {
		return allowed
	}
	return applyOverwrite(allowed, ws.overwrite, action)
}

// WorkspacePermissions returns every grid action userID holds in workspaceID (feeds /api/workspaces/{id}/me).
func (s *Service) WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string {
	if userID == "" {
		return []string{}
	}
	ws := s.workspaceLayers(ctx, userID, workspaceID)
	all := permissions.AllActions()
	out := make([]string, 0, len(all))
	for _, a := range all {
		if ws.owner || ws.has(a) {
			out = append(out, string(a))
		}
	}
	return out
}

// applyOverwrite: deny wins over allow within a row; a row naming neither leaves the state untouched.
func applyOverwrite(allowed bool, ow *Overwrite, action permissions.Action) bool {
	if ow.Deny.Has(action) {
		return false
	}
	if ow.Allow.Has(action) {
		return true
	}
	return allowed
}

// Can reports whether userID may perform action on docID.
func (s *Service) Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error) {
	if userID == "" {
		return false, nil
	}
	return s.HasPermission(ctx, userID, s.resolveDocWorkspace(ctx, docID), action, resourceTypeDoc, docID), nil
}

// resolveDocWorkspace resolves docID's workspace via its project for HasPermission's workspace-scoped layers.
func (s *Service) resolveDocWorkspace(ctx context.Context, docID string) string {
	if s.docWorkspace == nil {
		return ""
	}
	workspaceID, err := s.docWorkspace.WorkspaceIDForDoc(ctx, docID)
	if err != nil {
		return ""
	}
	return workspaceID
}

// resolvePlayWorkspace resolves playID's own workspace (a play carries it directly, unlike a doc).
func (s *Service) resolvePlayWorkspace(ctx context.Context, playID string) string {
	if s.playWorkspace == nil {
		return ""
	}
	workspaceID, err := s.playWorkspace.WorkspaceIDForPlay(ctx, playID)
	if err != nil {
		return ""
	}
	return workspaceID
}

// GrantCreator grants the document creator CreatorGrant, called from the docs domain's Create use-case.
func (s *Service) GrantCreator(ctx context.Context, docID, creatorID string) error {
	if docID == "" || creatorID == "" {
		return fmt.Errorf("%w: doc and creator are required", apperrs.ErrInvalid)
	}
	if err := s.repo.Set(ctx, resourceTypeDoc, docID, creatorID, permissions.CreatorGrant, nil); err != nil {
		return fmt.Errorf("grant creator %s on %s: %w", creatorID, docID, err)
	}
	return nil
}

// DeleteByDoc has no FK to docs to rely on since resource_id spans every resource type.
func (s *Service) DeleteByDoc(ctx context.Context, docID string) error {
	if err := s.repo.DeleteByResource(ctx, resourceTypeDoc, docID); err != nil {
		return fmt.Errorf("delete overwrites %s: %w", docID, err)
	}
	return nil
}

// SetGrants applies actions to every (docID, userID) pair in one bulk operation (the permissions modal).
func (s *Service) SetGrants(ctx context.Context, actorID string, docIDs, userIDs []string, actions []permissions.Action, grant bool) error {
	return s.setGrants(ctx, actorID, resourceTypeDoc, docIDs, userIDs, actions, grant)
}

// SetPlayGrants denies or un-denies plays:run for users on plays — the exclusion dialog (ticket 21). Only
// plays:run is a meaningful entry on a play; any other action is rejected.
func (s *Service) SetPlayGrants(ctx context.Context, actorID string, playIDs, userIDs []string, actions []permissions.Action, grant bool) error {
	for _, a := range actions {
		if a != permissions.PlaysRun {
			return fmt.Errorf("%w: only plays:run may be set on a play", apperrs.ErrInvalid)
		}
	}
	return s.setGrants(ctx, actorID, resourceTypePlay, playIDs, userIDs, actions, grant)
}

func (s *Service) setGrants(ctx context.Context, actorID, resourceType string, resourceIDs, userIDs []string, actions []permissions.Action, grant bool) error {
	if actorID == "" {
		return fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	if len(resourceIDs) == 0 || len(userIDs) == 0 || len(actions) == 0 {
		return fmt.Errorf("%w: at least one resource, one user, and one action are required", apperrs.ErrInvalid)
	}
	for _, id := range resourceIDs {
		ok, err := s.canManage(ctx, actorID, resourceType, id)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: manage permission required on %s %s", apperrs.ErrForbidden, resourceType, id)
		}
	}
	for _, id := range resourceIDs {
		for _, userID := range userIDs {
			if err := s.applyGrant(ctx, resourceType, id, userID, actions, grant); err != nil {
				return err
			}
		}
	}
	return nil
}

// ListGrants returns the overwrites on a doc, owner- or permissions:write-gated.
func (s *Service) ListGrants(ctx context.Context, actorID, docID string) ([]*Overwrite, error) {
	return s.listGrants(ctx, actorID, resourceTypeDoc, docID)
}

// ListPlayGrants returns the exclusion overwrites on a play, for its settings page (ticket 21).
func (s *Service) ListPlayGrants(ctx context.Context, actorID, playID string) ([]*Overwrite, error) {
	return s.listGrants(ctx, actorID, resourceTypePlay, playID)
}

func (s *Service) listGrants(ctx context.Context, actorID, resourceType, resourceID string) ([]*Overwrite, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	ok, err := s.canManage(ctx, actorID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: manage permission required on %s %s", apperrs.ErrForbidden, resourceType, resourceID)
	}
	grants, err := s.repo.ListByResource(ctx, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list overwrites %s/%s: %w", resourceType, resourceID, err)
	}
	return grants, nil
}

// ListUsers is gated to an instance admin or anyone holding permissions:write on at least one document.
func (s *Service) ListUsers(ctx context.Context, actorID string) ([]*User, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	u, err := s.users.GetUserByID(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("resolve user %s: %w", actorID, err)
	}
	if !u.CanCreateWorkspace {
		has, err := s.repo.HasAllowAny(ctx, resourceTypeDoc, actorID, permissions.PermissionsWrite)
		if err != nil {
			return nil, fmt.Errorf("check permissions:write for %s: %w", actorID, err)
		}
		if !has {
			return nil, fmt.Errorf("%w: owner or permissions:write required", apperrs.ErrForbidden)
		}
	}
	users, err := s.users.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// accountsByID indexes every account by id, so a listed overwrite can say whose it is.
func (s *Service) accountsByID(ctx context.Context) (map[string]*User, error) {
	users, err := s.users.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	byID := make(map[string]*User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	return byID, nil
}

// InstanceAdminID returns "" when no instance admin exists yet; used to resolve the MCP acting user.
func (s *Service) InstanceAdminID(ctx context.Context) (string, error) {
	users, err := s.users.ListUsers(ctx)
	if err != nil {
		return "", fmt.Errorf("list users: %w", err)
	}
	for _, u := range users {
		if u.CanCreateWorkspace {
			return u.ID, nil
		}
	}
	return "", nil
}

// canManage grants no instance-wide bypass, since that would cross workspace isolation boundaries. A doc's
// manage action is the cross-cutting permissions:write share grant; a play's is plays:write, the same bit
// that gates editing the play's own definition.
func (s *Service) canManage(ctx context.Context, actorID, resourceType, resourceID string) (bool, error) {
	if resourceType == resourceTypePlay {
		workspaceID := s.resolvePlayWorkspace(ctx, resourceID)
		return s.HasPermission(ctx, actorID, workspaceID, permissions.PlaysWrite, resourceTypePlay, resourceID), nil
	}
	workspaceID := s.resolveDocWorkspace(ctx, resourceID)
	return s.HasPermission(ctx, actorID, workspaceID, permissions.PermissionsWrite, resourceTypeDoc, resourceID), nil
}

// applyGrant maintains allow for a doc (default-deny: an explicit allow is what grants access) and deny for
// a play (default-allow via the role grid: an explicit deny is the exclusion; grant=true clears it).
func (s *Service) applyGrant(ctx context.Context, resourceType, resourceID, userID string, actions []permissions.Action, grant bool) error {
	var allow, deny permissions.Set
	existing, err := s.repo.Get(ctx, resourceType, resourceID, userID)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("get overwrite %s/%s/%s: %w", resourceType, resourceID, userID, err)
	}
	if err == nil {
		allow, deny = existing.Allow, existing.Deny
	}
	for _, a := range actions {
		if resourceType == resourceTypePlay {
			if grant {
				deny = deny.Without(a)
				continue
			}
			deny = deny.With(a)
			continue
		}
		if grant {
			allow = allow.With(a)
			deny = deny.Without(a)
			continue
		}
		allow = allow.Without(a)
	}
	return s.repo.Set(ctx, resourceType, resourceID, userID, allow, deny)
}
