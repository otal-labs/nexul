package workspace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func notificationTools(s *NotificationService) map[string]mcptool.Tool {
	byName := map[string]mcptool.Tool{}
	for _, tool := range NotificationMCPTools(s) {
		byName[tool.Name] = tool
	}
	return byName
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func TestNotificationMCPTools_NoActor_IsUnauthorized(t *testing.T) {
	tools := notificationTools(newTestNotifService(newFakeNotifRepo(), newFakeNotifUsers()))
	for _, name := range []string{"notification_list", "notification_mark_read", "notification_mark_all_read"} {
		t.Run(name, func(t *testing.T) {
			_, err := tools[name].Call(context.Background(), map[string]any{"id": "n1"})
			require.ErrorIs(t, err, apperrs.ErrUnauthorized)
		})
	}
}

func TestNotificationMCPTools_List(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.create(t, mkNotif("n1", "u1"))
	tools := notificationTools(newTestNotifService(repo, newFakeNotifUsers()))

	t.Run("lists the caller's notifications", func(t *testing.T) {
		out, err := tools["notification_list"].Call(as("u1"), map[string]any{})
		require.NoError(t, err)
		ns, ok := out.([]*Notification)
		require.True(t, ok)
		require.Len(t, ns, 1)
		assert.Equal(t, "n1", ns[0].ID)
	})
	t.Run("a user_id argument cannot reach another inbox", func(t *testing.T) {
		out, err := tools["notification_list"].Call(as("u2"), map[string]any{"user_id": "u1"})
		require.NoError(t, err)
		assert.Empty(t, out)
	})
}

func TestNotificationMCPTools_MarkRead(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.create(t, mkNotif("n1", "u1"))
	s := newTestNotifService(repo, newFakeNotifUsers())
	tools := notificationTools(s)

	t.Run("missing id is invalid", func(t *testing.T) {
		_, err := tools["notification_mark_read"].Call(as("u1"), map[string]any{})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("another user cannot mark it read", func(t *testing.T) {
		_, _ = tools["notification_mark_read"].Call(as("u2"), map[string]any{"id": "n1"})
		ns, err := s.List(context.Background(), "u1", 0)
		require.NoError(t, err)
		assert.False(t, ns[0].Read)
	})
	t.Run("marks the caller's notification read", func(t *testing.T) {
		out, err := tools["notification_mark_read"].Call(as("u1"), map[string]any{"id": "n1"})
		require.NoError(t, err)
		status, ok := out.(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "read", status["status"])
		ns, err := s.List(context.Background(), "u1", 0)
		require.NoError(t, err)
		assert.True(t, ns[0].Read)
	})
}

func TestNotificationMCPTools_MarkAllRead(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.create(t, mkNotif("n1", "u1"))
	repo.create(t, mkNotif("n2", "u1"))
	s := newTestNotifService(repo, newFakeNotifUsers())
	tools := notificationTools(s)

	_, err := tools["notification_mark_all_read"].Call(as("u2"), map[string]any{})
	require.NoError(t, err)
	ns, err := s.List(context.Background(), "u1", 0)
	require.NoError(t, err)
	for _, n := range ns {
		assert.False(t, n.Read, "another user's mark-all leaves this inbox alone")
	}

	out, err := tools["notification_mark_all_read"].Call(as("u1"), map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, out)
	ns, err = s.List(context.Background(), "u1", 0)
	require.NoError(t, err)
	for _, n := range ns {
		assert.True(t, n.Read)
	}
}
