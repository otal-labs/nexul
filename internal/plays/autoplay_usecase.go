package plays

import (
	"context"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// AutoPlayInput is the editable surface of an auto play; conditions and priority are replaced whole.
type AutoPlayInput struct {
	Moment            Moment
	MomentStage       *Stage
	Conditions        Conditions
	Priority          Priority
	OnceWithinMinutes int
	RunOn             RunOn
	// ComputerIDs are the computers a RunOnComputer auto play lands on, in order. Nil keeps the stored list and its
	// person; a list, even an empty one (the picker's own order), is a pick made as the caller.
	ComputerIDs *[]string
	MemoryIDs   []string
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
	if err := s.pinComputers(ctx, a, nil, in.ComputerIDs != nil); err != nil {
		return nil, err
	}
	if err := a.Validate(p.Type); err != nil {
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
	was := slices.Clone(a.ComputerIDs)
	a.Enabled = enabled
	a.apply(in)
	a.UpdatedAt = s.now().UTC()
	a.normalize(p.Type)
	if err := s.pinComputers(ctx, a, was, in.ComputerIDs != nil); err != nil {
		return nil, err
	}
	if err := a.Validate(p.Type); err != nil {
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

// AutoPlayLimits are workspaceID's bounds on automatic runs (autoplays:read).
func (s *Service) AutoPlayLimits(ctx context.Context, workspaceID string) (AutoPlayLimits, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if err := s.require(ctx, workspaceID, permissions.AutoplaysRead); err != nil {
		return AutoPlayLimits{}, err
	}
	limits, err := s.repo.AutoPlayLimits(ctx, workspaceID)
	if err != nil {
		return AutoPlayLimits{}, fmt.Errorf("get the auto play limits of workspace %s: %w", workspaceID, err)
	}
	return limits, nil
}

// AutoPlayLimitsPatch names the limits to change; a nil field keeps its value.
type AutoPlayLimitsPatch struct {
	DailyCapPerTicket *int `json:"daily_cap_per_ticket"`
}

// SetAutoPlayLimits changes workspaceID's bounds on automatic runs (autoplays:write).
func (s *Service) SetAutoPlayLimits(ctx context.Context, workspaceID string, patch AutoPlayLimitsPatch) (AutoPlayLimits, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if err := s.require(ctx, workspaceID, permissions.AutoplaysWrite); err != nil {
		return AutoPlayLimits{}, err
	}
	if patch.DailyCapPerTicket == nil {
		return AutoPlayLimits{}, fmt.Errorf("%w: send daily_cap_per_ticket", apperrs.ErrInvalid)
	}
	if c := patch.DailyCapPerTicket; c != nil && (*c < MinAutoPlayDailyCap || *c > MaxAutoPlayDailyCap) {
		return AutoPlayLimits{}, fmt.Errorf("%w: the daily cap on automatic runs per ticket is %d to %d", apperrs.ErrInvalid, MinAutoPlayDailyCap, MaxAutoPlayDailyCap)
	}
	limits, err := s.repo.AutoPlayLimits(ctx, workspaceID)
	if err != nil {
		return AutoPlayLimits{}, fmt.Errorf("get the auto play limits of workspace %s: %w", workspaceID, err)
	}
	if patch.DailyCapPerTicket != nil {
		limits.DailyCapPerTicket = *patch.DailyCapPerTicket
	}
	evt := s.event(TopicAutoPlayLimitsUpdated, AutoPlayLimitsEvent{WorkspaceID: workspaceID, DailyCapPerTicket: limits.DailyCapPerTicket})
	if err := s.repo.SetAutoPlayLimits(ctx, workspaceID, limits, evt); err != nil {
		return AutoPlayLimits{}, fmt.Errorf("set the auto play limits of workspace %s: %w", workspaceID, err)
	}
	return limits, nil
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

// pinComputers records the caller as the person a run-on-computer auto play runs as when a pick changes its list or
// it has no person yet, and checks they own each computer or hold computers:run_agents on it; an unchanged list keeps
// its person. Without a pick the list and its person stay as they were.
func (s *Service) pinComputers(ctx context.Context, a *AutoPlay, was []string, picked bool) error {
	if a.RunOn != RunOnComputer || !picked {
		return nil
	}
	if len(a.ComputerIDs) > MaxAutoPlayComputers {
		return fmt.Errorf("%w: an auto play runs on at most %d computers", apperrs.ErrInvalid, MaxAutoPlayComputers)
	}
	if slices.Equal(a.ComputerIDs, was) && a.RunAs != "" {
		return nil
	}
	for _, id := range a.ComputerIDs {
		ok, err := mayRunAgents(ctx, s.computers, actorID(ctx), id)
		if err != nil {
			return fmt.Errorf("check computer %s: %w", id, err)
		}
		if !ok {
			return fmt.Errorf("%w: pick computers you own or may run agents on", apperrs.ErrInvalid)
		}
	}
	a.RunAs = actorID(ctx)
	return nil
}

func (a *AutoPlay) apply(in AutoPlayInput) {
	a.Moment = in.Moment
	a.MomentStage = in.MomentStage
	a.Conditions = in.Conditions
	a.Priority = in.Priority
	a.OnceWithinMinutes = in.OnceWithinMinutes
	a.RunOn = in.RunOn
	if in.ComputerIDs != nil {
		a.ComputerIDs = *in.ComputerIDs
	}
	a.MemoryIDs = in.MemoryIDs
}
