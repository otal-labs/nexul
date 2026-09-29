package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/tenancy"
)

var _ tenancy.Repo = (*WorkspacesRepo)(nil)

// WorkspacesRepo persists the tenancy domain's Workspace entity (ticket 07).
type WorkspacesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *WorkspacesRepo) Create(ctx context.Context, ws *tenancy.Workspace) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateWorkspace(ctx, sqlcgen.CreateWorkspaceParams{
			ID: ws.ID, Name: ws.Name, MentionChipTemplate: ws.MentionChipTemplate,
			CreatedAt: ws.CreatedAt.Unix(), UpdatedAt: ws.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert workspace %s: %w", ws.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *WorkspacesRepo) Update(ctx context.Context, ws *tenancy.Workspace) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateWorkspace(ctx, sqlcgen.UpdateWorkspaceParams{
			Name: ws.Name, MentionChipTemplate: ws.MentionChipTemplate, UpdatedAt: ws.UpdatedAt.Unix(), ID: ws.ID,
		})
		if err != nil {
			return fmt.Errorf("update workspace %s: %w", ws.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update workspace %s: %w", ws.ID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *WorkspacesRepo) Get(ctx context.Context, id string) (*tenancy.Workspace, error) {
	row, err := r.q.GetWorkspace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get workspace %s: %w", id, notFoundIfNoRows(err))
	}
	return toWorkspace(row), nil
}

func (r *WorkspacesRepo) ListForUser(ctx context.Context, userID string) ([]*tenancy.Workspace, error) {
	rows, err := r.q.ListWorkspacesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces for user %s: %w", userID, err)
	}
	var out []*tenancy.Workspace
	for _, row := range rows {
		out = append(out, toWorkspace(row))
	}
	return out, nil
}

func (r *WorkspacesRepo) ListWithRoles(ctx context.Context) ([]*tenancy.TeamWorkspace, error) {
	rows, err := r.q.ListTeamWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	roleRows, err := r.q.ListTeamRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	out := make([]*tenancy.TeamWorkspace, 0, len(rows))
	byID := make(map[string]*tenancy.TeamWorkspace, len(rows))
	for _, row := range rows {
		w := &tenancy.TeamWorkspace{ID: row.ID, Name: row.Name, Roles: []*tenancy.TeamRole{}}
		out = append(out, w)
		byID[row.ID] = w
	}
	for _, role := range roleRows {
		if w, ok := byID[role.WorkspaceID]; ok {
			w.Roles = append(w.Roles, &tenancy.TeamRole{ID: role.ID, Name: role.Name, IsOwner: role.IsOwnerRole != 0})
		}
	}
	return out, nil
}

func toWorkspace(row sqlcgen.Workspace) *tenancy.Workspace {
	return &tenancy.Workspace{
		ID:                  row.ID,
		Name:                row.Name,
		MentionChipTemplate: row.MentionChipTemplate,
		CreatedAt:           time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:           time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

var _ tenancy.MemberRepo = (*WorkspaceMembersRepo)(nil)

// workspaceResourceType is the permission_overwrites resource type of a workspace-wide overwrite (ADR 0042).
const workspaceResourceType = "workspace"

// WorkspaceMembersRepo persists the tenancy domain's workspace_members join table (ticket 07).
type WorkspaceMembersRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *WorkspaceMembersRepo) AddMember(ctx context.Context, m *tenancy.Member, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).AddWorkspaceMember(ctx, sqlcgen.AddWorkspaceMemberParams{
			UserID: m.UserID, WorkspaceID: m.WorkspaceID, RoleID: m.RoleID, CreatedAt: m.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("add member %s to workspace %s: %w", m.UserID, m.WorkspaceID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WorkspaceMembersRepo) RoleIDFor(ctx context.Context, workspaceID, userID string) (string, error) {
	roleID, err := r.q.GetWorkspaceMemberRole(ctx, sqlcgen.GetWorkspaceMemberRoleParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return "", fmt.Errorf("role for user %s in workspace %s: %w", userID, workspaceID, notFoundIfNoRows(err))
	}
	return roleID, nil
}

func (r *WorkspaceMembersRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]*tenancy.Member, error) {
	rows, err := r.q.ListWorkspaceMembers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members for workspace %s: %w", workspaceID, err)
	}
	var out []*tenancy.Member
	for _, row := range rows {
		out = append(out, toMember(row))
	}
	return out, nil
}

func (r *WorkspaceMembersRepo) RemoveMember(ctx context.Context, workspaceID, userID string, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).RemoveWorkspaceMember(ctx, sqlcgen.RemoveWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID})
		if err != nil {
			return fmt.Errorf("remove member %s from workspace %s: %w", userID, workspaceID, err)
		}
		if n == 0 {
			return fmt.Errorf("remove member %s from workspace %s: %w", userID, workspaceID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WorkspaceMembersRepo) SetRole(ctx context.Context, workspaceID, userID, roleID string, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).SetWorkspaceMemberRole(ctx, sqlcgen.SetWorkspaceMemberRoleParams{
			RoleID: roleID, WorkspaceID: workspaceID, UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("set role for %s in workspace %s: %w", userID, workspaceID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("set role for %s in workspace %s: %w", userID, workspaceID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WorkspaceMembersRepo) Overrides(ctx context.Context, workspaceID, userID string) (permissions.Set, permissions.Set, error) {
	row, err := r.q.GetOverwrite(ctx, sqlcgen.GetOverwriteParams{ResourceType: workspaceResourceType, ResourceID: workspaceID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get overrides for %s in workspace %s: %w", userID, workspaceID, err)
	}
	o, err := toOverwrite(row)
	if err != nil {
		return nil, nil, err
	}
	return o.Allow, o.Deny, nil
}

func (r *WorkspaceMembersRepo) SetOverrides(ctx context.Context, workspaceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if len(allow) == 0 && len(deny) == 0 {
			_, err := q.DeleteOverwrite(ctx, sqlcgen.DeleteOverwriteParams{ResourceType: workspaceResourceType, ResourceID: workspaceID, UserID: userID})
			if err != nil {
				return fmt.Errorf("clear overrides for %s in workspace %s: %w", userID, workspaceID, err)
			}
			return insertOutboxRows(ctx, tx, events)
		}
		_, err := q.UpsertOverwrite(ctx, sqlcgen.UpsertOverwriteParams{
			ResourceType: workspaceResourceType, ResourceID: workspaceID, UserID: userID,
			Allow: setJSON(allow), Deny: setJSON(deny), Now: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("set overrides for %s in workspace %s: %w", userID, workspaceID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WorkspaceMembersRepo) ListAllMemberships(ctx context.Context) ([]*tenancy.TeamMembership, error) {
	rows, err := r.q.ListTeamMemberships(ctx)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	out := make([]*tenancy.TeamMembership, 0, len(rows))
	for _, row := range rows {
		allow, err := parseSet(row.Allow)
		if err != nil {
			return nil, fmt.Errorf("decode allow for %s in workspace %s: %w", row.UserID, row.WorkspaceID, err)
		}
		deny, err := parseSet(row.Deny)
		if err != nil {
			return nil, fmt.Errorf("decode deny for %s in workspace %s: %w", row.UserID, row.WorkspaceID, err)
		}
		out = append(out, &tenancy.TeamMembership{
			UserID: row.UserID, WorkspaceID: row.WorkspaceID, WorkspaceName: row.WorkspaceName,
			RoleID: row.RoleID, RoleName: row.RoleName, IsOwner: row.IsOwnerRole != 0, Allow: allow, Deny: deny,
		})
	}
	return out, nil
}

func toMember(row sqlcgen.WorkspaceMember) *tenancy.Member {
	return &tenancy.Member{
		UserID:      row.UserID,
		WorkspaceID: row.WorkspaceID,
		RoleID:      row.RoleID,
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
	}
}

var _ tenancy.InviteRepo = (*WorkspaceInvitesRepo)(nil)

// WorkspaceInvitesRepo persists the tenancy domain's workspace_invites table.
type WorkspaceInvitesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *WorkspaceInvitesRepo) Upsert(ctx context.Context, inv *tenancy.Invite) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).UpsertWorkspaceInvite(ctx, sqlcgen.UpsertWorkspaceInviteParams{
			WorkspaceID: inv.WorkspaceID, Login: inv.Login, RoleID: inv.RoleID,
			InvitedBy: inv.InvitedBy, CreatedAt: inv.CreatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("upsert invite for %s to workspace %s: %w", inv.Login, inv.WorkspaceID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *WorkspaceInvitesRepo) Delete(ctx context.Context, workspaceID, login string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).DeleteWorkspaceInvite(ctx, sqlcgen.DeleteWorkspaceInviteParams{WorkspaceID: workspaceID, Login: login})
		if err != nil {
			return fmt.Errorf("delete invite for %s from workspace %s: %w", login, workspaceID, err)
		}
		return nil
	})
}

func (r *WorkspaceInvitesRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]*tenancy.Invite, error) {
	rows, err := r.q.ListWorkspaceInvitesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list invites for workspace %s: %w", workspaceID, err)
	}
	return toInvites(rows), nil
}

func (r *WorkspaceInvitesRepo) ListByLogin(ctx context.Context, login string) ([]*tenancy.Invite, error) {
	rows, err := r.q.ListWorkspaceInvitesByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("list invites for login %s: %w", login, err)
	}
	return toInvites(rows), nil
}

func toInvites(rows []sqlcgen.WorkspaceInvite) []*tenancy.Invite {
	var out []*tenancy.Invite
	for _, row := range rows {
		out = append(out, &tenancy.Invite{
			WorkspaceID: row.WorkspaceID,
			Login:       row.Login,
			RoleID:      row.RoleID,
			InvitedBy:   row.InvitedBy,
			CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		})
	}
	return out
}
