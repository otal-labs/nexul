package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newTestNotification(id, userID string, read bool) *workspace.Notification {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	return &workspace.Notification{
		ID: id, UserID: userID, Kind: workspace.KindDocCreated,
		SubjectType: workspace.SubjectDoc, SubjectID: "d-1", SubjectTitle: "Spec",
		Read: read, CreatedAt: now,
	}
}

func mustCreateUser(t *testing.T, s *Store, id, login string) {
	t.Helper()
	_, _, err := s.Users.UpsertUser(context.Background(), &auth.User{
		ID: id, Provider: auth.ProviderGitHub, ProviderUserID: "p-" + id, Login: login,
	})
	require.NoError(t, err)
}

func TestNotificationsRepo_CreateMany_DuplicateID_IsNoOp(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))

	ns, err := s.Notifications.List(context.Background(), "u1", 50)
	require.NoError(t, err)
	require.Len(t, ns, 1)
}

func TestNotificationsRepo_CreateMany_WritesRowsAndOutboxInSameTx(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	mustCreateUser(t, s, "u2", "alice")
	ns := []*workspace.Notification{newTestNotification("n1", "u1", false), newTestNotification("n2", "u2", false)}
	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: workspace.TopicNotificationCreated, Payload: workspace.NotificationCreatedEvent{}}
	require.NoError(t, s.Notifications.CreateMany(context.Background(), ns, evt))

	got, err := s.Notifications.List(context.Background(), "u1", 50)
	require.NoError(t, err)
	require.Len(t, got, 1)

	var count int
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM outbox WHERE id = 'evt-1'`).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestNotificationsRepo_CreateMany_AllRowsConflict_SkipsOutbox(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	n := newTestNotification("n1", "u1", false)
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{n}))

	// Retried delivery of the same source event must not re-enqueue the outbox event.
	evt := eventbus.OutboxEvent{ID: "evt-2", Topic: workspace.TopicNotificationCreated, Payload: workspace.NotificationCreatedEvent{}}
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{n}, evt))

	var count int
	require.NoError(t, s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM outbox WHERE id = 'evt-2'`).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestNotificationsRepo_CreateMany_CollapsesWhileUnread(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")

	first := newTestNotification("c1", "u1", false)
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{first}))

	// Same subject + kind while c1 is unread: collapsed, no second inbox row
	// (collab commits fire doc.updated every few seconds mid-edit).
	repeat := newTestNotification("c2", "u1", false)
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{repeat}))
	ns, err := s.Notifications.List(context.Background(), "u1", 50)
	require.NoError(t, err)
	require.Len(t, ns, 1)

	// Reading the row re-arms the next notification for that subject.
	require.NoError(t, s.Notifications.MarkRead(context.Background(), "u1", "c1"))
	again := newTestNotification("c3", "u1", false)
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{again}))
	ns, err = s.Notifications.List(context.Background(), "u1", 50)
	require.NoError(t, err)
	require.Len(t, ns, 2)
}

func TestNotificationsRepo_List_ScopesByUserAndOrdersNewestFirst(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	mustCreateUser(t, s, "u2", "alice")
	n1 := newTestNotification("n1", "u1", false)
	n1.CreatedAt = time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	n3 := newTestNotification("n3", "u1", true)
	// A different subject: n1 is still unread for d-1, and the unread-collapse
	// guard would (correctly) suppress a second d-1 row.
	n3.SubjectID = "d-2"
	n3.CreatedAt = time.Date(2026, 8, 2, 11, 0, 0, 0, time.UTC)
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{n1}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n2", "u2", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{n3}))

	ns, err := s.Notifications.List(context.Background(), "u1", 50)
	require.NoError(t, err)
	require.Len(t, ns, 2)
	assert.Equal(t, "n3", ns[0].ID) // newest first
	assert.Equal(t, "n1", ns[1].ID)
	assert.True(t, ns[0].Read)
	assert.False(t, ns[1].Read)
}

func TestNotificationsRepo_List_RespectsLimit(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n2", "u1", false)}))

	ns, err := s.Notifications.List(context.Background(), "u1", 1)
	require.NoError(t, err)
	require.Len(t, ns, 1)
}

func TestNotificationsRepo_UnreadCount(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n2", "u1", true)}))

	n, err := s.Notifications.UnreadCount(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestNotificationsRepo_MarkRead_NotFoundForOtherUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	mustCreateUser(t, s, "u2", "alice")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))

	err := s.Notifications.MarkRead(context.Background(), "u2", "n1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	require.NoError(t, s.Notifications.MarkRead(context.Background(), "u1", "n1"))
	ns, err := s.Notifications.List(context.Background(), "u1", 50)
	require.NoError(t, err)
	assert.True(t, ns[0].Read)
}

func TestNotificationsRepo_MarkAllRead_ScopedToUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	mustCreateUser(t, s, "u2", "alice")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n2", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n3", "u2", false)}))

	require.NoError(t, s.Notifications.MarkAllRead(context.Background(), "u1"))

	u1, err := s.Notifications.List(context.Background(), "u1", 50)
	require.NoError(t, err)
	for _, n := range u1 {
		assert.True(t, n.Read)
	}
	u2, err := s.Notifications.List(context.Background(), "u2", 50)
	require.NoError(t, err)
	for _, n := range u2 {
		assert.False(t, n.Read)
	}
}
