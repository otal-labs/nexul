package storage_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
)

// settle closes the windows due by at and returns what settled, after checking each became a doc.settled outbox row.
func settle(t *testing.T, s *storage.Store, at time.Time) []docs.SettledEvent {
	t.Helper()
	var got []docs.SettledEvent
	before, err := s.Outbox.Unpublished(t.Context(), 1000)
	require.NoError(t, err)
	err = s.Docs.SettleDue(t.Context(), at, func(settled []docs.SettledEvent) []eventbus.OutboxEvent {
		got = settled
		out := make([]eventbus.OutboxEvent, 0, len(settled))
		for _, e := range settled {
			out = append(out, eventbus.OutboxEvent{ID: ids.New(), Topic: docs.TopicSettled, Payload: e})
		}
		return out
	})
	require.NoError(t, err)
	after, err := s.Outbox.Unpublished(t.Context(), 1000)
	require.NoError(t, err)
	require.Len(t, after, len(before)+len(got))
	return got
}

func TestDocSettles_Integration_ACreateEditedInsideTheWindowSettlesOnceAsCreated(t *testing.T) {
	ctx := actorCtx()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	svc := docs.NewService(s.Docs, allowAll{}, nil)

	d, err := svc.Create(ctx, testutil.GeneralProjectID, "Spec", "draft")
	require.NoError(t, err)
	_, err = svc.Update(ctx, d.ID, "Spec v2", "more")
	require.NoError(t, err)

	due, ok, err := s.Docs.NextSettleDue(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(docs.SettleWindow), due, time.Minute)
	assert.Empty(t, settle(t, s, time.Now()), "nothing settles inside the window")

	got := settle(t, s, due)
	assert.Equal(t, []docs.SettledEvent{{
		Doc: docs.WatchedDoc{ID: d.ID, ProjectID: testutil.GeneralProjectID, Title: "Spec v2"}, First: true, ActorID: "tester",
	}}, got)
	_, ok, err = s.Docs.NextSettleDue(ctx)
	require.NoError(t, err)
	assert.False(t, ok, "a settled window is closed")

	_, err = svc.Update(ctx, d.ID, "Spec v3", "more")
	require.NoError(t, err)
	got = settle(t, s, time.Now().Add(docs.SettleWindow+time.Minute))
	require.Len(t, got, 1)
	assert.False(t, got[0].First, "an edit after the doc settled is a change")
}

func TestDocSettles_Integration_AWindowClosesWithoutAnEventForAGoneOrArchivedDoc(t *testing.T) {
	ctx := actorCtx()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	svc := docs.NewService(s.Docs, allowAll{}, nil)
	archived, err := svc.Create(ctx, testutil.GeneralProjectID, "Archived", "draft")
	require.NoError(t, err)
	deleted, err := svc.Create(ctx, testutil.GeneralProjectID, "Deleted", "draft")
	require.NoError(t, err)

	_, err = svc.Archive(ctx, archived.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, deleted.ID))

	assert.Empty(t, settle(t, s, time.Now().Add(docs.SettleWindow+time.Minute)))
	_, ok, err := s.Docs.NextSettleDue(ctx)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDocSettles_Integration_AWindowDueWhileTheServerWasDownSettlesAfterARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.OpenDB(path)
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	require.NoError(t, testutil.SeedGeneralProject(db))
	d, err := docs.NewService(storage.New(db, []byte("0123456789abcdef0123456789abcdef")).Docs, allowAll{}, nil).
		Create(actorCtx(), testutil.GeneralProjectID, "Spec", "draft")
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = storage.OpenDB(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	got := settle(t, s, time.Now().Add(docs.SettleWindow+time.Hour))
	require.Len(t, got, 1)
	assert.Equal(t, d.ID, got[0].Doc.ID)
}
