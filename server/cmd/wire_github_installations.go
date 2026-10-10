package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/workspace"
)

// githubInstallationScope holds every linked repository operation to its project's installation link, and an attach to
// the attacher's own GitHub view (ADR 0147).
type githubInstallationScope struct {
	appConfigs connectors.AppConfigStore
	projects   interface {
		Get(ctx context.Context, id string) (*workspace.Project, error)
	}
	// repos is the repository use-case's checks, set once it is built, since it reads GitHub through this scope.
	repos interface {
		RequireAssigned(ctx context.Context, workspaceID, owner, name string) error
		RequireAttach(ctx context.Context, workspaceID, owner, name string) error
	}
}

// Require is background work's check: once a key is set, the repository's installation must be linked to the
// project's workspace.
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
	return s.repos.RequireAssigned(ctx, project.WorkspaceID, owner, name)
}

// RequireAttach is an attach's check: a person attaches only what their own GitHub token lists, whatever the key;
// the server's own attaches keep Require's.
func (s *githubInstallationScope) RequireAttach(ctx context.Context, projectID, owner, name, connectorID string) error {
	if connectorID != githubConnectorID {
		return nil
	}
	project, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return err
	}
	return s.repos.RequireAttach(ctx, project.WorkspaceID, owner, name)
}
