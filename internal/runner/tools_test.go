package runner

import (
	"context"
	"errors"
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

	err := c.installTool(t.Context(), "node", "")
	require.ErrorContains(t, err, "only git and gh can be installed")

	err = c.installTool(t.Context(), "gh", "")
	require.ErrorContains(t, err, "no package manager Nexul installs gh with")

	c.cfg.Personal = false
	require.ErrorContains(t, c.installTool(t.Context(), "gh", ""), "not a personal runner")
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

	answer, facts := installAnswer(t, "")

	assert.Equal(t, Frame{Type: FrameInstallToolResult, ID: "i1"}, answer)
	assert.Contains(t, facts.Tools, ToolFacts{Name: "gh", Path: gh, Version: "2.63.2"})
}

// standInSudo refuses sudo -n for want of a password and, with -k -S -p "", runs the command once stdin gives hunter2.
const standInSudo = `#!/bin/sh
if [ "$1" = -n ]; then echo 'sudo: a password is required' >&2; exit 1; fi
[ "$1 $2 $3 $4" = "-k -S -p " ] || { echo "unexpected sudo flags: $*" >&2; exit 2; }
read -r password
if [ "$password" != hunter2 ]; then echo 'Sorry, try again.' >&2; echo 'sudo: 1 incorrect password attempt' >&2; exit 1; fi
shift 4
exec "$@"
`

func TestPersonalRunner_InstallThatNeedsASudoPassword(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sudo is how Nexul installs with a Linux package manager")
	}
	if isRoot() {
		t.Skip("root installs without sudo")
	}
	tests := []struct {
		name     string
		password string
		want     Frame
	}{
		{"no password says how to give one", "", Frame{Type: FrameInstallToolResult, ID: "i1", SudoPasswordNeeded: true,
			Error: "installing gh needs your sudo password: enter it from Install gh in Settings → Computers, or run sudo apt-get install -y gh on the computer"}},
		{"a wrong password asks again", "hunter3", Frame{Type: FrameInstallToolResult, ID: "i1", SudoPasswordNeeded: true, Error: "that sudo password was wrong"}},
		{"the right password installs", "hunter2", Frame{Type: FrameInstallToolResult, ID: "i1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin := t.TempDir()
			gh := filepath.Join(bin, "gh")
			installs := "#!/bin/sh\nprintf '#!/bin/sh\\necho \"gh version 2.63.2 (2024-12-05)\"\\n' > " + gh + "\n/bin/chmod +x " + gh + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(bin, "apt-get"), []byte(installs), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(bin, "sudo"), []byte(standInSudo), 0o755))
			t.Setenv("PATH", bin)

			answer, _ := installAnswer(t, tt.password)

			assert.Equal(t, tt.want, answer)
		})
	}
}

func TestSudoRefusal(t *testing.T) {
	apt := []string{"sudo", "-n", "apt-get", "install", "-y", "git"}
	failed := errors.New("exit status 1")
	tests := []struct {
		name   string
		cmd    []string
		out    string
		err    error
		sent   bool
		wanted string
	}{
		{"an install that worked", apt, "", nil, false, ""},
		{"a command with no sudo", []string{"brew", "install", "git"}, "a password is required", failed, false, ""},
		{"sudo wants a password", apt, "sudo: a password is required", failed, false,
			"installing git needs your sudo password: enter it from Install git in Settings → Computers, or run sudo apt-get install -y git on the computer"},
		{"the password sent was wrong", apt, "Sorry, try again.\nsudo: 1 incorrect password attempt", failed, true, "that sudo password was wrong"},
		{"the package manager failed after sudo let it run", apt, "E: Unable to locate package git", failed, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sudoRefusal("git", tt.cmd, []byte(tt.out), tt.err, tt.sent)
			if tt.wanted == "" {
				require.NoError(t, err)
				return
			}
			var sudo *SudoPasswordError
			require.ErrorAs(t, err, &sudo)
			assert.Equal(t, tt.wanted, sudo.Detail)
		})
	}
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

			require.ErrorIs(t, f.h.InstallTool(t.Context(), "c-laptop", "git", ""), ErrCannotInstall)
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
	install := func() error { return f.h.InstallTool(t.Context(), "c-laptop", "git", "") }
	identity := func() error {
		return f.h.SetGitIdentity(t.Context(), "c-laptop", "Alice", "1+alice@users.noreply.github.com")
	}

	require.NoError(t, answer(install, Frame{Type: FrameInstallTool, Tool: "git"}, Frame{Type: FrameInstallToolResult}))
	err = answer(install, Frame{Type: FrameInstallTool, Tool: "git"}, Frame{Type: FrameInstallToolResult, Error: "winget install failed"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "winget install failed")
	require.NoError(t, answer(identity, Frame{Type: FrameGitIdentity, GitName: "Alice", GitEmail: "1+alice@users.noreply.github.com"}, Frame{Type: FrameGitIdentityResult}))
}

func TestInstallTool_PassesASudoPasswordOnlyToARunnerThatTakesOne(t *testing.T) {
	for _, tt := range []struct {
		version string
		takes   bool
	}{{lastWithoutSudoPassword, false}, {"v0.3.79-beta", true}} {
		t.Run("version "+tt.version, func(t *testing.T) {
			f := newT3Fixture(t)
			conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version="+tt.version)
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.CloseNow() })
			eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })
			if !tt.takes {
				require.ErrorIs(t, f.h.InstallTool(t.Context(), "c-laptop", "git", "hunter2"), ErrCannotTakeSudoPassword)
				return
			}
			got := make(chan error, 1)
			go func() { got <- f.h.InstallTool(t.Context(), "c-laptop", "git", "hunter2") }()
			var req Frame
			for req.Type != FrameInstallTool {
				require.NoError(t, wsjson.Read(t.Context(), conn, &req))
			}
			assert.Equal(t, "hunter2", req.SudoPassword)
			require.NoError(t, wsjson.Write(t.Context(), conn, Frame{Type: FrameInstallToolResult, ID: req.ID, SudoPasswordNeeded: true, Error: "that sudo password was wrong"}))
			var sudo *SudoPasswordError
			require.ErrorAs(t, <-got, &sudo)
			assert.Equal(t, "that sudo password was wrong", sudo.Detail)
		})
	}
}

// installAnswer asks a personal runner to install gh, with sudoPassword when given, and returns its answer and the facts report that follows it, or its answer alone when it failed.
func installAnswer(t *testing.T, sudoPassword string) (Frame, Facts) {
	t.Helper()
	type result struct {
		answer Frame
		facts  Facts
	}
	got := make(chan result, 1)
	personalClient(t, t.TempDir(), "", nil, func(ctx context.Context, conn *websocket.Conn) {
		_ = wsjson.Write(ctx, conn, Frame{Type: FrameInstallTool, ID: "i1", Tool: "gh", SudoPassword: sudoPassword})
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
