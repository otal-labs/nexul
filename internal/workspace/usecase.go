// Package workspace implements projects, repositories, ticket moves, and the notifications inbox (ADR 0019).
package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
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
func (s *Service) Create(ctx context.Context, userID, workspaceID, name, prefix string, icon ProjectIcon) (*Project, error) {
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
func (s *Service) Rename(ctx context.Context, userID, id, name string, icon *ProjectIcon) (*Project, error) {
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
func (s *Service) SetPrefix(ctx context.Context, userID, id, prefix string) (*Project, error) {
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
func (s *Service) Reorder(ctx context.Context, userID, workspaceID string, ids []string) error {
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
func (s *Service) Delete(ctx context.Context, userID, id string) error {
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
func (s *Service) AddRepo(ctx context.Context, userID, projectID, owner, name, connectorID string, role RepoRole) error {
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
func (s *Service) SetTestsLocation(ctx context.Context, userID, projectID string, location TestsLocation) (*Project, error) {
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
func (s *Service) RemoveRepo(ctx context.Context, userID, owner, name string) error {
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
	if err := s.requireOnTicket(ctx, ticketID); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	if err := s.repo.MoveTicket(ctx, ticketID, projectID); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	return nil
}

// requireOnTicket checks tickets:write in the project a ticket sits in now.
func (s *Service) requireOnTicket(ctx context.Context, ticketID string) error {
	if s.tickets == nil {
		return permissions.Ungated(ctx)
	}
	projectID, err := s.tickets.ProjectOfTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	return s.requireOn(ctx, projectID, permissions.TicketsWrite)
}

// CreateCategory adds a category to a project with the next display position (projects:write).
func (s *Service) CreateCategory(ctx context.Context, userID, projectID, name string, color colors.Color) (*Category, error) {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: category name is required", apperrs.ErrInvalid)
	}
	color = colors.Color(strings.TrimSpace(string(color)))
	if color != "" && !colors.Valid(color) {
		return nil, fmt.Errorf("%w: category color must be one of the suggested colors", apperrs.ErrInvalid)
	}
	cats, err := s.cats.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list categories for project %s: %w", projectID, err)
	}
	position := len(cats)
	for _, c := range cats {
		if c.Position >= position {
			position = c.Position + 1
		}
	}
	now := s.now().UTC()
	c := &Category{ID: ids.New(), ProjectID: projectID, Name: name, Position: position, Color: color, CreatedAt: now, UpdatedAt: now}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCategoryCreated, Payload: CategoryEvent{Category: *c}}
	if err := s.cats.Create(ctx, c, evt); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return c, nil
}

// GetCategory returns a category by id.
func (s *Service) GetCategory(ctx context.Context, id string) (*Category, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: category id is required", apperrs.ErrInvalid)
	}
	c, err := s.cats.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get category %s: %w", id, err)
	}
	if err := s.requireOn(ctx, c.ProjectID, permissions.Member); err != nil {
		return nil, fmt.Errorf("get category %s: %w", id, err)
	}
	return c, nil
}

// ListCategories returns all categories ordered by project, position.
func (s *Service) ListCategories(ctx context.Context) ([]*Category, error) {
	cats, err := s.cats.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return permissions.Filter(cats, func(c *Category) string { return c.ProjectID }, func(projectID string) error {
		return s.requireOn(ctx, projectID, permissions.Member)
	})
}

// ListCategoriesByProject returns a project's categories ordered by position.
func (s *Service) ListCategoriesByProject(ctx context.Context, projectID string) ([]*Category, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireOn(ctx, projectID, permissions.Member); err != nil {
		return nil, err
	}
	cats, err := s.cats.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list categories for project %s: %w", projectID, err)
	}
	return cats, nil
}

// RenameCategory updates a category's name and/or color (projects:write); its identity never changes.
func (s *Service) RenameCategory(ctx context.Context, userID, id, name string, color colors.Color) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: category name is required", apperrs.ErrInvalid)
	}
	color = colors.Color(strings.TrimSpace(string(color)))
	if color != "" && !colors.Valid(color) {
		return nil, fmt.Errorf("%w: category color must be one of the suggested colors", apperrs.ErrInvalid)
	}
	current, err := s.cats.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("rename category %s: %w", id, err)
	}
	if err := s.requireOn(ctx, current.ProjectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	updated := *current
	updated.Name = name
	updated.Color = color
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCategoryUpdated, Payload: CategoryEvent{Category: updated}}
	if err := s.cats.Update(ctx, &updated, evt); err != nil {
		return nil, fmt.Errorf("rename category %s: %w", id, err)
	}
	return &updated, nil
}

// ReorderCategories sets a project's category display order (projects:write); every category appears exactly once.
func (s *Service) ReorderCategories(ctx context.Context, userID, projectID string, ids []string) error {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return err
	}
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	cats, err := s.cats.ListByProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list categories for project %s: %w", projectID, err)
	}
	existingIDs := make([]string, len(cats))
	for i, c := range cats {
		existingIDs[i] = c.ID
	}
	if err := validateReorderIDs(ids, existingIDs, "category"); err != nil {
		return err
	}
	return s.cats.Reorder(ctx, projectID, ids)
}

// DeleteCategory removes a category (projects:delete); its tickets become uncategorized rather than being deleted.
func (s *Service) DeleteCategory(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: category id is required", apperrs.ErrInvalid)
	}
	current, err := s.cats.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("delete category %s: %w", id, err)
	}
	if err := s.requireOn(ctx, current.ProjectID, permissions.ProjectsDelete); err != nil {
		return err
	}
	if err := s.cats.Delete(ctx, id, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicCategoryDeleted, Payload: CategoryEvent{Category: Category{ID: id}}}); err != nil {
		return fmt.Errorf("delete category %s: %w", id, err)
	}
	return nil
}

// MoveTicketToCategory moves a ticket into a category; an empty categoryID uncategorizes it.
func (s *Service) MoveTicketToCategory(ctx context.Context, ticketID, categoryID string) error {
	if strings.TrimSpace(ticketID) == "" {
		return fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(categoryID) != "" {
		if _, err := s.cats.Get(ctx, categoryID); err != nil {
			return fmt.Errorf("move ticket %s to category %s: %w", ticketID, categoryID, err)
		}
	}
	if err := s.requireOnTicket(ctx, ticketID); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicTicketCategoryChanged, Payload: TicketCategoryChangedEvent{TicketID: ticketID, CategoryID: strings.TrimSpace(categoryID)}}
	if err := s.cats.SetTicketCategory(ctx, ticketID, strings.TrimSpace(categoryID), evt); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	return nil
}

// CreateTicketType adds a project ticket type with the next display position (projects:write).
func (s *Service) CreateTicketType(ctx context.Context, userID, projectID, name string, color colors.Color) (*TicketType, error) {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: ticket type name is required", apperrs.ErrInvalid)
	}
	color = colors.Color(strings.TrimSpace(string(color)))
	if color != "" && !colors.Valid(color) {
		return nil, fmt.Errorf("%w: ticket type color must be one of the suggested colors", apperrs.ErrInvalid)
	}
	types, err := s.types.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list ticket types for project %s: %w", projectID, err)
	}
	position := len(types)
	for _, t := range types {
		if t.Position >= position {
			position = t.Position + 1
		}
	}
	now := s.now().UTC()
	tt := &TicketType{ID: ids.New(), ProjectID: projectID, Name: name, Position: position, Color: color, CreatedAt: now, UpdatedAt: now}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicTicketTypeCreated, Payload: TicketTypeEvent{TicketType: *tt}}
	if err := s.types.Create(ctx, tt, evt); err != nil {
		return nil, fmt.Errorf("create ticket type: %w", err)
	}
	return tt, nil
}

// GetTicketType returns a ticket type by id.
func (s *Service) GetTicketType(ctx context.Context, id string) (*TicketType, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: ticket type id is required", apperrs.ErrInvalid)
	}
	tt, err := s.types.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get ticket type %s: %w", id, err)
	}
	if err := s.requireOn(ctx, tt.ProjectID, permissions.Member); err != nil {
		return nil, fmt.Errorf("get ticket type %s: %w", id, err)
	}
	return tt, nil
}

// ListTicketTypesByProject returns a project's ticket types ordered by position; there is no unscoped list.
func (s *Service) ListTicketTypesByProject(ctx context.Context, projectID string) ([]*TicketType, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireOn(ctx, projectID, permissions.Member); err != nil {
		return nil, err
	}
	types, err := s.types.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list ticket types for project %s: %w", projectID, err)
	}
	return types, nil
}

// RenameTicketType updates a ticket type's name and/or color (projects:write); its identity never changes.
func (s *Service) RenameTicketType(ctx context.Context, userID, id, name string, color colors.Color) (*TicketType, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: ticket type name is required", apperrs.ErrInvalid)
	}
	color = colors.Color(strings.TrimSpace(string(color)))
	if color != "" && !colors.Valid(color) {
		return nil, fmt.Errorf("%w: ticket type color must be one of the suggested colors", apperrs.ErrInvalid)
	}
	current, err := s.types.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("rename ticket type %s: %w", id, err)
	}
	if err := s.requireOn(ctx, current.ProjectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	updated := *current
	updated.Name = name
	updated.Color = color
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicTicketTypeUpdated, Payload: TicketTypeEvent{TicketType: updated}}
	if err := s.types.Update(ctx, &updated, evt); err != nil {
		return nil, fmt.Errorf("rename ticket type %s: %w", id, err)
	}
	return &updated, nil
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

// seedTicketTypes is DefaultTicketTypes with the instance's body templates, copied into the project once (ADR 0103).
func (s *Service) seedTicketTypes(ctx context.Context) ([]TicketType, error) {
	types := slices.Clone(DefaultTicketTypes)
	if s.instance == nil {
		return types, nil
	}
	for i := range types {
		body, err := s.instance.Effective(ctx, TemplateKind, types[i].Name)
		if err != nil {
			return nil, fmt.Errorf("get the instance body template for %s: %w", types[i].Name, err)
		}
		types[i].BodyTemplate = body
	}
	return types, nil
}

// TicketTypeNamed returns projectID's ticket type whose name matches name, ignoring case (membership).
func (s *Service) TicketTypeNamed(ctx context.Context, projectID, name string) (*TicketType, error) {
	types, err := s.ListTicketTypesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, t := range types {
		if strings.EqualFold(t.Name, strings.TrimSpace(name)) {
			return t, nil
		}
	}
	return nil, fmt.Errorf("%w: project %s has no ticket type named %q", apperrs.ErrNotFound, projectID, name)
}

// SetTicketTypeTemplate replaces a type's body template (projects:write); existing tickets keep the body they were born with.
func (s *Service) SetTicketTypeTemplate(ctx context.Context, userID, id, template string) (*TicketType, error) {
	current, err := s.types.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("set ticket type template %s: %w", id, err)
	}
	if err := s.requireOn(ctx, current.ProjectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	updated := *current
	updated.BodyTemplate = template
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicTicketTypeUpdated, Payload: TicketTypeEvent{TicketType: updated}}
	if err := s.types.Update(ctx, &updated, evt); err != nil {
		return nil, fmt.Errorf("set ticket type template %s: %w", id, err)
	}
	return &updated, nil
}

// ReorderTicketTypes sets a project's ticket type display order (projects:write); every type appears exactly once.
func (s *Service) ReorderTicketTypes(ctx context.Context, userID, projectID string, ids []string) error {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	types, err := s.types.ListByProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list ticket types for project %s: %w", projectID, err)
	}
	existingIDs := make([]string, len(types))
	for i, t := range types {
		existingIDs[i] = t.ID
	}
	if err := validateReorderIDs(ids, existingIDs, "ticket type"); err != nil {
		return err
	}
	return s.types.Reorder(ctx, projectID, ids)
}

// DeleteTicketType removes a ticket type (projects:delete); refused while tickets still use it, so none is left orphaned.
func (s *Service) DeleteTicketType(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: ticket type id is required", apperrs.ErrInvalid)
	}
	current, err := s.types.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("delete ticket type %s: %w", id, err)
	}
	if err := s.requireOn(ctx, current.ProjectID, permissions.ProjectsDelete); err != nil {
		return err
	}
	count, err := s.types.CountTickets(ctx, id)
	if err != nil {
		return fmt.Errorf("count tickets of type %s: %w", id, err)
	}
	if count > 0 {
		return fmt.Errorf("%w: ticket type %s has %d ticket(s); move them to another type first", apperrs.ErrConflict, id, count)
	}
	if err := s.types.Delete(ctx, id, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicTicketTypeDeleted, Payload: TicketTypeEvent{TicketType: TicketType{ID: id}}}); err != nil {
		return fmt.Errorf("delete ticket type %s: %w", id, err)
	}
	return nil
}

// CreateStatus adds a project status column with the next display position (projects:write).
func (s *Service) CreateStatus(ctx context.Context, userID, projectID, name string, kind StatusKind, icon StatusIcon) (*Status, error) {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: status name is required", apperrs.ErrInvalid)
	}
	kind = StatusKind(strings.TrimSpace(string(kind)))
	if kind == "" {
		kind = StatusKindBacklog
	}
	if !kind.Valid() {
		return nil, fmt.Errorf("%w: status kind must be one of %s", apperrs.ErrInvalid, strings.Join(statusKindNames(), ", "))
	}
	icon = StatusIcon(strings.TrimSpace(string(icon)))
	if icon != "" && !validStatusIcons[icon] {
		return nil, fmt.Errorf("%w: status icon must be one of the suggested icons", apperrs.ErrInvalid)
	}
	statuses, err := s.statuses.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list statuses for project %s: %w", projectID, err)
	}
	position := len(statuses)
	for _, st := range statuses {
		if st.Position >= position {
			position = st.Position + 1
		}
	}
	now := s.now().UTC()
	st := &Status{ID: ids.New(), ProjectID: projectID, Name: name, Position: position, Kind: kind, Icon: icon, CreatedAt: now, UpdatedAt: now}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicStatusCreated, Payload: StatusEvent{Status: *st}}
	if err := s.statuses.Create(ctx, st, evt); err != nil {
		return nil, fmt.Errorf("create status: %w", err)
	}
	return st, nil
}

// GetStatus returns a status column by id.
func (s *Service) GetStatus(ctx context.Context, id string) (*Status, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: status id is required", apperrs.ErrInvalid)
	}
	st, err := s.statuses.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get status %s: %w", id, err)
	}
	if err := s.requireOn(ctx, st.ProjectID, permissions.Member); err != nil {
		return nil, fmt.Errorf("get status %s: %w", id, err)
	}
	return st, nil
}

// ListStatusesByProject returns a project's status columns ordered by position; there is no unscoped list.
func (s *Service) ListStatusesByProject(ctx context.Context, projectID string) ([]*Status, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	if err := s.requireOn(ctx, projectID, permissions.Member); err != nil {
		return nil, err
	}
	statuses, err := s.statuses.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list statuses for project %s: %w", projectID, err)
	}
	return statuses, nil
}

// RenameStatus updates a status column's name, kind, and icon (projects:write); existing tickets stay on it.
func (s *Service) RenameStatus(ctx context.Context, userID, id, name string, kind StatusKind, icon StatusIcon) (*Status, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: status name is required", apperrs.ErrInvalid)
	}
	kind = StatusKind(strings.TrimSpace(string(kind)))
	if !kind.Valid() {
		return nil, fmt.Errorf("%w: status kind must be one of %s", apperrs.ErrInvalid, strings.Join(statusKindNames(), ", "))
	}
	icon = StatusIcon(strings.TrimSpace(string(icon)))
	if icon != "" && !validStatusIcons[icon] {
		return nil, fmt.Errorf("%w: status icon must be one of the suggested icons", apperrs.ErrInvalid)
	}
	current, err := s.statuses.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("rename status %s: %w", id, err)
	}
	if err := s.requireOn(ctx, current.ProjectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	updated := *current
	updated.Name = name
	updated.Kind = kind
	updated.Icon = icon
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicStatusUpdated, Payload: StatusEvent{Status: updated}}
	if err := s.statuses.Update(ctx, &updated, evt); err != nil {
		return nil, fmt.Errorf("rename status %s: %w", id, err)
	}
	return &updated, nil
}

// ReorderStatuses sets a project's status column order (projects:write); every status must appear exactly once.
func (s *Service) ReorderStatuses(ctx context.Context, userID, projectID string, ids []string) error {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	statuses, err := s.statuses.ListByProject(ctx, projectID)
	if err != nil {
		return fmt.Errorf("list statuses for project %s: %w", projectID, err)
	}
	existingIDs := make([]string, len(statuses))
	for i, st := range statuses {
		existingIDs[i] = st.ID
	}
	if err := validateReorderIDs(ids, existingIDs, "status"); err != nil {
		return err
	}
	return s.statuses.Reorder(ctx, projectID, ids)
}

// DeleteStatus removes a status column (projects:delete); refused while tickets still use it, so none is left orphaned.
func (s *Service) DeleteStatus(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: status id is required", apperrs.ErrInvalid)
	}
	current, err := s.statuses.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("delete status %s: %w", id, err)
	}
	if err := s.requireOn(ctx, current.ProjectID, permissions.ProjectsDelete); err != nil {
		return err
	}
	count, err := s.statuses.CountTickets(ctx, id)
	if err != nil {
		return fmt.Errorf("count tickets in status %s: %w", id, err)
	}
	if count > 0 {
		return fmt.Errorf("%w: status %s holds %d ticket(s); move them to another column first", apperrs.ErrConflict, id, count)
	}
	if err := s.statuses.Delete(ctx, id, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicStatusDeleted, Payload: StatusEvent{Status: Status{ID: id}}}); err != nil {
		return fmt.Errorf("delete status %s: %w", id, err)
	}
	return nil
}

// NotificationService is the notifications use-case layer (ADR 0019): the inbox plus v1's fan-out generation rules.
type NotificationService struct {
	repo     NotificationRepo
	users    UserStore
	members  WorkspaceMemberStore
	access   PermissionChecker
	projects ProjectReader
	tickets  TicketProjects
	watchers DocWatchers
	now      func() time.Time
}

// SetTicketProjects wires the ticket lookup a ticket notice whose event names no project is checked through.
func (s *NotificationService) SetTicketProjects(t TicketProjects) { s.tickets = t }

// NewNotificationService wires the notification use-cases; delivery goes through the outbox, not a direct
// publish. members and access back memory.updated's permission-gated fan-out to a workspace's members, and
// projects resolves a ticket's or doc's workspace.
func NewNotificationService(repo NotificationRepo, users UserStore, members WorkspaceMemberStore, access PermissionChecker, projects ProjectReader) *NotificationService {
	return &NotificationService{repo: repo, users: users, members: members, access: access, projects: projects, now: time.Now}
}

// WithDocWatchers sets who a doc's changes notify; without it a doc save notifies only the people it mentions.
func (s *NotificationService) WithDocWatchers(w DocWatchers) *NotificationService {
	s.watchers = w
	return s
}

// List returns a user's notifications in one workspace (every workspace when workspaceID is empty), newest first.
func (s *NotificationService) List(ctx context.Context, userID, workspaceID string, limit int) ([]*Notification, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if limit < 1 {
		limit = 50
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID != "" {
		if err := s.requireMember(ctx, userID, workspaceID); err != nil {
			return nil, err
		}
	}
	ns, err := s.repo.List(ctx, userID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	ns, err = permissions.Filter(ns, func(n *Notification) string { return n.WorkspaceID }, func(workspaceID string) error {
		if workspaceID == "" {
			return nil
		}
		return s.requireMember(ctx, userID, workspaceID)
	})
	if err != nil {
		return nil, err
	}
	return permissions.Filter(ns, func(n *Notification) string { return n.ProjectID }, func(projectID string) error {
		if s.opensProject(ctx, userID, projectID) {
			return nil
		}
		return apperrs.ErrNotFound
	})
}

// opensProject keeps a notice about a project its reader can no longer open out of their inbox; the row stays, so
// access given back brings it back.
func (s *NotificationService) opensProject(ctx context.Context, userID, projectID string) bool {
	if projectID == "" || s.access == nil {
		return true
	}
	return s.access.CanInProject(ctx, userID, projectID, permissions.Member)
}

// UnreadCount returns how many of the user's notifications in one workspace (or every workspace) are unread.
func (s *NotificationService) UnreadCount(ctx context.Context, userID, workspaceID string) (int, error) {
	byWorkspace, err := s.UnreadByWorkspace(ctx, userID, workspaceID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range byWorkspace {
		n += c
	}
	return n, nil
}

// UnreadByWorkspace returns the user's unread notification count per workspace that has any, the switcher's badges.
func (s *NotificationService) UnreadByWorkspace(ctx context.Context, userID, workspaceID string) (map[string]int, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	groups, err := s.repo.UnreadByProject(ctx, userID, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("unread count: %w", err)
	}
	out := map[string]int{}
	for _, g := range groups {
		if s.opensProject(ctx, userID, g.ProjectID) {
			out[g.WorkspaceID] += g.Unread
		}
	}
	return out, nil
}

// MarkRead marks one of the user's notifications as read.
func (s *NotificationService) MarkRead(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: notification id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.MarkRead(ctx, userID, id, s.now().UTC()); err != nil {
		return fmt.Errorf("mark notification %s read: %w", id, err)
	}
	return nil
}

// MarkAllRead marks every notification of the user in one workspace (or every workspace) as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID, workspaceID string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.MarkAllRead(ctx, userID, strings.TrimSpace(workspaceID), s.now().UTC()); err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	return nil
}

// Retention keeps the inbox bounded, since nothing else ever deletes a notification (ADR 0100).
const (
	readNotificationRetention   = 90 * 24 * time.Hour
	notificationRetention       = 180 * 24 * time.Hour
	notificationCleanupDelay    = time.Minute
	notificationCleanupInterval = 24 * time.Hour
)

// CleanupExpired applies the retention rule (ADR 0100), unchecked because it runs from a background loop, not a request.
func (s *NotificationService) CleanupExpired(ctx context.Context) (read, old int64, err error) {
	now := s.now()
	read, old, err = s.repo.DeleteExpired(ctx, now.Add(-readNotificationRetention), now.Add(-notificationRetention))
	if err != nil {
		return 0, 0, fmt.Errorf("clean up expired notifications: %w", err)
	}
	return read, old, nil
}

// RunCleanupLoop runs CleanupExpired a minute after start and daily after that, until ctx is cancelled.
func (s *NotificationService) RunCleanupLoop(ctx context.Context, log *slog.Logger) {
	timer := time.NewTimer(notificationCleanupDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		timer.Reset(notificationCleanupInterval)
		read, old, err := s.CleanupExpired(ctx)
		if err != nil {
			log.Warn("notification retention cleanup failed", "error", err)
			continue
		}
		if read+old > 0 {
			log.Info("notification retention cleanup", "read_deleted", read, "old_deleted", old)
		}
	}
}

// requireMember keeps a person's inbox to the workspaces they still belong to; one they left reads as not found.
func (s *NotificationService) requireMember(ctx context.Context, userID, workspaceID string) error {
	if s.members == nil {
		return nil
	}
	ids, err := s.members.ListMemberUserIDs(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("list members of workspace %s: %w", workspaceID, err)
	}
	if !slices.Contains(ids, userID) {
		return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
	}
	return nil
}

// canOpen keeps a notice from naming a ticket or doc its recipient may not read.
func (s *NotificationService) canOpen(ctx context.Context, n notice, userID string) bool {
	if s.access == nil {
		return true
	}
	switch n.subjectType {
	case SubjectTicket:
		if projectID := s.ticketProject(ctx, n); projectID != "" {
			return s.access.CanInProject(ctx, userID, projectID, permissions.TicketsRead)
		}
		return s.access.HasPermission(ctx, userID, n.workspaceID, permissions.TicketsRead)
	case SubjectDoc:
		return s.access.CanReadDoc(ctx, userID, n.subjectID)
	}
	return true
}

// ticketProject is the project a ticket notice is about: named by its event, else read from the ticket itself.
func (s *NotificationService) ticketProject(ctx context.Context, n notice) string {
	if n.projectID != "" || s.tickets == nil {
		return n.projectID
	}
	projectID, err := s.tickets.ProjectOfTicket(ctx, n.subjectID)
	if err != nil {
		return ""
	}
	return projectID
}

// onTicketCreated fans out ticket.assigned to the developer and tester and ticket.mentioned to every @-mentioned user.
func (s *NotificationService) onTicketCreated(ctx context.Context, t ticketRef, mentionedIDs []string) error {
	actorID, err := s.userIDForLogin(ctx, t.Reporter.Login)
	if err != nil {
		return err
	}
	n, err := s.notice(ctx, SubjectTicket, t.ID, t.Title, t.ProjectID, actorID)
	if err != nil {
		return err
	}
	mentionTitle := s.mentionTitle(ctx, n)
	recipients := []recipient{{Login: t.Developer, Kind: KindTicketAssigned}, {Login: t.Tester, Kind: KindTicketAssigned}}
	for _, login := range extractMentions(t.Title + " " + t.Body) {
		recipients = append(recipients, recipient{Login: login, Kind: KindTicketMentioned, Title: mentionTitle})
	}
	for _, id := range mentionedIDs {
		recipients = append(recipients, recipient{UserID: id, Kind: KindTicketMentioned, Title: mentionTitle})
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onTicketUpdated fans out ticket.mentioned to the people an edit newly @-mentions.
func (s *NotificationService) onTicketUpdated(ctx context.Context, e ticketUpdatedEvent) error {
	if len(e.MentionedUserIDs) == 0 {
		return nil
	}
	n, err := s.notice(ctx, SubjectTicket, e.Ticket.ID, e.Ticket.Title, e.Ticket.ProjectID, e.ActorID)
	if err != nil {
		return err
	}
	mentionTitle := s.mentionTitle(ctx, n)
	recipients := make([]recipient, 0, len(e.MentionedUserIDs))
	for _, id := range e.MentionedUserIDs {
		recipients = append(recipients, recipient{UserID: id, Kind: KindTicketMentioned, Title: mentionTitle})
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onTicketStatusChanged fans out ticket.status_changed to the developer, tester, and @-mentioned users of the ticket.
func (s *NotificationService) onTicketStatusChanged(ctx context.Context, t ticketRef, actorID string) error {
	recipients := []recipient{{Login: t.Developer, Kind: KindTicketStatus}, {Login: t.Tester, Kind: KindTicketStatus}}
	for _, login := range extractMentions(t.Title + " " + t.Body) {
		if strings.EqualFold(login, t.Developer) || strings.EqualFold(login, t.Tester) {
			continue
		}
		recipients = append(recipients, recipient{Login: login, Kind: KindTicketStatus})
	}
	n, err := s.notice(ctx, SubjectTicket, t.ID, t.Title, t.ProjectID, actorID)
	if err != nil {
		return err
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onDocActivity tells the doc's watchers but the actor, and newly mentioned members with doc.mentioned (ADR 0101).
func (s *NotificationService) onDocActivity(ctx context.Context, e docEvent, kind Kind) error {
	n, err := s.notice(ctx, SubjectDoc, e.Doc.ID, e.Doc.Title, e.Doc.ProjectID, e.ActorID)
	if err != nil {
		return err
	}
	if n.workspaceID == "" {
		return nil
	}
	recipients, err := s.docMentionRecipients(ctx, n, e.MentionedUserIDs)
	if err != nil {
		return err
	}
	if s.watchers != nil {
		ids, err := s.watchers.ListDocWatcherIDs(ctx, e.Doc.ID)
		if err != nil {
			return fmt.Errorf("list watchers of doc %s: %w", e.Doc.ID, err)
		}
		for _, id := range ids {
			recipients = append(recipients, recipient{UserID: id, Kind: kind})
		}
	}
	return s.fanOut(ctx, evtKey(ctx), n, recipients)
}

// onDocQuestionsPosted tells the doc's watchers but the round's starter; the starter's own post is no news to them.
func (s *NotificationService) onDocQuestionsPosted(ctx context.Context, e docQuestionsPostedEvent) error {
	if e.NoGaps || e.QuestionCount <= 0 || s.watchers == nil {
		return nil
	}
	n, err := s.notice(ctx, SubjectDoc, e.Doc.ID, "New questions on "+e.Doc.Title, e.Doc.ProjectID, e.StartedBy)
	if err != nil || n.workspaceID == "" {
		return err
	}
	ids, err := s.watchers.ListDocWatcherIDs(ctx, e.Doc.ID)
	if err != nil {
		return fmt.Errorf("list watchers of doc %s: %w", e.Doc.ID, err)
	}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, KindDocQuestionsAsked, ids)
}

// onDocRoundAnswered tells the round's starter, unless the answer that finished it was their own.
func (s *NotificationService) onDocRoundAnswered(ctx context.Context, e docRoundAnsweredEvent) error {
	n, err := s.notice(ctx, SubjectDoc, e.Doc.ID, "Questions answered on "+e.Doc.Title, e.Doc.ProjectID, e.ActorID)
	if err != nil || n.workspaceID == "" {
		return err
	}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, KindDocQuestionsAnswered, []string{e.StartedBy})
}

// docMentionRecipients keeps the mentioned people who are members of the doc's workspace.
func (s *NotificationService) docMentionRecipients(ctx context.Context, n notice, mentioned []string) ([]recipient, error) {
	if len(mentioned) == 0 || s.members == nil {
		return nil, nil
	}
	members, err := s.members.ListMemberUserIDs(ctx, n.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members for doc mentions: %w", err)
	}
	mentionTitle := s.mentionTitle(ctx, n)
	var out []recipient
	for _, id := range mentioned {
		if slices.Contains(members, id) {
			out = append(out, recipient{UserID: id, Kind: KindDocMentioned, Title: mentionTitle})
		}
	}
	return out, nil
}

// mentionTitle words a mention as "<author> mentioned you in <title>"; with no known author it stays the title.
func (s *NotificationService) mentionTitle(ctx context.Context, n notice) string {
	if n.actorID == "" {
		return n.subjectTitle
	}
	name, err := s.users.NameForUserID(ctx, n.actorID)
	if err != nil || name == "" {
		return n.subjectTitle
	}
	return fmt.Sprintf("%s mentioned you in %s", name, n.subjectTitle)
}

// onMemoryUpdated fans out to every member of the memory's workspace who may read memories in its project, excluding
// the author; membership and Project access decide, not every registered user.
func (s *NotificationService) onMemoryUpdated(ctx context.Context, m memoryRef, authorID, authorVia string) error {
	if s.members == nil || s.access == nil {
		return nil
	}
	userIDs, err := s.members.ListMemberUserIDs(ctx, m.WorkspaceID)
	if err != nil {
		return fmt.Errorf("list members for workspace %s: %w", m.WorkspaceID, err)
	}
	authorLabel := authorID
	if login, err := s.users.LoginForUserID(ctx, authorID); err == nil && login != "" {
		authorLabel = login
	}
	if authorVia == "mcp" {
		authorLabel = "Agent via " + authorLabel
	}
	subjectTitle := fmt.Sprintf("%s — v%d by %s", m.Title, m.Version, authorLabel)
	var recipients []string
	for _, uid := range userIDs {
		if !s.access.CanInProject(ctx, uid, m.ProjectID, permissions.MemoriesRead) {
			continue
		}
		recipients = append(recipients, uid)
	}
	n := notice{subjectType: SubjectMemory, subjectID: m.ID, subjectTitle: subjectTitle, workspaceID: m.WorkspaceID, actorID: authorID}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, KindMemoryUpdated, recipients)
}

// onPlayRunFinished tells the starter how their run ended; the subject is the target so the inbox opens it.
func (s *NotificationService) onPlayRunFinished(ctx context.Context, e playRunFinishedEvent) error {
	outcome := "finished"
	if e.Outcome == "failed" || e.Outcome == "interrupted" {
		outcome = e.Outcome
	}
	return s.notifyPlayStarter(ctx, e, outcome, KindPlayRunFinished)
}

// onPlayRunWaiting tells the starter their run needs an answer before it can go on.
func (s *NotificationService) onPlayRunWaiting(ctx context.Context, e playRunFinishedEvent) error {
	return s.notifyPlayStarter(ctx, e, "needs your answer", KindPlayRunWaiting)
}

// notifyPlayStarter has no actor to exclude: the run, not the starter, is what finished or asked.
func (s *NotificationService) notifyPlayStarter(ctx context.Context, e playRunFinishedEvent, outcome string, kind Kind) error {
	subjectType := SubjectTicket
	if e.TargetType == string(SubjectDoc) {
		subjectType = SubjectDoc
	}
	title := e.TargetTitle
	if title == "" {
		title = e.TargetID
	}
	n := notice{subjectType: subjectType, subjectID: e.TargetID, subjectTitle: fmt.Sprintf("%s %s on %s", e.PlayLabel, outcome, title), workspaceID: e.WorkspaceID}
	return s.fanOutByUserID(ctx, evtKey(ctx), n, kind, []string{e.StarterID})
}

// notice is what one source event is about: its subject, the subject's workspace, and whose action it was.
type notice struct {
	subjectType  SubjectType
	subjectID    string
	subjectTitle string
	workspaceID  string
	projectID    string
	actorID      string
}

// notice resolves the subject's project to its workspace; a project deleted since the event leaves it unscoped.
func (s *NotificationService) notice(ctx context.Context, subjectType SubjectType, subjectID, subjectTitle, projectID, actorID string) (notice, error) {
	n := notice{subjectType: subjectType, subjectID: subjectID, subjectTitle: subjectTitle, projectID: projectID, actorID: actorID}
	if s.projects == nil || strings.TrimSpace(projectID) == "" {
		return n, nil
	}
	p, err := s.projects.Get(ctx, projectID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return n, nil
	}
	if err != nil {
		return n, fmt.Errorf("resolve workspace of project %s: %w", projectID, err)
	}
	n.workspaceID = p.WorkspaceID
	return n, nil
}

// userIDForLogin resolves an event's actor login; an unknown login falls back to itself, since the tickets domain
// records the raw user id when its own lookup failed.
func (s *NotificationService) userIDForLogin(ctx context.Context, login string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return "", nil
	}
	u, err := s.users.GetUserByLogin(ctx, login)
	if errors.Is(err, apperrs.ErrNotFound) || (err == nil && u == nil) {
		return login, nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve actor %s: %w", login, err)
	}
	return u.ID, nil
}

// recipient is one pending fan-out row: who by login or user id, the kind to create, and a title replacing the subject's.
type recipient struct {
	Login  string
	UserID string
	Kind   Kind
	Title  string
}

// fanOut creates one notification per known recipient in a single outbox transaction, idempotent per event; a
// person named twice keeps their first row.
func (s *NotificationService) fanOut(ctx context.Context, key string, n notice, recipients []recipient) error {
	now := s.now().UTC()
	seen := map[string]bool{}
	var toCreate []*Notification
	for _, r := range recipients {
		userID, err := s.recipientID(ctx, r)
		if err != nil {
			return err
		}
		if userID == "" || seen[userID] {
			continue
		}
		seen[userID] = true
		row := n.row(key, userID, r.Kind, now)
		if r.Title != "" {
			row.SubjectTitle = r.Title
		}
		toCreate = append(toCreate, row)
	}
	return s.create(ctx, n, toCreate)
}

// recipientID resolves a recipient to a user id; an unknown login is nobody to notify.
func (s *NotificationService) recipientID(ctx context.Context, r recipient) (string, error) {
	if id := strings.TrimSpace(r.UserID); id != "" {
		return id, nil
	}
	login := strings.TrimSpace(r.Login)
	if login == "" {
		return "", nil
	}
	user, err := s.users.GetUserByLogin(ctx, login)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve recipient %s: %w", login, err)
	}
	if user == nil {
		return "", nil
	}
	return strings.TrimSpace(user.ID), nil
}

// fanOutByUserID creates one notification per already-resolved user id, skipping fanOut's login lookup for
// recipient lists that came from a workspace-membership scan rather than a login.
func (s *NotificationService) fanOutByUserID(ctx context.Context, key string, n notice, kind Kind, userIDs []string) error {
	now := s.now().UTC()
	var toCreate []*Notification
	for _, uid := range userIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" {
			continue
		}
		toCreate = append(toCreate, n.row(key, uid, kind, now))
	}
	return s.create(ctx, n, toCreate)
}

func (n notice) row(key, userID string, kind Kind, now time.Time) *Notification {
	return &Notification{
		ID:           key + ":" + userID,
		UserID:       userID,
		WorkspaceID:  n.workspaceID,
		Kind:         kind,
		SubjectType:  n.subjectType,
		SubjectID:    n.subjectID,
		SubjectTitle: n.subjectTitle,
		CreatedAt:    now,
	}
}

// create writes the rows with both outbox events; every fan-out passes through here, so the actor is dropped once for all kinds.
func (s *NotificationService) create(ctx context.Context, n notice, toCreate []*Notification) error {
	toCreate = slices.DeleteFunc(toCreate, func(row *Notification) bool {
		return row.UserID == n.actorID || !s.canOpen(ctx, n, row.UserID)
	})
	if len(toCreate) == 0 {
		return nil
	}
	items := make([]NotificationPushItem, 0, len(toCreate))
	for _, row := range toCreate {
		items = append(items, NotificationPushItem{ID: row.ID, UserID: row.UserID, WorkspaceID: row.WorkspaceID})
	}
	created := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicNotificationCreated, Payload: NotificationCreatedEvent{}}
	push := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicNotificationPushRequested, Payload: NotificationPushRequestedEvent{Notifications: items}}
	if err := s.repo.CreateMany(ctx, toCreate, created, push); err != nil {
		return fmt.Errorf("create notifications: %w", err)
	}
	return nil
}

// evtKey returns a stable per-event key for idempotent fan-out, injected by the composition root via the context.
func evtKey(ctx context.Context) string {
	if k, ok := ctx.Value(eventKeyCtx{}).(string); ok && k != "" {
		return k
	}
	return ids.New()
}

type eventKeyCtx struct{}

// CtxWithEventKey carries the source event ID the fan-out uses to derive idempotent notification IDs.
func CtxWithEventKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, eventKeyCtx{}, key)
}
