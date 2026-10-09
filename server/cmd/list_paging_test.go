package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/mcp/composite"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// tool finds name among tools, failing the test when it is missing.
func tool(t *testing.T, tools []mcptool.Tool, name string) mcptool.Tool {
	t.Helper()
	for _, tl := range tools {
		if tl.Name == name {
			return tl
		}
	}
	t.Fatalf("no tool %s", name)
	return mcptool.Tool{}
}

// callPage calls a list tool as user on a fresh request memo and decodes its page, ids and all.
func callPage(t *testing.T, a *access.Service, tl mcptool.Tool, user, args string) (ids []string, page mcptool.Page[json.RawMessage]) {
	t.Helper()
	ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), a.NewMemo())
	out, err := tl.Call(ctx, json.RawMessage(args))
	require.NoError(t, err, "%s as %s", tl.Name, user)
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &page))
	for _, item := range page.Items {
		var withID struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(item, &withID))
		ids = append(ids, withID.ID)
	}
	return ids, page
}

// TestRows_MessageListReadsOnlyItsPage: a page of 50 from a 20,000-message conversation hands back about 50 rows, where
// paging in memory read the whole history, and the page query walks the live-message index without sorting.
func TestRows_MessageListReadsOnlyItsPage(t *testing.T) {
	f, st, db := newCountedFixtureDB(t)
	_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 20000)
		INSERT INTO messages (id, conversation_id, author_id, body, created_at, updated_at, deleted_at)
		SELECT printf('m-%05d', i), ?, ?, 'status update ' || i, 1700000000 + i / 3, 1700000000 + i / 3, CASE WHEN i % 10 = 0 THEN 1700000000 END FROM n`,
		f.channel.ID, uOwner)
	require.NoError(t, err)
	messageList := tool(t, chat.MCPTools(f.svc.chatSvc), "message_list")

	st.Reset()
	ids, page := callPage(t, f.svc.accessSvc, messageList, uOwner, fmt.Sprintf(`{"conversation_id":%q,"limit":50,"offset":100}`, f.channel.ID))

	assert.Len(t, ids, 50)
	assert.Equal(t, 18000, page.Total, "every tenth message is deleted")
	assert.Equal(t, 150, page.NextOffset)
	assert.Less(t, st.Rows(), int64(100), "rows handed back for one page")

	plan := explain(t, db, `SELECT * FROM messages WHERE conversation_id = ? AND deleted_at IS NULL ORDER BY created_at DESC, id DESC LIMIT 50 OFFSET 100`, f.channel.ID)
	assert.Contains(t, plan, "idx_messages_live")
	assert.NotContains(t, plan, "TEMP B-TREE")
}

// TestRows_ListToolsReadOnlyTheirPage: over 3,000 tickets, docs, deploys, notifications, and memories, a page of 50
// hands back about its own rows and a count, where paging in memory read every row or the first thousand.
func TestRows_ListToolsReadOnlyTheirPage(t *testing.T) {
	f, st, db := newCountedFixtureDB(t)
	for _, stmt := range []string{
		`INSERT INTO tickets (id, title, body, status, project_id, created_at, updated_at, number, position)
			SELECT printf('t-%05d', i), 'Login times out ' || i, '', 'open', 'project-general', 1700000000 + i, 1700000000 + i, 100 + i, i FROM n`,
		`INSERT INTO docs (id, title, body, body_md, version, created_at, updated_at, project_id, folder_id)
			SELECT printf('d-%05d', i), 'Runbook ' || i, '{"type":"doc","content":[]}', 'rollback step ' || i, 1, 1700000000 + i, 1700000000 + i, 'project-general', 'folder-general-main' FROM n`,
		`INSERT INTO deploys (id, service, target, image, status, strategy, created_at, updated_at, stack_id)
			SELECT printf('dep-%05d', i), 'web', 'm1', 'web:' || i, 'healthy', 'run', 1700000000 + i, 1700000000 + i, 'stack-1' FROM n`,
		`INSERT INTO notifications (id, user_id, workspace_id, kind, subject_type, subject_id, subject_title, read, created_at)
			SELECT printf('n-%05d', i), 'u-owner', 'workspace-default', 'ticket_assigned', 'ticket', printf('t-%05d', i), 'Ticket', 0, 1700000000 + i FROM n`,
		`INSERT INTO memories (id, workspace_id, project_id, title, when_to_use, body, created_at, updated_at)
			SELECT printf('mem-%05d', i), 'workspace-default', 'project-general', 'Note ' || i, 'when ' || i, 'body', 1700000000 + i, 1700000000 + i FROM n`,
	} {
		_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 3000) ` + stmt)
		require.NoError(t, err)
	}
	s := f.svc
	var tools []mcptool.Tool
	tools = append(tools, composite.TicketTools(s.ticketsSvc, s.workspaceSvc, s.reviewSvc)...)
	tools = append(tools, docs.MCPTools(s.docsSvc)...)
	tools = append(tools, deploy.MCPTools(s.deploySvc)...)
	tools = append(tools, workspace.NotificationMCPTools(s.notifSvc)...)
	tools = append(tools, memories.MCPTools(s.memoriesSvc)...)
	for _, tc := range []struct{ tool, args string }{
		{"ticket_list", `{}`},
		{"ticket_list", `{"query":"login"}`},
		{"doc_list", `{}`},
		{"doc_list", `{"query":"rollback"}`},
		{"deploy_list", `{}`},
		{"notification_list", `{"offset":1500}`},
		{"memory_list", `{"project_id":"project-general"}`},
	} {
		t.Run(tc.tool+" "+tc.args, func(t *testing.T) {
			st.Reset()
			ids, page := callPage(t, s.accessSvc, tool(t, tools, tc.tool), uOwner, tc.args)

			assert.Len(t, ids, 50)
			assert.GreaterOrEqual(t, page.Total, 3000)
			assert.Less(t, st.Rows(), int64(250), "rows handed back for one page")
		})
	}
}

// explain is SQLite's query plan for query, its detail lines joined.
func explain(t *testing.T, db interface {
	Query(string, ...any) (*sql.Rows, error)
}, query string, args ...any) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var lines []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notused, &detail))
		lines = append(lines, detail)
	}
	require.NoError(t, rows.Err())
	return strings.Join(lines, "\n")
}

// pageThrough calls a list tool page by page as user, two items at a time, and returns every id it gave, failing on a
// repeat or on a total that does not match what the pages held.
func pageThrough(t *testing.T, a *access.Service, tl mcptool.Tool, user, filter string) []string {
	t.Helper()
	var all []string
	seen := map[string]bool{}
	for offset := 0; ; {
		args := fmt.Sprintf(`{%s"limit":2,"offset":%d}`, filter, offset)
		ids, page := callPage(t, a, tl, user, args)
		for _, id := range ids {
			require.False(t, seen[id], "%s gave %s twice as %s", tl.Name, id, user)
			seen[id] = true
		}
		all = append(all, ids...)
		if !page.HasMore {
			require.Equal(t, len(all), page.Total, "%s %s as %s", tl.Name, filter, user)
			return all
		}
		offset = page.NextOffset
	}
}

// listParityFixture is the permission fixture with uClient restricted to p-client, a ticket, stack, and deploy in each
// project, blockers across them, a doc each in a private and a shared state, an instance stack and a deleted stack's
// deploy, and notices in every state an inbox filters.
func listParityFixture(t *testing.T) permFixture {
	t.Helper()
	f := newPermFixture(t)
	restrictedClient(t, f)
	ctx := context.Background()
	now := time.Now()
	clientTicket, err := f.svc.ticketsSvc.Create(ctx, pClient, "Login page for the portal", "", "", "")
	require.NoError(t, err)
	blocker, err := f.svc.ticketsSvc.Create(ctx, pGeneral, "Login rate limit", "", "", "")
	require.NoError(t, err)
	require.NoError(t, f.store.Tickets.PutLink(ctx, tickets.TicketLink{TicketID: f.ticket.ID, Kind: tickets.LinkBlockedBy, TargetID: clientTicket.ID, CreatedAt: now}))
	require.NoError(t, f.store.Tickets.PutLink(ctx, tickets.TicketLink{TicketID: clientTicket.ID, Kind: tickets.LinkBlockedBy, TargetID: blocker.ID, CreatedAt: now}))

	for id, project := range map[string]string{"doc-private": pGeneral, "doc-shared": pGeneral, "doc-client": pClient} {
		require.NoError(t, f.store.Docs.Create(ctx, &docs.Doc{ID: id, ProjectID: project, Title: "Spec " + id, Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
	}
	require.NoError(t, f.store.Access.Set(ctx, "doc", "doc-private", uReader, nil, grant("docs:read")))
	require.NoError(t, f.store.Access.Set(ctx, "doc", "doc-shared", uPlain, grant("docs:read"), nil))
	require.NoError(t, f.store.Access.Set(ctx, "doc", "doc-shared", uClient, grant("docs:read"), nil))

	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-client", ProjectID: pClient, Name: "portal", Slug: "portal", Machine: "m1", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-gone", ProjectID: pGeneral, Name: "old", Slug: "old", Machine: "m1", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	for i, stack := range []string{f.infra, "stack-client", "stack-gone", f.stack} {
		at := now.Add(time.Duration(i+1) * time.Second)
		require.NoError(t, f.store.Deploys.Create(ctx, &deploy.Deploy{ID: "deploy-" + stack, StackID: stack, Service: stack, Status: deploy.StatusHealthy, CreatedAt: at, UpdatedAt: at}))
	}
	require.NoError(t, f.store.Stacks.Delete(ctx, "stack-gone"))

	var notices []*workspace.Notification
	for i, user := range []string{uOwner, uReader, uPlain, uClient, uOutsider} {
		for j, n := range []struct{ kind, subject, workspace string }{
			{string(workspace.SubjectTicket), f.ticket.ID, wsDefault}, {string(workspace.SubjectTicket), clientTicket.ID, wsDefault},
			{string(workspace.SubjectDoc), "doc-client", wsDefault}, {string(workspace.SubjectDoc), "doc-deleted", wsDefault},
			{string(workspace.SubjectTicket), f.ticket.ID, "workspace-left"}, {string(workspace.SubjectDoc), "doc-shared", ""},
		} {
			at := now.Add(time.Duration(i*10+j) * time.Second)
			notices = append(notices, &workspace.Notification{ID: fmt.Sprintf("n-%s-%d", user, j), UserID: user, WorkspaceID: n.workspace,
				Kind: workspace.Kind(fmt.Sprintf("kind-%d", j)), SubjectType: workspace.SubjectType(n.kind), SubjectID: n.subject, CreatedAt: at})
		}
	}
	require.NoError(t, f.store.Notifications.CreateMany(ctx, notices))
	return f
}

// TestListTools_ShowWhatTheRowByRowCheckShowed holds each list tool, now filtered in SQL, to what the path it replaced
// showed each viewer: the use-cases that still check row by row, or the per-row rule the adapter applied.
func TestListTools_ShowWhatTheRowByRowCheckShowed(t *testing.T) {
	f := listParityFixture(t)
	s := f.svc
	ctx := context.Background()
	ticketList := tool(t, composite.TicketTools(s.ticketsSvc, s.workspaceSvc, s.reviewSvc), "ticket_list")
	docList := tool(t, docs.MCPTools(s.docsSvc), "doc_list")
	deployList := tool(t, deploy.MCPTools(s.deploySvc), "deploy_list")
	notificationList := tool(t, workspace.NotificationMCPTools(s.notifSvc), "notification_list")
	viewers := []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider, uClient}

	for _, user := range viewers {
		viewer := as(user)
		t.Run(user, func(t *testing.T) {
			listed, err := s.ticketsSvc.List(viewer)
			require.NoError(t, err)
			assert.Equal(t, ticketIDs(listed), pageThrough(t, s.accessSvc, ticketList, user, ""), "ticket_list")

			hits, err := s.ticketsSvc.Search(viewer, "login", 1000)
			require.NoError(t, err)
			var searched []string
			for _, h := range hits {
				searched = append(searched, h.ID)
			}
			assert.Equal(t, searched, pageThrough(t, s.accessSvc, ticketList, user, `"query":"login",`), "ticket_list query")

			blockers, err := s.ticketsSvc.UnclearedBlockers(viewer)
			require.NoError(t, err)
			var blocked []string
			for _, tk := range listed {
				if len(blockers[tk.ID]) > 0 {
					blocked = append(blocked, tk.ID)
				}
			}
			assert.Equal(t, blocked, pageThrough(t, s.accessSvc, ticketList, user, `"blocked_only":true,`), "ticket_list blocked_only")

			docItems, err := s.docsSvc.List(viewer)
			require.NoError(t, err)
			var docIDs, openable []string
			for _, d := range docItems {
				docIDs = append(docIDs, d.ID)
				if d.CanOpen {
					openable = append(openable, d.ID)
				}
			}
			assert.Equal(t, docIDs, pageThrough(t, s.accessSvc, docList, user, ""), "doc_list")
			_, page := callPage(t, s.accessSvc, docList, user, `{"limit":100}`)
			var openedNow []string
			for _, raw := range page.Items {
				var item docs.DocListItem
				require.NoError(t, json.Unmarshal(raw, &item))
				if item.CanOpen {
					openedNow = append(openedNow, item.ID)
				}
			}
			assert.Equal(t, openable, openedNow, "doc_list can_open")

			docHits, err := s.docsSvc.Search(viewer, "spec", 1000)
			require.NoError(t, err)
			var ranked []string
			for _, h := range docHits {
				if slices.Contains(docIDs, h.ID) {
					ranked = append(ranked, h.ID)
				}
			}
			// Docs that rank alike come in insertion order from Search and by id from the page; the storage test pins order.
			assert.ElementsMatch(t, ranked, pageThrough(t, s.accessSvc, docList, user, `"query":"spec",`), "doc_list query")

			ds, err := s.deploySvc.List(viewer)
			require.NoError(t, err)
			var deployIDs []string
			for _, d := range slices.Backward(ds) {
				deployIDs = append(deployIDs, d.ID)
			}
			assert.Equal(t, deployIDs, pageThrough(t, s.accessSvc, deployList, user, ""), "deploy_list")

			all, _, err := f.store.Notifications.Page(ctx, user, workspace.InboxFilter{}, nil, paging.Window{Limit: 100})
			require.NoError(t, err)
			memberOf, _, err := s.accessSvc.ProjectsAnywhere(ctx, user, permissions.Member)
			require.NoError(t, err)
			var inbox []string
			for _, n := range all {
				member := n.WorkspaceID == "" || slices.Contains(memberOf, n.WorkspaceID)
				opens := n.ProjectID == "" || s.accessSvc.CanInProject(ctx, user, n.ProjectID, permissions.Member)
				if member && opens {
					inbox = append(inbox, n.ID)
				}
			}
			assert.Equal(t, inbox, pageThrough(t, s.accessSvc, notificationList, user, ""), "notification_list")
		})
	}
}

func ticketIDs(ts []*tickets.Ticket) []string {
	var out []string
	for _, tk := range ts {
		out = append(out, tk.ID)
	}
	return out
}
