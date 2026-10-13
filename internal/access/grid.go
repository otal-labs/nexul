package access

import (
	"context"
	"fmt"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// GridSetup is whether a workspace's one-time access setup is done, and who finished it when; nobody for a workspace
// that started set up.
type GridSetup struct {
	Finished   bool       `json:"finished"`
	FinishedBy string     `json:"finished_by,omitempty"`
	Login      string     `json:"finished_by_login,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// GridRole is one column of the Access page: a role other than the Owner's, and how many people hold it.
type GridRole struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Holders int    `json:"holders"`
}

// GridProject is one row of the Projects grid: the roles and the people its rules list (In this project allowed).
type GridProject struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Prefix  string   `json:"prefix"`
	RoleIDs []string `json:"role_ids"`
	UserIDs []string `json:"user_ids"`
}

// GridChannel is one row of the Public channels grid: the roles its rules hide it from (channels:read denied).
type GridChannel struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	HiddenRoleIDs []string `json:"hidden_role_ids"`
}

// Grid is the Access page of one workspace as the caller may manage it: every project for the Owner and the projects
// they manage for anyone else, and the public channels for the Owner alone.
type Grid struct {
	WorkspaceID string        `json:"workspace_id"`
	Setup       GridSetup     `json:"setup"`
	Roles       []GridRole    `json:"roles"`
	Projects    []GridProject `json:"projects"`
	Channels    []GridChannel `json:"channels"`
}

// GridSource is what the Access page is built from, as stored: the grid without its rules applied, the workspace's
// members, and every rule on its projects and channels.
type GridSource struct {
	Grid
	MemberIDs []string
	Rules     []*Overwrite
}

// GridProjectSave is who a project should list: every role and person not named here is taken off it.
type GridProjectSave struct {
	ProjectID string   `json:"project_id"`
	RoleIDs   []string `json:"role_ids"`
	UserIDs   []string `json:"user_ids"`
}

// GridChannelSave is which roles a public channel should be hidden from: every other role sees it.
type GridChannelSave struct {
	ChannelID     string   `json:"channel_id"`
	HiddenRoleIDs []string `json:"hidden_role_ids"`
}

// GridSave is one save of the Access page: the rows it changes, and Finish, which ends the one-time setup.
type GridSave struct {
	Projects []GridProjectSave `json:"projects"`
	Channels []GridChannelSave `json:"channels"`
	Finish   bool              `json:"finish"`
}

// Grid is workspaceID's Access page for actorID, read in the same statements at any number of projects and channels.
func (s *Service) Grid(ctx context.Context, actorID, workspaceID string) (*Grid, error) {
	src, managed, owner, err := s.gridFor(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	g := src.Grid
	g.Projects = make([]GridProject, 0, len(managed))
	for _, p := range src.Projects {
		if !slices.Contains(managed, p.ID) {
			continue
		}
		p.RoleIDs, p.UserIDs = src.listed(p.ID)
		g.Projects = append(g.Projects, p)
	}
	g.Channels = []GridChannel{}
	if !owner {
		return &g, nil
	}
	for _, c := range src.Channels {
		c.HiddenRoleIDs = src.hiddenFrom(c.ID)
		g.Channels = append(g.Channels, c)
	}
	return &g, nil
}

// SaveGrid writes one save of the Access page in one transaction: each named project lists exactly the roles and
// people given, each named channel is hidden from exactly the roles given, and Finish ends the setup. Only the
// projects the actor manages may be named; channels and Finish are the Owner's.
func (s *Service) SaveGrid(ctx context.Context, actorID, workspaceID string, in GridSave) error {
	src, managed, owner, err := s.gridFor(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}
	if !owner && (in.Finish || len(in.Channels) > 0) {
		return fmt.Errorf("%w: only an Owner hides public channels or finishes the access setup", apperrs.ErrForbidden)
	}
	plan := src.newPlan()
	for _, p := range in.Projects {
		if err := plan.project(p, managed); err != nil {
			return err
		}
	}
	for _, c := range in.Channels {
		if err := plan.channel(c); err != nil {
			return err
		}
	}
	finishedBy := ""
	if in.Finish && !src.Setup.Finished {
		finishedBy = actorID
	}
	if len(plan.changed) == 0 && finishedBy == "" {
		return nil
	}
	event := GrantEvent{ResourceType: resourceTypeWorkspace, ResourceID: workspaceID, Target: TargetRole, ActorID: actorID}
	if err := s.repo.SaveGrid(ctx, workspaceID, plan.writes(), finishedBy, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicGrantChanged, Payload: event}); err != nil {
		return fmt.Errorf("save the access of workspace %s: %w", workspaceID, err)
	}
	return nil
}

// gridFor reads the grid for a member who manages some of workspaceID's projects: the projects they manage, every one
// for the Owner, and whether they are the Owner. Outside the workspace it is not found.
func (s *Service) gridFor(ctx context.Context, actorID, workspaceID string) (*GridSource, []string, bool, error) {
	if actorID == "" {
		return nil, nil, false, fmt.Errorf("%w: actor is required", apperrs.ErrUnauthorized)
	}
	ctx = s.memoized(ctx)
	ws := s.layers(ctx, actorID, workspaceID, "")
	if !ws.member {
		return nil, nil, false, fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
	}
	managed, err := s.ProjectsWith(ctx, actorID, workspaceID, permissions.ProjectsAccess)
	if err != nil {
		return nil, nil, false, err
	}
	if !ws.owner && len(managed) == 0 {
		return nil, nil, false, fmt.Errorf("%w: manage access on a project required", apperrs.ErrForbidden)
	}
	src, err := s.repo.Grid(ctx, workspaceID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("read the access of workspace %s: %w", workspaceID, err)
	}
	return src, managed, ws.owner, nil
}

// listed is the roles and people a project's rules let in, roles in column order.
func (src *GridSource) listed(projectID string) (roleIDs, userIDs []string) {
	roleIDs, userIDs = []string{}, []string{}
	for _, r := range src.Roles {
		if ow := src.rule(resourceTypeProject, projectID, TargetRole, r.ID); ow != nil && stateOf(ow, permissions.ProjectsOpen) == StateAllow {
			roleIDs = append(roleIDs, r.ID)
		}
	}
	for _, ow := range src.Rules {
		if ow.ResourceType == resourceTypeProject && ow.ResourceID == projectID && ow.Target == TargetPerson && stateOf(ow, permissions.ProjectsOpen) == StateAllow {
			userIDs = append(userIDs, ow.UserID)
		}
	}
	return roleIDs, userIDs
}

// hiddenFrom is the roles a channel's rules deny reading it, in column order.
func (src *GridSource) hiddenFrom(channelID string) []string {
	out := []string{}
	for _, r := range src.Roles {
		if ow := src.rule(resourceTypeChannel, channelID, TargetRole, r.ID); ow != nil && stateOf(ow, permissions.ChannelsRead) == StateDeny {
			out = append(out, r.ID)
		}
	}
	return out
}

func (src *GridSource) rule(resourceType, resourceID, target, targetID string) *Overwrite {
	for _, ow := range src.Rules {
		if ow.ResourceType == resourceType && ow.ResourceID == resourceID && ow.Target == target && ruleTarget(ow) == targetID {
			return ow
		}
	}
	return nil
}

func ruleTarget(ow *Overwrite) string {
	if ow.Target == TargetRole {
		return ow.RoleID
	}
	return ow.UserID
}

// gridPlan is the rules a save moves, each keyed by what it names, starting from the stored ones.
type gridPlan struct {
	src     *GridSource
	changed map[[4]string]*Overwrite
}

func (src *GridSource) newPlan() *gridPlan {
	return &gridPlan{src: src, changed: map[[4]string]*Overwrite{}}
}

// project lists exactly the roles and people in p on its project, moving In this project to Allow for whoever joins
// and back to Fallback for whoever leaves; a Deny on a role it doesn't name stays.
func (p *gridPlan) project(in GridProjectSave, managed []string) error {
	if !slices.Contains(managed, in.ProjectID) {
		return fmt.Errorf("%w: project %s", apperrs.ErrNotFound, in.ProjectID)
	}
	if err := p.knownRoles(in.RoleIDs); err != nil {
		return err
	}
	for _, id := range in.UserIDs {
		if !slices.Contains(p.src.MemberIDs, id) {
			return fmt.Errorf("%w: %s is not a member of this workspace", apperrs.ErrInvalid, id)
		}
	}
	for _, r := range p.src.Roles {
		p.set(resourceTypeProject, in.ProjectID, TargetRole, r.ID, permissions.ProjectsOpen, StateAllow, slices.Contains(in.RoleIDs, r.ID))
	}
	_, listed := p.src.listed(in.ProjectID)
	for _, id := range append(listed, in.UserIDs...) {
		p.set(resourceTypeProject, in.ProjectID, TargetPerson, id, permissions.ProjectsOpen, StateAllow, slices.Contains(in.UserIDs, id))
	}
	return nil
}

// channel hides a public channel from exactly the roles in c: channels:read to Deny for them, Fallback for the rest.
func (p *gridPlan) channel(in GridChannelSave) error {
	if !slices.ContainsFunc(p.src.Channels, func(c GridChannel) bool { return c.ID == in.ChannelID }) {
		return fmt.Errorf("%w: public channel %s", apperrs.ErrNotFound, in.ChannelID)
	}
	if err := p.knownRoles(in.HiddenRoleIDs); err != nil {
		return err
	}
	for _, r := range p.src.Roles {
		p.set(resourceTypeChannel, in.ChannelID, TargetRole, r.ID, permissions.ChannelsRead, StateDeny, slices.Contains(in.HiddenRoleIDs, r.ID))
	}
	return nil
}

func (p *gridPlan) knownRoles(roleIDs []string) error {
	for _, id := range roleIDs {
		if !slices.ContainsFunc(p.src.Roles, func(r GridRole) bool { return r.ID == id }) {
			return fmt.Errorf("%w: role %s is not one of this workspace's roles other than the Owner's", apperrs.ErrNotFound, id)
		}
	}
	return nil
}

// set moves action on one rule to state when on and the rule isn't there yet, and from it to Fallback when off and it
// is; any other state of the rule is left as it was.
func (p *gridPlan) set(resourceType, resourceID, target, targetID string, action permissions.Action, state string, on bool) {
	key := [4]string{resourceType, resourceID, target, targetID}
	ow := p.changed[key]
	if ow == nil {
		ow = p.src.rule(resourceType, resourceID, target, targetID)
	}
	if ow == nil {
		ow = &Overwrite{ResourceType: resourceType, ResourceID: resourceID, Target: target}
		ow.UserID, ow.RoleID = targetID, ""
		if target == TargetRole {
			ow.UserID, ow.RoleID = "", targetID
		}
	}
	current := stateOf(ow, action) == state
	if current == on {
		return
	}
	next := *ow
	next.Allow, next.Deny = moveTo(ow.Allow, ow.Deny, action, StateFallback)
	if on {
		next.Allow, next.Deny = moveTo(ow.Allow, ow.Deny, action, state)
	}
	p.changed[key] = &next
}

// writes is every moved rule, in a stable order.
func (p *gridPlan) writes() []*Overwrite {
	out := make([]*Overwrite, 0, len(p.changed))
	for _, ow := range p.changed {
		out = append(out, ow)
	}
	slices.SortFunc(out, func(a, b *Overwrite) int {
		return slices.Compare([]string{a.ResourceType, a.ResourceID, a.Target, ruleTarget(a)}, []string{b.ResourceType, b.ResourceID, b.Target, ruleTarget(b)})
	})
	return out
}
