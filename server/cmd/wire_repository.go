package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/gitprovider/github"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/repository"
)

// githubConnectorID is the one connector repository discovery reads today.
const githubConnectorID = "github"

// repositoryScanner answers the repository domain's App-side questions (ADR 0017): which mode Nexul reads GitHub in,
// where the App installs, and the accounts behind installations for background work. Discovery never comes here.
type repositoryScanner struct {
	git        gitProviderRouter
	appConfigs connectors.AppConfigStore
}

func newRepositoryScanner(git gitProviderRouter, appConfigs connectors.AppConfigStore) repositoryScanner {
	return repositoryScanner{git: git, appConfigs: appConfigs}
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

// InstallationAccounts implements repository.AccountResolver as the App; before a key there is nothing to resolve.
func (s repositoryScanner) InstallationAccounts(ctx context.Context) ([]repository.Installation, error) {
	app, err := s.git.githubApp(ctx)
	if err != nil || app == nil {
		return nil, err
	}
	installs, err := app.InstallationAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", refusedApp(err))
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

// refusedApp turns GitHub refusing the App's key into a 403 naming it; a 401 would sign the browser out.
func refusedApp(err error) error {
	if !errors.Is(err, apperrs.ErrUnauthorized) {
		return err
	}
	return fmt.Errorf("GitHub refused the App's private key, replace it in Settings → Connectors → GitHub App: %w", apperrs.ErrForbidden)
}

// githubTokens is auth's person-token read (ADR 0147): their own token, refreshed, or the error saying how to connect.
type githubTokens interface {
	GitHubToken(ctx context.Context, userID string) (string, error)
}

// githubPeople opens each person's GitHub view with their own token (ADR 0147), never the App's or the connector's.
type githubPeople struct {
	tokens     githubTokens
	appConfigs connectors.AppConfigStore
	repos      *personRepoCache
}

func newGitHubPeople(tokens githubTokens, appConfigs connectors.AppConfigStore) githubPeople {
	return githubPeople{tokens: tokens, appConfigs: appConfigs, repos: &personRepoCache{byUser: map[string]personRepos{}}}
}

// GitHubView implements repository.People.
func (p githubPeople) GitHubView(ctx context.Context, userID string) (repository.GitHubView, error) {
	token, err := p.tokens.GitHubToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	client, err := githubClientFor(ctx, p.appConfigs, token)
	if err != nil {
		return nil, err
	}
	return personView{userID: userID, client: client, repos: p.repos, appConfigs: p.appConfigs}, nil
}

// githubClientFor is a GitHub client reading with token against the App's configured server.
func githubClientFor(ctx context.Context, appConfigs connectors.AppConfigStore, token string) (*github.Client, error) {
	cfg, err := appConfigs.GetAppConfig(ctx, githubConnectorID)
	if err != nil {
		return nil, fmt.Errorf("github app config: %w", err)
	}
	if cfg.BaseURL == "" {
		return github.New(token), nil
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse github base url: %w", err)
	}
	return github.New(token, github.WithBaseURL(u)), nil
}

// personView is GitHub as one person's token shows it: the intersection of what they can open and where the App is.
type personView struct {
	userID     string
	client     *github.Client
	repos      *personRepoCache
	appConfigs connectors.AppConfigStore
}

func (v personView) Repos(ctx context.Context, refresh bool) ([]repository.Repo, error) {
	return v.repos.get(v.userID, refresh, func() ([]repository.Repo, error) {
		repos, err := v.client.ListInstallationRepos(ctx)
		if err != nil {
			return nil, refusedPerson(err)
		}
		out := make([]repository.Repo, 0, len(repos))
		for _, r := range repos {
			out = append(out, toRepositoryRepo(r))
		}
		return out, nil
	})
}

func (v personView) Installations(ctx context.Context) ([]repository.Installation, error) {
	installs, err := v.client.ListInstallations(ctx)
	if err != nil {
		return nil, refusedPerson(err)
	}
	return toRepositoryInstallations(installs), nil
}

func (v personView) GetTree(ctx context.Context, owner, name, ref string) (string, []repository.TreeEntry, error) {
	resolvedRef := ref
	if resolvedRef == "" {
		repo, err := v.client.GetRepo(ctx, owner, name)
		if err != nil {
			return "", nil, v.mapNotInstalled(ctx, fmt.Errorf("resolve default branch for %s/%s: %w", owner, name, err))
		}
		resolvedRef = repo.DefaultBranch
	}
	entries, err := v.client.GetTree(ctx, owner, name, resolvedRef)
	// GitHub answers 409 for the tree of a repository with no commits yet; that is an empty scan, not a failure.
	if errors.Is(err, apperrs.ErrConflict) {
		return resolvedRef, []repository.TreeEntry{}, nil
	}
	if err != nil {
		return "", nil, v.mapNotInstalled(ctx, fmt.Errorf("get tree %s/%s@%s: %w", owner, name, resolvedRef, err))
	}
	out := make([]repository.TreeEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, repository.TreeEntry{Path: e.Path, Type: e.Type})
	}
	return resolvedRef, out, nil
}

func (v personView) GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error) {
	b, err := v.client.GetFile(ctx, owner, name, ref, path)
	if err != nil {
		return nil, v.mapNotInstalled(ctx, fmt.Errorf("get file %s/%s@%s:%s: %w", owner, name, ref, path, err))
	}
	return b, nil
}

// mapNotInstalled turns a not-found/unauthorized scan failure into apperrs.ErrNotFound with the App's install
// URL in the message: "you cannot open it" and "the App is not installed on it" look the same, and the link fixes one.
func (v personView) mapNotInstalled(ctx context.Context, err error) error {
	if !errors.Is(err, apperrs.ErrNotFound) && !errors.Is(err, apperrs.ErrUnauthorized) {
		return err
	}
	msg := "your GitHub account cannot open this repository, or the GitHub App is not installed on it"
	if appCfg, cfgErr := v.appConfigs.GetAppConfig(ctx, githubConnectorID); cfgErr == nil && appCfg.AppSlug != "" {
		msg = fmt.Sprintf("%s: install it at %s", msg, githubAppInstallURL(appCfg))
	}
	return fmt.Errorf("%s: %w", msg, apperrs.ErrNotFound)
}

// refusedPerson turns GitHub refusing a person's token into a 403 asking them to reconnect; a 401 would sign them out.
func refusedPerson(err error) error {
	if !errors.Is(err, apperrs.ErrUnauthorized) {
		return err
	}
	return fmt.Errorf("GitHub refused your GitHub sign-in, reconnect GitHub in Settings → Profile: %w", apperrs.ErrForbidden)
}

// personReposTTL keeps per-keystroke searches from walking every installation and page on GitHub each time.
const personReposTTL = time.Minute

// personRepoCache holds each person's last walk; one lock, so concurrent searches wait for a walk instead of starting
// their own. ponytail: one lock for everyone and an entry per person who ever searched; per-person locks and eviction
// if many people search at once.
type personRepoCache struct {
	mu     sync.Mutex
	byUser map[string]personRepos
}

type personRepos struct {
	at    time.Time
	repos []repository.Repo
}

func (c *personRepoCache) get(userID string, refresh bool, load func() ([]repository.Repo, error)) ([]repository.Repo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if hit, ok := c.byUser[userID]; ok && !refresh && time.Since(hit.at) < personReposTTL {
		return hit.repos, nil
	}
	repos, err := load()
	if err != nil {
		return nil, err
	}
	c.byUser[userID] = personRepos{at: time.Now(), repos: repos}
	return repos, nil
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
