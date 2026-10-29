package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
)

// TestPlayRuns_ShowWhatTheTrailByTrailCheckShowed: trail_list with play_id alone pages, for every kind of viewer, exactly
// the runs a single trail read lets them open, newest first, and a play they can't see is not found.
func TestPlayRuns_ShowWhatTheTrailByTrailCheckShowed(t *testing.T) {
	t.Parallel()
	f := listParityFixture(t)
	_, err := f.db.Exec(`
INSERT INTO plays (id, workspace_id, label, type, created_at, updated_at) VALUES ('play-runs', 'workspace-default', 'Check the runbook', 'ticket', 1, 1);
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, starter_id, via, state, started_at)
SELECT 'run-' || t.id, 'workspace-default', 'play-runs', 'Check the runbook', 'ticket', t.id, t.project_id, 'u-owner', 'web', 'done', 1700000000 + t.rowid FROM tickets t;
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, starter_id, via, state, started_at)
SELECT 'run-' || d.id, 'workspace-default', 'play-runs', 'Check the runbook', 'doc', d.id, d.project_id, 'u-owner', 'web', 'waiting', 1700000500 + d.rowid FROM docs d;`)
	require.NoError(t, err)
	runner := plays.NewRunner(plays.RunnerConfig{Plays: f.store.Plays, Trails: f.store.PlayTrails, Perm: f.svc.accessSvc})
	trailList := tool(t, plays.RunMCPTools(runner), "trail_list")
	rows, err := f.db.Query(`SELECT id FROM play_trails WHERE play_id = 'play-runs' ORDER BY started_at DESC, id DESC`)
	require.NoError(t, err)
	var all []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		all = append(all, id)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.Greater(t, len(all), 4)

	for _, user := range []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider, uClient, uUnlisted, uNoRole} {
		t.Run(user, func(t *testing.T) {
			if !f.svc.accessSvc.HasPermission(as(user), user, wsDefault, permissions.PlaysRead, "play", "play-runs") {
				_, err := trailList.Call(as(user), []byte(`{"play_id":"play-runs"}`))
				assert.ErrorIs(t, err, apperrs.ErrNotFound)
				return
			}
			want := []string{}
			for _, id := range all {
				if _, err := runner.GetTrail(as(user), id); err == nil {
					want = append(want, id)
				}
			}
			got := pageThrough(t, f.svc.accessSvc, trailList, user, `"play_id":"play-runs",`)
			if got == nil {
				got = []string{}
			}
			assert.Equal(t, want, got)
		})
	}
}

// TestRows_PlayRunsReadOnlyTheirPage: a page of a play's 3000 runs hands back about one page of rows, and the plays
// list reads the play's health as one row, not its runs.
func TestRows_PlayRunsReadOnlyTheirPage(t *testing.T) {
	t.Parallel()
	f, st, db := newCountedFixtureDB(t)
	_, err := db.Exec(`
INSERT INTO plays (id, workspace_id, label, type, enabled, created_at, updated_at) VALUES ('play-busy', 'workspace-default', 'Busy play', 'ticket', 1, 1, 1);
WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < 3000)
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, starter_id, via, state, started_at)
SELECT printf('run-%05d', i), 'workspace-default', 'play-busy', 'Busy play', 'ticket', ?, 'project-general', 'u-owner', 'web', 'done', 1700000000 + i FROM n`, f.ticket.ID)
	require.NoError(t, err)
	runner := plays.NewRunner(plays.RunnerConfig{Plays: f.store.Plays, Trails: f.store.PlayTrails, Perm: f.svc.accessSvc})

	st.Reset()
	ids, page := callPage(t, f.svc.accessSvc, tool(t, plays.RunMCPTools(runner), "trail_list"), uOwner, `{"play_id":"play-busy"}`)
	assert.Len(t, ids, 50)
	assert.Equal(t, 3000, page.Total)
	assert.Less(t, st.Rows(), int64(250), "rows handed back for one page")

	st.Reset()
	list, err := f.svc.playsSvc.List(as(uOwner), wsDefault)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	assert.Less(t, st.Rows(), int64(250), "rows handed back for the plays list with its health")
}
