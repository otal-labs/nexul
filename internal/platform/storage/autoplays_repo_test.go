package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"
)

func newTestAutoPlay(id, playID string, at time.Time) *plays.AutoPlay {
	return &plays.AutoPlay{
		ID: id, PlayID: playID, WorkspaceID: "ws-1", Moment: plays.MomentDocChanged, RunOn: plays.RunOnCauser,
		Conditions: plays.Conditions{Match: plays.MatchAll, Groups: []plays.Group{}},
		Priority:   plays.Priority{Rules: []plays.PriorityRule{}, Otherwise: plays.LevelNormal},
		CreatedAt:  at, UpdatedAt: at,
	}
}

func seedPlays(t *testing.T, s *Store, ids ...string) {
	t.Helper()
	require.NoError(t, s.Workspaces.Create(t.Context(), newTestWorkspace("ws-1", "Acme")))
	for _, id := range ids {
		require.NoError(t, s.Plays.Create(t.Context(), newTestPlay(id, "ws-1")))
	}
}

func TestAutoPlaysRepo_RoundTripsTheTreesAndPublishesInTheSameWrite(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1")
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	stage := plays.StageDone
	want := newTestAutoPlay("ap-1", "play-1", at)
	want.Moment, want.MomentStage, want.RunOn, want.OnceWithinMinutes = plays.MomentTicketEnteredStage, &stage, plays.RunOnTester, 90
	want.Conditions = plays.Conditions{Match: plays.MatchAny, Groups: []plays.Group{{Match: plays.MatchAll, Rules: []plays.Rule{
		{Field: plays.FieldType, Op: plays.OpIs, Values: []string{"Bug"}}, {Field: plays.FieldBlocked, Op: plays.OpUnset},
	}}}}
	want.Priority = plays.Priority{Otherwise: plays.LevelLow, Rules: []plays.PriorityRule{
		{Level: plays.LevelHigh, When: plays.Group{Match: plays.MatchAll, Rules: []plays.Rule{{Field: plays.FieldLabel, Op: plays.OpIs, Values: []string{"urgent"}}}}},
	}}
	want.CreatedBy = "u-1"
	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: plays.TopicAutoPlayCreated, Payload: plays.AutoPlayEvent{AutoPlay: *want}}
	require.NoError(t, s.Plays.CreateAutoPlay(ctx, want, evt))

	got, err := s.Plays.GetAutoPlay(ctx, "ap-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
	var n int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE topic = ?`, plays.TopicAutoPlayCreated).Scan(&n))
	assert.Equal(t, 1, n)

	got.Enabled, got.MomentStage, got.Moment = true, nil, plays.MomentTicketCreated
	got.UpdatedAt = at.Add(time.Hour)
	require.NoError(t, s.Plays.UpdateAutoPlay(ctx, got))
	again, err := s.Plays.GetAutoPlay(ctx, "ap-1")
	require.NoError(t, err)
	assert.Equal(t, got, again)
}

func TestAutoPlaysRepo_ListsEachPlaysOldestFirstInOneRead(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1", "play-2", "play-3")
	ctx := t.Context()
	at := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	for _, a := range []*plays.AutoPlay{
		newTestAutoPlay("ap-late", "play-1", at.Add(time.Minute)),
		newTestAutoPlay("ap-early", "play-1", at),
		newTestAutoPlay("ap-other", "play-2", at),
		newTestAutoPlay("ap-unasked", "play-3", at),
	} {
		require.NoError(t, s.Plays.CreateAutoPlay(ctx, a))
	}
	list, err := s.Plays.ListAutoPlays(ctx, []string{"play-1", "play-2"})
	require.NoError(t, err)
	ids := make([]string, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	assert.Equal(t, []string{"ap-early", "ap-late", "ap-other"}, ids)

	none, err := s.Plays.ListAutoPlays(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestAutoPlaysRepo_GoWithTheirPlay(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1")
	ctx := t.Context()
	require.NoError(t, s.Plays.CreateAutoPlay(ctx, newTestAutoPlay("ap-1", "play-1", time.Now())))
	require.NoError(t, s.Plays.Delete(ctx, "play-1"))
	_, err := s.Plays.GetAutoPlay(ctx, "ap-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestAutoPlaysRepo_MissingRowsAndAMissingPlay(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1")
	ctx := t.Context()
	require.ErrorIs(t, s.Plays.UpdateAutoPlay(ctx, newTestAutoPlay("ghost", "play-1", time.Now())), apperrs.ErrNotFound)
	require.ErrorIs(t, s.Plays.DeleteAutoPlay(ctx, "ghost"), apperrs.ErrNotFound)
	require.Error(t, s.Plays.CreateAutoPlay(ctx, newTestAutoPlay("ap-1", "no-such-play", time.Now())), "the play must exist")

	require.NoError(t, s.Plays.CreateAutoPlay(ctx, newTestAutoPlay("ap-1", "play-1", time.Now())))
	require.NoError(t, s.Plays.DeleteAutoPlay(ctx, "ap-1"))
	_, err := s.Plays.GetAutoPlay(ctx, "ap-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestAutoPlaysRepo_DailyCapStartsAtFive(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	ctx := t.Context()
	limit, err := s.Plays.AutoPlayDailyCap(ctx, "ws-1")
	require.NoError(t, err)
	assert.Equal(t, plays.DefaultAutoPlayDailyCap, limit)
	require.NoError(t, s.Plays.SetAutoPlayDailyCap(ctx, "ws-1", 9))
	limit, err = s.Plays.AutoPlayDailyCap(ctx, "ws-1")
	require.NoError(t, err)
	assert.Equal(t, 9, limit)

	_, err = s.Plays.AutoPlayDailyCap(ctx, "ghost")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, s.Plays.SetAutoPlayDailyCap(ctx, "ghost", 3), apperrs.ErrNotFound)
}
