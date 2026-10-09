// Package access implements permission overwrites, the HasPermission check, and doc-permission use-cases.
package access

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/wake"
)

// resourceTypeDoc is the resource_type value documents use in permission_overwrites.
const resourceTypeDoc = "doc"

// resourceTypePlay is the resource_type value plays use for per-user exclusion (ticket 21): a deny-only
// overwrite of plays:run, managed from the play's settings page with the doc sharing dialog generalised.
const resourceTypePlay = "play"

// resourceTypeWorkspace keys a workspace-wide override by resource_id = the workspace id.
const resourceTypeWorkspace = "workspace"

// resourceTypeProject keys a Restricted member's Project access by resource_id = the project id (ADR 0097).
const resourceTypeProject = "project"

// Service is the access use-case layer (ADR 0019): permission overwrites and the HasPermission/Can checks.
type Service struct {
	repo          Repo
	users         Users
	roles         RoleResolver
	docWorkspace  DocWorkspaceResolver
	playWorkspace PlayWorkspaceResolver
	scopes        Scopes
	commits       *wake.Broadcast
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

// HasPermission checks Owner bypass, then role set, then workspace overwrite, then resource overwrite, in order. For
// a Restricted member a doc or "project" resource puts the project's access between the workspace and the resource.
func (s *Service) HasPermission(ctx context.Context, userID, workspaceID string, action permissions.Action, resourceType, resourceID string) bool {
	if userID == "" {
		return false
	}
	projectID := ""
	switch resourceType {
	case resourceTypeProject:
		projectID = resourceID
	case resourceTypeDoc:
		_, projectID = s.resolveDoc(ctx, resourceID)
	}
	return s.check(ctx, userID, workspaceID, projectID, action, resourceType, resourceID)
}

func (s *Service) check(ctx context.Context, userID, workspaceID, projectID string, action permissions.Action, resourceType, resourceID string) bool {
	return decide(s.layers(ctx, userID, workspaceID, projectID), action, func() *Overwrite {
		if resourceType == "" || resourceType == resourceTypeProject || resourceID == "" {
			return nil
		}
		ow, err := s.overwrite(ctx, resourceType, resourceID, userID)
		if err != nil {
			return nil
		}
		return ow
	})
}

// decide answers a check from layers already read; resource, the resource's own overwrite, is read only when the
// layers leave the answer open.
func decide(ws workspaceLayers, action permissions.Action, resource func() *Overwrite) bool {
	if ws.owner {
		return true
	}
	if ws.hidden() {
		return false
	}
	if action == permissions.Member {
		return ws.member
	}
	allowed := ws.has(action)
	if ow := resource(); ow != nil {
		allowed = applyOverwrite(allowed, ow, action)
	}
	return allowed
}

// workspaceLayers is the role set, workspace-wide overwrite, and for a Restricted member the Project access of the
// project asked about, fetched once so a whole-grid answer costs the same lookups as a single check.
type workspaceLayers struct {
	member     bool
	owner      bool
	restricted bool
	role       permissions.Set
	overwrite  *Overwrite
	inProject  bool
	project    permissions.Set
}

func (s *Service) workspaceLayers(ctx context.Context, userID, workspaceID string) workspaceLayers {
	var ws workspaceLayers
	if workspaceID == "" {
		return ws
	}
	if s.roles != nil {
		if info, err := s.memberRole(ctx, workspaceID, userID); err == nil {
			ws.member = true
			ws.owner = info.IsOwnerRole
			ws.restricted = info.Restricted && !info.IsOwnerRole
			ws.role = info.Permissions
		}
	}
	if ow, err := s.overwrite(ctx, resourceTypeWorkspace, workspaceID, userID); err == nil {
		ws.overwrite = ow
	}
	return ws
}

// layers adds projectID's Project access to a Restricted member's workspace layers; the rows are read only while the
// member is restricted, so a From role member's stored rows change nothing.
func (s *Service) layers(ctx context.Context, userID, workspaceID, projectID string) workspaceLayers {
	ws := s.workspaceLayers(ctx, userID, workspaceID)
	if !ws.restricted || projectID == "" {
		return ws
	}
	ws.inProject = true
	if ow, err := s.overwrite(ctx, resourceTypeProject, projectID, userID); err == nil {
		ws.project = ow.Allow
	}
	return ws
}

// hidden is a project a Restricted member holds no access to: it reads as not found, whatever else would allow.
func (ws workspaceLayers) hidden() bool {
	return ws.restricted && ws.inProject && len(ws.project) == 0
}

func (ws workspaceLayers) has(action permissions.Action) bool {
	if ws.restricted {
		switch permissions.AreaOf(action) {
		case permissions.AreaInstance:
			return false
		case permissions.AreaProject:
			return ws.project.Has(action)
		}
	}
	allowed := ws.role.Has(action)
	if ws.overwrite == nil {
		return allowed
	}
	return applyOverwrite(allowed, ws.overwrite, action)
}

func (ws workspaceLayers) actions() []string {
	all := permissions.AllActions()
	out := make([]string, 0, len(all))
	for _, a := range all {
		if ws.owner || ws.has(a) {
			out = append(out, string(a))
		}
	}
	return out
}

// WorkspacePermissions returns every grid action userID holds in workspaceID with no project named, which for a
// Restricted member leaves out every project and instance area (feeds /api/workspaces/{id}/me).
func (s *Service) WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string {
	if userID == "" {
		return []string{}
	}
	return s.workspaceLayers(ctx, userID, workspaceID).actions()
}

// ProjectPermissions returns every grid action userID holds inside projectID; opens is false when the project is
// unknown, outside their workspaces, or hidden from them.
func (s *Service) ProjectPermissions(ctx context.Context, userID, projectID string) (actions []string, opens bool) {
	if userID == "" {
		return []string{}, false
	}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		return []string{}, false
	}
	ws := s.layers(ctx, userID, workspaceID, projectID)
	if !ws.member || (!ws.owner && ws.hidden()) {
		return []string{}, false
	}
	return ws.actions(), true
}

// CanInProject reports whether userID holds action inside projectID; the empty action asks whether they may open it.
func (s *Service) CanInProject(ctx context.Context, userID, projectID string, action permissions.Action) bool {
	if userID == "" {
		return false
	}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		return false
	}
	return s.check(ctx, userID, workspaceID, projectID, action, "", "")
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

// Can reports whether userID may perform action on docID; in a project hidden from a Restricted member the doc's
// own overwrite allows nothing.
func (s *Service) Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error) {
	return s.CanDocs(ctx, userID, "", []string{docID}, action)[docID], nil
}

// CanDocs answers Can for each of docIDs from one read of userID's overwrites on them, with each project's layers
// read once; a list of docs would otherwise pay for them per row. projectID, when every doc is in it, saves reading
// where each doc lives; empty, that is read for all of them at once.
func (s *Service) CanDocs(ctx context.Context, userID, projectID string, docIDs []string, action permissions.Action) map[string]bool {
	out := make(map[string]bool, len(docIDs))
	if userID == "" || len(docIDs) == 0 {
		return out
	}
	ctx = s.memoized(ctx)
	scopes := s.docScopesIn(ctx, projectID, docIDs)
	var own map[string]*Overwrite
	for _, id := range docIDs {
		scope := scopes[id]
		out[id] = decide(s.layers(ctx, userID, scope.WorkspaceID, scope.ProjectID), action, func() *Overwrite {
			if own == nil {
				own = s.docOverwrites(ctx, docIDs, userID)
			}
			return own[id]
		})
	}
	return out
}

// DocsWith is what a doc list filters by in SQL for action: the projects in which userID holds it on every doc without
// an overwrite of its own, and the docs whose own overwrite turns that answer, allowed in a project outside the set or
// denied in one inside it. A doc passes exactly when CanDocs would pass it (ADR 0140).
func (s *Service) DocsWith(ctx context.Context, userID string, action permissions.Action) (projectIDs, allowed, denied []string, err error) {
	allowed, denied = []string{}, []string{}
	ctx = s.memoized(ctx)
	_, projectIDs, err = s.ProjectsAnywhere(ctx, userID, action)
	if err != nil || userID == "" {
		return projectIDs, allowed, denied, err
	}
	own, err := s.repo.ListByUser(ctx, resourceTypeDoc, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("doc overwrites of %s: %w", userID, err)
	}
	docIDs := slices.Sorted(maps.Keys(own))
	scopes := s.docScopes(ctx, docIDs)
	for _, id := range docIDs {
		scope := scopes[id]
		inSet := scope.ProjectID != "" && slices.Contains(projectIDs, scope.ProjectID)
		can := decide(s.layers(ctx, userID, scope.WorkspaceID, scope.ProjectID), action, func() *Overwrite { return own[id] })
		if can && !inSet {
			allowed = append(allowed, id)
		}
		if !can && inSet {
			denied = append(denied, id)
		}
	}
	return projectIDs, allowed, denied, nil
}

// docOverwrites is userID's overwrite on each of docIDs that has one; a failed read counts as none, as a single
// check's does.
func (s *Service) docOverwrites(ctx context.Context, docIDs []string, userID string) map[string]*Overwrite {
	own, err := s.overwrites(ctx, resourceTypeDoc, docIDs, userID)
	if err != nil {
		return map[string]*Overwrite{}
	}
	return own
}

// resolveDoc resolves docID's workspace and project for the workspace- and project-scoped layers.
func (s *Service) resolveDoc(ctx context.Context, docID string) (workspaceID, projectID string) {
	scope := s.docScopes(ctx, []string{docID})[docID]
	return scope.WorkspaceID, scope.ProjectID
}

// docScopesIn is where each of docIDs lives: all in projectID when one is named, an unknown project counting as none.
func (s *Service) docScopesIn(ctx context.Context, projectID string, docIDs []string) map[string]DocScope {
	if projectID == "" {
		return s.docScopes(ctx, docIDs)
	}
	scope := DocScope{ProjectID: projectID}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		scope = DocScope{}
	}
	scope.WorkspaceID = workspaceID
	scopes := make(map[string]DocScope, len(docIDs))
	for _, id := range docIDs {
		scopes[id] = scope
	}
	return scopes
}

// docScopes is where each of docIDs lives; a doc it cannot resolve has an empty scope.
func (s *Service) docScopes(ctx context.Context, docIDs []string) map[string]DocScope {
	if s.docWorkspace == nil {
		return map[string]DocScope{}
	}
	scopes, err := rememberEach(ctx, docIDs, func(id string) any { return docKey{id} }, func(missing []string) (map[string]DocScope, error) {
		return s.docWorkspace.DocScopes(ctx, missing)
	})
	if err != nil {
		return map[string]DocScope{}
	}
	return scopes
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
			if err := s.applyGrant(ctx, actorID, resourceType, id, userID, actions, grant); err != nil {
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

// ListUsers is gated to a holder of accounts:read or anyone holding permissions:write on at least one document.
func (s *Service) ListUsers(ctx context.Context, actorID string) ([]*User, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	readsAccounts, err := s.HoldsAnywhere(ctx, actorID, permissions.AccountsRead)
	if err != nil {
		return nil, err
	}
	if !readsAccounts {
		has, err := s.repo.HasAllowAny(ctx, resourceTypeDoc, actorID, permissions.PermissionsWrite)
		if err != nil {
			return nil, fmt.Errorf("check permissions:write for %s: %w", actorID, err)
		}
		if !has {
			return nil, fmt.Errorf("%w: accounts:read or permissions:write required", apperrs.ErrForbidden)
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

// canManage grants no instance-wide bypass, since that would cross workspace isolation boundaries. A doc's
// manage action is the cross-cutting permissions:write share grant; a play's is plays:write, the same bit
// that gates editing the play's own definition.
func (s *Service) canManage(ctx context.Context, actorID, resourceType, resourceID string) (bool, error) {
	if resourceType == resourceTypePlay {
		workspaceID := s.resolvePlayWorkspace(ctx, resourceID)
		return s.HasPermission(ctx, actorID, workspaceID, permissions.PlaysWrite, resourceTypePlay, resourceID), nil
	}
	return s.Can(ctx, actorID, resourceID, permissions.PermissionsWrite)
}

// applyGrant maintains allow for a doc (default-deny: an explicit allow is what grants access) and deny for
// a play (default-allow via the role grid: an explicit deny is the exclusion; grant=true clears it).
func (s *Service) applyGrant(ctx context.Context, actorID, resourceType, resourceID, userID string, actions []permissions.Action, grant bool) error {
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
	event := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicGrantChanged, Payload: GrantEvent{ResourceType: resourceType, ResourceID: resourceID, UserID: userID, ActorID: actorID}}
	return s.repo.Set(ctx, resourceType, resourceID, userID, allow, deny, event)
}
