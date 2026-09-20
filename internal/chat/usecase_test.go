package chat

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

// fakeRepo is an in-memory chat.Repo for use-case tests.
type fakeRepo struct {
	mu            sync.Mutex
	conversations map[string]*Conversation
	participants  map[string]map[string]bool // conversationID -> userID -> true
	messages      map[string]*Message
	unreadState   map[string]map[string]int64 // conversationID -> userID -> last_read_at unix
	events        []eventbus.OutboxEvent

	createConversationErr error
	createMessageErr      error
	updateMessageErr      error
	deleteMessageErr      error
	markReadErr           error
	unreadCountsErr       error
	hasTicketThreadsErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		conversations: map[string]*Conversation{},
		participants:  map[string]map[string]bool{},
		messages:      map[string]*Message{},
		unreadState:   map[string]map[string]int64{},
	}
}

func (f *fakeRepo) CreateConversation(_ context.Context, c *Conversation, participantIDs []string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createConversationErr != nil {
		return f.createConversationErr
	}
	if c.Kind == KindChannel {
		for _, existing := range f.conversations {
			if existing.WorkspaceID == c.WorkspaceID && existing.Kind == KindChannel && existing.Name == c.Name {
				return apperrs.ErrConflict
			}
		}
	}
	if c.Kind == KindTicketThread {
		for _, existing := range f.conversations {
			if existing.TicketID == c.TicketID {
				return apperrs.ErrConflict
			}
		}
	}
	if c.Kind == KindDocThread {
		for _, existing := range f.conversations {
			if existing.DocID == c.DocID {
				return apperrs.ErrConflict
			}
		}
	}
	cp := *c
	f.conversations[c.ID] = &cp
	set := map[string]bool{}
	for _, uid := range participantIDs {
		set[uid] = true
	}
	f.participants[c.ID] = set
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) GetConversation(_ context.Context, id string) (*Conversation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.conversations[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return c, nil
}

func (f *fakeRepo) GetChannelByName(_ context.Context, workspaceID, name string) (*Conversation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.conversations {
		if c.WorkspaceID == workspaceID && c.Kind == KindChannel && c.Name == name {
			return c, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) GetTicketThread(_ context.Context, ticketID string) (*Conversation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.conversations {
		if c.Kind == KindTicketThread && c.TicketID == ticketID {
			return c, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) GetDocThread(_ context.Context, docID string) (*Conversation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.conversations {
		if c.Kind == KindDocThread && c.DocID == docID {
			return c, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) HasTicketThreads(_ context.Context, ticketIDs []string) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hasTicketThreadsErr != nil {
		return nil, f.hasTicketThreadsErr
	}
	want := map[string]bool{}
	for _, id := range ticketIDs {
		want[id] = true
	}
	out := map[string]bool{}
	for _, c := range f.conversations {
		if c.Kind == KindTicketThread && want[c.TicketID] {
			out[c.TicketID] = true
		}
	}
	return out, nil
}

func (f *fakeRepo) ListConversationsForUser(_ context.Context, workspaceID, userID string) ([]*Conversation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Conversation
	for _, c := range f.conversations {
		if c.WorkspaceID != workspaceID {
			continue
		}
		if c.Kind == KindChannel || c.Kind == KindDocThread || f.participants[c.ID][userID] {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (f *fakeRepo) CreateMessage(_ context.Context, m *Message, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createMessageErr != nil {
		return f.createMessageErr
	}
	if _, ok := f.conversations[m.ConversationID]; !ok {
		return apperrs.ErrConflict
	}
	cp := *m
	f.messages[m.ID] = &cp
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) GetMessage(_ context.Context, id string) (*Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.messages[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

func (f *fakeRepo) ListMessages(_ context.Context, conversationID string, limit int) ([]*Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Message
	for _, m := range f.messages {
		if m.ConversationID == conversationID {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeRepo) ListMessagesSince(_ context.Context, conversationID string, since time.Time) ([]*Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*Message
	for _, m := range f.messages {
		if m.ConversationID == conversationID && m.DeletedAt == nil && m.CreatedAt.After(since) {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (f *fakeRepo) SetAgentThread(_ context.Context, conversationID, threadID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.conversations[conversationID]
	if !ok {
		return apperrs.ErrNotFound
	}
	c.AgentThreadID = threadID
	return nil
}

func (f *fakeRepo) SetAgentSyncedAt(_ context.Context, conversationID string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.conversations[conversationID]
	if !ok {
		return apperrs.ErrNotFound
	}
	c.AgentSyncedAt = at
	return nil
}

func (f *fakeRepo) UpdateMessage(_ context.Context, id, body string, mentions []Mention, editedAt time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateMessageErr != nil {
		return f.updateMessageErr
	}
	m, ok := f.messages[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	m.Body = body
	m.Mentions = mentions
	m.EditedAt = &editedAt
	m.UpdatedAt = editedAt
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) DeleteMessage(_ context.Context, id string, deletedAt time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteMessageErr != nil {
		return f.deleteMessageErr
	}
	m, ok := f.messages[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	m.Body = ""
	m.Mentions = nil
	m.DeletedAt = &deletedAt
	m.UpdatedAt = deletedAt
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) MarkRead(_ context.Context, conversationID, userID string, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.markReadErr != nil {
		return f.markReadErr
	}
	if f.unreadState[conversationID] == nil {
		f.unreadState[conversationID] = map[string]int64{}
	}
	f.unreadState[conversationID][userID] = at.Unix()
	return nil
}

func (f *fakeRepo) UnreadCounts(_ context.Context, workspaceID, userID string) ([]UnreadCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.unreadCountsErr != nil {
		return nil, f.unreadCountsErr
	}
	var out []UnreadCount
	for _, c := range f.conversations {
		if c.WorkspaceID != workspaceID {
			continue
		}
		if c.Kind != KindChannel && c.Kind != KindDocThread && !f.participants[c.ID][userID] {
			continue
		}
		lastRead := f.unreadState[c.ID][userID]
		count := 0
		for _, m := range f.messages {
			if m.ConversationID != c.ID || m.DeletedAt != nil || m.AuthorID == userID {
				continue
			}
			if m.CreatedAt.Unix() > lastRead {
				count++
			}
		}
		out = append(out, UnreadCount{ConversationID: c.ID, Kind: c.Kind, DocID: c.DocID, Count: count})
	}
	return out, nil
}

func (f *fakeRepo) eventsFor(topic string) []eventbus.OutboxEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []eventbus.OutboxEvent
	for _, e := range f.events {
		if e.Topic == topic {
			out = append(out, e)
		}
	}
	return out
}

func newTestService(repo *fakeRepo) *Service {
	s := NewService(repo)
	s.now = func() time.Time { return fixedNow }
	return s
}

// fakeDocAccess is an in-memory chat.DocAccess for doc thread permission tests; allowed keys "userID:docID".
type fakeDocAccess struct {
	allowed map[string]bool
	err     error
}

func newFakeDocAccess(allowed ...string) *fakeDocAccess {
	set := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		set[a] = true
	}
	return &fakeDocAccess{allowed: set}
}

func (f *fakeDocAccess) Can(_ context.Context, userID, docID string, _ permissions.Action) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.allowed[userID+":"+docID], nil
}

func newTestServiceWithDocAccess(repo *fakeRepo, access DocAccess) *Service {
	s := newTestService(repo)
	s.SetDocAccess(access)
	return s
}

func TestCreateChannel(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateChannel(context.Background(), "", "u-1", "eng")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty creator id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateChannel(context.Background(), "w-1", "", "eng")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateChannel(context.Background(), "w-1", "u-1", "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("creates a lower-cased channel and adds the creator as a participant", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "Engineering")
		require.NoError(t, err)
		assert.Equal(t, "engineering", c.Name)
		assert.Equal(t, KindChannel, c.Kind)
		assert.True(t, repo.participants[c.ID]["u-1"])
		evts := repo.eventsFor(TopicConversationCreated)
		require.Len(t, evts, 1)
		e, ok := evts[0].Payload.(ConversationCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, c.ID, e.Conversation.ID)
	})
	t.Run("duplicate channel name in the workspace is a conflict", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		_, err = s.CreateChannel(context.Background(), "w-1", "u-2", "eng")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
}

func TestCreateVoiceChannel(t *testing.T) {
	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateVoiceChannel(context.Background(), "w-1", "u-1", "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("creates a lower-cased voice channel and adds the creator as a participant", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateVoiceChannel(context.Background(), "w-1", "u-1", "War Room")
		require.NoError(t, err)
		assert.Equal(t, "war room", c.Name)
		assert.Equal(t, KindVoiceChannel, c.Kind)
		assert.True(t, repo.participants[c.ID]["u-1"])
		evts := repo.eventsFor(TopicConversationCreated)
		require.Len(t, evts, 1)
		e, ok := evts[0].Payload.(ConversationCreatedEvent)
		require.True(t, ok)
		assert.Equal(t, c.ID, e.Conversation.ID)
	})
}

func TestEnsureGeneralChannel(t *testing.T) {
	t.Run("creates #general when missing", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		require.NoError(t, s.EnsureGeneralChannel(context.Background(), "w-1", "u-1"))
		c, err := repo.GetChannelByName(context.Background(), "w-1", GeneralChannelName)
		require.NoError(t, err)
		assert.Equal(t, "u-1", c.CreatedBy)
	})
	t.Run("is idempotent when #general already exists", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		require.NoError(t, s.EnsureGeneralChannel(context.Background(), "w-1", "u-1"))
		require.NoError(t, s.EnsureGeneralChannel(context.Background(), "w-1", "u-1"))
		var count int
		for _, c := range repo.conversations {
			if c.WorkspaceID == "w-1" && c.Kind == KindChannel {
				count++
			}
		}
		assert.Equal(t, 1, count)
	})
	t.Run("propagates a non-conflict error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createConversationErr = errors.New("db down")
		s := newTestService(repo)
		err := s.EnsureGeneralChannel(context.Background(), "w-1", "u-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}

func TestCreateDM(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.CreateDM(context.Background(), "", "u-1", []string{"u-2"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("just the creator makes a self-DM (notes to self)", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateDM(context.Background(), "w-1", "u-1", nil)
		require.NoError(t, err)
		assert.Equal(t, KindDM, c.Kind)
		assert.Len(t, repo.participants[c.ID], 1)
		assert.True(t, repo.participants[c.ID]["u-1"])
	})
	t.Run("folds the creator into the participant set and dedupes", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateDM(context.Background(), "w-1", "u-1", []string{"u-2", "u-1", "u-2"})
		require.NoError(t, err)
		assert.Equal(t, KindDM, c.Kind)
		assert.Len(t, repo.participants[c.ID], 2)
		assert.True(t, repo.participants[c.ID]["u-1"])
		assert.True(t, repo.participants[c.ID]["u-2"])
	})
}

func TestGetOrCreateTicketThread(t *testing.T) {
	t.Run("empty ticket id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.GetOrCreateTicketThread(context.Background(), "w-1", "", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("creates the thread on first use", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.GetOrCreateTicketThread(context.Background(), "w-1", "t-1", "u-1")
		require.NoError(t, err)
		assert.Equal(t, KindTicketThread, c.Kind)
		assert.Equal(t, "t-1", c.TicketID)
		require.Len(t, repo.eventsFor(TopicConversationCreated), 1)
	})
	t.Run("returns the existing thread without creating a second one", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		first, err := s.GetOrCreateTicketThread(context.Background(), "w-1", "t-1", "u-1")
		require.NoError(t, err)
		second, err := s.GetOrCreateTicketThread(context.Background(), "w-1", "t-1", "u-2")
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.Len(t, repo.eventsFor(TopicConversationCreated), 1)
	})
	t.Run("a losing race re-fetches the winner's thread instead of erroring", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		// Simulate a concurrent winner: the row exists by the time our create lands, so the fake's own duplicate-ticket-id check trips ErrConflict, same as the real unique index would.
		existing, err := s.GetOrCreateTicketThread(context.Background(), "w-1", "t-1", "u-1")
		require.NoError(t, err)
		repo.createConversationErr = nil // conflict comes from the duplicate ticket_id check, not a forced repo error
		got, err := s.createConversation(context.Background(), &Conversation{WorkspaceID: "w-1", Kind: KindTicketThread, TicketID: "t-1", CreatedBy: "u-2"}, []string{"u-2"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		refetched, err := s.GetOrCreateTicketThread(context.Background(), "w-1", "t-1", "u-2")
		require.NoError(t, err)
		assert.Equal(t, existing.ID, refetched.ID)
		assert.Nil(t, got)
	})
}

func TestGetOrCreateDocThread(t *testing.T) {
	t.Run("empty doc id is invalid", func(t *testing.T) {
		s := newTestServiceWithDocAccess(newFakeRepo(), newFakeDocAccess("u-1:"))
		_, err := s.GetOrCreateDocThread(context.Background(), "w-1", "", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("no wired DocAccess denies, even the creator", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("caller without docs:thread is forbidden, not an empty result", func(t *testing.T) {
		s := newTestServiceWithDocAccess(newFakeRepo(), newFakeDocAccess())
		_, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("creates the thread on first use", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestServiceWithDocAccess(repo, newFakeDocAccess("u-1:doc-1"))
		c, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-1")
		require.NoError(t, err)
		assert.Equal(t, KindDocThread, c.Kind)
		assert.Equal(t, "doc-1", c.DocID)
		require.Len(t, repo.eventsFor(TopicConversationCreated), 1)
	})
	t.Run("a second doc gets its own thread; one doc never gets two", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestServiceWithDocAccess(repo, newFakeDocAccess("u-1:doc-1", "u-1:doc-2"))
		first, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-1")
		require.NoError(t, err)
		second, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-2", "u-1")
		require.NoError(t, err)
		assert.NotEqual(t, first.ID, second.ID)
		again, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-1")
		require.NoError(t, err)
		assert.Equal(t, first.ID, again.ID)
		require.Len(t, repo.eventsFor(TopicConversationCreated), 2)
	})
	t.Run("a losing race re-fetches the winner's thread instead of erroring", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestServiceWithDocAccess(repo, newFakeDocAccess("u-1:doc-1", "u-2:doc-1"))
		existing, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-1")
		require.NoError(t, err)
		got, err := s.createConversation(context.Background(), &Conversation{WorkspaceID: "w-1", Kind: KindDocThread, DocID: "doc-1", CreatedBy: "u-2"}, []string{"u-2"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
		refetched, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "u-2")
		require.NoError(t, err)
		assert.Equal(t, existing.ID, refetched.ID)
		assert.Nil(t, got)
	})
}

func TestListMessages_DocThread(t *testing.T) {
	t.Run("a caller without docs:thread is forbidden, not an empty list", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestServiceWithDocAccess(repo, newFakeDocAccess("owner:doc-1"))
		thread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "owner")
		require.NoError(t, err)
		_, err = s.ListMessages(context.Background(), thread.ID, "other-user", 50)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("a caller with docs:thread reads the messages", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestServiceWithDocAccess(repo, newFakeDocAccess("owner:doc-1"))
		thread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "owner")
		require.NoError(t, err)
		ms, err := s.ListMessages(context.Background(), thread.ID, "owner", 50)
		require.NoError(t, err)
		assert.Empty(t, ms)
	})
	t.Run("a non-doc-thread conversation is untouched by the gate", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		_, err = s.ListMessages(context.Background(), c.ID, "anyone", 50)
		require.NoError(t, err)
	})
}

func TestPostMessage_DocThread(t *testing.T) {
	t.Run("a caller without docs:thread is forbidden", func(t *testing.T) {
		repo := newFakeRepo()
		access := newFakeDocAccess("owner:doc-1")
		s := newTestServiceWithDocAccess(repo, access)
		thread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "owner")
		require.NoError(t, err)
		_, err = s.PostMessage(context.Background(), thread.ID, "other-user", "hi")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("a caller with docs:thread posts", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestServiceWithDocAccess(repo, newFakeDocAccess("owner:doc-1"))
		thread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "owner")
		require.NoError(t, err)
		m, err := s.PostMessage(context.Background(), thread.ID, "owner", "hi")
		require.NoError(t, err)
		assert.Equal(t, "hi", m.Body)
	})
}

func TestHasTicketThreads(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.GetOrCreateTicketThread(context.Background(), "w-1", "t-1", "u-1")
	require.NoError(t, err)
	out, err := s.HasTicketThreads(context.Background(), []string{"t-1", "t-2", "", "  "})
	require.NoError(t, err)
	assert.True(t, out["t-1"])
	assert.False(t, out["t-2"])
}

func TestListConversations(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ListConversations(context.Background(), "", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("returns channels plus the user's own DMs", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.CreateChannel(context.Background(), "w-1", "u-1", "general")
		require.NoError(t, err)
		dm, err := s.CreateDM(context.Background(), "w-1", "u-1", []string{"u-2"})
		require.NoError(t, err)
		_, err = s.CreateDM(context.Background(), "w-1", "u-3", []string{"u-4"})
		require.NoError(t, err)

		got, err := s.ListConversations(context.Background(), "w-1", "u-1")
		require.NoError(t, err)
		require.Len(t, got, 2)
		var ids []string
		for _, c := range got {
			ids = append(ids, c.ID)
		}
		assert.Contains(t, ids, dm.ID)
	})
	t.Run("filters out doc threads the caller lacks docs:thread on", func(t *testing.T) {
		repo := newFakeRepo()
		access := newFakeDocAccess("owner:doc-1")
		s := newTestServiceWithDocAccess(repo, access)
		thread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "owner")
		require.NoError(t, err)

		got, err := s.ListConversations(context.Background(), "w-1", "owner")
		require.NoError(t, err)
		var ids []string
		for _, c := range got {
			ids = append(ids, c.ID)
		}
		assert.Contains(t, ids, thread.ID, "owner holds docs:thread and sees it")

		got, err = s.ListConversations(context.Background(), "w-1", "other-user")
		require.NoError(t, err)
		ids = nil
		for _, c := range got {
			ids = append(ids, c.ID)
		}
		assert.NotContains(t, ids, thread.ID, "other-user lacks docs:thread and never sees it")
	})
}

func TestPostMessage(t *testing.T) {
	t.Run("empty conversation id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.PostMessage(context.Background(), "", "u-1", "hi")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty body is invalid", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		_, err = s.PostMessage(context.Background(), c.ID, "u-1", "   ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("posting to a nonexistent conversation is a conflict", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.PostMessage(context.Background(), "missing", "u-1", "hi")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrConflict))
	})
	t.Run("posts and parses mentions, enqueueing chat.message.created", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		m, err := s.PostMessage(context.Background(), c.ID, "u-1", "hey @onik97 and @Agent")
		require.NoError(t, err)
		assert.Equal(t, c.ID, m.ConversationID)
		assert.Len(t, m.Mentions, 2)
		evts := repo.eventsFor(TopicMessageCreated)
		require.Len(t, evts, 1)
	})
}

func TestEditMessage(t *testing.T) {
	setup := func(t *testing.T) (*fakeRepo, *Service, *Conversation) {
		t.Helper()
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		return repo, s, c
	}
	t.Run("empty message id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.EditMessage(context.Background(), "", "u-1", "hi")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing message is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.EditMessage(context.Background(), "missing", "u-1", "hi")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("only the author may edit", func(t *testing.T) {
		repo, s, c := setup(t)
		m, err := s.PostMessage(context.Background(), c.ID, "u-1", "hi")
		require.NoError(t, err)
		_, err = s.EditMessage(context.Background(), m.ID, "u-2", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
		_ = repo
	})
	t.Run("cannot edit a deleted message", func(t *testing.T) {
		_, s, c := setup(t)
		m, err := s.PostMessage(context.Background(), c.ID, "u-1", "hi")
		require.NoError(t, err)
		require.NoError(t, s.DeleteMessage(context.Background(), m.ID, "u-1"))
		_, err = s.EditMessage(context.Background(), m.ID, "u-1", "edited")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("updates the body, re-parses mentions, and stamps edited_at", func(t *testing.T) {
		repo, s, c := setup(t)
		m, err := s.PostMessage(context.Background(), c.ID, "u-1", "hi")
		require.NoError(t, err)
		got, err := s.EditMessage(context.Background(), m.ID, "u-1", "hi @onik97")
		require.NoError(t, err)
		assert.Equal(t, "hi @onik97", got.Body)
		assert.Len(t, got.Mentions, 1)
		require.NotNil(t, got.EditedAt)
		require.Len(t, repo.eventsFor(TopicMessageUpdated), 1)
	})
}

func TestDeleteMessage(t *testing.T) {
	t.Run("empty message id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.DeleteMessage(context.Background(), "", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing message is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.DeleteMessage(context.Background(), "missing", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("only the author may delete", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		m, err := s.PostMessage(context.Background(), c.ID, "u-1", "hi")
		require.NoError(t, err)
		err = s.DeleteMessage(context.Background(), m.ID, "u-2")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("deleting twice is a no-op", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		m, err := s.PostMessage(context.Background(), c.ID, "u-1", "hi")
		require.NoError(t, err)
		require.NoError(t, s.DeleteMessage(context.Background(), m.ID, "u-1"))
		require.NoError(t, s.DeleteMessage(context.Background(), m.ID, "u-1"))
		require.Len(t, repo.eventsFor(TopicMessageDeleted), 1)
	})
	t.Run("soft-deletes and enqueues chat.message.deleted", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		m, err := s.PostMessage(context.Background(), c.ID, "u-1", "hi")
		require.NoError(t, err)
		require.NoError(t, s.DeleteMessage(context.Background(), m.ID, "u-1"))
		stored := repo.messages[m.ID]
		assert.Empty(t, stored.Body)
		assert.NotNil(t, stored.DeletedAt)
		evts := repo.eventsFor(TopicMessageDeleted)
		require.Len(t, evts, 1)
		e, ok := evts[0].Payload.(MessageDeletedEvent)
		require.True(t, ok)
		assert.Equal(t, m.ID, e.MessageID)
	})
}

func TestMarkRead(t *testing.T) {
	t.Run("empty conversation id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		err := s.MarkRead(context.Background(), "", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("advances the read cursor", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)
		require.NoError(t, s.MarkRead(context.Background(), c.ID, "u-2"))
		assert.Equal(t, fixedNow.Unix(), repo.unreadState[c.ID]["u-2"])
	})
}

func TestGetConversation(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)
	got, err := s.GetConversation(context.Background(), c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, got.ID)
	_, err = s.GetConversation(context.Background(), "missing")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestListMessagesSince(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)
	_, err = s.PostMessage(context.Background(), c.ID, "u-1", "before")
	require.NoError(t, err)
	cutoff := fixedNow
	s.now = func() time.Time { return fixedNow.Add(time.Minute) }
	after, err := s.PostMessage(context.Background(), c.ID, "u-1", "after")
	require.NoError(t, err)

	got, err := s.ListMessagesSince(context.Background(), c.ID, cutoff)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, after.ID, got[0].ID)
}

func TestAgentThreadAndSyncTracking(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)

	require.NoError(t, s.SetAgentThread(context.Background(), c.ID, "thread-1"))
	assert.Equal(t, "thread-1", repo.conversations[c.ID].AgentThreadID)

	syncedAt := fixedNow.Add(time.Hour)
	require.NoError(t, s.MarkAgentSynced(context.Background(), c.ID, syncedAt))
	assert.True(t, repo.conversations[c.ID].AgentSyncedAt.Equal(syncedAt))

	require.Error(t, s.SetAgentThread(context.Background(), "", "thread-1"))
}

func TestPostAgentAndSystemMessages(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
	require.NoError(t, err)

	agentMsg, err := s.PostAgentMessage(context.Background(), c.ID, "u-1", "here's the reply")
	require.NoError(t, err)
	assert.Equal(t, AuthorAgent, agentMsg.AuthorKind)
	assert.Equal(t, "u-1", agentMsg.AuthorID)

	sysMsg, err := s.PostSystemMessage(context.Background(), c.ID, "u-1", "connect T3 Code in settings")
	require.NoError(t, err)
	assert.Equal(t, AuthorSystem, sysMsg.AuthorKind)

	userMsg, err := s.PostMessage(context.Background(), c.ID, "u-1", "hi")
	require.NoError(t, err)
	assert.Equal(t, AuthorUser, userMsg.AuthorKind)
}

func TestUnreadCounts(t *testing.T) {
	t.Run("empty workspace id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.UnreadCounts(context.Background(), "", "u-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("counts messages since the read cursor, excluding own and deleted messages", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		c, err := s.CreateChannel(context.Background(), "w-1", "u-1", "eng")
		require.NoError(t, err)

		_, err = s.PostMessage(context.Background(), c.ID, "u-2", "first")
		require.NoError(t, err)
		require.NoError(t, s.MarkRead(context.Background(), c.ID, "u-1"))

		s.now = func() time.Time { return fixedNow.Add(time.Minute) }
		_, err = s.PostMessage(context.Background(), c.ID, "u-2", "second")
		require.NoError(t, err)
		_, err = s.PostMessage(context.Background(), c.ID, "u-1", "own message, should not count")
		require.NoError(t, err)
		toDelete, err := s.PostMessage(context.Background(), c.ID, "u-2", "will be deleted")
		require.NoError(t, err)
		require.NoError(t, s.DeleteMessage(context.Background(), toDelete.ID, "u-2"))

		counts, err := s.UnreadCounts(context.Background(), "w-1", "u-1")
		require.NoError(t, err)
		assert.Equal(t, 1, counts[c.ID])
	})
	t.Run("omits a doc thread the caller lacks docs:thread on", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestServiceWithDocAccess(repo, newFakeDocAccess("owner:doc-1"))
		thread, err := s.GetOrCreateDocThread(context.Background(), "w-1", "doc-1", "owner")
		require.NoError(t, err)
		_, err = s.PostMessage(context.Background(), thread.ID, "owner", "hi")
		require.NoError(t, err)

		ownerCounts, err := s.UnreadCounts(context.Background(), "w-1", "owner")
		require.NoError(t, err)
		assert.Contains(t, ownerCounts, thread.ID)

		otherCounts, err := s.UnreadCounts(context.Background(), "w-1", "other-user")
		require.NoError(t, err)
		assert.NotContains(t, otherCounts, thread.ID)
	})
}
