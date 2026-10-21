//go:build !windows

package install

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Stand-ins for the runner and T3 Code: each reports how it was started on a FIFO the test reads, then waits to be
// stopped. A runner that finds NEXUL_TEST_REMOVED asks for its removal, as a removed runner does, and exits.
const (
	standInRunner = `#!/bin/sh
printf 'runner|%s|%s\n' "$NEXUL_RUNNER_NAME" "$NEXUL_T3_SUPERVISED" >"$NEXUL_TEST_FIFO"
[ -n "$NEXUL_TEST_REMOVED" ] && { : >remove-requested; exit 0; }
exec sleep 3600
`
	standInT3 = `#!/bin/sh
printf 't3|%s\n' "$*" >"$NEXUL_TEST_FIFO"
exec sleep 3600
`
)

// noSystemd makes the test host a Linux machine that is not running systemd, with alice at its keyboard and no sudo.
func (th *testHost) noSystemd(t *testing.T) {
	t.Helper()
	th.Getuid = func() int { return 1000 }
	t.Setenv("SUDO_USER", "")
	withVersion(t, "v0.3.1")
	th.Paths.SystemdProbe = filepath.Join(th.root, "no-systemd")
	th.missing("t3")
	th.aliceHas(t, ".local/bin/t3", standInT3, 0o755)
	th.chowned = map[string]string{} // what the install itself hands over is what the tests look at
	th.release.files["nexul-runner-linux-amd64"] = []byte(standInRunner)
}

// fifoLines reads the lines the stand-ins print. The FIFO is opened for writing too, so no stand-in waits for a reader.
func fifoLines(t *testing.T) <-chan string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fifo")
	require.NoError(t, syscall.Mkfifo(path, 0o600))
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	t.Setenv("NEXUL_TEST_FIFO", path)
	lines := make(chan string, 8)
	go func() {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	return lines
}

// installUntilRunning starts the computer install, which stays in the foreground, and returns once both stand-ins
// have started; the install's result arrives on the returned channel after cancel.
func (th *testHost) installUntilRunning(t *testing.T, args []string, lines <-chan string, wantLines int) (cancel context.CancelFunc, result <-chan error, got []string) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- th.runInstall(ctx, args) }()
	for len(got) < wantLines {
		select {
		case line := <-lines:
			got = append(got, line)
		case err := <-done:
			require.FailNow(t, "the install ended before the runner and T3 Code ran", "%v\n%s", err, th.out.String())
		}
	}
	return cancel, done, got
}

func TestInstallComputer_WithoutSystemd_RunsTheRunnerAndT3CodeUnderTheSupervisorAndRegistersNoService(t *testing.T) {
	tests := []struct {
		name  string
		setup func(th *testHost)
		flags []string
	}{
		{"a host that is not running systemd", func(*testHost) {}, nil},
		{"--foreground on a host that is", func(th *testHost) { require.NoError(t, os.MkdirAll(th.root+"/no-systemd", 0o755)) }, []string{"--foreground"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.noSystemd(t)
			tt.setup(th)
			lines := fifoLines(t)
			args := append([]string{"computer", "--token", computerToken(t, th.web.URL+"/")}, tt.flags...)

			cancel, result, got := th.installUntilRunning(t, args, lines, 2)
			cancel()
			require.NoError(t, <-result, "a stop is how the supervisor ends, not a failure")

			dir := filepath.Join(th.Home, ".local", "share", "nexul", "computer")
			assert.ElementsMatch(t, []string{
				"runner|computer-ab12cd34|1",
				fmt.Sprintf("t3|serve --host 127.0.0.1 --port %d", th.t3.port()),
			}, got, "the runner is told T3 Code is supervised; T3 Code serves on loopback")
			var u Unit
			require.NoError(t, json.Unmarshal([]byte(readFile(t, filepath.Join(dir, "unit.json"))), &u))
			assert.True(t, u.Foreground)
			assert.Equal(t, []string{filepath.Join(th.Home, ".local", "bin", "t3"), "serve", "--host", "127.0.0.1", "--port", fmt.Sprint(th.t3.port())}, u.T3)
			assert.Equal(t, th.web.URL, u.Env["NEXUL_SERVER_URL"])
			assert.Equal(t, "personal", u.Env["NEXUL_RUNNER_MODE"])
			assert.Equal(t, "cred-laptop\n", readFile(t, filepath.Join(dir, "credential")))
			assert.Equal(t, "this-binary", readFile(t, filepath.Join(th.Home, ".local", "bin", "nexul")))
			assert.Empty(t, th.exec.calls, "no systemctl, loginctl, runuser or sudo: there is no service manager to ask")
			assert.Empty(t, th.chowned, "alice made it all herself")
			assert.NoFileExists(t, filepath.Join(th.Paths.Services, "nexul-computer.service"))
			assert.NoFileExists(t, filepath.Join(th.root, "libexec", "nexul-computer-uninstall"))
			assert.NoDirExists(t, filepath.Join(th.root, "var-lib-nexul-computer"), "no root cleanup to ask")
			ctl := filepath.Join(th.Home, ".local", "bin", "nexul")
			assert.Contains(t, th.out.String(), ctl+" run computer", "the command that starts it again, printed once")
			assert.Equal(t, 1, strings.Count(th.out.String(), "run computer"))
			assert.Contains(t, th.out.String(), "@reboot")
		})
	}
}

func TestInstallComputer_WithoutSystemd_ARemovedRunnerTakesTheInstallAwayAndEndsTheSupervisor(t *testing.T) {
	th := newTestHost(t)
	th.noSystemd(t)
	t.Setenv("NEXUL_TEST_REMOVED", "1")
	fifoLines(t)

	err := th.runInstall(t.Context(), []string{"computer", "--token", computerToken(t, th.web.URL)})

	require.NoError(t, err, "the supervisor ended by itself, with nothing stopping it")
	assert.NoDirExists(t, filepath.Join(th.Home, ".local", "share", "nexul", "computer"))
	assert.NoFileExists(t, filepath.Join(th.Home, ".local", "bin", "nexul"))
}

func TestInstallComputer_Foreground_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(th *testHost, t *testing.T)
		wantErr string
	}{
		{"on a Mac", func(th *testHost, _ *testing.T) { th.GOOS = "darwin" }, "--foreground is for Linux hosts with no systemd"},
		{"on Windows", func(th *testHost, _ *testing.T) { th.GOOS = "windows" }, "--foreground is for Linux hosts with no systemd"},
		{"under sudo, which would make the runner someone else's", func(th *testHost, t *testing.T) {
			th.Getuid = func() int { return 0 }
			t.Setenv("SUDO_USER", "alice")
		}, "run this without sudo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.noSystemd(t)
			tt.setup(th, t)

			err := th.runInstall(t.Context(), []string{"computer", "--token", computerToken(t, th.web.URL), "--foreground"})

			require.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, th.exec.calls)
			assert.Empty(t, th.instance.calls("/api/runners/enroll"), "the code is not spent")
		})
	}
}

// foregroundInstallOnDisk is what a foreground install leaves for alice: her runner's folder and the nexul command.
func foregroundInstallOnDisk(t *testing.T, th *testHost) Unit {
	t.Helper()
	p, err := th.selfPerson()
	require.NoError(t, err)
	th.Paths = th.computerPaths(p)
	dir := th.unitDir(computerUnit)
	u := Unit{Name: computerUnit, Kind: kindComputer, Host: "computer-ab12cd34", Version: "v0.3.1", Dir: dir, WorkDir: dir,
		Exec: filepath.Join(dir, "nexul-runner"), Foreground: true,
		Env: map[string]string{"NEXUL_SERVER_URL": th.web.URL, "NEXUL_CREDENTIAL_FILE": filepath.Join(dir, "credential")}}
	require.NoError(t, th.saveUnit(u))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "credential"), []byte("cred-laptop\n"), 0o600))
	require.NoError(t, os.MkdirAll(th.Paths.BinDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(th.Paths.BinDir, "nexul"), []byte("this-binary"), 0o755))
	return u
}

func TestUninstallComputer_Foreground_AsRemovedRunnerLeavesTheRequestForItsSupervisor(t *testing.T) {
	th := newTestHost(t)
	th.noSystemd(t)
	u := foregroundInstallOnDisk(t, th)

	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--detach"}))

	assert.FileExists(t, filepath.Join(u.Dir, removeRequest))
	assert.DirExists(t, u.Dir, "the supervisor removes it once the runner has exited")
	assert.Empty(t, th.exec.calls)
	assert.NoDirExists(t, th.Paths.Requests, "there is no root cleanup to ask")
	assert.Empty(t, th.instance.calls("/api/runners/self/remove"), "the instance removed the runner; it is not told again")
}

// lockHolder is another process supervising dir's runner, as `nexul run computer` would.
func lockHolder(t *testing.T, dir string) (pid int, wait func()) {
	t.Helper()
	self, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(self)
	cmd.Env = append(os.Environ(), helperRole+"=lock", "HELPER_DIR="+dir, "GORACE=atexit_sleep_ms=0")
	out, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	sc := bufio.NewScanner(out)
	for sc.Scan() && sc.Text() != "locked" {
	}
	var done bool
	wait = func() {
		if !done {
			done = true
			_ = cmd.Wait() // the holder ends by SIGTERM
		}
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill() // fails once it has exited
		wait()
	})
	return cmd.Process.Pid, wait
}

func TestRunComputer_WhileAnotherSupervisorRuns_StartsNothing(t *testing.T) {
	th := newTestHost(t)
	th.noSystemd(t)
	u := foregroundInstallOnDisk(t, th)
	pid, _ := lockHolder(t, u.Dir)

	err := th.RunComputer(t.Context())

	require.ErrorContains(t, err, fmt.Sprintf("already supervised, by pid %d", pid), "a second runner on the same credential would fight the first")
}

func TestUninstallComputer_Foreground_StopsItsSupervisorThenRemovesEverythingAndTellsTheInstance(t *testing.T) {
	th := newTestHost(t)
	th.noSystemd(t)
	u := foregroundInstallOnDisk(t, th)
	_, wait := lockHolder(t, u.Dir)

	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}))
	wait()

	remove := th.instance.calls("/api/runners/self/remove")
	require.Len(t, remove, 1)
	assert.Equal(t, "Bearer cred-laptop", remove[0].auth)
	assert.NoDirExists(t, u.Dir)
	assert.NoFileExists(t, filepath.Join(th.Paths.BinDir, "nexul"))
	assert.Empty(t, th.exec.calls, "no service to stop or remove")
}
