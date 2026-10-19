package tenancy

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// ViewedMember is who a viewer may view Nexul as, in the one workspace the view is held to.
type ViewedMember struct {
	WorkspaceID string `json:"workspace_id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url"`
}

// CheckViewAs is the one check behind starting a view and every request made in one (ADR 0160): viewerID holds
// members:view_as in workspaceID, targetID is another member there who is not an Owner, and the roles the target holds
// there give nothing viewerID could not give.
func (s *Service) CheckViewAs(ctx context.Context, viewerID, workspaceID, targetID string) (*ViewedMember, error) {
	workspaceID, targetID = strings.TrimSpace(workspaceID), strings.TrimSpace(targetID)
	if err := s.requireWorkspacePermission(ctx, viewerID, workspaceID, permissions.MembersViewAs); err != nil {
		return nil, err
	}
	member, err := s.Member(ctx, workspaceID, targetID)
	if err != nil {
		return nil, err
	}
	if targetID == viewerID {
		return nil, fmt.Errorf("%w: You can't view as yourself", apperrs.ErrInvalid)
	}
	owner, err := s.ownerRoleAmong(ctx, workspaceID, member.Held())
	if err != nil {
		return nil, err
	}
	if owner {
		return nil, fmt.Errorf("%w: An Owner sees everything in a workspace, so there's nothing to check", apperrs.ErrInvalid)
	}
	var given permissions.Set
	for _, roleID := range member.Held() {
		perms, err := s.roleNames.RolePermissions(ctx, workspaceID, roleID)
		if err != nil {
			return nil, fmt.Errorf("read role %s: %w", roleID, err)
		}
		given = permissions.SetOf(append(given.Actions(), perms...)...)
	}
	account, err := s.accounts.Account(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("get account %s: %w", targetID, err)
	}
	person := toPerson(account)
	name := person.DisplayName
	if name == "" {
		name = person.Login
	}
	// Hold-what-you-give (ADR 0088): a view must not show what the viewer couldn't hand out themselves.
	missing := given.Except(permissions.SetOfStrings(s.wsPerms.GrantablePermissions(ctx, viewerID, workspaceID)))
	if len(missing) > 0 {
		return nil, ViewAsRefusal(fmt.Sprintf("%s's roles give permissions you don't hold (%s), so you can't view as them", name, labels(missing)))
	}
	return &ViewedMember{WorkspaceID: workspaceID, UserID: targetID, Name: name, AvatarURL: person.AvatarURL}, nil
}

// ViewAsRefusal is a forbidden answer about a view that carries its sentence, which the web shows instead of the message.
func ViewAsRefusal(sentence string) error {
	return viewAsRefusal{sentence}
}

type viewAsRefusal struct{ sentence string }

func (e viewAsRefusal) Error() string     { return apperrs.ErrForbidden.Error() + ": " + e.sentence }
func (e viewAsRefusal) Unwrap() error     { return apperrs.ErrForbidden }
func (e viewAsRefusal) ErrorDetails() any { return map[string]string{"sentence": e.sentence} }

func labels(actions permissions.Set) string {
	byValue := map[permissions.Action]string{}
	for _, info := range permissions.Catalog() {
		byValue[info.Value] = info.Label
	}
	out := make([]string, 0, len(actions))
	for _, a := range actions {
		out = append(out, cmp.Or(byValue[a], string(a)))
	}
	return strings.Join(out, ", ")
}

// StartViewAs runs the check as the caller; the audit row of the route that calls it records who viewed as whom.
func (s *Service) StartViewAs(ctx context.Context, viewerID, workspaceID, targetID string) (*ViewedMember, error) {
	viewed, err := s.CheckViewAs(ctx, viewerID, workspaceID, targetID)
	if err != nil {
		return nil, err
	}
	logging.FromCtx(ctx).Info("view as started", "viewer_id", viewerID, "target_id", viewed.UserID, "workspace_id", viewed.WorkspaceID)
	return viewed, nil
}

// heldToViewedWorkspace keeps only the workspace a view-as request is held to.
func heldToViewedWorkspace(ctx context.Context, workspaces []*Workspace) []*Workspace {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.Viewer == nil {
		return workspaces
	}
	out := make([]*Workspace, 0, 1)
	for _, w := range workspaces {
		if w.ID == actor.Viewer.WorkspaceID {
			out = append(out, w)
		}
	}
	return out
}
