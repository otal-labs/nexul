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
	"github.com/otal-labs/nexul/internal/workspace"
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

// githubInstallers trades an installer's code for their own token, which lists only the installations they can see
// and says whether they administer the installation's account.
type githubInstallers struct {
	oauth      connectors.OAuthClient
	appConfigs connectors.AppConfigStore
}

func (g githubInstallers) Installer(ctx context.Context, code string, installationID int64) (repository.Installer, error) {
	if g.oauth == nil {
		return repository.Installer{}, fmt.Errorf("%w: the GitHub connector has no OAuth client", apperrs.ErrInvalid)
	}
	ts, err := g.oauth.Exchange(ctx, code)
	if err != nil {
		return repository.Installer{}, err
	}
	cfg, err := g.appConfigs.GetAppConfig(ctx, githubConnectorID)
	if err != nil {
		return repository.Installer{}, err
	}
	var opts []github.Option
	if cfg.BaseURL != "" {
		u, err := url.Parse(cfg.BaseURL)
		if err != nil {
			return repository.Installer{}, fmt.Errorf("parse github base url: %w", err)
		}
		opts = append(opts, github.WithBaseURL(u))
	}
	client := github.New(ts.AccessToken, opts...)
	installs, err := client.ListInstallations(ctx)
	if err != nil {
		return repository.Installer{}, err
	}
	for _, inst := range installs {
		if inst.ID != installationID {
			continue
		}
		admin, err := client.AdministersAccount(ctx, inst)
		if err != nil {
			return repository.Installer{}, err
		}
		return repository.Installer{AccountID: inst.AccountID, AccountLogin: inst.AccountLogin, Admin: admin}, nil
	}
	return repository.Installer{}, fmt.Errorf("%w: the installer cannot see installation %d", apperrs.ErrForbidden, installationID)
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

// githubInstallationScope holds every linked repository operation to its project's installation assignment.
type githubInstallationScope struct {
	appConfigs connectors.AppConfigStore
	projects   interface {
		Get(ctx context.Context, id string) (*workspace.Project, error)
	}
	// assigned is the repository use-case's check, set once it is built, since it reads GitHub through this scope.
	assigned interface {
		RequireAssigned(ctx context.Context, workspaceID, owner, name string) error
	}
}

func (s *githubInstallationScope) Require(ctx context.Context, projectID, owner, name, connectorID string) error {
	if connectorID != githubConnectorID {
		return nil
	}
	cfg, err := s.appConfigs.GetAppConfig(ctx, githubConnectorID)
	if err != nil {
		return err
	}
	if cfg.PrivateKey == "" {
		return nil
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return err
	}
	return s.assigned.RequireAssigned(ctx, project.WorkspaceID, owner, name)
}
