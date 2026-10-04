package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TestMigration0067_SourcesAndDraftsOutliveTheMemoryButNotTheProject upgrades a populated database, then walks the rules.
func TestMigration0067_SourcesAndDraftsOutliveTheMemoryButNotTheProject(t *testing.T) {
	db := migrateBefore(t, "0067")
	_, err := db.Exec(`
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES
    ('p-web', 'Web', 'WEB', 0, 'workspace-default', 0, 0),
    ('p-old', 'Old', 'OLD', 1, 'workspace-default', 0, 0);
INSERT INTO memories (id, workspace_id, project_id, kind, title, body, always_included, created_at, updated_at) VALUES
    ('m-interview', 'workspace-default', 'p-web', 'interview', 'Interview', '', 1, 0, 0);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0067 applies on top, as an upgrade would")

	ctx := t.Context()
	repo := New(db, testEncKey).Memories
	at := time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC)
	source := func(id, kind, ref string) *memories.InterviewSource {
		return &memories.InterviewSource{ID: id, WorkspaceID: "workspace-default", ProjectID: "p-web", Kind: kind, Ref: ref, Stance: memories.StanceFollow, AddedBy: "u-1", AddedAt: at, UpdatedAt: at}
	}
	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: memories.TopicSourceAdded, Payload: memories.SourceEvent{ProjectID: "p-web"}}
	require.NoError(t, repo.InsertSource(ctx, source("s-1", memories.SourcePath, "docs"), evt))
	require.NoError(t, repo.InsertSource(ctx, source("s-2", memories.SourceProject, "p-old")))
	require.ErrorIs(t, repo.InsertSource(ctx, source("s-3", memories.SourcePath, "docs")), apperrs.ErrConflict, "one project points at a ref once")
	text := source("s-4", memories.SourceText, "")
	text.Label, text.Body = "Wiki", "Squash merge."
	require.NoError(t, repo.InsertSource(ctx, text))
	text.ID = "s-5"
	require.NoError(t, repo.InsertSource(ctx, text), "pasted text may repeat")

	var outboxRows int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE id = 'evt-1'`).Scan(&outboxRows))
	assert.Equal(t, 1, outboxRows, "the event is written with the source")

	n, err := repo.CountSources(ctx, "p-web")
	require.NoError(t, err)
	assert.Equal(t, 4, n)

	got, err := repo.GetSource(ctx, "s-4")
	require.NoError(t, err)
	assert.Equal(t, "Squash merge.", got.Body)
	got.Stance, got.Label, got.UpdatedAt = memories.StanceQuestion, "Team wiki", at.Add(time.Hour)
	require.NoError(t, repo.UpdateSource(ctx, got))
	got, err = repo.GetSource(ctx, "s-4")
	require.NoError(t, err)
	assert.Equal(t, memories.StanceQuestion, got.Stance)
	assert.Equal(t, "Team wiki", got.Label)
	assert.Equal(t, at.Add(time.Hour), got.UpdatedAt)

	require.NoError(t, repo.DeleteSource(ctx, "s-5"))
	require.ErrorIs(t, repo.DeleteSource(ctx, "s-5"), apperrs.ErrNotFound)
	require.ErrorIs(t, repo.UpdateSource(ctx, &memories.InterviewSource{ID: "s-5"}), apperrs.ErrNotFound)
	_, err = repo.GetSource(ctx, "s-5")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	draft := &memories.InterviewDraft{
		ID: "d-1", WorkspaceID: "workspace-default", ProjectID: "p-web", Question: "Tests", Selected: []string{"Unit"},
		SourceIDs: []string{"s-1"}, Where: "docs/testing.md", TrailID: "trail-1", DraftedBy: "u-1", DraftedAt: at,
	}
	require.NoError(t, repo.SaveDrafts(ctx, []*memories.InterviewDraft{draft}))
	redraft := *draft
	redraft.ID, redraft.Selected, redraft.Text = "d-2", []string{}, "Integration"
	require.NoError(t, repo.SaveDrafts(ctx, []*memories.InterviewDraft{&redraft}))
	ds, err := repo.ListDrafts(ctx, "p-web")
	require.NoError(t, err)
	require.Len(t, ds, 1, "one draft per question; a later one replaces it")
	assert.Equal(t, "d-2", ds[0].ID)
	assert.Equal(t, "Integration", ds[0].Text)
	assert.Equal(t, []string{"s-1"}, ds[0].SourceIDs)
	_, err = repo.GetDraft(ctx, "d-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	other := redraft
	other.ID, other.Question = "d-3", "Stack"
	require.NoError(t, repo.SaveDrafts(ctx, []*memories.InterviewDraft{&other}))
	require.NoError(t, repo.DeleteDraft(ctx, "d-3"))

	_, err = db.Exec(`DELETE FROM projects WHERE id = 'p-old'`)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM memories WHERE id = 'm-interview'`)
	require.NoError(t, err)
	srcs, err := repo.ListSources(ctx, "p-web")
	require.NoError(t, err)
	assert.Equal(t, []string{"s-1", "s-2", "s-4"}, []string{srcs[0].ID, srcs[1].ID, srcs[2].ID}, "a deleted ref and a deleted memory keep the sources")
	ds, err = repo.ListDrafts(ctx, "p-web")
	require.NoError(t, err)
	assert.Len(t, ds, 1, "deleting the interview memory keeps the drafts")

	_, err = db.Exec(`DELETE FROM projects WHERE id = 'p-web'`)
	require.NoError(t, err)
	srcs, err = repo.ListSources(ctx, "p-web")
	require.NoError(t, err)
	assert.Empty(t, srcs, "deleting the project removes its sources")
	ds, err = repo.ListDrafts(ctx, "p-web")
	require.NoError(t, err)
	assert.Empty(t, ds, "and its drafts")
	require.ErrorIs(t, repo.DeleteDraft(ctx, "d-2"), apperrs.ErrNotFound)
}
