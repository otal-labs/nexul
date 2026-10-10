package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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
// yet, which is the whole point of scanning it. It is also the repository domain's InstallationLister.
type repositoryScanner struct {
	git        gitProviderRouter
	appConfigs connectors.AppConfigStore
	repos      *installationRepoCache
}

func newRepositoryScanner(git gitProviderRouter, appConfigs connectors.AppConfigStore) repositoryScanner {
	return repositoryScanner{git: git, appConfigs: appConfigs, repos: &installationRepoCache{}}
}

// installationReposTTL keeps per-keystroke searches from walking every installation and page on GitHub each time.
const installationReposTTL = time.Minute

// installationRepoCache holds one walk at a time so concurrent searches wait for it instead of starting their own.
type installationRepoCache struct {
	mu    sync.Mutex
	at    time.Time
	asApp bool
	repos []repository.Repo
}

// get serves the last walk while it is fresh and was read the same way, as the App or as the connected account.
func (c *installationRepoCache) get(refresh, asApp bool, load func() ([]repository.Repo, error)) ([]repository.Repo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !refresh && !c.at.IsZero() && c.asApp == asApp && time.Since(c.at) < installationReposTTL {
		return c.repos, nil
	}
	repos, err := load()
	if err != nil {
		return nil, err
	}
	c.repos, c.at, c.asApp = repos, time.Now(), asApp
	return repos, nil
}

// ReadsAsApp implements repository.InstallationLister: true once the App's private key is set.
func (s repositoryScanner) ReadsAsApp(ctx context.Context) (bool, error) {
	app, err := s.git.githubApp(ctx)
	return app != nil, err
}

// InstallURL implements repository.InstallationLister: GitHub's page for installing the App on another account.
func (s repositoryScanner) InstallURL(ctx context.Context) (string, error) {
	cfg, err := s.appConfigs.GetAppConfig(ctx, githubConnectorID)
	if err != nil || cfg.AppSlug == "" {
		return "", err
	}
	return githubAppInstallURL(cfg), nil
}

func githubAppInstallURL(cfg connectors.AppConfig) string {
	return "https://github.com/apps/" + cfg.AppSlug + "/installations/new"
}

func (s repositoryScanner) ListInstallationRepos(ctx context.Context, refresh bool) ([]repository.Repo, error) {
	asApp, err := s.ReadsAsApp(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation repositories: %w", err)
	}
	return s.repos.get(refresh, asApp, func() ([]repository.Repo, error) { return s.listInstallationRepos(ctx, asApp) })
}

func (s repositoryScanner) listInstallationRepos(ctx context.Context, asApp bool) ([]repository.Repo, error) {
	p, err := s.git.installations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation repositories: %w", err)
	}
	repos, err := p.ListInstallationRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installation repositories: %w", refused(err, asApp))
	}
	out := make([]repository.Repo, 0, len(repos))
	for _, r := range repos {
		out = append(out, toRepositoryRepo(r))
	}
	return out, nil
}

func (s repositoryScanner) ListInstallations(ctx context.Context) ([]repository.Installation, error) {
	asApp, err := s.ReadsAsApp(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	p, err := s.git.installations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	installs, err := p.ListInstallations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", refused(err, asApp))
	}
	return toRepositoryInstallations(installs), nil
}

// InstallationAccounts implements repository.AccountResolver: as the App, only the installations list; before a key,
// what the connected account sees.
func (s repositoryScanner) InstallationAccounts(ctx context.Context) ([]repository.Installation, error) {
	app, err := s.git.githubApp(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	if app == nil {
		return s.ListInstallations(ctx)
	}
	installs, err := app.InstallationAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", refused(err, true))
	}
	return toRepositoryInstallations(installs), nil
}

// AccountOf implements repository.AccountResolver; only the App answers it, being the only reader that needs it.
func (s repositoryScanner) AccountOf(ctx context.Context, owner, name string) (int64, error) {
	app, err := s.git.githubApp(ctx)
	if err != nil {
		return 0, err
	}
	if app == nil {
		return 0, fmt.Errorf("%w: GitHub is read as the connected account", apperrs.ErrNotFound)
	}
	return app.AccountOf(ctx, owner, name)
}

func toRepositoryInstallations(installs []*gitprovider.Installation) []repository.Installation {
	out := make([]repository.Installation, 0, len(installs))
	for _, i := range installs {
		out = append(out, repository.Installation{
			ID: i.ID, AccountID: i.AccountID, AccountLogin: i.AccountLogin, AccountType: i.AccountType,
			AccountAvatarURL: i.AccountAvatarURL, RepositorySelection: i.RepositorySelection,
			RepositoryCount: i.RepositoryCount, HTMLURL: i.HTMLURL, Problem: i.Problem,
		})
	}
	return out
}

func (s repositoryScanner) GetTree(ctx context.Context, owner, name, ref string) (string, []repository.TreeEntry, error) {
	p, err := s.git.githubForRepo(ctx, owner, name)
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
	p, err := s.git.githubForRepo(ctx, owner, name)
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
		msg = fmt.Sprintf("%s: install it at %s", msg, githubAppInstallURL(appCfg))
	}
	return fmt.Errorf("%s: %w", msg, apperrs.ErrNotFound)
}

// refused turns GitHub refusing Nexul's credential into a 403 naming it; a 401 would sign the browser out.
func refused(err error, asApp bool) error {
	if !errors.Is(err, apperrs.ErrUnauthorized) {
		return err
	}
	if asApp {
		return fmt.Errorf("GitHub refused the App's private key, replace it in Settings → Connectors → GitHub App: %w", apperrs.ErrForbidden)
	}
	return fmt.Errorf("GitHub refused the connector's token, reconnect GitHub in Settings → Connectors: %w", apperrs.ErrForbidden)
}

func toRepositoryRepo(r *gitprovider.Repo) repository.Repo {
	return repository.Repo{
		ID:            r.ID,
		Owner:         r.Owner,
		Name:          r.Name,
		FullName:      r.FullName,
		DefaultBranch: r.DefaultBranch,
		HTMLURL:       r.HTMLURL,
		Provider:      githubConnectorID,
		AccountID:     r.AccountID,
	}
}
