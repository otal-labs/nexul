// Package workspace implements projects, repositories, ticket moves, and the notifications inbox (ADR 0019).
package workspace

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// projectPrefixPattern matches the immutable 2-5 letter project prefix (ADR 0004), checked after uppercasing.
var projectPrefixPattern = regexp.MustCompile(`^[A-Z]{2,5}$`)

// Service is the workspace use-case layer (ADR 0019): projects, categories, ticket types, and status columns.
type Service struct {
	repo       Repo
	cats       CategoryRepo
	types      TicketTypeRepo
	statuses   StatusRepo
	admin      InstanceAdminGate
	workspaces WorkspaceGate
	now        func() time.Time
}

// NewService wires the workspace use-cases over the given repos and gates.
func NewService(repo Repo, cats CategoryRepo, types TicketTypeRepo, statuses StatusRepo, admin InstanceAdminGate, workspaces WorkspaceGate) *Service {
	return &Service{repo: repo, cats: cats, types: types, statuses: statuses, admin: admin, workspaces: workspaces, now: time.Now}
}

// Create adds a project to a workspace with the next display position (owner only).
func (s *Service) Create(ctx context.Context, userID, workspaceID, name, prefix string, icon ProjectIcon) (*Project, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
		return nil, err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required — create a workspace before creating a project", apperrs.ErrInvalid)
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
		return nil, fmt.Errorf("%w: project prefix must be 2-5 letters", apperrs.ErrInvalid)
	}
	icon = ProjectIcon(strings.TrimSpace(string(icon)))
	if icon != "" && !validProjectIcons[icon] {
		return nil, fmt.Errorf("%w: project icon must be one of the suggested icons", apperrs.ErrInvalid)
	}
	projects, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	for _, p := range projects {
		if p.Prefix == prefix {
			return nil, fmt.Errorf("%w: project prefix %q is already in use", apperrs.ErrInvalid, prefix)
		}
	}
	position := len(projects)
	for _, p := range projects {
		if p.Position >= position {
			position = p.Position + 1
		}
	}
	now := s.now().UTC()
	p := &Project{ID: ids.New(), Name: name, Prefix: prefix, Position: position, WorkspaceID: workspaceID, Icon: icon, CreatedAt: now, UpdatedAt: now}
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
	return p, nil
}

// List returns a workspace's projects ordered by position.
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Project, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	projects, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

// Rename updates a project's name and/or icon (owner only); icon nil keeps current, "" clears it.
func (s *Service) Rename(ctx context.Context, userID, id, name string, icon *ProjectIcon) (*Project, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
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

// SetPrefix is owner-only and refused if the project already has one.
func (s *Service) SetPrefix(ctx context.Context, userID, id, prefix string) (*Project, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
		return nil, err
	}
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	if !projectPrefixPattern.MatchString(prefix) {
		return nil, fmt.Errorf("%w: project prefix must be 2-5 letters", apperrs.ErrInvalid)
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

// Reorder sets a workspace's project display order (owner only); every project must appear exactly once.
func (s *Service) Reorder(ctx context.Context, userID, workspaceID string, ids []string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
		return err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
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
	return DeleteImpact{Tickets: tickets, Repos: repos, Services: services}, nil
}

// Delete removes a project (owner only); refused while it has tickets, repos, or services, to avoid orphans.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
		return err
	}
	impact, err := s.DeleteImpact(ctx, id)
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

// AddRepo associates a repository with a project (owner only); an already-owned repo conflicts.
func (s *Service) AddRepo(ctx context.Context, userID, projectID, owner, name, connectorID string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
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
	if _, err := s.repo.Get(ctx, projectID); err != nil {
		return fmt.Errorf("add repo to project %s: %w", projectID, err)
	}
	if err := s.repo.AddRepo(ctx, projectID, RepoRef{Owner: owner, Name: name, FullName: owner + "/" + name, ConnectorID: connectorID}); err != nil {
		return fmt.Errorf("add repo %s/%s: %w", owner, name, err)
	}
	return nil
}

// RemoveRepo dissociates a repository from its project (owner only).
func (s *Service) RemoveRepo(ctx context.Context, userID, owner, name string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
		return err
	}
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" || name == "" {
		return fmt.Errorf("%w: repo owner and name are required", apperrs.ErrInvalid)
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
	if err := s.repo.MoveTicket(ctx, ticketID, projectID); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	return nil
}

// requireOwner enforces the instance-admin permission bit; an empty userID means a trusted adapter.
func (s *Service) requireOwner(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	ok, err := s.admin.CanCreateWorkspace(ctx, userID)
	if err != nil {
		return fmt.Errorf("check instance admin: %w", err)
	}
	if !ok {
		return fmt.Errorf("%w: owner role required", apperrs.ErrForbidden)
	}
	return nil
}

// CreateCategory adds a category to a project with the next display position (owner only).
func (s *Service) CreateCategory(ctx context.Context, userID, projectID, name string, color colors.Color) (*Category, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
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
	return c, nil
}

// ListCategories returns all categories ordered by project, position.
func (s *Service) ListCategories(ctx context.Context) ([]*Category, error) {
	cats, err := s.cats.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return cats, nil
}

// ListCategoriesByProject returns a project's categories ordered by position.
func (s *Service) ListCategoriesByProject(ctx context.Context, projectID string) ([]*Category, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	cats, err := s.cats.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list categories for project %s: %w", projectID, err)
	}
	return cats, nil
}

// RenameCategory updates a category's name and/or color (owner only); its identity never changes.
func (s *Service) RenameCategory(ctx context.Context, userID, id, name string, color colors.Color) (*Category, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
		return nil, err
	}
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

// ReorderCategories sets a project's category display order (owner only); every category appears exactly once.
func (s *Service) ReorderCategories(ctx context.Context, userID, projectID string, ids []string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
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

// DeleteCategory removes a category (owner only); its tickets become uncategorized rather than being deleted.
func (s *Service) DeleteCategory(ctx context.Context, userID, id string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: category id is required", apperrs.ErrInvalid)
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
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicTicketCategoryChanged, Payload: TicketCategoryChangedEvent{TicketID: ticketID, CategoryID: strings.TrimSpace(categoryID)}}
	if err := s.cats.SetTicketCategory(ctx, ticketID, strings.TrimSpace(categoryID), evt); err != nil {
		return fmt.Errorf("move ticket %s: %w", ticketID, err)
	}
	return nil
}

// CreateTicketType adds a project ticket type with the next display position (owner only).
func (s *Service) CreateTicketType(ctx context.Context, userID, projectID, name string, color colors.Color) (*TicketType, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
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
	return tt, nil
}

// ListTicketTypesByProject returns a project's ticket types ordered by position; there is no unscoped list.
func (s *Service) ListTicketTypesByProject(ctx context.Context, projectID string) ([]*TicketType, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	types, err := s.types.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list ticket types for project %s: %w", projectID, err)
	}
	return types, nil
}

// RenameTicketType updates a ticket type's name and/or color (owner only); its identity never changes.
func (s *Service) RenameTicketType(ctx context.Context, userID, id, name string, color colors.Color) (*TicketType, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
		return nil, err
	}
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

// SetTicketTypeTemplate replaces a type's body template (owner only); existing tickets keep the body they were born with.
func (s *Service) SetTicketTypeTemplate(ctx context.Context, userID, id, template string) (*TicketType, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
		return nil, err
	}
	current, err := s.types.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("set ticket type template %s: %w", id, err)
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

// ReorderTicketTypes sets a project's ticket type display order (owner only); every type appears exactly once.
func (s *Service) ReorderTicketTypes(ctx context.Context, userID, projectID string, ids []string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
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

// DeleteTicketType removes a ticket type (owner only); refused while tickets still use it, so none is left orphaned.
func (s *Service) DeleteTicketType(ctx context.Context, userID, id string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: ticket type id is required", apperrs.ErrInvalid)
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

// CreateStatus adds a project status column with the next display position (owner only).
func (s *Service) CreateStatus(ctx context.Context, userID, projectID, name string, kind StatusKind, icon StatusIcon) (*Status, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
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
	return st, nil
}

// ListStatusesByProject returns a project's status columns ordered by position; there is no unscoped list.
func (s *Service) ListStatusesByProject(ctx context.Context, projectID string) ([]*Status, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	statuses, err := s.statuses.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list statuses for project %s: %w", projectID, err)
	}
	return statuses, nil
}

// RenameStatus updates a status column's name, kind, and icon (owner only); existing tickets stay on it.
func (s *Service) RenameStatus(ctx context.Context, userID, id, name string, kind StatusKind, icon StatusIcon) (*Status, error) {
	if err := s.requireOwner(ctx, userID); err != nil {
		return nil, err
	}
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

// ReorderStatuses sets a project's status column order (owner only, ticket 03); every status must appear exactly once.
func (s *Service) ReorderStatuses(ctx context.Context, userID, projectID string, ids []string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
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

// DeleteStatus removes a status column (owner only); refused while tickets still use it, so none is left orphaned.
func (s *Service) DeleteStatus(ctx context.Context, userID, id string) error {
	if err := s.requireOwner(ctx, userID); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: status id is required", apperrs.ErrInvalid)
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
	repo    NotificationRepo
	users   UserStore
	members WorkspaceMemberStore
	access  PermissionChecker
	now     func() time.Time
}

// NewNotificationService wires the notification use-cases; delivery goes through the outbox, not a direct
// publish. members and access back the workspace-scoped, permission-gated fan-out memory.updated needs.
func NewNotificationService(repo NotificationRepo, users UserStore, members WorkspaceMemberStore, access PermissionChecker) *NotificationService {
	return &NotificationService{repo: repo, users: users, members: members, access: access, now: time.Now}
}

// List returns a user's notifications, newest first, limited to limit rows.
func (s *NotificationService) List(ctx context.Context, userID string, limit int) ([]*Notification, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if limit < 1 {
		limit = 50
	}
	ns, err := s.repo.List(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return ns, nil
}

// UnreadCount returns how many of the user's notifications are unread.
func (s *NotificationService) UnreadCount(ctx context.Context, userID string) (int, error) {
	if strings.TrimSpace(userID) == "" {
		return 0, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	n, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("unread count: %w", err)
	}
	return n, nil
}

// MarkRead marks one of the user's notifications as read.
func (s *NotificationService) MarkRead(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: notification id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		return fmt.Errorf("mark notification %s read: %w", id, err)
	}
	return nil
}

// MarkAllRead marks every notification of the user as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.MarkAllRead(ctx, userID); err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	return nil
}

// onTicketCreated fans out ticket.assigned to the developer and tester and ticket.mentioned to every @-mentioned user.
func (s *NotificationService) onTicketCreated(ctx context.Context, t ticketRef) error {
	recipients := []recipient{{Login: t.Developer, Kind: KindTicketAssigned}, {Login: t.Tester, Kind: KindTicketAssigned}}
	for _, login := range extractMentions(t.Title + " " + t.Body) {
		recipients = append(recipients, recipient{Login: login, Kind: KindTicketMentioned})
	}
	return s.fanOut(ctx, evtKey(ctx), SubjectTicket, t.ID, t.Title, recipients)
}

// onTicketStatusChanged fans out ticket.status_changed to the developer, tester, and @-mentioned users of the ticket.
func (s *NotificationService) onTicketStatusChanged(ctx context.Context, t ticketRef) error {
	recipients := []recipient{{Login: t.Developer, Kind: KindTicketStatus}, {Login: t.Tester, Kind: KindTicketStatus}}
	for _, login := range extractMentions(t.Title + " " + t.Body) {
		if strings.EqualFold(login, t.Developer) || strings.EqualFold(login, t.Tester) {
			continue
		}
		recipients = append(recipients, recipient{Login: login, Kind: KindTicketStatus})
	}
	return s.fanOut(ctx, evtKey(ctx), SubjectTicket, t.ID, t.Title, recipients)
}

// onDocActivity fans out to every user who can access the doc; v1 access is workspace-level, so every member.
func (s *NotificationService) onDocActivity(ctx context.Context, d docRef, kind Kind) error {
	users, err := s.users.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users for doc fan-out: %w", err)
	}
	recipients := make([]recipient, 0, len(users))
	for _, u := range users {
		recipients = append(recipients, recipient{Login: u.Login, Kind: kind})
	}
	return s.fanOut(ctx, evtKey(ctx), SubjectDoc, d.ID, d.Title, recipients)
}

// onMemoryUpdated fans out to every member of the memory's workspace who holds memories:read, excluding the
// author (ticket 17); unlike onDocActivity, memories are genuinely workspace-scoped, so this filters by
// membership and the permission bit rather than every registered user.
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
		if uid == authorID {
			continue
		}
		if !s.access.HasPermission(ctx, uid, m.WorkspaceID, permissions.MemoriesRead) {
			continue
		}
		recipients = append(recipients, uid)
	}
	return s.fanOutByUserID(ctx, evtKey(ctx), SubjectMemory, m.ID, subjectTitle, KindMemoryUpdated, recipients)
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

func (s *NotificationService) notifyPlayStarter(ctx context.Context, e playRunFinishedEvent, outcome string, kind Kind) error {
	subjectType := SubjectTicket
	if e.TargetType == string(SubjectDoc) {
		subjectType = SubjectDoc
	}
	title := e.TargetTitle
	if title == "" {
		title = e.TargetID
	}
	subjectTitle := fmt.Sprintf("%s %s on %s", e.PlayLabel, outcome, title)
	return s.fanOutByUserID(ctx, evtKey(ctx), subjectType, e.TargetID, subjectTitle, kind, []string{e.StarterID})
}

// recipient is one pending fan-out row: the recipient login and the notification kind to create.
type recipient struct {
	Login string
	Kind  Kind
}

// fanOut creates one notification per known recipient in a single outbox transaction, idempotent per event.
func (s *NotificationService) fanOut(ctx context.Context, key string, subjectType SubjectType, subjectID, subjectTitle string, recipients []recipient) error {
	now := s.now().UTC()
	seen := map[string]bool{}
	var toCreate []*Notification
	for _, r := range recipients {
		login := strings.TrimSpace(r.Login)
		if login == "" || seen[login] {
			continue
		}
		seen[login] = true
		user, err := s.users.GetUserByLogin(ctx, login)
		if err != nil {
			if errors.Is(err, apperrs.ErrNotFound) {
				continue // not a member — no notification
			}
			return fmt.Errorf("resolve recipient %s: %w", login, err)
		}
		if user == nil || strings.TrimSpace(user.ID) == "" {
			continue
		}
		toCreate = append(toCreate, &Notification{
			ID:           key + ":" + user.ID,
			UserID:       user.ID,
			Kind:         r.Kind,
			SubjectType:  subjectType,
			SubjectID:    subjectID,
			SubjectTitle: subjectTitle,
			Read:         false,
			CreatedAt:    now,
		})
	}
	if len(toCreate) == 0 {
		return nil
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicNotificationCreated, Payload: NotificationCreatedEvent{}}
	if err := s.repo.CreateMany(ctx, toCreate, evt); err != nil {
		return fmt.Errorf("create notifications: %w", err)
	}
	return nil
}

// fanOutByUserID creates one notification per already-resolved user id, skipping fanOut's login lookup for
// recipient lists that came from a workspace-membership scan (memory.updated) rather than a login.
func (s *NotificationService) fanOutByUserID(ctx context.Context, key string, subjectType SubjectType, subjectID, subjectTitle string, kind Kind, userIDs []string) error {
	now := s.now().UTC()
	var toCreate []*Notification
	for _, uid := range userIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" {
			continue
		}
		toCreate = append(toCreate, &Notification{
			ID:           key + ":" + uid,
			UserID:       uid,
			Kind:         kind,
			SubjectType:  subjectType,
			SubjectID:    subjectID,
			SubjectTitle: subjectTitle,
			Read:         false,
			CreatedAt:    now,
		})
	}
	if len(toCreate) == 0 {
		return nil
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicNotificationCreated, Payload: NotificationCreatedEvent{}}
	if err := s.repo.CreateMany(ctx, toCreate, evt); err != nil {
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
