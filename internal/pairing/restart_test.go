package pairing

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeRestarter records the computers it was asked to restart the Nexul app on, failing with err.
type fakeRestarter struct {
	mu    sync.Mutex
	asked []string
	err   error
}

func (f *fakeRestarter) RestartApp(_ context.Context, computerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, computerID)
	return f.err
}

func newRestartService(t *testing.T) (*Service, *fakeRunners, *fakeRestarter) {
	t.Helper()
	svc, repo, runners := newComputerService()
	restarter := &fakeRestarter{}
	svc.restarter = restarter
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "Laptop"}
	return svc, runners, restarter
}

func TestRestartApp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		userID     string
		runner     *ComputerRunner
		restartErr error
		wantErr    error
		wantAsked  []string
	}{
		{"someone else's computer is not found", "u2", &ComputerRunner{Connected: true}, nil, apperrs.ErrNotFound, nil},
		{"no Nexul app yet", "u1", nil, nil, apperrs.ErrConflict, nil},
		{"offline", "u1", &ComputerRunner{Connected: false}, nil, apperrs.ErrConflict, nil},
		{"the runner refused", "u1", &ComputerRunner{Connected: true}, apperrs.ErrConflict, apperrs.ErrConflict, []string{"c1"}},
		{"online restarts it", "u1", &ComputerRunner{Connected: true}, nil, nil, []string{"c1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, runners, restarter := newRestartService(t)
			if tt.runner != nil {
				runners.runners["c1"] = *tt.runner
			}
			restarter.err = tt.restartErr

			err := svc.RestartApp(t.Context(), tt.userID, "c1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			if tt.wantErr == nil {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantAsked, restarter.asked)
		})
	}
}

func TestRestartApp_NotWired_IsFatal(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newComputerService()
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "Laptop"}

	require.ErrorIs(t, svc.RestartApp(t.Context(), "u1", "c1"), apperrs.ErrFatal)
}
