package main

import (
	"context"
	"strings"

	"github.com/otal-labs/nexul/internal/workspace"
)

// projectRepoLister is the one read pairing needs of a project's repositories.
type projectRepoLister interface {
	ListRepos(ctx context.Context, projectID string) ([]workspace.RepoRef, error)
}

// pairingProjectRemotes gives pairing a project's GitHub repositories as T3 normalizes a checkout's remote, so a run
// finds the project's checkout on any of its person's computers (ADR 0165).
type pairingProjectRemotes struct {
	projects projectRepoLister
}

func (a pairingProjectRemotes) ProjectRemotes(ctx context.Context, projectID string) ([]string, error) {
	repos, err := a.projects.ListRepos(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		if r.ConnectorID == "github" {
			out = append(out, "github.com/"+strings.ToLower(r.FullName))
		}
	}
	return out, nil
}
