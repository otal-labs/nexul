package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
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
	assert.Equal(t, ProjectSetupChangedEvent{ProjectID: "p-1", WorkspaceID: "ws-1", Setup: ProjectSetup{Finished: true, Steps: want.Steps}}, repo.saved[1].Payload,
		"the frame reaches members without stacks:read, so it names no service")
}

// writerWithoutStacksRead holds every permission on a project except reading its stacks.
type writerWithoutStacksRead struct{ fakeGate }

func (g *writerWithoutStacksRead) Require(ctx context.Context, id string, action permissions.Action) error {
	if action == permissions.StacksRead {
		return apperrs.ErrForbidden
	}
	return g.fakeGate.Require(ctx, id, action)
}

func (g *writerWithoutStacksRead) RequireProject(ctx context.Context, id string, action permissions.Action) error {
	return g.Require(ctx, id, action)
}

func projectWithService() *Project {
	return &Project{ID: "p-1", WorkspaceID: "ws-1", Prefix: "ACM", Setup: ProjectSetup{
		StackID: "stack-1", EnvKeys: []string{"STRIPE_SECRET_KEY"}, Steps: map[SetupStep]SetupMark{"service": SetupDone},
	}}
}

func TestProjectReads_WithoutStacksRead_HideTheSetupService(t *testing.T) {
	reads := map[string]func(*Service) (*Project, error){
		"get": func(s *Service) (*Project, error) { return s.Get(t.Context(), "p-1") },
		"list": func(s *Service) (*Project, error) {
			projects, err := s.List(t.Context(), "ws-1")
			if err != nil || len(projects) != 1 {
				return nil, errors.Join(err, errors.New("want one project"))
			}
			return projects[0], nil
		},
		"change setup": func(s *Service) (*Project, error) {
			return s.ChangeSetup(t.Context(), "p-1", SetupChange{Steps: map[SetupStep]SetupMark{"reach": SetupSkipped}})
		},
		"rename": func(s *Service) (*Project, error) { return s.Rename(t.Context(), "p-1", "Acme", nil) },
		"tests location": func(s *Service) (*Project, error) {
			return s.SetTestsLocation(t.Context(), "p-1", TestsLocationSame)
		},
		"same prefix": func(s *Service) (*Project, error) { return s.SetPrefix(t.Context(), "p-1", "ACM") },
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			blindRepo := newFakeRepo()
			blindRepo.projects["p-1"] = projectWithService()
			blind := newTestService(blindRepo, &writerWithoutStacksRead{fakeGate{allow: true}})
			got, err := read(blind)
			require.NoError(t, err)
			raw, err := json.Marshal(got)
			require.NoError(t, err)
			assert.NotContains(t, string(raw), "stack-1")
			assert.NotContains(t, string(raw), "STRIPE_SECRET_KEY")
			assert.Equal(t, SetupDone, got.Setup.Steps["service"], "the step marks still show")

			sighted, repo, _ := newOwnerRepo(t, true)
			repo.projects["p-1"] = projectWithService()
			got, err = read(sighted)
			require.NoError(t, err)
			assert.Equal(t, "stack-1", got.Setup.StackID)
			assert.Equal(t, []string{"STRIPE_SECRET_KEY"}, got.Setup.EnvKeys)
		})
	}
}
