package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// accessPlayWorkspaceResolver reads from the repo directly to avoid recursing into a permission check;
// unlike a doc, a play already carries its own workspace id (ticket 21).
type accessPlayWorkspaceResolver struct {
	plays *storage.PlaysRepo
}

func (a accessPlayWorkspaceResolver) WorkspaceIDForPlay(ctx context.Context, playID string) (string, error) {
	p, err := a.plays.Get(ctx, playID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

// accessRoleResolver adapts tenancy+roles for HasPermission's role-mask layer.
type accessRoleResolver struct {
	tenancy *tenancy.Service
	roles   *roles.Service
}

func (a accessRoleResolver) MemberRole(ctx context.Context, workspaceID, userID string) (access.RoleInfo, error) {
	m, err := a.tenancy.Member(ctx, workspaceID, userID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	r, err := a.roles.Get(ctx, workspaceID, m.RoleID)
	if err != nil {
		return access.RoleInfo{}, err
	}
	return access.RoleInfo{IsOwnerRole: r.IsOwnerRole, Permissions: r.Permissions, Restricted: m.Restricted}, nil
}

// accessDocWorkspaceResolver reads from the repo directly to avoid recursing into a permission check.
type accessDocWorkspaceResolver struct {
	docs     *storage.DocsRepo
	projects *storage.ProjectsRepo
}

func (a accessDocWorkspaceResolver) DocScope(ctx context.Context, docID string) (string, string, error) {
	d, err := a.docs.GetByID(ctx, docID)
	if err != nil {
		return "", "", err
	}
	if d.ProjectID == "" {
		return "", "", nil
	}
	p, err := a.projects.Get(ctx, d.ProjectID)
	if err != nil {
		return "", "", err
	}
	return p.WorkspaceID, d.ProjectID, nil
}

// accessUsers adapts UsersRepo to access's Users interface, mapping auth.User to access.User (ADR 0017).
type accessUsers struct {
	users *storage.UsersRepo
}

func (a accessUsers) ListUsers(ctx context.Context) ([]*access.User, error) {
	us, err := a.users.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*access.User, 0, len(us))
	for _, u := range us {
		out = append(out, mapAccessUser(u))
	}
	return out, nil
}

func mapAccessUser(u *auth.User) *access.User {
	return &access.User{ID: u.ID, Login: u.Login, Name: u.Name}
}

// accessScopes reads projects and memberships from storage: a check that went through a gated use-case would
// ask itself for permission.
type accessScopes struct {
	projects *storage.ProjectsRepo
	members  *storage.WorkspaceMembersRepo
}

func (a accessScopes) WorkspaceIDForProject(ctx context.Context, projectID string) (string, error) {
	p, err := a.projects.Get(ctx, projectID)
	if err != nil {
		return "", err
	}
	return p.WorkspaceID, nil
}

func (a accessScopes) UnrestrictedWorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error) {
	return a.members.UnrestrictedWorkspaceIDs(ctx, userID)
}

// workspaceTicketProjects reads a ticket's project from storage for the workspace domain's ticket moves (ADR 0017).
type workspaceTicketProjects struct {
	tickets *storage.TicketsRepo
}

func (a workspaceTicketProjects) ProjectOfTicket(ctx context.Context, ticketID string) (string, error) {
	t, err := a.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return "", err
	}
	return t.ProjectID, nil
}

// projectEntityGate resolves a repository or a ticket to its project from storage, then asks access for the action
// in that project's workspace; a repository no project links, or an unknown ticket, is not found.
type projectEntityGate struct {
	access   *access.Service
	projects *storage.ProjectsRepo
	tickets  *storage.TicketsRepo
}

func (g projectEntityGate) RequireRepo(ctx context.Context, owner, name string, action permissions.Action) error {
	if identity.Internal(ctx) {
		return nil
	}
	ref, err := g.projects.GetRepoByFullName(ctx, owner, name)
	if err != nil {
		return err
	}
	return g.access.RequireProject(ctx, ref.ProjectID, action)
}

func (g projectEntityGate) RequireTicket(ctx context.Context, ticketID string, action permissions.Action) error {
	if identity.Internal(ctx) {
		return nil
	}
	t, err := g.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return err
	}
	return g.access.RequireProject(ctx, t.ProjectID, action)
}

// ticketProjectPeople resolves a ticket person's login and asks access whether they may open the project; a login
// nobody signed in with stays accepted, as before people were checked.
type ticketProjectPeople struct {
	users  userLookupGate
	access *access.Service
}

func (g ticketProjectPeople) MayOpen(ctx context.Context, login, projectID string) (bool, error) {
	userID, found, err := g.users.UserIDForLogin(ctx, login)
	if err != nil || !found {
		return !found, err
	}
	return g.access.CanInProject(ctx, userID, projectID, permissions.Member), nil
}
