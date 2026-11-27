package plays

import (
	"context"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/labels"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// AutoPlayInput is the editable surface of an auto play; conditions and priority are replaced whole.
type AutoPlayInput struct {
	Moment            Moment
	MomentStage       *Stage
	Schedule          *Schedule
	Conditions        Conditions
	Priority          Priority
	OnceWithinMinutes int
	RunOn             RunOn
	// RunAsMe makes the caller the person a RunOnPerson auto play runs as; a new one runs as its creator without it.
	RunAsMe    bool
	MemoryIDs  []string
	OneAtATime bool
	Resource   string
	// ComputerRule narrows the run-as person's computers, ids needing run_on person; nil keeps the auto play's.
	ComputerRule *labels.ComputerRule
}

// ListAutoPlays returns a play's auto plays, oldest first (autoplays:read).
func (s *Service) ListAutoPlays(ctx context.Context, workspaceID, playID string) ([]*AutoPlay, error) {
	if err := s.require(ctx, strings.TrimSpace(workspaceID), permissions.AutoplaysRead); err != nil {
		return nil, err
	}
	p, err := s.seen(ctx, workspaceID, playID)
	if err != nil {
		return nil, err
	}
	list, err := s.repo.ListAutoPlays(ctx, []string{p.ID})
	if err != nil {
		return nil, fmt.Errorf("list auto plays of play %s: %w", p.ID, err)
	}
	return list, nil
}

// AutoPlaysByPlay returns the auto plays of workspaceID's plays named in playIDs in one read, keyed by play
// (autoplays:read); every named play the caller can see has an entry, empty when it has none.
func (s *Service) AutoPlaysByPlay(ctx context.Context, workspaceID string, playIDs []string) (map[string][]*AutoPlay, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if err := s.require(ctx, workspaceID, permissions.AutoplaysRead); err != nil {
		return nil, err
	}
	sees := s.perm.PlaysWith(ctx, actorID(ctx), workspaceID, permissions.PlaysRead, playIDs)
	playIDs = slices.DeleteFunc(slices.Clone(playIDs), func(id string) bool { return !sees[id] })
	out := make(map[string][]*AutoPlay, len(playIDs))
	for _, id := range playIDs {
		out[id] = []*AutoPlay{}
	}
	if len(playIDs) == 0 {
		return out, nil
	}
	list, err := s.repo.ListAutoPlays(ctx, playIDs)
	if err != nil {
		return nil, fmt.Errorf("list auto plays in workspace %s: %w", workspaceID, err)
	}
	for _, a := range list {
		if a.WorkspaceID == workspaceID {
			out[a.PlayID] = append(out[a.PlayID], a)
		}
	}
	return out, nil
}

// CreateAutoPlay adds an auto play to a play, switched off until someone turns it on (autoplays:write).
func (s *Service) CreateAutoPlay(ctx context.Context, workspaceID, playID string, in AutoPlayInput) (*AutoPlay, error) {
	if err := s.require(ctx, strings.TrimSpace(workspaceID), permissions.AutoplaysWrite); err != nil {
		return nil, err
	}
	p, err := s.seen(ctx, workspaceID, playID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	a := &AutoPlay{ID: ids.New(), PlayID: p.ID, WorkspaceID: p.WorkspaceID, CreatedBy: actorID(ctx), CreatedAt: now, UpdatedAt: now}
	a.apply(in)
	a.normalize(p.Type)
	s.pinPerson(ctx, a, true)
	if err := a.Validate(p.Type); err != nil {
		return nil, err
	}
	if err := s.checkComputers(ctx, a); err != nil {
		return nil, err
	}
	if err := s.repo.CreateAutoPlay(ctx, a, s.event(TopicAutoPlayCreated, AutoPlayEvent{AutoPlay: *a})); err != nil {
		return nil, fmt.Errorf("create auto play on play %s: %w", p.ID, err)
	}
	return a, nil
}

// UpdateAutoPlay replaces an auto play's editable fields and its switch (autoplays:write).
func (s *Service) UpdateAutoPlay(ctx context.Context, workspaceID, playID, id string, enabled bool, in AutoPlayInput) (*AutoPlay, error) {
	if err := s.require(ctx, strings.TrimSpace(workspaceID), permissions.AutoplaysWrite); err != nil {
		return nil, err
	}
	p, a, err := s.autoPlayInPlay(ctx, workspaceID, playID, id)
	if err != nil {
		return nil, err
	}
	a.Enabled = enabled
	a.apply(in)
	a.UpdatedAt = s.now().UTC()
	a.normalize(p.Type)
	s.pinPerson(ctx, a, in.RunAsMe)
	if err := a.Validate(p.Type); err != nil {
		return nil, err
	}
	if err := s.checkComputers(ctx, a); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAutoPlay(ctx, a, s.event(TopicAutoPlayUpdated, AutoPlayEvent{AutoPlay: *a})); err != nil {
		return nil, fmt.Errorf("update auto play %s: %w", a.ID, err)
	}
	return a, nil
}

// DeleteAutoPlay removes an auto play from its play (autoplays:delete).
func (s *Service) DeleteAutoPlay(ctx context.Context, workspaceID, playID, id string) error {
	if err := s.require(ctx, strings.TrimSpace(workspaceID), permissions.AutoplaysDelete); err != nil {
		return err
	}
	_, a, err := s.autoPlayInPlay(ctx, workspaceID, playID, id)
	if err != nil {
		return err
	}
	evt := s.event(TopicAutoPlayDeleted, AutoPlayDeletedEvent{ID: a.ID, PlayID: a.PlayID, WorkspaceID: a.WorkspaceID})
	if err := s.repo.DeleteAutoPlay(ctx, a.ID, evt); err != nil {
		return fmt.Errorf("delete auto play %s: %w", a.ID, err)
	}
	return nil
}

// autoPlayInPlay finds an auto play under its play and workspace; one under another play reads as not found.
func (s *Service) autoPlayInPlay(ctx context.Context, workspaceID, playID, id string) (*Play, *AutoPlay, error) {
	p, err := s.seen(ctx, workspaceID, playID)
	if err != nil {
		return nil, nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, nil, fmt.Errorf("%w: auto play id is required", apperrs.ErrInvalid)
	}
	a, err := s.repo.GetAutoPlay(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("get auto play %s: %w", id, err)
	}
	if a.PlayID != p.ID {
		return nil, nil, fmt.Errorf("get auto play %s: %w", id, apperrs.ErrNotFound)
	}
	return p, a, nil
}

// pinPerson makes the caller the person a run-as-person auto play runs as when picked; it then takes their order.
func (s *Service) pinPerson(ctx context.Context, a *AutoPlay, picked bool) {
	if a.RunOn != RunOnPerson || !picked {
		return
	}
	a.RunAs = actorID(ctx)
}

func (a *AutoPlay) apply(in AutoPlayInput) {
	a.Moment = in.Moment
	a.MomentStage = in.MomentStage
	a.Schedule = in.Schedule
	a.OneAtATime, a.Resource = in.OneAtATime, in.Resource
	a.Conditions = in.Conditions
	a.Priority = in.Priority
	a.OnceWithinMinutes = in.OnceWithinMinutes
	a.RunOn = in.RunOn
	a.MemoryIDs = in.MemoryIDs
	if in.ComputerRule != nil {
		a.ComputerRule = *in.ComputerRule
	}
}

// checkComputers refuses ids in a's computer rule that its run-as person may not run on: not theirs and not shared by Run agents.
func (s *Service) checkComputers(ctx context.Context, a *AutoPlay) error {
	if len(a.ComputerRule.IDs) == 0 {
		return nil
	}
	if a.RunAs == "" {
		return fmt.Errorf("%w: pick the person it runs as before naming computers", apperrs.ErrInvalid)
	}
	for _, id := range a.ComputerRule.IDs {
		if s.computers == nil || !s.computers.MayRunOn(ctx, a.RunAs, id) {
			return fmt.Errorf("%w: computer %s is not one the person it runs as owns or may run agents on", apperrs.ErrInvalid, id)
		}
	}
	return nil
}
