package tenancy

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Archive hides workspace id from its members until the instance Owner restores it; needs workspaces:delete there, and never the default workspace.
func (s *Service) Archive(ctx context.Context, actorID, id string) (*Workspace, error) {
	id = strings.TrimSpace(id)
	if id == DefaultWorkspaceID {
		return nil, fmt.Errorf("%w: the instance's own workspace can't be archived; its Owner is the instance Owner who restores the others", apperrs.ErrInvalid)
	}
	if err := s.requireWorkspacePermission(ctx, actorID, id, permissions.WorkspacesDelete); err != nil {
		return nil, err
	}
	return s.setArchived(ctx, actorID, id, true)
}

// Restore brings an archived workspace back for everyone in it; needs instance:write, the instance Owner's.
func (s *Service) Restore(ctx context.Context, actorID, id string) (*Workspace, error) {
	if err := s.requireInstanceWrite(ctx, actorID); err != nil {
		return nil, err
	}
	return s.setArchived(ctx, actorID, strings.TrimSpace(id), false)
}

// ListArchived is the archived workspaces, newest archive first, for the instance Owner who restores them.
func (s *Service) ListArchived(ctx context.Context, actorID string) ([]*Workspace, error) {
	if err := s.requireInstanceWrite(ctx, actorID); err != nil {
		return nil, err
	}
	ws, err := s.repo.ListArchived(ctx)
	if err != nil {
		return nil, fmt.Errorf("list archived workspaces: %w", err)
	}
	return ws, s.withChips(ctx, ws...)
}

func (s *Service) requireInstanceWrite(ctx context.Context, actorID string) error {
	ok, err := s.perm.HoldsAnywhere(ctx, actorID, permissions.InstanceWrite)
	if err != nil {
		return fmt.Errorf("check %s: %w", permissions.InstanceWrite, err)
	}
	if !ok {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.InstanceWrite)
	}
	return nil
}

// setArchived flips the state and names the members in the event, since an archived workspace resolves none; a repeat is a no-op.
func (s *Service) setArchived(ctx context.Context, actorID, id string, archive bool) (*Workspace, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if (current.ArchivedAt != nil) == archive {
		return current, nil
	}
	members, err := s.members.ListByWorkspace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list members of workspace %s: %w", id, err)
	}
	userIDs := make([]string, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	now := s.now().UTC()
	topic, at := TopicWorkspaceRestored, (*time.Time)(nil)
	if archive {
		topic, at = TopicWorkspaceArchived, &now
	}
	event := eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: WorkspaceArchiveEvent{WorkspaceID: id, Name: current.Name, Slug: current.Slug, ActorID: actorID, UserIDs: userIDs}}
	if err := s.repo.SetArchived(ctx, id, at, now, event); err != nil {
		return nil, fmt.Errorf("set archived on workspace %s: %w", id, err)
	}
	updated := *current
	updated.ArchivedAt, updated.UpdatedAt = at, now
	return &updated, nil
}
