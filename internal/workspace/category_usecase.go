package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// CreateCategory adds a category to a project with the next display position (projects:write).
func (s *Service) CreateCategory(ctx context.Context, projectID, name string, color colors.Color) (*Category, error) {
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
func (s *Service) RenameCategory(ctx context.Context, id, name string, color colors.Color) (*Category, error) {
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
func (s *Service) ReorderCategories(ctx context.Context, projectID string, ids []string) error {
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
func (s *Service) DeleteCategory(ctx context.Context, id string) error {
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
