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
// what every member may read (chat, the inbox, People). A caller outside the workspace gets ErrNotFound, so a
// direct fetch never confirms what the workspace holds; a member without the action gets ErrForbidden. With no
// project named a Restricted member holds no project area. A call with no actor is the server's own (event
// consumers, workers) and passes: every adapter attaches one.
func (s *Service) Require(ctx context.Context, workspaceID string, action permissions.Action) error {
	return s.require(ctx, workspaceID, "", action)
}

func (s *Service) require(ctx context.Context, workspaceID, projectID string, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return defaultAutomationInside(ctx, workspaceID)
	}
	ws := s.layers(ctx, userID, workspaceID, projectID)
	if !ws.owner && ws.hidden() {
		return apperrs.ErrNotFound
	}
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

// RequireProject is Require inside projectID, in the workspace it belongs to. An unknown project, or one hidden from
// a Restricted member, is ErrNotFound; the empty action asks whether the caller may open the project at all.
func (s *Service) RequireProject(ctx context.Context, projectID string, action permissions.Action) error {
	if _, checked := caller(ctx); !checked && defaultAutomationWorkspace(ctx) == "" {
		return nil
	}
	workspaceID, err := s.projectWorkspace(ctx, projectID)
	if err != nil {
		return err
	}
	return s.require(ctx, workspaceID, projectID, action)
}

// RequireAnywhere checks an instance-level action (runners, topology, a stack outside every project): the
// caller must hold it in at least one workspace they belong to, the way the sidebar shows the area.
func (s *Service) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	userID, checked := caller(ctx)
	if !checked {
		return nil
	}
	held, err := s.HoldsAnywhere(ctx, userID, action)
	if err != nil {
		return err
	}
	if !held {
		return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
	}
	return nil
}

// HoldsAnywhere reports whether userID holds action in at least one workspace they belong to unrestricted; an Owner
// of any workspace holds every action, which is all instance-level power there is (ADR 0088, ADR 0097).
func (s *Service) HoldsAnywhere(ctx context.Context, userID string, action permissions.Action) (bool, error) {
	workspaceIDs, err := s.workspacesOf(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, workspaceID := range workspaceIDs {
		if s.HasPermission(ctx, userID, workspaceID, action, "", "") {
			return true, nil
		}
	}
	return false, nil
}

// PermissionsAnywhere is every action userID holds in at least one workspace they belong to unrestricted, in catalog
// order: what an instance-level area answers to, so a client shows the same areas the server lets through.
func (s *Service) PermissionsAnywhere(ctx context.Context, userID string) ([]string, error) {
	workspaceIDs, err := s.workspacesOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	held := map[string]bool{}
	for _, workspaceID := range workspaceIDs {
		for _, a := range s.WorkspacePermissions(ctx, userID, workspaceID) {
			held[a] = true
		}
	}
	out := make([]string, 0, len(held))
	for _, a := range permissions.AllActions() {
		if held[string(a)] {
			out = append(out, string(a))
		}
	}
	return out, nil
}

func (s *Service) workspacesOf(ctx context.Context, userID string) ([]string, error) {
	if userID == "" || s.scopes == nil {
		return nil, nil
	}
	workspaceIDs, err := s.scopes.UnrestrictedWorkspaceIDsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces of %s: %w", userID, err)
	}
	return workspaceIDs, nil
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

// defaultAutomationInside holds a shipped default automation, whose scopes are its whole grant, to its own workspace.
func defaultAutomationInside(ctx context.Context, workspaceID string) error {
	home := defaultAutomationWorkspace(ctx)
	if home == "" || home == workspaceID {
		return nil
	}
	return fmt.Errorf("%w: workspace %s", apperrs.ErrNotFound, workspaceID)
}

// defaultAutomationWorkspace is the workspace of the creator-less automation on ctx, or "" for any other caller.
func defaultAutomationWorkspace(ctx context.Context) string {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID != "" || actor.Automation == nil {
		return ""
	}
	return actor.Automation.WorkspaceID
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
