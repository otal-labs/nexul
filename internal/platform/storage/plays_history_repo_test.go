package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/plays"
)

// historyStore is a store with workspace ws-1 and a play-1 created at version 1 holding "First.".
func historyStore(t *testing.T) (*Store, *plays.Play) {
	t.Helper()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	p := newTestPlay("play-1", "ws-1")
	p.Instructions = "First."
	require.NoError(t, s.Plays.Create(t.Context(), p, plays.Revision{Instructions: &plays.NewVersion{Body: "First.", EditorIDs: []string{"u-1"}}}))
	return s, p
}

func save(t *testing.T, s *Store, p *plays.Play, at time.Time, rev plays.Revision) {
	t.Helper()
	p.UpdatedAt = at
	require.NoError(t, s.Plays.Update(t.Context(), p, rev))
}

func TestMigration0140_ExistingPlaysStartAtVersionOneAndAnUnsavedBuiltInIsTheDefault(t *testing.T) {
	db := migrateBefore(t, "0140")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-seed', 'Seed', 'seed', 1, 1);
INSERT INTO plays (id, workspace_id, label, type, description, instructions, enabled, show_when_stage, excluded_project_ids, builtin_key, created_by, created_at, updated_at) VALUES
    ('builtin-untouched', 'ws-seed', 'Untouched', 'doc', '', 'Seeded text.', 1, NULL, '[]', 'clarify', '', 100, 100),
    ('builtin-saved',     'ws-seed', 'Saved',     'doc', '', 'Edited text.', 1, NULL, '[]', 'audit',   '', 100, 200),
    ('custom',            'ws-seed', 'Custom',    'doc', '', 'My text.',     1, NULL, '[]', '',        'u-1', 100, 100);
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, starter_id, via, state, started_at)
    VALUES ('trail-old', 'ws-seed', 'custom', 'Custom', 'doc', 'd-1', 'p-1', 'u-1', 'web', 'done', 1);
`)
	require.NoError(t, err)
	script, err := migrationFS.ReadFile("migrations/0140_play_history.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0140_play_history", string(script)))
	require.NoError(t, Migrate(db), "every later migration applies on top, as an upgrade would")
	s := New(db, testEncKey)

	for _, tt := range []struct {
		id, body    string
		builtinFlag bool
	}{
		{"builtin-untouched", "Seeded text.", true},
		{"builtin-saved", "Edited text.", false},
		{"custom", "My text.", false},
	} {
		h, err := s.Plays.ListHistory(t.Context(), tt.id)
		require.NoError(t, err, tt.id)
		require.Len(t, h.Versions, 1, tt.id)
		assert.Equal(t, []any{1, tt.builtinFlag}, []any{h.Versions[0].Version, h.Versions[0].BuiltinDefault}, tt.id)
		v, err := s.Plays.GetVersion(t.Context(), tt.id, 1)
		require.NoError(t, err, tt.id)
		assert.Equal(t, tt.body, v.Body, tt.id)
		assert.Empty(t, h.Changes, tt.id)
	}
	trail, err := s.PlayTrails.GetTrail(t.Context(), "trail-old")
	require.NoError(t, err)
	assert.Nil(t, trail.InstructionsVersion, "a run from before history names no version")
}

func TestPlaysRepo_Update_WritesTheNextVersionAndChangesAndListsThemNewestFirst(t *testing.T) {
	s, p := historyStore(t)
	actor := plays.Actor{Kind: plays.ActorKindAutomation, AutomationID: "auto-1", AutomationName: "Nightly"}
	p.Instructions, p.Label = "Second.", "Renamed"
	save(t, s, p, p.CreatedAt.Add(time.Hour), plays.Revision{
		Instructions: &plays.NewVersion{Body: "Second.", EditorIDs: []string{}, Via: "Nightly"},
		Changes:      []plays.Change{{ID: "c-1", Field: plays.ChangedLabel, From: "Doc play", To: "Renamed", Actor: actor, CreatedAt: p.UpdatedAt}},
	})

	h, err := s.Plays.ListHistory(t.Context(), p.ID)
	require.NoError(t, err)
	require.Len(t, h.Versions, 2)
	assert.Equal(t, []any{2, "Nightly", []string{}}, []any{h.Versions[0].Version, h.Versions[0].Via, h.Versions[0].EditorIDs})
	assert.Equal(t, []any{1, []string{"u-1"}}, []any{h.Versions[1].Version, h.Versions[1].EditorIDs})
	require.Len(t, h.Changes, 1)
	assert.Equal(t, []any{"label", "Doc play", "Renamed", actor}, []any{h.Changes[0].Field, h.Changes[0].From, h.Changes[0].To, h.Changes[0].Actor})

	old, err := s.Plays.GetVersion(t.Context(), p.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "First.", old.Body)
	_, err = s.Plays.GetVersion(t.Context(), p.ID, 3)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlaysRepo_Update_AFailedSaveWritesNoHistory(t *testing.T) {
	s, p := historyStore(t)
	other := newTestPlay("play-2", "ws-1")
	other.Label = "Other"
	require.NoError(t, s.Plays.Create(t.Context(), other, plays.Revision{}))

	p.Label, p.Instructions = "other", "Second."
	p.UpdatedAt = p.CreatedAt.Add(time.Hour)
	err := s.Plays.Update(t.Context(), p, plays.Revision{
		Instructions: &plays.NewVersion{Body: "Second."},
		Changes:      []plays.Change{{ID: "c-1", Field: plays.ChangedLabel, From: "Doc play", To: "other", CreatedAt: p.UpdatedAt}},
	})
	require.ErrorIs(t, err, apperrs.ErrConflict)

	h, err := s.Plays.ListHistory(t.Context(), p.ID)
	require.NoError(t, err)
	assert.Len(t, h.Versions, 1)
	assert.Empty(t, h.Changes)
}

func TestPlaysRepo_Update_KeepsTheNewestTenVersionsAndHundredChanges(t *testing.T) {
	s, p := historyStore(t)
	for i := 2; i <= 12; i++ {
		save(t, s, p, p.CreatedAt.Add(time.Duration(i)*time.Minute), plays.Revision{Instructions: &plays.NewVersion{Body: fmt.Sprintf("v%d", i)}})
	}
	for i := range 105 {
		at := p.CreatedAt.Add(time.Duration(i) * time.Second)
		save(t, s, p, at, plays.Revision{Changes: []plays.Change{{ID: fmt.Sprintf("c-%03d", i), Field: plays.ChangedLabel, From: "a", To: "b", CreatedAt: at}}})
	}

	h, err := s.Plays.ListHistory(t.Context(), p.ID)
	require.NoError(t, err)
	require.Len(t, h.Versions, plays.KeptInstructionVersions)
	assert.Equal(t, []int{12, 3}, []int{h.Versions[0].Version, h.Versions[9].Version})
	_, err = s.Plays.GetVersion(t.Context(), p.ID, 2)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "version 2 fell past the cap")

	require.Len(t, h.Changes, plays.KeptChanges)
	assert.Equal(t, []string{"c-104", "c-005"}, []string{h.Changes[0].ID, h.Changes[99].ID})
}

func TestPlaysRepo_Update_PruningOnePlayLeavesAnotherPlaysHistory(t *testing.T) {
	s, p := historyStore(t)
	other := newTestPlay("play-2", "ws-1")
	other.Label = "Other"
	require.NoError(t, s.Plays.Create(t.Context(), other, plays.Revision{Instructions: &plays.NewVersion{Body: "Other."}}))
	for i := 2; i <= 12; i++ {
		save(t, s, p, p.CreatedAt.Add(time.Duration(i)*time.Minute), plays.Revision{Instructions: &plays.NewVersion{Body: fmt.Sprintf("v%d", i)}})
	}

	h, err := s.Plays.ListHistory(t.Context(), other.ID)
	require.NoError(t, err)
	assert.Len(t, h.Versions, 1)
}

func TestPlayTrailsRepo_CreateTrail_RecordsTheInstructionsVersionTheRunStartedWith(t *testing.T) {
	s, p := historyStore(t)
	save(t, s, p, p.CreatedAt.Add(time.Hour), plays.Revision{Instructions: &plays.NewVersion{Body: "Second."}})
	bare := newTestPlay("play-2", "ws-1")
	bare.Label = "Bare"
	require.NoError(t, s.Plays.Create(t.Context(), bare, plays.Revision{}))

	for _, tt := range []struct {
		id, playID string
		want       *int
	}{
		{"trail-a", p.ID, new(2)},
		{"trail-b", bare.ID, nil},
	} {
		trail := &plays.Trail{
			ID: tt.id, WorkspaceID: "ws-1", PlayID: tt.playID, PlayLabel: "Play", TargetType: plays.TargetDoc, TargetID: "d-1", ProjectID: "p-1",
			StarterID: "u-1", Via: plays.ViaWeb, State: plays.TrailStarting, StartedAt: time.Now(),
		}
		require.NoError(t, s.PlayTrails.CreateTrail(t.Context(), trail), tt.id)
		assert.Equal(t, tt.want, trail.InstructionsVersion, "the created trail carries it: %s", tt.id)
		got, err := s.PlayTrails.GetTrail(t.Context(), tt.id)
		require.NoError(t, err, tt.id)
		assert.Equal(t, tt.want, got.InstructionsVersion, tt.id)
	}

	save(t, s, p, p.CreatedAt.Add(2*time.Hour), plays.Revision{Instructions: &plays.NewVersion{Body: "Third."}})
	got, err := s.PlayTrails.GetTrail(t.Context(), "trail-a")
	require.NoError(t, err)
	assert.Equal(t, new(2), got.InstructionsVersion, "a later save does not move the version a run used")
}

func TestPlaysRepo_Delete_TakesItsHistoryWithIt(t *testing.T) {
	s, p := historyStore(t)
	save(t, s, p, p.CreatedAt.Add(time.Hour), plays.Revision{Changes: []plays.Change{{ID: "c-1", Field: plays.ChangedLabel, From: "a", To: "b", CreatedAt: p.CreatedAt}}})
	require.NoError(t, s.Plays.Delete(t.Context(), p.ID))

	var n int
	require.NoError(t, s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM play_instruction_versions WHERE play_id = ?1) + (SELECT COUNT(*) FROM play_changes WHERE play_id = ?1)`, p.ID).Scan(&n))
	assert.Zero(t, n)
}
