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

// GetTrail returns one trail; plays:read in its workspace, and docs:thread on the doc for a doc trail.
func (r *Runner) GetTrail(ctx context.Context, id string) (*Trail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: trail id is required", apperrs.ErrInvalid)
	}
	t, err := r.trails.GetTrail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trail %s: %w", id, err)
	}
	if !r.opensProject(ctx, t) {
		return nil, fmt.Errorf("get trail %s: %w", id, apperrs.ErrNotFound)
	}
	if err := r.requireTrailAccess(ctx, t.WorkspaceID, t.TargetType, t.TargetID); err != nil {
		return nil, err
	}
	return t, nil
}

// opensProject keeps a trail on a project its reader can no longer open out of every read; the row stays, so access
// given back brings it back.
func (r *Runner) opensProject(ctx context.Context, t *Trail) bool {
	return t.ProjectID == "" || r.perm.HasPermission(ctx, actorID(ctx), t.WorkspaceID, permissions.Member, resourceTypeProject, t.ProjectID)
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
	list = slices.DeleteFunc(list, func(t *Trail) bool { return !r.opensProject(ctx, t) })
	if len(list) == 0 {
		return nil, nil
	}
	if err := r.requireTrailAccess(ctx, list[0].WorkspaceID, targetType, targetID); err != nil {
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
	actor := actorID(ctx)
	for _, t := range list {
		if r.perm.HasPermission(ctx, actor, t.WorkspaceID, permissions.PlaysRead, "", "") && r.opensProject(ctx, t) {
			out[t.TargetID] = t
		}
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
	if !r.perm.HasPermission(ctx, actorID(ctx), play.WorkspaceID, permissions.PlaysRead, "", "") {
		return nil, fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.PlaysRead)
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

func (r *Runner) requireTrailAccess(ctx context.Context, workspaceID string, targetType TargetType, targetID string) error {
	actor := actorID(ctx)
	if !r.perm.HasPermission(ctx, actor, workspaceID, permissions.PlaysRead, "", "") {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, permissions.PlaysRead)
	}
	if targetType == TargetDoc && !r.perm.HasPermission(ctx, actor, workspaceID, permissions.DocsThread, resourceTypeDoc, targetID) {
		return fmt.Errorf("%w: %s required on doc %s", apperrs.ErrForbidden, permissions.DocsThread, targetID)
	}
	return nil
}
