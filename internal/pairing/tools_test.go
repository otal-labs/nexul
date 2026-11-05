package pairing

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeTools records what the runner was asked to install and set, failing with err.
type fakeTools struct {
	mu            sync.Mutex
	installed     []string
	sudoPasswords []string
	identities    []GitIdentity
	err           error
}

func (f *fakeTools) InstallTool(_ context.Context, computerID, tool, sudoPassword string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.installed = append(f.installed, computerID+"/"+tool)
	f.sudoPasswords = append(f.sudoPasswords, sudoPassword)
	return f.err
}

func (f *fakeTools) SetGitIdentity(_ context.Context, _, name, email string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.identities = append(f.identities, GitIdentity{Name: name, Email: email})
	return f.err
}

type fakeIdentities struct {
	id  GitIdentity
	err error
}

func (f fakeIdentities) GitHubIdentity(context.Context, string) (GitIdentity, error) {
	return f.id, f.err
}

var aliceOnGitHub = GitIdentity{Name: "Alice Example", Email: "1+alice@users.noreply.github.com"}

func newToolsService(t *testing.T, facts *Facts) (*Service, *fakeRunners, *fakeTools) {
	t.Helper()
	svc, repo, runners := newComputerService()
	tools := &fakeTools{}
	svc.tools, svc.identities = tools, fakeIdentities{id: aliceOnGitHub}
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "Laptop", Facts: facts}
	runners.runners["c1"] = ComputerRunner{Connected: true}
	return svc, runners, tools
}

var winget = []string{"winget", "install", "--id", "Git.Git"}

func TestInstallTool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tool    string
		facts   *Facts
		offline bool
		wantErr error
		want    []string
	}{
		{"a tool Nexul does not install", "node", &Facts{}, false, apperrs.ErrInvalid, nil},
		{"offline", "git", &Facts{Tools: []ToolFacts{{Name: "git", Install: winget}}}, true, apperrs.ErrConflict, nil},
		{"no facts yet", "git", nil, false, apperrs.ErrConflict, nil},
		{"a runner that reports no tools", "git", &Facts{}, false, apperrs.ErrConflict, nil},
		{"already installed", "git", &Facts{Tools: []ToolFacts{{Name: "git", Path: `C:\Program Files\Git\cmd\git.exe`}}}, false, apperrs.ErrConflict, nil},
		{"no package manager to install with", "git", &Facts{Tools: []ToolFacts{{Name: "git"}}}, false, apperrs.ErrConflict, nil},
		{"installs a missing tool", "git", &Facts{Tools: []ToolFacts{{Name: "git", Install: winget}}}, false, nil, []string{"c1/git"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, runners, tools := newToolsService(t, tt.facts)
			if tt.offline {
				runners.runners["c1"] = ComputerRunner{}
			}

			err := svc.InstallTool(t.Context(), "u1", "c1", tt.tool, "")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			}
			if tt.wantErr == nil {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, tools.installed)
		})
	}
}

func TestInstallTool_SudoWantsAPassword_AsksForItOnThatComputer(t *testing.T) {
	t.Parallel()
	apt := []string{"sudo", "-n", "apt-get", "install", "-y", "git"}
	svc, _, tools := newToolsService(t, &Facts{Tools: []ToolFacts{{Name: "git", Install: apt}}})
	tools.err = &SudoPasswordError{Detail: "that sudo password was wrong"}

	err := svc.InstallTool(t.Context(), "u1", "c1", "git", "hunter2")

	require.ErrorIs(t, err, apperrs.ErrConflict)
	var sudo *SudoPasswordError
	require.ErrorAs(t, err, &sudo)
	assert.Equal(t, "that sudo password was wrong", err.Error())
	assert.Equal(t, SudoPasswordDetails{SudoPasswordNeeded: true, ComputerID: "c1"}, sudo.ErrorDetails())
	assert.Equal(t, []string{"hunter2"}, tools.sudoPasswords, "the password goes to the computer as given")
}

func TestInstallTool_SomeoneElsesComputerIsNotFound(t *testing.T) {
	t.Parallel()
	svc, _, _ := newToolsService(t, &Facts{Tools: []ToolFacts{{Name: "git", Install: winget}}})

	require.ErrorIs(t, svc.InstallTool(t.Context(), "u2", "c1", "git", ""), apperrs.ErrNotFound)
}

func TestSetGitIdentity(t *testing.T) {
	t.Parallel()
	git := ToolFacts{Name: "git", Path: "/usr/bin/git"}
	tests := []struct {
		name    string
		facts   *Facts
		wantErr error
		want    GitIdentity
	}{
		{"neither set takes both", &Facts{Tools: []ToolFacts{git}}, nil, aliceOnGitHub},
		{"an email already set keeps it", &Facts{Tools: []ToolFacts{git}, GitEmail: "alice@example.com"}, nil, GitIdentity{Name: "Alice Example"}},
		{"both set is refused", &Facts{Tools: []ToolFacts{git}, GitName: "Alice", GitEmail: "alice@example.com"}, apperrs.ErrConflict, GitIdentity{}},
		{"no git names it missing", &Facts{Tools: []ToolFacts{{Name: "git"}}}, apperrs.ErrConflict, GitIdentity{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, _, tools := newToolsService(t, tt.facts)

			got, err := svc.SetGitIdentity(t.Context(), "u1", "c1")

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, tools.identities)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, []GitIdentity{tt.want}, tools.identities)
		})
	}
}

func TestSetGitIdentity_WithoutGit_OffersToInstallIt(t *testing.T) {
	t.Parallel()
	svc, _, _ := newToolsService(t, &Facts{Tools: []ToolFacts{{Name: "git"}}})

	_, err := svc.SetGitIdentity(t.Context(), "u1", "c1")

	var missing *MissingToolError
	require.ErrorAs(t, err, &missing)
	assert.Equal(t, MissingToolDetails{MissingTool: "git", ComputerID: "c1"}, missing.ErrorDetails())
}

func TestGitHubGitIdentity_WithoutAGitHubSignIn_IsAConflict(t *testing.T) {
	t.Parallel()
	svc, _, _ := newToolsService(t, &Facts{})
	svc.identities = fakeIdentities{err: apperrs.ErrNotFound}

	_, err := svc.GitHubGitIdentity(t.Context(), "u1")
	require.ErrorIs(t, err, apperrs.ErrConflict)

	svc.identities = fakeIdentities{err: errors.New("disk")}
	_, err = svc.GitHubGitIdentity(t.Context(), "u1")
	require.Error(t, err)
	assert.NotErrorIs(t, err, apperrs.ErrConflict)
}

func TestTools_NotWired_IsFatal(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newComputerService()
	repo.computers["c1"] = Computer{ID: "c1", UserID: "u1", Name: "Laptop"}

	require.ErrorIs(t, svc.InstallTool(t.Context(), "u1", "c1", "git", ""), apperrs.ErrFatal)
	_, err := svc.GitHubGitIdentity(t.Context(), "u1")
	require.ErrorIs(t, err, apperrs.ErrFatal)
}
