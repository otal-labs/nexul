package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// archiveFixture is Acme, a second workspace with a project, a ticket and a doc, a member reader, and uClerk as its Owner but not the instance Owner.
type archiveFixture struct {
	permFixture
	acme   *tenancy.Workspace
	ticket *tickets.Ticket
}

func newArchiveFixture(t *testing.T) archiveFixture {
	t.Helper()
	f := newPermFixture(t)
	owner := as(uOwner)
	acme, err := f.svc.tenancySvc.Create(owner, uOwner, "Acme")
	require.NoError(t, err)
	project, err := f.svc.workspaceSvc.Create(owner, acme.ID, "Web", "WEB", "")
	require.NoError(t, err)
	ticket, err := f.svc.ticketsSvc.Create(owner, project.ID, "Lantern flicker", "", "", "")
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, f.store.Docs.Create(owner, &docs.Doc{ID: "doc-acme", ProjectID: project.ID, Title: "Lantern", Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Docs.Create(owner, &docs.Doc{ID: "doc-general", ProjectID: pGeneral, Title: "Lantern maintenance schedule for the evening shift", Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Roles.Create(owner, &roles.Role{ID: "role-acme-reader", WorkspaceID: acme.ID, Name: "Reader", Permissions: grant("tickets:read", "docs:read"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(owner, &tenancy.Member{UserID: uReader, WorkspaceID: acme.ID, RoleID: "role-acme-reader", CreatedAt: now}))
	require.NoError(t, f.store.Access.SetRole(owner, "project", project.ID, "role-acme-reader", grant(string(permissions.ProjectsOpen)), nil))
	ownerRole, err := f.svc.tenancySvc.MemberRoleID(owner, acme.ID, uOwner)
	require.NoError(t, err)
	require.NoError(t, f.store.WorkspaceMembers.AddMember(owner, &tenancy.Member{UserID: uClerk, WorkspaceID: acme.ID, RoleID: ownerRole, CreatedAt: now}))
	// A computer notice names no project, so only the workspace's own state can take it out of the badge.
	require.NoError(t, f.store.Notifications.CreateMany(owner, []*workspace.Notification{{
		ID: "n-acme", UserID: uReader, WorkspaceID: acme.ID, Kind: workspace.KindComputerMove, SubjectType: workspace.SubjectComputer,
		SubjectID: "computer-1", SubjectTitle: "Laptop", CreatedAt: now,
	}}))
	return archiveFixture{permFixture: f, acme: acme, ticket: ticket}
}

// seen is what a member finds of Acme on each surface, so one comparison after an archive says which surface let it go.
type seen struct {
	listed, access, tickets, docs, inbox, badge, projects, ticketFrame bool
}

func (f archiveFixture) seenBy(t *testing.T, userID string) seen {
	t.Helper()
	ctx := as(userID)
	var s seen
	list, err := f.svc.tenancySvc.ListForUser(ctx, userID)
	require.NoError(t, err)
	for _, w := range list {
		s.listed = s.listed || w.ID == f.acme.ID
	}
	_, err = f.svc.tenancySvc.MyAccess(ctx, f.acme.ID, userID)
	s.access = err == nil
	found, err := f.svc.ticketsSvc.Search(ctx, "lantern", 10)
	require.NoError(t, err)
	s.tickets = slices.ContainsFunc(found, func(r tickets.SearchResult) bool { return r.ID == f.ticket.ID })
	hits, err := f.svc.docsSvc.Search(ctx, "lantern", 10)
	require.NoError(t, err)
	s.docs = slices.ContainsFunc(hits, func(r docs.SearchResult) bool { return r.ID == "doc-acme" })
	page, _, err := f.svc.notifSvc.Page(ctx, userID, workspace.InboxFilter{}, paging.Window{Limit: 100})
	require.NoError(t, err)
	s.inbox = slices.ContainsFunc(page, func(n *workspace.Notification) bool { return n.ID == "n-acme" })
	unread, err := f.svc.notifSvc.UnreadByWorkspace(ctx, userID, "")
	require.NoError(t, err)
	s.badge = unread[f.acme.ID] > 0
	_, err = f.svc.workspaceSvc.List(ctx, f.acme.ID)
	s.projects = err == nil
	frame, err := json.Marshal(map[string]any{"ticket": map[string]any{"project_id": f.ticket.ProjectID}})
	require.NoError(t, err)
	audience := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	s.ticketFrame = audience.allows(ctx, tickets.TopicCreated, json.RawMessage(frame))
	return s
}

// TestIntegration_ArchivedWorkspace_LeavesEverySurfaceAndRestoreBringsItBack: archiving takes a workspace off its members' surfaces and restoring returns all of it.
func TestIntegration_ArchivedWorkspace_LeavesEverySurfaceAndRestoreBringsItBack(t *testing.T) {
	f := newArchiveFixture(t)
	everything := seen{true, true, true, true, true, true, true, true}
	require.Equal(t, everything, f.seenBy(t, uReader), "before the archive the reader finds Acme everywhere")

	archived, err := f.svc.tenancySvc.Archive(as(uClerk), uClerk, f.acme.ID)
	require.NoError(t, err)
	require.NotNil(t, archived.ArchivedAt)
	assert.Equal(t, seen{}, f.seenBy(t, uReader), "an archived workspace is found on none of them")
	assert.Equal(t, seen{}, f.seenBy(t, uClerk), "its Owner loses it too")
	assert.NotContains(t, workspaceIDs(t, f, uOwner), f.acme.ID)

	t.Run("a hit from an archived workspace takes no place in the limit", func(t *testing.T) {
		hits, err := f.svc.docsSvc.Search(as(uReader), "lantern", 1)
		require.NoError(t, err)
		require.Len(t, hits, 1)
		assert.Equal(t, "doc-general", hits[0].ID, "Acme's doc ranks first, and the reader's other workspace still answers")
	})

	t.Run("only the instance Owner sees the archived list and restores", func(t *testing.T) {
		list, err := f.svc.tenancySvc.ListArchived(as(uOwner), uOwner)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, f.acme.ID, list[0].ID)
		assert.NotNil(t, list[0].ArchivedAt)

		for _, who := range []string{uClerk, uReader, uOutsider} {
			_, err := f.svc.tenancySvc.ListArchived(as(who), who)
			assert.ErrorIs(t, err, apperrs.ErrForbidden, who)
			_, err = f.svc.tenancySvc.Restore(as(who), who, f.acme.ID)
			assert.ErrorIs(t, err, apperrs.ErrForbidden, who)
		}
		assert.Equal(t, seen{}, f.seenBy(t, uReader), "a refused restore changes nothing")
	})

	t.Run("the archive frame reaches the members it names and whoever restores", func(t *testing.T) {
		var event tenancy.WorkspaceArchiveEvent
		var raw string
		require.NoError(t, f.db.QueryRow(`SELECT CAST(payload AS TEXT) FROM outbox WHERE topic = ?`, tenancy.TopicWorkspaceArchived).Scan(&raw))
		require.NoError(t, json.Unmarshal([]byte(raw), &event))
		assert.ElementsMatch(t, []string{uOwner, uClerk, uReader}, event.UserIDs)
		audience := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
		for who, want := range map[string]bool{uReader: true, uClerk: true, uOwner: true, uSteward: true, uPlain: false, uOutsider: false} {
			assert.Equal(t, want, audience.allows(as(who), tenancy.TopicWorkspaceArchived, json.RawMessage(raw)), who)
		}
	})

	restored, err := f.svc.tenancySvc.Restore(as(uOwner), uOwner, f.acme.ID)
	require.NoError(t, err)
	assert.Nil(t, restored.ArchivedAt)
	assert.Equal(t, everything, f.seenBy(t, uReader), "restoring brings back everything the reader had")
	held, err := f.svc.tenancySvc.MemberRoles(as(uReader), f.acme.ID, uReader)
	require.NoError(t, err)
	require.Len(t, held, 1)
	assert.Equal(t, "role-acme-reader", held[0].ID, "and the role they held")
}

func workspaceIDs(t *testing.T, f archiveFixture, userID string) []string {
	t.Helper()
	list, err := f.svc.tenancySvc.ListForUser(as(userID), userID)
	require.NoError(t, err)
	out := make([]string, 0, len(list))
	for _, w := range list {
		out = append(out, w.ID)
	}
	return out
}

func TestIntegration_ArchiveIsRefusedWithoutWorkspacesDeleteAndForTheInstancesOwnWorkspace(t *testing.T) {
	f := newArchiveFixture(t)

	now := time.Now()
	require.NoError(t, f.store.Roles.Create(as(uOwner), &roles.Role{ID: "role-acme-editor", WorkspaceID: f.acme.ID, Name: "Editor", Permissions: grant("workspaces:write"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(as(uOwner), &tenancy.Member{UserID: uEditor, WorkspaceID: f.acme.ID, RoleID: "role-acme-editor", CreatedAt: now}))

	_, err := f.svc.tenancySvc.Archive(as(uReader), uReader, f.acme.ID)
	assert.ErrorIs(t, err, apperrs.ErrForbidden, "a member without workspaces:delete")
	_, err = f.svc.tenancySvc.Archive(as(uEditor), uEditor, f.acme.ID)
	assert.ErrorIs(t, err, apperrs.ErrForbidden, "renaming a workspace is not removing it")
	_, err = f.svc.tenancySvc.Archive(as(uOutsider), uOutsider, f.acme.ID)
	assert.ErrorIs(t, err, apperrs.ErrForbidden, "someone outside it")
	_, err = f.svc.tenancySvc.Archive(as(uOwner), uOwner, wsDefault)
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "the Owner of the instance's own workspace is who restores")

	list, err := f.svc.tenancySvc.ListArchived(as(uOwner), uOwner)
	require.NoError(t, err)
	assert.Empty(t, list, "none of those archived anything")
}

func TestArchiveRoutes_ArchivingTakesTheWorkspaceOutOfTheListAndRestoreBringsItBack(t *testing.T) {
	h, store, svc := newRouter(t)
	_, token := signedInOwner(t, store, svc)
	ids := func(path string) []string {
		rec := callAPI(t, h, http.MethodGet, path, token, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var list []tenancy.Workspace
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		out := make([]string, 0, len(list))
		for _, w := range list {
			out = append(out, w.ID)
		}
		return out
	}

	rec := callAPI(t, h, http.MethodPost, "/api/workspaces", token, `{"name":"Acme"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var acme tenancy.Workspace
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &acme))
	require.Contains(t, ids("/api/workspaces"), acme.ID)

	rec = callAPI(t, h, http.MethodPost, "/api/workspaces/"+acme.ID+"/archive", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, ids("/api/workspaces"), acme.ID)
	assert.Equal(t, []string{acme.ID}, ids("/api/workspaces/archived"))
	assert.Equal(t, http.StatusNotFound, callAPI(t, h, http.MethodGet, "/api/workspaces/"+acme.ID+"/me", token, "").Code)

	rec = callAPI(t, h, http.MethodPost, "/api/workspaces/"+acme.ID+"/restore", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, ids("/api/workspaces"), acme.ID)
	assert.Empty(t, ids("/api/workspaces/archived"))
}
