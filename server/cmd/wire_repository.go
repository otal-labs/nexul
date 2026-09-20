package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/repository"
)

// githubConnectorID is the one connector repository scanning dispatches to today — ListInstallationRepos is a
// GitHub App concept with no GitLab/Gitea equivalent yet, so there is nothing to route per-repo (unlike
// gitProviderRouter's PR/webhook methods, which do vary per linked repo).
const githubConnectorID = "github"

// repositoryScanner adapts gitProviderRouter to repository.Scanner (ADR 0017): the repository domain never imports
// gitprovider, so this is the one place their types meet. It resolves the GitHub connector directly, skipping
// gitProviderRouter's per-repo workspace lookup — a scan targets a repo that usually isn't linked to a project
// yet, which is the whole point of scanning it.
type repositoryScanner struct {
	git        gitProviderRouter
	appConfigs connectors.AppConfigStore
}

func (s repositoryScanner) provider(ctx context.Context) (gitprovider.GitProvider, error) {
	return s.git.resolveConnector(ctx, githubConnectorID)
}

func (s repositoryScanner) ListInstallationRepos(ctx context.Context) ([]repository.Repo, error) {
	p, err := s.provider(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation repositories: %w", err)
	}
	repos, err := p.ListInstallationRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation repositories: %w", err)
	}
	out := make([]repository.Repo, 0, len(repos))
	for _, r := range repos {
		out = append(out, toRepositoryRepo(r))
	}
	return out, nil
}

func (s repositoryScanner) GetTree(ctx context.Context, owner, name, ref string) (string, []repository.TreeEntry, error) {
	p, err := s.provider(ctx)
	if err != nil {
		return "", nil, err
	}
	resolvedRef := ref
	if resolvedRef == "" {
		repo, err := p.GetRepo(ctx, owner, name)
		if err != nil {
			return "", nil, s.mapNotInstalled(ctx, fmt.Errorf("resolve default branch for %s/%s: %w", owner, name, err))
		}
		resolvedRef = repo.DefaultBranch
	}
	entries, err := p.GetTree(ctx, owner, name, resolvedRef)
	// GitHub answers 409 for the tree of a repository with no commits yet; that is an empty scan, not a failure.
	if errors.Is(err, apperrs.ErrConflict) {
		return resolvedRef, []repository.TreeEntry{}, nil
	}
	if err != nil {
		return "", nil, s.mapNotInstalled(ctx, fmt.Errorf("get tree %s/%s@%s: %w", owner, name, resolvedRef, err))
	}
	out := make([]repository.TreeEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, repository.TreeEntry{Path: e.Path, Type: e.Type})
	}
	return resolvedRef, out, nil
}

func (s repositoryScanner) GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error) {
	p, err := s.provider(ctx)
	if err != nil {
		return nil, err
	}
	b, err := p.GetFile(ctx, owner, name, ref, path)
	if err != nil {
		return nil, s.mapNotInstalled(ctx, fmt.Errorf("get file %s/%s@%s:%s: %w", owner, name, ref, path, err))
	}
	return b, nil
}

// mapNotInstalled turns a not-found/unauthorized scan failure into apperrs.ErrNotFound with the App's install
// URL in the message — to the caller, "repo doesn't exist" and "App not installed on it" look the same, and
// the install link is the fix for both.
func (s repositoryScanner) mapNotInstalled(ctx context.Context, err error) error {
	if !errors.Is(err, apperrs.ErrNotFound) && !errors.Is(err, apperrs.ErrUnauthorized) {
		return err
	}
	msg := "repository not found, or the GitHub App is not installed on it"
	if appCfg, cfgErr := s.appConfigs.GetAppConfig(ctx, githubConnectorID); cfgErr == nil && appCfg.AppSlug != "" {
		msg = fmt.Sprintf("%s: install it at https://github.com/apps/%s/installations/new", msg, appCfg.AppSlug)
	}
	return fmt.Errorf("%s: %w", msg, apperrs.ErrNotFound)
}

func toRepositoryRepo(r *gitprovider.Repo) repository.Repo {
	return repository.Repo{
		ID:            r.ID,
		Owner:         r.Owner,
		Name:          r.Name,
		FullName:      r.FullName,
		DefaultBranch: r.DefaultBranch,
		HTMLURL:       r.HTMLURL,
	}
}
