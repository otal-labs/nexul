package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/botwebhook"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// uBotter holds every botwebhook permission and nothing else; uTicketBotter also reads tickets; uBotReader only reads bots.
const (
	uBotter       = "u-botter"
	uTicketBotter = "u-ticket-botter"
	uBotReader    = "u-bot-reader"
)

// TestIntegration_BotsFollowTheirConversationsGate asks each conversation kind, through the wired services, who may
// see and add its bots: botwebhook bits are not enough without reading the conversation the way chat does.
func TestIntegration_BotsFollowTheirConversationsGate(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	ctx := context.Background()
	now := time.Now()
	for user, perms := range map[string][]string{
		uBotter:       {"botwebhook:read", "botwebhook:write", "botwebhook:delete"},
		uTicketBotter: {"botwebhook:read", "botwebhook:write", "tickets:read"},
		uBotReader:    {"botwebhook:read"},
	} {
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: user, Provider: auth.ProviderGitHub, ProviderUserID: user, Login: user})
		require.NoError(t, err)
		role := &roles.Role{ID: "role-" + user, WorkspaceID: "workspace-default", Name: user, Permissions: grant(perms...), CreatedAt: now, UpdatedAt: now}
		require.NoError(t, f.store.Roles.Create(ctx, role))
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: user, WorkspaceID: "workspace-default", RoleID: role.ID, CreatedAt: now}))
	}
	ticketThread, err := s.chatSvc.ExistingThread(as(uOwner), chat.KindTicketThread, f.ticket.ID, uOwner)
	require.NoError(t, err)
	docThread, err := s.chatSvc.GetOrCreateDocThread(as(uOwner), "workspace-default", f.doc, uOwner)
	require.NoError(t, err)
	private, err := s.chatSvc.CreatePrivateChannel(as(uOwner), "workspace-default", uOwner, "leads", chat.KindChannel, nil)
	require.NoError(t, err)

	create := func(conversationID string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			_, err := s.botwebhookSvc.Create(ctx, conversationID, "CI", "")
			return err
		}
	}
	cases := []struct {
		name string
		call func(ctx context.Context) error
		want map[string]string
	}{
		{"channel", create(f.channel.ID), map[string]string{uBotter: ok, uBotReader: forbidden, uPlain: forbidden, uOutsider: notFound}},
		{"DM", create(f.dm.ID), map[string]string{uOwner: ok, uBotter: notFound}},
		{"private channel", create(private.ID), map[string]string{uOwner: ok, uBotter: notFound}},
		{"ticket thread", create(ticketThread.ID), map[string]string{uTicketBotter: ok, uBotter: forbidden}},
		{"doc thread", create(docThread.ID), map[string]string{uOwner: ok, uBotter: forbidden}},
	}
	for _, tc := range cases {
		for user, want := range tc.want {
			assert.Equal(t, want, outcome(tc.call(as(user))), "%s as %s", tc.name, user)
		}
	}

	for user, wantURL := range map[string]bool{uBotter: true, uBotReader: false} {
		bots, err := s.botwebhookSvc.List(as(user), f.channel.ID, false)
		require.NoError(t, err, user)
		require.Len(t, bots, 1, user)
		assert.Equal(t, wantURL, bots[0].URL != "", "only botwebhook:write sees the URL, as %s", user)
	}
}

// TestIntegration_BotLiveFrames reach whoever lists the conversation's bots, and nobody else.
func TestIntegration_BotLiveFrames(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	audience := liveAudience{access: s.accessSvc, tickets: s.ticketsSvc, chat: s.chatSvc, deploy: s.deploySvc}
	frame := func(c *chat.Conversation) json.RawMessage {
		raw, err := json.Marshal(botwebhook.Event{BotwebhookID: "b-1", ConversationID: c.ID, WorkspaceID: c.WorkspaceID, Name: "CI"})
		require.NoError(t, err)
		return raw
	}
	for _, topic := range botwebhook.Topics() {
		assert.True(t, audience.allows(as(uOwner), topic, frame(f.channel)), topic)
		assert.False(t, audience.allows(as(uPlain), topic, frame(f.channel)), "%s without botwebhook:read", topic)
		assert.False(t, audience.allows(as(uOutsider), topic, frame(f.channel)), topic)
	}
	assert.False(t, audience.allows(as(uReader), botwebhook.TopicCreated, frame(f.dm)), "a DM's bots stay with its participants")
}
