package deploy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{"pending to healthy", StatusPending, StatusHealthy, true},
		{"pending to failed", StatusPending, StatusFailed, true},
		{"pending to running", StatusPending, StatusRunning, false},
		{"running to healthy", StatusRunning, StatusHealthy, true},
		{"running to failed", StatusRunning, StatusFailed, true},
		{"running to pending", StatusRunning, StatusPending, false},
		{"healthy to running", StatusHealthy, StatusRunning, false},
		{"healthy to failed", StatusHealthy, StatusFailed, false},
		{"failed to healthy", StatusFailed, StatusHealthy, false},
		{"failed to running", StatusFailed, StatusRunning, false},
		{"terminal stays terminal", StatusHealthy, StatusHealthy, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CanTransition(tt.from, tt.to))
		})
	}
}

func TestDeployRequestValidate(t *testing.T) {
	valid := func() DeployRequest {
		return DeployRequest{
			StackID: "stack-1",
			Image:   "ghcr.io/onik/api:v1",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*DeployRequest)
		wantErr error
	}{
		{"valid image", func(*DeployRequest) {}, nil},
		{"valid ref", func(r *DeployRequest) { r.Image = ""; r.Ref = "main" }, nil},
		{"empty stack", func(r *DeployRequest) { r.StackID = "  " }, apperrs.ErrInvalid},
		{"neither image nor ref", func(r *DeployRequest) { r.Image = "" }, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := valid()
			tt.mutate(&r)
			err := r.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.wantErr), "want %v, got %v", tt.wantErr, err)
		})
	}
}

func TestDeployRequest_Normalize_FoldsDeprecatedServiceID(t *testing.T) {
	r := DeployRequest{ServiceID: "stack-1"}
	r.normalize()
	assert.Equal(t, "stack-1", r.StackID)

	r2 := DeployRequest{StackID: "stack-2", ServiceID: "stack-1"}
	r2.normalize()
	assert.Equal(t, "stack-2", r2.StackID, "an explicit StackID wins over the deprecated alias")
}

func TestStackValidate(t *testing.T) {
	valid := func() Stack {
		return Stack{
			ProjectID:   "proj-1",
			Name:        "api",
			Machine:     "10.0.0.1:22",
			Strategy:    StrategyCompose,
			ComposePath: "/srv/api/docker-compose.yml",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*Stack)
		wantErr error
	}{
		{"valid compose", func(*Stack) {}, nil},
		{"valid run requires network", func(s *Stack) {
			s.Strategy = StrategyRun
			s.ComposePath = ""
			s.DockerNetwork = "net1"
		}, nil},
		{"empty name", func(s *Stack) { s.Name = "" }, apperrs.ErrInvalid},
		{"empty project", func(s *Stack) { s.ProjectID = "" }, apperrs.ErrInvalid},
		{"empty machine", func(s *Stack) { s.Machine = "" }, apperrs.ErrInvalid},
		{"run missing docker network", func(s *Stack) { s.Strategy = StrategyRun }, apperrs.ErrInvalid},
		{"unknown strategy", func(s *Stack) { s.Strategy = "kube" }, apperrs.ErrInvalid},
		{"build source without repo", func(s *Stack) {
			s.BuildSource = &BuildSource{RepoOwner: "", RepoName: ""}
		}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := valid()
			tt.mutate(&s)
			err := s.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.wantErr), "want %v, got %v", tt.wantErr, err)
		})
	}
}
