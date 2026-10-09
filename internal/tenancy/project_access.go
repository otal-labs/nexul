package tenancy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SetProjects wires the project lookups Every project and Project access are checked through.
func (s *Service) SetProjects(g ProjectGate) { s.projects = g }

// SetEveryProject switches a member between From role ("role") and None ("none", a Restricted member) (ADR 0097).
// Switching keeps their Project access rows, so a round trip loses nothing. From role hands out the role's project
// areas on every project, so the giver must hold those with no project named, which a Restricted giver never does.
func (s *Service) SetEveryProject(ctx context.Context, actorID, workspaceID, userID, every string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	if every != EveryProjectRole && every != EveryProjectNone {
		return fmt.Errorf("%w: every_project must be %q or %q", apperrs.ErrInvalid, EveryProjectRole, EveryProjectNone)
	}
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	m, err := s.nonOwnerMember(ctx, workspaceID, userID, "the workspace Owner always sees every project")
	if err != nil {
		return err
	}
	restricted := every == EveryProjectNone
	if m.Restricted == restricted {
		return nil
	}
	if !restricted {
		if err := s.requireHoldsEveryProject(ctx, actorID, workspaceID, userID, m.RoleID); err != nil {
			return err
		}
	}
	evt, err := s.memberUpdated(ctx, actorID, workspaceID, userID)
	if err != nil {
		return err
	}
	if err := s.members.SetRestricted(ctx, workspaceID, userID, restricted, evt); err != nil {
		return fmt.Errorf("set every project for %s in workspace %s: %w", userID, workspaceID, err)
	}
	return nil
}

// requireHoldsEveryProject refuses From role unless the giver holds, with no project, every project area the
// member's role and allow overrides would hand out on every project.
func (s *Service) requireHoldsEveryProject(ctx context.Context, actorID, workspaceID, userID, roleID string) error {
	role, err := s.roleNames.RolePermissions(ctx, workspaceID, roleID)
	if err != nil {
		return fmt.Errorf("read role %s: %w", roleID, err)
	}
	allow, _, err := s.members.Overrides(ctx, workspaceID, userID)
	if err != nil {
		return fmt.Errorf("read overrides for %s in workspace %s: %w", userID, workspaceID, err)
	}
	return s.requireHolds(ctx, actorID, workspaceID, projectAreas(slices.Concat(role, allow)))
}

// SetProjectAccess replaces a member's levels on one project of workspaceID; an empty allow takes the project away.
// Only project-area actions belong there, and a giver may add only what they hold on that project themselves.
func (s *Service) SetProjectAccess(ctx context.Context, actorID, workspaceID, userID, projectID string, allow permissions.Set) error {
	workspaceID = strings.TrimSpace(workspaceID)
	userID = strings.TrimSpace(userID)
	projectID = strings.TrimSpace(projectID)
	if err := validateProjectLevels(allow); err != nil {
		return err
	}
	if err := s.requireManageWorkspaceMembers(ctx, actorID, workspaceID); err != nil {
		return err
	}
	held, err := s.heldOnProject(ctx, actorID, workspaceID, projectID)
	if err != nil {
		return err
	}
	if _, err := s.nonOwnerMember(ctx, workspaceID, userID, "the workspace Owner bypasses Project access, so none can be set"); err != nil {
		return err
	}
	current, err := s.projectAccessOn(ctx, workspaceID, userID, projectID)
	if err != nil {
		return err
	}
	next := permissions.SetOf(allow...)
	if slices.Equal(current, next) {
		return nil
	}
	if err := permissions.RequireHeld(next.Except(current), held); err != nil {
		return err
	}
	event := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicProjectAccessChanged, Payload: ProjectAccessEvent{ResourceType: "project", ResourceID: projectID, UserID: userID, ActorID: actorID}}
	if err := s.members.SetProjectAccess(ctx, projectID, userID, next, event); err != nil {
		return fmt.Errorf("set project access for %s on %s: %w", userID, projectID, err)
	}
	return nil
}

// heldOnProject is what actorID holds inside projectID; a project outside workspaceID, or hidden from them, is not found.
func (s *Service) heldOnProject(ctx context.Context, actorID, workspaceID, projectID string) (permissions.Set, error) {
	if s.projects == nil {
		return nil, fmt.Errorf("%w: no project lookup wired", apperrs.ErrForbidden)
	}
	inWorkspace, err := s.projects.ProjectWorkspace(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve project %s: %w", projectID, err)
	}
	held, opens := s.projects.ProjectPermissions(ctx, actorID, projectID)
	if inWorkspace != workspaceID || !opens {
		return nil, fmt.Errorf("%w: project %s", apperrs.ErrNotFound, projectID)
	}
	return permissions.SetOfStrings(held), nil
}

func (s *Service) projectAccessOn(ctx context.Context, workspaceID, userID, projectID string) (permissions.Set, error) {
	rows, err := s.members.ProjectAccess(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("read project access for %s in workspace %s: %w", userID, workspaceID, err)
	}
	for _, row := range rows {
		if row.ProjectID == projectID {
			return row.Allow, nil
		}
	}
	return nil, nil
}

// nonOwnerMember returns userID's membership, refusing the Owner with ownerReason.
func (s *Service) nonOwnerMember(ctx context.Context, workspaceID, userID, ownerReason string) (*Member, error) {
	m, err := s.members.Member(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve membership of %s in workspace %s: %w", userID, workspaceID, err)
	}
	isOwner, err := s.roleNames.IsOwnerRole(ctx, workspaceID, m.RoleID)
	if err != nil {
		return nil, fmt.Errorf("check owner role %s: %w", m.RoleID, err)
	}
	if isOwner {
		return nil, fmt.Errorf("%w: %s", apperrs.ErrInvalid, ownerReason)
	}
	return m, nil
}

// validateProjectLevels accepts only grid actions of a project area, the only ones Project access answers.
func validateProjectLevels(allow permissions.Set) error {
	for _, action := range allow {
		if _, ok := permissions.ParseAction(string(action)); !ok {
			return fmt.Errorf("%w: %q is not a permission; the permission catalog lists them", apperrs.ErrInvalid, action)
		}
		if permissions.AreaOf(action) != permissions.AreaProject {
			return fmt.Errorf("%w: %s is not a project area, so Project access cannot carry it", apperrs.ErrInvalid, action)
		}
	}
	return nil
}

func projectAreas(set permissions.Set) permissions.Set {
	return permissions.SetOf(slices.DeleteFunc(slices.Clone(set), func(a permissions.Action) bool {
		return permissions.AreaOf(a) != permissions.AreaProject
	})...)
}

// MeProject is one project a Restricted member may open and the project-area actions they hold there.
type MeProject struct {
	ProjectID string          `json:"project_id"`
	Actions   permissions.Set `json:"actions"`
}

// MemberProjects tells a member whether they are restricted in workspaceID and, when they are, the projects they
// may open with what they hold in each, so a client hides what the server refuses.
func (s *Service) MemberProjects(ctx context.Context, workspaceID, userID string) (bool, []MeProject, error) {
	m, err := s.Member(ctx, workspaceID, userID)
	if err != nil {
		return false, nil, err
	}
	if !m.Restricted {
		return false, nil, nil
	}
	rows, err := s.members.ProjectAccess(ctx, workspaceID, userID)
	if err != nil {
		return false, nil, fmt.Errorf("read project access for %s in workspace %s: %w", userID, workspaceID, err)
	}
	out := make([]MeProject, len(rows))
	for i, row := range rows {
		out[i] = MeProject{ProjectID: row.ProjectID, Actions: row.Allow}
	}
	return true, out, nil
}

// ListProjectPeople is the workspace's People who may open projectID, for a ticket's developer and tester pickers;
// anyone who may open the project reads it.
func (s *Service) ListProjectPeople(ctx context.Context, actorID, projectID string) ([]Person, error) {
	if s.projects == nil {
		return nil, fmt.Errorf("%w: no project lookup wired", apperrs.ErrForbidden)
	}
	workspaceID, err := s.projects.ProjectWorkspace(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("resolve project %s: %w", projectID, err)
	}
	if _, opens := s.projects.ProjectPermissions(ctx, actorID, projectID); !opens {
		return nil, fmt.Errorf("%w: project %s", apperrs.ErrNotFound, projectID)
	}
	people, err := s.ListPeople(ctx, actorID, workspaceID)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(people, func(p Person) bool {
		_, opens := s.projects.ProjectPermissions(ctx, p.UserID, projectID)
		return !opens
	}), nil
}

// teamProjects fills each membership's Every project and the Project access the viewer may see: only projects
// they may open themselves, so the Team never names a project hidden from them.
func (s *Service) teamProjects(ctx context.Context, actorID string, memberships []*TeamMembership) error {
	rows, err := s.members.ListAllProjectAccess(ctx)
	if err != nil {
		return fmt.Errorf("list project access: %w", err)
	}
	visible := map[string]bool{}
	byMembership := map[string][]*ProjectAccess{}
	for _, row := range rows {
		seen, ok := visible[row.ProjectID]
		if !ok && s.projects != nil {
			_, seen = s.projects.ProjectPermissions(ctx, actorID, row.ProjectID)
			visible[row.ProjectID] = seen
		}
		if seen {
			key := row.WorkspaceID + "\x00" + row.UserID
			byMembership[key] = append(byMembership[key], row)
		}
	}
	for _, m := range memberships {
		m.EveryProject = everyProject(m.Restricted)
		m.Projects = byMembership[m.WorkspaceID+"\x00"+m.UserID]
	}
	return nil
}
