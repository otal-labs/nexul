package access

import (
	"context"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SetNames wires what a source sentence names; unwired, a sentence names ids.
func (s *Service) SetNames(n Names) {
	s.names = n
}

// The actions each entity's rules may set: a doc's docs actions and its access, a play's See, Run, Edit and Delete, a stack's
// stacks then deploys actions, and on a project In this project and every project-area action.
var (
	docActions     = catalogActions(func(i permissions.Info) bool { return i.Domain == "docs" || i.Value == permissions.PermissionsWrite })
	playActions    = []permissions.Action{permissions.PlaysRead, permissions.PlaysRun, permissions.PlaysWrite, permissions.PlaysDelete}
	stackActions   = append(catalogActions(func(i permissions.Info) bool { return i.Domain == "stacks" }), catalogActions(func(i permissions.Info) bool { return i.Domain == "deploys" })...)
	projectActions = append([]permissions.Action{permissions.ProjectsOpen}, catalogActions(func(i permissions.Info) bool { return i.Area == permissions.AreaProject })...)
)

func catalogActions(keep func(permissions.Info) bool) []permissions.Action {
	var out []permissions.Action
	for _, info := range permissions.Catalog() {
		if keep(info) {
			out = append(out, info.Value)
		}
	}
	return out
}

// entity is what an explanation is about: where it lives, what opens and manages it, and what its rules may set.
type entity struct {
	kind, id, workspaceID, projectID, folderID string
	open, manage                               permissions.Action
	actions                                    []permissions.Action
}

func (s *Service) entityOf(ctx context.Context, resourceType, resourceID string) (entity, error) {
	e := entity{kind: resourceType, id: resourceID}
	switch resourceType {
	case resourceTypeDoc:
		scope := s.docScopes(ctx, []string{resourceID})[resourceID]
		e.workspaceID, e.projectID, e.folderID = scope.WorkspaceID, scope.ProjectID, scope.FolderID
		e.open, e.manage, e.actions = permissions.DocsRead, permissions.PermissionsWrite, docActions
		return e, nil
	case resourceTypeFolder:
		scope := s.folderScope(ctx, resourceID)
		if scope.ProjectID == "" {
			return e, fmt.Errorf("%w: folder %s", apperrs.ErrNotFound, resourceID)
		}
		// A folder opens to whoever is in its project; its rules decide only the docs in it.
		e.workspaceID, e.projectID = scope.WorkspaceID, scope.ProjectID
		e.open, e.manage, e.actions = permissions.Member, permissions.PermissionsWrite, docActions
		return e, nil
	case resourceTypeStack:
		return s.stackEntity(ctx, e)
	case resourceTypePlay:
		e.workspaceID = s.resolvePlayWorkspace(ctx, resourceID)
		e.open, e.manage, e.actions = permissions.PlaysRead, permissions.PlaysWrite, playActions
		return e, nil
	case resourceTypeProject:
		workspaceID, err := s.projectWorkspace(ctx, resourceID)
		if err != nil {
			return e, err
		}
		e.workspaceID, e.projectID = workspaceID, resourceID
		e.open, e.manage, e.actions = permissions.Member, permissions.ProjectsAccess, projectActions
		return e, nil
	}
	return e, fmt.Errorf("%w: resource_type %q is not doc, folder, play, project, or stack", apperrs.ErrInvalid, resourceType)
}

// stackEntity is a project's stack; an instance stack answers to its instance area and a branch deployment to its base
// stack's rules, so neither carries rules of its own (ADR 0079).
func (s *Service) stackEntity(ctx context.Context, e entity) (entity, error) {
	scope, err := s.stackScopeOf(ctx, e.id)
	if err != nil {
		return e, err
	}
	if scope.ProjectID == "" {
		return e, fmt.Errorf("%w: stack %s belongs to no project; the instance's own stacks carry no rules", apperrs.ErrInvalid, e.id)
	}
	if scope.BaseID != "" {
		return e, fmt.Errorf("%w: stack %s is a branch deployment; set the rules on its base stack %s", apperrs.ErrInvalid, e.id, scope.BaseID)
	}
	e.workspaceID, e.projectID = scope.WorkspaceID, scope.ProjectID
	e.open, e.manage, e.actions = permissions.StacksRead, permissions.StacksWrite, stackActions
	return e, nil
}

// answer is userID's answer for action on e, from the check that enforces it; with own false, e's rule naming them
// holds action at Fallback, the rest of it and its rules for their roles still applying.
func (s *Service) answer(ctx context.Context, userID string, e entity, action permissions.Action, own bool) (bool, Source) {
	if own {
		return s.check(ctx, userID, e.workspaceID, e.projectID, action, e.kind, e.id)
	}
	ws := s.layers(ctx, userID, e.workspaceID, e.projectID)
	if e.kind == resourceTypeProject {
		return decide(ws.in(e.projectID, personAtFallback(ws.project, action)), action, noRules)
	}
	return decide(ws, action, func() []*Rules {
		chain := s.chain(ctx, e.kind, e.id, userID)
		if len(chain) > 0 {
			chain[len(chain)-1] = personAtFallback(chain[len(chain)-1], action)
		}
		return chain
	})
}

// personAtFallback is r with the rule naming the person saying nothing about action.
func personAtFallback(r *Rules, action permissions.Action) *Rules {
	if r == nil || r.Person == nil {
		return r
	}
	return &Rules{Person: withoutAction(r.Person, action), Roles: r.Roles}
}

// withoutAction is ow holding action at Fallback.
func withoutAction(ow *Overwrite, action permissions.Action) *Overwrite {
	next := *ow
	next.Allow, next.Deny = ow.Allow.Without(action), ow.Deny.Without(action)
	return &next
}

func noRules() []*Rules { return nil }

// Decide is userID's answer for action on a doc, folder, play, project, or stack, with what decided it: the evaluator every check
// uses, so an explanation can never disagree with enforcement.
func (s *Service) Decide(ctx context.Context, userID, resourceType, resourceID string, action permissions.Action) (bool, Source, error) {
	e, err := s.entityOf(ctx, resourceType, resourceID)
	if err != nil {
		return false, Source{}, err
	}
	held, src := s.answer(ctx, userID, e, action, true)
	return held, src, nil
}

// Explain answers each action a doc's, folder's, play's, project's, stack's, or instance area's rules may set for userID, the
// caller when empty, with its source as a sentence. Explaining yourself takes opening the entity, anyone else its
// manage permission; an entity the caller can't open is not found.
func (s *Service) Explain(ctx context.Context, actorID, resourceType, resourceID, userID string) (*Explanation, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	if userID == "" {
		userID = actorID
	}
	if resourceType == resourceTypeComputer {
		return s.explainComputer(ctx, actorID, resourceID, userID)
	}
	ctx = s.memoized(ctx)
	if resourceType == resourceTypeInstanceArea {
		return s.explainArea(ctx, actorID, resourceID, userID)
	}
	e, err := s.entityOf(ctx, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if opens, _ := s.answer(ctx, actorID, e, e.open, true); !opens {
		return nil, fmt.Errorf("%w: %s %s", apperrs.ErrNotFound, resourceType, resourceID)
	}
	if userID != actorID {
		if manages, src := s.answer(ctx, actorID, e, e.manage, true); !manages {
			return nil, s.refusal(ctx, e.manage, src)
		}
	}
	n := s.namer(ctx)
	out := &Explanation{ResourceType: resourceType, ResourceID: resourceID, UserID: userID, Effective: make([]Answer, 0, len(e.actions))}
	for _, a := range e.actions {
		held, src := s.answer(ctx, userID, e, a, true)
		inherited, from := s.answer(ctx, userID, e, a, false)
		out.Effective = append(out.Effective, Answer{Action: a, Verdict: Verdict{held, n.named(src)}, Fallback: Verdict{inherited, n.named(from)}})
	}
	return out, nil
}

// ExplainRole answers each action a doc's, folder's, play's, project's, or stack's rules may set for a holder of roleID alone, with the
// role's own rule there, and with that rule holding the action at Fallback: what the panel shows under a role's
// Fallback. It takes managing the
// entity, and a role of another workspace is not found.
func (s *Service) ExplainRole(ctx context.Context, actorID, resourceType, resourceID, roleID string) (*Explanation, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	ctx = s.memoized(ctx)
	if resourceType == resourceTypeInstanceArea {
		return s.explainRoleArea(ctx, actorID, resourceID, roleID)
	}
	e, err := s.entityOf(ctx, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if opens, _ := s.answer(ctx, actorID, e, e.open, true); !opens {
		return nil, fmt.Errorf("%w: %s %s", apperrs.ErrNotFound, resourceType, resourceID)
	}
	if manages, src := s.answer(ctx, actorID, e, e.manage, true); !manages {
		return nil, s.refusal(ctx, e.manage, src)
	}
	if err := s.requireRolesOf(ctx, resourceType, resourceID, []string{roleID}); err != nil {
		return nil, err
	}
	held, err := s.roleWorkspace.HeldRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("read role %s: %w", roleID, err)
	}
	own, err := s.repo.RoleRules(ctx, e.kind, []string{e.id}, []string{roleID})
	if err != nil {
		return nil, fmt.Errorf("rules for role %s on %s %s: %w", roleID, e.kind, e.id, err)
	}
	ws := workspaceLayers{workspaceID: e.workspaceID, member: true, owner: held.IsOwner, roles: RolesHeld(held)}
	var inProject, inFolder []*Overwrite
	if e.projectID != "" && e.kind != resourceTypeProject {
		if inProject, err = s.roleRulesOn(ctx, resourceTypeProject, e.projectID, roleID); err != nil {
			return nil, err
		}
	}
	if e.folderID != "" {
		if inFolder, err = s.roleRulesOn(ctx, resourceTypeFolder, e.folderID, roleID); err != nil {
			return nil, err
		}
	}
	n := s.namer(ctx)
	out := &Explanation{ResourceType: resourceType, ResourceID: resourceID, RoleID: roleID, Effective: make([]Answer, 0, len(e.actions))}
	for _, a := range e.actions {
		with, src := roleAnswer(ws, e, roleID, a, inProject, inFolder, own[e.id])
		without, from := roleAnswer(ws, e, roleID, a, inProject, inFolder, atFallback(own[e.id], a))
		out.Effective = append(out.Effective, Answer{Action: a, Verdict: Verdict{with, n.named(src)}, Fallback: Verdict{without, n.named(from)}})
	}
	return out, nil
}

// roleRulesOn is roleID's rules on one entity.
func (s *Service) roleRulesOn(ctx context.Context, resourceType, resourceID, roleID string) ([]*Overwrite, error) {
	got, err := s.repo.RoleRules(ctx, resourceType, []string{resourceID}, []string{roleID})
	if err != nil {
		return nil, fmt.Errorf("rules for role %s on %s %s: %w", roleID, resourceType, resourceID, err)
	}
	return got[resourceID], nil
}

// atFallback is rules each holding action at Fallback.
func atFallback(rules []*Overwrite, action permissions.Action) []*Overwrite {
	out := make([]*Overwrite, len(rules))
	for i, ow := range rules {
		out[i] = withoutAction(ow, action)
	}
	return out
}

// roleAnswer is a holder of roleID alone asking for action on e, given its rules on e's project, a doc's folder and e.
func roleAnswer(ws workspaceLayers, e entity, roleID string, action permissions.Action, inProject, inFolder, own []*Overwrite) (bool, Source) {
	chain := []*Rules{{Roles: inFolder}, {Roles: own}}
	if e.kind == resourceTypeProject {
		inProject, chain = own, nil
	}
	if e.projectID != "" {
		ws = ws.in(e.projectID, &Rules{Roles: inProject})
	}
	allowed, src := decide(ws, action, func() []*Rules { return chain })
	if src.Kind == SourceNotInProject {
		src.TargetType, src.TargetID = TargetRole, roleID
	}
	return allowed, src
}

// RefuseDoc is the forbidden answer to userID's action on docID, naming the rule or role that refused it.
func (s *Service) RefuseDoc(ctx context.Context, userID, docID string, action permissions.Action) error {
	// A doc always resolves to an entity; one that doesn't exist answers from no layers at all.
	e, _ := s.entityOf(ctx, resourceTypeDoc, docID)
	_, src := s.answer(ctx, userID, e, action, true)
	return s.refusal(ctx, action, src)
}

func (s *Service) refusal(ctx context.Context, action permissions.Action, src Source) error {
	return refusal{ctx: ctx, s: s, action: action, src: src}
}

// refusal is a forbidden answer naming its source; the names are read only when the message is, so a list that drops
// refused rows reads none.
type refusal struct {
	ctx    context.Context
	s      *Service
	action permissions.Action
	src    Source
}

func (r refusal) Error() string {
	return fmt.Sprintf("%v: %s required (%s)", apperrs.ErrForbidden, r.action, r.s.namer(r.ctx).named(r.src).Sentence)
}

func (r refusal) Unwrap() error { return apperrs.ErrForbidden }

// namer resolves what sources name, each name read once.
type namer struct {
	ctx   context.Context
	names Names
	areas []permissions.InstanceArea
	seen  map[[2]string]string
}

func (s *Service) namer(ctx context.Context) *namer {
	return &namer{ctx: ctx, names: s.names, areas: s.InstanceAreas(), seen: map[[2]string]string{}}
}

// name is entityType id's name, or the id itself when it has none or the read fails.
func (n *namer) name(entityType, id string) string {
	key := [2]string{entityType, id}
	if got, ok := n.seen[key]; ok {
		return got
	}
	if entityType == resourceTypeInstanceArea {
		n.seen[key] = n.areaLabel(id)
		return n.seen[key]
	}
	name := id
	if n.names != nil && id != "" {
		if got, err := n.names.Name(n.ctx, entityType, id); err == nil && got != "" {
			name = got
		}
	}
	n.seen[key] = name
	return name
}

// areaLabel is the instance area keyed key's name in a sentence, or the key itself for one no longer wired.
func (n *namer) areaLabel(key string) string {
	i := slices.IndexFunc(n.areas, func(a permissions.InstanceArea) bool { return a.Key == key })
	if i < 0 {
		return key
	}
	return n.areas[i].Label
}

func (n *namer) named(src Source) NamedSource {
	r := NamedSource{Source: src, EntityName: n.name(src.EntityType, src.EntityID), TargetName: n.name(src.TargetType, src.TargetID)}
	r.Sentence = sentence(r)
	return r
}

// sentence is the line a person reads for r (ADR 0148, the source sentence).
func sentence(r NamedSource) string {
	allowed := r.State == StateAllow
	switch r.Kind {
	case SourceOwner:
		return "Owner of " + r.EntityName
	case SourceMember:
		return memberSentence(r, allowed)
	case SourceRole:
		return roleSentence(r, allowed)
	case SourceRule:
		return ruleSentence(r, allowed)
	case SourceOutsideProject:
		return "Not allowed: it counts only inside a project"
	case SourceNotInProject:
		if r.TargetType == TargetRole {
			return "Not listed on project " + r.EntityName
		}
		return "Not found: " + r.TargetName + " isn't in project " + r.EntityName
	}
	return ""
}

func roleSentence(r NamedSource, allowed bool) string {
	if r.TargetType == TargetPerson {
		return "Not allowed: " + r.TargetName + " holds no role"
	}
	if allowed {
		return "Allowed by role " + r.TargetName
	}
	return "Not allowed: role " + r.TargetName + " doesn't grant it"
}

func memberSentence(r NamedSource, allowed bool) string {
	if allowed {
		return "Member of " + r.EntityName
	}
	return "Not found: " + r.TargetName + " isn't a member of " + r.EntityName
}

func ruleSentence(r NamedSource, allowed bool) string {
	verb, target := "Denied", r.TargetName
	if allowed {
		verb = "Allowed"
	}
	if r.TargetType == TargetRole {
		target = "role " + target
	}
	if r.EntityType == resourceTypeInstanceArea {
		return fmt.Sprintf("%s on %s for %s", verb, r.EntityName, target)
	}
	return fmt.Sprintf("%s on %s %s for %s", verb, r.EntityType, r.EntityName, target)
}
