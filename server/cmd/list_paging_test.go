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
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/runner"
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
	t.Parallel()
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
// hands back about its own rows and a count, where paging in memory read every row or the first thousand; a reader's
// doc_list filters by their access in the same query.
func TestRows_ListToolsReadOnlyTheirPage(t *testing.T) {
	t.Parallel()
	f, st, db := newCountedFixtureDB(t)
	for _, stmt := range []string{
		`INSERT INTO tickets (id, title, body, status, project_id, created_at, updated_at, number, position, due_on)
			SELECT printf('t-%05d', i), 'Login times out ' || i, '', 'open', 'project-general', 1700000000 + i, 1700000000 + i, 100 + i, i, printf('2026-10-%02d', 1 + i % 28) FROM n`,
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
	for _, tc := range []struct{ tool, args, viewer string }{
		{"ticket_list", `{}`, uOwner},
		{"ticket_list", `{"query":"login"}`, uOwner},
		{"ticket_list", `{"due_from":"2026-10-01","due_to":"2026-10-31"}`, uOwner},
		{"doc_list", `{}`, uOwner},
		{"doc_list", `{}`, uReader},
		{"doc_list", `{"query":"rollback"}`, uOwner},
		{"deploy_list", `{}`, uOwner},
		{"notification_list", `{"offset":1500}`, uOwner},
		{"memory_list", `{"project_id":"project-general"}`, uOwner},
	} {
		t.Run(tc.tool+" "+tc.args+" as "+tc.viewer, func(t *testing.T) {
			st.Reset()
			ids, page := callPage(t, s.accessSvc, tool(t, tools, tc.tool), tc.viewer, tc.args)

			assert.Len(t, ids, 50)
			assert.GreaterOrEqual(t, page.Total, 3000)
			assert.Less(t, st.Rows(), int64(250), "rows handed back for one page")
		})
	}
}

// TestRows_DocListReadsNoneOfTheReadersDocRules: a reader's doc rules, thousands in the listed project and thousands
// in another, cost doc_list no statement and no row handed back, since the page decides them in SQL.
func TestRows_DocListReadsNoneOfTheReadersDocRules(t *testing.T) {
	t.Parallel()
	f, st, db := newCountedFixtureDB(t)
	now := time.Now()
	require.NoError(t, f.store.Projects.Create(t.Context(), &workspace.Project{ID: "p-far", Name: "Far", Prefix: "FAR", WorkspaceID: wsDefault, CreatedAt: now, UpdatedAt: now}))
	listRolesIn(t, db, "p-far")
	_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 2000)
		INSERT INTO docs (id, title, body, body_md, version, created_at, updated_at, project_id, folder_id)
		SELECT printf('d-%s-%05d', p.id, i), 'Runbook ' || i, '{"type":"doc","content":[]}', '', 1, 1700000000 + i, 1700000000 + i, p.id,
			(SELECT id FROM doc_folders WHERE project_id = p.id LIMIT 1)
		FROM n, projects p WHERE p.id IN ('project-general', 'p-far')`)
	require.NoError(t, err)
	docList := tool(t, docs.MCPTools(f.svc.docsSvc), "doc_list")
	cost := func(args string) [2]int64 {
		st.Reset()
		ids, _ := callPage(t, f.svc.accessSvc, docList, uReader, args)
		require.Len(t, ids, 20)
		return [2]int64{st.Count(), st.Rows()}
	}
	listed, everywhere := `{"project_id":"project-general","limit":20}`, `{"limit":20}`
	before := [2][2]int64{cost(listed), cost(everywhere)}

	_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at)
		SELECT 'doc', d.id, 'person', ?, NULL, '["docs:read","docs:thread"]', '[]', 1, 1 FROM docs d WHERE d.id LIKE 'd-%'
		UNION ALL SELECT 'doc', d.id, 'role', NULL, 'role-reader', '["docs:read"]', '[]', 1, 1 FROM docs d WHERE d.id LIKE 'd-p-far-%'`, uReader)
	require.NoError(t, err)

	assert.Equal(t, before, [2][2]int64{cost(listed), cost(everywhere)}, "statements and rows, with no rule and with 6,000")
}

// TestRows_MachineListsReadOnlyTheMachinesTheCallerSees: over 3,000 machines with a runner each, a person on five of
// them gets the machine list, the runner list and machine_list back with about those rows, where listing every machine
// and dropping the unseen ones in Go read all 3,000.
func TestRows_MachineListsReadOnlyTheMachinesTheCallerSees(t *testing.T) {
	t.Parallel()
	f, st, db := newCountedFixtureDB(t)
	for _, insert := range []string{
		`INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) SELECT printf('mach-%05d', i), printf('box-%05d', i), '/data/nexul', '', 1, 1 FROM n`,
		`INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id) SELECT printf('r-%05d', i), printf('runner-%05d', i), 1, 0, 1, 'v1', printf('mach-%05d', i) FROM n`,
		`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at)
			SELECT 'machine', printf('mach-%05d', i), 'person', 'u-writer', NULL, '["machines:see"]', '[]', 1, 1 FROM n WHERE i <= 5`,
	} {
		_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 3000) ` + insert)
		require.NoError(t, err)
	}
	svc := runner.NewService(f.store.Runners, fakeUpgradeDispatch{}).WithMachines(f.store.Machines).WithGate(f.svc.accessSvc)
	machineList := tool(t, runner.MCPTools(svc), "machine_list")

	st.Reset()
	ids, page := callPage(t, f.svc.accessSvc, machineList, uWriter, `{}`)
	assert.Len(t, ids, 5)
	assert.Equal(t, 5, page.Total)
	assert.Less(t, st.Rows(), int64(100), "rows handed back for machine_list")

	st.Reset()
	runners, err := svc.ListRunners(as(uWriter))
	require.NoError(t, err)
	assert.Len(t, runners, 5)
	assert.Less(t, st.Rows(), int64(100), "rows handed back for the runner list")
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

// listParityFixture is the permission fixture with uClient listed in p-client only, uUnlisted in no project and uNoRole
// holding no role, a ticket, stack, and deploy in each project, blockers across them, a doc each in a private and a
// shared state, rules for roles on the docs, a folder whose rules reach two of them, rules on the stacks and a branch
// deployment following them, an instance stack and a deleted stack's deploy, and notices in every state an inbox filters.
func listParityFixture(t *testing.T) permFixture {
	t.Helper()
	db := migrated.DB(t)
	svc, store := wiredOver(t, db)
	f := withListParityRows(t, seedPermFixture(t, db, svc, store))
	withMachineM1(t, f, "role-reader")
	withRoleRulesOnDocs(t, db)
	withFolderRules(t, f)
	withStackRules(t, f)
	withMachineRules(t, db)
	return f
}

// withMachineRules gives the fixture five machines, each with a runner, and rules of every shape: a role seeing one, a
// person seeing another though their role is denied it, a person denied one their role sees, a machine with no rule,
// and one a role sees that its holder's own rule hides.
func withMachineRules(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) VALUES
    ('mach-a', 'box-a', '/data/nexul', '', 1, 1), ('mach-b', 'box-b', '/data/nexul', '', 1, 1), ('mach-c', 'box-c', '/data/nexul', '', 1, 1),
    ('mach-d', 'box-d', '/data/nexul', '', 1, 1), ('mach-e', 'box-e', '/data/nexul', '', 1, 1);
INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id)
SELECT 'r-' || substr(id, 6), 'runner-' || substr(id, 6), 1, 0, 1, 'v1', id FROM machines;
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at) VALUES
    ('machine', 'mach-a', 'role', NULL, 'role-reader', '["machines:see"]', '[]', 1, 1),
    ('machine', 'mach-a', 'role', NULL, 'role-plain', '["machines:see"]', '[]', 1, 1),
    ('machine', 'mach-b', 'role', NULL, 'role-writer', '[]', '["machines:see"]', 1, 1),
    ('machine', 'mach-b', 'person', 'u-writer', NULL, '["machines:see"]', '[]', 1, 1),
    ('machine', 'mach-c', 'role', NULL, 'role-plain', '["machines:see"]', '[]', 1, 1),
    ('machine', 'mach-c', 'person', 'u-plain', NULL, '[]', '["machines:see"]', 1, 1),
    ('machine', 'mach-e', 'role', NULL, 'role-reader', '["machines:see"]', '[]', 1, 1),
    ('machine', 'mach-e', 'person', 'u-reader', NULL, '[]', '["machines:see"]', 1, 1)`)
	require.NoError(t, err)
}

// rowByRowMachines is every machine a single See check opens for user, in name order.
func rowByRowMachines(t *testing.T, f permFixture, user string) []string {
	t.Helper()
	ms, err := f.store.Machines.List(context.Background())
	require.NoError(t, err)
	var out []string
	for _, m := range ms {
		if f.svc.accessSvc.RequireMachine(as(user), m.ID, permissions.MachinesSee) == nil {
			out = append(out, m.ID)
		}
	}
	return out
}

// withStackRules gives the fixture's stacks rules of every shape and stack-1 a branch deployment that follows them: a
// role denied one stack's deploys, a person denied its read though their role and project rule allow its deploys, a
// person allowed what their role lacks, and a client denied their project's stack's deploys.
func withStackRules(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-1-feat", ProjectID: pGeneral, Name: "web-feat", Slug: "web-feat", Machine: "m1",
		Strategy: deploy.StrategyRun, DerivedFrom: f.stack, Branch: "feat", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Deploys.Create(ctx, &deploy.Deploy{ID: "deploy-stack-1-feat", StackID: "stack-1-feat", Service: "web-feat", Status: deploy.StatusHealthy,
		CreatedAt: now.Add(10 * time.Second), UpdatedAt: now}))
	require.NoError(t, f.store.Access.SetRole(ctx, "stack", f.stack, "role-reader", nil, grant("deploys:read")))
	require.NoError(t, f.store.Access.SetRole(ctx, "stack", f.stack, "role-plain", grant("deploys:read"), nil))
	require.NoError(t, f.store.Access.Set(ctx, "stack", f.stack, uOverwrite, nil, grant("stacks:read")))
	require.NoError(t, f.store.Access.Set(ctx, "stack", f.stack, uPlain, grant("stacks:read", "deploys:read"), nil))
	require.NoError(t, f.store.Access.Set(ctx, "stack", "stack-client", uClient, nil, grant("deploys:read")))
	require.NoError(t, f.store.Access.SetRole(ctx, "stack", "stack-client", "role-client", grant("stacks:write"), nil))
}

// rowByRowStacks is every base stack a single read check opens for user, in name order.
func rowByRowStacks(t *testing.T, f permFixture, user string) []string {
	t.Helper()
	all, err := f.store.Stacks.ListByProject(context.Background(), "")
	require.NoError(t, err)
	var out []string
	for _, st := range all {
		if _, err := f.svc.deploySvc.GetStack(as(user), st.ID); err == nil && st.DerivedFrom == "" {
			out = append(out, st.ID)
		}
	}
	return out
}

// parityFolder is the folder in project-general holding doc-shared and doc-folder, with rules of every shape.
const parityFolder = "folder-sales"

// withFolderRules puts doc-shared and doc-folder in parityFolder, its rules of every shape, one beaten by doc-folder's own.
func withFolderRules(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Docs.CreateFolder(ctx, &docs.Folder{ID: parityFolder, ProjectID: pGeneral, Name: "Sales", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Docs.Create(ctx, &docs.Doc{ID: "doc-folder", ProjectID: pGeneral, FolderID: parityFolder, Title: "Spec doc-folder",
		Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Docs.SetDocFolder(ctx, "doc-shared", parityFolder))
	read := grant("docs:read")
	require.NoError(t, f.store.Access.SetRole(ctx, "folder", parityFolder, "role-reader", nil, read))
	require.NoError(t, f.store.Access.SetRole(ctx, "folder", parityFolder, "role-plain", read, nil))
	require.NoError(t, f.store.Access.Set(ctx, "folder", parityFolder, uWriter, read, nil))
	require.NoError(t, f.store.Access.Set(ctx, "folder", parityFolder, uOverwrite, nil, read))
	require.NoError(t, f.store.Access.Set(ctx, "folder", parityFolder, uClient, read, nil))
	require.NoError(t, f.store.Access.SetRole(ctx, "doc", "doc-folder", "role-reader", read, nil))
}

// withRoleRulesOnDocs gives the docs rules for roles: one opening a doc to a role outside its project's answer, one
// closing a shared doc to a role, and one a person's own rule beats.
func withRoleRulesOnDocs(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at) VALUES
    ('doc', 'doc-1', 'role', 'role-plain', '["docs:read","docs:thread"]', '[]', 1, 1),
    ('doc', 'doc-client', 'role', 'role-writer', '["docs:read"]', '[]', 1, 1),
    ('doc', 'doc-client', 'role', 'role-client', '[]', '["docs:write"]', 1, 1),
    ('doc', 'doc-shared', 'role', 'role-reader', '[]', '["docs:read"]', 1, 1),
    ('doc', 'doc-private', 'role', 'role-reader', '["docs:read"]', '[]', 1, 1)`)
	require.NoError(t, err)
}

// withListParityRows adds listParityFixture's rows to a permission fixture.
func withListParityRows(t *testing.T, f permFixture) permFixture {
	t.Helper()
	listedClient(t, f)
	unlistedMembers(t, f)
	ctx := context.Background()
	now := time.Now()
	clientTicket, err := f.svc.ticketsSvc.Create(ctx, pClient, "Login page for the portal", "", "", "")
	require.NoError(t, err)
	blocker, err := f.svc.ticketsSvc.Create(ctx, pGeneral, "Login rate limit", "", "", "")
	require.NoError(t, err)
	require.NoError(t, f.store.Tickets.PutLink(ctx, tickets.TicketLink{TicketID: f.ticket.ID, Kind: tickets.LinkBlockedBy, TargetID: clientTicket.ID, CreatedAt: now}))
	require.NoError(t, f.store.Tickets.PutLink(ctx, tickets.TicketLink{TicketID: clientTicket.ID, Kind: tickets.LinkBlockedBy, TargetID: blocker.ID, CreatedAt: now}))
	for id, due := range map[string]string{f.ticket.ID: "2026-10-05", clientTicket.ID: "2026-10-05", blocker.ID: "2026-11-03"} {
		_, err := f.svc.ticketsSvc.SetDueOn(ctx, id, due)
		require.NoError(t, err)
	}

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
	for i, user := range []string{uOwner, uReader, uPlain, uClient, uUnlisted, uNoRole, uOutsider} {
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
	t.Parallel()
	f := listParityFixture(t)
	s := f.svc
	ticketList := tool(t, composite.TicketTools(s.ticketsSvc, s.workspaceSvc, s.reviewSvc), "ticket_list")
	docList := tool(t, docs.MCPTools(s.docsSvc), "doc_list")
	deployList := tool(t, deploy.MCPTools(s.deploySvc), "deploy_list")
	stackList := tool(t, deploy.MCPTools(s.deploySvc), "stack_list")
	notificationList := tool(t, workspace.NotificationMCPTools(s.notifSvc), "notification_list")
	runners := runner.NewService(f.store.Runners, fakeUpgradeDispatch{}).WithMachines(f.store.Machines).WithGate(s.accessSvc)
	machineList := tool(t, runner.MCPTools(runners), "machine_list")

	for _, user := range []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider, uClient, uUnlisted, uNoRole} {
		t.Run(user, func(t *testing.T) {
			listed, searched, blocked, due := rowByRowTickets(t, f, user)
			assert.Equal(t, listed, pageThrough(t, s.accessSvc, ticketList, user, ""), "ticket_list")
			assert.Equal(t, searched, pageThrough(t, s.accessSvc, ticketList, user, `"query":"login",`), "ticket_list query")
			assert.Equal(t, blocked, pageThrough(t, s.accessSvc, ticketList, user, `"blocked_only":true,`), "ticket_list blocked_only")
			assert.Equal(t, due, pageThrough(t, s.accessSvc, ticketList, user, `"due_from":"2026-10-01","due_to":"2026-10-31",`), "ticket_list due range")

			docIDs, openable, ranked := rowByRowDocs(t, f, user)
			assert.Equal(t, openable, docIDs, "listed equals openable")
			assert.Equal(t, docIDs, pageThrough(t, s.accessSvc, docList, user, ""), "doc_list")
			// Docs that rank alike come in insertion order from Search and by id from the page; the storage test pins order.
			assert.ElementsMatch(t, ranked, pageThrough(t, s.accessSvc, docList, user, `"query":"spec",`), "doc_list query")

			ds, err := s.deploySvc.List(as(user))
			require.NoError(t, err)
			var deployIDs []string
			for _, d := range slices.Backward(ds) {
				deployIDs = append(deployIDs, d.ID)
			}
			assert.Equal(t, deployIDs, pageThrough(t, s.accessSvc, deployList, user, ""), "deploy_list")
			assert.Equal(t, rowByRowStacks(t, f, user), pageThrough(t, s.accessSvc, stackList, user, ""), "stack_list")

			assert.Equal(t, rowByRowInbox(t, f, user), pageThrough(t, s.accessSvc, notificationList, user, ""), "notification_list")

			assert.Equal(t, rowByRowMachines(t, f, user), pageThrough(t, s.accessSvc, machineList, user, ""), "machine_list")
			rs, err := runners.ListRunners(as(user))
			require.NoError(t, err)
			var seenRunners []string
			for _, r := range rs {
				seenRunners = append(seenRunners, "r-"+r.Name[len("runner-"):])
			}
			wantRunners := make([]string, 0)
			for _, id := range rowByRowMachines(t, f, user) {
				wantRunners = append(wantRunners, "r-"+id[len("mach-"):])
			}
			assert.ElementsMatch(t, wantRunners, seenRunners, "the runners on the machines seen")
		})
	}
}

// rowByRowTickets is what the tickets use-cases that check each row show user: every ticket, a search, the blocked, and
// those due in October by due date then id.
func rowByRowTickets(t *testing.T, f permFixture, user string) (listed, searched, blocked, due []string) {
	t.Helper()
	ts, err := f.svc.ticketsSvc.List(as(user))
	require.NoError(t, err)
	hits, err := f.svc.ticketsSvc.Search(as(user), "login", 1000)
	require.NoError(t, err)
	for _, h := range hits {
		searched = append(searched, h.ID)
	}
	blockers, err := f.svc.ticketsSvc.UnclearedBlockers(as(user))
	require.NoError(t, err)
	for _, tk := range ts {
		if len(blockers[tk.ID]) > 0 {
			blocked = append(blocked, tk.ID)
		}
	}
	inOctober := slices.DeleteFunc(slices.Clone(ts), func(tk *tickets.Ticket) bool { return tk.DueOn < "2026-10-01" || tk.DueOn > "2026-10-31" })
	slices.SortFunc(inOctober, func(a, b *tickets.Ticket) int { return strings.Compare(a.DueOn+" "+a.ID, b.DueOn+" "+b.ID) })
	return ticketIDs(ts), searched, blocked, ticketIDs(inOctober)
}

// rowByRowDocs is what the docs use-cases that check each row show user: every doc listed, every doc a single read
// check opens, and a search.
func rowByRowDocs(t *testing.T, f permFixture, user string) (listed, openable, searched []string) {
	t.Helper()
	items, err := f.svc.docsSvc.List(as(user))
	require.NoError(t, err)
	for _, d := range items {
		listed = append(listed, d.ID)
	}
	all, err := f.store.Docs.List(context.Background())
	require.NoError(t, err)
	for _, d := range all {
		if can, err := f.svc.accessSvc.Can(context.Background(), user, d.ID, permissions.DocsRead); err == nil && can {
			openable = append(openable, d.ID)
		}
	}
	hits, err := f.svc.docsSvc.Search(as(user), "spec", 1000)
	require.NoError(t, err)
	for _, h := range hits {
		if slices.Contains(listed, h.ID) {
			searched = append(searched, h.ID)
		}
	}
	return listed, openable, searched
}

// rowByRowInbox is user's inbox as the adapter filtered it: a notice from a workspace they belong to, about a project
// they may open.
func rowByRowInbox(t *testing.T, f permFixture, user string) []string {
	t.Helper()
	ctx := context.Background()
	all, _, err := f.store.Notifications.Page(ctx, user, workspace.InboxFilter{}, nil, paging.Window{Limit: 100})
	require.NoError(t, err)
	memberOf, _, err := f.svc.accessSvc.ProjectsAnywhere(ctx, user, permissions.Member)
	require.NoError(t, err)
	var inbox []string
	for _, n := range all {
		member := n.WorkspaceID == "" || slices.Contains(memberOf, n.WorkspaceID)
		if member && (n.ProjectID == "" || f.svc.accessSvc.CanInProject(ctx, user, n.ProjectID, permissions.Member)) {
			inbox = append(inbox, n.ID)
		}
	}
	return inbox
}

func ticketIDs(ts []*tickets.Ticket) []string {
	var out []string
	for _, tk := range ts {
		out = append(out, tk.ID)
	}
	return out
}
