package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/plays"
)

func newTestPlay(id, workspaceID string) *plays.Play {
	now := time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)
	return &plays.Play{
		ID: id, WorkspaceID: workspaceID, Label: "Doc play", Type: plays.TypeDoc,
		Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
}

func TestPlaysRepo_Create_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	stage := plays.StageProgress
	want := newTestPlay("play-1", "ws-1")
	want.Label = "Fix with AI"
	want.Type = plays.TypeTicket
	want.ShowWhenStage = &stage
	want.ExcludedProjectIDs = []string{"proj-2", "proj-1"} // repo persists the order the caller (usecase) hands it
	want.CreatedBy = "user-1"
	require.NoError(t, s.Plays.Create(context.Background(), want))

	got, err := s.Plays.Get(context.Background(), "play-1")
	require.NoError(t, err)
	assert.Equal(t, "Fix with AI", got.Label)
	assert.Equal(t, plays.TypeTicket, got.Type)
	require.NotNil(t, got.ShowWhenStage)
	assert.Equal(t, plays.StageProgress, *got.ShowWhenStage)
	assert.Equal(t, []string{"proj-2", "proj-1"}, got.ExcludedProjectIDs)
	assert.Equal(t, "user-1", got.CreatedBy)
	assert.True(t, got.Enabled)
}

func TestPlaysRepo_Create_DocPlay_HasNoStage(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Plays.Create(context.Background(), newTestPlay("play-1", "ws-1")))

	got, err := s.Plays.Get(context.Background(), "play-1")
	require.NoError(t, err)
	assert.Nil(t, got.ShowWhenStage)
	assert.Equal(t, []string{}, got.ExcludedProjectIDs)
}

func TestPlaysRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	_, err := s.Plays.Get(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlaysRepo_Create_UnknownWorkspace_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Plays.Create(context.Background(), newTestPlay("play-1", "missing-workspace"))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestPlaysRepo_List_ScopesToWorkspaceAndSortsByLabel(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-2", "Other Co")))
	zeta := newTestPlay("play-zeta", "ws-1")
	zeta.Label = "Zeta"
	alpha := newTestPlay("play-alpha", "ws-1")
	alpha.Label = "Alpha"
	other := newTestPlay("play-other", "ws-2")
	require.NoError(t, s.Plays.Create(context.Background(), zeta))
	require.NoError(t, s.Plays.Create(context.Background(), alpha))
	require.NoError(t, s.Plays.Create(context.Background(), other))

	got, err := s.Plays.List(context.Background(), "ws-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Alpha", got[0].Label)
	assert.Equal(t, "Zeta", got[1].Label)
}

func TestPlaysRepo_Update(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	p := newTestPlay("play-1", "ws-1")
	require.NoError(t, s.Plays.Create(context.Background(), p))

	p.Label = "Renamed"
	p.Enabled = false
	p.ExcludedProjectIDs = []string{"proj-1"}
	p.UpdatedAt = time.Now()
	require.NoError(t, s.Plays.Update(context.Background(), p))

	got, err := s.Plays.Get(context.Background(), "play-1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Label)
	assert.False(t, got.Enabled)
	assert.Equal(t, []string{"proj-1"}, got.ExcludedProjectIDs)
}

func TestPlaysRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Plays.Update(context.Background(), newTestPlay("missing", "ws-1"))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlaysRepo_Delete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Plays.Create(context.Background(), newTestPlay("play-1", "ws-1")))

	require.NoError(t, s.Plays.Delete(context.Background(), "play-1"))
	_, err := s.Plays.Get(context.Background(), "play-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlaysRepo_Migration_SeedsDefaultWorkspaceWithTheDefaultPlays(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	got, err := s.Plays.List(context.Background(), "workspace-default")
	require.NoError(t, err)
	require.Len(t, got, 4)
	byLabel := map[string]*plays.Play{}
	for _, p := range got {
		byLabel[p.Label] = p
	}
	progress, testingStage := plays.StageProgress, plays.StageTesting
	tests := []struct {
		label    string
		wantType plays.Type
		wantShow *plays.Stage
		mention  string
	}{
		{"Fix with AI", plays.TypeTicket, &progress, "`link_pr`"},
		{"To tickets via AI", plays.TypeDoc, nil, "ticket_create"},
		{"Interview", plays.TypeInterview, nil, "`kind` `interview`"},
		{"Test with AI", plays.TypeTicket, &testingStage, "ticket_test_report"},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			p := byLabel[tt.label]
			require.NotNil(t, p)
			assert.Equal(t, tt.wantType, p.Type)
			assert.Equal(t, tt.wantShow, p.ShowWhenStage)
			assert.Contains(t, p.Instructions, tt.mention)
			assert.True(t, p.Enabled)
		})
	}
}

func TestPlaysRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	err := s.Plays.Delete(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
