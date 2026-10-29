package plays

import (
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// runScope is the runs the caller may see on a play they see: those in projects they open, and a doc's run only where
// they may see the doc's thread, as requireThread asks of a doc's own trail list.
func runScope(ctx context.Context, perm PermissionGate) (RunScope, error) {
	projectIDs, all, err := perm.CallerProjects(ctx, permissions.Member)
	if err != nil {
		return RunScope{}, fmt.Errorf("list the projects the caller may open: %w", err)
	}
	scope := RunScope{ProjectScope: ProjectScope{All: all, ProjectIDs: projectIDs}}
	if all {
		return scope, nil
	}
	docs, err := perm.DocsWith(ctx, actorID(ctx), "", permissions.DocsThread)
	if err != nil {
		return RunScope{}, fmt.Errorf("list the docs whose thread the caller may see: %w", err)
	}
	scope.Docs = &docs
	return scope, nil
}

// ListPlayRuns pages a play's runs across every target the caller may see, newest first, or with waiting the runs
// waiting on an answer, oldest first; a play they can't see is not found.
func (r *Runner) ListPlayRuns(ctx context.Context, playID string, waiting bool, w paging.Window) ([]PlayRun, int, error) {
	playID = strings.TrimSpace(playID)
	if playID == "" {
		return nil, 0, fmt.Errorf("%w: play id is required", apperrs.ErrInvalid)
	}
	play, err := r.plays.Get(ctx, playID)
	if err != nil {
		return nil, 0, fmt.Errorf("get play %s: %w", playID, err)
	}
	if !r.perm.HasPermission(ctx, actorID(ctx), play.WorkspaceID, permissions.PlaysRead, resourceTypePlay, play.ID) {
		return nil, 0, fmt.Errorf("get play %s: %w", playID, apperrs.ErrNotFound)
	}
	scope, err := runScope(ctx, r.perm)
	if err != nil {
		return nil, 0, err
	}
	runs, total, err := r.trails.ListPlayRuns(ctx, play.ID, scope, waiting, w.Clamped())
	if err != nil {
		return nil, 0, err
	}
	for i := range runs {
		if runs[i].StarterID != actorID(ctx) {
			runs[i].PassedOver = nil
		}
	}
	return runs, total, nil
}

// withHealth fills each listed play's RunHealth from the runs the caller may see, in one read for the whole list.
func (s *Service) withHealth(ctx context.Context, list []Listed) error {
	if len(list) == 0 {
		return nil
	}
	scope, err := runScope(ctx, s.perm)
	if err != nil {
		return err
	}
	ids := make([]string, len(list))
	for i := range list {
		ids[i] = list[i].ID
	}
	health, err := s.repo.RunHealth(ctx, ids, scope, s.now().Add(-healthWindow))
	if err != nil {
		return fmt.Errorf("read the health of the plays: %w", err)
	}
	for i := range list {
		list[i].RunHealth = health[list[i].ID]
	}
	return nil
}
