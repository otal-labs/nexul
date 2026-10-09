package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
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
	deleteErr error
	cutoffs   [][2]time.Time
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

// pushItems returns the rows named by every notification.push_requested event enqueued so far.
func (f *fakeNotifRepo) pushItems() []NotificationPushItem {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []NotificationPushItem
	for _, evt := range f.outbox {
		if e, ok := evt.Payload.(NotificationPushRequestedEvent); ok {
			out = append(out, e.Notifications...)
		}
	}
	return out
}

// createdEvents returns every notification.created payload enqueued so far.
func (f *fakeNotifRepo) createdEvents() []NotificationCreatedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []NotificationCreatedEvent
	for _, evt := range f.outbox {
		if e, ok := evt.Payload.(NotificationCreatedEvent); ok {
			out = append(out, e)
		}
	}
	return out
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

// inWorkspace mirrors the storage filter: an empty workspaceID spans every workspace.
func inWorkspace(n *Notification, workspaceID string) bool {
	return workspaceID == "" || n.WorkspaceID == workspaceID
}

func (f *fakeNotifRepo) List(_ context.Context, userID, workspaceID string, limit int) ([]*Notification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var all []*Notification
	for _, n := range f.byUser[userID] {
		if inWorkspace(n, workspaceID) {
			all = append(all, n)
		}
	}
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (f *fakeNotifRepo) UnreadByProject(_ context.Context, userID, workspaceID string) ([]UnreadGroup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.countErr != nil {
		return nil, f.countErr
	}
	var out []UnreadGroup
	for _, notif := range f.byUser[userID] {
		if !notif.Read && inWorkspace(notif, workspaceID) {
			out = append(out, UnreadGroup{WorkspaceID: notif.WorkspaceID, ProjectID: notif.ProjectID, Unread: 1})
		}
	}
	return out, nil
}

func (f *fakeNotifRepo) MarkRead(_ context.Context, userID, id string, _ time.Time) error {
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

func (f *fakeNotifRepo) MarkAllRead(_ context.Context, userID, workspaceID string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.markErr != nil {
		return f.markErr
	}
	for _, n := range f.byUser[userID] {
		if inWorkspace(n, workspaceID) {
			n.Read = true
		}
	}
	return nil
}

// DeleteExpired records each pass's cutoffs (read before, created before); deleteErr fails one pass, then clears.
func (f *fakeNotifRepo) DeleteExpired(_ context.Context, readBefore, createdBefore time.Time) (int64, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cutoffs = append(f.cutoffs, [2]time.Time{readBefore, createdBefore})
	if err := f.deleteErr; err != nil {
		f.deleteErr = nil
		return 0, 0, err
	}
	return 1, 1, nil
}

func (f *fakeNotifRepo) cleanupPasses() [][2]time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][2]time.Time(nil), f.cutoffs...)
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

func (f *fakeNotifUsers) NameForUserID(ctx context.Context, userID string) (string, error) {
	return f.LoginForUserID(ctx, userID)
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
	err         error
}

func (f *fakeMemberStore) ListMemberUserIDs(_ context.Context, workspaceID string) ([]string, error) {
	return f.byWorkspace[workspaceID], f.err
}

// fakeProjects is a ProjectReader stub mapping project id to its workspace.
type fakeProjects struct {
	workspaceOf map[string]string
	err         error
}

func (f *fakeProjects) Get(_ context.Context, id string) (*Project, error) {
	if f.err != nil {
		return nil, f.err
	}
	ws, ok := f.workspaceOf[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &Project{ID: id, WorkspaceID: ws}, nil
}

// fakeAccessChecker is a PermissionChecker stub gating memory.updated fan-out by a fixed allow/deny set.
type fakeAccessChecker struct {
	denyUserIDs    map[string]bool
	denyInProject  map[string]bool // "user:project"
	inProjectCalls int
}

func (f *fakeAccessChecker) CanInProject(_ context.Context, userID, projectID string, _ permissions.Action) bool {
	f.inProjectCalls++
	return !f.denyUserIDs[userID] && !f.denyInProject[userID+":"+projectID]
}

func (f *fakeAccessChecker) HasPermission(_ context.Context, userID, _ string, _ permissions.Action) bool {
	return !f.denyUserIDs[userID]
}

func (f *fakeAccessChecker) CanReadDoc(_ context.Context, userID, _ string) bool {
	return !f.denyUserIDs[userID]
}

func newTestNotifService(repo *fakeNotifRepo, users *fakeNotifUsers) *NotificationService {
	return newTestNotifServiceWith(repo, users, nil, &fakeAccessChecker{})
}

func newTestNotifServiceWith(repo *fakeNotifRepo, users *fakeNotifUsers, members WorkspaceMemberStore, access PermissionChecker) *NotificationService {
	s := NewNotificationService(repo, users, members, access, &fakeProjects{workspaceOf: map[string]string{"p-1": "ws-1"}})
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
		_, err := s.List(context.Background(), "  ", "", 10)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.listErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers())
		_, err := s.List(context.Background(), "u1", "", 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listErr)
	})
	t.Run("defaults limit to 50", func(t *testing.T) {
		repo := newFakeNotifRepo()
		for i := 0; i < 60; i++ {
			repo.create(t, &Notification{ID: notifID(i), UserID: "u1"})
		}
		s := newTestNotifService(repo, newFakeNotifUsers())
		ns, err := s.List(context.Background(), "u1", "", 0)
		require.NoError(t, err)
		require.Len(t, ns, 50)
	})
	t.Run("limits rows", func(t *testing.T) {
		repo := newFakeNotifRepo()
		for i := 0; i < 5; i++ {
			repo.create(t, &Notification{ID: notifID(i), UserID: "u1"})
		}
		s := newTestNotifService(repo, newFakeNotifUsers())
		ns, err := s.List(context.Background(), "u1", "", 2)
		require.NoError(t, err)
		require.Len(t, ns, 2)
	})
}

func TestNotifList_ChecksEachProjectOnce(t *testing.T) {
	repo := newFakeNotifRepo()
	for i := range 20 {
		repo.create(t, &Notification{ID: notifID(i), UserID: "u1", ProjectID: []string{"p-open", "p-closed"}[i%2]})
	}
	access := &fakeAccessChecker{denyInProject: map[string]bool{"u1:p-closed": true}}
	s := newTestNotifServiceWith(repo, newFakeNotifUsers(), nil, access)

	ns, err := s.List(t.Context(), "u1", "", 50)

	require.NoError(t, err)
	assert.Len(t, ns, 10)
	assert.Equal(t, 2, access.inProjectCalls)
}

func TestNotifUnreadCount(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		_, err := s.UnreadCount(context.Background(), "", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.countErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers())
		_, err := s.UnreadCount(context.Background(), "u1", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.countErr)
	})
	t.Run("counts only unread", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, &Notification{ID: "n1", UserID: "u1"})
		repo.create(t, &Notification{ID: "n2", UserID: "u1"})
		require.NoError(t, repo.MarkRead(context.Background(), "u1", "n1", notifFixedNow))
		s := newTestNotifService(repo, newFakeNotifUsers())
		n, err := s.UnreadCount(context.Background(), "u1", "")
		require.NoError(t, err)
		assert.Equal(t, 1, n)
	})
}

func TestNotifUnreadByWorkspace(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		_, err := s.UnreadByWorkspace(context.Background(), "", "")
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("counts each workspace apart and leaves fully read ones out", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, &Notification{ID: "n1", UserID: "u1", WorkspaceID: "ws-1"})
		repo.create(t, &Notification{ID: "n2", UserID: "u1", WorkspaceID: "ws-1"})
		repo.create(t, &Notification{ID: "n3", UserID: "u1", WorkspaceID: "ws-2"})
		repo.create(t, &Notification{ID: "n4", UserID: "u1", WorkspaceID: "ws-3"})
		require.NoError(t, repo.MarkRead(context.Background(), "u1", "n4", notifFixedNow))
		s := newTestNotifService(repo, newFakeNotifUsers())
		got, err := s.UnreadByWorkspace(context.Background(), "u1", "")
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"ws-1": 2, "ws-2": 1}, got)
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
		got, err := s.List(context.Background(), "u1", "", 0)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.True(t, got[0].Read)
	})
}

func TestNotifMarkAllRead(t *testing.T) {
	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := s.MarkAllRead(context.Background(), "", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.markErr = errors.New("db down")
		s := newTestNotifService(repo, newFakeNotifUsers())
		err := s.MarkAllRead(context.Background(), "u1", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.markErr)
	})
	t.Run("marks all read for the user only", func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.create(t, &Notification{ID: "n1", UserID: "u1"})
		repo.create(t, &Notification{ID: "n2", UserID: "u1"})
		repo.create(t, &Notification{ID: "n3", UserID: "u2"})
		s := newTestNotifService(repo, newFakeNotifUsers())
		require.NoError(t, s.MarkAllRead(context.Background(), "u1", ""))
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

	t.Run("skips a member who may not read memories in the memory's project", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"))
		members := &fakeMemberStore{byWorkspace: map[string][]string{"ws-1": {"u1", "u2"}}}
		access := &fakeAccessChecker{denyInProject: map[string]bool{"u2:p-2": true}}
		s := newTestNotifServiceWith(repo, users, members, access)
		hidden := map[string]any{
			"memory":    map[string]any{"id": "m-2", "workspace_id": "ws-1", "project_id": "p-2", "title": "Hidden", "version": 1},
			"author_id": "u1",
		}

		require.NoError(t, HandleMemoryUpdated(context.Background(), s, notifEvFor(t, "memory.updated", hidden)))

		assert.Empty(t, repo.notifsFor("u2"))
	})

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
		assert.Equal(t, []NotificationPushItem{{ID: n.ID, UserID: "u2", WorkspaceID: "ws-1"}}, repo.pushItems(), "the push request names the memory's workspace")
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
		s := NewNotificationService(newFakeNotifRepo(), newFakeNotifUsers(), nil, nil, nil)
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
	t.Run("fans out to developer and mentions", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"), notifUser("u3", "lena"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{
				"id": "t-1", "title": "Fix @alice bug", "body": "@onik97 please triage", "developer": "onik97", "tester": "lena",
			},
		}
		require.NoError(t, HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload)))

		onik := repo.notifsFor("u1")
		require.Len(t, repo.notifsFor("u3"), 1)
		assert.Equal(t, KindTicketAssigned, repo.notifsFor("u3")[0].Kind)
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

		assert.Equal(t, []string{TopicNotificationCreated, TopicNotificationPushRequested}, repo.topics())
		assert.ElementsMatch(t, []NotificationPushItem{{ID: "evt-1:u1", UserID: "u1"}, {ID: "evt-1:u2", UserID: "u2"}, {ID: "evt-1:u3", UserID: "u3"}}, repo.pushItems())
	})
	t.Run("unknown people and mentions are skipped", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{"id": "t-1", "title": "@ghost bug", "body": "@nobody", "developer": "ghost", "tester": "phantom"},
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
		payload := map[string]any{"ticket": map[string]any{"id": "t-1", "developer": "onik97"}}
		err := HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload))
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
}

func TestHandleTicketStatusChanged(t *testing.T) {
	t.Run("fans out to developer and mentions", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{"id": "t-1", "title": "Fix @alice", "developer": "onik97"},
		}
		require.NoError(t, HandleTicketStatusChanged(context.Background(), s, notifEvFor(t, "ticket.status_changed", payload)))
		require.Len(t, repo.notifsFor("u1"), 1)
		assert.Equal(t, KindTicketStatus, repo.notifsFor("u1")[0].Kind)
		require.Len(t, repo.notifsFor("u2"), 1)
		assert.Equal(t, KindTicketStatus, repo.notifsFor("u2")[0].Kind)
	})
	t.Run("mention matching the developer or tester is not duplicated", func(t *testing.T) {
		repo := newFakeNotifRepo()
		users := newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"))
		s := newTestNotifService(repo, users)
		payload := map[string]any{
			"ticket": map[string]any{"id": "t-1", "title": "@onik97 @alice fix", "developer": "onik97", "tester": "alice"},
		}
		require.NoError(t, HandleTicketStatusChanged(context.Background(), s, notifEvFor(t, "ticket.status_changed", payload)))
		require.Len(t, repo.notifsFor("u1"), 1)
		require.Len(t, repo.notifsFor("u2"), 1)
	})
	t.Run("no people and no mentions is a no-op", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97")))
		payload := map[string]any{"ticket": map[string]any{"id": "t-1", "title": "x"}}
		require.NoError(t, HandleTicketStatusChanged(context.Background(), s, notifEvFor(t, "ticket.status_changed", payload)))
		assert.Empty(t, repo.notifsFor("u1"))
	})
}

// fakeDocWatchers maps a doc id to its watchers' user ids.
type fakeDocWatchers struct {
	byDoc map[string][]string
	err   error
}

func (f *fakeDocWatchers) ListDocWatcherIDs(_ context.Context, docID string) ([]string, error) {
	return f.byDoc[docID], f.err
}

// TestHandleDocCreatedAndUpdated covers the handlers' failure and redelivery paths; who is told is checked end to end
// over real storage in server/cmd's doc watcher tests.
func TestHandleDocCreatedAndUpdated(t *testing.T) {
	docPayload := map[string]any{"doc": map[string]any{"id": "d-1", "title": "Spec", "project_id": "p-1"}}
	users := func() *fakeNotifUsers { return newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice")) }
	wsMembers := func() *fakeMemberStore {
		return &fakeMemberStore{byWorkspace: map[string][]string{"ws-1": {"u1", "u2"}}}
	}
	watching := func(ids ...string) *fakeDocWatchers { return &fakeDocWatchers{byDoc: map[string][]string{"d-1": ids}} }

	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers())
		err := HandleDocCreated(context.Background(), s, eventbus.Event{ID: "e1", Payload: []byte(`{`)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("fan-out is idempotent per source event", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifServiceWith(repo, users(), wsMembers(), &fakeAccessChecker{}).WithDocWatchers(watching("u1"))
		ev := notifEvFor(t, "doc.updated", docPayload)
		require.NoError(t, HandleDocUpdated(context.Background(), s, ev))
		require.NoError(t, HandleDocUpdated(context.Background(), s, ev))
		require.Len(t, repo.notifsFor("u1"), 1)
	})
	t.Run("a lock or unlock tells no watcher, the page flips through its live push", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifServiceWith(repo, users(), wsMembers(), &fakeAccessChecker{}).WithDocWatchers(watching("u1"))
		locked := map[string]any{"doc": docPayload["doc"], "actor_id": "u2", "lock_changed": true}
		require.NoError(t, HandleDocUpdated(context.Background(), s, notifEvFor(t, "doc.updated", locked)))
		assert.Empty(t, repo.notifsFor("u1"))
	})
	t.Run("watcher lookup failure propagates so the bus retries", func(t *testing.T) {
		watchers := watching("u1")
		watchers.err = errors.New("db down")
		s := newTestNotifServiceWith(newFakeNotifRepo(), users(), wsMembers(), &fakeAccessChecker{}).WithDocWatchers(watchers)
		err := HandleDocUpdated(context.Background(), s, notifEvFor(t, "doc.updated", docPayload))
		assert.ErrorIs(t, err, watchers.err)
	})
	t.Run("member store failure propagates when the save mentions someone", func(t *testing.T) {
		members := wsMembers()
		members.err = errors.New("db down")
		s := newTestNotifServiceWith(newFakeNotifRepo(), users(), members, &fakeAccessChecker{})
		mentioning := map[string]any{"doc": docPayload["doc"], "mentioned_user_ids": []string{"u2"}}
		err := HandleDocCreated(context.Background(), s, notifEvFor(t, "doc.created", mentioning))
		assert.ErrorIs(t, err, members.err)
	})
}

func TestFanOut_StampsTheSubjectsWorkspace(t *testing.T) {
	t.Run("project lookup failure propagates so the bus retries", func(t *testing.T) {
		repo := newFakeNotifRepo()
		lookupErr := errors.New("db down")
		s := NewNotificationService(repo, newFakeNotifUsers(notifUser("u1", "onik97")), &fakeMemberStore{}, &fakeAccessChecker{}, &fakeProjects{err: lookupErr})
		payload := map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": "p-1", "developer": "onik97"}}
		err := HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload))
		require.ErrorIs(t, err, lookupErr)
		assert.Empty(t, repo.notifsFor("u1"))
	})
	t.Run("a project deleted since the event leaves the notification unscoped", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u1", "onik97")))
		payload := map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": "p-gone", "developer": "onik97"}}
		require.NoError(t, HandleTicketCreated(context.Background(), s, notifEvFor(t, "ticket.created", payload)))
		require.Len(t, repo.notifsFor("u1"), 1)
		assert.Empty(t, repo.notifsFor("u1")[0].WorkspaceID)
	})

	tests := []struct {
		name    string
		handle  func(context.Context, *NotificationService, eventbus.Event) error
		topic   string
		body    map[string]any
		project string
	}{
		{"ticket.created", HandleTicketCreated, "ticket.created",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": "p-1", "developer": "alice"}}, "p-1"},
		{"ticket.status_changed", HandleTicketStatusChanged, "ticket.status_changed",
			map[string]any{"ticket": map[string]any{"id": "t-1", "project_id": "p-1", "developer": "alice"}}, "p-1"},
		{"play.run_finished", HandlePlayRunFinished, "play.run_finished",
			map[string]any{"trail_id": "tr-1", "target_type": "ticket", "target_id": "t-1", "starter_id": "u2", "workspace_id": "ws-1", "outcome": "done"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeNotifRepo()
			s := newTestNotifService(repo, newFakeNotifUsers(notifUser("u2", "alice")))
			require.NoError(t, tt.handle(context.Background(), s, notifEvFor(t, tt.topic, tt.body)))
			require.Len(t, repo.notifsFor("u2"), 1)
			assert.Equal(t, "ws-1", repo.notifsFor("u2")[0].WorkspaceID)
			assert.Equal(t, []NotificationPushItem{{ID: "evt-1:u2", UserID: "u2", WorkspaceID: "ws-1"}}, repo.pushItems())
			assert.Equal(t, []NotificationCreatedEvent{{UserIDs: []string{"u2"}, WorkspaceID: "ws-1", ProjectID: tt.project}}, repo.createdEvents(), "the live frame names its recipients and place")
		})
	}
}

func TestFanOut_TheActorIsNeverNotifiedOfTheirOwnAction(t *testing.T) {
	ticket := map[string]any{"id": "t-1", "project_id": "p-1", "title": "Fix @alice", "developer": "onik97", "tester": "alice"}
	doc := map[string]any{"id": "d-1", "title": "Spec", "project_id": "p-1"}
	selfFiled := maps.Clone(ticket)
	selfFiled["reporter"] = map[string]any{"kind": "user", "login": "onik97"}
	tests := []struct {
		name   string
		handle func(context.Context, *NotificationService, eventbus.Event) error
		topic  string
		body   map[string]any
	}{
		{"ticket.created by its developer", HandleTicketCreated, "ticket.created",
			map[string]any{"ticket": selfFiled}},
		{"ticket.status_changed moved by its developer", HandleTicketStatusChanged, "ticket.status_changed",
			map[string]any{"ticket": ticket, "actor": map[string]any{"kind": "user", "user_id": "u1"}}},
		{"doc.updated by a watcher", HandleDocUpdated, "doc.updated", map[string]any{"doc": doc, "actor_id": "u1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeNotifRepo()
			members := &fakeMemberStore{byWorkspace: map[string][]string{"ws-1": {"u1", "u2"}}}
			s := newTestNotifServiceWith(repo, newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice")), members, &fakeAccessChecker{}).
				WithDocWatchers(&fakeDocWatchers{byDoc: map[string][]string{"d-1": {"u1", "u2"}}})
			require.NoError(t, tt.handle(context.Background(), s, notifEvFor(t, tt.topic, tt.body)))
			assert.Empty(t, repo.notifsFor("u1"), "the actor is not notified")
			assert.Len(t, repo.notifsFor("u2"), 1, "everyone else still is")
			for _, item := range repo.pushItems() {
				assert.NotEqual(t, "u1", item.UserID, "and never pushed to")
			}
		})
	}
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

func TestNotificationRunCleanupLoop_RunsAMinuteAfterStartThenDaily_SurvivingAFailedPass(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := newFakeNotifRepo()
		repo.deleteErr = errors.New("database is locked")
		s := NewNotificationService(repo, nil, nil, nil, nil)
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		start := time.Now()
		go func() {
			s.RunCleanupLoop(ctx, slog.New(slog.DiscardHandler))
			close(done)
		}()

		synctest.Wait()
		assert.Empty(t, repo.cleanupPasses(), "nothing is deleted while the server is still starting")

		time.Sleep(time.Minute)
		synctest.Wait()
		passes := repo.cleanupPasses()
		require.Len(t, passes, 1, "the first pass runs shortly after start, not a day later")
		firstRun := start.Add(time.Minute)
		assert.Equal(t, firstRun.Add(-90*24*time.Hour), passes[0][0], "read notifications are kept 90 days after they were read")
		assert.Equal(t, firstRun.Add(-180*24*time.Hour), passes[0][1], "every notification is kept 180 days after it was sent")

		time.Sleep(24 * time.Hour)
		synctest.Wait()
		assert.Len(t, repo.cleanupPasses(), 2, "a failed pass does not stop the next day's")

		cancel()
		<-done
	})
}

func TestHandleDocClarificationNotices(t *testing.T) {
	round := func(extra map[string]any) map[string]any {
		out := map[string]any{"doc": map[string]any{"id": "d-1", "title": "Spec", "project_id": "p-1"}, "round": 2, "started_by": "u1", "actor_id": "u1"}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	users := func() *fakeNotifUsers {
		return newFakeNotifUsers(notifUser("u1", "onik97"), notifUser("u2", "alice"), notifUser("u3", "bob"))
	}
	service := func(repo *fakeNotifRepo, access *fakeAccessChecker, watching ...string) *NotificationService {
		return newTestNotifServiceWith(repo, users(), nil, access).WithDocWatchers(&fakeDocWatchers{byDoc: map[string][]string{"d-1": watching}})
	}

	t.Run("malformed payloads are fatal", func(t *testing.T) {
		s := service(newFakeNotifRepo(), &fakeAccessChecker{})
		bad := eventbus.Event{ID: "e1", Payload: []byte(`{`)}
		assert.ErrorIs(t, HandleDocQuestionsPosted(context.Background(), s, bad), apperrs.ErrFatal)
		assert.ErrorIs(t, HandleDocRoundAnswered(context.Background(), s, bad), apperrs.ErrFatal)
		assert.ErrorIs(t, HandleDocRoundAnswered(context.Background(), s, notifEvFor(t, "doc.clarification.round_answered", map[string]any{"doc": map[string]any{"id": "d-1"}})), apperrs.ErrFatal, "no starter")
	})
	t.Run("a round that found no gaps or asked nothing tells nobody", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := service(repo, &fakeAccessChecker{}, "u2")
		require.NoError(t, HandleDocQuestionsPosted(context.Background(), s, notifEvFor(t, "doc.clarification.round_posted", round(map[string]any{"no_gaps": true}))))
		require.NoError(t, HandleDocQuestionsPosted(context.Background(), s, notifEvFor(t, "doc.clarification.round_posted", round(map[string]any{"question_count": 0}))))
		assert.Empty(t, repo.notifsFor("u2"))
	})
	t.Run("the starter is never told of their own round, though they watch the doc", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := service(repo, &fakeAccessChecker{}, "u1", "u2")
		require.NoError(t, HandleDocQuestionsPosted(context.Background(), s, notifEvFor(t, "doc.clarification.round_posted", round(map[string]any{"question_count": 3}))))
		assert.Empty(t, repo.notifsFor("u1"))
		assert.Len(t, repo.notifsFor("u2"), 1)
	})
	t.Run("a watcher who cannot open the doc gets nothing", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := service(repo, &fakeAccessChecker{denyUserIDs: map[string]bool{"u3": true}}, "u2", "u3")
		require.NoError(t, HandleDocQuestionsPosted(context.Background(), s, notifEvFor(t, "doc.clarification.round_posted", round(map[string]any{"question_count": 1}))))
		assert.Empty(t, repo.notifsFor("u3"))
		got := repo.notifsFor("u2")
		require.Len(t, got, 1)
		assert.Equal(t, KindDocQuestionsAsked, got[0].Kind)
		assert.Equal(t, SubjectDoc, got[0].SubjectType)
		assert.Equal(t, "d-1", got[0].SubjectID)
		assert.Equal(t, "New questions on Spec", got[0].SubjectTitle)
	})
	t.Run("watcher lookup failure propagates so the bus retries", func(t *testing.T) {
		watchers := &fakeDocWatchers{err: errors.New("db down")}
		s := newTestNotifServiceWith(newFakeNotifRepo(), users(), nil, &fakeAccessChecker{}).WithDocWatchers(watchers)
		err := HandleDocQuestionsPosted(context.Background(), s, notifEvFor(t, "doc.clarification.round_posted", round(map[string]any{"question_count": 1})))
		assert.ErrorIs(t, err, watchers.err)
	})
	t.Run("the starter hears the last answer, once per event", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := service(repo, &fakeAccessChecker{}, "u2")
		ev := notifEvFor(t, "doc.clarification.round_answered", round(map[string]any{"actor_id": "u2"}))
		require.NoError(t, HandleDocRoundAnswered(context.Background(), s, ev))
		require.NoError(t, HandleDocRoundAnswered(context.Background(), s, ev))
		got := repo.notifsFor("u1")
		require.Len(t, got, 1)
		assert.Equal(t, KindDocQuestionsAnswered, got[0].Kind)
		assert.Equal(t, "Questions answered on Spec", got[0].SubjectTitle)
		assert.Empty(t, repo.notifsFor("u2"), "watchers are not told an answer landed")
	})
	t.Run("the starter answering their own round is not told", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := service(repo, &fakeAccessChecker{})
		require.NoError(t, HandleDocRoundAnswered(context.Background(), s, notifEvFor(t, "doc.clarification.round_answered", round(nil))))
		assert.Empty(t, repo.notifsFor("u1"))
	})
	t.Run("a starter who can no longer open the doc is not told", func(t *testing.T) {
		repo := newFakeNotifRepo()
		s := service(repo, &fakeAccessChecker{denyUserIDs: map[string]bool{"u1": true}})
		require.NoError(t, HandleDocRoundAnswered(context.Background(), s, notifEvFor(t, "doc.clarification.round_answered", round(map[string]any{"actor_id": "u2"}))))
		assert.Empty(t, repo.notifsFor("u1"))
	})
}
