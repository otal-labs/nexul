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
		MemoryIDs:  []string{}, ComputerIDs: []string{}, CreatedAt: at, UpdatedAt: at,
	}
}

func seedPlays(t *testing.T, s *Store, ids ...string) {
	t.Helper()
	require.NoError(t, s.Workspaces.Create(t.Context(), newTestWorkspace("ws-1", "Acme")))
	for _, id := range ids {
		p := newTestPlay(id, "ws-1")
		p.Label = id
		require.NoError(t, s.Plays.Create(t.Context(), p))
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

func TestAutoPlaysRepo_RoundTripsTheComputersInOrderAndThePersonWhoPickedThem(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1")
	at := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	want := newTestAutoPlay("ap-1", "play-1", at)
	want.Moment, want.RunOn, want.ComputerIDs, want.ComputerID, want.RunAs = plays.MomentStackServiceCrashed, plays.RunOnComputer, []string{"c-2", "c-1"}, "c-2", "u-alice"
	require.NoError(t, s.Plays.CreateAutoPlay(t.Context(), want))

	got, err := s.Plays.GetAutoPlay(t.Context(), "ap-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)

	got.ComputerIDs, got.ComputerID, got.RunAs = []string{"c-3"}, "c-3", "u-bob"
	require.NoError(t, s.Plays.UpdateAutoPlay(t.Context(), got))
	again, err := s.Plays.GetAutoPlay(t.Context(), "ap-1")
	require.NoError(t, err)
	assert.Equal(t, got, again)

	got.ComputerIDs, got.ComputerID = []string{}, ""
	require.NoError(t, s.Plays.UpdateAutoPlay(t.Context(), got))
	yours, err := s.Plays.GetAutoPlay(t.Context(), "ap-1")
	require.NoError(t, err)
	assert.Equal(t, []any{[]string{}, "", "u-bob"}, []any{yours.ComputerIDs, yours.ComputerID, yours.RunAs}, "the picker's own order is an empty list with a person")
	var older string
	require.NoError(t, s.db.QueryRowContext(t.Context(), `SELECT computer_id FROM auto_plays WHERE id = 'ap-1'`).Scan(&older))
	assert.Empty(t, older, "computer_id stays the list's first, for a downgrade")
}

func TestAutoPlaysRepo_RoundTripsTheMemories(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1")
	a := newTestAutoPlay("ap-1", "play-1", time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC))
	a.MemoryIDs = []string{"m-1", "m-2"}
	require.NoError(t, s.Plays.CreateAutoPlay(t.Context(), a))
	got, err := s.Plays.GetAutoPlay(t.Context(), "ap-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"m-1", "m-2"}, got.MemoryIDs)

	got.MemoryIDs = nil
	require.NoError(t, s.Plays.UpdateAutoPlay(t.Context(), got))
	again, err := s.Plays.GetAutoPlay(t.Context(), "ap-1")
	require.NoError(t, err)
	assert.Equal(t, []string{}, again.MemoryIDs, "none reads back as an empty list, never null on the wire")
}

func TestAutoPlaysRepo_ForgetADeletedMemory(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s, "play-1")
	ctx := t.Context()
	for _, id := range []string{"mem-1", "mem-2"} {
		require.NoError(t, s.Memories.Create(ctx, newTestMemory(id, "project-general"), ""))
	}
	a := newTestAutoPlay("ap-1", "play-1", time.Now())
	a.MemoryIDs = []string{"mem-1", "mem-2"}
	require.NoError(t, s.Plays.CreateAutoPlay(ctx, a))

	require.NoError(t, s.Memories.Delete(ctx, "mem-1"))
	got, err := s.Plays.GetAutoPlay(ctx, "ap-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"mem-2"}, got.MemoryIDs)

	require.NoError(t, s.Memories.Delete(ctx, "mem-2"))
	got, err = s.Plays.GetAutoPlay(ctx, "ap-1")
	require.NoError(t, err)
	assert.Equal(t, []string{}, got.MemoryIDs)
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

func TestAutoPlaysRepo_LimitsStartAtFiveADay(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedPlays(t, s)
	ctx := t.Context()
	limits, err := s.Plays.AutoPlayLimits(ctx, "ws-1")
	require.NoError(t, err)
	assert.Equal(t, plays.AutoPlayLimits{DailyCapPerTicket: plays.DefaultAutoPlayDailyCap}, limits)
	want := plays.AutoPlayLimits{DailyCapPerTicket: 9}
	evt := eventbus.OutboxEvent{ID: "evt-cap", Topic: plays.TopicAutoPlayLimitsUpdated, Payload: plays.AutoPlayLimitsEvent{WorkspaceID: "ws-1", DailyCapPerTicket: 9}}
	require.NoError(t, s.Plays.SetAutoPlayLimits(ctx, "ws-1", want, evt))
	limits, err = s.Plays.AutoPlayLimits(ctx, "ws-1")
	require.NoError(t, err)
	assert.Equal(t, want, limits)
	var topic string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT topic FROM outbox WHERE id = 'evt-cap'`).Scan(&topic))
	assert.Equal(t, plays.TopicAutoPlayLimitsUpdated, topic)

	_, err = s.Plays.AutoPlayLimits(ctx, "ghost")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, s.Plays.SetAutoPlayLimits(ctx, "ghost", want), apperrs.ErrNotFound)
}
