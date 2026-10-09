package main

import (
	"context"
	"maps"
	"slices"

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
	docs *storage.DocsRepo
}

func (a accessDocWorkspaceResolver) DocScopes(ctx context.Context, docIDs []string) (map[string]access.DocScope, error) {
	return a.docs.ScopesOf(ctx, docIDs)
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

func (a accessScopes) ProjectIDs(ctx context.Context, workspaceID string) ([]string, error) {
	ps, err := a.projects.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(ps))
	for i, p := range ps {
		ids[i] = p.ID
	}
	return ids, nil
}

func (a accessScopes) UnrestrictedWorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error) {
	return a.members.UnrestrictedWorkspaceIDs(ctx, userID)
}

func (a accessScopes) WorkspaceIDsForUser(ctx context.Context, userID string) ([]string, error) {
	return a.members.WorkspaceIDs(ctx, userID)
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

// RequireTickets is RequireTicket for many tickets, asking access once per project rather than once per ticket.
func (g projectEntityGate) RequireTickets(ctx context.Context, ticketIDs []string, action permissions.Action) (map[string]bool, error) {
	out := make(map[string]bool, len(ticketIDs))
	if identity.Internal(ctx) {
		for _, id := range ticketIDs {
			out[id] = true
		}
		return out, nil
	}
	projectOf, err := g.tickets.ProjectsOf(ctx, ticketIDs)
	if err != nil {
		return nil, err
	}
	known := slices.Collect(maps.Keys(projectOf))
	allowed, err := permissions.Filter(known, func(id string) string { return projectOf[id] }, func(projectID string) error {
		return g.access.RequireProject(ctx, projectID, action)
	})
	if err != nil {
		return nil, err
	}
	for _, id := range allowed {
		out[id] = true
	}
	return out, nil
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
