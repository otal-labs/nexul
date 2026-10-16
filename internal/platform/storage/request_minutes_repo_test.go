package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
)

var minute0 = time.Date(2026, 10, 16, 10, 42, 0, 0, time.UTC)

func TestRequestMinutesRepo_AddSumsIntoOneMinuteAndSinceReadsThemInOrder(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{
		{StackID: "stack-1", Minute: minute0.Add(time.Minute), Requests: 1},
		{StackID: "stack-1", Minute: minute0, Requests: 10, Errors: 2},
	}))
	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 5, Errors: 1}}))

	got, err := s.RequestMinutes.Since(ctx, "stack-1", minute0)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, minute0, got[0].Minute)
	assert.EqualValues(t, 15, *got[0].Requests)
	assert.EqualValues(t, 3, *got[0].Errors)
	assert.EqualValues(t, 1, *got[1].Requests)
	later, err := s.RequestMinutes.Since(ctx, "stack-1", minute0.Add(time.Minute))
	require.NoError(t, err)
	assert.Len(t, later, 1, "Since starts at the minute it is given")
}

func TestRequestMinutesRepo_AMissingMinuteHasNoCountsAndIsFilledByTheNextAdd(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, []string{"stack-1"}, minute0))

	got, err := s.RequestMinutes.Since(ctx, "stack-1", minute0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Nil(t, got[0].Requests, "missing is not zero")
	assert.Nil(t, got[0].Errors)

	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 4, Errors: 1}}))
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, []string{"stack-1"}, minute0))
	got, err = s.RequestMinutes.Since(ctx, "stack-1", minute0)
	require.NoError(t, err)
	assert.EqualValues(t, 4, *got[0].Requests, "marking a counted minute missing changes nothing")
}

func TestRequestMinutesRepo_PruneDropsOlderMinutesAndEmptyInputsDoNothing(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.RequestMinutes.Add(ctx, nil))
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, nil, minute0))
	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{
		{StackID: "stack-1", Minute: minute0.Add(-25 * time.Hour), Requests: 1},
		{StackID: "stack-1", Minute: minute0, Requests: 2},
	}))

	require.NoError(t, s.RequestMinutes.Prune(ctx, minute0.Add(-24*time.Hour)))

	got, err := s.RequestMinutes.Since(ctx, "stack-1", minute0.Add(-48*time.Hour))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, minute0, got[0].Minute)
}

func TestRequestMinutesRepo_ADeletedStackTakesItsMinutesWithIt(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 2}}))
	_, err := s.db.Exec(`DELETE FROM stacks WHERE id = 'stack-1'`)
	require.NoError(t, err)

	got, err := s.RequestMinutes.Since(ctx, "stack-1", minute0)
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Error(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 1}}), "a minute of a stack that is gone is refused")
}
