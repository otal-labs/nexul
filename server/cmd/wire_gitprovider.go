package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/gitprovider/github"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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

// openPRLister adapts tickets.PRRef to gitprovider.PRRef for the PR state sweep.
type openPRLister struct {
	svc *tickets.Service
}

func (l openPRLister) ListOpenPRs(ctx context.Context) ([]gitprovider.PRRef, error) {
	refs, err := l.svc.OpenPRs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]gitprovider.PRRef, 0, len(refs))
	for _, r := range refs {
		out = append(out, gitprovider.PRRef{Owner: r.Owner, Repo: r.Repo, Number: r.Number})
	}
	return out, nil
}

// sweepPRReader reads a linked PR whether or not its repo is attached to a project, since a ticket can link any
// repo the installation reaches.
type sweepPRReader struct {
	router gitProviderRouter
}

func (r sweepPRReader) GetPR(ctx context.Context, owner, name string, number int) (*gitprovider.PR, error) {
	connectorID := "github"
	ref, err := r.router.workspace.GetRepoByFullName(ctx, owner, name)
	if err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("resolve git provider for %s/%s: %w", owner, name, err)
	}
	if err == nil {
		connectorID = ref.ConnectorID
	}
	p, err := r.router.resolveConnector(ctx, connectorID)
	if err != nil {
		return nil, err
	}
	return p.GetPR(ctx, owner, name, number)
}

// gitProviderRouter resolves, per repo, which connector backs it; built fresh per call, never cached.
type gitProviderRouter struct {
	workspace  gitRepoResolver
	connectors gitTokenResolver
	appConfigs connectors.AppConfigStore
}

// gitRepoResolver is the slice of workspace.Service the router needs (ADR 0017), narrow enough to fake in tests.
type gitRepoResolver interface {
	GetRepoByFullName(ctx context.Context, owner, name string) (workspace.RepoRef, error)
}

// gitTokenResolver is the slice of connectors.Service the router needs (ADR 0017).
type gitTokenResolver interface {
	AccessToken(ctx context.Context, connectorID string) (string, error)
}

// resolve builds a fresh GitProvider for owner/name's linked connector; failures are scoped to this call only.
func (g gitProviderRouter) resolve(ctx context.Context, owner, name string) (gitprovider.GitProvider, error) {
	ref, err := g.workspace.GetRepoByFullName(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("resolve git provider for %s/%s: %w", owner, name, err)
	}
	p, err := g.resolveConnector(ctx, ref.ConnectorID)
	if err != nil {
		return nil, fmt.Errorf("resolve git provider for %s/%s: %w", owner, name, err)
	}
	return p, nil
}

// resolveConnector builds a fresh GitProvider for connectorID directly, skipping the per-repo workspace lookup;
// used for installation-wide operations that have no linked repo yet (ListInstallationRepos).
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

func (g gitProviderRouter) DeleteWebhook(ctx context.Context, owner, name, hookID string) error {
	p, err := g.resolve(ctx, owner, name)
	if err != nil {
		return err
	}
	return p.DeleteWebhook(ctx, owner, name, hookID)
}

func (g gitProviderRouter) ListInstallationRepos(ctx context.Context) ([]*gitprovider.Repo, error) {
	p, err := g.resolveConnector(ctx, "github")
	if err != nil {
		return nil, err
	}
	return p.ListInstallationRepos(ctx)
}

func (g gitProviderRouter) ListInstallations(ctx context.Context) ([]*gitprovider.Installation, error) {
	p, err := g.resolveConnector(ctx, "github")
	if err != nil {
		return nil, err
	}
	return p.ListInstallations(ctx)
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
