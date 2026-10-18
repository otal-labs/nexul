package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func newTestEpisode(id, stackID string) *deploy.ErrorRateEpisode {
	return &deploy.ErrorRateEpisode{
		ID: id, StackID: stackID, StartedAt: time.Date(2026, 10, 1, 10, 42, 0, 0, time.UTC), PeakPercent: 7.5,
		Requests: 400, Errors: 30, LimitPercent: 5, WindowMinutes: 5, MinRequests: 20,
	}
}

func episodeEvent(id, topic string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: id, Topic: topic, Payload: deploy.ErrorRateEvent{StackID: "stack-1", EpisodeID: "ep-1"}}
}

func TestErrorRateRepo_OpenAndGetOpen_RoundTripWithItsEvent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-1", "stack-1"), episodeEvent("evt-1", deploy.TopicErrorRateCrossed)))

	got, err := s.ErrorRate.GetOpen(ctx, "stack-1")
	require.NoError(t, err)
	assert.Equal(t, newTestEpisode("ep-1", "stack-1"), got)
	assert.Equal(t, []string{deploy.TopicErrorRateCrossed}, outboxTopicsOf(t, s), "the event is written with the episode")
}

func TestErrorRateRepo_Open_ASecondOpenEpisodeForAStackConflictsAndWritesNoEvent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-1", "stack-1")))

	err := s.ErrorRate.Open(ctx, newTestEpisode("ep-2", "stack-1"), episodeEvent("evt-2", deploy.TopicErrorRateCrossed))

	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Empty(t, outboxTopicsOf(t, s))
}

func TestErrorRateRepo_Open_AnUnknownStackFails(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.Error(t, s.ErrorRate.Open(context.Background(), newTestEpisode("ep-1", "nope")))
}

func TestErrorRateRepo_UpdatePeak_RecordsTheWorstWindow(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	e := newTestEpisode("ep-1", "stack-1")
	require.NoError(t, s.ErrorRate.Open(ctx, e))

	e.PeakPercent, e.Requests, e.Errors = 31.25, 800, 250
	require.NoError(t, s.ErrorRate.UpdatePeak(ctx, e))

	got, err := s.ErrorRate.GetOpen(ctx, "stack-1")
	require.NoError(t, err)
	assert.Equal(t, e, got)
}

func TestErrorRateRepo_Close_EndsItAndAllowsTheNextEpisode(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-1", "stack-1")))
	end := time.Date(2026, 10, 1, 10, 50, 0, 0, time.UTC)

	require.NoError(t, s.ErrorRate.Close(ctx, "ep-1", end, episodeEvent("evt-1", deploy.TopicErrorRateRecovered)))

	_, err := s.ErrorRate.GetOpen(ctx, "stack-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	got, err := s.ErrorRate.LatestEnd(ctx, "stack-1")
	require.NoError(t, err)
	assert.Equal(t, end, got)
	assert.Equal(t, []string{deploy.TopicErrorRateRecovered}, outboxTopicsOf(t, s))
	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-2", "stack-1")), "the stack can cross again")
}

func TestErrorRateRepo_Close_AnEndedOrUnknownEpisodeIsNotFoundAndWritesNoEvent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-1", "stack-1")))
	require.NoError(t, s.ErrorRate.Close(ctx, "ep-1", time.Unix(100, 0)))

	for _, id := range []string{"ep-1", "nope"} {
		err := s.ErrorRate.Close(ctx, id, time.Unix(200, 0), episodeEvent("evt-"+id, deploy.TopicErrorRateRecovered))
		require.ErrorIs(t, err, apperrs.ErrNotFound, id)
	}
	assert.Empty(t, outboxTopicsOf(t, s))
}

func TestErrorRateRepo_LatestEnd(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")

	got, err := s.ErrorRate.LatestEnd(ctx, "stack-1")
	require.NoError(t, err)
	assert.True(t, got.IsZero(), "a stack that never crossed has no end")

	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-1", "stack-1")))
	got, err = s.ErrorRate.LatestEnd(ctx, "stack-1")
	require.NoError(t, err)
	assert.True(t, got.IsZero(), "an open episode has not ended")

	require.NoError(t, s.ErrorRate.Close(ctx, "ep-1", time.Unix(500, 0)))
	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-2", "stack-1")))
	require.NoError(t, s.ErrorRate.Close(ctx, "ep-2", time.Unix(900, 0)))
	got, err = s.ErrorRate.LatestEnd(ctx, "stack-1")
	require.NoError(t, err)
	assert.Equal(t, time.Unix(900, 0).UTC(), got)
}

func TestErrorRateRepo_DeletingTheStackRemovesItsEpisodes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.ErrorRate.Open(ctx, newTestEpisode("ep-1", "stack-1")))

	require.NoError(t, s.Stacks.Delete(ctx, "stack-1"))

	_, err := s.ErrorRate.GetOpen(ctx, "stack-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestStacksRepo_ErrorRate_RoundTripAndListsOnlyStacksWithTheRuleOn(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	on, off := newTestStack("svc-1"), newTestStack("svc-2")
	off.Name, off.Slug = "web", "web"
	on.ErrorRate = deploy.ErrorRateRule{Enabled: true, Percent: 2.5, Minutes: 10, MinRequests: 50}
	off.ErrorRate = deploy.ErrorRateRule{Percent: 5, Minutes: 5, MinRequests: 20}
	require.NoError(t, s.Stacks.Create(ctx, on))
	require.NoError(t, s.Stacks.Create(ctx, off))

	got, err := s.Stacks.GetByID(ctx, "svc-1")
	require.NoError(t, err)
	assert.Equal(t, on.ErrorRate, got.ErrorRate)

	list, err := s.Stacks.ListWithErrorRate(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "svc-1", list[0].ID)

	off.ErrorRate.Enabled, on.ErrorRate.Enabled = true, false
	require.NoError(t, s.Stacks.Update(ctx, off))
	require.NoError(t, s.Stacks.Update(ctx, on))
	list, err = s.Stacks.ListWithErrorRate(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "svc-2", list[0].ID)
}
