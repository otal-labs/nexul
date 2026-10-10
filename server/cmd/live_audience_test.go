package main

import (
	"encoding/json"
	"flag"
	"maps"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/repository"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/runner"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

var updateLiveTopics = flag.Bool("update-live-topics", false, "rewrite the browser's copy of the pushed topic list")

// liveTopicsFile is what the browser's contract test reads to check its topic map against the rules.
const liveTopicsFile = "../../web/src/hooks/liveTopics.generated.json"

// TestLiveRules_MatchWhatIsPushed fails when a topic is bridged to the browser without saying who may receive it,
// since a topic with no rule silently reaches nobody, and when a rule names a topic nothing pushes.
func TestLiveRules_MatchWhatIsPushed(t *testing.T) {
	direct := []string{topicPresenceChanged, topicTopologyCanvas, agent.TopicAgentStream, plays.TopicPlayRun}
	assert.ElementsMatch(t, append(slices.Clone(livePushTopics), direct...), slices.Collect(maps.Keys(liveRules)))
}

// TestLiveTopicsFile_MatchesTheRules fails when the generated topic list is stale, the way sqlc diff does.
func TestLiveTopicsFile_MatchesTheRules(t *testing.T) {
	want, err := json.MarshalIndent(slices.Sorted(maps.Keys(liveRules)), "", "  ")
	require.NoError(t, err)
	want = append(want, '\n')
	if *updateLiveTopics {
		require.NoError(t, os.WriteFile(liveTopicsFile, want, 0o644))
	}
	got, err := os.ReadFile(liveTopicsFile)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got), "stale; regenerate with make live-topics")
}

// TestLiveAudience_FramesFollowTheEntitysRead pushes real payloads through the wired audience: a frame reaches the
// people who could load its entity and nobody else.
func TestLiveAudience_FramesFollowTheEntitysRead(t *testing.T) {
	f := newPermFixture(t)
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	stack, err := f.store.Stacks.GetByID(t.Context(), f.stack)
	require.NoError(t, err)
	cases := []struct {
		topic   string
		payload any
		want    map[string]bool
	}{
		{tickets.TopicCreated, tickets.CreatedEvent{Ticket: *f.ticket}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
		{chat.TopicMessageCreated, chat.MessageCreatedEvent{Message: chat.Message{ConversationID: f.dm.ID, Body: "hi"}}, map[string]bool{uWriter: true, uPlain: false, uOutsider: false}},
		{deploy.TopicStackUpdated, deploy.StackEvent{Stack: *stack}, map[string]bool{uReader: true, uPlain: false}},
		{auth.TopicSessionCreated, auth.SessionChangedEvent{UserID: uPlain}, map[string]bool{uPlain: true, uOwner: false}},
		{tenancy.TopicWorkspaceUpdated, tenancy.WorkspaceEvent{WorkspaceID: "workspace-default", Name: "Acme", Slug: "acme"}, map[string]bool{uOwner: true, uPlain: true, uOutsider: false}},
		{access.TopicGrantChanged, access.GrantEvent{ResourceType: "doc", ResourceID: "doc-1", UserID: uPlain}, map[string]bool{uPlain: true, uOwner: false, uOutsider: false}},
		{roles.TopicUpdated, roles.RoleEvent{RoleID: "role-1", WorkspaceID: "workspace-default"}, map[string]bool{uOwner: true, uPlain: true, uOutsider: false}},
		{docs.TopicFolderCreated, docs.FolderEvent{Folder: docs.Folder{ID: "f-1", ProjectID: "project-general", Name: "GetSource"}}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
		{memories.TopicUpdated, memories.UpdatedEvent{Memory: memories.MemoryRef{ID: "memory-1", WorkspaceID: "workspace-default", ProjectID: "project-general"}}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
		{memories.TopicAnswerSaved, memories.AnswerEvent{WorkspaceID: "workspace-default", ProjectID: "project-general", Question: "Testing"}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
		{memories.TopicSourceAdded, memories.SourceEvent{WorkspaceID: "workspace-default", ProjectID: "project-general", SourceID: "s-1", Kind: memories.SourceDoc}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
		{memories.TopicDraftSaved, memories.DraftEvent{WorkspaceID: "workspace-default", ProjectID: "project-general", DraftID: "d-1", Question: "Testing"}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
		{workspace.TopicProjectSetupChanged, workspace.ProjectSetupChangedEvent{ProjectID: "project-general", WorkspaceID: "workspace-default", Setup: workspace.NewSetup(true)}, map[string]bool{uReader: true, uOwner: true, uOutsider: false}},
		{docs.TopicWatchersChanged, docs.WatchersChangedEvent{Doc: docs.WatchedDoc{ID: f.doc, ProjectID: "project-general"}, UserID: uReader, Watching: true}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.payload)
		require.NoError(t, err)
		for user, want := range tc.want {
			assert.Equal(t, want, a.allows(as(user), tc.topic, json.RawMessage(raw)), "%s as %s", tc.topic, user)
		}
	}
}

// TestLiveAudience_MemoryDeletedStaysInItsProject: the deleted frame carries the memory's title, so it reaches
// readers of the memory's own project, not someone who reads memories in another workspace, and a frame from before
// it named its project reaches nobody.
func TestLiveAudience_MemoryDeletedStaysInItsProject(t *testing.T) {
	f := newPermFixture(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, f.store.Workspaces.Create(ctx, &tenancy.Workspace{ID: "workspace-other", Name: "Other", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-other-reader", WorkspaceID: "workspace-other", Name: "Reader", Permissions: grant("memories:read"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uOutsider, WorkspaceID: "workspace-other", RoleID: "role-other-reader", CreatedAt: now}))
	a := liveAudience{access: f.svc.accessSvc}
	deleted := memories.DeletedEvent{ID: "memory-1", WorkspaceID: "workspace-default", ProjectID: "project-general", Title: "Deploy keys", AuthorID: uOwner}
	raw, err := json.Marshal(deleted)
	require.NoError(t, err)
	for user, want := range map[string]bool{uReader: true, uPlain: false, uOutsider: false} {
		assert.Equal(t, want, a.allows(as(user), memories.TopicDeleted, json.RawMessage(raw)), "as %s", user)
	}
	deleted.ProjectID = ""
	raw, err = json.Marshal(deleted)
	require.NoError(t, err)
	assert.False(t, a.allows(as(uOwner), memories.TopicDeleted, json.RawMessage(raw)))
}

// TestLiveAudience_PlayFramesReachWhoSeesThePlay: a play reaches whoever lists the workspace's plays and whoever is
// offered it as a button through plays:run on that play, never a member who sees neither.
func TestLiveAudience_PlayFramesReachWhoSeesThePlay(t *testing.T) {
	f := newPermFixture(t)
	ctx := t.Context()
	now := time.Now()
	const uPlayReader = "u-play-reader"
	_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: uPlayReader, Provider: auth.ProviderGitHub, ProviderUserID: uPlayReader, Login: uPlayReader})
	require.NoError(t, err)
	require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-play-reader", WorkspaceID: "workspace-default", Name: "Play reader", Permissions: grant("plays:read"), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: uPlayReader, WorkspaceID: "workspace-default", RoleID: "role-play-reader", CreatedAt: now}))
	require.NoError(t, f.store.Access.Set(ctx, "play", "play-1", uPlain, grant("plays:run"), nil))
	a := liveAudience{access: f.svc.accessSvc}
	cases := []struct {
		topic   string
		payload any
		want    map[string]bool
	}{
		{plays.TopicCreated, plays.CreatedEvent{Play: plays.Play{ID: "play-1", WorkspaceID: "workspace-default"}}, map[string]bool{uOwner: true, uPlayReader: true, uPlain: true, uReader: false, uOutsider: false}},
		{plays.TopicUpdated, plays.UpdatedEvent{Play: plays.Play{ID: "play-2", WorkspaceID: "workspace-default"}}, map[string]bool{uPlayReader: true, uPlain: false, uReader: false}},
		{plays.TopicDeleted, plays.DeletedEvent{ID: "play-1", Label: "Ship it", WorkspaceID: "workspace-default"}, map[string]bool{uPlayReader: true, uPlain: true, uReader: false, uOutsider: false}},
		{plays.TopicDeleted, plays.DeletedEvent{ID: "play-1", Label: "Ship it"}, map[string]bool{uOwner: false}},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.payload)
		require.NoError(t, err)
		for user, want := range tc.want {
			assert.Equal(t, want, a.allows(as(user), tc.topic, json.RawMessage(raw)), "%s as %s", tc.topic, user)
		}
	}
}

// TestLiveAudience_AutoPlayFramesReachTheirReaders: an auto play reaches whoever holds autoplays:read in its workspace,
// not a play reader without it, and a frame naming no workspace reaches nobody.
func TestLiveAudience_AutoPlayFramesReachTheirReaders(t *testing.T) {
	f := newPermFixture(t)
	ctx := t.Context()
	now := time.Now()
	const uAutoReader, uPlayReader = "u-auto-reader", "u-play-reader"
	for user, perm := range map[string]string{uAutoReader: "autoplays:read", uPlayReader: "plays:read"} {
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user})
		require.NoError(t, err)
		require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-" + user, WorkspaceID: "workspace-default", Name: user, Permissions: grant(perm), CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: "role-" + user, CreatedAt: now}))
	}
	a := liveAudience{access: f.svc.accessSvc}
	auto := plays.AutoPlay{ID: "ap-1", PlayID: "play-1", WorkspaceID: "workspace-default"}
	cases := []struct {
		topic   string
		payload any
		want    map[string]bool
	}{
		{plays.TopicAutoPlayCreated, plays.AutoPlayEvent{AutoPlay: auto}, map[string]bool{uOwner: true, uAutoReader: true, uPlayReader: false, uOutsider: false}},
		{plays.TopicAutoPlayUpdated, plays.AutoPlayEvent{AutoPlay: auto}, map[string]bool{uAutoReader: true, uPlayReader: false}},
		{plays.TopicAutoPlayDeleted, plays.AutoPlayDeletedEvent{ID: "ap-1", PlayID: "play-1", WorkspaceID: "workspace-default"}, map[string]bool{uAutoReader: true, uPlayReader: false, uOutsider: false}},
		{plays.TopicAutoPlayDeleted, plays.AutoPlayDeletedEvent{ID: "ap-1", PlayID: "play-1"}, map[string]bool{uOwner: false}},
		{plays.TopicAutoPlayLimitsUpdated, plays.AutoPlayLimitsEvent{WorkspaceID: "workspace-default", DailyCapPerTicket: 3}, map[string]bool{uAutoReader: true, uPlayReader: false, uOutsider: false}},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.payload)
		require.NoError(t, err)
		for user, want := range tc.want {
			assert.Equal(t, want, a.allows(as(user), tc.topic, json.RawMessage(raw)), "%s as %s", tc.topic, user)
		}
	}
}

// TestLiveAudience_InstanceAndDeletedTicketFrames: the upgrade card follows instance:read, held in any workspace,
// a deleted ticket's title reaches only readers of its project's tickets, and a new notice only its recipients.
func TestLiveAudience_InstanceAndDeletedTicketFrames(t *testing.T) {
	f := newPermFixture(t)
	a := liveAudience{access: f.svc.accessSvc}
	cases := []struct {
		topic   string
		payload any
		want    map[string]bool
	}{
		{runner.TopicInstanceUpgradeChanged, runner.Upgrade{ID: "upgrade-1", ToVersion: "v0.3.0", Status: runner.UpgradeStatusPending}, map[string]bool{uOwner: true, uSteward: true, uReader: false, uPlain: false, uOutsider: false}},
		{tickets.TopicDeleted, tickets.DeletedEvent{ID: "t-1", Title: "Login times out", ProjectID: "project-general"}, map[string]bool{uReader: true, uPlain: false, uOutsider: false}},
		{tickets.TopicDeleted, tickets.DeletedEvent{ID: "t-1", Title: "Login times out"}, map[string]bool{uOwner: false}},
		{workspace.TopicNotificationCreated, workspace.NotificationCreatedEvent{UserIDs: []string{uPlain, uReader}, WorkspaceID: "workspace-default"}, map[string]bool{uPlain: true, uReader: true, uOwner: false, uOutsider: false}},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.payload)
		require.NoError(t, err)
		for user, want := range tc.want {
			assert.Equal(t, want, a.allows(as(user), tc.topic, json.RawMessage(raw)), "%s as %s", tc.topic, user)
		}
	}
}

// TestLiveAudience_InstallationFramesReachWhoSeesTheWorkspacesInstallations: an assignment change reaches who reads
// the workspace's installations and who lists its repositories in the wizard, and nobody outside the workspace.
func TestLiveAudience_InstallationFramesReachWhoSeesTheWorkspacesInstallations(t *testing.T) {
	f := newPermFixture(t)
	ctx := t.Context()
	now := time.Now()
	const uConnectors, uWizard, uTickets = "u-connectors", "u-wizard", "u-tickets"
	for user, perm := range map[string]string{uConnectors: "connectors:read", uWizard: "projects:write", uTickets: "tickets:read"} {
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user})
		require.NoError(t, err)
		require.NoError(t, f.store.Roles.Create(ctx, &roles.Role{ID: "role-" + user, WorkspaceID: "workspace-default", Name: user, Permissions: grant(perm), CreatedAt: now, UpdatedAt: now}))
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: "role-" + user, CreatedAt: now}))
	}
	a := liveAudience{access: f.svc.accessSvc}
	for _, topic := range []string{repository.TopicInstallationAssigned, repository.TopicInstallationUnassigned} {
		raw, err := json.Marshal(repository.InstallationEvent{AccountID: 11, AccountLogin: "acme", WorkspaceID: "workspace-default"})
		require.NoError(t, err)
		for user, want := range map[string]bool{uOwner: true, uConnectors: true, uWizard: true, uTickets: false, uOutsider: false} {
			assert.Equal(t, want, a.allows(as(user), topic, json.RawMessage(raw)), "%s as %s", topic, user)
		}
		raw, err = json.Marshal(repository.InstallationEvent{AccountID: 11, AccountLogin: "acme"})
		require.NoError(t, err)
		assert.False(t, a.allows(as(uOwner), topic, json.RawMessage(raw)), "a frame naming no workspace reaches nobody")
	}
}
