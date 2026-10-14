package deploy

import (
	"context"
	"slices"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// withoutEnvValues copies the stack with env and rule-override values emptied, keys kept, and EnvMasked set.
func (s Stack) withoutEnvValues() Stack {
	s.EnvMasked = s.holdsEnvValues()
	s.Env = blankValues(s.Env)
	if len(s.BranchDeployRules) > 0 {
		s.BranchDeployRules = slices.Clone(s.BranchDeployRules)
		for i := range s.BranchDeployRules {
			s.BranchDeployRules[i].Overrides = blankValues(s.BranchDeployRules[i].Overrides)
		}
	}
	return s
}

func blankValues(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	blank := make(map[string]string, len(m))
	for k := range m {
		blank[k] = ""
	}
	return blank
}

func (s *Stack) holdsEnvValues() bool {
	return len(s.Env) > 0 || slices.ContainsFunc(s.BranchDeployRules, func(r BranchDeployRule) bool { return len(r.Overrides) > 0 })
}

// covers reports whether the scope keeps the stack, as the list queries filter it.
func (sc DeployScope) covers(stack *Stack) bool {
	inProject := slices.Contains(sc.ProjectIDs, stack.ProjectID) && !slices.Contains(sc.Denied, stack.ID)
	return sc.All || (stack.ProjectID == "" && sc.Anywhere) || inProject || slices.Contains(sc.Allowed, stack.ID)
}

// canWrite reports whether the caller may change the stack, which is what makes its env values theirs to read.
func (s *Service) canWrite(ctx context.Context, stack *Stack) (bool, error) {
	err := s.requireStack(ctx, stack, permissions.StacksWrite)
	if err == nil {
		return true, nil
	}
	if permissions.Refused(err) {
		return false, nil
	}
	return false, err
}

// readableBy empties the env values of a stack the caller can read but not write; the keys stay.
func (s *Service) readableBy(ctx context.Context, stack *Stack) (*Stack, error) {
	if !stack.holdsEnvValues() {
		return stack, nil
	}
	writable, err := s.canWrite(ctx, stack)
	if err != nil {
		return nil, err
	}
	if writable {
		return stack, nil
	}
	masked := stack.withoutEnvValues()
	return &masked, nil
}

// readableBranchesBy is readableBy for a base stack's branch deployments, which follow the base stack's rules.
func (s *Service) readableBranchesBy(ctx context.Context, base *Stack, branches []*Stack) ([]*Stack, error) {
	if !slices.ContainsFunc(branches, (*Stack).holdsEnvValues) {
		return branches, nil
	}
	writable, err := s.canWrite(ctx, base)
	if err != nil {
		return nil, err
	}
	if writable {
		return branches, nil
	}
	return maskedAll(branches, func(*Stack) bool { return false }), nil
}

// readableListBy is readableBy for a list, asking access once for all of them.
func (s *Service) readableListBy(ctx context.Context, stacks []*Stack, workspaceID string, instance bool) ([]*Stack, error) {
	if !slices.ContainsFunc(stacks, (*Stack).holdsEnvValues) {
		return stacks, nil
	}
	writable, err := s.listScope(ctx, workspaceID, permissions.StacksWrite, instance)
	if err != nil {
		return nil, err
	}
	return maskedAll(stacks, writable.covers), nil
}

// maskedAll empties the env values of every stack that keep rejects.
func maskedAll(stacks []*Stack, keep func(*Stack) bool) []*Stack {
	out := make([]*Stack, len(stacks))
	for i, stack := range stacks {
		out[i] = stack
		if stack.holdsEnvValues() && !keep(stack) {
			masked := stack.withoutEnvValues()
			out[i] = &masked
		}
	}
	return out
}
