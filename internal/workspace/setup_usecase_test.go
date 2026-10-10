package workspace

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestChangeSetup(t *testing.T) {
	finished := true
	steps := func(pairs ...string) map[SetupStep]SetupMark {
		m := map[SetupStep]SetupMark{}
		for i := 0; i < len(pairs); i += 2 {
			m[SetupStep(pairs[i])] = SetupMark(pairs[i+1])
		}
		return m
	}
	tests := []struct {
		name      string
		allow     bool
		repoErr   error
		change    SetupChange
		wantErr   error
		wantSetup ProjectSetup
	}{
		{name: "a member without projects:write is refused", change: SetupChange{Finished: &finished}, wantErr: apperrs.ErrForbidden},
		{name: "a step outside the wizard is invalid", allow: true, change: SetupChange{Steps: steps("deploy", "done")}, wantErr: apperrs.ErrInvalid},
		{name: "a mark other than done or skipped is invalid", allow: true, change: SetupChange{Steps: steps("reach", "maybe")}, wantErr: apperrs.ErrInvalid},
		{name: "a failed write is returned", allow: true, repoErr: errors.New("disk full"), change: SetupChange{Steps: steps("reach", "skipped")}, wantErr: errors.New("disk full")},
		{
			name: "skipping a step marks it and keeps setup open", allow: true, change: SetupChange{Steps: steps("repository", "skipped")},
			wantSetup: ProjectSetup{Steps: steps("project", "done", "repository", "skipped")},
		},
		{
			name: "finishing with steps skipped finishes", allow: true, change: SetupChange{Finished: &finished, Steps: steps("service", "skipped")},
			wantSetup: ProjectSetup{Finished: true, Steps: steps("project", "done", "service", "skipped")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, repo, _ := newOwnerRepo(t, tt.allow)
			repo.projects["p-1"] = &Project{ID: "p-1", WorkspaceID: "ws-1", Setup: NewSetup(false)}
			repo.updateErr = tt.repoErr

			got, err := s.ChangeSetup(t.Context(), "p-1", tt.change)

			if tt.wantErr != nil {
				require.Error(t, err)
				if !errors.Is(err, tt.wantErr) {
					assert.ErrorContains(t, err, tt.wantErr.Error())
				}
				assert.Equal(t, NewSetup(false), repo.projects["p-1"].Setup, "nothing is written")
				assert.Empty(t, repo.saved)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSetup, got.Setup)
			assert.Equal(t, tt.wantSetup, repo.projects["p-1"].Setup)
			require.Len(t, repo.saved, 1)
			assert.Equal(t, TopicProjectSetupChanged, repo.saved[0].Topic)
			assert.Equal(t, ProjectSetupChangedEvent{ProjectID: "p-1", WorkspaceID: "ws-1", Setup: tt.wantSetup}, repo.saved[0].Payload)
		})
	}
}

func TestChangeSetup_NothingNew_WritesNothing(t *testing.T) {
	for name, mark := range map[string]SetupMark{"a repeated mark": SetupDone, "a skip of a step already done": SetupSkipped} {
		t.Run(name, func(t *testing.T) {
			s, repo, _ := newOwnerRepo(t, true)
			repo.projects["p-1"] = &Project{ID: "p-1", WorkspaceID: "ws-1", Setup: NewSetup(false)}

			got, err := s.ChangeSetup(t.Context(), "p-1", SetupChange{Steps: map[SetupStep]SetupMark{"project": mark}})

			require.NoError(t, err)
			assert.Equal(t, SetupDone, got.Setup.Steps["project"])
			assert.Empty(t, repo.saved, "no frame goes to every member's sidebar")
		})
	}
}

func TestChangeSetup_ServiceContextSurvivesLaterMarks(t *testing.T) {
	s, repo, _ := newOwnerRepo(t, true)
	repo.services["p-1"] = []string{"stack-1"}
	repo.projects["p-1"] = &Project{ID: "p-1", WorkspaceID: "ws-1", Setup: NewSetup(false)}
	stackID := "stack-1"
	envKeys := []string{"PORT"}

	got, err := s.ChangeSetup(t.Context(), "p-1", SetupChange{StackID: &stackID, EnvKeys: &envKeys})
	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
	assert.Equal(t, "stack-1", got.Setup.StackID)
	assert.Equal(t, []string{"PORT"}, got.Setup.EnvKeys)

	finished := true
	got, err = s.ChangeSetup(t.Context(), "p-1", SetupChange{Finished: &finished, Steps: map[SetupStep]SetupMark{"env": SetupSkipped}})
	require.NoError(t, err)
	want := ProjectSetup{Finished: true, StackID: "stack-1", EnvKeys: []string{"PORT"}, Steps: map[SetupStep]SetupMark{"project": SetupDone, "env": SetupSkipped}}
	assert.Equal(t, want, got.Setup)
	require.Len(t, repo.saved, 2)
	assert.Equal(t, ProjectSetupChangedEvent{ProjectID: "p-1", WorkspaceID: "ws-1", Setup: want}, repo.saved[1].Payload)
}
