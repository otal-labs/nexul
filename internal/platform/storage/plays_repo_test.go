package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
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
	require.NoError(t, s.Plays.Create(context.Background(), want, plays.Revision{}))

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
	require.NoError(t, s.Plays.Create(context.Background(), newTestPlay("play-1", "ws-1"), plays.Revision{}))

	got, err := s.Plays.Get(context.Background(), "play-1")
	require.NoError(t, err)
	assert.Nil(t, got.ShowWhenStage)
	assert.Equal(t, []string{}, got.ExcludedProjectIDs)
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
	require.NoError(t, s.Plays.Create(context.Background(), zeta, plays.Revision{}))
	require.NoError(t, s.Plays.Create(context.Background(), alpha, plays.Revision{}))
	require.NoError(t, s.Plays.Create(context.Background(), other, plays.Revision{}))

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
	require.NoError(t, s.Plays.Create(context.Background(), p, plays.Revision{}))

	p.Label = "Renamed"
	p.Enabled = false
	p.ExcludedProjectIDs = []string{"proj-1"}
	p.UpdatedAt = time.Now()
	require.NoError(t, s.Plays.Update(context.Background(), p, plays.Revision{}))

	got, err := s.Plays.Get(context.Background(), "play-1")
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Label)
	assert.False(t, got.Enabled)
	assert.Equal(t, []string{"proj-1"}, got.ExcludedProjectIDs)
}

func TestPlaysRepo_Delete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Plays.Create(context.Background(), newTestPlay("play-1", "ws-1"), plays.Revision{}))

	require.NoError(t, s.Plays.Delete(context.Background(), "play-1"))
	_, err := s.Plays.Get(context.Background(), "play-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlaysRepo_Migration_SeedsDefaultWorkspaceWithTheDefaultPlays(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	got, err := s.Plays.List(context.Background(), "workspace-default")
	require.NoError(t, err)
	require.Len(t, got, 10)
	byLabel := map[string]*plays.Play{}
	for _, p := range got {
		byLabel[p.Label] = p
	}
	progress, testingStage, done := plays.StageProgress, plays.StageTesting, plays.StageDone
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
		{"Draft interview", plays.TypeInterview, nil, "`trail_id`"},
		{"Clarify via AI", plays.TypeDoc, nil, "`no_gaps`"},
		{"Audit via AI", plays.TypeInterview, nil, "doc_create"},
		{"Decisions check", plays.TypeTicket, &done, "`decisions_log`"},
		{"Investigate a crash", plays.TypeStack, nil, "stack_get"},
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

func TestPlaysRepo_CreateAndUpdate_TakenLabel_ReturnsConflict(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-2", "Globex")))
	require.NoError(t, s.Plays.Create(ctx, newTestPlay("play-1", "ws-1"), plays.Revision{}))
	other := newTestPlay("play-2", "ws-1")
	other.Label = "Other"
	require.NoError(t, s.Plays.Create(ctx, other, plays.Revision{}))

	taken := newTestPlay("play-3", "ws-1")
	taken.Label = "DOC PLAY"
	require.ErrorIs(t, s.Plays.Create(ctx, taken, plays.Revision{}), apperrs.ErrConflict)
	other.Label = "doc play"
	require.ErrorIs(t, s.Plays.Update(ctx, other, plays.Revision{}), apperrs.ErrConflict)
	require.NoError(t, s.Plays.Create(ctx, newTestPlay("play-4", "ws-2"), plays.Revision{}), "another workspace's labels are its own")
}

func TestPlaysRepo_GetByLabel_IgnoresCaseWithinTheWorkspace(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-1", "Acme")))
	require.NoError(t, s.Workspaces.Create(ctx, newTestWorkspace("ws-2", "Globex")))
	require.NoError(t, s.Plays.Create(ctx, newTestPlay("play-1", "ws-1"), plays.Revision{}))

	got, err := s.Plays.GetByLabel(ctx, "ws-1", "DOC play")
	require.NoError(t, err)
	assert.Equal(t, "play-1", got.ID)
	_, err = s.Plays.GetByLabel(ctx, "ws-2", "Doc play")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlaysRepo_AutomationNamesByLabel_ReadsOnlyQuotedLabelsInActiveOrPendingCode(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	automation := func(id, name, workspaceID string, codes ...string) {
		require.NoError(t, s.Automations.Create(ctx, &automations.Automation{
			ID: id, Name: name, WorkspaceID: workspaceID, Kind: automations.KindCustom, Scopes: []string{"x"},
			TokenHash: "h-" + id, CreatedAt: now, UpdatedAt: now,
		}))
		for i, code := range codes {
			v := &automations.Version{ID: fmt.Sprintf("%s-v%d", id, i+1), AutomationID: id, Code: code, PusherID: "u", CreatedAt: now}
			if i == 0 {
				v.Status = automations.VersionActive
				require.NoError(t, s.AutomationVersions.InsertActive(ctx, v))
				continue
			}
			v.Status = automations.VersionPending
			require.NoError(t, s.AutomationVersions.ReplacePending(ctx, v))
		}
	}
	automation("a-double", "Deploy notifier", "ws-1", `await runPlay("Fix with AI", t); await runPlay('Fix with AI', t)`)
	automation("a-single", "Nightly triage", "ws-1", `runPlay('fix WITH ai', ticket)`)
	automation("a-backtick", "Pending caller", "ws-1", `log("nothing yet")`, "runPlay(`FIX WITH AI`, ticket)")
	automation("a-old", "Old caller", "ws-1", `runPlay("Fix with AI", t)`, `runPlay("Test with AI", t)`)
	_, err := s.AutomationVersions.Activate(ctx, "a-old", "a-old-v2")
	require.NoError(t, err)
	automation("a-prose", "Mentions it in prose", "ws-1", `log("Fix with AI later")`)
	automation("a-elsewhere", "Other workspace", "ws-2", `runPlay("Fix with AI", t)`)

	names, err := s.Plays.AutomationNamesByLabel(ctx, "ws-1", "Fix with AI")
	require.NoError(t, err)
	assert.Equal(t, []string{"Deploy notifier", "Nightly triage", "Pending caller"}, names,
		"each quote style and case counts once per automation; an inactive version, a longer string, and another workspace do not")

	names, err = s.Plays.AutomationNamesByLabel(ctx, "ws-1", "Write release note")
	require.NoError(t, err)
	assert.Empty(t, names)
}
