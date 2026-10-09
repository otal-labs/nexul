package workspace

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

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
