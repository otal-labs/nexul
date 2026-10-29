package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
)

var runsBase = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)

// newRunsStore holds one play's runs on a ticket, a doc, an interview and a stack in project p-web, and one in p-ops.
func newRunsStore(t *testing.T) *Store {
	t.Helper()
	s := newTrailStore(t)
	_, err := s.db.ExecContext(t.Context(), `
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES
  ('p-web', 'Web', 'WEB', 0, 'ws-1', 0, 0), ('p-ops', 'Ops', 'OPS', 1, 'ws-1', 0, 0);
INSERT INTO tickets (id, project_id, title, status, number, created_at, updated_at) VALUES
  ('t-1', 'p-web', 'Login times out', 'open', 7, 1, 1), ('t-2', 'p-ops', 'Disk fills up', 'open', 3, 1, 1);
INSERT INTO docs (id, project_id, title, body, version, created_at, updated_at) VALUES ('d-1', 'p-web', 'Release runbook', '', 1, 1, 1);
INSERT INTO stacks (id, project_id, name, machine, strategy, created_at, updated_at) VALUES ('s-1', 'p-web', 'api', 'm-1', 'compose', 1, 1);`)
	require.NoError(t, err)
	runs := []struct {
		id         string
		targetType plays.TargetType
		targetID   string
		projectID  string
		state      plays.TrailState
		age        time.Duration
	}{
		{"ticket", plays.TargetTicket, "t-1", "p-web", plays.TrailDone, 0},
		{"doc", plays.TargetDoc, "d-1", "p-web", plays.TrailWaiting, time.Hour},
		{"interview", plays.TargetInterview, "p-web", "p-web", plays.TrailFailed, 2 * time.Hour},
		{"stack", plays.TargetStack, "s-1", "p-web", plays.TrailWaiting, 3 * time.Hour},
		{"other-project", plays.TargetTicket, "t-2", "p-ops", plays.TrailDone, 4 * time.Hour},
	}
	for _, run := range runs {
		tr := newTestTrail(run.id, runsBase.Add(-run.age))
		tr.TargetType, tr.TargetID, tr.ProjectID, tr.State = run.targetType, run.targetID, run.projectID, run.state
		require.NoError(t, s.PlayTrails.CreateTrail(t.Context(), tr))
	}
	other := newTestTrail("other-play", runsBase.Add(time.Minute))
	other.PlayID = "play-2"
	require.NoError(t, s.PlayTrails.CreateTrail(t.Context(), other))
	_, err = s.db.ExecContext(t.Context(), `
INSERT INTO play_queue (id, workspace_id, project_id, target_type, target_id, play_id, play_label, auto_play_id, run_on, moment, via, priority, status, trail_id, queued_at, not_before)
VALUES ('q-1', 'ws-1', 'p-web', 'ticket', 't-1', 'play-1', 'Fix with AI', 'ap-1', 'developer', 'ticket.entered_stage', 'web', 0, 'started', 'ticket', 1, 1)`)
	require.NoError(t, err)
	return s
}

func runIDs(runs []plays.PlayRun) []string {
	out := make([]string, 0, len(runs))
	for _, run := range runs {
		out = append(out, run.ID)
	}
	return out
}

func TestPlayTrailsRepo_ListPlayRuns_KeepsTheScopeAndOrder(t *testing.T) {
	t.Parallel()
	web := plays.ProjectScope{ProjectIDs: []string{"p-web"}}
	tests := []struct {
		name      string
		scope     plays.RunScope
		waiting   bool
		w         paging.Window
		wantIDs   []string
		wantTotal int
	}{
		{"every project, newest first", plays.RunScope{ProjectScope: plays.ProjectScope{All: true}}, false, paging.Window{Limit: 10},
			[]string{"ticket", "doc", "interview", "stack", "other-project"}, 5},
		{"only the projects the caller opens", plays.RunScope{ProjectScope: web}, false, paging.Window{Limit: 10},
			[]string{"ticket", "doc", "interview", "stack"}, 4},
		{"a doc the caller may not see drops its run alone", plays.RunScope{ProjectScope: web, Docs: &permissions.DocReach{ProjectIDs: []string{"p-web"}, Denied: []string{"d-1"}}}, false, paging.Window{Limit: 10},
			[]string{"ticket", "interview", "stack"}, 3},
		{"a doc reached by its own rule outside the projects shows", plays.RunScope{ProjectScope: web, Docs: &permissions.DocReach{ProjectIDs: []string{}, Allowed: []string{"d-1"}}}, false, paging.Window{Limit: 10},
			[]string{"ticket", "doc", "interview", "stack"}, 4},
		{"waiting runs alone, oldest first", plays.RunScope{ProjectScope: web}, true, paging.Window{Limit: 10},
			[]string{"stack", "doc"}, 2},
		{"one window, every match counted", plays.RunScope{ProjectScope: web}, false, paging.Window{Offset: 1, Limit: 2},
			[]string{"doc", "interview"}, 4},
		{"past the end", plays.RunScope{ProjectScope: web}, false, paging.Window{Offset: 9, Limit: 2},
			[]string{}, 4},
	}
	s := newRunsStore(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runs, total, err := s.PlayTrails.ListPlayRuns(t.Context(), "play-1", tt.scope, tt.waiting, tt.w)
			require.NoError(t, err)
			assert.Equal(t, tt.wantIDs, runIDs(runs))
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestPlayTrailsRepo_ListPlayRuns_NamesWhatEachRunRanOnAndItsAutoPlay(t *testing.T) {
	t.Parallel()
	s := newRunsStore(t)
	runs, _, err := s.PlayTrails.ListPlayRuns(t.Context(), "play-1", plays.RunScope{ProjectScope: plays.ProjectScope{All: true}}, false, paging.Window{Limit: 10})
	require.NoError(t, err)
	got := map[string][3]string{}
	for _, run := range runs {
		got[run.ID] = [3]string{run.TargetKey, run.TargetTitle, run.AutoPlayID}
	}
	assert.Equal(t, map[string][3]string{
		"ticket": {"WEB-7", "Login times out", "ap-1"}, "doc": {"", "Release runbook", ""}, "interview": {"", "Web", ""},
		"stack": {"", "api", ""}, "other-project": {"OPS-3", "Disk fills up", ""},
	}, got)
}

func TestPlaysRepo_RunHealth_CountsTheRunsInScope(t *testing.T) {
	t.Parallel()
	s := newRunsStore(t)
	since := runsBase.Add(-150 * time.Minute)
	tests := []struct {
		name  string
		scope plays.RunScope
		want  map[string]plays.RunHealth
	}{
		{"every project", plays.RunScope{ProjectScope: plays.ProjectScope{All: true}}, map[string]plays.RunHealth{
			"play-1": {LastRun: &plays.LastRun{State: plays.TrailDone, StartedAt: runsBase}, Runs7d: 3, Failed7d: 1, Waiting: 2},
			"play-2": {LastRun: &plays.LastRun{State: plays.TrailStarting, StartedAt: runsBase.Add(time.Minute)}, Runs7d: 1, Running: 1},
		}},
		{"a hidden doc's waiting run is not counted", plays.RunScope{ProjectScope: plays.ProjectScope{ProjectIDs: []string{"p-web"}}, Docs: &permissions.DocReach{ProjectIDs: []string{"p-web"}, Denied: []string{"d-1"}}}, map[string]plays.RunHealth{
			"play-1": {LastRun: &plays.LastRun{State: plays.TrailDone, StartedAt: runsBase}, Runs7d: 2, Failed7d: 1, Waiting: 1},
		}},
		{"no project open: no health", plays.RunScope{ProjectScope: plays.ProjectScope{ProjectIDs: []string{}}}, map[string]plays.RunHealth{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Plays.RunHealth(t.Context(), []string{"play-1", "play-2", "play-never"}, tt.scope, since)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlaysRepo_RunHealth_NoPlaysReadsNothing(t *testing.T) {
	t.Parallel()
	got, err := newTestStore(t).Plays.RunHealth(context.Background(), nil, plays.RunScope{}, runsBase)
	require.NoError(t, err)
	assert.Empty(t, got)
}
