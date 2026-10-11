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

// resourceTypeChannel keys a channel's rules by resource_id = the conversation id (ADR 0148).
const resourceTypeChannel = "channel"

// Service is the access use-case layer (ADR 0019): permission overwrites and the HasPermission/Can checks.
type Service struct {
	repo          Repo
	users         Users
	roles         RoleResolver
	docWorkspace  DocWorkspaceResolver
	playWorkspace PlayWorkspaceResolver
	scopes        Scopes
	names         Names
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
	held, _ := s.check(ctx, userID, workspaceID, projectID, action, resourceType, resourceID)
	return held
}

func (s *Service) check(ctx context.Context, userID, workspaceID, projectID string, action permissions.Action, resourceType, resourceID string) (bool, Source) {
	return decide(s.layers(ctx, userID, workspaceID, projectID), action, func() *Rules {
		if resourceType == "" || resourceType == resourceTypeProject || resourceID == "" {
			return nil
		}
		return s.rules(ctx, resourceType, resourceID, userID)
	})
}

// decide answers a check from layers already read, with what decided it as ids; resource, the resource's own rules,
// is read only when the layers leave the answer open.
func decide(ws workspaceLayers, action permissions.Action, resource func() *Rules) (bool, Source) {
	if ws.owner {
		return true, sourceOf(SourceOwner, resourceTypeWorkspace, ws.workspaceID, TargetRole, ws.roles.sourceRole(""), true)
	}
	if ws.hidden() {
		return false, sourceOf(SourceNotInProject, resourceTypeProject, ws.projectID, TargetPerson, ws.userID, false)
	}
	if action == permissions.Member {
		return ws.member, sourceOf(SourceMember, resourceTypeWorkspace, ws.workspaceID, TargetPerson, ws.userID, ws.member)
	}
	allowed, src := ws.has(action)
	return applyRules(allowed, src, resource(), ws.roles.roleIDs(), action)
}

// workspaceLayers is the role set, the workspace-wide overwrite, and for a Restricted member the Project access of the
// project asked about, fetched once so a whole-grid answer costs the same lookups as a single check.
type workspaceLayers struct {
	userID      string
	workspaceID string
	projectID   string
	roles       RoleInfo
	member      bool
	owner       bool
	restricted  bool
	overwrite   *Overwrite
	inProject   bool
	project     permissions.Set
}

func (s *Service) workspaceLayers(ctx context.Context, userID, workspaceID string) workspaceLayers {
	ws := workspaceLayers{userID: userID, workspaceID: workspaceID}
	if workspaceID == "" {
		return ws
	}
	if s.roles != nil {
		if info, err := s.memberRole(ctx, workspaceID, userID); err == nil {
			ws.member = true
			ws.owner = info.IsOwnerRole
			ws.restricted = info.Restricted && !info.IsOwnerRole
			ws.roles = info
		}
	}
	// The workspace itself carries no rule for a role (ADR 0148), so its person's row is read alone.
	if ow, err := s.overwrite(ctx, resourceTypeWorkspace, workspaceID, userID); err == nil {
		ws.overwrite = ow
	}
	return ws
}

// layers adds projectID's Project access to a Restricted member's workspace layers; the rows are read only while the
// member is restricted, so a From role member's stored rows change nothing.
func (s *Service) layers(ctx context.Context, userID, workspaceID, projectID string) workspaceLayers {
	ws := s.workspaceLayers(ctx, userID, workspaceID)
	ws.projectID = projectID
	if !ws.restricted || projectID == "" {
		return ws
	}
	ws.inProject = true
	if r := s.rules(ctx, resourceTypeProject, projectID, userID); r != nil && r.Person != nil {
		ws.project = r.Person.Allow
	}
	return ws
}

// hidden is a project a Restricted member holds no access to: it reads as not found, whatever else would allow.
func (ws workspaceLayers) hidden() bool {
	return ws.restricted && ws.inProject && len(ws.project) == 0
}

func sourceOf(kind, entityType, entityID, targetType, targetID string, allowed bool) Source {
	state := StateDeny
	if allowed {
		state = StateAllow
	}
	return Source{Kind: kind, EntityType: entityType, EntityID: entityID, TargetType: targetType, TargetID: targetID, State: state}
}

func (ws workspaceLayers) has(action permissions.Action) (bool, Source) {
	if ws.restricted {
		switch permissions.AreaOf(action) {
		case permissions.AreaInstance:
			return false, sourceOf(SourceRestricted, resourceTypeWorkspace, ws.workspaceID, TargetPerson, ws.userID, false)
		case permissions.AreaProject:
			held := ws.project.Has(action)
			if !ws.inProject {
				return held, sourceOf(SourceRestricted, resourceTypeWorkspace, ws.workspaceID, TargetPerson, ws.userID, held)
			}
			return held, sourceOf(SourceRestricted, resourceTypeProject, ws.projectID, TargetPerson, ws.userID, held)
		}
	}
	allowed := ws.roles.Permissions.Has(action)
	src := sourceOf(SourceRole, resourceTypeWorkspace, ws.workspaceID, TargetRole, ws.roles.sourceRole(action), allowed)
	if !ws.member {
		src = sourceOf(SourceMember, resourceTypeWorkspace, ws.workspaceID, TargetPerson, ws.userID, false)
	}
	return applyPersonRule(allowed, src, ws.overwrite, action)
}

func (ws workspaceLayers) actions() []string {
	all := permissions.AllActions()
	out := make([]string, 0, len(all))
	for _, a := range all {
		if held, _ := ws.has(a); ws.owner || held {
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
	held, _ := s.check(ctx, userID, workspaceID, projectID, action, "", "")
	return held
}

// applyRules lays an entity's rules over the answer from further out: the rules for the roles held, of which Allow
// wins, then the rule naming the person, which beats them (ADR 0148). A rule that sets the action becomes the source;
// one naming neither leaves the answer and its source untouched.
func applyRules(allowed bool, src Source, r *Rules, roleIDs []string, action permissions.Action) (bool, Source) {
	if r == nil {
		return allowed, src
	}
	allowed, src = applyRoleRules(allowed, src, r.Roles, roleIDs, action)
	return applyPersonRule(allowed, src, r.Person, action)
}

func applyPersonRule(allowed bool, src Source, ow *Overwrite, action permissions.Action) (bool, Source) {
	if ow == nil {
		return allowed, src
	}
	state := stateOf(ow, action)
	if state == "" {
		return allowed, src
	}
	return state == StateAllow, ruleSource(ow, state)
}

// applyRoleRules merges the rules for the roles in roleIDs: Allow if any allows, else Deny if any denies, the first in
// the entity's target order being the source.
func applyRoleRules(allowed bool, src Source, rules []*Overwrite, roleIDs []string, action permissions.Action) (bool, Source) {
	var denied *Overwrite
	for _, ow := range rules {
		if !slices.Contains(roleIDs, ow.RoleID) {
			continue
		}
		state := stateOf(ow, action)
		if state == StateAllow {
			return true, ruleSource(ow, state)
		}
		if state == StateDeny && denied == nil {
			denied = ow
		}
	}
	if denied == nil {
		return allowed, src
	}
	return false, ruleSource(denied, StateDeny)
}

// stateOf is what one rule sets action to, deny winning over allow inside it; "" is Fallback.
func stateOf(ow *Overwrite, action permissions.Action) string {
	if ow.Deny.Has(action) {
		return StateDeny
	}
	if ow.Allow.Has(action) {
		return StateAllow
	}
	return ""
}

func ruleSource(ow *Overwrite, state string) Source {
	if ow.Target == TargetRole {
		return Source{Kind: SourceRule, EntityType: ow.ResourceType, EntityID: ow.ResourceID, TargetType: TargetRole, TargetID: ow.RoleID, State: state}
	}
	return Source{Kind: SourceRule, EntityType: ow.ResourceType, EntityID: ow.ResourceID, TargetType: TargetPerson, TargetID: ow.UserID, State: state}
}

// Can reports whether userID may perform action on docID; in a project hidden from a Restricted member the doc's
// own overwrite allows nothing.
func (s *Service) Can(ctx context.Context, userID, docID string, action permissions.Action) (bool, error) {
	return s.CanDocs(ctx, userID, "", []string{docID}, action)[docID], nil
}

// CanDocs answers Can for each of docIDs from one read of the rules reaching userID on them, with each project's layers
// read once; a list of docs would otherwise pay for them per row. projectID, when every doc is in it, saves reading
// where each doc lives; empty, that is read for all of them at once.
func (s *Service) CanDocs(ctx context.Context, userID, projectID string, docIDs []string, action permissions.Action) map[string]bool {
	out := make(map[string]bool, len(docIDs))
	if userID == "" || len(docIDs) == 0 {
		return out
	}
	ctx = s.memoized(ctx)
	scopes := s.docScopesIn(ctx, projectID, docIDs)
	var own map[string]*Rules
	for _, id := range docIDs {
		scope := scopes[id]
		out[id], _ = decide(s.layers(ctx, userID, scope.WorkspaceID, scope.ProjectID), action, func() *Rules {
			if own == nil {
				own = s.docRules(ctx, docIDs, userID)
			}
			return own[id]
		})
	}
	return out
}

// DocsWith is what a doc list filters by in SQL for action: the projects in which userID holds it on every doc without
// rules of its own, and the docs whose rules for userID or their roles turn that answer, allowed in a project outside
// the set or denied in one inside it, read in one statement bounded by the rules. A doc passes exactly when CanDocs
// would pass it (ADR 0140).
func (s *Service) DocsWith(ctx context.Context, userID string, action permissions.Action) (projectIDs, allowed, denied []string, err error) {
	allowed, denied = []string{}, []string{}
	ctx = s.memoized(ctx)
	_, projectIDs, err = s.ProjectsAnywhere(ctx, userID, action)
	if err != nil || userID == "" {
		return projectIDs, allowed, denied, err
	}
	own, err := s.repo.RulesOfType(ctx, resourceTypeDoc, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("doc rules of %s: %w", userID, err)
	}
	docIDs := slices.Sorted(maps.Keys(own))
	scopes := s.docScopes(ctx, docIDs)
	for _, id := range docIDs {
		scope := scopes[id]
		inSet := scope.ProjectID != "" && slices.Contains(projectIDs, scope.ProjectID)
		can, _ := decide(s.layers(ctx, userID, scope.WorkspaceID, scope.ProjectID), action, func() *Rules { return own[id] })
		if can && !inSet {
			allowed = append(allowed, id)
		}
		if !can && inSet {
			denied = append(denied, id)
		}
	}
	return projectIDs, allowed, denied, nil
}

// docRules is the rules reaching userID on each of docIDs that has any; a failed read counts as none, as a single
// check's does.
func (s *Service) docRules(ctx context.Context, docIDs []string, userID string) map[string]*Rules {
	own, err := s.rulesOn(ctx, resourceTypeDoc, docIDs, userID)
	if err != nil {
		return map[string]*Rules{}
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

// The two older writes `grant` still means beside the three states.
const (
	// stateRevoke is a doc's grant=false: the allow goes and a Deny stays.
	stateRevoke = "revoke"
	// stateLift is a play's grant=true: the Deny goes and an Allow stays.
	stateLift = "lift"
)

// GrantState is the state an older write's grant means on resourceType, for a caller that sends no state.
func GrantState(resourceType string, grant bool) string {
	if resourceType == resourceTypePlay && grant {
		return stateLift
	}
	if resourceType == resourceTypePlay {
		return StateDeny
	}
	if grant {
		return StateAllow
	}
	return stateRevoke
}

// SetGrants sets actions to state for every (docID, userID) pair in one bulk operation (the permissions panel).
func (s *Service) SetGrants(ctx context.Context, actorID string, docIDs, userIDs []string, actions []permissions.Action, state string) error {
	return s.setGrants(ctx, actorID, resourceTypeDoc, docIDs, userIDs, actions, state)
}

// SetPlayGrants sets plays:run to state for users on plays; it is the only action a play rule sets until See per play.
func (s *Service) SetPlayGrants(ctx context.Context, actorID string, playIDs, userIDs []string, actions []permissions.Action, state string) error {
	for _, a := range actions {
		if a != permissions.PlaysRun {
			return fmt.Errorf("%w: only plays:run may be set on a play", apperrs.ErrInvalid)
		}
	}
	return s.setGrants(ctx, actorID, resourceTypePlay, playIDs, userIDs, actions, state)
}

// ruleWrite is one person's rule on one resource as it will be stored.
type ruleWrite struct {
	resourceID, userID string
	allow, deny        permissions.Set
}

// setGrants checks every write before storing any: the actor manages each resource, and holds there each action the
// write allows or lifts a Deny on (ADR 0088).
func (s *Service) setGrants(ctx context.Context, actorID, resourceType string, resourceIDs, userIDs []string, actions []permissions.Action, state string) error {
	if actorID == "" {
		return fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	if len(resourceIDs) == 0 || len(userIDs) == 0 || len(actions) == 0 {
		return fmt.Errorf("%w: at least one resource, one user, and one action are required", apperrs.ErrInvalid)
	}
	if !slices.Contains([]string{StateAllow, StateFallback, StateDeny, stateRevoke, stateLift}, state) {
		return fmt.Errorf("%w: state %q is not allow, fallback, or deny", apperrs.ErrInvalid, state)
	}
	writes := make([]ruleWrite, 0, len(resourceIDs)*len(userIDs))
	for _, id := range resourceIDs {
		ok, err := s.canManage(ctx, actorID, resourceType, id)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: manage permission required on %s %s", apperrs.ErrForbidden, resourceType, id)
		}
		for _, userID := range userIDs {
			w, err := s.nextRule(ctx, actorID, resourceType, id, userID, actions, state)
			if err != nil {
				return err
			}
			writes = append(writes, w)
		}
	}
	for _, w := range writes {
		event := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicGrantChanged, Payload: GrantEvent{ResourceType: resourceType, ResourceID: w.resourceID, Target: TargetPerson, UserID: w.userID, ActorID: actorID}}
		if err := s.repo.Set(ctx, resourceType, w.resourceID, w.userID, w.allow, w.deny, event); err != nil {
			return fmt.Errorf("set overwrite %s/%s/%s: %w", resourceType, w.resourceID, w.userID, err)
		}
	}
	return nil
}

// nextRule is userID's rule on the resource with actions moved to state, refused when it allows or lifts a Deny on an
// action the actor doesn't hold there.
func (s *Service) nextRule(ctx context.Context, actorID, resourceType, resourceID, userID string, actions []permissions.Action, state string) (ruleWrite, error) {
	w := ruleWrite{resourceID: resourceID, userID: userID}
	existing, err := s.repo.Get(ctx, resourceType, resourceID, userID)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return w, fmt.Errorf("get overwrite %s/%s/%s: %w", resourceType, resourceID, userID, err)
	}
	if err == nil {
		w.allow, w.deny = existing.Allow, existing.Deny
	}
	for _, a := range actions {
		allow, deny := moveTo(w.allow, w.deny, a, state)
		gives := (allow.Has(a) && !w.allow.Has(a)) || (w.deny.Has(a) && !deny.Has(a))
		if gives && !s.holds(ctx, actorID, resourceType, resourceID, a) {
			return w, fmt.Errorf("%w: you can't allow, or lift a Deny on, %s: you don't hold it on this %s", apperrs.ErrForbidden, a, resourceType)
		}
		w.allow, w.deny = allow, deny
	}
	return w, nil
}

func moveTo(allow, deny permissions.Set, a permissions.Action, state string) (permissions.Set, permissions.Set) {
	switch state {
	case StateAllow:
		return allow.With(a), deny.Without(a)
	case StateDeny:
		return allow.Without(a), deny.With(a)
	case stateRevoke:
		return allow.Without(a), deny
	case stateLift:
		return allow, deny.Without(a)
	}
	return allow.Without(a), deny.Without(a)
}

// holds is whether actorID holds action on the doc or play, its own rule for them included.
func (s *Service) holds(ctx context.Context, actorID, resourceType, resourceID string, action permissions.Action) bool {
	if resourceType == resourceTypePlay {
		return s.HasPermission(ctx, actorID, s.resolvePlayWorkspace(ctx, resourceID), action, resourceTypePlay, resourceID)
	}
	can, _ := s.Can(ctx, actorID, resourceID, action)
	return can
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
