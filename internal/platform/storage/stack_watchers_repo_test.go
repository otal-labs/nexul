package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/workspace"
)

func TestStackWatchersRepo_OptingOutAndBackIn(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedStackForContainers(t, s, "stack-1")
	at := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)

	out, err := s.StackWatchers.Unwatching(ctx, "stack-1", "alice")
	require.NoError(t, err)
	assert.False(t, out, "nobody has opted out of a new stack")

	require.NoError(t, s.StackWatchers.SetWatching(ctx, "stack-1", "alice", false, at))
	require.NoError(t, s.StackWatchers.SetWatching(ctx, "stack-1", "alice", false, at.Add(time.Hour)), "opting out twice is the same row")
	require.NoError(t, s.StackWatchers.SetWatching(ctx, "stack-1", "bob", false, at))
	out, err = s.StackWatchers.Unwatching(ctx, "stack-1", "alice")
	require.NoError(t, err)
	assert.True(t, out)
	ids, err := s.StackWatchers.ListUnwatcherIDs(ctx, "stack-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, ids)

	require.NoError(t, s.StackWatchers.SetWatching(ctx, "stack-1", "alice", true, at))
	require.NoError(t, s.StackWatchers.SetWatching(ctx, "stack-1", "alice", true, at), "turning on when already on is a no-op")
	ids, err = s.StackWatchers.ListUnwatcherIDs(ctx, "stack-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"bob"}, ids, "watching again removes the opt-out")
}

func TestStackWatchersRepo_ADeletedStackTakesItsOptOutsWithIt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.StackWatchers.SetWatching(ctx, "stack-1", "alice", false, time.Now()))

	require.NoError(t, s.Stacks.Delete(ctx, "stack-1"))

	ids, err := s.StackWatchers.ListUnwatcherIDs(ctx, "stack-1")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestNotificationsRepo_AStackNoticeFollowsItsProjectsAccess(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "alice")
	seedProject(t, s, "p-hidden", "workspace-default", "HID")
	seedStackForContainers(t, s, "stack-open")
	hidden := newTestStack("stack-hidden")
	hidden.Name, hidden.Slug, hidden.ProjectID = "hidden", "hidden", "p-hidden"
	require.NoError(t, s.Stacks.Create(ctx, hidden))
	var rows []*workspace.Notification
	for _, id := range []string{"stack-open", "stack-hidden"} {
		n := newTestNotification("n-"+id, "u1", false)
		n.WorkspaceID, n.Kind, n.SubjectType, n.SubjectID = "workspace-default", workspace.KindServiceStopped, workspace.SubjectStack, id
		rows = append(rows, n)
	}
	require.NoError(t, s.Notifications.CreateMany(ctx, rows))
	scope := &workspace.InboxScope{WorkspaceIDs: []string{"workspace-default"}, ProjectIDs: []string{"project-general"}}

	ns, total, err := s.Notifications.Page(ctx, "u1", workspace.InboxFilter{}, scope, paging.Window{Limit: 10})

	require.NoError(t, err)
	require.Len(t, ns, 1)
	assert.Equal(t, "stack-open", ns[0].SubjectID, "a notice about a stack of a project the reader cannot open is left out")
	assert.Equal(t, 1, total)
	assert.Equal(t, "project-general", ns[0].ProjectID)

	unread, err := s.Notifications.UnreadByProject(ctx, "u1", "")
	require.NoError(t, err)
	byProject := map[string]int{}
	for _, g := range unread {
		byProject[g.ProjectID] += g.Unread
	}
	assert.Equal(t, map[string]int{"project-general": 1, "p-hidden": 1}, byProject)
}
