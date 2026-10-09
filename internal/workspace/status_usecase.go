package workspace

import (
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

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
