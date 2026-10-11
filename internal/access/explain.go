package access

import (
	"context"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SetNames wires what a source sentence names; unwired, a sentence names ids.
func (s *Service) SetNames(n Names) {
	s.names = n
}

// The actions each entity's rules may set today: a doc's sharing, a play's exclusion, a project's Project access.
var (
	docActions     = catalogActions(func(i permissions.Info) bool { return i.Domain == "docs" || i.Value == permissions.PermissionsWrite })
	playActions    = []permissions.Action{permissions.PlaysRun}
	projectActions = catalogActions(func(i permissions.Info) bool { return i.Area == permissions.AreaProject })
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
	kind, id, workspaceID, projectID string
	open, manage                     permissions.Action
	actions                          []permissions.Action
}

func (s *Service) entityOf(ctx context.Context, resourceType, resourceID string) (entity, error) {
	e := entity{kind: resourceType, id: resourceID}
	switch resourceType {
	case resourceTypeDoc:
		scope := s.docScopes(ctx, []string{resourceID})[resourceID]
		e.workspaceID, e.projectID = scope.WorkspaceID, scope.ProjectID
		e.open, e.manage, e.actions = permissions.DocsRead, permissions.PermissionsWrite, docActions
		return e, nil
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
		e.open, e.manage, e.actions = permissions.Member, permissions.MembersWrite, projectActions
		return e, nil
	}
	return e, fmt.Errorf("%w: resource_type %q is not doc, play, or project", apperrs.ErrInvalid, resourceType)
}

// answer is userID's answer for action on e, from the check that enforces it; with own false, e's own rule for them
// counts as Fallback.
func (s *Service) answer(ctx context.Context, userID string, e entity, action permissions.Action, own bool) (bool, Source) {
	if e.kind == resourceTypeProject && !own {
		ws := s.layers(ctx, userID, e.workspaceID, e.projectID)
		ws.project = nil
		return decide(ws, action, noOverwrite)
	}
	resourceType := e.kind
	if !own {
		resourceType = ""
	}
	return s.check(ctx, userID, e.workspaceID, e.projectID, action, resourceType, e.id)
}

func noOverwrite() *Overwrite { return nil }

// Decide is userID's answer for action on a doc, play, or project, with what decided it: the evaluator every check
// uses, so an explanation can never disagree with enforcement.
func (s *Service) Decide(ctx context.Context, userID, resourceType, resourceID string, action permissions.Action) (bool, Source, error) {
	e, err := s.entityOf(ctx, resourceType, resourceID)
	if err != nil {
		return false, Source{}, err
	}
	held, src := s.answer(ctx, userID, e, action, true)
	return held, src, nil
}

// Explain answers each action a doc's, play's, or project's rules may set for userID, the caller when empty, with
// its source as a sentence. Explaining yourself takes opening the entity, anyone else its manage permission; an
// entity the caller can't open is not found.
func (s *Service) Explain(ctx context.Context, actorID, resourceType, resourceID, userID string) (*Explanation, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	if userID == "" {
		userID = actorID
	}
	ctx = s.memoized(ctx)
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
	seen  map[[2]string]string
}

func (s *Service) namer(ctx context.Context) *namer {
	return &namer{ctx: ctx, names: s.names, seen: map[[2]string]string{}}
}

// name is entityType id's name, or the id itself when it has none or the read fails.
func (n *namer) name(entityType, id string) string {
	key := [2]string{entityType, id}
	if got, ok := n.seen[key]; ok {
		return got
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
		if allowed {
			return "Allowed by role " + r.TargetName
		}
		return "Not allowed: role " + r.TargetName + " doesn't grant it"
	case SourceRule:
		return ruleSentence(r, allowed)
	case SourceRestricted:
		return restrictedSentence(r, allowed)
	case SourceNotInProject:
		return "Not found: " + r.TargetName + " isn't in project " + r.EntityName
	}
	return ""
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
	return fmt.Sprintf("%s on %s %s for %s", verb, r.EntityType, r.EntityName, target)
}

func restrictedSentence(r NamedSource, allowed bool) string {
	if r.EntityType == resourceTypeWorkspace {
		return "Not allowed: " + r.TargetName + " is a Restricted member of " + r.EntityName
	}
	if allowed {
		return "Allowed by " + r.TargetName + "'s Project access on project " + r.EntityName
	}
	return "Not allowed: " + r.TargetName + "'s Project access on project " + r.EntityName + " doesn't grant it"
}
