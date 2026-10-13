package access

import (
	"context"
	"errors"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SetScopes wires the project and membership lookups the Require checks resolve through.
func (s *Service) SetScopes(sc Scopes) {
	s.scopes = sc
}

// Require checks the calling actor holds action in workspaceID; an empty action asks only for membership, for
// what every member may read (chat, the inbox, People). A caller outside the workspace gets ErrNotFound, so a
// direct fetch never confirms what the workspace holds; a member without the action gets ErrForbidden. With no
// project named nobody but the Owner holds a project area. A call with no actor is the server's own (event
// consumers, workers) and passes: every adapter attaches one.
func (s *Service) Require(ctx context.Context, workspaceID string, action permissions.Action) error {
	return s.require(ctx, workspaceID, "", action)
}

func (s *Service) require(ctx context.Context, workspaceID, projectID string, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return defaultAutomationInside(ctx, workspaceID)
	}
	ws := s.layers(ctx, userID, workspaceID, projectID)
	if ws.hidden() {
		return apperrs.ErrNotFound
	}
	allowed, src := decide(ws, action, noRules)
	if userID != "" && allowed {
		return nil
	}
	if !ws.member {
		return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
	}
	return s.refusal(ctx, action, src)
}

// RequireProject is Require inside projectID, in the workspace it belongs to. An unknown project, or one the caller is
// not in, is ErrNotFound; the empty action asks whether the caller may open the project at all.
func (s *Service) RequireProject(ctx context.Context, projectID string, action permissions.Action) error {
	if _, checked := caller(ctx); !checked && defaultAutomationWorkspace(ctx) == "" {
		return nil
	}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		return err
	}
	return s.require(ctx, workspaceID, projectID, action)
}

// RequireStack is RequireProject on a stack of projectID, through the rules on stackID, a branch deployment's base
// (ADR 0148). A stack whose stacks:read a rule denies, or one the caller can't read refusing them, is ErrNotFound, so
// the answer never confirms it exists; a refusal on a stack they read names its source.
func (s *Service) RequireStack(ctx context.Context, projectID, stackID string, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return s.RequireProject(ctx, projectID, action)
	}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		return err
	}
	ws := s.layers(ctx, userID, workspaceID, projectID)
	if !ws.member || ws.hidden() {
		return fmt.Errorf("%w: stack %s", apperrs.ErrNotFound, stackID)
	}
	rules := func() []*Rules { return []*Rules{s.rules(ctx, resourceTypeStack, stackID, userID)} }
	reads, readSource := decide(ws, permissions.StacksRead, rules)
	if !reads && readSource.Kind == SourceRule {
		return fmt.Errorf("%w: stack %s", apperrs.ErrNotFound, stackID)
	}
	held, src := decide(ws, action, rules)
	if held {
		return nil
	}
	if !reads {
		return fmt.Errorf("%w: stack %s", apperrs.ErrNotFound, stackID)
	}
	return s.refusal(ctx, action, src)
}

// CallerStacks is what a stack or deploy list in workspaceID, or every workspace when empty, filters the caller on ctx
// by in SQL for action: the projects RequireProject lets them act in, and the stacks whose rules turn their project's
// answer as RequireStack turns it. Those come from one read of only the rules naming the caller or their roles on
// stacks of projects they are in, never an Owner's, so the list costs the same at any length and its clause names a
// stack only when a rule turns it (ADR 0140, ADR 0148). all is CallerProjects'.
func (s *Service) CallerStacks(ctx context.Context, workspaceID string, action permissions.Action) (permissions.StackReach, bool, error) {
	userID, checked := caller(ctx)
	if !checked {
		projectIDs, all, err := s.callerProjectsIn(ctx, workspaceID, action)
		return permissions.StackReach{ProjectIDs: projectIDs, Allowed: []string{}, Denied: []string{}}, all, err
	}
	ctx = s.memoized(ctx)
	workspaceIDs := []string{workspaceID}
	if workspaceID == "" {
		var err error
		if workspaceIDs, err = s.workspacesOf(ctx, userID); err != nil {
			return permissions.StackReach{}, false, err
		}
	}
	reach := permissions.StackReach{ProjectIDs: []string{}, Allowed: []string{}, Denied: []string{}}
	for _, id := range workspaceIDs {
		in, err := s.ProjectsWith(ctx, userID, id, action)
		if err != nil {
			return permissions.StackReach{}, false, err
		}
		reach.ProjectIDs = append(reach.ProjectIDs, in...)
	}
	ruled, err := s.ruledProjects(ctx, userID, workspaceIDs)
	if err != nil || len(ruled) == 0 {
		return reach, false, err
	}
	if reach.Allowed, reach.Denied, err = s.repo.StackRuleTurns(ctx, userID, action, ruled, reach.ProjectIDs); err != nil {
		return permissions.StackReach{}, false, fmt.Errorf("stack rules of %s: %w", userID, err)
	}
	return reach, false, nil
}

// callerProjectsIn is CallerProjectsIn, or CallerProjects with no workspace named.
func (s *Service) callerProjectsIn(ctx context.Context, workspaceID string, action permissions.Action) ([]string, bool, error) {
	if workspaceID == "" {
		return s.CallerProjects(ctx, action)
	}
	return s.CallerProjectsIn(ctx, workspaceID, action)
}

// RequireAnywhere checks an instance-level action (runners, topology, a stack outside every project) on the instance
// area it belongs to: the caller must hold it in at least one workspace they belong to, or by that area's rules.
func (s *Service) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	return s.RequireArea(ctx, permissions.InstanceAreaOf(action), action)
}

// ProjectsAnywhere lists the workspaces userID belongs to and, across them, the projects in which they hold action
// (permissions.Member: may open them), so a list spanning workspaces filters by both in SQL (ADR 0140).
func (s *Service) ProjectsAnywhere(ctx context.Context, userID string, action permissions.Action) (workspaceIDs, projectIDs []string, err error) {
	if userID == "" || s.scopes == nil {
		return []string{}, []string{}, nil
	}
	ctx = s.memoized(ctx)
	workspaceIDs, err = remember(ctx, membershipKey{userID}, func() ([]string, error) {
		return s.scopes.WorkspaceIDsForUser(ctx, userID)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list workspaces of %s: %w", userID, err)
	}
	projectIDs = []string{}
	for _, workspaceID := range workspaceIDs {
		ids, err := s.ProjectsWith(ctx, userID, workspaceID, action)
		if err != nil {
			return nil, nil, err
		}
		projectIDs = append(projectIDs, ids...)
	}
	return workspaceIDs, projectIDs, nil
}

// CallerProjects lists the projects in which RequireProject lets the caller on ctx act with action, so a list filters
// by them in SQL instead of row by row. all is true for a caller every project check passes, the server's own calls.
func (s *Service) CallerProjects(ctx context.Context, action permissions.Action) (projectIDs []string, all bool, err error) {
	userID, checked := caller(ctx)
	if checked {
		_, projectIDs, err = s.ProjectsAnywhere(ctx, userID, action)
		return projectIDs, false, err
	}
	home := defaultAutomationWorkspace(ctx)
	if home == "" {
		return nil, true, nil
	}
	if s.scopes == nil {
		return []string{}, false, nil
	}
	projectIDs, err = s.scopes.ProjectIDs(ctx, home)
	if err != nil {
		return nil, false, fmt.Errorf("list projects of workspace %s: %w", home, err)
	}
	return projectIDs, false, nil
}

// CallerProjectsIn is CallerProjects inside workspaceID alone, from one ProjectsWith batch: what a workspace's own
// project list filters by.
func (s *Service) CallerProjectsIn(ctx context.Context, workspaceID string, action permissions.Action) (projectIDs []string, all bool, err error) {
	userID, checked := caller(ctx)
	if checked {
		projectIDs, err = s.ProjectsWith(ctx, userID, workspaceID, action)
		return projectIDs, false, err
	}
	if defaultAutomationInside(ctx, workspaceID) != nil {
		return []string{}, false, nil
	}
	return nil, true, nil
}

// WorkspacesWith lists the workspaces in which the caller on ctx holds action; none for the server's own calls.
func (s *Service) WorkspacesWith(ctx context.Context, action permissions.Action) ([]string, error) {
	userID, checked := caller(ctx)
	if !checked {
		return []string{}, nil
	}
	workspaceIDs, err := s.workspacesOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, workspaceID := range workspaceIDs {
		if s.HasPermission(ctx, userID, workspaceID, action, "", "") {
			out = append(out, workspaceID)
		}
	}
	return out, nil
}

// HoldsAnywhere is HoldsArea on the instance area action belongs to; an Owner of any workspace holds every action,
// which is all instance-level power there is (ADR 0088).
func (s *Service) HoldsAnywhere(ctx context.Context, userID string, action permissions.Action) (bool, error) {
	return s.HoldsArea(ctx, userID, permissions.InstanceAreaOf(action), action)
}

// PermissionsAnywhere is every action userID holds on at least one instance area, or by a role in any workspace for an
// action outside every area, in catalog order: so a client shows the same areas the server lets through.
func (s *Service) PermissionsAnywhere(ctx context.Context, userID string) ([]string, error) {
	ctx = s.memoized(ctx)
	out := []string{}
	for _, a := range permissions.AllActions() {
		held, err := s.heldOnAnyArea(ctx, userID, a)
		if err != nil {
			return nil, err
		}
		if held {
			out = append(out, string(a))
		}
	}
	return out, nil
}

func (s *Service) workspacesOf(ctx context.Context, userID string) ([]string, error) {
	if userID == "" || s.scopes == nil {
		return nil, nil
	}
	workspaceIDs, err := remember(ctx, membershipKey{userID}, func() ([]string, error) {
		return s.scopes.WorkspaceIDsForUser(ctx, userID)
	})
	if err != nil {
		return nil, fmt.Errorf("list workspaces of %s: %w", userID, err)
	}
	return workspaceIDs, nil
}

// caller is the person a check is about; checked is false for the server's own calls and for a shipped default
// automation, which has no creator to resolve and whose token scopes the gateway already held it to.
func caller(ctx context.Context) (userID string, checked bool) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok {
		return "", false
	}
	if actor.ID == "" && actor.Automation != nil {
		return "", false
	}
	return actor.ID, true
}

// defaultAutomationInside holds a shipped default automation, whose scopes are its whole grant, to its own workspace.
func defaultAutomationInside(ctx context.Context, workspaceID string) error {
	home := defaultAutomationWorkspace(ctx)
	if home == "" || home == workspaceID {
		return nil
	}
	return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
}

// defaultAutomationWorkspace is the workspace of the creator-less automation on ctx, or "" for any other caller.
func defaultAutomationWorkspace(ctx context.Context) string {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID != "" || actor.Automation == nil {
		return ""
	}
	return actor.Automation.WorkspaceID
}

func (s *Service) projectWorkspace(ctx context.Context, projectID string) (string, error) {
	if s.scopes == nil {
		return "", fmt.Errorf("%w: no project lookup wired", apperrs.ErrForbidden)
	}
	workspaceID, err := remember(ctx, projectKey{projectID}, func() (string, error) {
		return s.scopes.WorkspaceIDForProject(ctx, projectID)
	})
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", fmt.Errorf("%w: project %s", apperrs.ErrNotFound, projectID)
	}
	if err != nil {
		return "", fmt.Errorf("resolve workspace of project %s: %w", projectID, err)
	}
	return workspaceID, nil
}

// IsOwner reports whether userID holds workspaceID's Owner role among their roles; someone outside it does not.
func (s *Service) IsOwner(ctx context.Context, userID, workspaceID string) (bool, error) {
	info, err := s.standing(ctx, userID, workspaceID)
	return info.IsOwnerRole, err
}

// HoldsRole reports whether userID holds roleID in a workspace they belong to: who a rule for that role reaches.
func (s *Service) HoldsRole(ctx context.Context, userID, roleID string) (bool, error) {
	if userID == "" || roleID == "" || s.scopes == nil {
		return false, nil
	}
	workspaceIDs, err := remember(ctx, membershipKey{userID}, func() ([]string, error) {
		return s.scopes.WorkspaceIDsForUser(ctx, userID)
	})
	if err != nil {
		return false, fmt.Errorf("list workspaces of %s: %w", userID, err)
	}
	for _, workspaceID := range workspaceIDs {
		info, err := s.standing(ctx, userID, workspaceID)
		if err != nil {
			return false, err
		}
		if slices.Contains(info.roleIDs(), roleID) {
			return true, nil
		}
	}
	return false, nil
}

// ChannelsDenied is which of channelIDs in workspaceID deny userID channels:read, in one read; never the Owner (ADR 0148).
func (s *Service) ChannelsDenied(ctx context.Context, userID, workspaceID string, channelIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	info, err := s.standing(ctx, userID, workspaceID)
	if err != nil || info.IsOwnerRole {
		return out, err
	}
	rules, err := s.channelRules(ctx, userID, channelIDs)
	if err != nil {
		return nil, fmt.Errorf("channel rules of %s: %w", userID, err)
	}
	for _, id := range channelIDs {
		if held, _ := applyRules(true, Source{}, rules[id], info.roleIDs(), permissions.ChannelsRead); !held {
			out[id] = true
		}
	}
	return out, nil
}

// PlaysWith is which of playIDs in workspaceID userID holds action on, their roles and each play's rules decided as
// HasPermission does, from one read of the play rules reaching them (ADR 0148).
func (s *Service) PlaysWith(ctx context.Context, userID, workspaceID string, action permissions.Action, playIDs []string) map[string]bool {
	out := make(map[string]bool, len(playIDs))
	if userID == "" || len(playIDs) == 0 {
		return out
	}
	ctx = s.memoized(ctx)
	ws := s.layers(ctx, userID, workspaceID, "")
	var rules map[string]*Rules
	for _, id := range playIDs {
		out[id], _ = decide(ws, action, func() []*Rules {
			if rules == nil {
				rules = s.playRules(ctx, userID, playIDs)
			}
			return []*Rules{rules[id]}
		})
	}
	return out
}

// playRules is channelRules for plays; a failed read counts as no rule, as a single check's does.
func (s *Service) playRules(ctx context.Context, userID string, playIDs []string) map[string]*Rules {
	read := func() (map[string]*Rules, error) { return s.rulesOn(ctx, resourceTypePlay, playIDs, userID) }
	if len(playIDs) > 1 {
		read = func() (map[string]*Rules, error) {
			return remember(ctx, typeRulesKey{resourceTypePlay, userID}, func() (map[string]*Rules, error) {
				return s.repo.RulesOfType(ctx, resourceTypePlay, userID)
			})
		}
	}
	rules, err := read()
	if err != nil || rules == nil {
		return map[string]*Rules{}
	}
	return rules
}

// channelRules reads a list's rules by type, which costs what the rules do rather than what naming each channel does.
func (s *Service) channelRules(ctx context.Context, userID string, channelIDs []string) (map[string]*Rules, error) {
	if len(channelIDs) == 1 {
		return s.rulesOn(ctx, resourceTypeChannel, channelIDs, userID)
	}
	return remember(ctx, typeRulesKey{resourceTypeChannel, userID}, func() (map[string]*Rules, error) {
		return s.repo.RulesOfType(ctx, resourceTypeChannel, userID)
	})
}

func (s *Service) standing(ctx context.Context, userID, workspaceID string) (RoleInfo, error) {
	if s.roles == nil {
		return RoleInfo{}, nil
	}
	info, err := s.memberRole(ctx, workspaceID, userID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return RoleInfo{}, nil
	}
	if err != nil {
		return RoleInfo{}, fmt.Errorf("membership of %s in workspace %s: %w", userID, workspaceID, err)
	}
	return info, nil
}

// ProjectActions is each project of workspaceID userID is in, with the project-area actions they hold there, from one
// read of the workspace's projects and one of the rules reaching them (feeds /api/workspaces/{id}/me).
func (s *Service) ProjectActions(ctx context.Context, userID, workspaceID string) ([]ProjectHeld, error) {
	if userID == "" || s.scopes == nil {
		return []ProjectHeld{}, nil
	}
	ctx = s.memoized(ctx)
	projectIDs, err := remember(ctx, workspaceProjectsKey{workspaceID}, func() ([]string, error) {
		return s.scopes.ProjectIDs(ctx, workspaceID)
	})
	if err != nil {
		return nil, fmt.Errorf("list projects of workspace %s: %w", workspaceID, err)
	}
	projects, err := s.layerRules(ctx, userID, workspaceID, s.roleIDsIn(ctx, userID, workspaceID), projectIDs)
	if err != nil {
		return nil, err
	}
	ws := s.layers(ctx, userID, workspaceID, "")
	out := make([]ProjectHeld, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		in := ws.in(projectID, projects[projectID])
		if !in.member || in.hidden() {
			continue
		}
		out = append(out, ProjectHeld{ProjectID: projectID, Actions: in.projectActions()})
	}
	return out, nil
}

// projectActions is every project-area action held inside the project ws is in.
func (ws workspaceLayers) projectActions() permissions.Set {
	var out []permissions.Action
	for _, a := range permissions.AllActions() {
		if permissions.AreaOf(a) != permissions.AreaProject {
			continue
		}
		if held, _ := decide(ws, a, noRules); held {
			out = append(out, a)
		}
	}
	return permissions.SetOf(out...)
}

// ProjectsWith lists the projects of workspaceID in which userID holds action (permissions.Member: may open them), from
// the layers and one read of the workspace's projects, so a list can filter by them in SQL instead of row by row.
func (s *Service) ProjectsWith(ctx context.Context, userID, workspaceID string, action permissions.Action) ([]string, error) {
	if userID == "" || s.scopes == nil {
		return []string{}, nil
	}
	ctx = s.memoized(ctx)
	projectIDs, err := remember(ctx, workspaceProjectsKey{workspaceID}, func() ([]string, error) {
		return s.scopes.ProjectIDs(ctx, workspaceID)
	})
	if err != nil {
		return nil, fmt.Errorf("list projects of workspace %s: %w", workspaceID, err)
	}
	projects, err := s.layerRules(ctx, userID, workspaceID, s.roleIDsIn(ctx, userID, workspaceID), projectIDs)
	if err != nil {
		return nil, err
	}
	ws := s.layers(ctx, userID, workspaceID, "")
	out := make([]string, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		if held, _ := decide(ws.in(projectID, projects[projectID]), action, noRules); held {
			out = append(out, projectID)
		}
	}
	return out, nil
}
