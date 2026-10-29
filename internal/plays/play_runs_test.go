package plays

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func TestListPlayRuns_RefusesWhatTheCallerMayNotRead(t *testing.T) {
	tests := []struct {
		name    string
		caller  string
		playID  string
		listErr error
		want    error
	}{
		{"no play", starter, " ", nil, apperrs.ErrInvalid},
		{"unknown play", starter, "nope", nil, apperrs.ErrNotFound},
		{"a play they can't see", "stranger", fixPlayID, nil, apperrs.ErrNotFound},
		{"the read fails", starter, fixPlayID, errors.New("db down"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			f.trails.listErr = tt.listErr
			_, _, err := f.runner.ListPlayRuns(ctxAs(tt.caller), tt.playID, false, paging.Window{})
			if tt.want == nil {
				assert.ErrorContains(t, err, "db down")
				return
			}
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestListPlayRuns_PagesTheVisibleRunsAndKeepsPassedOverToTheStarter(t *testing.T) {
	f := newRunnerFixture()
	f.perm.grants["u-2"] = []permissions.Action{permissions.PlaysRead}
	f.perm.projects = []string{projectID}
	for i, id := range []string{"tr-1", "tr-2", "tr-3"} {
		tr := seededTrail(f, id, TargetTicket, ticketID, fixedNow.Add(time.Duration(i)*time.Minute))
		tr.PassedOver = []PassedOver{{ComputerID: "c-1"}}
		require.NoError(t, f.trails.UpdateTrail(t.Context(), tr))
		<-f.trails.terminal
	}
	hidden := seededTrail(f, "tr-hidden", TargetTicket, "t-foreign", fixedNow.Add(time.Hour))
	hidden.ProjectID = otherProj
	require.NoError(t, f.trails.UpdateTrail(t.Context(), hidden))
	<-f.trails.terminal

	runs, total, err := f.runner.ListPlayRuns(ctxAs(starter), fixPlayID, false, paging.Window{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total, "the run in a project the caller can't open is not counted")
	require.Len(t, runs, 2)
	assert.Equal(t, []string{"tr-3", "tr-2"}, []string{runs[0].ID, runs[1].ID})
	assert.NotEmpty(t, runs[0].PassedOver, "the starter reads the computers their run passed over")

	others, _, err := f.runner.ListPlayRuns(ctxAs("u-2"), fixPlayID, false, paging.Window{Limit: 2})
	require.NoError(t, err)
	require.Len(t, others, 2)
	assert.Empty(t, others[0].PassedOver, "nobody else does")
}

func TestServiceList_FillsEachPlaysHealthFromTheCallersScope(t *testing.T) {
	repo := newFakeRepo()
	perm := allowAll("u-owner")
	perm.projects = []string{"p-1"}
	repo.byID["p-a"] = &Play{ID: "p-a", WorkspaceID: workspaceID, Label: "A", Type: TypeDoc}
	repo.byID["p-b"] = &Play{ID: "p-b", WorkspaceID: workspaceID, Label: "B", Type: TypeDoc}
	repo.health = map[string]RunHealth{"p-a": {LastRun: &LastRun{State: TrailFailed, StartedAt: fixedNow}, Runs7d: 4, Failed7d: 1, Waiting: 2}}
	s := newTestService(repo, perm)

	list, err := s.List(ctxAs("u-owner"), workspaceID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, repo.health["p-a"], list[0].RunHealth)
	assert.Equal(t, RunHealth{}, list[1].RunHealth, "a play with no runs has no health")
	assert.Equal(t, []string{"p-1"}, repo.healthScope.ProjectIDs)
	require.NotNil(t, repo.healthScope.Docs, "a caller kept to some projects is kept to the doc threads they may see too")

	repo.healthErr = errors.New("db down")
	_, err = s.List(ctxAs("u-owner"), workspaceID)
	assert.ErrorContains(t, err, "db down")
}
