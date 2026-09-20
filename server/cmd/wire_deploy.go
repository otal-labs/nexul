package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/workspace"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

// deployProjectStore adapts workspace projects to deploy's ProjectStore seam so deploy never imports workspace (ADR 0017).
type deployProjectStore struct {
	projects *storage.ProjectsRepo
}

func (a deployProjectStore) ProjectExists(ctx context.Context, projectID string) (bool, error) {
	_, err := a.projects.Get(ctx, projectID)
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (a deployProjectStore) RepoInProject(ctx context.Context, projectID, owner, name string) (bool, error) {
	repos, err := a.projects.ListRepos(ctx, projectID)
	if err != nil {
		return false, err
	}
	for _, r := range repos {
		if r.Owner == owner && r.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// LinkRepo attaches a repository under the GitHub connector. A repository belongs to exactly one project, so a
// conflict is fine only when it is already this project's (the wizard re-running its service step); otherwise
// it surfaces, naming the clash, instead of letting CreateStack fail with a misleading "not in project".
func (a deployProjectStore) LinkRepo(ctx context.Context, projectID, owner, name string) error {
	err := a.projects.AddRepo(ctx, projectID, workspace.RepoRef{Owner: owner, Name: name, FullName: owner + "/" + name, ConnectorID: "github"})
	if err == nil {
		return nil
	}
	if !errors.Is(err, apperrs.ErrConflict) {
		return err
	}
	in, lookupErr := a.RepoInProject(ctx, projectID, owner, name)
	if lookupErr != nil {
		return lookupErr
	}
	if in {
		return nil
	}
	return fmt.Errorf("%w: repository %s/%s already belongs to another project", apperrs.ErrConflict, owner, name)
}
