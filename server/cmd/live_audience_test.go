package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/plays"
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
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.payload)
		require.NoError(t, err)
		for user, want := range tc.want {
			assert.Equal(t, want, a.allows(as(user), tc.topic, json.RawMessage(raw)), "%s as %s", tc.topic, user)
		}
	}
}
