package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/mentions"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

func mentionsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func seedMentionsUser(t *testing.T, s *storage.Store, id string, owner bool) {
	t.Helper()
	_, _, err := s.Users.UpsertUser(context.Background(), &auth.User{ID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: id})
	require.NoError(t, err)
	if owner {
		require.NoError(t, s.Users.SetCanCreateWorkspace(context.Background(), id, true))
	}
}

// TestIntegration_MentionsOverRealStorage covers ticket/doc chip resolution over real SQLite, access-aware.
func TestIntegration_MentionsOverRealStorage(t *testing.T) {
	ctx := context.Background()
	db := mentionsTestDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))

	seedMentionsUser(t, s, "u-alice", false)
	seedMentionsUser(t, s, "u-owner", true)

	aliceCtx := identity.WithActor(ctx, identity.Actor{ID: "u-alice"})
	ownerCtx := identity.WithActor(ctx, identity.Actor{ID: "u-owner"})

	// Two docs: one granted to alice, one she cannot open.
	require.NoError(t, s.Docs.Create(ctx, &docs.Doc{ID: "d-open", Title: "Architecture notes", Body: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`, Version: 1}))
	require.NoError(t, s.Docs.Create(ctx, &docs.Doc{ID: "d-locked", Title: "Secret vault", Body: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"hi"}]}]}`, Version: 1}))
	require.NoError(t, s.Access.Set(ctx, "doc", "d-open", "u-alice", permissions.SetOf(permissions.DocsRead), nil))

	ticket := &tickets.Ticket{ID: "tk-1", ProjectID: "project-general", Title: "Fix the bug", Body: "body", Status: tickets.StatusOpen, Assignee: "onik97"}
	require.NoError(t, s.Tickets.Create(ctx, ticket))
	require.NoError(t, s.Tickets.UpdateStatus(ctx, ticket.ID, tickets.StatusOpen))

	accessSvc := access.NewService(s.Access, accessUsers{users: s.Users})
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
		// "project-general" has no prefix, so ProjectPrefix is empty though Number is real; TypeLabel resolves to "task".
		assert.Equal(t, "", chips[0].ProjectPrefix)
		assert.Equal(t, 1, chips[0].ProjectNumber)
		assert.Equal(t, "task", chips[0].TypeLabel)
		assert.Equal(t, "onik97", chips[0].AssigneeLabel)
		assert.Equal(t, "", chips[0].DueLabel, "no due-date concept exists yet (ticket 05 flagged decision)")

		assert.Equal(t, "Architecture notes", chips[1].Title)
		assert.True(t, chips[1].CanOpen)

		assert.Equal(t, "Secret vault", chips[2].Title, "inaccessible target still discloses title")
		assert.False(t, chips[2].CanOpen, "inaccessible target is inert")
	})

	t.Run("granted user opens the doc", func(t *testing.T) {
		// can_create_workspace doesn't bypass doc checks; u-owner proves access via an explicit grant like any other user.
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
		results, err := svc.Search(aliceCtx, "vault", 10)
		require.NoError(t, err)
		for _, r := range results {
			assert.NotEqual(t, "d-locked", r.ID, "unreadable doc never appears in the picker")
		}
	})

	t.Run("picker search finds tickets with status label", func(t *testing.T) {
		results, err := svc.Search(aliceCtx, "fix", 10)
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

		results, err := svc.Search(aliceCtx, "ERF-1", 10)
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
