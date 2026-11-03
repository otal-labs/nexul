package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestPackageCommand(t *testing.T) {
	winget := []string{"winget", "install", "--id", "Git.Git", "--exact", "--source", "winget", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}
	tests := []struct {
		name    string
		manager string
		tool    string
		root    bool
		want    []string
	}{
		{"winget installs Git for Windows by its id", "winget", "git", false, winget},
		{"brew needs no sudo", "brew", "gh", false, []string{"brew", "install", "gh"}},
		{"apt-get through sudo without a prompt", "apt-get", "git", false, []string{"sudo", "-n", "apt-get", "install", "-y", "git"}},
		{"apt-get as root needs no sudo", "apt-get", "gh", true, []string{"apt-get", "install", "-y", "gh"}},
		{"dnf", "dnf", "gh", false, []string{"sudo", "-n", "dnf", "install", "-y", "gh"}},
		{"pacman names gh github-cli", "pacman", "gh", false, []string{"sudo", "-n", "pacman", "-S", "--noconfirm", "github-cli"}},
		{"a tool Nexul does not install", "brew", "node", false, nil},
		{"a package manager Nexul does not know", "zypper", "git", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, packageCommand(tt.manager, toolPackages[tt.manager][tt.tool], tt.root))
		})
	}
}

func TestInstallCommand_UsesTheFirstPackageManagerTheComputerHas(t *testing.T) {
	bin := t.TempDir()
	fakeCommand(t, bin, "dnf")
	fakeCommand(t, bin, "pacman")
	t.Setenv("PATH", bin)
	stubSavedPath(t, func() (string, error) { return "", nil })

	assert.Equal(t, []string{"dnf", "install", "-y", "git"}, installCommand("linux", "git", true))
	assert.Nil(t, installCommand("darwin", "git", false), "no brew here")
}

func TestInstallTool_Refusals(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	stubSavedPath(t, func() (string, error) { return "", nil })
	c := newTestClient("ws://unused", &fakeExecutor{})
	c.cfg.Personal = true

	err := c.installTool(t.Context(), "node")
	require.ErrorContains(t, err, "only git and gh can be installed")

	err = c.installTool(t.Context(), "gh")
	require.ErrorContains(t, err, "no package manager Nexul installs gh with")

	c.cfg.Personal = false
	require.ErrorContains(t, c.installTool(t.Context(), "gh"), "not a personal runner")
}

// TestPersonalRunner_InstallsAMissingToolThenReportsItsFactsAgain: the runner runs its package manager's command, the
// tool is then on its PATH, and the facts report that follows says so.
func TestPersonalRunner_InstallsAMissingToolThenReportsItsFactsAgain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake package manager is a shell script")
	}
	bin := t.TempDir()
	gh := filepath.Join(bin, "gh")
	installs := "#!/bin/sh\nprintf '#!/bin/sh\\necho \"gh version 2.63.2 (2024-12-05)\"\\n' > " + gh + "\n/bin/chmod +x " + gh + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "brew"), []byte(installs), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "apt-get"), []byte(installs), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "sudo"), []byte("#!/bin/sh\nshift\nexec \"$@\"\n"), 0o755))
	t.Setenv("PATH", bin)

	answer, facts := installAnswer(t)

	assert.Equal(t, Frame{Type: FrameInstallToolResult, ID: "i1"}, answer)
	assert.Contains(t, facts.Tools, ToolFacts{Name: "gh", Path: gh, Version: "2.63.2"})
}

func TestPersonalRunner_InstallThatNeedsASudoPassword_SaysWhatToRun(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sudo -n is how Nexul installs with a Linux package manager")
	}
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "apt-get"), []byte("#!/bin/sh\nexit 1\n"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "sudo"), []byte("#!/bin/sh\necho 'sudo: a password is required' >&2\nexit 1\n"), 0o755))
	t.Setenv("PATH", bin)
	if isRoot() {
		t.Skip("root installs without sudo")
	}

	answer, _ := installAnswer(t)

	assert.Equal(t, "installing gh needs your sudo password; run sudo apt-get install -y gh on the computer", answer.Error)
}

func TestSetGitIdentity_SetsOnlyWhatIsGiven(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	require.NoError(t, err)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("PATH", filepath.Dir(gitPath))
	c := newTestClient("ws://unused", &fakeExecutor{})
	c.cfg.Personal = true

	require.NoError(t, c.setGitIdentity(t.Context(), "Alice Example", ""))
	require.NoError(t, c.setGitIdentity(t.Context(), "", "1+alice@users.noreply.github.com"))

	assert.Equal(t, "Alice Example", commandLine(t.Context(), "git", "config", "--global", "user.name"))
	assert.Equal(t, "1+alice@users.noreply.github.com", commandLine(t.Context(), "git", "config", "--global", "user.email"))
}

func TestSetGitIdentity_WithoutGit_NamesIt(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	stubSavedPath(t, func() (string, error) { return "", nil })
	c := newTestClient("ws://unused", &fakeExecutor{})
	c.cfg.Personal = true

	err := c.setGitIdentity(t.Context(), "Alice Example", "")

	assert.Equal(t, "git", missingTool(err))
}

func TestPersonalRunner_CloneWithoutGit_NamesTheMissingTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	stubSavedPath(t, func() (string, error) { return "", nil })

	answer, _ := gitCloneAnswer(t)

	assert.Equal(t, "git", answer.Tool)
	assert.Equal(t, "git isn't installed on this computer", answer.Error)
}

func TestInstallTool_AsksOnlyARunnerNewEnough(t *testing.T) {
	for _, v := range []string{"", "v0.3.60-beta", lastWithoutInstall} {
		t.Run("version "+v, func(t *testing.T) {
			f := newT3Fixture(t)
			conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version="+v)
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.CloseNow() })
			eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

			require.ErrorIs(t, f.h.InstallTool(t.Context(), "c-laptop", "git"), ErrCannotInstall)
			require.ErrorIs(t, f.h.SetGitIdentity(t.Context(), "c-laptop", "Alice", ""), ErrCannotInstall)
		})
	}
}

func TestInstallToolAndGitIdentity_ReturnWhatTheRunnerAnswered(t *testing.T) {
	f := newT3Fixture(t)
	conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version=v0.3.78-beta")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })
	eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

	answer := func(ask func() error, want Frame, reply Frame) error {
		got := make(chan error, 1)
		go func() { got <- ask() }()
		var req Frame
		for req.Type != want.Type {
			require.NoError(t, wsjson.Read(t.Context(), conn, &req))
		}
		want.ID = req.ID
		assert.Equal(t, want, req)
		reply.ID = req.ID
		require.NoError(t, wsjson.Write(t.Context(), conn, reply))
		return <-got
	}
	install := func() error { return f.h.InstallTool(t.Context(), "c-laptop", "git") }
	identity := func() error {
		return f.h.SetGitIdentity(t.Context(), "c-laptop", "Alice", "1+alice@users.noreply.github.com")
	}

	require.NoError(t, answer(install, Frame{Type: FrameInstallTool, Tool: "git"}, Frame{Type: FrameInstallToolResult}))
	err = answer(install, Frame{Type: FrameInstallTool, Tool: "git"}, Frame{Type: FrameInstallToolResult, Error: "winget install failed"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "winget install failed")
	require.NoError(t, answer(identity, Frame{Type: FrameGitIdentity, GitName: "Alice", GitEmail: "1+alice@users.noreply.github.com"}, Frame{Type: FrameGitIdentityResult}))
}

// installAnswer asks a personal runner to install gh, and returns its answer and the facts report that follows it, or its answer alone when it failed.
func installAnswer(t *testing.T) (Frame, Facts) {
	t.Helper()
	type result struct {
		answer Frame
		facts  Facts
	}
	got := make(chan result, 1)
	personalClient(t, t.TempDir(), "", nil, func(ctx context.Context, conn *websocket.Conn) {
		_ = wsjson.Write(ctx, conn, Frame{Type: FrameInstallTool, ID: "i1", Tool: "gh"})
		var answer *Frame
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameInstallToolResult && f.Error != "" {
				got <- result{answer: f}
				return
			}
			if f.Type == FrameInstallToolResult {
				answer = &f
				continue
			}
			if f.Type == FrameFacts && answer != nil && slices.ContainsFunc(f.Facts.Tools, func(tf ToolFacts) bool { return tf.Path != "" }) {
				got <- result{*answer, *f.Facts}
				return
			}
		}
	})
	select {
	case r := <-got:
		return r.answer, r.facts
	case <-time.After(30 * time.Second):
		t.Fatal("no install_tool_result and facts within 30s")
		return Frame{}, Facts{}
	}
}

// fakeCommand puts an empty command called name in dir, which is all lookTool needs to find it.
func fakeCommand(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0o755))
}
