package workspace

import (
	"context"
	"fmt"
	"maps"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SetupChange patches a project's setup record: Finished when set, and a mark for each step named.
type SetupChange struct {
	Finished *bool
	Steps    map[SetupStep]SetupMark
}

func (c SetupChange) validate() error {
	for step, mark := range c.Steps {
		if !slices.Contains(SetupSteps, step) {
			return fmt.Errorf("%w: %q is not a project wizard step; the steps are project, repository, service, env, reach, and branches", apperrs.ErrInvalid, step)
		}
		if mark != SetupDone && mark != SetupSkipped {
			return fmt.Errorf("%w: step %s must be marked %q or %q", apperrs.ErrInvalid, step, SetupDone, SetupSkipped)
		}
	}
	return nil
}

// apply returns setup with change laid over it; a skip never undoes a step already done.
func (setup ProjectSetup) apply(change SetupChange) ProjectSetup {
	next := ProjectSetup{Finished: setup.Finished, Steps: maps.Clone(setup.Steps)}
	if next.Steps == nil {
		next.Steps = map[SetupStep]SetupMark{}
	}
	if change.Finished != nil {
		next.Finished = *change.Finished
	}
	for step, mark := range change.Steps {
		if mark == SetupSkipped && next.Steps[step] == SetupDone {
			continue
		}
		next.Steps[step] = mark
	}
	return next
}

// ChangeSetup records the project wizard's progress on a project (projects:write); only Finished moves finished (ADR 0143).
func (s *Service) ChangeSetup(ctx context.Context, projectID string, change SetupChange) (*Project, error) {
	if err := s.requireOn(ctx, projectID, permissions.ProjectsWrite); err != nil {
		return nil, err
	}
	if err := change.validate(); err != nil {
		return nil, err
	}
	current, err := s.repo.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("change setup of project %s: %w", projectID, err)
	}
	setup := current.Setup.apply(change)
	if setup.Finished == current.Setup.Finished && maps.Equal(setup.Steps, current.Setup.Steps) {
		return current, nil
	}
	updated := *current
	updated.Setup = setup
	updated.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicProjectSetupChanged, Payload: ProjectSetupChangedEvent{
		ProjectID: updated.ID, WorkspaceID: updated.WorkspaceID, Setup: setup,
	}}
	if err := s.repo.SaveSetup(ctx, &updated, evt); err != nil {
		return nil, fmt.Errorf("change setup of project %s: %w", projectID, err)
	}
	return &updated, nil
}
