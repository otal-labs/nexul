package access

import (
	"context"
	"errors"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// SetScopes wires the project and membership lookups the Require checks resolve through.
func (s *Service) SetScopes(sc Scopes) {
	s.scopes = sc
}

// Require checks the calling actor holds action in workspaceID; an empty action asks only for membership, for
// what every member may read (chat, the inbox, the project list). A caller outside the workspace gets ErrNotFound,
// so a direct fetch never confirms what the workspace holds; a member without the action gets ErrForbidden.
// A call with no actor is the server's own (event consumers, workers) and passes: every adapter attaches one.
func (s *Service) Require(ctx context.Context, workspaceID string, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return nil
	}
	ws := s.workspaceLayers(ctx, userID, workspaceID)
	allowed := ws.owner || ws.has(action)
	if action == "" {
		allowed = ws.member
	}
	if userID != "" && allowed {
		return nil
	}
	if !ws.member {
		return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
	}
	return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
}

// RequireProject is Require in the workspace projectID belongs to; an unknown project is ErrNotFound.
func (s *Service) RequireProject(ctx context.Context, projectID string, action permissions.Action) error {
	if _, checked := caller(ctx); !checked {
		return nil
	}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		return err
	}
	return s.Require(ctx, workspaceID, action)
}

// RequireAnywhere checks an instance-level action (runners, topology, a stack outside every project): the
// caller must hold it in at least one workspace they belong to, the way the sidebar shows the area.
func (s *Service) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return nil
	}
	if userID != "" && s.scopes != nil {
		workspaceIDs, err := s.scopes.WorkspaceIDsForUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("list workspaces of %s: %w", userID, err)
		}
		for _, workspaceID := range workspaceIDs {
			if s.HasPermission(ctx, userID, workspaceID, action, "", "") {
				return nil
			}
		}
	}
	return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
}

// caller is the person a check is about; checked is false for the server's own calls and for a shipped default
// automation, which has no creator to resolve and whose token scopes the gateway already held it to.
func caller(ctx context.Context) (userID string, checked bool) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok {
		return "", false
	}
	if actor.ID == "" && actor.Automation != nil {
		return "", false
	}
	return actor.ID, true
}

func (s *Service) projectWorkspace(ctx context.Context, projectID string) (string, error) {
	if s.scopes == nil {
		return "", fmt.Errorf("%w: no project lookup wired", apperrs.ErrForbidden)
	}
	workspaceID, err := s.scopes.WorkspaceIDForProject(ctx, projectID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", fmt.Errorf("%w: project %s", apperrs.ErrNotFound, projectID)
	}
	if err != nil {
		return "", fmt.Errorf("resolve workspace of project %s: %w", projectID, err)
	}
	return workspaceID, nil
}
