// Package access implements permission overwrites, the HasPermission check, and doc-permission use-cases.
package access

import (
	"context"
	"errors"
	"fmt"
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

// resourceTypePlay keys a play's rules, which set its See, Run, Edit and Delete, by resource_id = the play id (ADR 0148).
const resourceTypePlay = "play"

// resourceTypeWorkspace names the workspace a source comes from; no rule sits on a workspace itself (ADR 0148).
const resourceTypeWorkspace = "workspace"

// resourceTypeProject keys a project's rules by resource_id = the project id (ADR 0148).
const resourceTypeProject = "project"

// resourceTypeChannel keys a channel's rules by resource_id = the conversation id (ADR 0148).
const resourceTypeChannel = "channel"

// resourceTypeFolder keys a doc folder's rules, which reach every doc in it, by resource_id = the folder id (ADR 0148).
const resourceTypeFolder = "folder"

// Service is the access use-case layer (ADR 0019): permission overwrites and the HasPermission/Can checks.
type Service struct {
	repo          Repo
	users         Users
	roles         RoleResolver
	roleWorkspace RoleWorkspaceResolver
	docWorkspace  DocWorkspaceResolver
	playWorkspace PlayWorkspaceResolver
	channels      ChannelWorkspaceResolver
	scopes        Scopes
	names         Names
	// ownerWorkspace is the default workspace, whose Owner is the instance Owner; connectorAreas are the connectors'.
	ownerWorkspace string
	connectorAreas []permissions.InstanceArea
	computers      Computers
	commits        *wake.Broadcast
	now            func() time.Time
}

// NewService wires the access use-cases over the given repo and user store.
func NewService(repo Repo, users Users) *Service {
	return &Service{repo: repo, users: users, now: time.Now}
}

// SetRoles wires the roles/tenancy domains' RoleResolver after both exist (see RoleResolver's doc comment).
func (s *Service) SetRoles(r RoleResolver) {
	s.roles = r
}

// SetRoleWorkspaces wires the roles domain's RoleWorkspaceResolver, which a rule for a role is checked against.
func (s *Service) SetRoleWorkspaces(r RoleWorkspaceResolver) {
	s.roleWorkspace = r
}

// SetDocWorkspaces wires the docs+workspace domains' DocWorkspaceResolver.
func (s *Service) SetDocWorkspaces(r DocWorkspaceResolver) {
	s.docWorkspace = r
}

// SetPlayWorkspaces wires the plays domain's PlayWorkspaceResolver.
func (s *Service) SetPlayWorkspaces(r PlayWorkspaceResolver) {
	s.playWorkspace = r
}

// SetChannelWorkspaces wires the chat domain's ChannelWorkspaceResolver, which a channel's rules are written through.
func (s *Service) SetChannelWorkspaces(r ChannelWorkspaceResolver) {
	s.channels = r
}

// HasPermission checks the Owner bypass, then for anything in a project whether userID is in it, then the role set,
// the project's rules and those of each entity inward to the resource, in order (ADR 0148).
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
	case resourceTypeFolder:
		projectID = s.folderScope(ctx, resourceID).ProjectID
	}
	held, _ := s.check(ctx, userID, workspaceID, projectID, action, resourceType, resourceID)
	return held
}

func (s *Service) check(ctx context.Context, userID, workspaceID, projectID string, action permissions.Action, resourceType, resourceID string) (bool, Source) {
	return decide(s.layers(ctx, userID, workspaceID, projectID), action, func() []*Rules {
		return s.chain(ctx, resourceType, resourceID, userID)
	})
}

// chain is the rules reaching userID on each entity from the project inward: a doc's folder then the doc, or the resource.
func (s *Service) chain(ctx context.Context, resourceType, resourceID, userID string) []*Rules {
	if resourceType == "" || resourceType == resourceTypeProject || resourceID == "" {
		return nil
	}
	if resourceType == resourceTypeDoc {
		return s.docRules(ctx, []string{resourceID}, userID)[resourceID].chain()
	}
	return []*Rules{s.rules(ctx, resourceType, resourceID, userID)}
}

// decide answers from layers already read, then from chain, read only if they leave it open, the nearest rule winning.
func decide(ws workspaceLayers, action permissions.Action, chain func() []*Rules) (bool, Source) {
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
	for _, r := range chain() {
		allowed, src = applyRules(allowed, src, r, ws.roles.roleIDs(), action)
	}
	return allowed, src
}

// workspaceLayers is the role set and the rules of the project asked about reaching the person, fetched once so a whole-grid answer costs the same lookups as a single check.
type workspaceLayers struct {
	userID      string
	workspaceID string
	projectID   string
	roles       RoleInfo
	member      bool
	owner       bool
	project     *Rules
	// listed is whether the project's rules let the person in: Allow of In this project, for a role or for them.
	listed bool
}

// layers reads userID's roles in workspaceID and projectID's rules for them and their roles, built
// once per request so a list's per-row checks cost one memo lookup.
func (s *Service) layers(ctx context.Context, userID, workspaceID, projectID string) workspaceLayers {
	// A failed read counts as no rule, as a check's always has, and the memo keeps nothing built on one.
	ws, _ := remember(ctx, layersKey{userID, workspaceID, projectID}, func() (workspaceLayers, error) {
		return s.readLayers(ctx, userID, workspaceID, projectID)
	})
	return ws
}

func (s *Service) readLayers(ctx context.Context, userID, workspaceID, projectID string) (workspaceLayers, error) {
	ws := workspaceLayers{userID: userID, workspaceID: workspaceID, projectID: projectID}
	if workspaceID == "" {
		return ws, nil
	}
	var failed error
	if s.roles != nil {
		info, err := s.memberRole(ctx, workspaceID, userID)
		if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			failed = err
		}
		if err == nil {
			ws.member = true
			ws.owner = info.IsOwnerRole
			ws.roles = info
		}
	}
	var projectIDs []string
	if projectID != "" {
		projectIDs = []string{projectID}
	}
	projects, err := s.layerRules(ctx, userID, workspaceID, ws.roles.roleIDs(), projectIDs)
	if err != nil {
		failed = err
	}
	return ws.in(projectID, projects[projectID]), failed
}

// in is ws inside projectID, whose rules reaching the person are r: the person is in it when they end in Allow of In
// this project, for a role they hold or for them by name (ADR 0148).
func (ws workspaceLayers) in(projectID string, r *Rules) workspaceLayers {
	ws.projectID, ws.project = projectID, r
	ws.listed, _ = applyRules(false, Source{}, r, ws.roles.roleIDs(), permissions.ProjectsOpen)
	return ws
}

// hidden is a project someone other than the Owner is not in, a non-member never being: it, and everything in it, reads
// as not found, a stale rule naming them inside it included.
func (ws workspaceLayers) hidden() bool {
	return !ws.owner && ws.projectID != "" && (!ws.member || !ws.listed)
}

func sourceOf(kind, entityType, entityID, targetType, targetID string, allowed bool) Source {
	state := StateDeny
	if allowed {
		state = StateAllow
	}
	return Source{Kind: kind, EntityType: entityType, EntityID: entityID, TargetType: targetType, TargetID: targetID, State: state}
}

// has is action from the roles and, inside a project, the project's rules; a project-area action with no project
// named is not held, since it counts only inside a project the person is in.
func (ws workspaceLayers) has(action permissions.Action) (bool, Source) {
	if ws.member && ws.projectID == "" && permissions.AreaOf(action) == permissions.AreaProject {
		return false, sourceOf(SourceOutsideProject, resourceTypeWorkspace, ws.workspaceID, TargetPerson, ws.userID, false)
	}
	allowed, src := ws.granted(action)
	if ws.projectID == "" {
		return allowed, src
	}
	return applyRules(allowed, src, ws.project, ws.roles.roleIDs(), action)
}

// granted is action from the roles held alone, whatever the project.
func (ws workspaceLayers) granted(action permissions.Action) (bool, Source) {
	allowed := ws.roles.Permissions.Has(action)
	src := sourceOf(SourceRole, resourceTypeWorkspace, ws.workspaceID, TargetRole, ws.roles.sourceRole(action), allowed)
	if len(ws.roles.roleIDs()) == 0 {
		src = sourceOf(SourceRole, resourceTypeWorkspace, ws.workspaceID, TargetPerson, ws.userID, false)
	}
	if !ws.member {
		src = sourceOf(SourceMember, resourceTypeWorkspace, ws.workspaceID, TargetPerson, ws.userID, false)
	}
	return allowed, src
}

// actions is every grid action held: in the project asked about, or from the roles alone when byRole.
func (ws workspaceLayers) actions(byRole bool) []string {
	all := permissions.AllActions()
	out := make([]string, 0, len(all))
	for _, a := range all {
		held, _ := ws.has(a)
		if byRole {
			held, _ = ws.granted(a)
		}
		if ws.owner || held {
			out = append(out, string(a))
		}
	}
	return out
}

// WorkspacePermissions returns every grid action userID's roles grant in workspaceID: what
// they may give, what an instance area answers to, and what holds in each project they are in before its rules, though
// Require on the workspace itself holds no project-area action (feeds /api/workspaces/{id}/me).
func (s *Service) WorkspacePermissions(ctx context.Context, userID, workspaceID string) []string {
	if userID == "" {
		return []string{}
	}
	return s.layers(ctx, userID, workspaceID, "").actions(true)
}

// ProjectPermissions returns every grid action userID holds inside projectID; opens is false when the project is
// unknown, outside their workspaces, or they are not in it.
func (s *Service) ProjectPermissions(ctx context.Context, userID, projectID string) (actions []string, opens bool) {
	if userID == "" {
		return []string{}, false
	}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		return []string{}, false
	}
	ws := s.layers(ctx, userID, workspaceID, projectID)
	if !ws.member || ws.hidden() {
		return []string{}, false
	}
	return ws.actions(false), true
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

// Can reports whether userID may perform action on docID; in a project they are not in, the doc's own rules allow
// nothing.
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
	for workspaceID, projectIDs := range projectsByWorkspace(scopes) {
		// Read up front so the docs' projects cost one read per workspace, not one each; a failure reads again per check.
		_, _ = s.layerRules(ctx, userID, workspaceID, s.roleIDsIn(ctx, userID, workspaceID), projectIDs)
	}
	var own map[string]*DocRules
	for _, id := range docIDs {
		scope := scopes[id]
		out[id], _ = decide(s.layers(ctx, userID, scope.WorkspaceID, scope.ProjectID), action, func() []*Rules {
			if own == nil {
				own = s.docRules(ctx, docIDs, userID)
			}
			return own[id].chain()
		})
	}
	return out
}

// projectsByWorkspace is the projects scopes name, by workspace.
func projectsByWorkspace(scopes map[string]DocScope) map[string][]string {
	out := map[string][]string{}
	for _, scope := range scopes {
		if scope.WorkspaceID == "" || scope.ProjectID == "" || slices.Contains(out[scope.WorkspaceID], scope.ProjectID) {
			continue
		}
		out[scope.WorkspaceID] = append(out[scope.WorkspaceID], scope.ProjectID)
	}
	return out
}

// DocsWith is what a doc list filters by in SQL for action; a doc passes exactly when CanDocs would (ADR 0140, ADR 0148).
func (s *Service) DocsWith(ctx context.Context, userID, projectID string, action permissions.Action) (permissions.DocReach, error) {
	ctx = s.memoized(ctx)
	workspaceIDs, projectIDs, err := s.ProjectsAnywhere(ctx, userID, action)
	if err != nil {
		return permissions.DocReach{}, err
	}
	ruled := []string{}
	for _, workspaceID := range workspaceIDs {
		info, err := s.standing(ctx, userID, workspaceID)
		if err != nil {
			return permissions.DocReach{}, err
		}
		if info.IsOwnerRole {
			continue
		}
		in, err := s.ProjectsWith(ctx, userID, workspaceID, permissions.Member)
		if err != nil {
			return permissions.DocReach{}, err
		}
		ruled = append(ruled, in...)
	}
	if projectID != "" {
		ruled = slices.DeleteFunc(ruled, func(id string) bool { return id != projectID })
	}
	if len(ruled) == 0 {
		return permissions.DocReach{ProjectIDs: projectIDs, AllowedFolders: []string{}, DeniedFolders: []string{}, Allowed: []string{}, Denied: []string{}}, nil
	}
	reach, err := s.repo.DocRuleTurns(ctx, userID, action, ruled, projectIDs)
	if err != nil {
		return permissions.DocReach{}, fmt.Errorf("doc rules of %s: %w", userID, err)
	}
	reach.ProjectIDs = projectIDs
	return reach, nil
}

// docRules is the rules reaching userID on each of docIDs and its folder; a failed read counts as none.
func (s *Service) docRules(ctx context.Context, docIDs []string, userID string) map[string]*DocRules {
	own, err := rememberEach(ctx, docIDs, func(id string) any { return docRulesKey{id, userID} }, func(missing []string) (map[string]*DocRules, error) {
		return s.repo.DocRules(ctx, missing, userID)
	})
	if err != nil {
		return map[string]*DocRules{}
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

// folderScope is where folderID lives; empty when it can't be resolved.
func (s *Service) folderScope(ctx context.Context, folderID string) DocScope {
	if s.docWorkspace == nil {
		return DocScope{}
	}
	scope, _ := remember(ctx, folderKey{folderID}, func() (DocScope, error) {
		return s.docWorkspace.FolderScope(ctx, folderID)
	})
	return scope
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
	return s.SetRules(ctx, actorID, resourceTypeDoc, docIDs, Targets{UserIDs: userIDs}, actions, state)
}

// SetPlayGrants sets actions to state for users on plays.
func (s *Service) SetPlayGrants(ctx context.Context, actorID string, playIDs, userIDs []string, actions []permissions.Action, state string) error {
	return s.SetRules(ctx, actorID, resourceTypePlay, playIDs, Targets{UserIDs: userIDs}, actions, state)
}

// Targets are whom a write's rules name: people by account id, and roles by id.
type Targets struct {
	UserIDs []string
	RoleIDs []string
}

// ruleWrite is one rule on one resource as it will be stored, naming a person or a role; existed is whether it was.
type ruleWrite struct {
	resourceID, target, targetID string
	allow, deny                  permissions.Set
	existed                      bool
}

// SetRules sets actions to state on every resource for every target, checking every write before storing any: the
// actor manages each resource (permissions:write on a doc or folder, plays:write on a play, Manage this project's
// access on a project, the Owner on a channel), each action is one the resource's rules may set, each role is the
// resource's workspace's, and the actor holds there each action the write allows or lifts a Deny on (ADR 0088).
func (s *Service) SetRules(ctx context.Context, actorID, resourceType string, resourceIDs []string, to Targets, actions []permissions.Action, state string) error {
	if err := checkWrite(actorID, resourceType, resourceIDs, to, actions, state); err != nil {
		return err
	}
	if resourceType == resourceTypeComputer {
		if err := checkComputerWrite(to, state); err != nil {
			return err
		}
	}
	writes := make([]ruleWrite, 0, len(resourceIDs)*(len(to.UserIDs)+len(to.RoleIDs)))
	for _, id := range resourceIDs {
		planned, err := s.planRules(ctx, actorID, resourceType, id, to, actions, state)
		if err != nil {
			return err
		}
		writes = append(writes, planned...)
	}
	for _, w := range writes {
		event := GrantEvent{ResourceType: resourceType, ResourceID: w.resourceID, Target: w.target, ActorID: actorID}
		if w.target == TargetRole {
			event.RoleID = w.targetID
		}
		if w.target == TargetPerson {
			event.UserID = w.targetID
		}
		outbox := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicGrantChanged, Payload: event}
		if resourceType == resourceTypeComputer {
			outbox = computerGrantEvent(w.resourceID, actorID, w.targetID, actorID, w.allow)
		}
		if err := s.store(ctx, resourceType, w, outbox); err != nil {
			return fmt.Errorf("set overwrite %s/%s/%s: %w", resourceType, w.resourceID, w.targetID, err)
		}
	}
	return nil
}

// checkWrite refuses a write missing what it needs, or setting what the resource's rules may not.
func checkWrite(actorID, resourceType string, resourceIDs []string, to Targets, actions []permissions.Action, state string) error {
	if actorID == "" {
		return fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	if len(resourceIDs) == 0 || len(to.UserIDs)+len(to.RoleIDs) == 0 || len(actions) == 0 {
		return fmt.Errorf("%w: at least one resource, one person or role, and one action are required", apperrs.ErrInvalid)
	}
	if !slices.Contains([]string{StateAllow, StateFallback, StateDeny, stateRevoke, stateLift}, state) {
		return fmt.Errorf("%w: state %q is not allow, fallback, or deny", apperrs.ErrInvalid, state)
	}
	return checkRuleActions(resourceType, actions)
}

// planRules is each target's next rule on one resource, refused before anything is stored; a Fallback where no rule
// is changes nothing and plans no write.
func (s *Service) planRules(ctx context.Context, actorID, resourceType, resourceID string, to Targets, actions []permissions.Action, state string) ([]ruleWrite, error) {
	if err := s.requireOpens(ctx, actorID, resourceType, resourceID); err != nil {
		return nil, err
	}
	ok, err := s.canManage(ctx, actorID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%w: manage permission required on %s %s", apperrs.ErrForbidden, resourceType, resourceID)
	}
	if err := s.checkArea(resourceType, resourceID, actions); err != nil {
		return nil, err
	}
	if err := s.requireRolesOf(ctx, resourceType, resourceID, to.RoleIDs); err != nil {
		return nil, err
	}
	var out []ruleWrite
	for _, t := range to.each() {
		w, err := s.nextRule(ctx, actorID, resourceType, resourceID, t, actions, state)
		if err != nil {
			return nil, err
		}
		if resourceType == resourceTypeComputer {
			w.allow, w.deny = computerRule(w.allow, actions, state), nil
			if err := s.checkComputerTarget(ctx, actorID, w); err != nil {
				return nil, err
			}
		}
		if w.existed || len(w.allow)+len(w.deny) > 0 {
			out = append(out, w)
		}
	}
	return out, nil
}

// each is every target as a (target type, id) pair, people first.
func (t Targets) each() [][2]string {
	out := make([][2]string, 0, len(t.UserIDs)+len(t.RoleIDs))
	for _, id := range t.UserIDs {
		out = append(out, [2]string{TargetPerson, id})
	}
	for _, id := range t.RoleIDs {
		out = append(out, [2]string{TargetRole, id})
	}
	return out
}

// ruleTypes are the entities rules sit on.
var ruleTypes = []string{resourceTypeDoc, resourceTypeFolder, resourceTypePlay, resourceTypeProject, resourceTypeChannel, resourceTypeInstanceArea, resourceTypeComputer}

// errNotARuleType refuses a resource_type no rule sits on.
func errNotARuleType(resourceType string) error {
	return fmt.Errorf("%w: resource_type %q is not doc, folder, play, project, channel, computer, or instance_area", apperrs.ErrInvalid, resourceType)
}

// checkRuleActions refuses an action resourceType's rules may not set: a play's rules set See, Run, Edit and Delete, a project's In this
// project and the project-area actions, a folder's the docs actions and their access, a channel's its read, an
// instance area's its own actions (checkArea), a computer's its three permissions and nothing else.
func checkRuleActions(resourceType string, actions []permissions.Action) error {
	if !slices.Contains(ruleTypes, resourceType) {
		return errNotARuleType(resourceType)
	}
	for _, a := range actions {
		if slices.Contains(permissions.ComputerActions, a) != (resourceType == resourceTypeComputer) {
			return fmt.Errorf("%w: a computer's rules set computers:see, computers:run_agents and computers:run_commands, and only they", apperrs.ErrInvalid)
		}
		if a == permissions.ProjectsOpen && resourceType != resourceTypeProject {
			return fmt.Errorf("%w: projects:open is set only on a project", apperrs.ErrInvalid)
		}
		if resourceType == resourceTypePlay && !slices.Contains(playActions, a) {
			return fmt.Errorf("%w: %s is not set on a play; its rules set plays:read, plays:run, plays:write and plays:delete", apperrs.ErrInvalid, a)
		}
		if resourceType == resourceTypeProject && !slices.Contains(projectActions, a) {
			return fmt.Errorf("%w: %s is not set on a project; its rules set projects:open and project-area actions", apperrs.ErrInvalid, a)
		}
		if resourceType == resourceTypeFolder && !slices.Contains(docActions, a) {
			return fmt.Errorf("%w: %s is not set on a folder; its rules set the docs actions and permissions:write", apperrs.ErrInvalid, a)
		}
		if resourceType == resourceTypeChannel && a != permissions.ChannelsRead {
			return fmt.Errorf("%w: only channels:read may be set on a channel", apperrs.ErrInvalid)
		}
	}
	return nil
}

// requireRolesOf refuses a role from outside the workspace the resource lives in, whose rule would reach nobody there.
func (s *Service) requireRolesOf(ctx context.Context, resourceType, resourceID string, roleIDs []string) error {
	if len(roleIDs) == 0 {
		return nil
	}
	workspaceID := s.workspaceOf(ctx, resourceType, resourceID)
	for _, roleID := range roleIDs {
		if s.roleWorkspace == nil {
			return fmt.Errorf("%w: no role lookup wired", apperrs.ErrForbidden)
		}
		got, err := s.roleWorkspace.WorkspaceIDForRole(ctx, roleID)
		if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			return fmt.Errorf("resolve role %s: %w", roleID, err)
		}
		// An instance area belongs to no workspace, so a role of any workspace reaches its holders there.
		if err != nil || (got != workspaceID && resourceType != resourceTypeInstanceArea) {
			return fmt.Errorf("%w: role %s is not a role of this %s's workspace", apperrs.ErrNotFound, roleID, resourceType)
		}
	}
	return nil
}

// requireOpens answers a project, or a folder of one, the actor is not in, or a play they can't see, as not found.
func (s *Service) requireOpens(ctx context.Context, actorID, resourceType, resourceID string) error {
	if resourceType != resourceTypeProject && resourceType != resourceTypeFolder && resourceType != resourceTypePlay {
		return nil
	}
	e, err := s.entityOf(ctx, resourceType, resourceID)
	if err != nil {
		return err
	}
	if opens, _ := s.answer(ctx, actorID, e, e.open, true); !opens {
		return fmt.Errorf("%w: %s %s", apperrs.ErrNotFound, resourceType, resourceID)
	}
	return nil
}

// workspaceOf is the workspace a doc, folder, play, project, or channel lives in; "" when it can't be resolved.
func (s *Service) workspaceOf(ctx context.Context, resourceType, resourceID string) string {
	switch resourceType {
	case resourceTypeFolder:
		return s.folderScope(ctx, resourceID).WorkspaceID
	case resourceTypePlay:
		return s.resolvePlayWorkspace(ctx, resourceID)
	case resourceTypeChannel:
		if s.channels == nil {
			return ""
		}
		workspaceID, _ := s.channels.WorkspaceIDForChannel(ctx, resourceID)
		return workspaceID
	case resourceTypeProject:
		workspaceID, _ := s.projectWorkspace(ctx, resourceID)
		return workspaceID
	}
	workspaceID, _ := s.resolveDoc(ctx, resourceID)
	return workspaceID
}

func (s *Service) store(ctx context.Context, resourceType string, w ruleWrite, event eventbus.OutboxEvent) error {
	if w.target == TargetRole {
		return s.repo.SetRole(ctx, resourceType, w.resourceID, w.targetID, w.allow, w.deny, event)
	}
	return s.repo.Set(ctx, resourceType, w.resourceID, w.targetID, w.allow, w.deny, event)
}

// nextRule is the target's rule on the resource with actions moved to state, refused when it allows or lifts a Deny on
// an action the actor doesn't hold there.
func (s *Service) nextRule(ctx context.Context, actorID, resourceType, resourceID string, target [2]string, actions []permissions.Action, state string) (ruleWrite, error) {
	w := ruleWrite{resourceID: resourceID, target: target[0], targetID: target[1]}
	existing, err := s.existingRule(ctx, resourceType, resourceID, w.target, w.targetID)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return w, fmt.Errorf("get overwrite %s/%s/%s: %w", resourceType, resourceID, w.targetID, err)
	}
	if err == nil {
		w.allow, w.deny, w.existed = existing.Allow, existing.Deny, true
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

func (s *Service) existingRule(ctx context.Context, resourceType, resourceID, target, targetID string) (*Overwrite, error) {
	if target == TargetRole {
		return s.repo.GetRole(ctx, resourceType, resourceID, targetID)
	}
	return s.repo.Get(ctx, resourceType, resourceID, targetID)
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

// holds is whether actorID holds action on the doc, folder, play, project, instance area, or computer, its own rules included.
func (s *Service) holds(ctx context.Context, actorID, resourceType, resourceID string, action permissions.Action) bool {
	if resourceType == resourceTypeInstanceArea {
		held, _ := s.HoldsArea(ctx, actorID, resourceID, action)
		return held
	}
	if resourceType == resourceTypeComputer {
		_, held, err := s.ComputerPermissions(ctx, actorID, resourceID)
		return err == nil && held.Has(action)
	}
	if resourceType != resourceTypeDoc {
		return s.HasPermission(ctx, actorID, s.workspaceOf(ctx, resourceType, resourceID), action, resourceType, resourceID)
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

// ListRules is the rules on a doc, folder, play, project, channel, or instance area, for whoever manages it; a project
// the actor is not in, a folder of one, or a play they can't see is not found, so listing its rules never confirms it exists.
func (s *Service) ListRules(ctx context.Context, actorID, resourceType, resourceID string) ([]*Overwrite, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	if !slices.Contains(ruleTypes, resourceType) {
		return nil, errNotARuleType(resourceType)
	}
	if err := s.checkArea(resourceType, resourceID, nil); err != nil {
		return nil, err
	}
	if err := s.requireOpens(ctx, actorID, resourceType, resourceID); err != nil {
		return nil, err
	}
	return s.listGrants(ctx, actorID, resourceType, resourceID)
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

// ListUsers is gated to a holder of accounts:read or anyone holding permissions:write on at least one doc or folder.
func (s *Service) ListUsers(ctx context.Context, actorID string) ([]*User, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	readsAccounts, err := s.HoldsAnywhere(ctx, actorID, permissions.AccountsRead)
	if err != nil {
		return nil, err
	}
	if !readsAccounts {
		has, err := s.sharesAnyDoc(ctx, actorID)
		if err != nil {
			return nil, err
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

// sharesAnyDoc is whether a rule allows actorID permissions:write on some doc or folder.
func (s *Service) sharesAnyDoc(ctx context.Context, actorID string) (bool, error) {
	for _, resourceType := range []string{resourceTypeDoc, resourceTypeFolder} {
		has, err := s.repo.HasAllowAny(ctx, resourceType, actorID, permissions.PermissionsWrite)
		if err != nil {
			return false, fmt.Errorf("check permissions:write for %s: %w", actorID, err)
		}
		if has {
			return true, nil
		}
	}
	return false, nil
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

// ManagesRules reports whether userID may write a doc's, folder's, play's, project's, channel's, or instance area's rules.
func (s *Service) ManagesRules(ctx context.Context, userID, resourceType, resourceID string) bool {
	if userID == "" {
		return false
	}
	ok, err := s.canManage(ctx, userID, resourceType, resourceID)
	return err == nil && ok
}

// canManage grants no instance-wide bypass, since that would cross workspace isolation boundaries. A doc's or a
// folder's manage action is the cross-cutting permissions:write share grant; a play's is plays:write, the same bit
// that gates editing the play's own definition; a project's is Manage this project's access; a channel's is the
// Owner's alone until channels carry their own manage permission; an instance area's the instance Owner's alone; a
// computer's its owner's, and to anyone else it is not found.
func (s *Service) canManage(ctx context.Context, actorID, resourceType, resourceID string) (bool, error) {
	if resourceType == resourceTypeInstanceArea {
		return s.IsInstanceOwner(ctx, actorID)
	}
	if resourceType == resourceTypeComputer {
		if err := s.ownedComputer(ctx, actorID, resourceID); err != nil {
			return false, err
		}
		return true, nil
	}
	if resourceType == resourceTypeChannel {
		workspaceID := s.workspaceOf(ctx, resourceType, resourceID)
		if workspaceID == "" {
			return false, fmt.Errorf("%w: channel %s", apperrs.ErrNotFound, resourceID)
		}
		return s.IsOwner(ctx, actorID, workspaceID)
	}
	if resourceType == resourceTypePlay {
		return s.holds(ctx, actorID, resourceType, resourceID, permissions.PlaysWrite), nil
	}
	if resourceType == resourceTypeProject {
		return s.holds(ctx, actorID, resourceType, resourceID, permissions.ProjectsAccess), nil
	}
	if resourceType == resourceTypeFolder {
		return s.holds(ctx, actorID, resourceType, resourceID, permissions.PermissionsWrite), nil
	}
	return s.Can(ctx, actorID, resourceID, permissions.PermissionsWrite)
}
