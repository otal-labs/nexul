package access

import (
	"context"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// resourceTypeInstanceArea keys an instance area's rules by resource_id = the area's key (ADR 0148).
const resourceTypeInstanceArea = "instance_area"

// SetInstanceAreas wires the workspace whose Owner is the instance Owner, the one person who writes an instance area's
// rules, and the connectors' areas beside the fixed ones.
func (s *Service) SetInstanceAreas(ownerWorkspaceID string, connectors []permissions.InstanceArea) {
	s.ownerWorkspace, s.connectorAreas = ownerWorkspaceID, connectors
}

// InstanceAreas is every instance area a rule may sit on, the connectors' last.
func (s *Service) InstanceAreas() []permissions.InstanceArea {
	return append(permissions.InstanceAreas(), s.connectorAreas...)
}

func (s *Service) instanceArea(key string) (permissions.InstanceArea, bool) {
	i := slices.IndexFunc(s.InstanceAreas(), func(a permissions.InstanceArea) bool { return a.Key == key })
	if i < 0 {
		return permissions.InstanceArea{}, false
	}
	return s.InstanceAreas()[i], true
}

// IsInstanceOwner reports whether userID holds the Owner role of the default workspace (ADR 0088).
func (s *Service) IsInstanceOwner(ctx context.Context, userID string) (bool, error) {
	if userID == "" || s.ownerWorkspace == "" {
		return false, nil
	}
	return s.IsOwner(ctx, userID, s.ownerWorkspace)
}

// RequireArea checks the caller on ctx holds action on the instance area keyed area; a forbidden answer names its
// source. The server's own calls pass.
func (s *Service) RequireArea(ctx context.Context, area string, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return nil
	}
	held, src, err := s.decideArea(ctx, userID, area, action, true)
	if err != nil {
		return err
	}
	if !held {
		return s.refusal(ctx, action, src)
	}
	return nil
}

// HoldsArea reports whether userID holds action on the instance area keyed area: an Owner of any workspace always,
// otherwise the roles they hold in any workspace, then the area's rules for those roles, of which Allow wins, then the
// rule naming them (ADR 0148). An empty area answers from the roles alone.
func (s *Service) HoldsArea(ctx context.Context, userID, area string, action permissions.Action) (bool, error) {
	held, _, err := s.decideArea(ctx, userID, area, action, true)
	return held, err
}

// decideArea is HoldsArea with its source; with own false, the area's rule naming userID holds action at Fallback.
func (s *Service) decideArea(ctx context.Context, userID, area string, action permissions.Action, own bool) (bool, Source, error) {
	workspaceIDs, err := s.workspacesOf(ctx, userID)
	if err != nil {
		return false, Source{}, err
	}
	allowed, src := false, sourceOf(SourceRole, resourceTypeWorkspace, "", TargetPerson, userID, false)
	var roleIDs []string
	for _, workspaceID := range workspaceIDs {
		ws := s.layers(ctx, userID, workspaceID, "")
		if ws.owner {
			held, src := decide(ws, action, noRules)
			return held, src, nil
		}
		roleIDs = append(roleIDs, ws.roles.roleIDs()...)
		if allowed {
			continue
		}
		// The source is the first workspace whose roles grant it, else the last one where they hold a role.
		if held, from := ws.granted(action); held || len(ws.roles.roleIDs()) > 0 {
			allowed, src = held, from
		}
	}
	if area == "" {
		return allowed, src, nil
	}
	rules, err := remember(ctx, typeRulesKey{resourceTypeInstanceArea, userID}, func() (map[string]*Rules, error) {
		return s.repo.RulesOfType(ctx, resourceTypeInstanceArea, userID)
	})
	if err != nil {
		return false, Source{}, fmt.Errorf("instance area rules of %s: %w", userID, err)
	}
	r := rules[area]
	if !own {
		r = personAtFallback(r, action)
	}
	allowed, src = applyRules(allowed, src, r, roleIDs, action)
	return allowed, src, nil
}

// heldOnAnyArea is whether userID holds action on at least one area listing it, or from the roles alone when none does.
func (s *Service) heldOnAnyArea(ctx context.Context, userID string, action permissions.Action) (bool, error) {
	var areas []string
	for _, a := range s.InstanceAreas() {
		if slices.Contains(a.Actions, action) {
			areas = append(areas, a.Key)
		}
	}
	if len(areas) == 0 {
		areas = []string{""}
	}
	for _, area := range areas {
		held, err := s.HoldsArea(ctx, userID, area, action)
		if err != nil || held {
			return held, err
		}
	}
	return false, nil
}

// checkArea refuses an instance-area rule on an unknown area, or setting an action the area's rules may not.
func (s *Service) checkArea(resourceType, key string, actions []permissions.Action) error {
	if resourceType != resourceTypeInstanceArea {
		return nil
	}
	area, ok := s.instanceArea(key)
	if !ok {
		return fmt.Errorf("%w: instance area %q", apperrs.ErrNotFound, key)
	}
	for _, a := range actions {
		if !slices.Contains(area.Actions, a) {
			return fmt.Errorf("%w: %s is not set on %s; its rules set %s", apperrs.ErrInvalid, a, area.Label, permissions.SetOf(area.Actions...))
		}
	}
	return nil
}

// explainArea answers each action an instance area's rules may set for userID; anyone explains their own, and only the
// instance Owner someone else's.
func (s *Service) explainArea(ctx context.Context, actorID, key, userID string) (*Explanation, error) {
	area, ok := s.instanceArea(key)
	if !ok {
		return nil, fmt.Errorf("%w: instance area %q", apperrs.ErrNotFound, key)
	}
	if userID != actorID {
		if err := s.requireInstanceOwner(ctx, actorID); err != nil {
			return nil, err
		}
	}
	n := s.namer(ctx)
	out := &Explanation{ResourceType: resourceTypeInstanceArea, ResourceID: key, UserID: userID, Effective: make([]Answer, 0, len(area.Actions))}
	for _, a := range area.Actions {
		held, src, err := s.decideArea(ctx, userID, key, a, true)
		if err != nil {
			return nil, err
		}
		inherited, from, err := s.decideArea(ctx, userID, key, a, false)
		if err != nil {
			return nil, err
		}
		out.Effective = append(out.Effective, Answer{Action: a, Verdict: Verdict{held, n.named(src)}, Fallback: Verdict{inherited, n.named(from)}})
	}
	return out, nil
}

// requireInstanceOwner refuses anyone but the instance Owner, who alone manages an instance area's access.
func (s *Service) requireInstanceOwner(ctx context.Context, actorID string) error {
	owner, err := s.IsInstanceOwner(ctx, actorID)
	if err != nil {
		return err
	}
	if !owner {
		return fmt.Errorf("%w: only the instance Owner manages an instance area's access", apperrs.ErrForbidden)
	}
	return nil
}

// explainRoleArea answers each action an instance area's rules may set for a holder of roleID alone, with the role's
// rule there and with it at Fallback; only the instance Owner asks.
func (s *Service) explainRoleArea(ctx context.Context, actorID, key, roleID string) (*Explanation, error) {
	area, ok := s.instanceArea(key)
	if !ok {
		return nil, fmt.Errorf("%w: instance area %q", apperrs.ErrNotFound, key)
	}
	if err := s.requireInstanceOwner(ctx, actorID); err != nil {
		return nil, err
	}
	if s.roleWorkspace == nil {
		return nil, fmt.Errorf("%w: no role lookup wired", apperrs.ErrForbidden)
	}
	workspaceID, err := s.roleWorkspace.WorkspaceIDForRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("resolve role %s: %w", roleID, err)
	}
	held, err := s.roleWorkspace.HeldRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("read role %s: %w", roleID, err)
	}
	own, err := s.repo.RoleRules(ctx, resourceTypeInstanceArea, []string{key}, []string{roleID})
	if err != nil {
		return nil, fmt.Errorf("rules for role %s on %s: %w", roleID, key, err)
	}
	held.ID = roleID
	ws := workspaceLayers{workspaceID: workspaceID, member: true, owner: held.IsOwner, roles: RolesHeld(held)}
	n := s.namer(ctx)
	out := &Explanation{ResourceType: resourceTypeInstanceArea, ResourceID: key, RoleID: roleID, Effective: make([]Answer, 0, len(area.Actions))}
	for _, a := range area.Actions {
		with, src := decide(ws, a, func() *Rules { return &Rules{Roles: own[key]} })
		without, from := decide(ws, a, func() *Rules { return &Rules{Roles: atFallback(own[key], a)} })
		out.Effective = append(out.Effective, Answer{Action: a, Verdict: Verdict{with, n.named(src)}, Fallback: Verdict{without, n.named(from)}})
	}
	return out, nil
}
