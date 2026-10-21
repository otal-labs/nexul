package pairing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

type cloneCall struct{ title, remoteURL, path string }

func newCloneService(t *testing.T, existing ...string) (*Service, *Computer, *[]cloneCall) {
	t.Helper()
	return newCloneServiceWhere(t, nil, existing...)
}

// newCloneServiceWhere is newCloneService on a T3 whose scope read answers canClone; nil answers yes.
func newCloneServiceWhere(t *testing.T, canClone func() (bool, error), existing ...string) (*Service, *Computer, *[]cloneCall) {
	t.Helper()
	var calls []cloneCall
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.34"}
	if canClone != nil {
		exch.CanCloneFn = func(context.Context, harness.Session) (bool, error) { return canClone() }
	}
	exch.ListProjectsFn = func(context.Context, harness.Session) ([]harness.Project, error) {
		projects := make([]harness.Project, 0, len(existing))
		for _, p := range existing {
			projects = append(projects, harness.Project{ID: p, Path: p})
		}
		return projects, nil
	}
	exch.CloneProjectFn = func(_ context.Context, _ harness.Session, title, remoteURL, path string) (harness.Project, error) {
		calls = append(calls, cloneCall{title, remoteURL, path})
		return harness.Project{ID: "t3-new", Title: title, Path: path}, nil
	}
	svc := newTestService(newFakeRepo(), exch)
	c, err := pairAt(svc, t.Context(), "u1", harness.KindT3Code, "Box", "https://box.example.com", "tok")
	require.NoError(t, err)
	return svc, c, &calls
}

func TestCloneProject_RefusesWhatIsNotAGitHubRepository(t *testing.T) {
	t.Parallel()
	for _, repository := range []string{"", "acme", "acme/api/extra", "acme/..", "../api", "acme/a b", "https://github.com/acme/api"} {
		t.Run(repository, func(t *testing.T) {
			t.Parallel()
			svc, c, calls := newCloneService(t)
			_, err := svc.CloneProject(t.Context(), "u1", c.ID, repository)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Empty(t, *calls, "nothing reaches the computer")
		})
	}
}

func TestCloneProject_SomeoneElsesComputerIsNotFound(t *testing.T) {
	t.Parallel()
	svc, c, calls := newCloneService(t)
	_, err := svc.CloneProject(t.Context(), "u2", c.ID, "acme/api")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Empty(t, *calls)
}

func TestCloneProject_LandsBesideTheComputersProjects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		existing []string
		want     string
	}{
		{"no projects", nil, "~/Code/api"},
		{"one project", []string{"/home/box/.t3/scratch"}, "/home/box/.t3/api"},
		{"the folder most share", []string{"/srv/one", "/home/me/Code/a", "/home/me/Code/b"}, "/home/me/Code/api"},
		{"windows paths", []string{`C:\Users\me\Code\shop`}, `C:\Users\me\Code\api`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, c, calls := newCloneService(t, tt.existing...)
			project, err := svc.CloneProject(t.Context(), "u1", c.ID, " acme/api ")
			require.NoError(t, err)
			assert.Equal(t, []cloneCall{{"api", "https://github.com/acme/api.git", tt.want}}, *calls)
			assert.Equal(t, "t3-new", project.ID)
		})
	}
}

func TestCloneProject_AComputerPairedBeforeCloningIsRefusedWithoutAskingT3ToClone(t *testing.T) {
	t.Parallel()
	svc, c, calls := newCloneServiceWhere(t, func() (bool, error) { return false, nil })
	_, err := svc.CloneProject(t.Context(), "u1", c.ID, "acme/api")
	require.ErrorIs(t, err, harness.ErrPairAgainToClone)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "pair it again")
	assert.Empty(t, *calls, "T3 never sees the clone")
}

func TestCloneProject_AScopeReadThatFailsLeavesTheDecisionToT3(t *testing.T) {
	t.Parallel()
	svc, c, calls := newCloneServiceWhere(t, func() (bool, error) { return false, assert.AnError })
	_, err := svc.CloneProject(t.Context(), "u1", c.ID, "acme/api")
	require.NoError(t, err)
	assert.Len(t, *calls, 1)
}

func TestCanClone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		read func() (bool, error)
		want bool
	}{
		{"granted", func() (bool, error) { return true, nil }, true},
		{"lacking", func() (bool, error) { return false, nil }, false},
		{"unreadable counts as able", func() (bool, error) { return false, assert.AnError }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, c, _ := newCloneServiceWhere(t, tt.read)
			got, err := svc.CanClone(t.Context(), "u1", c.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
	svc, c, _ := newCloneService(t)
	_, err := svc.CanClone(t.Context(), "u2", c.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
