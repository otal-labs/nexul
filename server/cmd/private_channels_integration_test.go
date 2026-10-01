package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/chat"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// TestIntegration_PrivateChannel walks every read of a private channel through the wired services: its members and
// the Owner read it, and to anyone else it is not found, in lists, messages, files, the voice call, and live frames.
func TestIntegration_PrivateChannel(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	room, err := s.chatSvc.CreatePrivateChannel(as(uWriter), "workspace-default", uWriter, "leads", chat.KindChannel, []string{uReader})
	require.NoError(t, err)
	call, err := s.chatSvc.CreatePrivateChannel(as(uWriter), "workspace-default", uWriter, "leads-call", chat.KindVoiceChannel, []string{uReader})
	require.NoError(t, err)
	file, err := s.attachmentsSvc.Upload(as(uWriter), attachments.Owner{ConversationID: room.ID}, "plan.png", []byte("\x89PNG\r\n\x1a\n"))
	require.NoError(t, err)
	frame, err := json.Marshal(chat.MessageCreatedEvent{Message: chat.Message{ConversationID: room.ID, Body: "hi"}})
	require.NoError(t, err)
	audience := liveAudience{access: s.accessSvc, tickets: s.ticketsSvc, chat: s.chatSvc, deploy: s.deploySvc}

	reads := map[string]string{uWriter: ok, uReader: ok, uOwner: ok, uPlain: notFound, uOutsider: notFound}
	lists := map[string]string{uWriter: ok, uReader: ok, uOwner: ok, uPlain: hidden, uOutsider: notFound}
	// A reader's join passes the read and stops at the LiveKit connector this instance has not configured.
	joins := map[string]string{uWriter: invalid, uReader: invalid, uOwner: invalid, uPlain: notFound, uOutsider: notFound}
	cases := []struct {
		name string
		call func(ctx context.Context, user string) error
		want map[string]string
	}{
		{"list", func(ctx context.Context, user string) error {
			cs, err := s.chatSvc.ListConversations(ctx, "workspace-default", user)
			return contains(cs, err, func(c *chat.Conversation) bool { return c.ID == room.ID })
		}, lists},
		{"messages", func(ctx context.Context, user string) error {
			_, err := s.chatSvc.ListMessages(ctx, room.ID, user, 50)
			return err
		}, reads},
		{"post", func(ctx context.Context, user string) error {
			_, err := s.chatSvc.PostMessage(ctx, room.ID, user, "hello")
			return err
		}, reads},
		{"attachment", func(ctx context.Context, _ string) error {
			_, err := s.attachmentsSvc.Get(ctx, file.ID)
			return err
		}, reads},
		{"voice join", func(ctx context.Context, user string) error {
			_, err := s.voiceSvc.Join(ctx, call.ID, user)
			return err
		}, joins},
		{"live frame", func(ctx context.Context, _ string) error {
			if !audience.allows(ctx, chat.TopicMessageCreated, json.RawMessage(frame)) {
				return errHidden
			}
			return nil
		}, map[string]string{uWriter: ok, uReader: ok, uOwner: ok, uPlain: hidden, uOutsider: hidden}},
	}
	for _, tc := range cases {
		for user, want := range tc.want {
			assert.Equal(t, want, outcome(tc.call(as(user), user)), "%s as %s", tc.name, user)
		}
	}

	deleted, err := s.chatSvc.DeleteChannel(as(uOwner), room.ID)
	require.NoError(t, err)
	raw, err := json.Marshal(chat.ConversationDeletedEvent{ConversationID: deleted.ID, WorkspaceID: deleted.WorkspaceID, Kind: deleted.Kind, Name: deleted.Name, Private: true, MemberIDs: deleted.ParticipantIDs})
	require.NoError(t, err)
	for user, want := range map[string]bool{uWriter: true, uReader: true, uOwner: true, uPlain: false, uOutsider: false} {
		assert.Equal(t, want, audience.allows(as(user), chat.TopicConversationDeleted, json.RawMessage(raw)), "deleted as %s", user)
	}
}

// TestIntegration_SwitchingPrivateReachesEveryOpenSidebar: the switch's frame reaches the people it removed, whose
// sidebars drop the channel, and switching back reaches everyone, with the history readable again.
func TestIntegration_SwitchingPrivateReachesEveryOpenSidebar(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	audience := liveAudience{access: s.accessSvc, tickets: s.ticketsSvc, chat: s.chatSvc, deploy: s.deploySvc}
	_, err := s.chatSvc.PostMessage(as(uPlain), f.channel.ID, uPlain, "said in public")
	require.NoError(t, err)
	reaches := func(c *chat.Conversation, removed []string, user string) bool {
		raw, err := json.Marshal(chat.ConversationMembersChangedEvent{ConversationID: c.ID, WorkspaceID: c.WorkspaceID, Private: c.Private, RemovedUserIDs: removed})
		require.NoError(t, err)
		return audience.allows(as(user), chat.TopicConversationMembersChanged, json.RawMessage(raw))
	}

	private, err := s.chatSvc.SetChannelPrivate(as(uWriter), f.channel.ID, true, []string{uReader})
	require.NoError(t, err)
	removed := []string{uOwner, uPlain}
	for user, want := range map[string]bool{uWriter: true, uReader: true, uOwner: true, uPlain: true, uOutsider: false} {
		assert.Equal(t, want, reaches(private, removed, user), "private as %s", user)
	}
	_, err = s.chatSvc.ListMessages(as(uPlain), f.channel.ID, uPlain, 50)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	public, err := s.chatSvc.SetChannelPrivate(as(uWriter), f.channel.ID, false, nil)
	require.NoError(t, err)
	for user, want := range map[string]bool{uPlain: true, uOverwrite: true, uOutsider: false} {
		assert.Equal(t, want, reaches(public, nil, user), "public as %s", user)
	}
	history, err := s.chatSvc.ListMessages(as(uPlain), f.channel.ID, uPlain, 50)
	require.NoError(t, err)
	require.Len(t, history, 1)
}
