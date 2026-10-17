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
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, deploy.SourceProxy, []string{"stack-1"}, minute0))

	got, err := s.RequestMinutes.Since(ctx, "stack-1", minute0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Nil(t, got[0].Requests, "missing is not zero")
	assert.Nil(t, got[0].Errors)

	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 4, Errors: 1}}))
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, deploy.SourceProxy, []string{"stack-1"}, minute0))
	got, err = s.RequestMinutes.Since(ctx, "stack-1", minute0)
	require.NoError(t, err)
	assert.EqualValues(t, 4, *got[0].Requests, "marking a counted minute missing changes nothing")
}

func TestRequestMinutesRepo_SetReplacesAMinuteAndFillsAMissingOne(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, deploy.SourceProxy, []string{"stack-1"}, minute0.Add(time.Minute)))
	require.NoError(t, s.RequestMinutes.Set(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 56, Errors: 4}}))
	require.NoError(t, s.RequestMinutes.Set(ctx, []deploy.RequestAdd{
		{StackID: "stack-1", Minute: minute0, Requests: 56, Errors: 4},
		{StackID: "stack-1", Minute: minute0.Add(time.Minute), Requests: 9},
	}))
	require.NoError(t, s.RequestMinutes.Set(ctx, nil))
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, deploy.SourceProxy, []string{"stack-1"}, minute0))

	got, err := s.RequestMinutes.Since(ctx, "stack-1", minute0)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.EqualValues(t, 56, *got[0].Requests, "setting the same minute twice does not double it")
	assert.EqualValues(t, 4, *got[0].Errors)
	assert.EqualValues(t, 9, *got[1].Requests)

	require.NoError(t, s.RequestMinutes.Set(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 3}}))
	got, err = s.RequestMinutes.Since(ctx, "stack-1", minute0)
	require.NoError(t, err)
	assert.EqualValues(t, 3, *got[0].Requests, "a newer total replaces the stored one")
	assert.Error(t, s.RequestMinutes.Set(ctx, []deploy.RequestAdd{{StackID: "gone", Minute: minute0}}), "a minute of a stack that is gone is refused")
}

func TestRequestMinutesRepo_PruneDropsOlderMinutesAndEmptyInputsDoNothing(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.RequestMinutes.Add(ctx, nil))
	require.NoError(t, s.RequestMinutes.MarkMissing(ctx, deploy.SourceProxy, nil, minute0))
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

func TestRequestMinutesRepo_ATunnelSetKeepsTheProxyCountOfTheSameMinute(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	seedStackForContainers(t, s, "stack-1")
	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 10, Errors: 2}}))
	require.NoError(t, s.RequestMinutes.Set(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 56, Errors: 4}}))
	require.NoError(t, s.RequestMinutes.Set(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 60, Errors: 5}}))
	require.NoError(t, s.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 1, Errors: 1}}))

	got, err := s.RequestMinutes.Since(ctx, "stack-1", minute0)

	require.NoError(t, err)
	require.Len(t, got, 1, "the sources read as one row a minute")
	assert.EqualValues(t, 71, *got[0].Requests, "60 from the tunnel, replaced once, and 11 from the proxy, summed")
	assert.EqualValues(t, 8, *got[0].Errors)
}

func TestRequestMinutesRepo_AMinuteIsMissingOnlyWhenEverySourceIsMissing(t *testing.T) {
	t.Parallel()
	mark := func(source deploy.RequestSource) func(*testing.T, *Store) {
		return func(t *testing.T, s *Store) {
			require.NoError(t, s.RequestMinutes.MarkMissing(t.Context(), source, []string{"stack-1"}, minute0))
		}
	}
	add := func(t *testing.T, s *Store) {
		require.NoError(t, s.RequestMinutes.Add(t.Context(), []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0, Requests: 7}}))
	}
	setZero := func(t *testing.T, s *Store) {
		require.NoError(t, s.RequestMinutes.Set(t.Context(), []deploy.RequestAdd{{StackID: "stack-1", Minute: minute0}}))
	}
	tests := []struct {
		name  string
		steps []func(*testing.T, *Store)
		want  *int64
	}{
		{"both sources missing", []func(*testing.T, *Store){mark(deploy.SourceProxy), mark(deploy.SourceTunnel)}, nil},
		{"the tunnel missing and the proxy counted", []func(*testing.T, *Store){mark(deploy.SourceTunnel), add}, new(int64(7))},
		{"the proxy missing and the tunnel counted at zero", []func(*testing.T, *Store){mark(deploy.SourceProxy), setZero}, new(int64(0))},
		{"marking a counted source missing changes nothing", []func(*testing.T, *Store){add, mark(deploy.SourceProxy), mark(deploy.SourceTunnel)}, new(int64(7))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t)
			seedStackForContainers(t, s, "stack-1")
			for _, step := range tt.steps {
				step(t, s)
			}

			got, err := s.RequestMinutes.Since(t.Context(), "stack-1", minute0)

			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, tt.want, got[0].Requests)
			assert.Equal(t, tt.want == nil, got[0].Errors == nil)
		})
	}
}
