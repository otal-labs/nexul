package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/gitprovider/github"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/githubapp"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// ticketLinker adapts tickets.PRRef to gitprovider.PRRef; identical types, kept decoupled so tickets and gitprovider stay independent.
type ticketLinker struct {
	svc *tickets.Service
}

func (l ticketLinker) LinkPR(ctx context.Context, ticketID string, ref gitprovider.PRRef) error {
	return l.svc.LinkPR(ctx, ticketID, tickets.PRRef{Owner: ref.Owner, Repo: ref.Repo, Number: ref.Number, Title: ref.Title, SHA: ref.SHA})
}

// gitProviderRouter resolves, per repo, which connector backs it; built fresh per call except the App's cached tokens.
type gitProviderRouter struct {
	workspace  gitRepoResolver
	connectors gitTokenResolver
	appConfigs connectors.AppConfigStore
	// apps is nil in tests that never read GitHub as the App.
	apps  *github.AppCache
	scope *githubInstallationScope
}

// gitRepoResolver is the slice of workspace.Service the router needs (ADR 0017), narrow enough to fake in tests.
type gitRepoResolver interface {
	GetRepoByFullName(ctx context.Context, owner, name string) (workspace.RepoRef, error)
}

// gitTokenResolver is the slice of connectors.Service the router needs (ADR 0017).
type gitTokenResolver interface {
	AccessToken(ctx context.Context, connectorID string) (string, error)
}

// githubApp is the App Nexul reads GitHub as once its private key is set (ADR 0144), else nil.
func (g gitProviderRouter) githubApp(ctx context.Context) (*github.App, error) {
	if g.apps == nil {
		return nil, nil
	}
	cfg, err := g.appConfigs.GetAppConfig(ctx, githubConnectorID)
	if err != nil {
		return nil, fmt.Errorf("github app config: %w", err)
	}
	if cfg.PrivateKey == "" {
		return nil, nil
	}
	return g.apps.Get(cfg.ClientID, cfg.PrivateKey, cfg.BaseURL)
}

// githubForRepo reads owner/name, linked or not, as the App's installation once a key is set, else as the connector.
func (g gitProviderRouter) githubForRepo(ctx context.Context, owner, name string) (gitprovider.GitProvider, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	app, err := g.githubApp(ctx)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return g.resolveConnector(ctx, githubConnectorID)
	}
	return app.ForRepo(ctx, owner, name)
}

// RepoToken is the token a runner clones fullName ("owner/name") with: once a key is set, one minted for that
// repository alone with contents:read, and only while its installation is assigned to the project's workspace.
func (g gitProviderRouter) RepoToken(ctx context.Context, fullName string) (runner.CloneCredentials, error) {
	owner, name, ok := strings.Cut(fullName, "/")
	if !ok {
		return runner.CloneCredentials{}, fmt.Errorf("%w: repository must be owner/name", apperrs.ErrInvalid)
	}
	app, err := g.cloneScope(ctx, owner, name)
	if err != nil {
		return runner.CloneCredentials{}, err
	}
	if app == nil {
		token, err := g.connectors.AccessToken(ctx, githubConnectorID)
		return runner.CloneCredentials{Token: token, AllowFallback: true}, err
	}
	token, err := app.CloneToken(ctx, owner, name)
	return runner.CloneCredentials{Token: token}, err
}

// cloneScope is RepoToken's check without the token: the App when it reads GitHub and owner/name is linked to a
// project whose workspace is assigned its installation, nil before a key is set.
func (g gitProviderRouter) cloneScope(ctx context.Context, owner, name string) (*github.App, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	app, err := g.githubApp(ctx)
	if err != nil || app == nil {
		return nil, err
	}
	ref, err := g.workspace.GetRepoByFullName(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("resolve git provider for %s/%s: %w", owner, name, err)
	}
	if ref.ConnectorID != githubConnectorID {
		return nil, fmt.Errorf("%w: %s/%s is not a GitHub repository", apperrs.ErrInvalid, owner, name)
	}
	if g.scope != nil {
		if err := g.scope.Require(ctx, ref.ProjectID, owner, name, ref.ConnectorID); err != nil {
			return nil, err
		}
	}
	return app, nil
}

// resolve builds a fresh GitProvider for owner/name's linked connector; failures are scoped to this call only.
func (g gitProviderRouter) resolve(ctx context.Context, owner, name string) (gitprovider.GitProvider, error) {
	if err := githubapp.ValidateRepository(owner, name); err != nil {
		return nil, err
	}
	ref, err := g.workspace.GetRepoByFullName(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("resolve git provider for %s/%s: %w", owner, name, err)
	}
	if g.scope != nil {
		if err := g.scope.Require(ctx, ref.ProjectID, owner, name, ref.ConnectorID); err != nil {
			return nil, err
		}
	}
	if ref.ConnectorID == githubConnectorID {
		p, err := g.githubForRepo(ctx, owner, name)
		if err != nil {
			return nil, fmt.Errorf("resolve git provider for %s/%s: %w", owner, name, err)
		}
		return p, nil
	}
	p, err := g.resolveConnector(ctx, ref.ConnectorID)
	if err != nil {
		return nil, fmt.Errorf("resolve git provider for %s/%s: %w", owner, name, err)
	}
	return p, nil
}

// resolveConnector builds a fresh GitProvider for connectorID directly, skipping the per-repo workspace lookup.
func (g gitProviderRouter) resolveConnector(ctx context.Context, connectorID string) (gitprovider.GitProvider, error) {
	token, err := g.connectors.AccessToken(ctx, connectorID)
	if err != nil {
		return nil, fmt.Errorf("connector %s has no live token: %w", connectorID, err)
	}
	appCfg, err := g.appConfigs.GetAppConfig(ctx, connectorID)
	if err != nil {
		return nil, fmt.Errorf("connector %s app config: %w", connectorID, err)
	}
	switch connectorID {
	case "github":
		var opts []github.Option
		if appCfg.BaseURL != "" {
			u, err := url.Parse(appCfg.BaseURL)
			if err != nil {
				return nil, fmt.Errorf("parse connector %s base url: %w", connectorID, err)
			}
			opts = append(opts, github.WithBaseURL(u))
		}
		return github.New(token, opts...), nil
	default:
		return nil, fmt.Errorf("%w: connector %q has no git provider implementation", apperrs.ErrInvalid, connectorID)
	}
}

func (g gitProviderRouter) GetRepo(ctx context.Context, owner, name string) (*gitprovider.Repo, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return p.GetRepo(ctx, owner, name)
}

func (g gitProviderRouter) ListPRs(ctx context.Context, owner, name string, opts gitprovider.PROpts) ([]*gitprovider.PR, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return p.ListPRs(ctx, owner, name, opts)
}

func (g gitProviderRouter) GetPR(ctx context.Context, owner, name string, number int) (*gitprovider.PR, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return p.GetPR(ctx, owner, name, number)
}

func (g gitProviderRouter) PRsForCommit(ctx context.Context, owner, name, sha string) ([]*gitprovider.PR, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return p.PRsForCommit(ctx, owner, name, sha)
}

func (g gitProviderRouter) CreateWebhook(ctx context.Context, owner, name string, cfg gitprovider.WebhookConfig) (string, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return "", err
	}
	return p.CreateWebhook(ctx, owner, name, cfg)
}

func (g gitProviderRouter) ListWebhooks(ctx context.Context, owner, name string) ([]gitprovider.Webhook, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return p.ListWebhooks(ctx, owner, name)
}

func (g gitProviderRouter) DeleteWebhook(ctx context.Context, owner, name, hookID string) error {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return err
	}
	return p.DeleteWebhook(ctx, owner, name, hookID)
}

func (g gitProviderRouter) GetTree(ctx context.Context, owner, name, ref string) ([]gitprovider.TreeEntry, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return p.GetTree(ctx, owner, name, ref)
}

func (g gitProviderRouter) GetFile(ctx context.Context, owner, name, ref, path string) ([]byte, error) {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return p.GetFile(ctx, owner, name, ref, path)
}
