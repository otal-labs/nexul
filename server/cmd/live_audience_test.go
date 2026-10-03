package main

import (
	"encoding/json"
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
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/tickets"
)

// TestLiveRules_EveryPushedTopicNamesItsRead fails when a topic is bridged to the browser without saying who may
// receive it, since a topic with no rule silently reaches nobody.
func TestLiveRules_EveryPushedTopicNamesItsRead(t *testing.T) {
	direct := []string{topicPresenceChanged, topicTopologyCanvas, agent.TopicAgentStream, plays.TopicPlayRun}
	for _, topic := range append(livePushTopics, direct...) {
		assert.Contains(t, liveRules, topic)
	}
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
