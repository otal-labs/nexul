package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/issuesync"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newIssueSyncStore(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	_, err := s.db.Exec(`
INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at) VALUES ('p-app', 'App', 0, 'workspace-default', 1, 1);
INSERT INTO project_repos (project_id, owner, name, full_name, added_at) VALUES ('p-app', 'acme', 'app', 'acme/app', 1);
INSERT INTO tickets (id, title, body, status, project_id, created_at, updated_at) VALUES ('t-1', 'Crash', '', 'open', 'p-app', 1, 1);
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1);
INSERT INTO user_identities (user_id, provider, provider_user_id, login, created_at) VALUES ('u-alice', 'github', '42', 'Alice-GH', 1);`)
	require.NoError(t, err)
	return s
}

// TestIssueSyncRepo_SyncGoesWithItsRepository: a sync round-trips, upserts on its repository, and leaves when the
// repository is detached from the project.
func TestIssueSyncRepo_SyncGoesWithItsRepository(t *testing.T) {
	t.Parallel()
	s := newIssueSyncStore(t)
	ctx := t.Context()
	_, err := s.IssueSync.GetSync(ctx, "acme", "app")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.Error(t, s.IssueSync.SaveSync(ctx, &issuesync.Sync{ProjectID: "p-app", Owner: "acme", Repo: "other", Label: "nexul", Mode: issuesync.ModeLabel}),
		"a repository the project never attached is refused")

	sync := &issuesync.Sync{ProjectID: "p-app", Owner: "acme", Repo: "app", Label: "nexul", Mode: issuesync.ModeLabel, Enabled: true}
	require.NoError(t, s.IssueSync.SaveSync(ctx, sync))
	swept := time.Unix(1_800_000_000, 0).UTC()
	require.NoError(t, s.IssueSync.MarkSwept(ctx, "acme", "app", swept))
	enabled, err := s.IssueSync.EnabledSyncs(ctx)
	require.NoError(t, err)
	sync.LastSweptAt = &swept
	assert.Equal(t, []*issuesync.Sync{sync}, enabled)

	sync.Enabled, sync.Mode = false, issuesync.ModeAll
	require.NoError(t, s.IssueSync.SaveSync(ctx, sync))
	got, err := s.IssueSync.SyncsOfProject(ctx, "p-app")
	require.NoError(t, err)
	assert.Equal(t, []*issuesync.Sync{sync}, got)

	_, err = s.db.Exec(`DELETE FROM project_repos WHERE owner = 'acme' AND name = 'app'`)
	require.NoError(t, err)
	_, err = s.IssueSync.GetSync(ctx, "acme", "app")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}

// TestIssueSyncRepo_LinksGoWithTheirTicket: one link per issue and per ticket, and a deleted ticket takes its links.
func TestIssueSyncRepo_LinksGoWithTheirTicket(t *testing.T) {
	t.Parallel()
	s := newIssueSyncStore(t)
	ctx := t.Context()
	link := &issuesync.Link{Owner: "acme", Repo: "app", Number: 7, IssueID: 900, TicketID: "t-1", LinkedAt: time.Unix(100, 0).UTC()}
	require.NoError(t, s.IssueSync.SaveLink(ctx, link))
	link.Paused, link.InboundHash, link.IssueUpdatedAt = true, "abc", time.Unix(200, 0).UTC()
	require.NoError(t, s.IssueSync.SaveLink(ctx, link))
	got, err := s.IssueSync.GetLink(ctx, "acme", "app", 7)
	require.NoError(t, err)
	assert.Equal(t, link, got)
	other := *link
	other.Number = 8
	assert.ErrorIs(t, s.IssueSync.SaveLink(ctx, &other), apperrs.ErrConflict, "a ticket follows one issue")

	require.NoError(t, s.IssueSync.SaveCommentLink(ctx, issuesync.CommentLink{CommentID: 55, MessageID: "m-1", TicketID: "t-1"}))
	cl, err := s.IssueSync.GetCommentLink(ctx, 55)
	require.NoError(t, err)
	assert.Equal(t, "m-1", cl.MessageID)

	_, err = s.db.Exec(`DELETE FROM tickets WHERE id = 't-1'`)
	require.NoError(t, err)
	_, err = s.IssueSync.GetLink(ctx, "acme", "app", 7)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.IssueSync.GetCommentLink(ctx, 55)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestIssueSyncRepo_GitHubUserID_MatchesTheLoginIgnoringCase(t *testing.T) {
	t.Parallel()
	s := newIssueSyncStore(t)
	id, ok, err := s.IssueSync.GitHubUserID(t.Context(), "alice-gh")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "u-alice", id)
	_, ok, err = s.IssueSync.GitHubUserID(t.Context(), "nobody")
	require.NoError(t, err)
	assert.False(t, ok)
}
