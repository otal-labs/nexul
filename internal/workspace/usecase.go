// Package workspace implements projects, repositories, ticket moves, and the notifications inbox (ADR 0019).
package workspace

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// projectPrefixPattern matches the immutable project prefix (ADR 0004): a letter then 1-4 letters or digits, checked after uppercasing.
var projectPrefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,4}$`)

// Service is the workspace use-case layer (ADR 0019): projects, categories, ticket types, and status columns.
type Service struct {
	repo       Repo
	cats       CategoryRepo
	types      TicketTypeRepo
	statuses   StatusRepo
	gate       Gate
	workspaces WorkspaceGate
	tickets    TicketProjects
	instance   InstanceTemplates
	now        func() time.Time
}

// InstanceTemplates reads the instance's text for a template kind and key, the code default until edited (ADR 0103).
type InstanceTemplates interface {
	Effective(ctx context.Context, kind, key string) (string, error)
}

// SetInstanceTemplates wires the instance layer a new project's ticket types take their body templates from.
func (s *Service) SetInstanceTemplates(t InstanceTemplates) { s.instance = t }

// NewService wires the workspace use-cases over the given repos and gates.
func NewService(repo Repo, cats CategoryRepo, types TicketTypeRepo, statuses StatusRepo, gate Gate, workspaces WorkspaceGate) *Service {
	return &Service{repo: repo, cats: cats, types: types, statuses: statuses, gate: gate, workspaces: workspaces, now: time.Now}
}

// SetTicketProjects wires the ticket lookup a ticket move checks its source project through.
func (s *Service) SetTicketProjects(t TicketProjects) { s.tickets = t }

// requireIn checks action in workspaceID; permissions.Member asks only for membership.
func (s *Service) requireIn(ctx context.Context, workspaceID string, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.Require(ctx, workspaceID, action)
}

// requireOn is requireIn in the workspace projectID belongs to.
func (s *Service) requireOn(ctx context.Context, projectID string, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.RequireProject(ctx, projectID, action)
}

// Create adds a project to a workspace with the next display position (projects:write).
func (s *Service) Create(ctx context.Context, workspaceID, name, prefix string, icon ProjectIcon) (*Project, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required — create a workspace before creating a project", apperrs.ErrInvalid)
	}
	if err := s.requireIn(ctx, workspaceID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	ok, err := s.workspaces.WorkspaceExists(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("check workspace %s: %w", workspaceID, err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: workspace %s not found", apperrs.ErrNotFound, workspaceID)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: project name is required", apperrs.ErrInvalid)
	}
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if !projectPrefixPattern.MatchString(prefix) {
		return nil, fmt.Errorf("%w: project prefix must be 2-5 letters or digits, starting with a letter", apperrs.ErrInvalid)
	}
	icon = ProjectIcon(strings.TrimSpace(string(icon)))
	if icon != "" && !validProjectIcons[icon] {
		return nil, fmt.Errorf("%w: project icon must be one of the suggested icons", apperrs.ErrInvalid)
	}
	position, err := s.nextPosition(ctx, workspaceID, prefix)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	types, err := s.seedTicketTypes(ctx)
	if err != nil {
		return nil, err
	}
	p := &Project{ID: ids.New(), Name: name, Prefix: prefix, Position: position, WorkspaceID: workspaceID, Icon: icon, CreatedAt: now, UpdatedAt: now, SeedTicketTypes: types}
	// Default statuses/types are seeded by ProjectsRepo.Create in the same transaction, not here.
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return p, nil
}

// Get returns a project by id.
func (s *Service) Get(ctx context.Context, id string) (*Project, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", id, err)
	}
	if err := s.requireOn(ctx, id, permissions.Member); err != nil {
		return nil, fmt.Errorf("get project %s: %w", id, err)
	}
	return p, nil
}

// List returns the workspace's projects the caller may open, ordered by position (ADR 0097).
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Project, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.requireIn(ctx, workspaceID, permissions.Member); err != nil {
		return nil, err
	}
	projects, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return permissions.Filter(projects, func(p *Project) string { return p.ID }, func(projectID string) error {
		return s.requireOn(ctx, projectID, permissions.Member)
	})
}

// ProjectAccess lists the Restricted members who may open projectID and what they hold there; it takes
// members:write in the project's workspace, since it shows the people a manager sets access for.
func (s *Service) ProjectAccess(ctx context.Context, projectID string) ([]ProjectAccessEntry, error) {
	p, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := s.requireIn(ctx, p.WorkspaceID, permissions.MembersWrite); err != nil {
		return nil, err
	}
	entries, err := s.repo.ListRestrictedAccess(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list access to project %s: %w", projectID, err)
	}
	return entries, nil
}

// Rename updates a project's name and/or icon (projects:write); icon nil keeps current, "" clears it.
func (s *Service) Rename(ctx context.Context, id, name string, icon *ProjectIcon) (*Project, error) {
	if err := s.requireOn(ctx, id, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: project name is required", apperrs.ErrInvalid)
	}
	if icon != nil {
		trimmed := ProjectIcon(strings.TrimSpace(string(*icon)))
		if trimmed != "" && !validProjectIcons[trimmed] {
			return nil, fmt.Errorf("%w: project icon must be one of the suggested icons", apperrs.ErrInvalid)
		}
		icon = &trimmed
	}
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("rename project %s: %w", id, err)
	}
	updated := *current
	updated.Name = name
	if icon != nil {
		updated.Icon = *icon
	}
	updated.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("rename project %s: %w", id, err)
	}
	return &updated, nil
}

// SetPrefix needs projects:write and is refused if the project already has one.
func (s *Service) SetPrefix(ctx context.Context, id, prefix string) (*Project, error) {
	if err := s.requireOn(ctx, id, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if !projectPrefixPattern.MatchString(prefix) {
		return nil, fmt.Errorf("%w: project prefix must be 2-5 letters or digits, starting with a letter", apperrs.ErrInvalid)
	}
	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("set prefix for project %s: %w", id, err)
	}
	// Same prefix again is a no-op, so the owner wizard's finish can be retried after a later step failed.
	if current.Prefix == prefix {
		return current, nil
	}
	if current.Prefix != "" {
		return nil, fmt.Errorf("%w: project %s already has a prefix", apperrs.ErrConflict, id)
	}
	projects, err := s.repo.List(ctx, current.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	for _, p := range projects {
		if p.ID != current.ID && p.Prefix == prefix {
			return nil, fmt.Errorf("%w: project prefix %q is already in use", apperrs.ErrInvalid, prefix)
		}
	}
	updated := *current
	updated.Prefix = prefix
	updated.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("set prefix for project %s: %w", id, err)
	}
	return &updated, nil
}

// validateReorderIDs checks ids has no blank/duplicate entries and lists every existingIDs entry exactly once.
func validateReorderIDs(ids []string, existingIDs []string, noun string) error {
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("%w: %s ids are required", apperrs.ErrInvalid, noun)
		}
		if seen[id] {
			return fmt.Errorf("%w: duplicate %s id %s", apperrs.ErrInvalid, noun, id)
		}
		seen[id] = true
	}
	if len(ids) != len(existingIDs) {
		return fmt.Errorf("%w: every %s must be listed in the new order", apperrs.ErrInvalid, noun)
	}
	for _, eid := range existingIDs {
		if !seen[eid] {
			return fmt.Errorf("%w: every %s must be listed in the new order", apperrs.ErrInvalid, noun)
		}
	}
	return nil
}

// Reorder sets a workspace's project display order (projects:write); every project must appear exactly once.
func (s *Service) Reorder(ctx context.Context, workspaceID string, ids []string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.requireIn(ctx, workspaceID, permissions.ProjectsWrite); err != nil {
		return err
	}
	projects, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}
	existingIDs := make([]string, len(projects))
	for i, p := range projects {
		existingIDs[i] = p.ID
	}
	if err := validateReorderIDs(ids, existingIDs, "project"); err != nil {
		return err
	}
	return s.repo.Reorder(ctx, ids)
}

// DeleteImpact reports what removing a project would affect, so the management UI can warn before the owner confirms.
func (s *Service) DeleteImpact(ctx context.Context, id string) (DeleteImpact, error) {
	if err := s.requireOn(ctx, id, permissions.ProjectsRead); err != nil {
		return DeleteImpact{}, fmt.Errorf("impact of deleting project %s: %w", id, err)
	}
	return s.deleteImpact(ctx, id)
}

func (s *Service) deleteImpact(ctx context.Context, id string) (DeleteImpact, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return DeleteImpact{}, fmt.Errorf("impact of deleting project %s: %w", id, err)
	}
	tickets, err := s.repo.CountTickets(ctx, id)
	if err != nil {
		return DeleteImpact{}, fmt.Errorf("impact of deleting project %s: %w", id, err)
	}
	repos, err := s.repo.CountRepos(ctx, id)
	if err != nil {
		return DeleteImpact{}, fmt.Errorf("impact of deleting project %s: %w", id, err)
	}
	services, err := s.repo.CountServices(ctx, id)
	if err != nil {
		return DeleteImpact{}, fmt.Errorf("impact of deleting project %s: %w", id, err)
	}
	access, err := s.repo.ListRestrictedAccess(ctx, id)
	if err != nil {
		return DeleteImpact{}, fmt.Errorf("impact of deleting project %s: %w", id, err)
	}
	losing := make([]RestrictedMember, len(access))
	for i, a := range access {
		losing[i] = a.RestrictedMember
	}
	return DeleteImpact{Tickets: tickets, Repos: repos, Services: services, RestrictedMembers: losing}, nil
}

// Delete removes a project (projects:delete); refused while it has tickets, repos, or services, to avoid orphans.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.requireOn(ctx, id, permissions.ProjectsDelete); err != nil {
		return err
	}
	impact, err := s.deleteImpact(ctx, id)
	if err != nil {
		return err
	}
	if impact.Tickets > 0 || impact.Repos > 0 || impact.Services > 0 {
		return fmt.Errorf("%w: project has %d ticket(s), %d repo(s) and %d service(s); move or delete them first",
			apperrs.ErrConflict, impact.Tickets, impact.Repos, impact.Services)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete project %s: %w", id, err)
	}
	return nil
}

// defaultConnectorID is assumed when the caller doesn't say; github is the only real connector today.
const defaultConnectorID = "github"

// AddRepo associates a repository with a project (projects:write); a tests repository also records tests as separate.
func (s *Service) AddRepo(ctx context.Context, projectID, owner, name, connectorID string, role RepoRole) error {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return err
	}
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" || name == "" {
		return fmt.Errorf("%w: repo owner and name are required", apperrs.ErrInvalid)
	}
	connectorID = strings.TrimSpace(connectorID)
	if connectorID == "" {
		connectorID = defaultConnectorID
	}
	if role == "" {
		role = RepoRoleApp
	}
	if !role.Valid() {
		return fmt.Errorf("%w: repo role must be %q or %q", apperrs.ErrInvalid, RepoRoleApp, RepoRoleTests)
	}
	project, err := s.repo.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("add repo to project %s: %w", projectID, err)
	}
	if err := s.repo.AddRepo(ctx, projectID, RepoRef{Owner: owner, Name: name, FullName: owner + "/" + name, ConnectorID: connectorID, Role: role}); err != nil {
		return fmt.Errorf("add repo %s/%s: %w", owner, name, err)
	}
	if role != RepoRoleTests || project.TestsLocation == TestsLocationSeparate {
		return nil
	}
	_, err = s.saveTestsLocation(ctx, project, TestsLocationSeparate)
	return err
}

// SetTestsLocation records where a project's tests live (projects:write); "" withdraws the answer.
func (s *Service) SetTestsLocation(ctx context.Context, projectID string, location TestsLocation) (*Project, error) {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	location = TestsLocation(strings.TrimSpace(string(location)))
	if !location.Valid() {
		return nil, fmt.Errorf("%w: tests location must be %q, %q, or empty", apperrs.ErrInvalid, TestsLocationSame, TestsLocationSeparate)
	}
	project, err := s.repo.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("set tests location for project %s: %w", projectID, err)
	}
	return s.saveTestsLocation(ctx, project, location)
}

func (s *Service) saveTestsLocation(ctx context.Context, project *Project, location TestsLocation) (*Project, error) {
	updated := *project
	updated.TestsLocation = location
	updated.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("set tests location for project %s: %w", project.ID, err)
	}
	return &updated, nil
}

// RemoveRepo dissociates a repository from its project (projects:write).
func (s *Service) RemoveRepo(ctx context.Context, owner, name string) error {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" || name == "" {
		return fmt.Errorf("%w: repo owner and name are required", apperrs.ErrInvalid)
	}
	ref, err := s.repo.GetRepoByFullName(ctx, owner, name)
	if err != nil {
		return fmt.Errorf("remove repo %s/%s: %w", owner, name, err)
	}
	if err := s.requireOn(ctx, ref.ProjectID, permissions.ProjectsWrite); err != nil {
		return err
	}
	if err := s.repo.RemoveRepo(ctx, owner, name); err != nil {
		return fmt.Errorf("remove repo %s/%s: %w", owner, name, err)
	}
	return nil
}

// GetRepoByFullName reverse-looks-up which RepoRef owner/name is linked under, regardless of project.
func (s *Service) GetRepoByFullName(ctx context.Context, owner, name string) (RepoRef, error) {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" || name == "" {
		return RepoRef{}, fmt.Errorf("%w: repo owner and name are required", apperrs.ErrInvalid)
	}
	ref, err := s.repo.GetRepoByFullName(ctx, owner, name)
	if err != nil {
		return RepoRef{}, fmt.Errorf("get repo %s/%s: %w", owner, name, err)
	}
	return ref, nil
}

// ListRepos returns the repositories associated with a project.
func (s *Service) ListRepos(ctx context.Context, projectID string) ([]RepoRef, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireOn(ctx, projectID, permissions.ProjectsRead); err != nil {
		return nil, err
	}
	repos, err := s.repo.ListRepos(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list repos for project %s: %w", projectID, err)
	}
	return repos, nil
}

// MoveTicket moves a ticket into a project without changing its identity.
func (s *Service) MoveTicket(ctx context.Context, ticketID, projectID string) error {
	if strings.TrimSpace(ticketID) == "" {
		return fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	if _, err := s.repo.Get(ctx, projectID); err != nil {
		return fmt.Errorf("move ticket to project %s: %w", projectID, err)
	}
	if err := s.requireOn(ctx, projectID, permissions.TicketsWrite); err != nil {
		return fmt.Errorf("move ticket to project %s: %w", projectID, err)
	}
	if _, err := s.requireOnTicket(ctx, ticketID); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	if err := s.repo.MoveTicket(ctx, ticketID, projectID); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	return nil
}

// requireOnTicket checks tickets:write in the project a ticket sits in now, and returns that project.
func (s *Service) requireOnTicket(ctx context.Context, ticketID string) (string, error) {
	if s.tickets == nil {
		return "", permissions.Ungated(ctx)
	}
	projectID, err := s.tickets.ProjectOfTicket(ctx, ticketID)
	if err != nil {
		return "", err
	}
	return projectID, s.requireOn(ctx, projectID, permissions.TicketsWrite)
}

// nextPosition is the display position after workspaceID's last project, refusing a prefix already in use there.
func (s *Service) nextPosition(ctx context.Context, workspaceID, prefix string) (int, error) {
	projects, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return 0, fmt.Errorf("list projects: %w", err)
	}
	position := len(projects)
	for _, p := range projects {
		if p.Prefix == prefix {
			return 0, fmt.Errorf("%w: project prefix %q is already in use", apperrs.ErrInvalid, prefix)
		}
		if p.Position >= position {
			position = p.Position + 1
		}
	}
	return position, nil
}
