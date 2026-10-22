package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/workspace"
)

// openProjects lets the caller open only the projects it names.
type openProjects map[string]bool

func (o openProjects) RequireProject(_ context.Context, projectID string, _ permissions.Action) error {
	if !o[projectID] {
		return apperrs.ErrNotFound
	}
	return nil
}

type cloneCredentials map[string]runner.CloneCredentials

func (c cloneCredentials) RepoToken(_ context.Context, repo string) (runner.CloneCredentials, error) {
	return c[repo], nil
}

// TestCloneTokens_OnlyForARepositoryOfAProjectThePersonOpens: a token reaches a person's computer only for a repository
// of a project they may open, and only one minted for that repository, never the instance's connector token.
func TestCloneTokens_OnlyForARepositoryOfAProjectThePersonOpens(t *testing.T) {
	t.Parallel()
	tokens := cloneTokens{
		repos: &fakeGitRepoResolver{repos: map[string]workspace.RepoRef{
			"acme/api":    {ProjectID: "p-api"},
			"acme/secret": {ProjectID: "p-secret"},
			"acme/legacy": {ProjectID: "p-api"},
		}},
		access: openProjects{"p-api": true},
		git: cloneCredentials{
			"acme/api":    {Token: "repo-tok"},
			"acme/secret": {Token: "secret-tok"},
			"acme/legacy": {Token: "connector-tok", AllowFallback: true},
		},
	}
	tests := []struct {
		repository string
		want       string
	}{
		{"acme/api", "repo-tok"},
		{"acme/secret", ""},
		{"acme/legacy", ""},
		{"acme/unlinked", ""},
	}
	for _, tt := range tests {
		t.Run(tt.repository, func(t *testing.T) {
			t.Parallel()
			got, err := tokens.token(t.Context(), tt.repository)
			if tt.want == "" {
				require.Error(t, err)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
