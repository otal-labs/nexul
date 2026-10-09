package main

import (
	"context"
	"encoding/json"
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
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
	"github.com/otal-labs/nexul/internal/tickets"
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
// role, membership, and Project access change clears it (ADRs 0087, 0088, 0097-0099, 0135).
func TestIntegration_PermissionSuitesAnswerTheSameWithAMemo(t *testing.T) {
	suites := map[string]func(*testing.T){
		"permission table":                  TestIntegration_PermissionTable,
		"grants never exceed the giver":     TestIntegration_GrantsNeverExceedTheGiver,
		"first user holds instance bits":    TestIntegration_FirstUserReachesEveryFormerAdminCapability,
		"restricted member":                 TestIntegration_RestrictedMember,
		"restricted notices and trails":     TestIntegration_RestrictedMember_NoticesAndTrailsFollowAccess,
		"restricted live frames":            TestIntegration_RestrictedMember_LiveFramesFollowAccess,
		"restricted granting and cleanup":   TestIntegration_RestrictedMember_GrantingAndCleanup,
		"private channel":                   TestIntegration_PrivateChannel,
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
// to the next frame after each kind of change: a role's permissions, a membership, a workspace-wide overwrite,
// Project access, and a doc's own overwrite.
func TestIntegration_LiveAudienceMemo_FollowsEveryChange(t *testing.T) {
	f := newPermFixture(t)
	restrictedClient(t, f)
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
	_, err = s.rolesSvc.Update(as(uOwner), wsDefault, "role-reader", uOwner, "Reader", grant("docs:read"))
	require.NoError(t, err)
	assert.False(t, reaches(uReader, ticketFrame), "a role losing tickets:read")
	_, err = s.rolesSvc.Update(as(uOwner), wsDefault, "role-reader", uOwner, "Reader", grant("tickets:read", "docs:read"))
	require.NoError(t, err)
	assert.True(t, reaches(uReader, ticketFrame), "the role given it back")
	require.NoError(t, s.tenancySvc.RemoveMember(as(uOwner), uOwner, wsDefault, uReader))
	assert.False(t, reaches(uReader, ticketFrame), "removed from the workspace")

	assert.False(t, reaches(uPlain, ticketFrame))
	allow := grant("tickets:read")
	require.NoError(t, s.tenancySvc.SetMemberOverrides(as(uOwner), uOwner, wsDefault, uPlain, &allow, nil))
	assert.True(t, reaches(uPlain, ticketFrame), "a workspace-wide allow")
	require.NoError(t, s.tenancySvc.SetMemberOverrides(as(uOwner), uOwner, wsDefault, uPlain, &permissions.Set{}, &allow))
	assert.False(t, reaches(uPlain, ticketFrame), "a workspace-wide deny")

	assert.False(t, reaches(uClient, ticketFrame))
	giveGeneral(t, f, "tickets:read")
	assert.True(t, reaches(uClient, ticketFrame), "Project access given")
	giveGeneral(t, f)
	assert.False(t, reaches(uClient, ticketFrame), "Project access taken")

	assert.False(t, reaches(uWriter, docFrame))
	require.NoError(t, s.accessSvc.SetGrants(as(uOwner), uOwner, []string{f.doc}, []string{uWriter}, []permissions.Action{permissions.DocsRead}, true))
	assert.True(t, reaches(uWriter, docFrame), "the doc shared")
	require.NoError(t, s.accessSvc.SetGrants(as(uOwner), uOwner, []string{f.doc}, []string{uWriter}, []permissions.Action{permissions.DocsRead}, false))
	assert.False(t, reaches(uWriter, docFrame), "the doc unshared")
}

// TestAccessMemo_ConcurrentRequestsShareNothing runs many people's requests at once, each on its own memo, beside the
// live audience's shared one, and every answer must be the one that person gets alone; -race watches the memos.
func TestAccessMemo_ConcurrentRequestsShareNothing(t *testing.T) {
	f := newPermFixture(t)
	restrictedClient(t, f)
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
	svc, store := wiredOver(t, db)
	return seedPermFixture(t, svc, store), st
}

// grow adds n of every listed thing to the fixture's project: docs shared with the reader, tickets with threads the
// reader is in, doc threads, and public and private channels.
func grow(t *testing.T, f permFixture, n int) []string {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	ticketIDs := []string{f.ticket.ID}
	for i := range n {
		docID := fmt.Sprintf("doc-%d-%d", n, i)
		require.NoError(t, f.store.Docs.Create(ctx, &docs.Doc{ID: docID, ProjectID: pGeneral, Title: "Spec", Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, f.store.Access.Set(ctx, "doc", docID, uReader, grant("docs:read", "docs:thread"), nil))
		_, err := f.svc.chatSvc.GetOrCreateDocThread(ctx, wsDefault, docID, uOwner)
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
	}
	cost := func(n int) map[string]int64 {
		f, st := newCountedFixture(t)
		ticketIDs := grow(t, f, n)
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
	f, st := newCountedFixture(t)
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
