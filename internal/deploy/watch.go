package deploy

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// StackWatchRepo persists who turned a stack's notices off; everyone else who reads the stack watches it.
type StackWatchRepo interface {
	Unwatching(ctx context.Context, stackID, userID string) (bool, error)
	ListUnwatcherIDs(ctx context.Context, stackID string) ([]string, error)
	// SetWatching starts watching by deleting the caller's opt-out and stops it by writing one.
	SetWatching(ctx context.Context, stackID, userID string, watching bool, at time.Time) error
}

// SetStackWatchRepo wires opt-out storage; unset, every viewer reads as watching and cannot change it.
func (s *Service) SetStackWatchRepo(r StackWatchRepo) { s.watchers = r }

// Watching says whether the caller gets the stack's notices; it needs stacks:read and is true until they opt out.
func (s *Service) Watching(ctx context.Context, stackID string) (bool, error) {
	if strings.TrimSpace(stackID) == "" {
		return false, fmt.Errorf("%w: stack id is required", apperrs.ErrInvalid)
	}
	if _, err := s.loadStack(ctx, stackID, permissions.StacksRead); err != nil {
		return false, fmt.Errorf("get stack %s: %w", stackID, err)
	}
	if s.watchers == nil {
		return true, nil
	}
	userID := watcherID(ctx)
	if userID == "" {
		return false, nil
	}
	out, err := s.watchers.Unwatching(ctx, stackID, userID)
	if err != nil {
		return false, fmt.Errorf("read watching of stack %s: %w", stackID, err)
	}
	return !out, nil
}

// SetWatching turns the caller's notices for a stack on or off, needing only stacks:read, and returns the new state.
func (s *Service) SetWatching(ctx context.Context, stackID string, watching bool) (bool, error) {
	userID := watcherID(ctx)
	if userID == "" {
		return false, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	if s.watchers == nil {
		return false, fmt.Errorf("%w: stack watching is not available", apperrs.ErrInvalid)
	}
	if _, err := s.loadStack(ctx, stackID, permissions.StacksRead); err != nil {
		return false, fmt.Errorf("get stack %s: %w", stackID, err)
	}
	if err := s.watchers.SetWatching(ctx, stackID, userID, watching, s.now().UTC()); err != nil {
		return false, fmt.Errorf("set watching on stack %s: %w", stackID, err)
	}
	return watching, nil
}

func watcherID(ctx context.Context) string {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok {
		return ""
	}
	return actor.ID
}
