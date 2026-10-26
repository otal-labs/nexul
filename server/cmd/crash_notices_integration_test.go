package main

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func aimCrashNotices(t *testing.T, f permFixture, conversationID, botID string) error {
	t.Helper()
	stack, err := f.svc.deploySvc.GetStack(as(uOwner), f.stack)
	require.NoError(t, err)
	stack.DockerNetwork, stack.CrashTickets = "net", true
	stack.CrashNoticeConversationID, stack.CrashNoticeBotID = conversationID, botID
	_, err = f.svc.deploySvc.UpdateStack(as(uOwner), *stack)
	return err
}

func channelMessages(t *testing.T, f permFixture) []*chat.Message {
	t.Helper()
	ms, err := f.svc.chatSvc.ListMessages(as(uOwner), f.channel.ID, uOwner, 50)
	require.NoError(t, err)
	return ms
}

// TestCrashNotices_ANamedChannelGetsOneNoticePerMoment walks a stack's crash through the wired consumers into a channel:
// the crash links its ticket, a redelivered event posts nothing new, the recovery follows it, and a bot posts as itself.
func TestCrashNotices_ANamedChannelGetsOneNoticePerMoment(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withMachineM1(t, f)
	d := f.svc.deploySvc
	ctx := context.Background()
	require.NoError(t, aimCrashNotices(t, f, f.channel.ID, ""))

	stopped := crashEvent(t, deploy.TopicContainerStopped, f.stack, "o1")
	require.NoError(t, d.HandleContainerStopped(ctx, stopped))
	require.NoError(t, d.HandleContainerStopped(ctx, stopped))
	require.NoError(t, d.HandleContainerRecovered(ctx, crashEvent(t, deploy.TopicContainerRecovered, f.stack, "o1")))

	ts := projectTickets(t, f)
	filed := ts[len(ts)-1]
	var bodies []string
	for _, m := range channelMessages(t, f) {
		assert.Equal(t, chat.AuthorSystem, m.AuthorKind)
		bodies = append(bodies, m.Body)
	}
	assert.ElementsMatch(t, []string{
		"`web` crashed on `web`: exit 137, out of memory. /default/tickets/" + filed.ID,
		"`web` came back on `web` after 3m0s down. /default/tickets/" + filed.ID,
	}, bodies, "a redelivery posts nothing new")

	bot, err := f.svc.botwebhookSvc.Create(as(uOwner), f.channel.ID, "Alerts", "")
	require.NoError(t, err)
	require.NoError(t, aimCrashNotices(t, f, f.channel.ID, bot.ID))
	asBot := crashEvent(t, deploy.TopicContainerStopped, f.stack, "o2")
	require.NoError(t, d.HandleContainerStopped(ctx, asBot))
	require.NoError(t, d.HandleContainerStopped(ctx, asBot))

	ms := channelMessages(t, f)
	require.Len(t, ms, 3, "a redelivery posts nothing new as the bot either")
	posted := slices.IndexFunc(ms, func(m *chat.Message) bool { return m.AuthorKind == chat.AuthorBot })
	require.NotEqual(t, -1, posted, "the bot posted the notice")
	assert.Equal(t, bot.ID, ms[posted].AuthorID)
	assert.Equal(t, "Alerts", ms[posted].AuthorName)

	_, err = f.svc.botwebhookSvc.Delete(as(uOwner), bot.ID)
	require.NoError(t, err)
	require.NoError(t, d.HandleContainerStopped(ctx, crashEvent(t, deploy.TopicContainerStopped, f.stack, "o3")), "a deleted bot drops the notice")
	assert.Len(t, channelMessages(t, f), 3)
}

func TestCrashNotices_AStackNamesOnlyAConversationItsEditorMayPostIn(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	withMachineM1(t, f)
	thread, err := f.svc.chatSvc.ExistingThread(as(uOwner), chat.KindTicketThread, f.ticket.ID, uOwner)
	require.NoError(t, err)
	elsewhere, err := f.svc.chatSvc.CreateChannel(as(uOwner), "workspace-default", uOwner, "ops")
	require.NoError(t, err)
	otherBot, err := f.svc.botwebhookSvc.Create(as(uOwner), elsewhere.ID, "CI", "")
	require.NoError(t, err)

	tests := []struct {
		name                string
		conversationID, bot string
	}{
		{"an unknown conversation", "no-such-conversation", ""},
		{"a ticket's thread", thread.ID, ""},
		{"a DM", f.dm.ID, ""},
		{"a bot of another conversation", f.channel.ID, otherBot.ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, aimCrashNotices(t, f, tt.conversationID, tt.bot), apperrs.ErrInvalid)
		})
	}
	assert.NoError(t, aimCrashNotices(t, f, elsewhere.ID, otherBot.ID), "a bot of the channel named")
}
