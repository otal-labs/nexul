package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

const (
	uClient   = "u-client"
	pClient   = "p-client"
	pGeneral  = "project-general"
	wsDefault = "workspace-default"
)

// restrictedClient adds uClient to the permission fixture as a Restricted member holding Project access to p-client
// only, on a role that carries reads, writes, and every instance-level bit, so anything they reach beyond p-client
// would be the resolver's fault, not the role's.
func restrictedClient(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uClient, Provider: auth.ProviderGitHub, ProviderUserID: uClient, Login: uClient})
	require.NoError(t, err)
	clientRole := append([]string{"tickets:read", "tickets:write", "docs:read", "stacks:read", "deploys:read", "projects:read", "projects:write", "plays:read", "chat:write"}, instanceBits...)
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-client", WorkspaceID: wsDefault, Name: "Client", Permissions: grant(clientRole...), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uClient, WorkspaceID: wsDefault, RoleID: "role-client", CreatedAt: now}))
	require.NoError(t, f.store.Projects.Create(ctx, &workspace.Project{ID: pClient, Name: "Client portal", Prefix: "CLI", WorkspaceID: wsDefault, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.svc.tenancySvc.SetEveryProject(ctx, uOwner, wsDefault, uClient, tenancy.EveryProjectNone))
	require.NoError(t, f.svc.tenancySvc.SetProjectAccess(ctx, uOwner, wsDefault, uClient, pClient, grant("tickets:read", "tickets:write", "docs:read")))
}

// giveGeneral hands uClient Project access to the fixture's project, or takes it away with no actions.
func giveGeneral(t *testing.T, f permFixture, actions ...string) {
	t.Helper()
	require.NoError(t, f.svc.tenancySvc.SetProjectAccess(context.Background(), uOwner, wsDefault, uClient, pGeneral, grant(actions...)))
}

func TestIntegration_RestrictedMember(t *testing.T) {
	f := newPermFixture(t)
	restrictedClient(t, f)
	s := f.svc
	client := as(uClient)

	file, err := s.attachmentsSvc.Upload(as(uOwner), attachments.Owner{TicketID: f.ticket.ID}, "shot.png", []byte("\x89PNG\r\n\x1a\n"))
	require.NoError(t, err)
	require.NoError(t, f.store.Access.Set(context.Background(), "doc", f.doc, uClient, grant("docs:read"), nil))

	t.Run("everything in a project they hold no access to reads as not found", func(t *testing.T) {
		for name, call := range map[string]func() error{
			"project":            func() error { _, err := s.workspaceSvc.Get(client, pGeneral); return err },
			"board settings":     func() error { _, err := s.workspaceSvc.ListStatusesByProject(client, pGeneral); return err },
			"ticket":             func() error { _, err := s.ticketsSvc.Get(client, f.ticket.ID); return err },
			"doc shared to them": func() error { _, err := s.docsSvc.Get(client, f.doc); return err },
			"stack":              func() error { _, err := s.deploySvc.GetStack(client, f.stack); return err },
			"deploy":             func() error { _, err := s.deploySvc.Get(client, f.deploy); return err },
			"ticket file":        func() error { _, err := s.attachmentsSvc.Get(client, file.ID); return err },
			"ticket's files": func() error {
				_, err := s.attachmentsSvc.List(client, attachments.Owner{TicketID: f.ticket.ID})
				return err
			},
		} {
			assert.Equal(t, notFound, outcome(call()), name)
		}
	})

	t.Run("the same reads through the HTTP gateway are 404", func(t *testing.T) {
		for path, routes := range map[string]http.Handler{
			"/api/projects/" + pGeneral:   workspace.NewHandler(s.workspaceSvc).Routes(),
			"/api/tickets/" + f.ticket.ID: tickets.NewHandler(s.ticketsSvc).Routes(),
			"/api/attachments/" + file.ID: attachments.NewHandler(s.attachmentsSvc).Routes(),
		} {
			rec := httptest.NewRecorder()
			routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil).WithContext(client))
			assert.Equal(t, http.StatusNotFound, rec.Code, path)
		}
	})

	t.Run("the project list holds only their project, and they cannot create one", func(t *testing.T) {
		projects, err := s.workspaceSvc.List(client, wsDefault)
		require.NoError(t, err)
		ids := []string{}
		for _, p := range projects {
			ids = append(ids, p.ID)
		}
		assert.Equal(t, []string{pClient}, ids)
		_, err = s.workspaceSvc.Create(client, wsDefault, "Mine", "MIN", workspace.ProjectIcon(""))
		assert.Equal(t, forbidden, outcome(err))
		_, err = s.ticketsSvc.Create(client, pClient, "Feedback", "", "", "")
		assert.NoError(t, err, "Project access answers inside their project")
	})

	t.Run("a restricted membership holds no instance-level bit, whatever its role carries", func(t *testing.T) {
		held, err := s.accessSvc.PermissionsAnywhere(context.Background(), uClient)
		require.NoError(t, err)
		assert.Empty(t, held, "instance_permissions on /api/auth/me reads this")
		runs, err := s.accessSvc.HoldsAnywhere(context.Background(), uClient, permissions.RunnersWrite)
		require.NoError(t, err)
		assert.False(t, runs)
	})

	t.Run("a developer or tester who cannot open the project is refused", func(t *testing.T) {
		_, err := s.ticketsSvc.SetPerson(as(uOwner), f.ticket.ID, tickets.RoleDeveloper, uClient)
		assert.Equal(t, invalid, outcome(err))
		own, err := s.ticketsSvc.Create(as(uOwner), pClient, "Theirs", "", "", "")
		require.NoError(t, err)
		_, err = s.ticketsSvc.SetPerson(as(uOwner), own.ID, tickets.RoleTester, uClient)
		assert.NoError(t, err)
	})

	t.Run("project settings lists who holds access, for a manager only", func(t *testing.T) {
		entries, err := s.workspaceSvc.ProjectAccess(as(uOwner), pClient)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, uClient, entries[0].UserID)
		_, err = s.workspaceSvc.ProjectAccess(client, pClient)
		assert.Equal(t, forbidden, outcome(err))
		people, err := s.tenancySvc.ListProjectPeople(context.Background(), uOwner, pGeneral)
		require.NoError(t, err)
		for _, p := range people {
			assert.NotEqual(t, uClient, p.UserID, "the developer picker leaves them out")
		}
	})
}

// Notices and trails on a hidden project vanish from reads and come back with access; the rows are kept.
func TestIntegration_RestrictedMember_NoticesAndTrailsFollowAccess(t *testing.T) {
	f := newPermFixture(t)
	restrictedClient(t, f)
	s := f.svc
	ctx := context.Background()
	require.NoError(t, f.store.Notifications.CreateMany(ctx, []*workspace.Notification{{
		ID: "n-1", UserID: uClient, WorkspaceID: wsDefault, Kind: workspace.KindTicketMentioned,
		SubjectType: workspace.SubjectTicket, SubjectID: f.ticket.ID, SubjectTitle: "Login times out", CreatedAt: time.Now(),
	}}))
	require.NoError(t, f.store.PlayTrails.CreateTrail(ctx, &plays.Trail{
		ID: "tr-1", WorkspaceID: wsDefault, PlayID: "play-1", PlayLabel: "Fix", TargetType: plays.TargetTicket, TargetID: f.ticket.ID,
		ProjectID: pGeneral, StarterID: uOwner, Via: plays.ViaWeb, State: plays.TrailDone, StartedAt: time.Now(), Activity: []plays.ActivityEntry{},
	}))
	runner := plays.NewRunner(plays.RunnerConfig{Plays: f.store.Plays, Trails: f.store.PlayTrails, Perm: s.accessSvc})

	visible := func() (notices, unread, trails int, trailErr string) {
		ns, _, err := s.notifSvc.Page(ctx, uClient, workspace.InboxFilter{WorkspaceID: wsDefault}, paging.Window{Limit: 50})
		require.NoError(t, err)
		n, err := s.notifSvc.UnreadCount(ctx, uClient, "")
		require.NoError(t, err)
		list, err := runner.ListTrails(as(uClient), plays.TargetTicket, f.ticket.ID)
		require.NoError(t, err)
		_, err = runner.GetTrail(as(uClient), "tr-1")
		return len(ns), n, len(list), outcome(err)
	}
	notices, unread, trails, get := visible()
	assert.Equal(t, []any{0, 0, 0, notFound}, []any{notices, unread, trails, get}, "hidden while the project is")

	giveGeneral(t, f, "tickets:read")
	notices, unread, trails, get = visible()
	assert.Equal(t, []any{1, 1, 1, ok}, []any{notices, unread, trails, get}, "back once access is given")
}

// A live ticket frame stops reaching a restricted socket the moment its project's access is taken.
func TestIntegration_RestrictedMember_LiveFramesFollowAccess(t *testing.T) {
	f := newPermFixture(t)
	restrictedClient(t, f)
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	frame := tickets.UpdatedEvent{Ticket: *f.ticket}
	reaches := func() bool { return a.allows(as(uClient), tickets.TopicUpdated, frame) }

	assert.False(t, reaches())
	giveGeneral(t, f, "tickets:read")
	assert.True(t, reaches())
	giveGeneral(t, f)
	assert.False(t, reaches(), "taken away, the next frame is refused")

	grantFrame := tenancy.ProjectAccessEvent{ResourceType: "project", ResourceID: pGeneral, UserID: uClient, ActorID: uOwner}
	assert.True(t, a.allows(as(uClient), tenancy.TopicProjectAccessChanged, grantFrame), "the person themselves")
	assert.True(t, a.allows(as(uManager), tenancy.TopicProjectAccessChanged, grantFrame), "a holder of members:write in the workspace")
	assert.False(t, a.allows(as(uReader), tenancy.TopicProjectAccessChanged, grantFrame))
}

// Setting access is judged by what it adds, and leaves no stale rows behind.
func TestIntegration_RestrictedMember_GrantingAndCleanup(t *testing.T) {
	f := newPermFixture(t)
	restrictedClient(t, f)
	s := f.svc
	ctx := context.Background()
	require.NoError(t, s.tenancySvc.SetEveryProject(ctx, uOwner, wsDefault, uManager, tenancy.EveryProjectNone))
	require.NoError(t, s.tenancySvc.SetProjectAccess(ctx, uOwner, wsDefault, uManager, pClient, grant("tickets:read")))

	t.Run("a restricted giver sets only levels they hold on that project", func(t *testing.T) {
		err := s.tenancySvc.SetProjectAccess(ctx, uManager, wsDefault, uClient, pClient, grant("tickets:read", "tickets:write", "docs:read", "docs:write"))
		assert.Equal(t, forbidden, outcome(err))
		err = s.tenancySvc.SetProjectAccess(ctx, uManager, wsDefault, uClient, pGeneral, grant("tickets:read"))
		assert.Equal(t, notFound, outcome(err), "a project hidden from the giver")
		assert.Equal(t, forbidden, outcome(s.tenancySvc.SetEveryProject(ctx, uManager, wsDefault, uClient, tenancy.EveryProjectRole)),
			"From role hands out project areas the giver holds only inside a project")
		_, err = s.rolesSvc.Update(ctx, wsDefault, "role-plain", uManager, "Member", grant("tickets:read"))
		assert.Equal(t, forbidden, outcome(err), "nor may they add a project area to a role")
	})

	t.Run("the change publishes the grant event through the outbox", func(t *testing.T) {
		entries, err := f.store.Outbox.Unpublished(ctx, 500)
		require.NoError(t, err)
		found := false
		for _, e := range entries {
			found = found || (e.Topic == tenancy.TopicProjectAccessChanged && strings.Contains(string(e.Payload), `"resource_type":"project"`))
		}
		assert.True(t, found)
	})

	t.Run("deleting a project names who loses access, then deletes their rows", func(t *testing.T) {
		impact, err := s.workspaceSvc.DeleteImpact(as(uOwner), pClient)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{uClient, uManager}, []string{impact.RestrictedMembers[0].UserID, impact.RestrictedMembers[1].UserID})
		require.NoError(t, s.workspaceSvc.Delete(as(uOwner), pClient))
		rows, err := f.store.WorkspaceMembers.ListAllProjectAccess(ctx)
		require.NoError(t, err)
		assert.Empty(t, rows)
	})

	t.Run("removing a member leaves none of their access", func(t *testing.T) {
		giveGeneral(t, f, "tickets:read")
		require.NoError(t, s.tenancySvc.RemoveMember(ctx, uOwner, wsDefault, uClient))
		rows, err := f.store.WorkspaceMembers.ListAllProjectAccess(ctx)
		require.NoError(t, err)
		assert.Empty(t, rows)
	})
}
