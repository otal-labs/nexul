package deploy

import (
	"context"
	"fmt"
	"strings"
)

// TestTargetKind says whether a tester lands on the branch's own preview or on an environment other work shares.
type TestTargetKind string

const (
	TestTargetPreview TestTargetKind = "preview"
	TestTargetShared  TestTargetKind = "shared"
)

// BranchRef is a repository branch whose work a tester is asked to check.
type BranchRef struct {
	Owner  string
	Repo   string
	Branch string
}

// TestTarget is where a tester checks a branch's work; an empty URL means no deployment is safe to test on.
type TestTarget struct {
	URL    string         `json:"url"`
	Kind   TestTargetKind `json:"kind,omitempty"`
	Branch string         `json:"branch,omitempty"`
}

// ResolveTestTarget offers the first branch's preview, else a shared test environment, and never production.
func (s *Service) ResolveTestTarget(ctx context.Context, projectID string, branches []BranchRef) (TestTarget, error) {
	stacks, err := s.ListStacks(ctx, projectID)
	if err != nil {
		return TestTarget{}, fmt.Errorf("resolve test target: %w", err)
	}
	return testTarget(stacks, branches), nil
}

func testTarget(stacks []*Stack, branches []BranchRef) TestTarget {
	for _, b := range branches {
		if target, ok := previewTarget(stacks, b); ok {
			return target
		}
	}
	for _, stack := range stacks {
		for _, rule := range stack.BranchDeployRules {
			if rule.IsWildcard() || !testable(stack, rule, rule.Pattern) {
				continue
			}
			return TestTarget{URL: ruleURL(rule, rule.Pattern), Kind: TestTargetShared, Branch: rule.Pattern}
		}
	}
	return TestTarget{}
}

func previewTarget(stacks []*Stack, b BranchRef) (TestTarget, bool) {
	for _, stack := range stacks {
		bs := stack.BuildSource
		if bs == nil || !strings.EqualFold(bs.RepoOwner, b.Owner) || !strings.EqualFold(bs.RepoName, b.Repo) {
			continue
		}
		rule, ok := matchingRule(stack.BranchDeployRules, b.Branch)
		if !ok || !rule.IsWildcard() || !testable(stack, rule, b.Branch) {
			continue
		}
		return TestTarget{URL: ruleURL(rule, b.Branch), Kind: TestTargetPreview, Branch: b.Branch}, true
	}
	return TestTarget{}, false
}

// testable refuses production: the default branch's deployment, the base redeployed in place, or its network with nothing overridden.
func testable(stack *Stack, rule BranchDeployRule, branch string) bool {
	if rule.HostnameTemplate == "" || !rule.DerivesClone() || branch == defaultBranch(stack) {
		return false
	}
	return rule.DockerNetwork != stack.DefaultNetwork() || len(rule.Overrides) > 0
}

// defaultBranch mirrors the project wizard's fallback when the build source names no branch.
func defaultBranch(stack *Stack) string {
	if stack.BuildSource != nil && stack.BuildSource.Branch != "" {
		return stack.BuildSource.Branch
	}
	return "main"
}

func ruleURL(rule BranchDeployRule, branch string) string {
	return "https://" + strings.ReplaceAll(rule.HostnameTemplate, "{branch}", rule.HostnameLabel(branch))
}
