package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
)

func TestAutomationCursorsRepo_Get_MissingRow_ReturnsNotOK(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, ok, err := s.AutomationCursors.Get(context.Background(), "a1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAutomationCursorsRepo_Set_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	want := automations.Cursor{CreatedAt: time.Now().Truncate(time.Second).UTC(), ID: "ev-1"}
	require.NoError(t, s.AutomationCursors.Set(context.Background(), "a1", want))

	got, ok, err := s.AutomationCursors.Get(context.Background(), "a1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, want.CreatedAt, got.CreatedAt)
	assert.Equal(t, want.ID, got.ID)
}

func TestAutomationCursorsRepo_Set_Overwrites(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	first := automations.Cursor{CreatedAt: time.Now().Truncate(time.Second).UTC(), ID: "ev-1"}
	require.NoError(t, s.AutomationCursors.Set(context.Background(), "a1", first))

	second := automations.Cursor{CreatedAt: first.CreatedAt.Add(time.Minute), ID: "ev-2"}
	require.NoError(t, s.AutomationCursors.Set(context.Background(), "a1", second))

	got, ok, err := s.AutomationCursors.Get(context.Background(), "a1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "ev-2", got.ID)
	assert.Equal(t, second.CreatedAt, got.CreatedAt)
}

func TestAutomationCursorsRepo_Set_IsolatedPerAutomation(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.AutomationCursors.Set(context.Background(), "a1", automations.Cursor{ID: "ev-1"}))
	require.NoError(t, s.AutomationCursors.Set(context.Background(), "a2", automations.Cursor{ID: "ev-2"}))

	got1, ok, err := s.AutomationCursors.Get(context.Background(), "a1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "ev-1", got1.ID)

	got2, ok, err := s.AutomationCursors.Get(context.Background(), "a2")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "ev-2", got2.ID)
}
