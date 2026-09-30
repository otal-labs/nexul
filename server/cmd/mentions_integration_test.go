package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

func mentionsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	require.NoError(t, testutil.SeedGeneralProject(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func seedMentionsUser(t *testing.T, s *storage.Store, id string) {
	t.Helper()
	_, _, err := s.Users.UpsertUser(context.Background(), &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: id})
	require.NoError(t, err)
}

// TestIntegration_MentionsOverRealStorage covers ticket/doc chip resolution over real SQLite, access-aware.
func TestIntegration_MentionsOverRealStorage(t *testing.T) {
	ctx := context.Background()
	db := mentionsTestDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))

	seedMentionsUser(t, s, "u-alice")
	seedMentionsUser(t, s, "u-owner")

	aliceCtx := identity.WithActor(ctx, identity.Actor{ID: "u-alice"})
	ownerCtx := identity.WithActor(ctx, identity.Actor{ID: "u-owner"})

	// Two docs: one granted to alice, one she cannot open.
	require.NoError(t, s.Docs.Create(ctx, &docs.Doc{ID: "d-open", Title: "Architecture notes", Body: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`, Version: 1}))
	require.NoError(t, s.Docs.Create(ctx, &docs.Doc{ID: "d-locked", Title: "Secret vault", Body: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`, Version: 1}))
	require.NoError(t, s.Access.Set(ctx, "doc", "d-open", "u-alice", permissions.SetOf(permissions.DocsRead), nil))

	ticket := &tickets.Ticket{ID: "tk-1", ProjectID: "project-general", Title: "Fix the bug", Body: "body", Status: tickets.StatusOpen, TypeID: "ticket-type-task", Developer: "onik97"}
	require.NoError(t, s.Tickets.Create(ctx, ticket))
	require.NoError(t, s.Tickets.UpdateStatus(ctx, ticket.ID, tickets.StatusOpen))

	accessSvc := access.NewService(s.Access, accessUsers{users: s.Users})
	accessSvc.SetScopes(accessScopes{projects: s.Projects, workspaces: s.Workspaces})
	require.NoError(t, s.Access.Set(ctx, "workspace", "workspace-default", "u-alice", permissions.SetOf(permissions.TicketsRead), nil))
	require.NoError(t, s.Access.Set(ctx, "workspace", "workspace-default", "u-owner", permissions.SetOf(permissions.TicketsRead), nil))
	svc := mentions.New(mentions.Config{
		Tickets:     mentionTicketSource{repo: s.Tickets},
		Docs:        mentionDocSource{repo: s.Docs},
		Statuses:    mentionStatusSource{repo: s.Statuses},
		Access:      accessSvc,
		Projects:    mentionProjectSource{repo: s.Projects},
		TicketTypes: mentionTicketTypeSource{repo: s.TicketTypes},
	})

	t.Run("resolves chips with current title/status and access", func(t *testing.T) {
		chips, err := svc.Resolve(aliceCtx, []mentions.Ref{
			{Type: "ticket", ID: ticket.ID},
			{Type: "doc", ID: "d-open"},
			{Type: "doc", ID: "d-locked"},
		})
		require.NoError(t, err)
		require.Len(t, chips, 3)

		assert.Equal(t, "Fix the bug", chips[0].Title)
		assert.Equal(t, "open", chips[0].Status)
		assert.Equal(t, "Open", chips[0].StatusLabel)
		assert.True(t, chips[0].CanOpen)
		// "project-general" has no prefix, so ProjectPrefix is empty though Number is real.
		assert.Equal(t, "", chips[0].ProjectPrefix)
		assert.Equal(t, 1, chips[0].ProjectNumber)
		assert.Equal(t, "task", chips[0].TypeLabel)
		assert.Equal(t, "onik97", chips[0].DeveloperLabel)
		assert.Equal(t, "", chips[0].DueLabel, "no due-date concept exists yet (ticket 05 flagged decision)")

		assert.Equal(t, "Architecture notes", chips[1].Title)
		assert.True(t, chips[1].CanOpen)

		assert.Equal(t, "Secret vault", chips[2].Title, "inaccessible target still discloses title")
		assert.False(t, chips[2].CanOpen, "inaccessible target is inert")
	})

	t.Run("granted user opens the doc", func(t *testing.T) {
		// u-owner is no workspace's Owner here, so it proves access via an explicit grant like any other user.
		require.NoError(t, s.Access.Set(ctx, "doc", "d-locked", "u-owner", permissions.SetOf(permissions.DocsRead), nil))
		chips, err := svc.Resolve(ownerCtx, []mentions.Ref{{Type: "doc", ID: "d-locked"}})
		require.NoError(t, err)
		require.Len(t, chips, 1)
		assert.True(t, chips[0].CanOpen)
	})

	t.Run("missing targets are omitted", func(t *testing.T) {
		chips, err := svc.Resolve(aliceCtx, []mentions.Ref{{Type: "ticket", ID: "nope"}})
		require.NoError(t, err)
		assert.Empty(t, chips)
	})

	t.Run("picker search excludes unreadable docs", func(t *testing.T) {
		results, err := svc.Search(aliceCtx, "vault", "", 10)
		require.NoError(t, err)
		for _, r := range results {
			assert.NotEqual(t, "d-locked", r.ID, "unreadable doc never appears in the picker")
		}
	})

	t.Run("picker search finds tickets with status label", func(t *testing.T) {
		results, err := svc.Search(aliceCtx, "fix", "", 10)
		require.NoError(t, err)
		var found *mentions.SearchResult
		for i := range results {
			if results[i].Type == "ticket" && results[i].ID == ticket.ID {
				found = &results[i]
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "Fix the bug", found.Title)
		assert.Equal(t, "Open", found.StatusLabel)
		assert.True(t, found.CanOpen)
	})

	t.Run("picker search resolves @PREFIX-NUMBER directly (ticket 06)", func(t *testing.T) {
		now := time.Now()
		require.NoError(t, s.Projects.Create(ctx, &workspace.Project{
			ID: "p-erf", Name: "Engineering", Prefix: "ERF", WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now,
		}))
		keyTicket := &tickets.Ticket{ID: "tk-erf-1", ProjectID: "p-erf", Title: "Ship the router rewrite", Body: "body", Status: tickets.StatusOpen}
		require.NoError(t, s.Tickets.Create(ctx, keyTicket)) // first ticket in p-erf, so number 1 -> ERF-1

		results, err := svc.Search(aliceCtx, "ERF-1", "", 10)
		require.NoError(t, err)
		require.NotEmpty(t, results, "typing @ERF-1 must resolve the ticket directly, not just via title search")
		assert.Equal(t, "ticket", results[0].Type)
		assert.Equal(t, keyTicket.ID, results[0].ID, "exact key match sorts first")
		assert.Equal(t, "Ship the router rewrite", results[0].Title)

		// Resolve renders {ticket.Project} as PREFIX-NUMBER for a project with a prefix, unlike "project-general" above.
		chips, err := svc.Resolve(aliceCtx, []mentions.Ref{{Type: "ticket", ID: keyTicket.ID}})
		require.NoError(t, err)
		require.Len(t, chips, 1)
		assert.Equal(t, "ERF", chips[0].ProjectPrefix)
		assert.Equal(t, 1, chips[0].ProjectNumber)
	})
}

// seedMentionPeople makes onik (writes docs and tickets), rixwavedev (reads them), and sam ("Rixa Stone", reads
// neither) members of the default workspace, and rixoutsider an account outside it.
func seedMentionPeople(t *testing.T, store *storage.Store) {
	t.Helper()
	ctx := context.Background()
	for id, login := range map[string]string{"u-onik": "onik", "u-rix": "rixwavedev", "u-sam": "sam", "u-out": "rixoutsider"} {
		_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: login})
		require.NoError(t, err)
	}
	for id, name := range map[string]string{"u-sam": "Rixa Stone", "u-onik": "Onik"} {
		require.NoError(t, store.Users.SetProfileOverride(ctx, id, &name, nil))
	}
	now := time.Now()
	for _, r := range []*roles.Role{
		{ID: "role-writer", Name: "Writer", Permissions: grant("docs:read", "docs:write", "tickets:read", "tickets:write")},
		{ID: "role-reader", Name: "Reader", Permissions: grant("docs:read", "tickets:read")},
		{ID: "role-member", Name: "Member"},
	} {
		r.WorkspaceID, r.CreatedAt, r.UpdatedAt = "workspace-default", now, now
		require.NoError(t, store.Roles.Create(ctx, r))
	}
	for user, role := range map[string]string{"u-onik": "role-writer", "u-rix": "role-reader", "u-sam": "role-member"} {
		require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role, CreatedAt: now}))
	}
}

// TestMentionSearch_FindsWorkspacePeople guards the doc and ticket @ picker finding the workspace's people, as the
// new-DM dialog does, by login and by display name, and never someone outside the workspace.
func TestMentionSearch_FindsWorkspacePeople(t *testing.T) {
	svc, store := newWired(t)
	seedMentionPeople(t, store)
	routes := mentions.NewHandler(svc.mentionsSvc).Routes()

	for _, workspaceID := range []string{"", "workspace-default"} {
		t.Run("workspace "+workspaceID, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/mentions/search?q=rix&limit=8&workspace_id="+workspaceID, nil)
			req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: "u-onik"}))
			rec := httptest.NewRecorder()
			routes.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			var body struct {
				Results []struct {
					Type  string `json:"type"`
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"results"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			got := map[string]string{}
			for _, r := range body.Results {
				if r.Type == "person" {
					got[r.ID] = r.Title
				}
			}
			assert.Equal(t, map[string]string{"u-rix": "rixwavedev", "u-sam": "Rixa Stone"}, got)
		})
	}
}

// deliverNotifications runs every unpublished doc and ticket event through the inbox's handlers, as the relay would.
func deliverNotifications(t *testing.T, svc *coreServices, store *storage.Store) {
	t.Helper()
	ctx := context.Background()
	handlers := map[string]func(context.Context, *workspace.NotificationService, eventbus.Event) error{
		docs.TopicCreated: workspace.HandleDocCreated, docs.TopicUpdated: workspace.HandleDocUpdated,
		tickets.TopicCreated: workspace.HandleTicketCreated, tickets.TopicUpdated: workspace.HandleTicketUpdated,
	}
	entries, err := store.Outbox.Unpublished(ctx, 500)
	require.NoError(t, err)
	for _, e := range entries {
		if handle, ok := handlers[e.Topic]; ok {
			require.NoError(t, handle(ctx, svc.notifSvc, eventbus.Event{ID: e.ID, Topic: e.Topic, Payload: e.Payload}))
		}
		require.NoError(t, store.Outbox.MarkPublished(ctx, e.ID))
	}
}

// inbox lists a person's notifications of one kind as their titles.
func inbox(t *testing.T, svc *coreServices, userID string, kind workspace.Kind) []string {
	t.Helper()
	ns, err := svc.notifSvc.List(context.Background(), userID, "", 100)
	require.NoError(t, err)
	var titles []string
	for _, n := range ns {
		if n.Kind == kind {
			titles = append(titles, n.SubjectTitle)
		}
	}
	return titles
}

func personMentionBody(userIDs ...string) string {
	nodes := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		nodes = append(nodes, `{"type":"mention","attrs":{"type":"person","id":"`+id+`","label":"x"}}`)
	}
	return `{"type":"doc","content":[{"type":"paragraph","content":[` + strings.Join(nodes, ",") + `]}]}`
}

// TestPersonMentions_NotifyOnceAndOnlyReaders saves a doc and a ticket the way the editor does and checks the inbox:
// a newly mentioned reader is told once, never the author, never someone who cannot read it, never a non-member.
func TestPersonMentions_NotifyOnceAndOnlyReaders(t *testing.T) {
	svc, store := newWired(t)
	seedMentionPeople(t, store)
	onik := as("u-onik")

	t.Run("doc", func(t *testing.T) {
		doc, err := svc.docsSvc.Create(onik, "project-general", "Launch plan", personMentionBody("u-rix", "u-onik"))
		require.NoError(t, err)
		deliverNotifications(t, svc, store)
		assert.Equal(t, []string{"Onik mentioned you in Launch plan"}, inbox(t, svc, "u-rix", workspace.KindDocMentioned))
		assert.Empty(t, inbox(t, svc, "u-onik", workspace.KindDocMentioned), "a self-mention notifies nobody")
		// Read, so the inbox's collapse of repeated unread rows cannot hide a second mention.
		require.NoError(t, svc.notifSvc.MarkAllRead(context.Background(), "u-rix", ""))

		_, err = svc.docsSvc.Update(onik, doc.ID, "Launch plan", personMentionBody("u-rix", "u-onik", "u-sam", "u-out"))
		require.NoError(t, err)
		deliverNotifications(t, svc, store)
		assert.Len(t, inbox(t, svc, "u-rix", workspace.KindDocMentioned), 1, "re-saving an existing mention does not notify again")
		assert.Empty(t, inbox(t, svc, "u-sam", workspace.KindDocMentioned), "sam cannot read docs")
		assert.Empty(t, inbox(t, svc, "u-out", workspace.KindDocMentioned), "rixoutsider is not a member")
	})

	t.Run("ticket", func(t *testing.T) {
		ticket, err := svc.ticketsSvc.Create(onik, "project-general", "Fix login", "", "", "")
		require.NoError(t, err)
		for range 2 {
			_, err = svc.ticketsSvc.UpdateTicket(onik, ticket.ID, "Fix login", personMentionBody("u-rix"))
			require.NoError(t, err)
			deliverNotifications(t, svc, store)
			require.NoError(t, svc.notifSvc.MarkAllRead(context.Background(), "u-rix", ""))
		}
		assert.Equal(t, []string{"Onik mentioned you in Fix login"}, inbox(t, svc, "u-rix", workspace.KindTicketMentioned))
	})
}
