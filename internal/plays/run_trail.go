package plays

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// GetTrail returns one trail to its starter or to someone who can see its play, and docs:thread on the doc for a doc
// trail; anyone else, or someone outside its project, gets not found.
func (r *Runner) GetTrail(ctx context.Context, id string) (*Trail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: trail id is required", apperrs.ErrInvalid)
	}
	t, err := r.trails.GetTrail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", id, err)
	}
	if len(r.readable(ctx, []*Trail{t})) == 0 {
		return nil, fmt.Errorf("get trail %s: %w", id, apperrs.ErrNotFound)
	}
	if err := r.requireThread(ctx, t.WorkspaceID, t.TargetType, t.TargetID); err != nil {
		return nil, err
	}
	return t, nil
}

// opensProject keeps a trail on a project its reader can no longer open out of every read; the row stays, so access
// given back brings it back.
func (r *Runner) opensProject(ctx context.Context, t *Trail) bool {
	return t.ProjectID == "" || r.perm.HasPermission(ctx, actorID(ctx), t.WorkspaceID, permissions.Member, resourceTypeProject, t.ProjectID)
}

// readable keeps the trails of list the caller reads: in a project they can open, and started by them or of a play they
// can see, from one read of the play rules per workspace (ADR 0148).
func (r *Runner) readable(ctx context.Context, list []*Trail) []*Trail {
	actor := actorID(ctx)
	byWorkspace := map[string][]string{}
	for _, t := range list {
		byWorkspace[t.WorkspaceID] = append(byWorkspace[t.WorkspaceID], t.PlayID)
	}
	sees := make(map[string]map[string]bool, len(byWorkspace))
	for workspaceID, playIDs := range byWorkspace {
		sees[workspaceID] = r.perm.PlaysWith(ctx, actor, workspaceID, permissions.PlaysRead, playIDs)
	}
	return slices.DeleteFunc(slices.Clone(list), func(t *Trail) bool {
		started := actor != "" && actor == t.StarterID
		return !r.opensProject(ctx, t) || (!started && !sees[t.WorkspaceID][t.PlayID])
	})
}

// ListTrails returns a target's trails newest first, under the same gate as GetTrail; a playID keeps that play's alone.
func (r *Runner) ListTrails(ctx context.Context, targetType TargetType, targetID, playID string) ([]*Trail, error) {
	targetID = strings.TrimSpace(targetID)
	if !targetType.valid() || targetID == "" {
		return nil, fmt.Errorf("%w: target type (ticket, doc, or interview) and target id are required", apperrs.ErrInvalid)
	}
	list, err := r.trails.ListTrailsByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("list trails for %s %s: %w", targetType, targetID, err)
	}
	list = r.readable(ctx, list)
	if len(list) == 0 {
		return nil, nil
	}
	if err := r.requireThread(ctx, list[0].WorkspaceID, targetType, targetID); err != nil {
		return nil, err
	}
	return slices.DeleteFunc(list, func(t *Trail) bool { return playID != "" && t.PlayID != playID }), nil
}

// ActiveTrails maps each target with an active trail to that trail; targets the caller cannot read are left out, not refused.
func (r *Runner) ActiveTrails(ctx context.Context, targetType TargetType, targetIDs []string) (map[string]*Trail, error) {
	if !targetType.valid() {
		return nil, fmt.Errorf("%w: target type (ticket, doc, or interview) is required", apperrs.ErrInvalid)
	}
	ids := make([]string, 0, len(targetIDs))
	for _, id := range targetIDs {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	out := map[string]*Trail{}
	if len(ids) == 0 {
		return out, nil
	}
	list, err := r.trails.ListActiveTrailsByTargets(ctx, targetType, ids)
	if err != nil {
		return nil, fmt.Errorf("list active trails: %w", err)
	}
	for _, t := range r.readable(ctx, list) {
		out[t.TargetID] = t
	}
	return out, nil
}

// LatestChoices returns what starterID last picked for playID in projectID; empty choices when never run.
func (r *Runner) LatestChoices(ctx context.Context, starterID, playID, projectID string) (*Choices, error) {
	starterID, playID, projectID = strings.TrimSpace(starterID), strings.TrimSpace(playID), strings.TrimSpace(projectID)
	if starterID == "" || playID == "" || projectID == "" {
		return nil, fmt.Errorf("%w: starter, play id, and project id are required", apperrs.ErrInvalid)
	}
	play, err := r.plays.Get(ctx, playID)
	if err != nil {
		return nil, fmt.Errorf("get play %s: %w", playID, err)
	}
	if !r.perm.HasPermission(ctx, actorID(ctx), play.WorkspaceID, permissions.PlaysRead, resourceTypePlay, play.ID) {
		return nil, fmt.Errorf("get play %s: %w", playID, apperrs.ErrNotFound)
	}
	t, err := r.trails.LatestTrailForChoices(ctx, starterID, playID, projectID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return &Choices{MemoryIDs: []string{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest trail for play %s: %w", playID, err)
	}
	return &Choices{MemoryIDs: t.SelectedMemoryIDs, ComputerID: t.ComputerID, Provider: t.Provider, Model: t.Model, ModelOptions: t.ModelOptions}, nil
}

// requireThread refuses a doc's runs to someone who may not see the doc's working thread.
func (r *Runner) requireThread(ctx context.Context, workspaceID string, targetType TargetType, targetID string) error {
	if targetType == TargetDoc && !r.perm.HasPermission(ctx, actorID(ctx), workspaceID, permissions.DocsThread, resourceTypeDoc, targetID) {
		return fmt.Errorf("%w: %s required on doc %s", apperrs.ErrForbidden, permissions.DocsThread, targetID)
	}
	return nil
}

// requireSteer lets a run's starter, or someone who can see its play and holds plays:write on it, act on the run; a run
// the caller can't read is not found.
func (r *Runner) requireSteer(ctx context.Context, t *Trail, act string) error {
	actor := actorID(ctx)
	if actor != "" && actor == t.StarterID {
		return nil
	}
	if len(r.readable(ctx, []*Trail{t})) == 0 {
		return fmt.Errorf("get trail %s: %w", t.ID, apperrs.ErrNotFound)
	}
	if !r.perm.HasPermission(ctx, actor, t.WorkspaceID, permissions.PlaysWrite, resourceTypePlay, t.PlayID) {
		return fmt.Errorf("%w: only the starter or a %s holder can %s", apperrs.ErrForbidden, permissions.PlaysWrite, act)
	}
	return nil
}
