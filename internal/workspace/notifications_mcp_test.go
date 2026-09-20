package workspace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestNotificationMCPTools_List(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.create(t, mkNotif("n1", "u1"))
	s := newTestNotifService(repo, newFakeNotifUsers())

	tools := NotificationMCPTools(s)
	byName := map[string]mcptool.Tool{}
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	require.Contains(t, byName, "notification_list")

	t.Run("missing user_id is invalid", func(t *testing.T) {
		_, err := byName["notification_list"].Call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
	t.Run("lists the user's notifications", func(t *testing.T) {
		out, err := byName["notification_list"].Call(context.Background(), map[string]any{"user_id": "u1"})
		require.NoError(t, err)
		ns, ok := out.([]*Notification)
		require.True(t, ok)
		require.Len(t, ns, 1)
		assert.Equal(t, "n1", ns[0].ID)
	})
}

func TestNotificationMCPTools_MarkRead(t *testing.T) {
	repo := newFakeNotifRepo()
	repo.create(t, mkNotif("n1", "u1"))
	s := newTestNotifService(repo, newFakeNotifUsers())

	tools := NotificationMCPTools(s)
	byName := map[string]mcptool.Tool{}
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	require.Contains(t, byName, "notification_mark_read")

	t.Run("missing args are invalid", func(t *testing.T) {
		_, err := byName["notification_mark_read"].Call(context.Background(), map[string]any{"user_id": "u1"})
		require.Error(t, err)
		_, err = byName["notification_mark_read"].Call(context.Background(), map[string]any{"id": "n1"})
		require.Error(t, err)
	})
	t.Run("marks read", func(t *testing.T) {
		out, err := byName["notification_mark_read"].Call(context.Background(), map[string]any{"user_id": "u1", "id": "n1"})
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

	tools := NotificationMCPTools(s)
	byName := map[string]mcptool.Tool{}
	for _, tool := range tools {
		byName[tool.Name] = tool
	}
	require.Contains(t, byName, "notification_mark_all_read")

	t.Run("missing user_id is invalid", func(t *testing.T) {
		_, err := byName["notification_mark_all_read"].Call(context.Background(), map[string]any{})
		require.Error(t, err)
	})
	t.Run("marks all read", func(t *testing.T) {
		out, err := byName["notification_mark_all_read"].Call(context.Background(), map[string]any{"user_id": "u1"})
		require.NoError(t, err)
		assert.NotNil(t, out)
		ns, err := s.List(context.Background(), "u1", 0)
		require.NoError(t, err)
		for _, n := range ns {
			assert.True(t, n.Read)
		}
	})
}
