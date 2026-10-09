package workspace

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

func callNotificationTool(ctx context.Context, t *testing.T, s *NotificationService, name, args string) (any, error) {
	t.Helper()
	for _, tool := range NotificationMCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func notifAs(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

// seededInbox gives u1 two unread notifications, one per workspace, and u2 one.
func seededInbox(t *testing.T) *NotificationService {
	t.Helper()
	repo := newFakeNotifRepo()
	n1, n2 := mkNotif("n1", "u1"), mkNotif("n2", "u1")
	n1.WorkspaceID, n2.WorkspaceID = "ws-1", "ws-2"
	n2.FolderID, n2.FolderName = "f-gs", "GetSource"
	repo.create(t, n1)
	repo.create(t, n2)
	repo.create(t, mkNotif("n3", "u2"))
	return newTestNotifService(repo, newFakeNotifUsers())
}

func readState(t *testing.T, s *NotificationService, userID string) map[string]bool {
	t.Helper()
	ns, _, err := s.Page(context.Background(), userID, InboxFilter{}, paging.Window{})
	require.NoError(t, err)
	out := map[string]bool{}
	for _, n := range ns {
		out[n.ID] = n.Read
	}
	return out
}

func TestNotificationMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range NotificationMCPTools(seededInbox(t)) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.True(t, tool.Hints.Local, tool.Name)
	}
	assert.Equal(t, []string{"notification_list", "notification_update"}, names)
}

func TestNotificationMCPTools_Errors(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		tool    string
		args    string
		wantErr error
	}{
		{"list without a caller", context.Background(), "notification_list", `{}`, apperrs.ErrUnauthorized},
		{"update without a caller", context.Background(), "notification_update", `{"all":true}`, apperrs.ErrUnauthorized},
		{"list naming another user's inbox", notifAs("u2"), "notification_list", `{"user_id":"u1"}`, apperrs.ErrInvalid},
		{"update with neither id nor all", notifAs("u1"), "notification_update", `{}`, apperrs.ErrInvalid},
		{"update with all false", notifAs("u1"), "notification_update", `{"all":false}`, apperrs.ErrInvalid},
		{"update with both id and all", notifAs("u1"), "notification_update", `{"id":"n1","all":true}`, apperrs.ErrInvalid},
		{"update a missing notification", notifAs("u1"), "notification_update", `{"id":"nope"}`, apperrs.ErrNotFound},
		{"update another user's notification", notifAs("u2"), "notification_update", `{"id":"n1"}`, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callNotificationTool(tt.ctx, t, seededInbox(t), tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNotificationUpdate_OneIDLeavesTheRestAsTheyWere(t *testing.T) {
	s := seededInbox(t)
	out, err := callNotificationTool(notifAs("u1"), t, s, "notification_update", `{"id":"n1"}`)
	require.NoError(t, err)
	assert.Equal(t, notificationUpdated{ID: "n1", Read: true}, out)
	assert.Equal(t, map[string]bool{"n1": true, "n2": false}, readState(t, s, "u1"))
	assert.Equal(t, map[string]bool{"n3": false}, readState(t, s, "u2"))
}

func TestNotificationUpdate_AllTouchesOnlyTheCallersInbox(t *testing.T) {
	s := seededInbox(t)
	out, err := callNotificationTool(notifAs("u1"), t, s, "notification_update", `{"all":true}`)
	require.NoError(t, err)
	assert.Equal(t, notificationUpdated{All: true, Read: true}, out)
	assert.Equal(t, map[string]bool{"n1": true, "n2": true}, readState(t, s, "u1"))
	assert.Equal(t, map[string]bool{"n3": false}, readState(t, s, "u2"))
}

func TestNotificationUpdate_AllWithAWorkspaceLeavesOtherWorkspacesUnread(t *testing.T) {
	s := seededInbox(t)
	_, err := callNotificationTool(notifAs("u1"), t, s, "notification_update", `{"all":true,"workspace_id":"ws-1"}`)
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"n1": true, "n2": false}, readState(t, s, "u1"))
}

func TestNotificationList(t *testing.T) {
	s := seededInbox(t)
	_, err := callNotificationTool(notifAs("u1"), t, s, "notification_update", `{"id":"n1"}`)
	require.NoError(t, err)

	page := func(args string) mcptool.Page[notificationResult] {
		out, err := callNotificationTool(notifAs("u1"), t, s, "notification_list", args)
		require.NoError(t, err)
		return out.(mcptool.Page[notificationResult])
	}
	assert.Equal(t, 2, page(`{}`).Total, "only the caller's notifications")
	unread := page(`{"unread_only":true}`)
	require.Len(t, unread.Items, 1)
	assert.Equal(t, "n2", unread.Items[0].ID)
	assert.Equal(t, "Spec", unread.Items[0].SubjectTitle)
	assert.Equal(t, "f-gs", unread.Items[0].FolderID, "a doc's folder reaches agents as it reaches the browser")
	assert.Equal(t, "GetSource", unread.Items[0].FolderName)
	assert.True(t, page(`{"limit":1}`).HasMore)
	inWS2 := page(`{"workspace_id":"ws-2"}`)
	require.Len(t, inWS2.Items, 1, "only the named workspace's notifications")
	assert.Equal(t, "ws-2", inWS2.Items[0].WorkspaceID)
}

func TestNotificationList_CountsPastAThousand(t *testing.T) {
	repo := newFakeNotifRepo()
	for i := range 1200 {
		repo.create(t, &Notification{ID: notifID(i), UserID: "u1", Read: i%2 == 0})
	}
	s := newTestNotifService(repo, newFakeNotifUsers())

	out, err := callNotificationTool(notifAs("u1"), t, s, "notification_list", `{"unread_only":true,"offset":500,"limit":50}`)

	require.NoError(t, err)
	page := out.(mcptool.Page[notificationResult])
	assert.Len(t, page.Items, 50)
	assert.Equal(t, 600, page.Total)
	assert.True(t, page.HasMore)
	assert.Equal(t, 550, page.NextOffset)
}
