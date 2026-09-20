package main

import "context"

// runnerGitTokenAdapter adapts the GitHub connector's live token to runner's GitTokenLookup seam (ADR 0017: runner never imports connectors).
type runnerGitTokenAdapter struct {
	token func(ctx context.Context) (string, error)
}

func (a runnerGitTokenAdapter) RepoToken(ctx context.Context, _ string) (string, error) {
	return a.token(ctx)
}
