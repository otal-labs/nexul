package chat

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// refuseGate passes everyone except the actions it lists per user, which are forbidden; the server's own calls pass.
type refuseGate map[string][]permissions.Action

func (g refuseGate) Require(ctx context.Context, _ string, action permissions.Action) error {
	if slices.Contains(g[actorID(ctx)], action) {
		return apperrs.ErrForbidden
	}
	return nil
}

// fakeMembers is a workspace's member ids.
type fakeMembers []string

func (m fakeMembers) IsMember(_ context.Context, userID, _ string) (bool, error) {
	return slices.Contains(m, userID), nil
}

func (m fakeMembers) MemberIDs(context.Context, string) ([]string, error) {
	return slices.Clone(m), nil
}

type fakeStanding struct {
	owners, restricted []string
}

func (f fakeStanding) IsOwner(_ context.Context, userID, _ string) (bool, error) {
	return slices.Contains(f.owners, userID), nil
}

func (f fakeStanding) IsRestricted(_ context.Context, userID, _ string) (bool, error) {
	return slices.Contains(f.restricted, userID), nil
}

// fakeThreads refuses the tickets and projects it lists as not found, the way a hidden project reads; the server's
// own calls pass.
type fakeThreads struct {
	hiddenTickets, hiddenProjects []string
}

func (f fakeThreads) RequireTicket(ctx context.Context, ticketID string, _ permissions.Action) error {
	if actorID(ctx) != "" && slices.Contains(f.hiddenTickets, ticketID) {
		return apperrs.ErrNotFound
	}
	return nil
}

func (f fakeThreads) RequireProject(ctx context.Context, projectID string, _ permissions.Action) error {
	if actorID(ctx) != "" && slices.Contains(f.hiddenProjects, projectID) {
		return apperrs.ErrNotFound
	}
	return nil
}

// privateFixture is workspace w-1 with members u-1 to u-3 and the Owner; u-2 and u-3 lack channels:write. #eng is
// public, and #secret is private to u-1 and u-2.
type privateFixture struct {
	repo        *fakeRepo
	s           *Service
	eng, secret *Conversation
}

func newPrivateFixture(t *testing.T, restricted ...string) privateFixture {
	t.Helper()
	repo := newFakeRepo()
	s := newTestService(repo)
	s.SetGate(refuseGate{"u-2": {permissions.ChannelsWrite}, "u-3": {permissions.ChannelsWrite}})
	s.SetMembership(fakeMembers{"u-1", "u-2", "u-3", "u-owner"})
	s.SetStanding(fakeStanding{owners: []string{"u-owner"}, restricted: restricted})
	eng, err := s.CreateChannel(as("u-1"), "w-1", "u-1", "eng")
	require.NoError(t, err)
	secret, err := s.CreatePrivateChannel(as("u-1"), "w-1", "u-1", "secret", KindChannel, []string{"u-2"})
	require.NoError(t, err)
	return privateFixture{repo: repo, s: s, eng: eng, secret: secret}
}

func (f privateFixture) membersChanged(t *testing.T) []ConversationMembersChangedEvent {
	t.Helper()
	var out []ConversationMembersChangedEvent
	for _, e := range f.repo.eventsFor(TopicConversationMembersChanged) {
		out = append(out, e.Payload.(ConversationMembersChangedEvent))
	}
	return out
}

func listedIDs(t *testing.T, s *Service, userID string) []string {
	t.Helper()
	cs, err := s.ListConversations(as(userID), "w-1", userID)
	require.NoError(t, err)
	var out []string
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}

func TestSetChannelPrivate_Refusals(t *testing.T) {
	f := newPrivateFixture(t, "u-2")
	require.NoError(t, f.s.EnsureGeneralChannel(context.Background(), "w-1", "u-1"))
	general, err := f.repo.GetChannelByName(context.Background(), "w-1", GeneralChannelName)
	require.NoError(t, err)
	dm, err := f.s.CreateDM(as("u-1"), "w-1", "u-1", []string{"u-3"})
	require.NoError(t, err)
	thread, err := f.s.GetOrCreateTicketThread(as("u-1"), "w-1", "ticket-1", "u-1")
	require.NoError(t, err)
	tests := []struct {
		name    string
		actor   string
		id      string
		private bool
		keep    []string
		wantErr error
	}{
		{"#general cannot go private", "u-1", general.ID, true, nil, apperrs.ErrInvalid},
		{"#general refuses even a switch to public", "u-1", general.ID, false, nil, apperrs.ErrInvalid},
		{"a DM is not a channel", "u-1", dm.ID, true, nil, apperrs.ErrInvalid},
		{"a ticket thread is not a channel", "u-1", thread.ID, true, nil, apperrs.ErrInvalid},
		{"a private channel the caller is not in reads as not found", "u-3", f.secret.ID, false, nil, apperrs.ErrNotFound},
		{"without channels:write", "u-3", f.eng.ID, true, nil, apperrs.ErrForbidden},
		{"keeping someone outside the workspace", "u-1", f.eng.ID, true, []string{"u-9"}, apperrs.ErrNotFound},
		{"a Restricted member never makes a channel public", "u-2", f.secret.ID, false, nil, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.SetChannelPrivate(as(tt.actor), tt.id, tt.private, tt.keep)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
	assert.Len(t, f.membersChanged(t), 0, "a refused switch publishes nothing")

	t.Run("a storage failure is returned", func(t *testing.T) {
		f.repo.membersErr = errors.New("disk full")
		t.Cleanup(func() { f.repo.membersErr = nil })
		_, err := f.s.SetChannelPrivate(as("u-1"), f.eng.ID, true, nil)
		require.ErrorContains(t, err, "disk full")
	})
}

func TestSetChannelPrivate_SwitchesBothWays(t *testing.T) {
	f := newPrivateFixture(t)
	_, err := f.s.PostMessage(as("u-1"), f.eng.ID, "u-1", "before it went private")
	require.NoError(t, err)

	private, err := f.s.SetChannelPrivate(as("u-1"), f.eng.ID, true, []string{"u-2", " u-2 "})
	require.NoError(t, err)
	assert.True(t, private.Private)
	assert.Equal(t, []string{"u-1", "u-2"}, private.ParticipantIDs, "the switcher always stays")
	_, err = f.s.GetConversation(as("u-3"), f.eng.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "everyone else loses it at once")
	assert.NotContains(t, listedIDs(t, f.s, "u-3"), f.eng.ID)
	assert.Contains(t, listedIDs(t, f.s, "u-owner"), f.eng.ID, "the Owner sees every private channel")

	again, err := f.s.SetChannelPrivate(as("u-1"), f.eng.ID, true, []string{"u-3"})
	require.NoError(t, err)
	assert.Equal(t, []string{"u-1", "u-2"}, again.ParticipantIDs, "switching to what it already is changes nothing")

	_, err = f.s.SetChannelPrivate(as("u-2"), f.eng.ID, false, nil)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "switching takes channels:write")
	public, err := f.s.SetChannelPrivate(as("u-1"), f.eng.ID, false, nil)
	require.NoError(t, err)
	assert.False(t, public.Private)
	assert.Empty(t, public.ParticipantIDs, "a public channel keeps no member list")
	history, err := f.s.ListMessages(as("u-3"), f.eng.ID, "u-3", 50)
	require.NoError(t, err)
	require.Len(t, history, 1, "going public opens the history to the workspace")

	events := f.membersChanged(t)
	require.Len(t, events, 2, "going private, then public; the no-op published nothing")
	assert.Equal(t, ConversationMembersChangedEvent{
		ConversationID: f.eng.ID, WorkspaceID: "w-1", Private: true,
		AddedUserIDs: []string{"u-1", "u-2"}, RemovedUserIDs: []string{"u-3", "u-owner"}, ActorID: "u-1",
	}, events[0], "a switch to private names everyone who lost it as removed")
	assert.Equal(t, ConversationMembersChangedEvent{
		ConversationID: f.eng.ID, WorkspaceID: "w-1", Private: false,
		AddedUserIDs: []string{}, RemovedUserIDs: []string{}, ActorID: "u-1",
	}, events[1])
}

func TestCreatePrivateChannel(t *testing.T) {
	t.Run("refusals", func(t *testing.T) {
		f := newPrivateFixture(t)
		tests := []struct {
			name    string
			actor   string
			kind    Kind
			members []string
			wantErr error
		}{
			{"a DM cannot be private", "u-1", KindDM, nil, apperrs.ErrInvalid},
			{"a starting member outside the workspace", "u-1", KindChannel, []string{"u-9"}, apperrs.ErrNotFound},
			{"without channels:write", "u-3", KindVoiceChannel, nil, apperrs.ErrForbidden},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := f.s.CreatePrivateChannel(as(tt.actor), "w-1", tt.actor, "new", tt.kind, tt.members)
				require.ErrorIs(t, err, tt.wantErr)
			})
		}
	})
	t.Run("a private channel starts with its creator and the members named", func(t *testing.T) {
		f := newPrivateFixture(t)
		assert.True(t, f.secret.Private)
		assert.Equal(t, []string{"u-1", "u-2"}, f.secret.ParticipantIDs)
		created := f.repo.eventsFor(TopicConversationCreated)
		assert.True(t, created[len(created)-1].Payload.(ConversationCreatedEvent).Conversation.Private)
		_, err := f.s.GetConversation(as("u-3"), f.secret.ID)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("a Restricted member creates only private channels", func(t *testing.T) {
		f := newPrivateFixture(t)
		f.s.SetStanding(fakeStanding{restricted: []string{"u-1"}})
		_, err := f.s.CreateChannel(as("u-1"), "w-1", "u-1", "open")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		_, err = f.s.CreateVoiceChannel(as("u-1"), "w-1", "u-1", "open")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		c, err := f.s.CreatePrivateChannel(as("u-1"), "w-1", "u-1", "client", KindVoiceChannel, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"u-1"}, c.ParticipantIDs, "created with them in it")
		assert.Contains(t, listedIDs(t, f.s, "u-1"), c.ID)
	})
}

func TestAddChannelMembers(t *testing.T) {
	f := newPrivateFixture(t)
	tests := []struct {
		name    string
		actor   string
		id      string
		users   []string
		wantErr error
	}{
		{"someone not in the private channel", "u-3", f.secret.ID, []string{"u-3"}, apperrs.ErrNotFound},
		{"a public channel has no members to add", "u-1", f.eng.ID, []string{"u-3"}, apperrs.ErrInvalid},
		{"someone outside the workspace", "u-2", f.secret.ID, []string{"u-9"}, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.AddChannelMembers(as(tt.actor), tt.id, tt.users)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}

	same, err := f.s.AddChannelMembers(as("u-2"), f.secret.ID, []string{"u-1", ""})
	require.NoError(t, err)
	assert.Equal(t, []string{"u-1", "u-2"}, same.ParticipantIDs)
	require.Empty(t, f.membersChanged(t), "adding someone already in it publishes nothing")

	added, err := f.s.AddChannelMembers(as("u-2"), f.secret.ID, []string{"u-3"})
	require.NoError(t, err, "anyone in a private channel adds people, channels:write or not")
	assert.Equal(t, []string{"u-1", "u-2", "u-3"}, added.ParticipantIDs)
	assert.Contains(t, listedIDs(t, f.s, "u-3"), f.secret.ID)
	assert.Equal(t, []string{"u-3"}, f.membersChanged(t)[0].AddedUserIDs)
}

func TestRemoveChannelMembers(t *testing.T) {
	f := newPrivateFixture(t)
	tests := []struct {
		name    string
		actor   string
		id      string
		users   []string
		wantErr error
	}{
		{"nobody named", "u-1", f.secret.ID, []string{" "}, apperrs.ErrInvalid},
		{"someone not in the private channel", "u-3", f.secret.ID, []string{"u-3"}, apperrs.ErrNotFound},
		{"a public channel has no members to remove", "u-1", f.eng.ID, []string{"u-1"}, apperrs.ErrInvalid},
		{"removing someone else without channels:write", "u-2", f.secret.ID, []string{"u-1"}, apperrs.ErrForbidden},
		{"everyone at once leaves nobody", "u-1", f.secret.ID, []string{"u-1", "u-2"}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.s.RemoveChannelMembers(as(tt.actor), tt.id, tt.users)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}

	left, err := f.s.RemoveChannelMembers(as("u-2"), f.secret.ID, []string{"u-2"})
	require.NoError(t, err, "leaving takes no channels:write")
	assert.Equal(t, []string{"u-1"}, left.ParticipantIDs)
	_, err = f.s.GetConversation(as("u-2"), f.secret.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, []string{"u-2"}, f.membersChanged(t)[0].RemovedUserIDs)

	_, err = f.s.RemoveChannelMembers(as("u-1"), f.secret.ID, []string{"u-1"})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "the last member cannot leave")

	_, err = f.s.AddChannelMembers(as("u-1"), f.secret.ID, []string{"u-3"})
	require.NoError(t, err)
	removed, err := f.s.RemoveChannelMembers(as("u-owner"), f.secret.ID, []string{"u-3"})
	require.NoError(t, err, "the Owner manages a private channel they are not in")
	assert.Equal(t, []string{"u-1"}, removed.ParticipantIDs)
}

func TestPrivateChannel_ANonMemberReadsNothingAndTheOwnerReadsAll(t *testing.T) {
	f := newPrivateFixture(t)
	_, err := f.s.PostMessage(as("u-2"), f.secret.ID, "u-2", "hello")
	require.NoError(t, err)

	reads := map[string]func(userID string) error{
		"get": func(u string) error { _, err := f.s.GetConversation(as(u), f.secret.ID); return err },
		"messages": func(u string) error {
			_, err := f.s.ListMessages(as(u), f.secret.ID, u, 50)
			return err
		},
		"post": func(u string) error { _, err := f.s.PostMessage(as(u), f.secret.ID, u, "hi"); return err },
		"read": func(u string) error { return f.s.MarkRead(as(u), f.secret.ID, u) },
	}
	for name, read := range reads {
		require.ErrorIs(t, read("u-3"), apperrs.ErrNotFound, name)
		require.NoError(t, read("u-owner"), name)
		require.NoError(t, read("u-2"), name)
	}
	assert.NotContains(t, listedIDs(t, f.s, "u-3"), f.secret.ID)
	assert.Contains(t, listedIDs(t, f.s, "u-owner"), f.secret.ID)
	counts, err := f.s.UnreadCounts(as("u-3"), "w-1", "u-3")
	require.NoError(t, err)
	assert.NotContains(t, counts, f.secret.ID)
	assert.Contains(t, counts, f.eng.ID)
	counts, err = f.s.UnreadCounts(as("u-owner"), "w-1", "u-owner")
	require.NoError(t, err)
	assert.Contains(t, counts, f.secret.ID)
}

func TestReadsDeleted_APrivateChannelReachesItsMembersAndTheOwner(t *testing.T) {
	f := newPrivateFixture(t)
	gone, err := f.s.DeleteChannel(as("u-1"), f.secret.ID)
	require.NoError(t, err)
	deleted := f.repo.eventsFor(TopicConversationDeleted)
	require.Len(t, deleted, 1)
	evt := deleted[0].Payload.(ConversationDeletedEvent)
	assert.Equal(t, gone.ParticipantIDs, evt.MemberIDs)
	for user, want := range map[string]bool{"u-1": true, "u-2": true, "u-owner": true, "u-3": false} {
		assert.Equal(t, want, f.s.ReadsDeleted(as(user), evt), user)
	}
	public := ConversationDeletedEvent{ConversationID: f.eng.ID, WorkspaceID: "w-1", Kind: KindChannel}
	assert.True(t, f.s.ReadsDeleted(as("u-3"), public))
}

func TestRestrictedMember_ReadsDMsTheirPrivateChannelsAndReadableThreads(t *testing.T) {
	f := newPrivateFixture(t, "u-2")
	f.s.SetThreadGate(fakeThreads{hiddenTickets: []string{"ticket-hidden"}, hiddenProjects: []string{"project-hidden"}})
	dm, err := f.s.CreateDM(as("u-1"), "w-1", "u-1", []string{"u-2"})
	require.NoError(t, err)
	var threads []*Conversation
	for _, ticket := range []string{"ticket-open", "ticket-hidden"} {
		c, err := f.s.GetOrCreateTicketThread(context.Background(), "w-1", ticket, "u-2")
		require.NoError(t, err)
		threads = append(threads, c)
	}
	for _, project := range []string{"project-open", "project-hidden"} {
		c, err := f.s.GetOrCreateInterviewThread(context.Background(), "w-1", project, "u-2")
		require.NoError(t, err)
		threads = append(threads, c)
	}

	assert.ElementsMatch(t, []string{f.secret.ID, dm.ID, threads[0].ID, threads[2].ID}, listedIDs(t, f.s, "u-2"))
	_, err = f.s.GetConversation(as("u-2"), f.eng.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a Restricted member reads no public channel")
	_, err = f.s.ListMessages(as("u-2"), threads[1].ID, "u-2", 50)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a thread of a hidden project reads as not found")
	marks, err := f.s.HasTicketThreads(as("u-2"), []string{"ticket-open", "ticket-hidden"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"ticket-open": true}, marks, "the board marks only threads of projects they hold")
	_, err = f.s.GetOrCreateTicketThread(as("u-2"), "w-1", "ticket-other", "u-2")
	require.NoError(t, err, "a ticket in a project they hold starts a thread")
	f.s.SetThreadGate(fakeThreads{hiddenTickets: []string{"ticket-new"}})
	_, err = f.s.GetOrCreateTicketThread(as("u-2"), "w-1", "ticket-new", "u-2")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a hidden ticket starts no thread")

	counts, err := f.s.UnreadCounts(as("u-2"), "w-1", "u-2")
	require.NoError(t, err)
	assert.NotContains(t, counts, f.eng.ID)
	assert.Contains(t, counts, f.secret.ID)
	assert.ElementsMatch(t, []string{f.secret.ID, f.eng.ID, dm.ID}, listedIDs(t, f.s, "u-1"), "everyone else lists as before")
}

func TestMessageEvents_MarkDMAndPrivateChannelMessagesMembersOnly(t *testing.T) {
	f := newPrivateFixture(t)
	dm, err := f.s.CreateDM(as("u-1"), "w-1", "u-1", []string{"u-3"})
	require.NoError(t, err)
	thread, err := f.s.GetOrCreateTicketThread(as("u-1"), "w-1", "ticket-1", "u-1")
	require.NoError(t, err)
	tests := []struct {
		name        string
		c           *Conversation
		membersOnly bool
	}{
		{"a public channel", f.eng, false},
		{"a ticket thread", thread, false},
		{"a private channel", f.secret, true},
		{"a DM", dm, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f.repo.events = nil
			m, err := f.s.PostMessage(as("u-1"), tt.c.ID, "u-1", "hi")
			require.NoError(t, err)
			_, err = f.s.EditMessage(as("u-1"), m.ID, "u-1", "edited")
			require.NoError(t, err)
			require.NoError(t, f.s.DeleteMessage(as("u-1"), m.ID, "u-1"))
			for _, topic := range []string{TopicMessageCreated, TopicMessageUpdated, TopicMessageDeleted} {
				evts := f.repo.eventsFor(topic)
				require.Len(t, evts, 1)
				raw, err := json.Marshal(evts[0].Payload)
				require.NoError(t, err)
				var payload map[string]any
				require.NoError(t, json.Unmarshal(raw, &payload))
				assert.Equal(t, tt.membersOnly, payload["members_only"] == true, topic)
			}
		})
	}
}
