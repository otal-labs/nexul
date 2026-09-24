package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// HandlePush deploys the branch for every base stack with a matching branch deploy rule.
func (s *Service) HandlePush(ctx context.Context, ev eventbus.Event) error {
	var p PushTrigger
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse push trigger: %w", err))
	}
	if p.Repo == "" || p.Branch == "" {
		return apperrs.Fatal(fmt.Errorf("%w: push trigger missing repo or branch", apperrs.ErrInvalid))
	}
	bases, err := s.stacks.ListByBuildRepo(ctx, p.Owner, p.Repo)
	if err != nil {
		return fmt.Errorf("list stacks for repo %s/%s: %w", p.Owner, p.Repo, err)
	}
	var errs []error
	for _, base := range bases {
		rule, ok := matchingRule(base.BranchDeployRules, p.Branch)
		if !ok {
			continue
		}
		if err := s.applyBranchDeployRule(ctx, base, rule, p); err != nil {
			errs = append(errs, fmt.Errorf("branch deploy %s@%s: %w", base.Name, p.Branch, err))
		}
	}
	return errors.Join(errs...)
}

// HandleBranchDeleted tears down deployments a matching rule maintained; an in-place rule has nothing to tear down.
func (s *Service) HandleBranchDeleted(ctx context.Context, ev eventbus.Event) error {
	var p BranchDeletedTrigger
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse branch-deleted trigger: %w", err))
	}
	if p.Repo == "" || p.Branch == "" {
		return apperrs.Fatal(fmt.Errorf("%w: branch-deleted trigger missing repo or branch", apperrs.ErrInvalid))
	}
	bases, err := s.stacks.ListByBuildRepo(ctx, p.Owner, p.Repo)
	if err != nil {
		return fmt.Errorf("list stacks for repo %s/%s: %w", p.Owner, p.Repo, err)
	}
	var errs []error
	for _, base := range bases {
		rule, ok := matchingRule(base.BranchDeployRules, p.Branch)
		if !ok || !rule.DerivesClone() {
			continue
		}
		name := base.Name + "-" + rule.CloneSuffix(p.Branch)
		derived, err := s.stacks.GetBySlugAndMachine(ctx, Slug(name, dnsLabelMaxLen), base.Machine)
		if err != nil {
			if errors.Is(err, apperrs.ErrNotFound) {
				continue
			}
			errs = append(errs, fmt.Errorf("teardown %s: %w", name, err))
			continue
		}
		if err := s.DeleteStack(ctx, derived.ID); err != nil && !errors.Is(err, apperrs.ErrNotFound) {
			errs = append(errs, fmt.Errorf("teardown %s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

// matchingRule returns the first rule whose pattern matches branch.
func matchingRule(rules []BranchDeployRule, branch string) (BranchDeployRule, bool) {
	for _, r := range rules {
		if r.Matches(branch) {
			return r, true
		}
	}
	return BranchDeployRule{}, false
}

// applyBranchDeployRule resolves the target, triggers its deploy, and ensures its hostname exposure if templated.
func (s *Service) applyBranchDeployRule(ctx context.Context, base *Stack, rule BranchDeployRule, p PushTrigger) error {
	target := base
	if rule.DerivesClone() {
		derived, err := s.upsertBranchDeployment(ctx, base, rule, p.Branch)
		if err != nil {
			return err
		}
		target = derived
	}
	if err := s.deployBranchTarget(ctx, target, base, p); err != nil {
		return err
	}
	if rule.HostnameTemplate != "" {
		if err := s.ensureBranchExposure(ctx, target, rule, p.Branch); err != nil {
			return err
		}
	}
	return nil
}

// upsertBranchDeployment creates or refreshes the derived clone; a collision with another stack is a clear error.
func (s *Service) upsertBranchDeployment(ctx context.Context, base *Stack, rule BranchDeployRule, branch string) (*Stack, error) {
	name := base.Name + "-" + rule.CloneSuffix(branch)
	existing, err := s.stacks.GetBySlugAndMachine(ctx, Slug(name, dnsLabelMaxLen), base.Machine)
	if err != nil {
		if !errors.Is(err, apperrs.ErrNotFound) {
			return nil, fmt.Errorf("get branch deployment %s: %w", name, err)
		}
		created, err := s.CreateStack(ctx, Stack{
			ProjectID:     base.ProjectID,
			Name:          name,
			Machine:       base.Machine,
			Strategy:      base.Strategy,
			ComposePath:   base.ComposePath,
			Env:           rule.ApplyOverrides(base.Env),
			DockerNetwork: rule.DockerNetwork,
			BuildSource:   base.BuildSource,
			DerivedFrom:   base.ID,
			Branch:        branch,
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("create branch deployment %s: %w", name, err)
		}
		return created, nil
	}
	if existing.DerivedFrom != base.ID || existing.Branch != branch {
		return nil, fmt.Errorf("%w: stack name %q is already in use and is not this branch's deployment", apperrs.ErrConflict, name)
	}
	existing.Strategy = base.Strategy
	existing.ComposePath = base.ComposePath
	existing.Env = rule.ApplyOverrides(base.Env)
	existing.DockerNetwork = rule.DockerNetwork
	existing.BuildSource = base.BuildSource
	updated, err := s.UpdateStack(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("update branch deployment %s: %w", name, err)
	}
	return updated, nil
}

// deployBranchTarget builds from the ref when buildable, else redeploys the last healthy image.
func (s *Service) deployBranchTarget(ctx context.Context, target, base *Stack, p PushTrigger) error {
	req := DeployRequest{
		StackID:  target.ID,
		RuleName: fmt.Sprintf("push:%s/%s@%s", p.Owner, p.Repo, p.Branch),
	}
	if base.BuildSource.Buildable() {
		req.Ref = p.SHA
		return s.deployUnlessConflict(ctx, target, req)
	}
	last, err := s.repo.LastHealthy(ctx, target.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve image to redeploy %s: %w", target.Name, err)
	}
	req.Image = last.Image
	return s.deployUnlessConflict(ctx, target, req)
}

// deployUnlessConflict treats an already-running deploy of the same stack as done, not as a failure.
func (s *Service) deployUnlessConflict(ctx context.Context, target *Stack, req DeployRequest) error {
	_, err := s.Deploy(ctx, req)
	if errors.Is(err, apperrs.ErrConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("deploy %s: %w", target.Name, err)
	}
	return nil
}

// ensureBranchExposure exposes target's hostname (rule template + HostnameLabel) on the rule's configured port.
func (s *Service) ensureBranchExposure(ctx context.Context, target *Stack, rule BranchDeployRule, branch string) error {
	if s.exposures == nil {
		return nil
	}
	hostname := strings.ReplaceAll(rule.HostnameTemplate, "{branch}", rule.HostnameLabel(branch))
	if err := s.exposures.EnsureExposure(ctx, hostname, target.Name, rule.Port, rule.DockerNetwork); err != nil {
		return fmt.Errorf("expose %s at %s: %w", target.Name, hostname, err)
	}
	return nil
}
