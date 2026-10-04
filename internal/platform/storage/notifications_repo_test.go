package storage

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/docs"
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
	_, _, err := s.Users.UpsertUser(context.Background(), &auth.Identity{
		UserID: id, Provider: auth.ProviderGitHub, ProviderUserID: "p-" + id, Login: login,
	})
	require.NoError(t, err)
}

func TestNotificationsRepo_CreateMany_DuplicateID_IsNoOp(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))

	ns, err := s.Notifications.List(context.Background(), "u1", "", 50)
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

	got, err := s.Notifications.List(context.Background(), "u1", "", 50)
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

func TestNotificationsRepo_CreateMany_RepeatWhileUnread_LiftsTheRow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	outboxRows := func(id string) int {
		var n int
		require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE id = ?`, id).Scan(&n))
		return n
	}

	first := newTestNotification("c1", "u1", false)
	first.Kind = workspace.KindDocQuestionsAsked
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{first}))

	repeat := newTestNotification("c2", "u1", false)
	repeat.Kind, repeat.SubjectTitle, repeat.CreatedAt = workspace.KindDocQuestionsAsked, "New questions on Spec v2", first.CreatedAt.Add(time.Hour)
	evt := eventbus.OutboxEvent{ID: "evt-repeat", Topic: workspace.TopicNotificationCreated, Payload: workspace.NotificationCreatedEvent{}}
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{repeat}, evt))
	ns, err := s.Notifications.List(ctx, "u1", "", 50)
	require.NoError(t, err)
	require.Len(t, ns, 1, "still one unread row for the subject")
	assert.Equal(t, "c2", ns[0].ID, "the row carries the newest event's id, the one its push names")
	assert.Equal(t, "New questions on Spec v2", ns[0].SubjectTitle)
	assert.True(t, ns[0].CreatedAt.Equal(repeat.CreatedAt), "it rises to the top of the inbox")
	assert.Equal(t, 1, outboxRows("evt-repeat"), "and its push fires")

	again := eventbus.OutboxEvent{ID: "evt-redelivered", Topic: workspace.TopicNotificationCreated, Payload: workspace.NotificationCreatedEvent{}}
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{repeat}, again))
	assert.Equal(t, 0, outboxRows("evt-redelivered"), "a redelivery of the same event changes nothing")

	require.NoError(t, s.Notifications.MarkRead(ctx, "u1", "c2", time.Now()))
	after := newTestNotification("c3", "u1", false)
	after.Kind = workspace.KindDocQuestionsAsked
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{after}))
	ns, err = s.Notifications.List(ctx, "u1", "", 50)
	require.NoError(t, err)
	require.Len(t, ns, 2, "a read row stays as it was and the next one is new")
}

func TestNotificationsRepo_CreateMany_EditRepeatWhileUnread_StaysCollapsed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")

	first := newTestNotification("e1", "u1", false)
	first.Kind = workspace.KindDocUpdated
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{first}))

	// Collab commits fire doc.updated every few seconds mid-edit; each must not push again.
	repeat := newTestNotification("e2", "u1", false)
	repeat.Kind, repeat.CreatedAt = workspace.KindDocUpdated, first.CreatedAt.Add(time.Minute)
	evt := eventbus.OutboxEvent{ID: "evt-edit", Topic: workspace.TopicNotificationCreated, Payload: workspace.NotificationCreatedEvent{}}
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{repeat}, evt))
	ns, err := s.Notifications.List(ctx, "u1", "", 50)
	require.NoError(t, err)
	require.Len(t, ns, 1)
	assert.Equal(t, "e1", ns[0].ID)
	var n int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE id = 'evt-edit'`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestNotificationsRepo_CreateMany_UnreadInAnotherWorkspace_DoesNotCollapse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{newTestNotification("old", "u1", false)}))

	scoped := newTestNotification("new", "u1", false)
	scoped.WorkspaceID = "ws-1"
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{scoped}))

	ns, err := s.Notifications.List(ctx, "u1", "ws-1", 50)
	require.NoError(t, err)
	assert.Len(t, ns, 1, "an unread row the ws-1 inbox never shows must not silence ws-1")
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

	ns, err := s.Notifications.List(context.Background(), "u1", "", 50)
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

	ns, err := s.Notifications.List(context.Background(), "u1", "", 1)
	require.NoError(t, err)
	require.Len(t, ns, 1)
}

func TestNotificationsRepo_UnreadCount(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n2", "u1", true)}))

	n, err := s.Notifications.UnreadByProject(context.Background(), "u1", "")
	require.NoError(t, err)
	assert.Equal(t, []workspace.UnreadGroup{{Unread: 1}}, n)
}

func TestNotificationsRepo_MarkRead_NotFoundForOtherUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	mustCreateUser(t, s, "u2", "alice")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))

	readAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	err := s.Notifications.MarkRead(context.Background(), "u2", "n1", readAt)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	require.NoError(t, s.Notifications.MarkRead(context.Background(), "u1", "n1", readAt))
	require.NoError(t, s.Notifications.MarkRead(context.Background(), "u1", "n1", readAt.Add(48*time.Hour)))
	ns, err := s.Notifications.List(context.Background(), "u1", "", 50)
	require.NoError(t, err)
	assert.True(t, ns[0].Read)
	require.NotNil(t, ns[0].ReadAt, "reading records when, which retention counts from")
	assert.Equal(t, readAt, *ns[0].ReadAt, "reading it again keeps the first read time, so retention is never extended")
}

func TestNotificationsRepo_MarkAllRead_ScopedToUser(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	mustCreateUser(t, s, "u2", "alice")
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n1", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n2", "u1", false)}))
	require.NoError(t, s.Notifications.CreateMany(context.Background(), []*workspace.Notification{newTestNotification("n3", "u2", false)}))

	readAt := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	require.NoError(t, s.Notifications.MarkAllRead(context.Background(), "u1", "", readAt))

	u1, err := s.Notifications.List(context.Background(), "u1", "", 50)
	require.NoError(t, err)
	for _, n := range u1 {
		assert.True(t, n.Read)
		require.NotNil(t, n.ReadAt)
		assert.Equal(t, readAt, *n.ReadAt)
	}
	u2, err := s.Notifications.List(context.Background(), "u2", "", 50)
	require.NoError(t, err)
	for _, n := range u2 {
		assert.False(t, n.Read)
		assert.Nil(t, n.ReadAt)
	}
}

func TestNotificationsRepo_WorkspaceFilter_ScopesListCountAndReadAll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	for _, n := range []struct{ id, ws, subject string }{{"n1", "ws-1", "d-1"}, {"n2", "ws-2", "d-2"}, {"n3", "ws-2", "d-3"}} {
		row := newTestNotification(n.id, "u1", false)
		row.WorkspaceID, row.SubjectID = n.ws, n.subject
		require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{row}))
	}

	ws1, err := s.Notifications.List(ctx, "u1", "ws-1", 50)
	require.NoError(t, err)
	require.Len(t, ws1, 1)
	assert.Equal(t, "ws-1", ws1[0].WorkspaceID, "the stored workspace round-trips")
	all, err := s.Notifications.List(ctx, "u1", "", 50)
	require.NoError(t, err)
	assert.Len(t, all, 3, "no workspace lists every workspace")

	count, err := s.Notifications.UnreadByProject(ctx, "u1", "ws-2")
	require.NoError(t, err)
	assert.Equal(t, []workspace.UnreadGroup{{WorkspaceID: "ws-2", Unread: 2}}, count)

	require.NoError(t, s.Notifications.MarkAllRead(ctx, "u1", "ws-2", time.Now()))
	count, err = s.Notifications.UnreadByProject(ctx, "u1", "")
	require.NoError(t, err)
	assert.Equal(t, []workspace.UnreadGroup{{WorkspaceID: "ws-1", Unread: 1}}, count, "read-all in ws-2 leaves ws-1 unread")
}

func TestNotificationsRepo_List_CarriesTheDocsCurrentFolder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	mustCreateUser(t, s, "u1", "onik97")
	require.NoError(t, s.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-gs", ProjectID: "project-general", Name: "GetSource"}))
	ep := newTestDoc("d-ep07")
	ep.FolderID = "f-gs"
	require.NoError(t, s.Docs.Create(ctx, ep))
	inMain := newTestDoc("d-main")
	inMain.FolderID = "folder-general-main"
	require.NoError(t, s.Docs.Create(ctx, inMain))

	docNote := newTestNotification("n-ep", "u1", false)
	docNote.SubjectID = "d-ep07"
	mainNote := newTestNotification("n-main", "u1", false)
	mainNote.SubjectID = "d-main"
	ticketNote := newTestNotification("n-ticket", "u1", false)
	ticketNote.Kind, ticketNote.SubjectType, ticketNote.SubjectID = workspace.KindTicketAssigned, workspace.SubjectTicket, "d-ep07"
	require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{docNote, mainNote, ticketNote}))

	byID := func() map[string]*workspace.Notification {
		ns, err := s.Notifications.List(ctx, "u1", "", 50)
		require.NoError(t, err)
		out := map[string]*workspace.Notification{}
		for _, n := range ns {
			out[n.ID] = n
		}
		return out
	}
	got := byID()
	assert.Equal(t, "f-gs", got["n-ep"].FolderID)
	assert.Equal(t, "GetSource", got["n-ep"].FolderName)
	assert.False(t, got["n-ep"].FolderIsDefault)
	assert.Equal(t, "folder-general-main", got["n-main"].FolderID)
	assert.True(t, got["n-main"].FolderIsDefault)
	assert.Empty(t, got["n-ticket"].FolderID, "a ticket sharing the doc's id is not a doc")

	require.NoError(t, s.Docs.SetDocFolder(ctx, "d-ep07", "folder-general-main"))
	got = byID()
	assert.Equal(t, "folder-general-main", got["n-ep"].FolderID, "the folder is read at list time, so a move shows")
	assert.True(t, got["n-ep"].FolderIsDefault)
}

func TestNotificationsRepo_DeleteExpired_DeletesExactlyPastEachCutoff(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s := newTestStore(t)
	mustCreateUser(t, s, "u1", "onik97")
	readBefore := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	createdBefore := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	at := func(t time.Time, seconds int) *time.Time {
		v := t.Add(time.Duration(seconds) * time.Second)
		return &v
	}
	rows := []struct {
		id        string
		createdAt time.Time
		readAt    *time.Time
	}{
		{"read-past", *at(createdBefore, 10), at(readBefore, -1)},
		{"read-at-cutoff", *at(createdBefore, 10), at(readBefore, 0)},
		{"unread-recent", *at(createdBefore, 10), nil},
		{"unread-past", *at(createdBefore, -1), nil},
		{"unread-at-cutoff", *at(createdBefore, 0), nil},
		{"read-recently-sent-long-ago", *at(createdBefore, -1), at(readBefore, 60)},
	}
	for i, r := range rows {
		n := newTestNotification(r.id, "u1", r.readAt != nil)
		n.SubjectID, n.CreatedAt, n.ReadAt = "d-"+strconv.Itoa(i), r.createdAt, r.readAt
		require.NoError(t, s.Notifications.CreateMany(ctx, []*workspace.Notification{n}))
	}

	read, old, err := s.Notifications.DeleteExpired(ctx, readBefore, createdBefore)
	require.NoError(t, err)
	assert.Equal(t, int64(1), read)
	assert.Equal(t, int64(2), old, "an old notification goes whether or not it was read")

	ns, err := s.Notifications.List(ctx, "u1", "", 50)
	require.NoError(t, err)
	var kept []string
	for _, n := range ns {
		kept = append(kept, n.ID)
	}
	assert.ElementsMatch(t, []string{"read-at-cutoff", "unread-recent", "unread-at-cutoff"}, kept)
}
