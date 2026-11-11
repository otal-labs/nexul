package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// TopicRunnerSecretsChanged is a machine runner's secret check, fed by the runner domain; consumed here, never part of the catalog.
const TopicRunnerSecretsChanged = "runner.secrets_changed"

// secretsChangedReport mirrors the runner's report independently (ADR 0017): the stacks of one machine, by slug, and
// the variable names whose secret changed. It never carries a value or a reference.
type secretsChangedReport struct {
	Machine string         `json:"machine"`
	Changes []secretChange `json:"changes"`
}

type secretChange struct {
	Slug      string   `json:"slug"`
	Variables []string `json:"variables"`
}

// HandleSecretsChanged is the runner.secrets_changed consumer: each stack named keeps a notice of the variables that
// changed, until its next successful deploy clears it. A variable already in the notice is not announced again, and a
// stack with a deploy under way is skipped, since that deploy resolves the secrets afresh.
func (s *Service) HandleSecretsChanged(ctx context.Context, ev eventbus.Event) error {
	var p secretsChangedReport
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicRunnerSecretsChanged, err))
	}
	if p.Machine == "" {
		return nil
	}
	var errs []error
	for _, c := range p.Changes {
		if err := s.noteSecretsChanged(ctx, p.Machine, c); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Service) noteSecretsChanged(ctx context.Context, machine string, c secretChange) error {
	stack, err := s.stacks.GetBySlugAndMachine(ctx, c.Slug, machine)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find stack %s on %s: %w", c.Slug, machine, err)
	}
	busy, err := s.repo.HasActive(ctx, stack.ID)
	if err != nil {
		return fmt.Errorf("check deploys of stack %s: %w", stack.ID, err)
	}
	if busy {
		return nil
	}
	names := slices.Compact(slices.Sorted(slices.Values(slices.Concat(stack.SecretsChanged, c.Variables))))
	if len(names) == len(stack.SecretsChanged) {
		return nil
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicSecretsChanged, Payload: SecretsChangedEvent{
		ProjectID: stack.ProjectID, StackID: stack.ID, StackName: stack.Name, Variables: names,
	}}
	if err := s.stacks.SetSecretsChanged(ctx, stack.ID, names, evt); err != nil {
		return fmt.Errorf("record changed secrets of stack %s: %w", stack.ID, err)
	}
	return nil
}

// clearSecretsChanged ends a stack's notice once a deploy of it is healthy; it is best-effort, like the outage close
// beside it, because the deploy has already landed.
func (s *Service) clearSecretsChanged(ctx context.Context, deployID string) {
	d, err := s.repo.GetByID(ctx, deployID)
	if err != nil {
		slog.Warn("secrets notice after deploy: get deploy failed", "deploy_id", deployID, "error", err)
		return
	}
	stack, err := s.stacks.GetByID(ctx, d.StackID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return
	}
	if err != nil {
		slog.Warn("secrets notice after deploy: get stack failed", "deploy_id", deployID, "stack_id", d.StackID, "error", err)
		return
	}
	if len(stack.SecretsChanged) == 0 {
		return
	}
	stack.SecretsChanged = nil
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicStackUpdated, Payload: newStackEvent(*stack, nil)}
	if err := s.stacks.SetSecretsChanged(ctx, stack.ID, nil, evt); err != nil {
		slog.Warn("secrets notice after deploy: clear failed", "deploy_id", deployID, "stack_id", stack.ID, "error", err)
	}
}
