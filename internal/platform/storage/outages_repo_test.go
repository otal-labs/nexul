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

func newTestOutage(id, stackID string) *deploy.Outage {
	return &deploy.Outage{
		ID: id, StackID: stackID, Service: "web", Container: "stack-1-web-1",
		StartedAt: time.Date(2026, 10, 1, 10, 42, 0, 0, time.UTC), ExitCode: 137, OOM: true,
	}
}

func outageEvent(id, topic string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: id, Topic: topic, Payload: deploy.OutageEvent{StackID: "stack-1", OutageID: "out-1"}}
}

func outboxTopicsOf(t *testing.T, s *Store) []string {
	t.Helper()
	rows, err := s.db.Query(`SELECT topic FROM outbox ORDER BY created_at, id`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	var topics []string
	for rows.Next() {
		var topic string
		require.NoError(t, rows.Scan(&topic))
		topics = append(topics, topic)
	}
	require.NoError(t, rows.Err())
	return topics
}

func TestOutagesRepo_OpenAndGetOpen_RoundTripWithItsEvent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-1", "stack-1"), outageEvent("evt-1", deploy.TopicContainerStopped)))

	got, err := s.Outages.GetOpen(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.Equal(t, newTestOutage("out-1", "stack-1"), got)
	assert.Equal(t, []string{deploy.TopicContainerStopped}, outboxTopicsOf(t, s), "the event is written with the outage")
}

func TestOutagesRepo_Open_ASecondOpenOutageForAServiceConflicts(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-1", "stack-1")))

	err := s.Outages.Open(ctx, newTestOutage("out-2", "stack-1"), outageEvent("evt-2", deploy.TopicContainerStopped))
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Empty(t, outboxTopicsOf(t, s), "a refused outage leaves no event behind")
}

func TestOutagesRepo_Open_AnUnknownStackFails(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.Error(t, s.Outages.Open(context.Background(), newTestOutage("out-1", "nope")))
}

func TestOutagesRepo_GetOpen_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Outages.GetOpen(context.Background(), "stack-1", "web")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestOutagesRepo_Update_RecordsAnotherCrash(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	o := newTestOutage("out-1", "stack-1")
	require.NoError(t, s.Outages.Open(ctx, o))

	o.Container, o.ExitCode, o.OOM, o.RestartCount = "stack-1-web-2", 1, false, 4
	require.NoError(t, s.Outages.Update(ctx, o))

	got, err := s.Outages.GetOpen(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.Equal(t, o, got)
}

func TestOutagesRepo_Close_EndsItAndFreesTheService(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-1", "stack-1")))
	end := time.Date(2026, 10, 1, 10, 50, 0, 0, time.UTC)

	require.NoError(t, s.Outages.Close(ctx, "out-1", end, outageEvent("evt-1", deploy.TopicContainerRecovered)))

	_, err := s.Outages.GetOpen(ctx, "stack-1", "web")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	got, err := s.Outages.LatestEnd(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.Equal(t, end, got)
	assert.Equal(t, []string{deploy.TopicContainerRecovered}, outboxTopicsOf(t, s))
	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-2", "stack-1")), "the service can go down again")
}

func TestOutagesRepo_Close_AnEndedOrUnknownOutageIsNotFoundAndWritesNoEvent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-1", "stack-1")))
	require.NoError(t, s.Outages.Close(ctx, "out-1", time.Unix(100, 0)))

	for _, id := range []string{"out-1", "nope"} {
		err := s.Outages.Close(ctx, id, time.Unix(200, 0), outageEvent("evt-"+id, deploy.TopicContainerRecovered))
		require.ErrorIs(t, err, apperrs.ErrNotFound, id)
	}
	assert.Empty(t, outboxTopicsOf(t, s))
}

func TestOutagesRepo_ListOpenByStack_OnlyThatStacksOpenOnes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	seedStackForContainers(t, s, "stack-2")
	worker := newTestOutage("out-2", "stack-1")
	worker.Service = "worker"
	worker.StartedAt = worker.StartedAt.Add(-time.Hour)
	for _, o := range []*deploy.Outage{newTestOutage("out-1", "stack-1"), worker, newTestOutage("out-3", "stack-2"), newTestOutage("out-4", "stack-1")} {
		if o.ID == "out-4" {
			o.Service = "done"
		}
		require.NoError(t, s.Outages.Open(ctx, o))
	}
	require.NoError(t, s.Outages.Close(ctx, "out-4", time.Unix(100, 0)))

	got, err := s.Outages.ListOpenByStack(ctx, "stack-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"worker", "web"}, []string{got[0].Service, got[1].Service}, "oldest first")

	none, err := s.Outages.ListOpenByStack(ctx, "stack-9")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestOutagesRepo_LatestEnd(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")

	got, err := s.Outages.LatestEnd(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.True(t, got.IsZero(), "a service that never went down has no end")

	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-1", "stack-1")))
	got, err = s.Outages.LatestEnd(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.True(t, got.IsZero(), "an open outage has not ended")

	require.NoError(t, s.Outages.Close(ctx, "out-1", time.Unix(500, 0)))
	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-2", "stack-1")))
	require.NoError(t, s.Outages.Close(ctx, "out-2", time.Unix(900, 0)))
	got, err = s.Outages.LatestEnd(ctx, "stack-1", "web")
	require.NoError(t, err)
	assert.Equal(t, time.Unix(900, 0).UTC(), got)
}

func TestOutagesRepo_DeletingTheStackRemovesItsOutages(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.Outages.Open(ctx, newTestOutage("out-1", "stack-1")))

	require.NoError(t, s.Stacks.Delete(ctx, "stack-1"))

	got, err := s.Outages.ListOpenByStack(ctx, "stack-1")
	require.NoError(t, err)
	assert.Empty(t, got)
}
