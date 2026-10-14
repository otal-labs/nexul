package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/mcp/composite"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

const (
	memoPerCall = "a memo per call"
	memoShared  = "one memo for the whole test"
)

// testMemo is what as() puts on a context besides the actor: nothing, a fresh memo per call as each request gets,
// or one memo a whole test shares, which every write the test makes has to clear.
var testMemo struct {
	mode   string
	access *access.Service
	shared *access.Memo
}

func withTestMemo(ctx context.Context) context.Context {
	if testMemo.access == nil {
		return ctx
	}
	switch testMemo.mode {
	case memoPerCall:
		return access.WithMemo(ctx, testMemo.access.NewMemo())
	case memoShared:
		return access.WithMemo(ctx, testMemo.shared)
	}
	return ctx
}

// useTestMemoOf points as() at the access service a fixture has just wired.
func useTestMemoOf(t *testing.T, a *access.Service) {
	testMemo.access, testMemo.shared = a, a.NewMemo()
	t.Cleanup(func() { testMemo.access, testMemo.shared = nil, nil })
}

// TestIntegration_PermissionSuitesAnswerTheSameWithAMemo runs the permission suites again with memoised access: a
// memo per call, as every request has, and one memo across a whole test, which only stays right if every grant,
// role, membership, and project rule change clears it (ADRs 0087, 0088, 0098, 0099, 0135, 0148).
func TestIntegration_PermissionSuitesAnswerTheSameWithAMemo(t *testing.T) {
	suites := map[string]func(*testing.T){
		"permission table":                  TestIntegration_PermissionTable,
		"grants never exceed the giver":     TestIntegration_GrantsNeverExceedTheGiver,
		"first user holds instance bits":    TestIntegration_FirstUserReachesEveryFormerAdminCapability,
		"not in a project":                  TestIntegration_NotInAProject,
		"not in: notices and trails":        TestIntegration_NotInAProject_NoticesAndTrailsFollowAccess,
		"not in: live frames":               TestIntegration_NotInAProject_LiveFramesFollowAccess,
		"project rules and cleanup":         TestIntegration_ProjectRules_GrantingAndCleanup,
		"private channel":                   TestIntegration_PrivateChannel,
		"channel rules":                     TestIntegration_ChannelRules,
		"the access grid":                   TestIntegration_AccessGrid,
		"switching private":                 TestIntegration_SwitchingPrivateReachesEveryOpenSidebar,
		"removal from a private voice call": TestIntegration_RemovalFromPrivateVoiceChannelEndsTheirCall,
		"live frames follow the read":       TestLiveAudience_FramesFollowTheEntitysRead,
		"deleted memory frames":             TestLiveAudience_MemoryDeletedStaysInItsProject,
		"play frames":                       TestLiveAudience_PlayFramesReachWhoSeesThePlay,
		"instance and deleted ticket":       TestLiveAudience_InstanceAndDeletedTicketFrames,
		"bots follow their conversation":    TestIntegration_BotsFollowTheirConversationsGate,
		"watching takes reading the doc":    TestDocWatchers_WatchingTakesReadingTheDoc,
		"doc sources follow access":         TestIntegration_DocSourceFollowsTheReadersAccess,
		"project targets":                   TestProjectTargets_ListsWhatTheCallerReads,
		"templates bit":                     TestIntegration_TemplatesBitGatesTheInstance,
	}
	for _, mode := range []string{memoPerCall, memoShared} {
		for name, suite := range suites {
			t.Run(mode+"/"+name, func(t *testing.T) {
				testMemo.mode = mode
				t.Cleanup(func() { testMemo.mode = "" })
				suite(t)
			})
		}
	}
}

// TestIntegration_LiveAudienceMemo_FollowsEveryChange holds the live hub's audience, whose memo lasts across frames,
// to the next frame after each kind of change: a role's permissions, a membership, a workspace-wide overwrite, a
// project's rule listing someone, and a doc's own overwrite.
func TestIntegration_LiveAudienceMemo_FollowsEveryChange(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)
	s := f.svc
	ctx := context.Background()
	audience := memoizedAudience(s)
	ticketFrame, err := json.Marshal(tickets.UpdatedEvent{Ticket: *f.ticket})
	require.NoError(t, err)
	docFrame, err := json.Marshal(docs.UpdatedEvent{Doc: docs.Doc{ID: f.doc, ProjectID: pGeneral}})
	require.NoError(t, err)
	reaches := func(user string, frame []byte) bool {
		topic := tickets.TopicUpdated
		if string(frame) == string(docFrame) {
			topic = docs.TopicUpdated
		}
		return audience(identity.WithActor(ctx, identity.Actor{ID: user}), topic, json.RawMessage(frame))
	}

	assert.True(t, reaches(uReader, ticketFrame))
	_, err = s.rolesSvc.Update(as(uOwner), wsDefault, "role-reader", uOwner, "Reader", grant("docs:read"), nil)
	require.NoError(t, err)
	assert.False(t, reaches(uReader, ticketFrame), "a role losing tickets:read")
	_, err = s.rolesSvc.Update(as(uOwner), wsDefault, "role-reader", uOwner, "Reader", grant("tickets:read", "docs:read"), nil)
	require.NoError(t, err)
	assert.True(t, reaches(uReader, ticketFrame), "the role given it back")
	require.NoError(t, s.tenancySvc.RemoveMember(as(uOwner), uOwner, wsDefault, uReader))
	assert.False(t, reaches(uReader, ticketFrame), "removed from the workspace")

	assert.False(t, reaches(uPlain, ticketFrame))
	require.NoError(t, s.tenancySvc.SetMemberRoles(as(uOwner), uOwner, wsDefault, uPlain, "role-writer"))
	assert.True(t, reaches(uPlain, ticketFrame), "given a role that reads tickets")
	require.NoError(t, s.tenancySvc.SetMemberRoles(as(uOwner), uOwner, wsDefault, uPlain, "role-plain"))
	assert.False(t, reaches(uPlain, ticketFrame), "the role taken back")

	assert.False(t, reaches(uClient, ticketFrame))
	listClientInGeneral(t, f, true)
	assert.True(t, reaches(uClient, ticketFrame), "listed in the project")
	listClientInGeneral(t, f, false)
	assert.False(t, reaches(uClient, ticketFrame), "taken back out")

	assert.False(t, reaches(uWriter, docFrame))
	require.NoError(t, s.accessSvc.SetGrants(as(uOwner), uOwner, []string{f.doc}, []string{uWriter}, []permissions.Action{permissions.DocsRead}, access.StateAllow))
	assert.True(t, reaches(uWriter, docFrame), "the doc shared")
	require.NoError(t, s.accessSvc.SetGrants(as(uOwner), uOwner, []string{f.doc}, []string{uWriter}, []permissions.Action{permissions.DocsRead}, access.StateFallback))
	assert.False(t, reaches(uWriter, docFrame), "the doc unshared")
}

// TestAccessMemo_ConcurrentRequestsShareNothing runs many people's requests at once, each on its own memo, beside the
// live audience's shared one, and every answer must be the one that person gets alone; -race watches the memos.
func TestAccessMemo_ConcurrentRequestsShareNothing(t *testing.T) {
	f := newPermFixture(t)
	listedClient(t, f)
	s := f.svc
	frame, err := json.Marshal(tickets.UpdatedEvent{Ticket: *f.ticket})
	require.NoError(t, err)
	audience := memoizedAudience(s)
	answers := func(ctx context.Context, user string) string {
		project := s.accessSvc.RequireProject(ctx, pGeneral, permissions.TicketsRead)
		client := s.accessSvc.RequireProject(ctx, pClient, permissions.TicketsRead)
		_, doc := s.docsSvc.Get(ctx, f.doc)
		cs, listErr := s.chatSvc.ListConversations(ctx, wsDefault, user)
		live := audience(identity.WithActor(context.Background(), identity.Actor{ID: user}), tickets.TopicUpdated, json.RawMessage(frame))
		return fmt.Sprint(outcome(project), outcome(client), outcome(doc), len(cs), outcome(listErr), live)
	}
	users := []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider, uClient}
	want := map[string]string{}
	for _, user := range users {
		want[user] = answers(identity.WithActor(context.Background(), identity.Actor{ID: user}), user)
	}

	var wg sync.WaitGroup
	got := make([][]string, 16)
	for g := range got {
		wg.Go(func() {
			for i := range 20 {
				user := users[(g+i)%len(users)]
				ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), s.accessSvc.NewMemo())
				got[g] = append(got[g], user+" "+answers(ctx, user))
			}
		})
	}
	wg.Wait()
	for _, answered := range got {
		for _, line := range answered {
			var user string
			_, err := fmt.Sscan(line, &user)
			require.NoError(t, err)
			assert.Equal(t, user+" "+want[user], line)
		}
	}
}

// newCountedFixture is the permission fixture over a database that counts every statement it runs.
func newCountedFixture(t *testing.T) (permFixture, *testutil.Statements) {
	t.Helper()
	f, st, _ := newCountedFixtureDB(t)
	return f, st
}

// newCountedFixtureDB is newCountedFixture with the database itself, for a test that bulk-loads rows in SQL.
func newCountedFixtureDB(t *testing.T) (permFixture, *testutil.Statements, *sql.DB) {
	t.Helper()
	db, st := countedDB(t)
	svc, store := wiredOver(t, db)
	return seedPermFixture(t, db, svc, store), st, db
}

// countedDB is a migrated database with the general project, counting every statement run on it.
func countedDB(t *testing.T) (*sql.DB, *testutil.Statements) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "counted.db")
	plain, err := storage.OpenDB(path)
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(plain))
	require.NoError(t, testutil.SeedGeneralProject(plain))
	require.NoError(t, plain.Close())
	// The pragmas storage.OpenDB opens with.
	inner, err := sqlite.NewConnector(path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_txlock=immediate")
	require.NoError(t, err)
	db, st := testutil.OpenCounted(inner)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, st
}

// grow adds n of every listed thing to the fixture's project: docs shared with the reader, tickets with threads the
// reader is in, doc threads, and public and private channels; and n projects every role is listed in, and n readers.
func grow(t *testing.T, f permFixture, n int) []string {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	ticketIDs := []string{f.ticket.ID}
	for i := range n {
		projectID := fmt.Sprintf("project-%d-%d", n, i)
		require.NoError(t, f.store.Projects.Create(ctx, &workspace.Project{ID: projectID, Name: "Mobile", Prefix: fmt.Sprintf("M%d%d", n, i), WorkspaceID: wsDefault, CreatedAt: now, UpdatedAt: now}))
		listRolesIn(t, f.db, projectID)
		member := fmt.Sprintf("u-reader-%d-%d", n, i)
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: member, Provider: auth.ProviderGitHub, ProviderUserID: member, Login: member})
		require.NoError(t, err)
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: member, WorkspaceID: wsDefault, RoleID: "role-reader", CreatedAt: now}))
		docID := fmt.Sprintf("doc-%d-%d", n, i)
		require.NoError(t, f.store.Docs.Create(ctx, &docs.Doc{ID: docID, ProjectID: pGeneral, Title: "Spec", Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, f.store.Access.Set(ctx, "doc", docID, uReader, grant("docs:read", "docs:thread"), nil))
		_, err = f.svc.chatSvc.GetOrCreateDocThread(ctx, wsDefault, docID, uOwner)
		require.NoError(t, err)
		ticket, err := f.svc.ticketsSvc.Create(ctx, pGeneral, fmt.Sprintf("Login times out %d", i), "", "", "")
		require.NoError(t, err)
		ticketIDs = append(ticketIDs, ticket.ID)
		_, err = f.svc.chatSvc.GetOrCreateTicketThread(ctx, wsDefault, ticket.ID, uReader)
		require.NoError(t, err)
		_, err = f.svc.chatSvc.CreateChannel(ctx, wsDefault, uOwner, fmt.Sprintf("room-%d-%d", n, i))
		require.NoError(t, err)
		_, err = f.svc.chatSvc.CreatePrivateChannel(as(uWriter), wsDefault, uWriter, fmt.Sprintf("quiet-%d-%d", n, i), chat.KindChannel, []string{uReader})
		require.NoError(t, err)
	}
	return ticketIDs
}

// guardPlay is the play queueRuns queues a run of on every ticket, for the play's queued-runs path.
const guardPlay = "play-guard"

// queueRuns queues a run of guardPlay on each ticket and lets the reader read auto plays, so the path has rows to filter.
func queueRuns(t *testing.T, f permFixture, ticketIDs []string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	stage := plays.StageProgress
	require.NoError(t, f.store.Plays.Create(ctx, &plays.Play{ID: guardPlay, WorkspaceID: wsDefault, Label: "Fix it", Type: plays.TypeTicket,
		Instructions: "Do it.", Enabled: true, ShowWhenStage: &stage, ExcludedProjectIDs: []string{}, CreatedAt: now, UpdatedAt: now}))
	for _, ticketID := range ticketIDs {
		_, err := f.store.PlayQueue.EnqueueRun(ctx, &plays.QueueItem{ID: ids.New(), WorkspaceID: wsDefault, ProjectID: pGeneral,
			TargetType: plays.TargetTicket, TargetID: ticketID, PlayID: guardPlay, PlayLabel: "Fix it", AutoPlayID: "ap-guard",
			PersonID: uWriter, RunOn: plays.RunOnDeveloper, Moment: plays.MomentTicketUnblocked, Priority: plays.LevelNormal,
			Status: plays.QueueQueued, Via: plays.ViaWeb, QueuedAt: now, NotBefore: now})
		require.NoError(t, err)
	}
	reader, err := f.store.Roles.Get(ctx, "role-reader")
	require.NoError(t, err)
	reader.Permissions = reader.Permissions.With(permissions.AutoplaysRead).With(permissions.PlaysRead).With(permissions.PlaysRun)
	require.NoError(t, f.store.Roles.Update(ctx, reader))
}

// addPlays adds n doc plays, each with a run on the fixture's ticket started by the writer, so the reader reads each run
// only by seeing its play.
func addPlays(t *testing.T, f permFixture, n int) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	for i := range n {
		id := fmt.Sprintf("play-%d-%d", n, i)
		require.NoError(t, f.store.Plays.Create(ctx, &plays.Play{ID: id, WorkspaceID: wsDefault, Label: "Notes " + id, Type: plays.TypeDoc,
			Instructions: "Write them.", Enabled: true, ExcludedProjectIDs: []string{}, CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, f.store.PlayTrails.CreateTrail(ctx, &plays.Trail{ID: "tr-" + id, WorkspaceID: wsDefault, PlayID: id, PlayLabel: "Notes",
			TargetType: plays.TargetTicket, TargetID: f.ticket.ID, ProjectID: pGeneral, StarterID: uWriter, Via: plays.ViaWeb, State: plays.TrailDone,
			StartedAt: now, Activity: []plays.ActivityEntry{}}))
	}
}

// TestStatements_PlayRulesCostTheSameAtAnyLength: rules for every role and the reader on every play cost the plays list,
// the run menus, play_list, a play's queued runs and a ticket's trails no statement more than none do, at a few plays or
// many, since the play rules reaching a person come in one read.
func TestStatements_PlayRulesCostTheSameAtAnyLength(t *testing.T) {
	runnerOf := func(f permFixture) *plays.Runner {
		return plays.NewRunner(plays.RunnerConfig{Plays: f.store.Plays, Trails: f.store.PlayTrails, Queue: f.store.PlayQueue, Perm: f.svc.accessSvc})
	}
	paths := map[string]func(ctx context.Context, f permFixture) error{
		"plays list": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.playsSvc.List(ctx, wsDefault)
			return err
		},
		"run menus": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.playsSvc.ListApplicable(ctx, wsDefault, uReader, pGeneral, plays.TypeDoc, nil)
			return err
		},
		"play_list": func(ctx context.Context, f permFixture) error {
			_, err := tool(t, plays.MCPTools(f.svc.playsSvc), "play_list").Call(ctx, json.RawMessage(`{"workspace_id":"`+wsDefault+`"}`))
			return err
		},
		"a play's queued runs": func(ctx context.Context, f permFixture) error {
			_, err := runnerOf(f).QueuedForPlay(ctx, guardPlay)
			return err
		},
		"a ticket's trails": func(ctx context.Context, f permFixture) error {
			_, err := runnerOf(f).ListTrails(ctx, plays.TargetTicket, f.ticket.ID, "")
			return err
		},
	}
	cost := func(n int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		queueRuns(t, f, grow(t, f, 1))
		addPlays(t, f, n)
		if rules {
			_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at)
SELECT 'play', p.id, 'role', r.id, '["plays:read","plays:run"]', '["plays:delete"]', 1, 1 FROM plays p, roles r
WHERE r.id IN ('role-reader', 'role-writer')`)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
SELECT 'play', p.id, 'person', ?, '["plays:run"]', '["plays:write"]', 1, 1 FROM plays p`, uReader)
			require.NoError(t, err)
		}
		out := map[string]int64{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f), name)
			out[name] = st.Count()
		}
		t.Logf("statements with %d plays, rules %v: %v", n, rules, out)
		return out
	}
	none := cost(12, false)
	assert.Equal(t, none, cost(2, true), "a few plays, each with rules")
	assert.Equal(t, none, cost(12, true), "many plays, each with rules")
}

// TestStatements_ListsCostTheSameAtAnyLength pins each list path's statements as independent of how many rows it
// checks: the same count with a few rows as with many, where checking each row used to read the access layers again.
func TestStatements_ListsCostTheSameAtAnyLength(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture, ticketIDs []string) error{
		"docs list": func(ctx context.Context, f permFixture, _ []string) error {
			_, err := f.svc.docsSvc.ListByProject(ctx, pGeneral)
			return err
		},
		"chat conversation list": func(ctx context.Context, f permFixture, _ []string) error {
			_, err := f.svc.chatSvc.ListConversations(ctx, wsDefault, uReader)
			return err
		},
		"chat unread counts": func(ctx context.Context, f permFixture, _ []string) error {
			_, err := f.svc.chatSvc.UnreadCounts(ctx, wsDefault, uReader)
			return err
		},
		"board thread markers": func(ctx context.Context, f permFixture, ticketIDs []string) error {
			_, err := f.svc.chatSvc.HasTicketThreads(ctx, ticketIDs)
			return err
		},
		"project list": func(ctx context.Context, f permFixture, ticketIDs []string) error {
			projects, err := f.svc.workspaceSvc.List(ctx, wsDefault)
			if err == nil && len(projects) != len(ticketIDs) {
				err = fmt.Errorf("listed %d projects, want %d", len(projects), len(ticketIDs))
			}
			return err
		},
		"project people": func(ctx context.Context, f permFixture, ticketIDs []string) error {
			listed, err := f.svc.tenancySvc.ListProjectPeople(ctx, uReader, pGeneral)
			// Every member but the outsider, and the readers grow added.
			if want := len(people) - 1 + len(ticketIDs) - 1; err == nil && len(listed) != want {
				err = fmt.Errorf("listed %d people, want %d", len(listed), want)
			}
			return err
		},
		"a play's queued runs": func(ctx context.Context, f permFixture, ticketIDs []string) error {
			runner := plays.NewRunner(plays.RunnerConfig{Plays: f.store.Plays, Trails: f.store.PlayTrails, Queue: f.store.PlayQueue, Perm: f.svc.accessSvc})
			items, err := runner.QueuedForPlay(ctx, guardPlay)
			if err == nil && len(items) != len(ticketIDs) {
				err = fmt.Errorf("listed %d queued runs, want %d", len(items), len(ticketIDs))
			}
			return err
		},
	}
	cost := func(n int) map[string]int64 {
		f, st := newCountedFixture(t)
		ticketIDs := grow(t, f, n)
		queueRuns(t, f, ticketIDs)
		out := map[string]int64{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f, ticketIDs), name)
			out[name] = st.Count()
		}
		return out
	}
	few, many := cost(2), cost(12)
	assert.Equal(t, few, many)
}

// TestStatements_TicketSearchFetchesOnlyEachResult: a search costs its results' own fetches and nothing per row
// for access, whose layers are read once for the lot.
func TestStatements_TicketSearchFetchesOnlyEachResult(t *testing.T) {
	cost := func(n int) (search, perTicket int64) {
		f, st := newCountedFixture(t)
		grow(t, f, n)
		ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
		st.Reset()
		results, err := f.svc.ticketsSvc.Search(ctx, "login", 0)
		require.NoError(t, err)
		require.Len(t, results, n+1)
		search = st.Count()
		st.Reset()
		_, err = f.store.Tickets.GetByID(context.Background(), f.ticket.ID)
		require.NoError(t, err)
		return search, st.Count()
	}
	few, perTicket := cost(2)
	many, _ := cost(12)
	assert.Equal(t, 10*perTicket, many-few)
}

// TestStatements_LiveFramesCostOneReadPerPersonOnceWarm: after the first frame, an agent stream's frame costs each
// connected person the conversation's own read and no access read, until something commits.
func TestStatements_LiveFramesCostOneReadPerPersonOnceWarm(t *testing.T) {
	f, st, db := newCountedFixtureDB(t)
	withRulesOnEveryChannel(t, db)
	audience := memoizedAudience(f.svc)
	frame, err := json.Marshal(agent.StreamFrame{ConversationID: f.channel.ID, Streaming: true, Text: "tok"})
	require.NoError(t, err)
	members := []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uSteward, uClerk, uManager, uCloner, uEditor, uCopier}
	publish := func() {
		for _, user := range members {
			assert.True(t, audience(identity.WithActor(context.Background(), identity.Actor{ID: user}), agent.TopicAgentStream, json.RawMessage(frame)), user)
		}
	}
	publish()
	st.Reset()
	_, err = f.store.Chat.GetConversation(context.Background(), f.channel.ID)
	require.NoError(t, err)
	perConversation := st.Count()

	st.Reset()
	for range 5 {
		publish()
	}

	assert.Equal(t, 5*int64(len(members))*perConversation, st.Count())
}

// withRulesOnEveryChannel writes Allow rules on every channel in SQL, since no writer names a channel yet.
func withRulesOnEveryChannel(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, user_id, allow, deny, created_at, updated_at)
SELECT 'channel', c.id, 'role', r.id, NULL, '["channels:read"]', '[]', 1, 1 FROM conversations c, roles r
WHERE c.kind IN ('channel', 'voice_channel') AND r.id IN ('role-reader', 'role-writer')
UNION ALL SELECT 'channel', c.id, 'person', NULL, 'u-reader', '["channels:read"]', '[]', 1, 1 FROM conversations c WHERE c.kind IN ('channel', 'voice_channel')`)
	require.NoError(t, err)
}

// TestStatements_ChannelRulesCostTheSameAtAnyLength: rules on every channel cost a chat list no statement at any length.
func TestStatements_ChannelRulesCostTheSameAtAnyLength(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture) error{
		"chat conversation list": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.chatSvc.ListConversations(ctx, wsDefault, uReader)
			return err
		},
		"chat unread counts": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.chatSvc.UnreadCounts(ctx, wsDefault, uReader)
			return err
		},
	}
	cost := func(n int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		grow(t, f, n)
		if rules {
			withRulesOnEveryChannel(t, db)
		}
		out := map[string]int64{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f), name)
			out[name] = st.Count()
		}
		return out
	}
	none := cost(12, false)
	assert.Equal(t, none, cost(2, true), "a few channels, each with rules")
	assert.Equal(t, none, cost(12, true), "many channels, each with rules")
}

// TestStatements_AccessGridCostsTheSameAtAnyLength: the Access page's read and its Finish, which takes every role off
// every project and hides every public channel from a role, cost the same statements at 3 projects as at 30.
func TestStatements_AccessGridCostsTheSameAtAnyLength(t *testing.T) {
	cost := func(n int) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		grow(t, f, n)
		_, err := db.Exec(`UPDATE workspaces SET access_setup_finished = 0`)
		require.NoError(t, err)
		ctx := func() context.Context {
			return access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uOwner}), f.svc.accessSvc.NewMemo())
		}
		st.Reset()
		grid, err := f.svc.accessSvc.Grid(ctx(), uOwner, wsDefault)
		require.NoError(t, err)
		read := st.Count()
		save := access.GridSave{Finish: true}
		for _, p := range grid.Projects {
			save.Projects = append(save.Projects, access.GridProjectSave{ProjectID: p.ID})
		}
		for _, c := range grid.Channels {
			save.Channels = append(save.Channels, access.GridChannelSave{ChannelID: c.ID, HiddenRoleIDs: []string{"role-reader"}})
		}
		require.Greater(t, len(save.Projects), n)
		st.Reset()
		require.NoError(t, f.svc.accessSvc.SaveGrid(ctx(), uOwner, wsDefault, save))
		return map[string]int64{"read": read, "finish": st.Count()}
	}
	assert.Equal(t, cost(3), cost(30))
}

// TestStatements_RoleRulesCostTheSameAtAnyLength: rules for roles on every doc cost a doc list, a doc_list page and a
// doc's read no statement more than no rules at all, at a few docs or many, since they come back with the person's own.
func TestStatements_RoleRulesCostTheSameAtAnyLength(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture) error{
		"docs list": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.docsSvc.ListByProject(ctx, pGeneral)
			return err
		},
		"doc_list": func(ctx context.Context, f permFixture) error {
			_, err := tool(t, docs.MCPTools(f.svc.docsSvc), "doc_list").Call(ctx, json.RawMessage(`{}`))
			return err
		},
		"a doc's read": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.docsSvc.Get(ctx, f.doc)
			return err
		},
	}
	cost := func(n int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		grow(t, f, n)
		if rules {
			_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at)
SELECT 'doc', d.id, 'role', r.id, '["docs:write"]', '["docs:delete"]', 1, 1 FROM docs d, roles r WHERE r.id IN ('role-reader', 'role-writer')`)
			require.NoError(t, err)
		}
		out := map[string]int64{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f), name)
			out[name] = st.Count()
		}
		return out
	}
	none := cost(12, false)
	assert.Equal(t, none, cost(2, true), "a few docs, each with rules")
	assert.Equal(t, none, cost(12, true), "many docs, each with rules")
}

// TestStatements_FolderRulesCostTheSameAtAnyLength: rules on every doc's folder cost the doc paths no statement at any length.
func TestStatements_FolderRulesCostTheSameAtAnyLength(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture) error{
		"docs list": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.docsSvc.ListByProject(ctx, pGeneral)
			return err
		},
		"doc_list": func(ctx context.Context, f permFixture) error {
			_, err := tool(t, docs.MCPTools(f.svc.docsSvc), "doc_list").Call(ctx, json.RawMessage(`{}`))
			return err
		},
		"a doc's read": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.docsSvc.Get(ctx, f.doc)
			return err
		},
		"folder list": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.docsSvc.ListFolders(ctx, pGeneral)
			return err
		},
	}
	cost := func(n int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		grow(t, f, n)
		if rules {
			_, err := db.Exec(`INSERT INTO doc_folders (id, project_id, name, is_default, created_at, updated_at)
SELECT 'f-' || id, project_id, 'Folder ' || id, 0, 1, 1 FROM docs`)
			require.NoError(t, err)
			_, err = db.Exec(`UPDATE docs SET folder_id = 'f-' || id`)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, user_id, allow, deny, created_at, updated_at)
SELECT 'folder', f.id, 'role', r.id, NULL, '["docs:write"]', '["docs:delete"]', 1, 1 FROM doc_folders f, roles r
WHERE f.is_default = 0 AND r.id IN ('role-reader', 'role-writer')
UNION ALL SELECT 'folder', f.id, 'person', NULL, ?, '["docs:read","docs:thread"]', '[]', 1, 1 FROM doc_folders f WHERE f.is_default = 0`, uReader)
			require.NoError(t, err)
		}
		out := map[string]int64{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f), name)
			out[name] = st.Count()
		}
		return out
	}
	none := cost(12, false)
	assert.Equal(t, none, cost(2, true), "a few docs, each in a folder with rules")
	assert.Equal(t, none, cost(12, true), "many docs, each in a folder with rules")
}

// TestStatements_StackRulesCostTheSameAtAnyLength: rules for roles and a person on every stack cost the stack and deploy
// lists, the topology canvas's stacks, a stack's and a deploy's read and a stack's frame no statement more than none, at
// a few stacks or many: a list reads the stacks with rules in one statement, a single check the one stack's.
func TestStatements_StackRulesCostTheSameAtAnyLength(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture) error{
		"stack_list": func(ctx context.Context, f permFixture) error {
			_, err := tool(t, deploy.MCPTools(f.svc.deploySvc), "stack_list").Call(ctx, json.RawMessage(`{}`))
			return err
		},
		"deploy_list": func(ctx context.Context, f permFixture) error {
			_, err := tool(t, deploy.MCPTools(f.svc.deploySvc), "deploy_list").Call(ctx, json.RawMessage(`{}`))
			return err
		},
		"the topology canvas's stacks": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.deploySvc.ListScopedStacks(ctx, "", wsDefault)
			return err
		},
		"a stack's read": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.deploySvc.GetStack(ctx, f.stack)
			return err
		},
		"a deploy's read": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.deploySvc.Get(ctx, f.deploy)
			return err
		},
		"a stack's frame": func(ctx context.Context, f permFixture) error {
			a := liveAudience{access: f.svc.accessSvc, deploy: f.svc.deploySvc}
			if !a.allows(ctx, deploy.TopicStackUpdated, deploy.StackEvent{Stack: deploy.Stack{ID: f.stack, ProjectID: pGeneral}}) {
				return errors.New("the reader's frame was dropped")
			}
			return nil
		},
	}
	cost := func(n int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
INSERT INTO stacks (id, project_id, name, slug, machine, strategy, env, created_at, updated_at)
SELECT 'stack-g-' || i, ?, 'web ' || i, 'web-' || i, 'm1', 'run', '{"TOKEN":"secret"}', 1, 1 FROM n`, n, pGeneral)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO deploys (id, stack_id, service, target, image, status, strategy, created_at, updated_at)
SELECT 'deploy-' || id, id, name, 'm1', 'web:1', 'healthy', 'run', 1, 1 FROM stacks WHERE id LIKE 'stack-g-%'`)
		require.NoError(t, err)
		if rules {
			_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, user_id, allow, deny, created_at, updated_at)
SELECT 'stack', s.id, 'role', r.id, NULL, '["deploys:write"]', '["stacks:delete"]', 1, 1 FROM stacks s, roles r
WHERE s.project_id IS NOT NULL AND r.id IN ('role-reader', 'role-writer')
UNION ALL SELECT 'stack', s.id, 'person', NULL, ?, '["stacks:read","deploys:read"]', '[]', 1, 1 FROM stacks s WHERE s.project_id IS NOT NULL`, uReader)
			require.NoError(t, err)
		}
		out := map[string]int64{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f), name)
			out[name] = st.Count()
		}
		return out
	}
	none := cost(12, false)
	// Each list costs one read past what it cost before stack rules, the reader's stack rules: the topology's no
	// membership read and no instance check, which its workspace's stacks never need. The two lists that reach the
	// instance's stacks add the machines the reader sees, one read of rules and of roles however many machines exist. The
	// stacks hold env, so the two stack lists read one more statement for the stacks the reader may change, whose values they keep.
	assert.Equal(t, map[string]int64{"stack_list": 10, "deploy_list": 11, "the topology canvas's stacks": 7, "a stack's read": 6,
		"a deploy's read": 7, "a stack's frame": 5}, none, "with no stack rules")
	assert.Equal(t, none, cost(2, true), "a few stacks, each with rules")
	assert.Equal(t, none, cost(12, true), "many stacks, each with rules")
}

// TestStatements_ProjectRulesCostTheSameAtAnyLength: rules for every role and every member on every project cost the
// project list, its people, ticket_list, doc_list, a ticket's read and a live frame no statement more than the fixture's
// listing alone, at a few projects or many, since every member's project rules come in one read with the workspace's.
func TestStatements_ProjectRulesCostTheSameAtAnyLength(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture) error{
		"project list": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.workspaceSvc.List(ctx, wsDefault)
			return err
		},
		"project people": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.tenancySvc.ListProjectPeople(ctx, uReader, pGeneral)
			return err
		},
		"/me": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.tenancySvc.MyAccess(ctx, wsDefault, uReader)
			return err
		},
		"ticket_list": func(ctx context.Context, f permFixture) error {
			_, err := tool(t, composite.TicketTools(f.svc.ticketsSvc, f.svc.workspaceSvc, f.svc.reviewSvc), "ticket_list").Call(ctx, json.RawMessage(`{}`))
			return err
		},
		"doc_list": func(ctx context.Context, f permFixture) error {
			_, err := tool(t, docs.MCPTools(f.svc.docsSvc), "doc_list").Call(ctx, json.RawMessage(`{}`))
			return err
		},
		"a ticket's read": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.ticketsSvc.Get(ctx, f.ticket.ID)
			return err
		},
		"a live frame": func(ctx context.Context, f permFixture) error {
			frame, err := json.Marshal(tickets.UpdatedEvent{Ticket: *f.ticket})
			if err == nil && !memoizedAudience(f.svc)(ctx, tickets.TopicUpdated, json.RawMessage(frame)) {
				err = errors.New("the reader's frame was dropped")
			}
			return err
		},
	}
	cost := func(n int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		grow(t, f, n)
		if rules {
			_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at)
SELECT 'project', p.id, 'role', r.id, '["projects:open","tickets:write"]', '["projects:delete"]', 1, 1 FROM projects p, roles r
WHERE r.workspace_id = 'workspace-default' AND r.is_owner_role = 0
ON CONFLICT(resource_type, resource_id, target, coalesce(user_id, role_id)) DO UPDATE SET allow = excluded.allow, deny = excluded.deny`)
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
SELECT 'project', p.id, 'person', m.user_id, '["projects:open"]', '["tickets:delete"]', 1, 1 FROM projects p, workspace_members m
WHERE m.user_id != 'u-owner'
ON CONFLICT(resource_type, resource_id, target, coalesce(user_id, role_id)) DO UPDATE SET allow = excluded.allow, deny = excluded.deny`)
			require.NoError(t, err)
		}
		out := map[string]int64{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f), name)
			out[name] = st.Count()
		}
		return out
	}
	none := cost(12, false)
	assert.Equal(t, none, cost(2, true), "a few projects, each with rules")
	assert.Equal(t, none, cost(12, true), "many projects, each with rules")
}

// TestRows_ASingleCheckReadsOnlyItsProjectsRules: a check, a ticket's read and a live frame, each naming one project,
// read that project's rules for the person and their roles and no other's, so rules naming them on every project cost
// the same rows and statements at a few projects as at many.
func TestRows_ASingleCheckReadsOnlyItsProjectsRules(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture) error{
		"a project check": func(ctx context.Context, f permFixture) error {
			return f.svc.accessSvc.RequireProject(ctx, pGeneral, permissions.TicketsRead)
		},
		"a ticket's read": func(ctx context.Context, f permFixture) error {
			_, err := f.svc.ticketsSvc.Get(ctx, f.ticket.ID)
			return err
		},
		"a live frame": func(ctx context.Context, f permFixture) error {
			frame, err := json.Marshal(tickets.UpdatedEvent{Ticket: *f.ticket})
			if err == nil && !memoizedAudience(f.svc)(ctx, tickets.TopicUpdated, json.RawMessage(frame)) {
				err = errors.New("the reader's frame was dropped")
			}
			return err
		},
	}
	type cost struct{ statements, rows int64 }
	costs := func(n int) map[string]cost {
		f, st, db := newCountedFixtureDB(t)
		grow(t, f, n)
		_, err := db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
SELECT 'project', p.id, 'person', m.user_id, '["projects:open"]', '[]', 1, 1 FROM projects p, workspace_members m WHERE m.user_id != 'u-owner'
ON CONFLICT(resource_type, resource_id, target, coalesce(user_id, role_id)) DO UPDATE SET allow = excluded.allow`)
		require.NoError(t, err)
		out := map[string]cost{}
		for name, run := range paths {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), f.svc.accessSvc.NewMemo())
			st.Reset()
			require.NoError(t, run(ctx, f), name)
			out[name] = cost{st.Count(), st.Rows()}
		}
		return out
	}
	assert.Equal(t, costs(2), costs(12))
}

// TestStatements_RoleSetCostsOneReadAtAnySize: one role or three cost one statement, and a check the same either way.
func TestStatements_RoleSetCostsOneReadAtAnySize(t *testing.T) {
	f, st, db := newCountedFixtureDB(t)
	cost := func() (roles, check int64) {
		st.Reset()
		_, err := f.store.WorkspaceMembers.MemberRole(context.Background(), wsDefault, uPlain)
		require.NoError(t, err)
		roles = st.Count()
		st.Reset()
		f.svc.accessSvc.WorkspacePermissions(access.WithMemo(context.Background(), f.svc.accessSvc.NewMemo()), uPlain, wsDefault)
		return roles, st.Count()
	}
	oneRoles, oneCheck := cost()
	_, err := db.Exec(`INSERT INTO member_roles (user_id, workspace_id, role_id, created_at) VALUES
    ('u-plain', 'workspace-default', 'role-reader', 2), ('u-plain', 'workspace-default', 'role-manager', 3)`)
	require.NoError(t, err)
	manyRoles, manyCheck := cost()

	assert.Equal(t, int64(1), oneRoles)
	assert.Equal(t, oneRoles, manyRoles)
	assert.Equal(t, oneCheck, manyCheck)
}

// TestStatements_InstanceAreaRulesCostNothingOnMe: rules on every instance area, for each person's roles and for them
// by name, cost /api/auth/me no statement more than none, at a few areas or many, since they come back in one read.
func TestStatements_InstanceAreaRulesCostNothingOnMe(t *testing.T) {
	cost := func(connectorCount int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		areas := []permissions.InstanceArea{}
		for i := range connectorCount {
			areas = append(areas, permissions.ConnectorArea(fmt.Sprintf("c%d", i), "Acme"))
		}
		f.svc.accessSvc.SetInstanceAreas(tenancy.DefaultWorkspaceID, areas)
		for _, area := range f.svc.accessSvc.InstanceAreas() {
			if !rules {
				break
			}
			deny, err := json.Marshal(area.Actions[:1])
			require.NoError(t, err)
			_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, user_id, allow, deny, created_at, updated_at)
SELECT 'instance_area', ?, 'role', id, NULL, '[]', ?, 1, 1 FROM roles WHERE workspace_id = ?
UNION ALL SELECT 'instance_area', ?, 'person', NULL, user_id, ?, '[]', 1, 1 FROM workspace_members WHERE workspace_id = ?`,
				area.Key, string(deny), wsDefault, area.Key, string(deny), wsDefault)
			require.NoError(t, err)
		}
		out := map[string]int64{}
		for _, user := range []string{uSteward, uPlain, uClerk} {
			ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), f.svc.accessSvc.NewMemo())
			st.Reset()
			_, err := f.svc.authSvc.Me(ctx, user)
			require.NoError(t, err)
			out[user] = st.Count()
		}
		return out
	}
	none := cost(0, false)
	assert.Equal(t, none, cost(0, true), "rules on every area")
	assert.Equal(t, none, cost(30, true), "rules on many more areas")
}

// TestStatements_WelcomeProgressCostsTheSameForAnyPage: account_list reads a page's Welcome wizard progress with one
// statement per seam, whether the page holds one person or fifty.
func TestStatements_WelcomeProgressCostsTheSameForAnyPage(t *testing.T) {
	cost := func(n int) int64 {
		f, st := newCountedFixture(t)
		ctx := context.Background()
		now := time.Now()
		require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-newcomer", WorkspaceID: wsDefault, Name: "Newcomer",
			WelcomeSteps: []string{"github", "computer", "projects", "deploy_server", "phone"}, CreatedAt: now, UpdatedAt: now}))
		var page []string
		for i := range n {
			id := fmt.Sprintf("u-newcomer-%d", i)
			_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: id})
			require.NoError(t, err)
			require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: id, WorkspaceID: wsDefault, RoleID: "role-newcomer", CreatedAt: now}))
			_, err = f.svc.welcomeSvc.Skip(ctx, id, "phone")
			require.NoError(t, err)
			page = append(page, id)
		}
		st.Reset()
		progress, err := f.svc.welcomeSvc.ProgressFor(ctx, page)
		require.NoError(t, err)
		require.Len(t, progress, n)
		return st.Count()
	}
	assert.Equal(t, cost(1), cost(50))
}

// TestStatements_TeamListCostsTheSameForAnyNumberOfPeople: the Team list reads every person's Welcome wizard progress
// and the invitation they joined by in a fixed number of statements, whether it holds a few people or forty.
func TestStatements_TeamListCostsTheSameForAnyNumberOfPeople(t *testing.T) {
	cost := func(n int) int64 {
		f, st, db := newCountedFixtureDB(t)
		ctx := context.Background()
		now := time.Now()
		require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-newcomer", WorkspaceID: wsDefault, Name: "Newcomer",
			WelcomeSteps: []string{"github", "computer", "projects", "deploy_server", "phone"}, CreatedAt: now, UpdatedAt: now}))
		for i := range n {
			id := fmt.Sprintf("u-newcomer-%d", i)
			_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: id, Login: id})
			require.NoError(t, err)
			require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: id, WorkspaceID: wsDefault, RoleID: "role-newcomer", CreatedAt: now}))
			_, err = db.ExecContext(ctx, `INSERT INTO invitations (id, token_hash, invited_by, created_at, expires_at, redeemed_at, redeemed_by)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, "inv-"+id, fmt.Sprintf("%064d", i), uOwner, now.Unix(), now.Add(24*time.Hour).Unix(), now.Unix(), id)
			require.NoError(t, err)
		}
		st.Reset()
		team, err := f.svc.tenancySvc.ListTeam(identity.WithActor(ctx, identity.Actor{ID: uOwner}), uOwner)
		require.NoError(t, err)
		joined := 0
		for _, p := range team.People {
			require.NotNil(t, p.Welcome, p.Login)
			if p.Joined != nil {
				joined++
			}
		}
		require.Equal(t, n, joined)
		return st.Count()
	}
	assert.Equal(t, cost(2), cost(40))
}

// TestStatements_MachineRulesCostTheSameAtAnyLength: rules for roles and a person on every machine cost a machine check,
// the machines a person sees, and a machine's four answers, for the instance Owner and for another member, no
// statement more than none, at a few machines or many: the rules reaching a person come in one read by type.
func TestStatements_MachineRulesCostTheSameAtAnyLength(t *testing.T) {
	paths := map[string]func(ctx context.Context, f permFixture, user string) error{
		"a machine check": func(ctx context.Context, f permFixture, _ string) error {
			return f.svc.accessSvc.RequireMachine(ctx, "mach-1", permissions.MachinesSee)
		},
		"the machines seen": func(ctx context.Context, f permFixture, user string) error {
			_, err := f.svc.accessSvc.MachinesSeen(ctx, user)
			return err
		},
		"a machine's permissions": func(ctx context.Context, f permFixture, user string) error {
			_, err := f.svc.accessSvc.MachinePermissions(ctx, user, "mach-1")
			return err
		},
	}
	cost := func(n int, rules bool) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) SELECT 'mach-' || i, 'box ' || i, '/data/nexul', '', 1, 1 FROM n`, n)
		require.NoError(t, err)
		if rules {
			_, err = db.Exec(`INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, user_id, allow, deny, created_at, updated_at)
SELECT 'machine', m.id, 'role', r.id, NULL, '["machines:see","machines:deploy"]', '["machines:delete"]', 1, 1 FROM machines m, roles r
WHERE r.id IN ('role-reader', 'role-writer')
UNION ALL SELECT 'machine', m.id, 'person', NULL, ?, '["machines:see","machines:write"]', '[]', 1, 1 FROM machines m`, uReader)
			require.NoError(t, err)
		}
		out := map[string]int64{}
		for _, user := range []string{uOwner, uReader} {
			for name, run := range paths {
				ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), f.svc.accessSvc.NewMemo())
				st.Reset()
				// Without rules another member finds nothing, which is an answer too.
				_ = run(ctx, f, user)
				out[user+" "+name] = st.Count()
			}
		}
		return out
	}
	none := cost(12, false)
	assert.Equal(t, none, cost(2, true), "a few machines, each with rules")
	assert.Equal(t, none, cost(12, true), "many machines, each with rules")
}

// TestStatements_MachineListCostsTheSameAtAnyLength: the machine list, the runner list, the queue and machine_list cost
// the same statements over 2 machines as over 12, each with a runner and rules for a person and for a role, for the
// instance Owner and for people who see them through rules: the machines a person sees come in one read and filter the
// lists in SQL, never a check per machine.
func TestStatements_MachineListCostsTheSameAtAnyLength(t *testing.T) {
	cost := func(n int) map[string]int64 {
		f, st, db := newCountedFixtureDB(t)
		svc := runner.NewService(f.store.Runners, fakeUpgradeDispatch{}).WithMachines(f.store.Machines).WithGate(f.svc.accessSvc)
		machineList := tool(t, runner.MCPTools(svc), "machine_list")
		for _, insert := range []string{
			`INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) SELECT 'mach-' || i, 'box-' || i, '/data/nexul', '', 1, 1 FROM n`,
			`INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id) SELECT 'r-' || i, 'runner-' || i, 1, 0, 1, 'v1', 'mach-' || i FROM n`,
			`INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at)
SELECT 'machine', 'mach-' || i, 'person', 'u-writer', NULL, '["machines:see","machines:deploy"]', '[]', 1, 1 FROM n
UNION ALL SELECT 'machine', 'mach-' || i, 'role', NULL, 'role-reader', '["machines:see"]', '[]', 1, 1 FROM n`,
		} {
			_, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?) `+insert, n)
			require.NoError(t, err)
		}
		paths := map[string]func(ctx context.Context) error{
			"the machine list": func(ctx context.Context) error { _, err := svc.ListMachines(ctx); return err },
			"the runner list":  func(ctx context.Context) error { _, err := svc.ListRunners(ctx); return err },
			"the queue":        func(ctx context.Context) error { _, err := svc.ListQueue(ctx); return err },
			"machine_list": func(ctx context.Context) error {
				_, err := machineList.Call(ctx, json.RawMessage(`{}`))
				return err
			},
		}
		out := map[string]int64{}
		for _, user := range []string{uOwner, uReader, uWriter} {
			for name, run := range paths {
				ctx := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: user}), f.svc.accessSvc.NewMemo())
				st.Reset()
				require.NoError(t, run(ctx), "%s as %s", name, user)
				out[user+" "+name] = st.Count()
			}
		}
		return out
	}
	assert.Equal(t, cost(2), cost(12))
}
