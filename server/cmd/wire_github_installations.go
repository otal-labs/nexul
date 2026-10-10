package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/gitprovider/github"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// githubOAuth is the GitHub connector's OAuth client from the wired registry, nil when it has none.
func githubOAuth(registry []connectors.Connector) connectors.OAuthClient {
	for _, c := range registry {
		if c.ID == githubConnectorID {
			return c.OAuth
		}
	}
	return nil
}

// githubInstallers trades an installer's code for their own token, which lists only the installations they can see.
type githubInstallers struct {
	oauth      connectors.OAuthClient
	appConfigs connectors.AppConfigStore
}

func (g githubInstallers) InstallerAccount(ctx context.Context, code string, installationID int64) (string, error) {
	if g.oauth == nil {
		return "", fmt.Errorf("%w: the GitHub connector has no OAuth client", apperrs.ErrInvalid)
	}
	ts, err := g.oauth.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	cfg, err := g.appConfigs.GetAppConfig(ctx, githubConnectorID)
	if err != nil {
		return "", err
	}
	var opts []github.Option
	if cfg.BaseURL != "" {
		u, err := url.Parse(cfg.BaseURL)
		if err != nil {
			return "", fmt.Errorf("parse github base url: %w", err)
		}
		opts = append(opts, github.WithBaseURL(u))
	}
	installs, err := github.New(ts.AccessToken, opts...).ListInstallations(ctx)
	if err != nil {
		return "", err
	}
	for _, inst := range installs {
		if inst.ID == installationID {
			return inst.AccountLogin, nil
		}
	}
	return "", fmt.Errorf("%w: the installer cannot see installation %d", apperrs.ErrForbidden, installationID)
}

// installationClaimer adapts the repository claim to auth's callback and lands the installer in the workspace.
type installationClaimer struct {
	svc        *repository.Service
	workspaces interface {
		Get(ctx context.Context, id string) (*tenancy.Workspace, error)
	}
}

func (c installationClaimer) ClaimsState(state string) bool {
	return c.svc.ClaimsState(state)
}

func (c installationClaimer) ClaimInstallation(ctx context.Context, state, code, installationID string) (string, error) {
	workspaceID, err := c.svc.ClaimInstallation(ctx, state, code, installationID)
	if err != nil {
		return "", err
	}
	landing := "/"
	if ws, err := c.workspaces.Get(ctx, workspaceID); err == nil {
		landing += ws.Slug
	}
	return landing, nil
}
