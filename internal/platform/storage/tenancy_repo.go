package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
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
			ID: ws.ID, Name: ws.Name, Slug: ws.Slug, MentionChipTemplate: storedChip(ws),
			CreatedAt: ws.CreatedAt.Unix(), UpdatedAt: ws.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert workspace %s: %w", ws.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *WorkspacesRepo) Update(ctx context.Context, ws *tenancy.Workspace, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateWorkspace(ctx, sqlcgen.UpdateWorkspaceParams{
			Name: ws.Name, Slug: ws.Slug, MentionChipTemplate: storedChip(ws), UpdatedAt: ws.UpdatedAt.Unix(), ID: ws.ID,
		})
		if err != nil {
			return fmt.Errorf("update workspace %s: %w", ws.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update workspace %s: %w", ws.ID, apperrs.ErrNotFound)
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WorkspacesRepo) Get(ctx context.Context, id string) (*tenancy.Workspace, error) {
	row, err := r.q.GetWorkspace(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get workspace %s: %w", id, notFoundIfNoRows(err))
	}
	return toWorkspace(row), nil
}

func (r *WorkspacesRepo) GetBySlug(ctx context.Context, slug string) (*tenancy.Workspace, error) {
	row, err := r.q.GetWorkspaceBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get workspace by slug %s: %w", slug, notFoundIfNoRows(err))
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
		ID:                        row.ID,
		Name:                      row.Name,
		Slug:                      row.Slug,
		MentionChipTemplate:       row.MentionChipTemplate,
		MentionChipTemplateEdited: row.MentionChipTemplate != "",
		AccessSetupFinished:       row.AccessSetupFinished != 0,
		CreatedAt:                 time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:                 time.Unix(row.UpdatedAt, 0).UTC(),
	}
}

// storedChip is the column's value: empty while the workspace follows the instance's chip template (ADR 0103).
func storedChip(ws *tenancy.Workspace) string {
	if !ws.MentionChipTemplateEdited {
		return ""
	}
	return ws.MentionChipTemplate
}

var _ tenancy.MemberRepo = (*WorkspaceMembersRepo)(nil)

// projectResourceType is the permission_overwrites resource type of a project's rules (ADR 0148).
const projectResourceType = "project"

// WorkspaceMembersRepo persists the tenancy domain's workspace_members join table (ticket 07).
type WorkspaceMembersRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *WorkspaceMembersRepo) AddMember(ctx context.Context, m *tenancy.Member, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		err := q.AddWorkspaceMember(ctx, sqlcgen.AddWorkspaceMemberParams{UserID: m.UserID, WorkspaceID: m.WorkspaceID, CreatedAt: m.CreatedAt.Unix()})
		if err != nil {
			return fmt.Errorf("add member %s to workspace %s: %w", m.UserID, m.WorkspaceID, classifyWriteErr(err))
		}
		for _, roleID := range m.Held() {
			if err := addMemberRole(ctx, q, m.WorkspaceID, m.UserID, roleID, m.CreatedAt); err != nil {
				return err
			}
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func addMemberRole(ctx context.Context, q *sqlcgen.Queries, workspaceID, userID, roleID string, at time.Time) error {
	err := q.AddMemberRole(ctx, sqlcgen.AddMemberRoleParams{UserID: userID, WorkspaceID: workspaceID, RoleID: roleID, CreatedAt: at.Unix()})
	if err != nil {
		return fmt.Errorf("give %s role %s in workspace %s: %w", userID, roleID, workspaceID, classifyWriteErr(err))
	}
	return nil
}

func (r *WorkspaceMembersRepo) RoleIDFor(ctx context.Context, workspaceID, userID string) (string, error) {
	m, err := r.Member(ctx, workspaceID, userID)
	if err != nil {
		return "", err
	}
	return m.RoleID, nil
}

func (r *WorkspaceMembersRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]*tenancy.Member, error) {
	rows, err := r.q.ListWorkspaceMembers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members for workspace %s: %w", workspaceID, err)
	}
	held, err := r.q.ListWorkspaceMemberRoles(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list roles of workspace %s's members: %w", workspaceID, err)
	}
	roleIDs := map[string][]string{}
	for _, h := range held {
		roleIDs[h.UserID] = append(roleIDs[h.UserID], h.RoleID)
	}
	var out []*tenancy.Member
	for _, row := range rows {
		out = append(out, toMember(row, roleIDs[row.UserID]))
	}
	return out, nil
}

var _ access.RoleResolver = (*WorkspaceMembersRepo)(nil)

// MemberRole is userID's roles in workspaceID with what each allows, read in one statement; not found for a non-member.
func (r *WorkspaceMembersRepo) MemberRole(ctx context.Context, workspaceID, userID string) (access.RoleInfo, error) {
	rows, err := r.q.ListMemberRoleGrants(ctx, sqlcgen.ListMemberRoleGrantsParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return access.RoleInfo{}, fmt.Errorf("roles of %s in workspace %s: %w", userID, workspaceID, err)
	}
	if len(rows) == 0 {
		return access.RoleInfo{}, fmt.Errorf("member %s of workspace %s: %w", userID, workspaceID, apperrs.ErrNotFound)
	}
	held := make([]access.HeldRole, 0, len(rows))
	for _, row := range rows {
		if !row.RoleID.Valid {
			continue
		}
		perms, err := parseSet(row.Permissions.String)
		if err != nil {
			return access.RoleInfo{}, fmt.Errorf("decode role %s: %w", row.RoleID.String, err)
		}
		held = append(held, access.HeldRole{ID: row.RoleID.String, IsOwner: row.IsOwnerRole.Int64 != 0, Permissions: perms})
	}
	return access.RolesHeld(held...), nil
}

// RemoveMember also deletes the rules naming them on the workspace's projects, so coming back starts from nothing.
func (r *WorkspaceMembersRepo) RemoveMember(ctx context.Context, workspaceID, userID string, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.RemoveWorkspaceMember(ctx, sqlcgen.RemoveWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID})
		if err != nil {
			return fmt.Errorf("remove member %s from workspace %s: %w", userID, workspaceID, err)
		}
		if n == 0 {
			return fmt.Errorf("remove member %s from workspace %s: %w", userID, workspaceID, apperrs.ErrNotFound)
		}
		if err := q.DeleteMemberProjectAccess(ctx, sqlcgen.DeleteMemberProjectAccessParams{UserID: nullString(userID), WorkspaceID: workspaceID}); err != nil {
			return fmt.Errorf("remove project access of %s in workspace %s: %w", userID, workspaceID, err)
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WorkspaceMembersRepo) Member(ctx context.Context, workspaceID, userID string) (*tenancy.Member, error) {
	row, err := r.q.GetWorkspaceMember(ctx, sqlcgen.GetWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("member %s of workspace %s: %w", userID, workspaceID, notFoundIfNoRows(err))
	}
	roleIDs, err := r.q.ListMemberRoleIDs(ctx, sqlcgen.ListMemberRoleIDsParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("roles of %s in workspace %s: %w", userID, workspaceID, err)
	}
	return toMember(row, roleIDs), nil
}

// WorkspaceIDs lists every workspace userID belongs to.
func (r *WorkspaceMembersRepo) WorkspaceIDs(ctx context.Context, userID string) ([]string, error) {
	ids, err := r.q.ListWorkspaceIDsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces of %s: %w", userID, err)
	}
	return ids, nil
}

// ProjectMemberIDs lists the members of workspaceID who may open projectID, in one read: the Owner, and anyone its
// rules let in, by name or through a role, unless a rule naming them keeps them out (ADR 0148).
func (r *WorkspaceMembersRepo) ProjectMemberIDs(ctx context.Context, workspaceID, projectID string) ([]string, error) {
	ids, err := r.q.ListProjectMemberIDs(ctx, sqlcgen.ListProjectMemberIDsParams{WorkspaceID: workspaceID, ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("list the members of workspace %s in project %s: %w", workspaceID, projectID, err)
	}
	return ids, nil
}

// ProjectRoleIDs lists the roles projectID's rules let in, in the order they were listed, in one read.
func (r *WorkspaceMembersRepo) ProjectRoleIDs(ctx context.Context, projectID string) ([]string, error) {
	rows, err := r.q.ListProjectRoleIDs(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list the roles in project %s: %w", projectID, err)
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.String)
	}
	return ids, nil
}

// ChangeRoles gives userID the roles in add and takes away those in remove, in one write; an added role comes after
// the ones already held.
func (r *WorkspaceMembersRepo) ChangeRoles(ctx context.Context, workspaceID, userID string, add, remove []string, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if _, err := q.GetWorkspaceMember(ctx, sqlcgen.GetWorkspaceMemberParams{WorkspaceID: workspaceID, UserID: userID}); err != nil {
			return fmt.Errorf("change roles of %s in workspace %s: %w", userID, workspaceID, notFoundIfNoRows(err))
		}
		for _, roleID := range remove {
			if err := q.DeleteMemberRole(ctx, sqlcgen.DeleteMemberRoleParams{WorkspaceID: workspaceID, UserID: userID, RoleID: roleID}); err != nil {
				return fmt.Errorf("take role %s from %s in workspace %s: %w", roleID, userID, workspaceID, err)
			}
		}
		for _, roleID := range add {
			if err := addMemberRole(ctx, q, workspaceID, userID, roleID, time.Now()); err != nil {
				return err
			}
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WorkspaceMembersRepo) ListAllMemberships(ctx context.Context) ([]*tenancy.TeamMembership, error) {
	rows, err := r.q.ListTeamMemberships(ctx)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	// A membership comes back once per role it holds, its rows together.
	out := make([]*tenancy.TeamMembership, 0, len(rows))
	var m *tenancy.TeamMembership
	for _, row := range rows {
		if m == nil || m.UserID != row.UserID || m.WorkspaceID != row.WorkspaceID {
			m = toTeamMembership(row)
			out = append(out, m)
		}
		if !row.RoleID.Valid {
			continue
		}
		role := &tenancy.TeamRole{ID: row.RoleID.String, Name: row.RoleName.String, IsOwner: row.IsOwnerRole.Int64 != 0}
		if len(m.Roles) == 0 {
			m.RoleID = role.ID
		}
		m.Roles = append(m.Roles, role)
		m.RoleName = tenancy.RoleNames(m.Roles)
		m.IsOwner = m.IsOwner || role.IsOwner
	}
	return out, nil
}

func toTeamMembership(row sqlcgen.ListTeamMembershipsRow) *tenancy.TeamMembership {
	return &tenancy.TeamMembership{UserID: row.UserID, WorkspaceID: row.WorkspaceID, WorkspaceName: row.WorkspaceName}
}

func toMember(row sqlcgen.WorkspaceMember, roleIDs []string) *tenancy.Member {
	m := &tenancy.Member{
		UserID:      row.UserID,
		WorkspaceID: row.WorkspaceID,
		RoleIDs:     roleIDs,
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
	}
	if len(roleIDs) > 0 {
		m.RoleID = roleIDs[0]
	}
	return m
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
