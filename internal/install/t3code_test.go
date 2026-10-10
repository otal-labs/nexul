package install

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// t3State arranges what T3 Code alice has before the install looks.
type t3State func(t *testing.T, th *testHost)

var (
	t3Answering        t3State = func(*testing.T, *testHost) {}
	t3ServiceAnswering t3State = func(t *testing.T, th *testHost) {
		th.aliceHas(t, ".t3/userdata/server-runtime.json", `{"version":1,"port":`+strconv.Itoa(th.t3.port())+`,"serviceManaged":true}`, 0o600)
	}
	t3Desktop t3State = func(t *testing.T, th *testHost) {
		t3Nothing(t, th)
		th.aliceHas(t, ".t3/bin/"+desktopLauncher(th), "", 0o755)
	}
	t3CommandLine t3State = func(t *testing.T, th *testHost) { t3Nothing(t, th); th.aliceHas(t, ".local/bin/t3", "", 0o755) }
	t3Nothing     t3State = func(t *testing.T, th *testHost) {
		th.t3.up.Store(false)
		require.NoError(t, os.Remove(filepath.Join(th.Home, ".t3", "userdata", "server-runtime.json")))
	}
)

func desktopLauncher(th *testHost) string {
	if th.GOOS == "windows" {
		return "t3.cmd"
	}
	return "t3"
}

var installerPath = regexp.MustCompile(`\S*/nexul-t3-[^/ ]+/install\.sh`)

// actAsAlice makes the commands the install runs as alice do what they would: T3 Code's installer puts t3 in her
// ~/.local/bin, and its service then answers. It records the installer each run was handed.
func actAsAlice(t *testing.T, th *testHost) *[]string {
	t.Helper()
	var installers []string
	th.exec.onRun = func(line string) {
		if script := installerPath.FindString(line); script != "" {
			installers = append(installers, readFile(t, script))
			th.aliceHas(t, ".local/bin/t3", "", 0o755)
		}
		if strings.HasSuffix(line, " service install") {
			th.t3.up.Store(true)
		}
	}
	return &installers
}

func TestInstallT3_ReusesWhatItFindsAndInstallsOnlyWhenThereIsNone(t *testing.T) {
	const (
		asAliceLinux  = "runuser -u alice -- env HOME={home} XDG_RUNTIME_DIR=/run/user/1000 "
		asAliceDarwin = "sudo -u alice env HOME={home} "
	)
	linuxService := []string{"loginctl enable-linger alice", "systemctl start user@1000.service", asAliceLinux + "{home}/.local/bin/t3 service install"}
	tests := []struct {
		goos      string
		name      string
		state     t3State
		wantCalls []string
		wantOut   string
	}{
		{"linux", "answering", t3Answering, nil, "running on port {port}\n"},
		{"linux", "answering under its own service", t3ServiceAnswering, []string{"loginctl enable-linger alice"}, "its service kept running after you log out"},
		{"linux", "desktop app closed", t3Desktop, nil, "Open T3 Code"},
		{"linux", "command line idle", t3CommandLine, linuxService, "running on port {port} as alice's background service"},
		{"linux", "none", t3Nothing, append([]string{"ldconfig -p", "apt-get update -qq", "apt-get install -y -qq libatomic1", asAliceLinux + "sh {installer}"}, linuxService...),
			"Pass --no-t3 to skip this."},
		{"darwin", "answering under its own service", t3ServiceAnswering, nil, "running on port {port}\n"},
		{"darwin", "desktop app closed", t3Desktop, nil, "Open T3 Code"},
		{"darwin", "command line idle", t3CommandLine, []string{asAliceDarwin + "{home}/.local/bin/t3 service install"}, "running on port {port} as alice's background service"},
		{"darwin", "none", t3Nothing, []string{asAliceDarwin + "sh {installer}", asAliceDarwin + "{home}/.local/bin/t3 service install"}, "installed, running on port {port}"},
		{"windows", "answering", t3Answering, nil, "running on port {port}\n"},
		{"windows", "desktop app closed", t3Desktop, nil, "Open T3 Code"},
		{"windows", "command line idle", t3CommandLine, nil, "start it with t3 serve"},
		{"windows", "none", t3Nothing, []string{"winget install --id T3Tools.T3Code --exact --silent --accept-source-agreements --accept-package-agreements"},
			"winget install T3Tools.T3Code). Pass --no-t3 to skip this."},
	}
	for _, tt := range tests {
		t.Run(tt.goos+", "+tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.sudoFromAlice(t)
			th.GOOS = tt.goos
			tt.state(t, th)
			installers := actAsAlice(t, th)
			alice := person{name: "alice", uid: 1000, gid: 1000, home: th.Home}
			placeholders := strings.NewReplacer("{home}", th.Home, "{port}", strconv.Itoa(th.t3.port()))

			require.NoError(t, th.installT3(t.Context(), alice, ComputerOptions{}))

			var calls []string
			for _, c := range th.exec.calls {
				calls = append(calls, installerPath.ReplaceAllString(c, "{installer}"))
			}
			var want []string
			for _, c := range tt.wantCalls {
				want = append(want, placeholders.Replace(c))
			}
			assert.Equal(t, want, calls)
			assert.Contains(t, th.out.String(), placeholders.Replace(tt.wantOut))
			for _, script := range *installers {
				assert.Equal(t, "#!/bin/sh\necho T3 Code's installer\n", script, "alice runs T3 Code's own installer")
			}
			assert.False(t, th.exec.ran("sh ") || th.exec.ran(th.Home), "nothing of T3 Code's runs as root")
		})
	}
}

// anotherPort makes the port T3 Code's own service takes one nothing listens on, so the fake's is another port.
func anotherPort(th *testHost) ComputerOptions {
	th.T3Port = 1
	return ComputerOptions{T3Port: th.t3.port()}
}

func TestInstallT3_OnAnotherPort(t *testing.T) {
	t.Run("on Linux, its service listens there through a drop-in of alice's", func(t *testing.T) {
		th := newTestHost(t)
		th.sudoFromAlice(t)
		t3Nothing(t, th)
		actAsAlice(t, th)

		require.NoError(t, th.installT3(t.Context(), person{name: "alice", uid: 1000, home: th.Home}, anotherPort(th)))

		assert.True(t, th.exec.ran("runuser -u alice -- env HOME="+th.Home+" XDG_RUNTIME_DIR=/run/user/1000 sh -c mkdir -p \"$1\" && printf '[Service]\\nEnvironment=T3CODE_PORT=%s\\n' \"$2\" >\"$1/nexul-port.conf\" sh "+
			filepath.Join(th.Home, ".config", "systemd", "user", "t3code.service.d")+" "+strconv.Itoa(th.t3.port())))
	})

	t.Run("on a Mac, whose service always takes its own port, it is refused before installing", func(t *testing.T) {
		th := newTestHost(t)
		th.sudoFromAlice(t)
		th.GOOS = "darwin"
		t3Nothing(t, th)

		err := th.installT3(t.Context(), person{name: "alice", uid: 1000, home: th.Home}, anotherPort(th))

		require.ErrorContains(t, err, "macOS service always starts on port 1")
		assert.Empty(t, th.exec.calls)
	})
}

func TestInstallComputer_WithNoT3_LeavesT3CodeAlone(t *testing.T) {
	th := newTestHost(t)
	th.sudoFromAlice(t)
	t3Nothing(t, th)

	require.NoError(t, th.runInstall(t.Context(), []string{"computer", "--token", computerToken(t, th.web.URL), "--no-t3"}))

	assert.Contains(t, th.out.String(), "T3 Code ......... skipped (--no-t3)")
	assert.NoFileExists(t, filepath.Join(th.Home, ".local", "bin", "t3"))
	for _, c := range []string{"ldconfig", "apt-get", "loginctl", "runuser -u alice -- env HOME=" + th.Home + " XDG_RUNTIME_DIR=/run/user/1000 sh"} {
		assert.False(t, th.exec.ran(c), c)
	}
	assert.Len(t, th.instance.calls("/api/runners/enroll"), 1, "the runner is still installed")
}

func TestInstallComputer_AT3CodeThatCannotBeStartedStopsBeforeTheCodeIsSpent(t *testing.T) {
	const asAlice = "runuser -u alice -- env HOME={home} XDG_RUNTIME_DIR=/run/user/1000 "
	tests := []struct {
		name    string
		fail    string
		wantErr string
	}{
		{"its installer fails", asAlice + "sh", "T3 Code's installer failed; install T3 Code yourself from https://t3.codes, or run the command again with --no-t3"},
		{"its service does not start", asAlice + "{home}/.local/bin/t3 service install", "background service did not start"},
		{"its service never answers", "", "nothing answers on port"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.sudoFromAlice(t)
			t3Nothing(t, th)
			th.exec.onRun = func(line string) {
				if installerPath.MatchString(line) {
					th.aliceHas(t, ".local/bin/t3", "", 0o755)
				}
			}
			if tt.fail != "" {
				th.set(strings.ReplaceAll(tt.fail, "{home}", th.Home), "", errors.New("exit status 1"))
			}

			err := th.runInstall(t.Context(), []string{"computer", "--token", computerToken(t, th.web.URL)})

			require.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, th.instance.calls("/api/runners/enroll"), "the code is not spent, so the same command works again")
			assert.NoDirExists(t, filepath.Join(th.Home, ".local", "share", "nexul", "computer"))
		})
	}
}
