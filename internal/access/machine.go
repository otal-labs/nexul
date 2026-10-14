package access

import (
	"context"
	"errors"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// resourceTypeMachine keys a machine's rules by resource_id = the machine id; they name a role of any workspace or a
// person (ADR 0151).
const resourceTypeMachine = "machine"

// SourceNotOnMachine is a machine no rule lets the person see: it is not found, and so is every action on it.
const SourceNotOnMachine = "not_on_machine"

// SourceNoDeployTo is the machine a stack deploys to when its viewer can't see it or holds no Deploy to there.
const SourceNoDeployTo = "no_deploy_to"

// MachineScopes reads which machines exist straight from storage, past every gate.
type MachineScopes interface {
	// MachineIDs lists every machine.
	MachineIDs(ctx context.Context) ([]string, error)
	// MachineIDByName is the id of the machine called name, or ErrNotFound.
	MachineIDByName(ctx context.Context, name string) (string, error)
}

// SetMachines wires the machine lookup a machine check resolves ids through.
func (s *Service) SetMachines(m MachineScopes) {
	s.machines = m
}

func (s *Service) machineIDs(ctx context.Context) ([]string, error) {
	if s.machines == nil {
		return nil, nil
	}
	return remember(ctx, machineIDsKey{}, func() ([]string, error) {
		return s.machines.MachineIDs(ctx)
	})
}

// machineRules is every rule on a machine reaching userID, in one read for any number of machines.
func (s *Service) machineRules(ctx context.Context, userID string) (map[string]*Rules, error) {
	rules, err := remember(ctx, typeRulesKey{resourceTypeMachine, userID}, func() (map[string]*Rules, error) {
		return s.repo.RulesOfType(ctx, resourceTypeMachine, userID)
	})
	if err != nil {
		return nil, fmt.Errorf("machine rules of %s: %w", userID, err)
	}
	return rules, nil
}

// instanceOwnerSource is why userID is the instance Owner; false for anyone else, an Owner of another workspace included.
func (s *Service) instanceOwnerSource(ctx context.Context, userID string) (Source, bool, error) {
	if userID == "" || s.ownerWorkspace == "" {
		return Source{}, false, nil
	}
	info, err := s.standing(ctx, userID, s.ownerWorkspace)
	if err != nil || !info.IsOwnerRole {
		return Source{}, false, err
	}
	return sourceOf(SourceOwner, resourceTypeWorkspace, s.ownerWorkspace, TargetRole, info.sourceRole(""), true), true, nil
}

// machineAnswer is one action on one machine: whether it is seen at all, whether the action is held, and why.
type machineAnswer struct {
	seen, held bool
	src        Source
}

// machineHolder is who a machine answer is about, with everything it needs already read.
type machineHolder struct {
	targetType, targetID string
	roleIDs              []string
	// bits is an action from the roles' own sets alone.
	bits func(permissions.Action) (bool, Source)
	// rules are the machine's rules reaching the holder.
	rules *Rules
	// known is whether the machine exists; owner, when set, is why the holder is allowed everything on it.
	known bool
	owner *Source
}

// answer is action on machineID: seen only through the machine's rules, then the roles' bits, the rules for the roles
// with Allow winning, and the rule naming the person (ADR 0151). With fallback, the holder's own rule says nothing about
// action.
func (h machineHolder) answer(machineID string, action permissions.Action, fallback bool) machineAnswer {
	if !h.known {
		return machineAnswer{src: sourceOf(SourceNotOnMachine, resourceTypeMachine, machineID, h.targetType, h.targetID, false)}
	}
	if h.owner != nil {
		return machineAnswer{seen: true, held: true, src: *h.owner}
	}
	seeRules := h.rules
	if fallback && action == permissions.MachinesSee {
		seeRules = h.without(action)
	}
	notOn := sourceOf(SourceNotOnMachine, resourceTypeMachine, machineID, h.targetType, h.targetID, false)
	seen, src := applyRules(false, notOn, seeRules, h.roleIDs, permissions.MachinesSee)
	if !seen || action == permissions.MachinesSee {
		return machineAnswer{seen: seen, held: seen, src: src}
	}
	rules := h.rules
	if fallback {
		rules = h.without(action)
	}
	held, from := h.bits(action)
	held, from = applyRules(held, from, rules, h.roleIDs, action)
	return machineAnswer{seen: true, held: held, src: from}
}

// without is the holder's rules with their own saying nothing about action.
func (h machineHolder) without(action permissions.Action) *Rules {
	if h.rules == nil || h.targetType != TargetRole {
		return personAtFallback(h.rules, action)
	}
	return &Rules{Person: h.rules.Person, Roles: atFallback(h.rules.Roles, action)}
}

// machineView answers actions on machineID for userID from one read of their roles and the rules reaching them.
type machineView func(action permissions.Action, fallback bool) machineAnswer

func (s *Service) machineView(ctx context.Context, userID, machineID string) (machineView, error) {
	h, err := s.machineHolderOf(ctx, userID, machineID)
	if err != nil {
		return nil, err
	}
	return func(action permissions.Action, fallback bool) machineAnswer {
		return h.answer(machineID, action, fallback)
	}, nil
}

func (s *Service) machineHolderOf(ctx context.Context, userID, machineID string) (machineHolder, error) {
	h := machineHolder{targetType: TargetPerson, targetID: userID}
	if userID == "" {
		return h, nil
	}
	src, owner, err := s.instanceOwnerSource(ctx, userID)
	if err != nil {
		return h, err
	}
	if owner {
		ids, err := s.machineIDs(ctx)
		h.known, h.owner = slices.Contains(ids, machineID), &src
		return h, err
	}
	// A machine that doesn't exist carries no rule, so for anyone else it is simply one they don't see.
	h.known = true
	if h.roleIDs, h.bits, err = s.rolesAcross(ctx, userID); err != nil {
		return h, err
	}
	rules, err := s.machineRules(ctx, userID)
	h.rules = rules[machineID]
	return h, err
}

// rolesAcross is every role userID holds in any workspace and an action from their sets: the first workspace whose roles
// grant it is the source, else the last where they hold a role. An Owner role carries no set, so it grants none here.
func (s *Service) rolesAcross(ctx context.Context, userID string) ([]string, func(permissions.Action) (bool, Source), error) {
	workspaceIDs, err := s.workspacesOf(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	var roleIDs []string
	layers := make([]workspaceLayers, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		ws := s.layers(ctx, userID, workspaceID, "")
		roleIDs = append(roleIDs, ws.roles.roleIDs()...)
		layers = append(layers, ws)
	}
	return roleIDs, func(action permissions.Action) (bool, Source) {
		allowed, src := false, sourceOf(SourceRole, resourceTypeWorkspace, "", TargetPerson, userID, false)
		for _, ws := range layers {
			if allowed {
				break
			}
			if held, from := ws.granted(action); held || len(ws.roles.roleIDs()) > 0 {
				allowed, src = held, from
			}
		}
		return allowed, src
	}, nil
}

// MachinePermissions is which of a machine's actions (See, Deploy to, Manage, Delete) userID holds; a machine they can't
// see is not found. The instance Owner holds all four on every machine.
func (s *Service) MachinePermissions(ctx context.Context, userID, machineID string) (permissions.Set, error) {
	view, err := s.machineView(s.memoized(ctx), userID, machineID)
	if err != nil {
		return nil, err
	}
	var held []permissions.Action
	for _, a := range permissions.MachineActions {
		answer := view(a, false)
		if !answer.seen {
			return nil, fmt.Errorf("%w: machine %s", apperrs.ErrNotFound, machineID)
		}
		if answer.held {
			held = append(held, a)
		}
	}
	return permissions.SetOf(held...), nil
}

// RequireMachine checks the caller on ctx holds action (one of MachineActions) on machineID. A machine they can't see is
// not found for every action; a refusal on one they see names its source. The server's own calls pass.
func (s *Service) RequireMachine(ctx context.Context, machineID string, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return nil
	}
	if !slices.Contains(permissions.MachineActions, action) {
		return fmt.Errorf("%w: %s is not a machine action", apperrs.ErrInvalid, action)
	}
	view, err := s.machineView(ctx, userID, machineID)
	if err != nil {
		return err
	}
	answer := view(action, false)
	if !answer.seen {
		return fmt.Errorf("%w: machine %s", apperrs.ErrNotFound, machineID)
	}
	if !answer.held {
		return s.refusal(ctx, action, answer.src)
	}
	return nil
}

// RequireMachineNamed is RequireMachine for a machine a person types by name, aiming a stack or gateway at it. A machine
// they can't See answers exactly as a name no machine bears, invalid input naming the name and no id, so no one finds
// out which names exist by trying them (ADR 0151). The server's own calls pass.
func (s *Service) RequireMachineNamed(ctx context.Context, name string, action permissions.Action) error {
	return s.requireNamed(ctx, name, action, errNoMachineNamed(name))
}

// RequireMachineOfStack is RequireMachineNamed for the machine a stack the caller already found carries: one they can't
// See is not found, naming the machine and no id.
func (s *Service) RequireMachineOfStack(ctx context.Context, name string, action permissions.Action) error {
	return s.requireNamed(ctx, name, action, fmt.Errorf("%w: machine %q", apperrs.ErrNotFound, name))
}

func errNoMachineNamed(name string) error {
	return fmt.Errorf("%w: machine %q does not exist; machine_list lists machines", apperrs.ErrInvalid, name)
}

// requireNamed resolves name and checks action on it; unseen is what a machine the caller can't See answers.
func (s *Service) requireNamed(ctx context.Context, name string, action permissions.Action, unseen error) error {
	userID, checked := caller(ctx)
	if !checked {
		return nil
	}
	if s.machines == nil {
		return fmt.Errorf("%w: no machine lookup wired", apperrs.ErrForbidden)
	}
	_, answer, err := s.answerNamed(ctx, s.viewerOf(userID), name, action)
	if err != nil {
		return err
	}
	if !answer.seen {
		return unseen
	}
	if !answer.held {
		return s.refusal(ctx, action, answer.src)
	}
	return nil
}

// machineViewer reads one holder's view of a machine, a person's or a role's.
type machineViewer func(ctx context.Context, machineID string) (machineView, error)

func (s *Service) viewerOf(userID string) machineViewer {
	return func(ctx context.Context, machineID string) (machineView, error) {
		return s.machineView(ctx, userID, machineID)
	}
}

// answerNamed is the holder's answer for action on the machine called name; an unknown name is errNoMachineNamed.
func (s *Service) answerNamed(ctx context.Context, viewer machineViewer, name string, action permissions.Action) (string, machineAnswer, error) {
	id, err := s.machines.MachineIDByName(ctx, name)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", machineAnswer{}, errNoMachineNamed(name)
	}
	if err != nil {
		return "", machineAnswer{}, fmt.Errorf("resolve machine %q: %w", name, err)
	}
	view, err := viewer(ctx, id)
	if err != nil {
		return "", machineAnswer{}, err
	}
	return id, view(action, false), nil
}

// noDeployTo must answer as requireNamed does for a deploy start, so explain never disagrees with the click.
func (s *Service) noDeployTo(ctx context.Context, viewer machineViewer, targetType, targetID, name string) (Source, bool, error) {
	if s.machines == nil {
		return Source{}, false, nil
	}
	id, answer, err := s.answerNamed(ctx, viewer, name, permissions.MachinesDeploy)
	if errors.Is(err, apperrs.ErrInvalid) {
		return sourceOf(SourceNoDeployTo, resourceTypeMachine, name, targetType, targetID, false), true, nil
	}
	if err != nil || (answer.seen && answer.held) {
		return Source{}, false, err
	}
	return sourceOf(SourceNoDeployTo, resourceTypeMachine, id, targetType, targetID, false), true, nil
}

// MachineViewers addresses a deleted machine's live frame, because its rules go with it.
func (s *Service) MachineViewers(ctx context.Context, machineID string) ([]string, error) {
	users, err := s.users.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	ctx = s.memoized(ctx)
	viewers := []string{}
	for _, u := range users {
		view, err := s.machineView(ctx, u.ID, machineID)
		if err != nil {
			return nil, err
		}
		if view(permissions.MachinesSee, false).seen {
			viewers = append(viewers, u.ID)
		}
	}
	return viewers, nil
}

// MachinesSeen is the ids of the machines userID sees, from one read of the rules reaching them however many machines
// carry any: every machine for the instance Owner. A list filters by it in SQL (ADR 0140).
func (s *Service) MachinesSeen(ctx context.Context, userID string) ([]string, error) {
	ids := []string{}
	if userID == "" {
		return ids, nil
	}
	ctx = s.memoized(ctx)
	_, owner, err := s.instanceOwnerSource(ctx, userID)
	if err != nil {
		return nil, err
	}
	if owner {
		all, err := s.machineIDs(ctx)
		return append(ids, all...), err
	}
	roleIDs, _, err := s.rolesAcross(ctx, userID)
	if err != nil {
		return nil, err
	}
	rules, err := s.machineRules(ctx, userID)
	if err != nil {
		return nil, err
	}
	for id, r := range rules {
		if seen, _ := applyRules(false, Source{}, r, roleIDs, permissions.MachinesSee); seen {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

// CallerMachines lists the machines the caller on ctx sees, so a list filters by them in SQL (ADR 0140). all is true for
// a caller who sees every machine without asking, the server's own calls and the instance Owner, and ids is then empty.
func (s *Service) CallerMachines(ctx context.Context) (machineIDs []string, all bool, err error) {
	userID, checked := caller(ctx)
	if !checked {
		return nil, true, nil
	}
	ctx = s.memoized(ctx)
	if _, owner, err := s.instanceOwnerSource(ctx, userID); err != nil || owner {
		return nil, owner, err
	}
	machineIDs, err = s.MachinesSeen(ctx, userID)
	return machineIDs, false, err
}

// requireMachineSeen is not found for a machine the actor can't see, so a write or a read of its rules never confirms it.
func (s *Service) requireMachineSeen(ctx context.Context, actorID, machineID string) error {
	view, err := s.machineView(ctx, actorID, machineID)
	if err != nil {
		return err
	}
	if !view(permissions.MachinesSee, false).seen {
		return fmt.Errorf("%w: machine %s", apperrs.ErrNotFound, machineID)
	}
	return nil
}

// holdsOnMachine is whether actorID holds action on machineID, its own rules included.
func (s *Service) holdsOnMachine(ctx context.Context, actorID, machineID string, action permissions.Action) bool {
	view, err := s.machineView(ctx, actorID, machineID)
	return err == nil && view(action, false).held
}

// decideMachine is Decide on a machine: the answer for action and what decided it, a machine not seen answering false
// with the source that hid it.
func (s *Service) decideMachine(ctx context.Context, userID, machineID string, action permissions.Action) (bool, Source, error) {
	view, err := s.machineView(ctx, userID, machineID)
	if err != nil {
		return false, Source{}, err
	}
	answer := view(action, false)
	return answer.held, answer.src, nil
}

// explainMachine answers each of a machine's four actions for userID, with the answer a rule naming them at Fallback would
// give; anyone who sees the machine explains themselves, and only a Manage holder someone else.
func (s *Service) explainMachine(ctx context.Context, actorID, machineID, userID string) (*Explanation, error) {
	actor, err := s.machineView(ctx, actorID, machineID)
	if err != nil {
		return nil, err
	}
	if !actor(permissions.MachinesSee, false).seen {
		return nil, fmt.Errorf("%w: machine %s", apperrs.ErrNotFound, machineID)
	}
	if manage := actor(permissions.MachinesWrite, false); userID != actorID && !manage.held {
		return nil, s.refusal(ctx, permissions.MachinesWrite, manage.src)
	}
	who := actor
	if userID != actorID {
		if who, err = s.machineView(ctx, userID, machineID); err != nil {
			return nil, err
		}
	}
	return s.machineExplanation(ctx, machineID, who, &Explanation{UserID: userID}), nil
}

// explainRoleMachine is explainMachine for a holder of roleID alone, with the role's rule and with it at Fallback; it
// takes managing the machine.
func (s *Service) explainRoleMachine(ctx context.Context, actorID, machineID, roleID string) (*Explanation, error) {
	actor, err := s.machineView(ctx, actorID, machineID)
	if err != nil {
		return nil, err
	}
	if !actor(permissions.MachinesSee, false).seen {
		return nil, fmt.Errorf("%w: machine %s", apperrs.ErrNotFound, machineID)
	}
	if manage := actor(permissions.MachinesWrite, false); !manage.held {
		return nil, s.refusal(ctx, permissions.MachinesWrite, manage.src)
	}
	if err := s.requireRolesOf(ctx, resourceTypeMachine, machineID, []string{roleID}); err != nil {
		return nil, err
	}
	who, err := s.roleMachineView(ctx, machineID, roleID)
	if err != nil {
		return nil, err
	}
	return s.machineExplanation(ctx, machineID, who, &Explanation{RoleID: roleID}), nil
}

// roleMachineView reads only the role's bits and rules; a rule naming a person never reaches a role.
func (s *Service) roleMachineView(ctx context.Context, machineID, roleID string) (machineView, error) {
	workspaceID, err := s.roleWorkspace.WorkspaceIDForRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("resolve role %s: %w", roleID, err)
	}
	held, err := s.roleWorkspace.HeldRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("read role %s: %w", roleID, err)
	}
	own, err := s.repo.RoleRules(ctx, resourceTypeMachine, []string{machineID}, []string{roleID})
	if err != nil {
		return nil, fmt.Errorf("rules for role %s on machine %s: %w", roleID, machineID, err)
	}
	held.ID = roleID
	ws := workspaceLayers{workspaceID: workspaceID, member: true, owner: held.IsOwner, roles: RolesHeld(held)}
	h := machineHolder{targetType: TargetRole, targetID: roleID, roleIDs: []string{roleID}, bits: ws.granted, rules: &Rules{Roles: own[machineID]}, known: true}
	if held.IsOwner && workspaceID == s.ownerWorkspace {
		src := sourceOf(SourceOwner, resourceTypeWorkspace, workspaceID, TargetRole, roleID, true)
		h.owner = &src
	}
	return func(action permissions.Action, fallback bool) machineAnswer {
		return h.answer(machineID, action, fallback)
	}, nil
}

func (s *Service) machineExplanation(ctx context.Context, machineID string, who machineView, out *Explanation) *Explanation {
	n := s.namer(ctx)
	out.ResourceType, out.ResourceID = resourceTypeMachine, machineID
	out.Effective = make([]Answer, 0, len(permissions.MachineActions))
	for _, a := range permissions.MachineActions {
		held, inherited := who(a, false), who(a, true)
		out.Effective = append(out.Effective, Answer{Action: a, Verdict: Verdict{held.held, n.named(held.src)}, Fallback: Verdict{inherited.held, n.named(inherited.src)}})
	}
	return out
}
