package automations

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// workspaceEvent is the part of workspace.archived and workspace.restored automations reads, declared consumer-side.
type workspaceEvent struct {
	WorkspaceID string `json:"workspace_id"`
}

// HandleWorkspaceArchived drops live connections, which were handed events and secrets before the archive.
func (s *Service) HandleWorkspaceArchived(ctx context.Context, ev eventbus.Event) error {
	return s.eachInWorkspace(ctx, ev, func(a *Automation) error {
		s.conns.Disconnect(a.ID, "workspace archived")
		return nil
	})
}

// HandleWorkspaceRestored skips what was published while archived, as switching an automation on does.
func (s *Service) HandleWorkspaceRestored(ctx context.Context, ev eventbus.Event) error {
	return s.eachInWorkspace(ctx, ev, func(a *Automation) error {
		if err := s.conns.SkipBacklog(ctx, a.ID); err != nil {
			return apperrs.Retryable(fmt.Errorf("skip automation %s backlog: %w", a.ID, err))
		}
		s.conns.Disconnect(a.ID, "workspace restored")
		return nil
	})
}

func (s *Service) eachInWorkspace(ctx context.Context, ev eventbus.Event, do func(*Automation) error) error {
	if s.conns == nil {
		return nil
	}
	var e workspaceEvent
	if err := json.Unmarshal(ev.Payload, &e); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	all, err := s.repo.List(ctx)
	if err != nil {
		return apperrs.Retryable(fmt.Errorf("list automations: %w", err))
	}
	for i := range all {
		if all[i].WorkspaceID != e.WorkspaceID {
			continue
		}
		if err := do(&all[i]); err != nil {
			return err
		}
	}
	return nil
}
