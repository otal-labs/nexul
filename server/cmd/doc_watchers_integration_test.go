package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

// seedSecondWriter adds kai, a second member who writes docs, beside seedMentionPeople's onik.
func seedSecondWriter(t *testing.T, store *storage.Store) {
	t.Helper()
	ctx := context.Background()
	_, _, err := store.Users.UpsertUser(ctx, &auth.Identity{UserID: "u-kai", Provider: auth.ProviderGitHub, ProviderUserID: "u-kai", Login: "kai"})
	require.NoError(t, err)
	require.NoError(t, store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: "u-kai", WorkspaceID: "workspace-default", RoleID: "role-writer", CreatedAt: time.Now()}))
}

func watcherIDs(t *testing.T, svc *coreServices, actor, docID string) []string {
	t.Helper()
	ws, err := svc.docsSvc.Watchers(as(actor), docID)
	require.NoError(t, err)
	var ids []string
	for _, w := range ws.Watchers {
		ids = append(ids, w.UserID)
	}
	return ids
}

// readAll empties a person's unread inbox, so the unread collapse cannot hide the next notification about a doc.
func readAll(t *testing.T, svc *coreServices, userIDs ...string) {
	t.Helper()
	for _, id := range userIDs {
		require.NoError(t, svc.notifSvc.MarkAllRead(context.Background(), id, ""))
	}
}

// TestDocWatchers_ChangesNotifyOnlyWatchers saves docs the way the editor and agents do and checks the inbox: a doc's
// creator and editors watch it, anyone who can read it may watch or stop, and only watchers hear about an edit.
func TestDocWatchers_ChangesNotifyOnlyWatchers(t *testing.T) {
	svc, store := newWired(t)
	seedMentionPeople(t, store)
	seedSecondWriter(t, store)
	plan, err := svc.docsSvc.Create(as("u-onik"), "project-general", "Launch plan", "")
	require.NoError(t, err)
	deliverNotifications(t, svc, store)

	t.Run("a new doc notifies nobody and is watched by its creator", func(t *testing.T) {
		for _, id := range []string{"u-nor", "u-kai", "u-onik"} {
			assert.Empty(t, inbox(t, svc, id, workspace.KindDocCreated), id)
		}
		assert.Equal(t, []string{"u-onik"}, watcherIDs(t, svc, "u-nor", plan.ID))
	})

	t.Run("an edit makes its editor a watcher and tells the other watchers", func(t *testing.T) {
		_, err := svc.docsSvc.Update(as("u-kai"), plan.ID, "Launch plan", "")
		require.NoError(t, err)
		deliverNotifications(t, svc, store)
		assert.ElementsMatch(t, []string{"u-onik", "u-kai"}, watcherIDs(t, svc, "u-onik", plan.ID))
		assert.Equal(t, []string{"Launch plan"}, inbox(t, svc, "u-onik", workspace.KindDocUpdated))
		assert.Empty(t, inbox(t, svc, "u-kai", workspace.KindDocUpdated), "the editor is not told of their own edit")
		assert.Empty(t, inbox(t, svc, "u-nor", workspace.KindDocUpdated), "a reader who does not watch is not told")
	})

	t.Run("a reader who watches is told of the next edit", func(t *testing.T) {
		readAll(t, svc, "u-onik", "u-kai", "u-nor")
		got, err := svc.docsSvc.SetWatching(as("u-nor"), plan.ID, true)
		require.NoError(t, err)
		assert.True(t, got.Watching)
		_, err = svc.docsSvc.Update(as("u-onik"), plan.ID, "Launch plan", "")
		require.NoError(t, err)
		deliverNotifications(t, svc, store)
		assert.Len(t, inbox(t, svc, "u-nor", workspace.KindDocUpdated), 1)
		assert.Len(t, inbox(t, svc, "u-kai", workspace.KindDocUpdated), 1)
	})

	t.Run("stopping sticks through the person's own later edit", func(t *testing.T) {
		readAll(t, svc, "u-onik", "u-kai", "u-nor")
		got, err := svc.docsSvc.SetWatching(as("u-kai"), plan.ID, false)
		require.NoError(t, err)
		assert.False(t, got.Watching)
		_, err = svc.docsSvc.Update(as("u-kai"), plan.ID, "Launch plan v2", "")
		require.NoError(t, err)
		assert.NotContains(t, watcherIDs(t, svc, "u-kai", plan.ID), "u-kai")
		_, err = svc.docsSvc.Update(as("u-onik"), plan.ID, "Launch plan v3", "")
		require.NoError(t, err)
		deliverNotifications(t, svc, store)
		assert.Len(t, inbox(t, svc, "u-kai", workspace.KindDocUpdated), 1, "only the edit from before kai stopped")
		assert.Len(t, inbox(t, svc, "u-onik", workspace.KindDocUpdated), 2, "kai's own edit still tells the other watchers")
	})

	t.Run("a mention reaches someone who is not watching and does not make them a watcher", func(t *testing.T) {
		readAll(t, svc, "u-onik", "u-kai", "u-nor")
		_, err := svc.docsSvc.Update(as("u-onik"), plan.ID, "Launch plan v3", personMentionBody("u-kai"))
		require.NoError(t, err)
		deliverNotifications(t, svc, store)
		assert.Equal(t, []string{"Onik mentioned you in Launch plan v3"}, inbox(t, svc, "u-kai", workspace.KindDocMentioned))
		assert.NotContains(t, watcherIDs(t, svc, "u-onik", plan.ID), "u-kai")
	})

	t.Run("a watcher who can no longer read the doc is told nothing", func(t *testing.T) {
		readAll(t, svc, "u-onik", "u-kai", "u-nor")
		require.Contains(t, watcherIDs(t, svc, "u-onik", plan.ID), "u-nor")
		before := len(inbox(t, svc, "u-nor", workspace.KindDocUpdated))
		require.NoError(t, store.WorkspaceMembers.SetRole(context.Background(), "workspace-default", "u-nor", "role-member"))
		_, err := svc.docsSvc.Update(as("u-onik"), plan.ID, "Launch plan v4", "")
		require.NoError(t, err)
		deliverNotifications(t, svc, store)
		assert.Len(t, inbox(t, svc, "u-nor", workspace.KindDocUpdated), before, "nothing new after losing docs:read")
	})

	t.Run("a collaborative save from the editor makes its editor a watcher too", func(t *testing.T) {
		notes, err := svc.docsSvc.Create(as("u-onik"), "project-general", "Meeting notes", "")
		require.NoError(t, err)
		require.NoError(t, svc.docsSvc.CommitCollab(as("u-kai"), notes.ID, "", personMentionBody()))
		assert.ElementsMatch(t, []string{"u-onik", "u-kai"}, watcherIDs(t, svc, "u-onik", notes.ID))
	})
}

// TestDocWatchers_WatchingTakesReadingTheDoc checks watching is gated where the doc lives: a member who cannot read
// docs is forbidden, and someone outside the workspace learns nothing about it.
func TestDocWatchers_WatchingTakesReadingTheDoc(t *testing.T) {
	svc, store := newWired(t)
	seedMentionPeople(t, store)
	plan, err := svc.docsSvc.Create(as("u-onik"), "project-general", "Launch plan", "")
	require.NoError(t, err)

	_, err = svc.docsSvc.SetWatching(as("u-sam"), plan.ID, true)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = svc.docsSvc.SetWatching(as("u-out"), plan.ID, true)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = svc.docsSvc.Watchers(as("u-sam"), plan.ID)
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	got, err := svc.docsSvc.SetWatching(as("u-nor"), plan.ID, true)
	require.NoError(t, err, "reading is enough; watching never needs docs:write")
	assert.True(t, got.Watching)
	assert.Contains(t, watcherIDs(t, svc, "u-onik", plan.ID), "u-nor")
}
