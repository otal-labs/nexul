package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

const (
	uClient = "u-client"
	// uUnlisted holds a role no project lists, and uNoRole holds no role at all.
	uUnlisted = "u-unlisted"
	uNoRole   = "u-no-role"
	pClient   = "p-client"
	pGeneral  = "project-general"
	wsDefault = "workspace-default"
)

// listedClient adds uClient to the permission fixture as a Client listed in p-client only, on a role that carries
// reads, writes, and every instance-level bit, so anything they reach beyond p-client would be the evaluator's fault,
// not the role's.
func listedClient(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uClient, Provider: auth.ProviderGitHub, ProviderUserID: uClient, Login: uClient})
	require.NoError(t, err)
	clientRole := append([]string{"tickets:read", "tickets:write", "docs:read", "stacks:read", "deploys:read", "projects:read", "projects:write", "plays:read", "chat:write"}, instanceBits...)
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-client", WorkspaceID: wsDefault, Name: "Client", Permissions: grant(clientRole...), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uClient, WorkspaceID: wsDefault, RoleID: "role-client", CreatedAt: now}))
	require.NoError(t, f.store.Projects.Create(ctx, &workspace.Project{ID: pClient, Name: "Client portal", Prefix: "CLI", WorkspaceID: wsDefault, CreatedAt: now, UpdatedAt: now}))
	listRolesIn(t, f.db, pClient)
}

// unlistedMembers adds uUnlisted, whose role reads tickets and docs but no project lists, and uNoRole, a member with
// no role: neither may reach anything in a project.
func unlistedMembers(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-unlisted", WorkspaceID: wsDefault, Name: "Unlisted", Permissions: grant("tickets:read", "docs:read", "chat:write"), CreatedAt: now, UpdatedAt: now}))
	for user, role := range map[string]string{uUnlisted: "role-unlisted", uNoRole: "role-unlisted"} {
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user})
		require.NoError(t, err)
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: wsDefault, RoleID: role, CreatedAt: now}))
	}
	_, err := f.db.Exec(`DELETE FROM member_roles WHERE user_id = ?`, uNoRole)
	require.NoError(t, err)
}

// listClientInGeneral lists uClient in the fixture's project by name, or takes that rule back to Fallback, which
// leaves them out, since their role isn't listed there.
func listClientInGeneral(t *testing.T, f permFixture, in bool) {
	t.Helper()
	state := access.StateFallback
	if in {
		state = access.StateAllow
	}
	require.NoError(t, f.svc.accessSvc.SetRules(context.Background(), uOwner, "project", []string{pGeneral}, access.Targets{UserIDs: []string{uClient}},
		[]permissions.Action{permissions.ProjectsOpen}, state))
}

func TestIntegration_NotInAProject(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)
	s := f.svc
	client := as(uClient)

	file, err := s.attachmentsSvc.Upload(as(uOwner), attachments.Owner{TicketID: f.ticket.ID}, "shot.png", []byte("\x89PNG\r\n\x1a\n"))
	require.NoError(t, err)
	require.NoError(t, f.store.Access.Set(context.Background(), "doc", f.doc, uClient, grant("docs:read"), nil))

	t.Run("everything in a project nothing lists them in reads as not found", func(t *testing.T) {
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

	t.Run("the project list holds only their project, and projects:write alone creates none", func(t *testing.T) {
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
		assert.NoError(t, err, "the role answers inside a project they are in")
	})

	t.Run("every membership counts for instance-level areas", func(t *testing.T) {
		held, err := s.accessSvc.PermissionsAnywhere(context.Background(), uClient)
		require.NoError(t, err)
		assert.Subset(t, held, instanceBits, "instance_permissions on /api/auth/me reads this")
		runs, err := s.accessSvc.HoldsAnywhere(context.Background(), uClient, permissions.MachinesCreate)
		require.NoError(t, err)
		assert.True(t, runs)
	})

	t.Run("a developer or tester who cannot open the project is refused", func(t *testing.T) {
		_, err := s.ticketsSvc.SetPerson(as(uOwner), f.ticket.ID, tickets.RoleDeveloper, uClient)
		assert.Equal(t, invalid, outcome(err))
		own, err := s.ticketsSvc.Create(as(uOwner), pClient, "Theirs", "", "", "")
		require.NoError(t, err)
		_, err = s.ticketsSvc.SetPerson(as(uOwner), own.ID, tickets.RoleTester, uClient)
		assert.NoError(t, err)
	})

	t.Run("the developer picker lists who the project's rules let in", func(t *testing.T) {
		people := func(projectID string) []string {
			ps, err := s.tenancySvc.ListProjectPeople(context.Background(), uOwner, projectID)
			require.NoError(t, err)
			ids := []string{}
			for _, p := range ps {
				ids = append(ids, p.UserID)
			}
			return ids
		}
		assert.NotContains(t, people(pGeneral), uClient)
		assert.Contains(t, people(pClient), uClient)
		listClientInGeneral(t, f, true)
		assert.Contains(t, people(pGeneral), uClient, "listed by name")
		require.NoError(t, s.accessSvc.SetRules(context.Background(), uOwner, "project", []string{pClient}, access.Targets{UserIDs: []string{uClient}},
			[]permissions.Action{permissions.ProjectsOpen}, access.StateDeny))
		assert.NotContains(t, people(pClient), uClient, "kept out by name, whatever their role")
		assert.Contains(t, people(pClient), uOwner, "the Owner is in every project")
	})
}

// Notices and trails on a project they are not in vanish from reads and come back once they are listed; the rows are
// kept.
func TestIntegration_NotInAProject_NoticesAndTrailsFollowAccess(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)
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
		list, err := runner.ListTrails(as(uClient), plays.TargetTicket, f.ticket.ID, "")
		require.NoError(t, err)
		_, err = runner.GetTrail(as(uClient), "tr-1")
		return len(ns), n, len(list), outcome(err)
	}
	notices, unread, trails, get := visible()
	assert.Equal(t, []any{0, 0, 0, notFound}, []any{notices, unread, trails, get}, "hidden while they are not in the project")

	listClientInGeneral(t, f, true)
	notices, unread, trails, get = visible()
	assert.Equal(t, []any{1, 1, 1, ok}, []any{notices, unread, trails, get}, "back once they are listed")
}

// A live ticket frame stops reaching a socket the moment its person leaves the project's list.
func TestIntegration_NotInAProject_LiveFramesFollowAccess(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	frame := tickets.UpdatedEvent{Ticket: *f.ticket}
	reaches := func() bool { return a.allows(as(uClient), tickets.TopicUpdated, frame) }

	assert.False(t, reaches())
	listClientInGeneral(t, f, true)
	assert.True(t, reaches())
	listClientInGeneral(t, f, false)
	assert.False(t, reaches(), "taken back out, the next frame is refused")

	person := access.GrantEvent{ResourceType: "project", ResourceID: pGeneral, Target: access.TargetPerson, UserID: uClient, ActorID: uOwner}
	assert.True(t, a.allows(as(uClient), access.TopicGrantChanged, person), "the person themselves")
	assert.True(t, a.allows(as(uManager), access.TopicGrantChanged, person), "a holder of projects:access in the project")
	assert.False(t, a.allows(as(uReader), access.TopicGrantChanged, person))
	role := access.GrantEvent{ResourceType: "project", ResourceID: pClient, Target: access.TargetRole, RoleID: "role-client", ActorID: uOwner}
	assert.True(t, a.allows(as(uClient), access.TopicGrantChanged, role), "a holder of the role")
	assert.False(t, a.allows(as(uReader), access.TopicGrantChanged, role))
}

// A project's rules are written by whoever manages its access, judged by what each write gives, and go with it.
func TestIntegration_ProjectRules_GrantingAndCleanup(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)
	s := f.svc
	ctx := context.Background()
	open := []permissions.Action{permissions.ProjectsOpen}
	mobile, err := s.workspaceSvc.Create(as(uWriter), wsDefault, "Mobile", "MOB", workspace.ProjectIcon(""))
	require.NoError(t, err)

	t.Run("the creator is in it, with their roles, and manages it", func(t *testing.T) {
		assert.True(t, s.accessSvc.CanInProject(ctx, uWriter, mobile.ID, permissions.ProjectsAccess))
		assert.False(t, s.accessSvc.CanInProject(ctx, uManager, mobile.ID, permissions.Member), "a role the creator lacks is not listed")
		require.NoError(t, s.accessSvc.SetRules(ctx, uWriter, "project", []string{mobile.ID}, access.Targets{UserIDs: []string{uManager}}, open, access.StateAllow))
		assert.True(t, s.accessSvc.CanInProject(ctx, uManager, mobile.ID, permissions.Member))
	})

	t.Run("nobody gives what they don't hold there, and keeping someone out is never limited", func(t *testing.T) {
		err := s.accessSvc.SetRules(ctx, uWriter, "project", []string{mobile.ID}, access.Targets{RoleIDs: []string{"role-plain"}}, []permissions.Action{permissions.TicketsDelete}, access.StateAllow)
		assert.Equal(t, forbidden, outcome(err))
		require.NoError(t, s.accessSvc.SetRules(ctx, uManager, "project", []string{mobile.ID}, access.Targets{UserIDs: []string{uWriter}}, open, access.StateDeny))
		assert.False(t, s.accessSvc.CanInProject(ctx, uWriter, mobile.ID, permissions.Member), "even the creator can be kept out")
		err = s.accessSvc.SetRules(ctx, uWriter, "project", []string{mobile.ID}, access.Targets{UserIDs: []string{uWriter}}, open, access.StateAllow)
		assert.Equal(t, notFound, outcome(err), "a project they are no longer in")
		err = s.accessSvc.SetRules(ctx, uReader, "project", []string{pGeneral}, access.Targets{UserIDs: []string{uClient}}, open, access.StateAllow)
		assert.Equal(t, forbidden, outcome(err), "writing a project's rules takes projects:access")
	})

	t.Run("each change publishes the grant event through the outbox", func(t *testing.T) {
		entries, err := f.store.Outbox.Unpublished(ctx, 500)
		require.NoError(t, err)
		found := false
		for _, e := range entries {
			found = found || (e.Topic == access.TopicGrantChanged && strings.Contains(string(e.Payload), `"resource_type":"project"`))
		}
		assert.True(t, found)
	})

	t.Run("deleting a project deletes its rules", func(t *testing.T) {
		require.NoError(t, s.workspaceSvc.Delete(as(uOwner), mobile.ID))
		rules, err := f.store.Access.ListByResource(ctx, "project", mobile.ID)
		require.NoError(t, err)
		assert.Empty(t, rules)
	})
}

// A mention chip of a ticket or doc in a project the reader is not in still shows what it points at, locked.
func TestIntegration_NotInAProject_PillsShowTheirContentLocked(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)
	s := f.svc
	_, err := s.ticketsSvc.SetPerson(as(uOwner), f.ticket.ID, tickets.RoleDeveloper, uWriter)
	require.NoError(t, err)

	chips, err := s.mentionsSvc.Resolve(as(uClient), wsDefault, []mentions.Ref{{Type: "ticket", ID: f.ticket.ID}, {Type: "doc", ID: f.doc}})
	require.NoError(t, err)
	require.Len(t, chips, 2)
	ticket, doc := chips[0], chips[1]
	assert.False(t, ticket.CanOpen)
	assert.Equal(t, "Login times out", ticket.Title)
	assert.NotEmpty(t, ticket.StatusLabel)
	assert.Equal(t, uWriter, ticket.DeveloperLabel)
	assert.False(t, doc.CanOpen)
	assert.Equal(t, "Spec", doc.Title)

	chips, err = s.mentionsSvc.Resolve(as(uReader), wsDefault, []mentions.Ref{{Type: "ticket", ID: f.ticket.ID}})
	require.NoError(t, err)
	assert.True(t, chips[0].CanOpen, "someone in the project opens it")
}

// A member holding no role still has what membership alone gives: People, public channels, DMs, the inbox, and a DM
// with an Owner; nothing in a project.
func TestIntegration_NoRole_ReadsWhatMembershipGivesAndNoProject(t *testing.T) {
	f := newPermFixture(t)
	unlistedMembers(t, f)
	s := f.svc
	none := as(uNoRole)

	people, err := s.tenancySvc.ListPeople(none, uNoRole, wsDefault)
	require.NoError(t, err)
	assert.NotEmpty(t, people, "People")
	conversations, err := s.chatSvc.ListConversations(none, wsDefault, uNoRole)
	require.NoError(t, err)
	assert.Contains(t, conversationIDs(conversations), f.channel.ID, "a public channel")
	_, _, err = s.notifSvc.Page(context.Background(), uNoRole, workspace.InboxFilter{WorkspaceID: wsDefault}, paging.Window{Limit: 50})
	require.NoError(t, err, "the inbox")

	dm, err := s.chatSvc.CreateDM(none, wsDefault, uNoRole, []string{uOwner})
	require.NoError(t, err, "a DM with an Owner")
	_, err = s.chatSvc.PostMessage(none, dm.ID, uNoRole, "Could I have a role?")
	require.NoError(t, err)
	_, err = s.chatSvc.CreateDM(none, wsDefault, uNoRole, []string{uWriter})
	assert.Equal(t, forbidden, outcome(err), "a DM with anyone else takes chat:write")

	projects, err := s.workspaceSvc.List(none, wsDefault)
	require.NoError(t, err)
	assert.Empty(t, projects)
	assert.Equal(t, notFound, outcome(s.accessSvc.RequireProject(none, pGeneral, permissions.Member)))
	_, err = s.ticketsSvc.Get(none, f.ticket.ID)
	assert.Equal(t, notFound, outcome(err))
}

func conversationIDs(cs []*chat.Conversation) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}

// A project's creator, and each role they hold but the Owner's, is listed in it in the same write, over HTTP and MCP.
func TestIntegration_ProjectCreate_ListsTheCreatorAndTheirRoles(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	ctx := context.Background()

	rec := httptest.NewRecorder()
	body := `{"workspace_id":"` + wsDefault + `","name":"Mobile","prefix":"MOB"}`
	workspace.NewHandler(s.workspaceSvc).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(body)).WithContext(as(uWriter)))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var overHTTP workspace.Project
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &overHTTP))

	out, err := tool(t, workspace.MCPTools(s.workspaceSvc), "project_create").Call(as(uWriter), json.RawMessage(`{"workspace_id":"`+wsDefault+`","name":"Desk","prefix":"DSK"}`))
	require.NoError(t, err)
	overMCP := out.(*workspace.Project)

	for _, p := range []string{overHTTP.ID, overMCP.ID} {
		assert.True(t, s.accessSvc.CanInProject(ctx, uWriter, p, permissions.ProjectsAccess), "the creator is in it and manages it")
		assert.False(t, s.accessSvc.CanInProject(ctx, uReader, p, permissions.Member), "a role the creator lacks is not listed")
		rules, err := f.store.Access.ListByResource(ctx, "project", p)
		require.NoError(t, err)
		require.Len(t, rules, 2)
		assert.Equal(t, uWriter, rules[0].UserID)
		assert.Equal(t, permissions.SetOf(permissions.ProjectsOpen, permissions.ProjectsAccess), rules[0].Allow)
		assert.Equal(t, "role-writer", rules[1].RoleID)
		assert.Equal(t, permissions.SetOf(permissions.ProjectsOpen), rules[1].Allow)
	}

	owners, err := s.workspaceSvc.Create(as(uOwner), wsDefault, "Ops", "OPS", workspace.ProjectIcon(""))
	require.NoError(t, err)
	rules, err := f.store.Access.ListByResource(ctx, "project", owners.ID)
	require.NoError(t, err)
	require.Len(t, rules, 1, "the Owner role is in every project, so no rule names it")
	assert.Equal(t, uOwner, rules[0].UserID)
}

// TestIntegration_MeReadsEachProjectFromItsOwnList: /me lists only the projects someone is in, with what they hold in
// each, and keeps project-area actions out of permissions, the shape a Restricted member read before the switch.
func TestIntegration_MeReadsEachProjectFromItsOwnList(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)

	me, err := f.svc.tenancySvc.MyAccess(as(uClient), wsDefault, uClient)
	require.NoError(t, err)
	assert.True(t, me.Restricted)
	assert.Contains(t, me.Permissions, "chat:write")
	assert.NotContains(t, me.Permissions, "tickets:read")
	require.Len(t, me.Projects, 1)
	assert.Equal(t, pClient, me.Projects[0].ProjectID)
	assert.True(t, me.Projects[0].Actions.Has(permissions.TicketsWrite))
	assert.False(t, me.Projects[0].Actions.Has(permissions.ProjectsDelete))

	owner, err := f.svc.tenancySvc.MyAccess(as(uOwner), wsDefault, uOwner)
	require.NoError(t, err)
	assert.False(t, owner.Restricted)
	assert.Contains(t, owner.Permissions, "tickets:read")
	assert.GreaterOrEqual(t, len(owner.Projects), 2, "the Owner is in every project")
}
