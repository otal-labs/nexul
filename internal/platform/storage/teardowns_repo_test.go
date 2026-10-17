package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/runner"
)

func teardownFrame(slug, strategy string) runner.Frame {
	return runner.Frame{Type: runner.FrameStackTeardown, StackSlug: slug, Strategy: strategy}
}

func TestTeardownsRepo_List_ReturnsTheMachinesOwnInTheOrderSaved(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	require.NoError(t, s.Teardowns.Save(ctx, "host1", teardownFrame("shop", "compose")))
	require.NoError(t, s.Teardowns.Save(ctx, "host2", teardownFrame("other", "run")))
	require.NoError(t, s.Teardowns.Save(ctx, "host1", teardownFrame("edge", "run")))

	got, err := s.Teardowns.List(ctx, "host1")
	require.NoError(t, err)
	assert.Equal(t, []runner.Frame{teardownFrame("shop", "compose"), teardownFrame("edge", "run")}, got)

	none, err := s.Teardowns.List(ctx, "host3")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestTeardownsRepo_Save_TheSameSlugReplacesItAndMovesLast(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	require.NoError(t, s.Teardowns.Save(ctx, "host1", teardownFrame("edge", "run")))
	require.NoError(t, s.Teardowns.Save(ctx, "host1", teardownFrame("shop", "run")))
	require.NoError(t, s.Teardowns.Save(ctx, "host1", teardownFrame("edge", "compose")))

	got, err := s.Teardowns.List(ctx, "host1")
	require.NoError(t, err)
	assert.Equal(t, []runner.Frame{teardownFrame("shop", "run"), teardownFrame("edge", "compose")}, got)
}

func TestTeardownsRepo_Delete_DropsOneAndIgnoresAMissingRow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)
	require.NoError(t, s.Teardowns.Save(ctx, "host1", teardownFrame("edge", "run")))
	require.NoError(t, s.Teardowns.Save(ctx, "host2", teardownFrame("edge", "run")))

	require.NoError(t, s.Teardowns.Delete(ctx, "host1", "edge"))
	require.NoError(t, s.Teardowns.Delete(ctx, "host1", "edge"), "a row already gone is not an error")

	left, err := s.Teardowns.List(ctx, "host1")
	require.NoError(t, err)
	assert.Empty(t, left)
	other, err := s.Teardowns.List(ctx, "host2")
	require.NoError(t, err)
	assert.Len(t, other, 1, "the same slug on another machine stays")
}
