package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var notifFixedNow = time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

// fakeNotifRepo is an in-memory NotificationRepo mirroring the storage layer's transactional-outbox contract: the event is only enqueued if at least one row was newly inserted.
type fakeNotifRepo struct {
	mu        sync.Mutex
	notifs    map[string]*Notification
	byUser    map[string][]*Notification
	outbox    []eventbus.OutboxEvent
	createErr error
	listErr   error
	countErr  error
	markErr   error
}

func newFakeNotifRepo() *fakeNotifRepo {
	return &fakeNotifRepo{notifs: map[string]*Notification{}, byUser: map[string][]*Notification{}}
}

func (f *fakeNotifRepo) CreateMany(_ context.Context, ns []*Notification, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	inserted := false
	for _, n := range ns {
		if _, ok := f.notifs[n.ID]; ok {
			continue // idempotent: same source event already created it
		}
		f.notifs[n.ID] = n
		f.byUser[n.UserID] = append(f.byUser[n.UserID], n)
		inserted = true
	}
	if inserted {
		f.outbox = append(f.outbox, evts...)
	}
	return nil
}

// create is a test-only convenience for seeding a single notification row outside of a fan-out.
func (f *fakeNotifRepo) create(t *testing.T, n *Notification) {
	t.Helper()
	require.NoError(t, f.CreateMany(context.Background(), []*Notification{n}))
}

// topics returns the topics of every outbox event enqueued so far.
func (f *fakeNotifRepo) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.outbox))
	for _, evt := range f.outbox {
		out = append(out, evt.Topic)
	}
	return out
}

func (f *fakeNotifRepo) List(_ context.Context, userID string, limit int) ([]*Notification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	all := append([]*Notification(nil), f.byUser[userID]...)
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (f *fakeNotifRepo) UnreadCount(_ context.Context, userID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.countErr != nil {
		return 0, f.countErr
	}
	n := 0
	for _, notif := range f.byUser[userID] {
		if !notif.Read {
			n++
		}
	}
	return n, nil
}

func (f *fakeNotifRepo) MarkRead(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.markErr != nil {
		return f.markErr
	}
	n, ok := f.notifs[id]
	if !ok || n.UserID != userID {
		return apperrs.ErrNotFound
	}
	n.Read = true
	return nil
}

func (f *fakeNotifRepo) MarkAllRead(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.markErr != nil {
		return f.markErr
	}
	for _, n := range f.byUser[userID] {
		n.Read = true
	}
	return nil
}

func (f *fakeNotifRepo) notifsFor(userID string) []*Notification {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*Notification(nil), f.byUser[userID]...)
}

// fakeNotifUsers resolves recipients for notification generation tests.
type fakeNotifUsers struct {
	mu      sync.Mutex
	byLogin map[string]*User
	list    []*User
	err     error
}

func newFakeNotifUsers(users ...*User) *fakeNotifUsers {
	f := &fakeNotifUsers{byLogin: map[string]*User{}}
	for _, u := range users {
		f.byLogin[strings.ToLower(u.Login)] = u
		f.list = append(f.list, u)
	}
	return f
}

func (f *fakeNotifUsers) GetUserByLogin(_ context.Context, login string) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	u, ok := f.byLogin[strings.ToLower(login)]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return u, nil
}

func (f *fakeNotifUsers) ListUsers(_ context.Context) ([]*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return append([]*User(nil), f.list...), nil
}

func (f *fakeNotifUsers) LoginForUserID(_ context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.list {
		if u.ID == userID {
			return u.Login, nil
		}
	}
	return "", apperrs.ErrNotFound
}

// fakeMemberStore is a WorkspaceMemberStore stub mapping workspace id to member user ids, for memory.updated
// fan-out tests.
type fakeMemberStore struct {
	byWorkspace map[string][]string
}

func (f *fakeMemberStore) ListMemberUserIDs(_ context.Context, workspaceID string) ([]string, error) {
	return f.byWorkspace[workspaceID], nil
}

// fakeAccessChecker is a PermissionChecker stub gating memory.updated fan-out by a fixed allow/deny set.
type fakeAccessChecker struct {
	denyUserIDs map[string]bool
}

func (f *fakeAccessChecker) HasPermission(_ context.Context, userID, _ string, _ permissions.Action) bool {
	return !f.denyUserIDs[userID]
}

func newTestNotifService(repo *fakeNotifRepo, users *fakeNotifUsers) *NotificationService {
	return newTestNotifServiceWith(repo, users, &fakeMemberStore{byWorkspace: map[string][]string{}}, &fakeAccessChecker{})
}

func newTestNotifServiceWith(repo *fakeNotifRepo, users *fakeNotifUsers, members WorkspaceMemberStore, access PermissionChecker) *NotificationService {
	s := NewNotificationService(repo, users, members, access)
	s.now = func() time.Time { return notifFixedNow }
	return s
}

func notifUser(id, login string) *User {
	return &User{ID: id, Login: login}
}

func notifID(i int) string {
	return "n" + strconv.Itoa(i)
}

func mkNotif(id, userID string) *Notification {
	return &Notification{ID: id, UserID: userID, Kind: KindDocCreated, SubjectType: SubjectDoc, SubjectID: "d-1", SubjectTitle: "Spec"}
}

func TestNotifList(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		_, err := s.List(context.Background(), "  ", 10)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.listErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers())
		_, err := s.List(context.Background(), "u1", 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listErr)
	})
	t.Run("defaults limit to 50", func(t *testing.T) {
		repo := newFakeNotifRepo()
		for i := 0; i < 60; i++ {
			repo.create(t, &Notification{ID: notifID(i), UserID: "u1"})
		}
		s := newTestNotifService(repo, newFakeNotifUsers())
		ns, err := s.List(context.Background(), "u1", 0)
		require.NoError(t, err)
		require.Len(t, ns, 50)
	})
	t.Run("limits rows", func(t *testing.T) {
		repo := newFakeNotifRepo()
		for i := 0; i < 5; i++ {
			repo.create(t, &Notification{ID: notifID(i), UserID: "u1"})
		}
		s := newTestNotifService(repo, newFakeNotifUsers())
		ns, err := s.List(context.Background(), "u1", 2)
		require.NoError(t, err)
		require.Len(t, ns, 2)
	})
}

func TestNotifUnreadCount(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		_, err := s.UnreadCount(context.Background(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.countErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers())
		_, err := s.UnreadCount(context.Background(), "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.countErr)
	})
	t.Run("counts only unread", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, &Notification{ID: "n1", UserID: "u1"})
		repo.create(t, &Notification{ID: "n2", UserID: "u1"})
		require.NoError(t, repo.MarkRead(context.Background(), "u1", "n1"))
		s := newTestNotifService(repo, newFakeNotifUsers())
		n, err := s.UnreadCount(context.Background(), "u1")
		require.NoError(t, err)
		assert.Equal(t, 1, n)
	})
}

func TestNotifMarkRead(t *testing.T) {
	t.Run("empty args are invalid", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := s.MarkRead(context.Background(), " ", "n1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
		err = s.MarkRead(context.Background(), "u1", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing notification is not found", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := s.MarkRead(context.Background(), "u1", "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.markErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers())
		err := s.MarkRead(context.Background(), "u1", "n1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.markErr)
	})
	t.Run("marks read", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, &Notification{ID: "n1", UserID: "u1"})
		s := newTestNotifService(repo, newFakeNotifUsers())
		require.NoError(t, s.MarkRead(context.Background(), "u1", "n1"))
		got, err := s.List(context.Background(), "u1", 0)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.True(t, got[0].Read)
	})
}

func TestNotifMarkAllRead(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := s.MarkAllRead(context.Background(), "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.markErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers())
		err := s.MarkAllRead(context.Background(), "u1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.markErr)
	})
	t.Run("marks all read for the user only", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, &Notification{ID: "n1", UserID: "u1"})
		repo.create(t, &Notification{ID: "n2", UserID: "u1"})
		repo.create(t, &Notification{ID: "n3", UserID: "u2"})
		s := newTestNotifService(repo, newFakeNotifUsers())
		require.NoError(t, s.MarkAllRead(context.Background(), "u1"))
		for _, n := range repo.notifsFor("u1") {
			assert.True(t, n.Read)
		}
		for _, n := range repo.notifsFor("u2") {
			assert.False(t, n.Read)
		}
	})
}

func TestHandleMemoryUpdated(t *testing.T) {
	payload := map[string]any{
		"memory":     map[string]any{"id": "m-1", "workspace_id": "ws-1", "title": "Deploy quirks", "version": 3},
		"author_id":  "u1",
		"author_via": "",
	}

	t.Run("fans out to every member with memories:read, excluding the author", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"), notifUser("u3", "bob"))
		members := &fakeMemberStore{byWorkspace: map[string][]string{"ws-1": {"u1", "u2", "u3"}}}
		access := &fakeAccessChecker{denyUserIDs: map[string]bool{"u3": true}}
		s := newTestNotifServiceWith(repo, users, members, access)

		require.NoError(t, HandleMemoryUpdated(context.Background(), s, notifEvFor(t, "memory.updated", payload)))

		assert.Empty(t, repo.notifsFor("u1")) // the author never notifies themself
		require.Len(t, repo.notifsFor("u2"), 1)
		n := repo.notifsFor("u2")[0]
		assert.Equal(t, KindMemoryUpdated, n.Kind)
		assert.Equal(t, SubjectMemory, n.SubjectType)
		assert.Equal(t, "m-1", n.SubjectID)
		assert.Contains(t, n.SubjectTitle, "Deploy quirks")
		assert.Contains(t, n.SubjectTitle, "v3")
		assert.Contains(t, n.SubjectTitle, "onik97") // the author's login, resolved from their id
		assert.Empty(t, repo.notifsFor("u3"))        // denied memories:read
	})

	t.Run("author_via mcp is attributed to the Agent", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"))
		members := &fakeMemberStore{byWorkspace: map[string][]string{"ws-1": {"u1", "u2"}}}
		s := newTestNotifServiceWith(repo, users, members, &fakeAccessChecker{})
		mcpPayload := map[string]any{
			"memory":    map[string]any{"id": "m-1", "workspace_id": "ws-1", "title": "Deploy quirks", "version": 2},
			"author_id": "u1", "author_via": "mcp",
		}
		require.NoError(t, HandleMemoryUpdated(context.Background(), s, notifEvFor(t, "memory.updated", mcpPayload)))
		require.Len(t, repo.notifsFor("u2"), 1)
		assert.Contains(t, repo.notifsFor("u2")[0].SubjectTitle, "Agent via onik97")
	})

	t.Run("no members or access gate wired is a no-op, not a panic", func(t *testing.T) {
		s := NewNotificationService(newFakeNotifRepo(), newFakeNotifUsers(), nil, nil)
		require.NoError(t, HandleMemoryUpdated(context.Background(), s, notifEvFor(t, "memory.updated", payload)))
	})

	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := HandleMemoryUpdated(context.Background(), s, eventbus.Event{ID: "e1", Payload: []byte(`{`)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})

	t.Run("missing memory id is fatal", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := HandleMemoryUpdated(context.Background(), s, notifEvFor(t, "memory.updated", map[string]any{"memory": map[string]any{}}))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

func notifEvFor(t *testing.T, topic string, payload any) eventbus.Event {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return eventbus.Event{ID: "evt-1", Topic: topic, Payload: b}
}

func TestHandleTicketCreated(t *testing.T) {
	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := HandleTicketCreated(context.Background(), s, eventbus.Event{ID: "e1", Payload: []byte(`{`)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("missing ticket id is fatal", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", map[string]any{"ticket": map[string]any{}}))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("fans out to assignee and mentions", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{
				"id": "t-1", "title": "Fix @alice bug", "body": "@onik97 please triage", "assignee": "onik97",
			},
		}
		require.NoError(t, HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload)))

		onik := repo.notifsFor("u1")
		require.Len(t, onik, 1)
		assert.Equal(t, KindTicketAssigned, onik[0].Kind)
		assert.Equal(t, SubjectTicket, onik[0].SubjectType)
		assert.Equal(t, "t-1", onik[0].SubjectID)
		assert.Equal(t, "Fix @alice bug", onik[0].SubjectTitle)
		assert.False(t, onik[0].Read)
		assert.Equal(t, notifFixedNow, onik[0].CreatedAt)

		alice := repo.notifsFor("u2")
		require.Len(t, alice, 1)
		assert.Equal(t, KindTicketMentioned, alice[0].Kind)
		assert.Equal(t, "evt-1:u2", alice[0].ID)

		assert.Equal(t, []string{TopicNotificationCreated}, repo.topics())
	})
	t.Run("unknown assignee and mentions are skipped", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{"id": "t-1", "title": "@ghost bug", "body": "@nobody", "assignee": "ghost"},
		}
		require.NoError(t, HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload)))
		assert.Empty(t, repo.notifsFor("u1"))
		assert.Empty(t, repo.topics())
	})
	t.Run("no recipients is a no-op", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97")))
		payload := map[string]any{"ticket": map[string]any{"id": "t-1", "title": "no recipients", "body": ""}}
		require.NoError(t, HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload)))
		assert.Empty(t, repo.notifsFor("u1"))
	})
	t.Run("repo failure propagates as retryable", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.createErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97")))
		payload := map[string]any{"ticket": map[string]any{"id": "t-1", "assignee": "onik97"}}
		err := HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload))
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
}

func TestHandleTicketStatusChanged(t *testing.T) {
	t.Run("fans out to assignee and mentions", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{"id": "t-1", "title": "Fix @alice", "assignee": "onik97"},
		}
		require.NoError(t, HandleTicketStatusChanged(context.Background(), s, notifEvFor(t, "ticket.status_changed", payload)))
		require.Len(t, repo.notifsFor("u1"), 1)
		assert.Equal(t, KindTicketStatus, repo.notifsFor("u1")[0].Kind)
		require.Len(t, repo.notifsFor("u2"), 1)
		assert.Equal(t, KindTicketStatus, repo.notifsFor("u2")[0].Kind)
	})
	t.Run("mention matching the assignee is not duplicated", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{"id": "t-1", "title": "@onik97 fix", "assignee": "onik97"},
		}
		require.NoError(t, HandleTicketStatusChanged(context.Background(), s, notifEvFor(t, "ticket.status_changed", payload)))
		require.Len(t, repo.notifsFor("u1"), 1)
	})
	t.Run("no assignee and no mentions is a no-op", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97")))
		payload := map[string]any{"ticket": map[string]any{"id": "t-1", "title": "x"}}
		require.NoError(t, HandleTicketStatusChanged(context.Background(), s, notifEvFor(t, "ticket.status_changed", payload)))
		assert.Empty(t, repo.notifsFor("u1"))
	})
}

func TestHandleDocCreatedAndUpdated(t *testing.T) {
	docPayload := map[string]any{"doc": map[string]any{"id": "d-1", "title": "Spec"}}

	t.Run("doc.created fans out to every member", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"))
		s := newTestNotifService(repo, users)
		require.NoError(t, HandleDocCreated(context.Background(), s, notifEvFor(t, "doc.created", docPayload)))
		require.Len(t, repo.notifsFor("u1"), 1)
		assert.Equal(t, KindDocCreated, repo.notifsFor("u1")[0].Kind)
		assert.Equal(t, SubjectDoc, repo.notifsFor("u1")[0].SubjectType)
		require.Len(t, repo.notifsFor("u2"), 1)
		assert.Equal(t, KindDocCreated, repo.notifsFor("u2")[0].Kind)
	})
	t.Run("doc.updated fans out to every member", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"))
		s := newTestNotifService(repo, users)
		require.NoError(t, HandleDocUpdated(context.Background(), s, notifEvFor(t, "doc.updated", docPayload)))
		require.Len(t, repo.notifsFor("u1"), 1)
		assert.Equal(t, KindDocUpdated, repo.notifsFor("u1")[0].Kind)
	})
	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := HandleDocCreated(context.Background(), s, eventbus.Event{ID: "e1", Payload: []byte(`{`)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("fan-out is idempotent per source event", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97")))
		ev := notifEvFor(t, "doc.created", docPayload)
		require.NoError(t, HandleDocCreated(context.Background(), s, ev))
		require.NoError(t, HandleDocCreated(context.Background(), s, ev))
		require.Len(t, repo.notifsFor("u1"), 1)
	})
	t.Run("user store failure propagates", func(t *testing.T) {
		users := newFakeNotifUsers(notifUser("u1", "onik97"))
		users.err = errors.New("db down")
		s := newTestNotifService(newFakeNotifRepo(), users)
		err := HandleDocCreated(context.Background(), s, notifEvFor(t, "doc.created", docPayload))
		require.Error(t, err)
		assert.ErrorIs(t, err, users.err)
	})
}

func TestExtractMentions(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"no mentions", "no tokens here", nil},
		{"single mention", "assign to @onik97", []string{"onik97"}},
		{"case-insensitive dedupe", "@Onik97 and @onik97", []string{"onik97"}},
		{"username with hyphen", "cc @jane-doe", []string{"jane-doe"}},
		{"no leading dash, trailing dash trimmed", "see @-x and @a-", []string{"a"}},
		{"email-like matches the local part", "mail me at a@b.com", []string{"b"}},
		{"mixed with duplicates", "@a @b @a @c", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractMentions(tt.in))
		})
	}
}

func TestHandlePlayRunFinished(t *testing.T) {
	payload := func(targetType, outcome string) map[string]any {
		return map[string]any{
			"trail_id": "tr-1", "play_id": "play-1", "play_label": "Fix with AI", "target_type": targetType,
			"target_id": "t-1", "target_title": "NEX-12", "starter_id": "u1", "via": "web", "outcome": outcome,
		}
	}
	tests := []struct {
		name       string
		targetType string
		outcome    string
		wantType   SubjectType
		wantTitle  string
	}{
		{"done on a ticket", "ticket", "done", SubjectTicket, "Fix with AI finished on NEX-12"},
		{"failed on a ticket", "ticket", "failed", SubjectTicket, "Fix with AI failed on NEX-12"},
		{"interrupted on a doc", "doc", "interrupted", SubjectDoc, "Fix with AI interrupted on NEX-12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeNotifRepo()
			s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice")))

			require.NoError(t, HandlePlayRunFinished(context.Background(), s, notifEvFor(t, "play.run_finished", payload(tt.targetType, tt.outcome))))

			require.Len(t, repo.notifsFor("u1"), 1, "exactly one notification for the starter")
			n := repo.notifsFor("u1")[0]
			assert.Equal(t, KindPlayRunFinished, n.Kind)
			assert.Equal(t, tt.wantType, n.SubjectType)
			assert.Equal(t, "t-1", n.SubjectID)
			assert.Equal(t, tt.wantTitle, n.SubjectTitle)
			assert.Empty(t, repo.notifsFor("u2"), "nobody but the starter is told")
		})
	}

	t.Run("missing starter is fatal", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := HandlePlayRunFinished(context.Background(), s, notifEvFor(t, "play.run_finished", map[string]any{"trail_id": "tr-1"}))
		require.ErrorIs(t, err, apperrs.ErrFatal)
	})
}

func TestHandlePlayRunWaiting(t *testing.T) {
	repo := newFakeNotifRepo()
	s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice")))
	payload := map[string]any{
		"trail_id": "tr-1", "play_id": "play-1", "play_label": "Fix with AI", "target_type": "ticket",
		"target_id": "t-1", "target_title": "NEX-12", "starter_id": "u1", "via": "web",
	}
	require.NoError(t, HandlePlayRunWaiting(context.Background(), s, notifEvFor(t, "play.run_waiting", payload)))

	require.Len(t, repo.notifsFor("u1"), 1)
	n := repo.notifsFor("u1")[0]
	assert.Equal(t, KindPlayRunWaiting, n.Kind)
	assert.Equal(t, SubjectTicket, n.SubjectType)
	assert.Equal(t, "Fix with AI needs your answer on NEX-12", n.SubjectTitle)
	assert.Empty(t, repo.notifsFor("u2"))

	err := HandlePlayRunWaiting(context.Background(), s, notifEvFor(t, "play.run_waiting", map[string]any{"trail_id": "tr-1"}))
	require.ErrorIs(t, err, apperrs.ErrFatal)
}
