package install

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

// asPerson makes the test host alice's own session: uid 1000 with a systemd user manager to talk to.
func (th *testHost) asPerson(t *testing.T) {
	t.Helper()
	th.Getuid = func() int { return 1000 }
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	withVersion(t, "v0.3.1")
}

// computerToken is a token as Add a computer signs it, naming server; the installer reads it but never checks it.
func computerToken(t *testing.T, server string) string {
	t.Helper()
	token, err := hostcred.SignToken(map[string]any{"server": server, "code": "nxe_code", "computer": "c-laptop", "exp": 4102444800}, []byte("instance key"))
	require.NoError(t, err)
	return token
}

func (th *testHost) installComputer(t *testing.T) error {
	t.Helper()
	return th.runInstall(t.Context(), []string{"computer", "--token", computerToken(t, th.web.URL+"/")})
}

func TestInstallComputer_RunsTheRunnerAsThePersonsOwnUserService(t *testing.T) {
	th := newTestHost(t)
	th.asPerson(t)
	th.set("loginctl show-user", "no", nil)

	require.NoError(t, th.installComputer(t))

	enroll := th.instance.calls("/api/runners/enroll")
	require.Len(t, enroll, 1)
	assert.Equal(t, map[string]string{"token": computerToken(t, th.web.URL+"/"), "os": "linux", "arch": "amd64", "version": "v0.3.1", "machine": "box-1"}, enroll[0].body,
		"the whole token goes to the instance it names, which checks it; the code names the runner, so no name is sent")
	dir := filepath.Join(th.Home, ".local", "share", "nexul", "computer")
	ctl := filepath.Join(th.Home, ".local", "bin", "nexul")
	assert.Equal(t, "this-binary", readFile(t, ctl), "the nexul command lands in ~/.local/bin")
	assert.Equal(t, "nexul-runner-binary", readFile(t, filepath.Join(dir, "nexul-runner")))
	assert.Equal(t, "cred-laptop\n", readFile(t, filepath.Join(dir, "credential")))
	for _, f := range []string{"credential", "env"} {
		info, err := os.Stat(filepath.Join(dir, f))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), f)
	}
	env, err := readEnvFile(filepath.Join(dir, "env"))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"NEXUL_SERVER_URL":      th.web.URL,
		"NEXUL_CREDENTIAL_FILE": filepath.Join(dir, "credential"),
		"NEXUL_CTL":             ctl,
		"NEXUL_RUNNER_NAME":     "computer-ab12cd34",
		"NEXUL_RUNNER_MODE":     "personal",
	}, env)
	assert.Equal(t, `[Unit]
Description=Nexul computer runner

[Service]
EnvironmentFile=`+filepath.Join(dir, "env")+`
WorkingDirectory=`+dir+`
ExecStart=`+filepath.Join(dir, "nexul-runner")+`
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
`, readFile(t, filepath.Join(th.Home, ".config", "systemd", "user", "nexul-computer.service")))
	assert.Equal(t, []string{"systemctl --user daemon-reload", "systemctl --user enable nexul-computer.service", "systemctl --user restart nexul-computer.service"},
		th.exec.callsTo("systemctl"))
	assert.Equal(t, []string{"loginctl show-user 1000 --property=Linger --value", "loginctl enable-linger"}, th.exec.callsTo("loginctl"))
	assert.False(t, th.exec.ran("docker"), "a computer needs no Docker")
	assert.False(t, th.exec.ran("sudo"), "lingering as the person needed no sudo")
	assert.NoDirExists(t, filepath.Join(th.root, "opt"), "nothing goes where root's services live")
	assert.Contains(t, th.out.String(), "nexul uninstall computer")
}

func TestInstallComputer_RefusesRootAndInstallsNothing(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.3.1")

	err := th.installComputer(t)

	require.ErrorContains(t, err, "not as root")
	assert.Empty(t, th.exec.calls)
	assert.Empty(t, th.instance.calls("/api/runners/enroll"), "the code is not spent")
	assert.NoDirExists(t, filepath.Join(th.Home, ".local"))
	assert.NoFileExists(t, filepath.Join(th.Paths.BinDir, "nexul"))
}

func TestInstallComputer_Lingering(t *testing.T) {
	tests := []struct {
		name  string
		setup func(th *testHost)
		want  string
		sudo  bool
	}{
		{"already on", func(th *testHost) { th.set("loginctl show-user", "yes", nil) }, "Lingering ....... on\n", false},
		{"turned on as the person", func(*testHost) {}, "turned on", false},
		{"turned on through passwordless sudo", func(th *testHost) {
			th.set("loginctl enable-linger", "", errors.New("access denied"))
		}, "turned on with sudo", true},
		{"left off, and said so", func(th *testHost) {
			th.set("loginctl enable-linger", "", errors.New("access denied"))
			th.set("sudo -n", "", errors.New("a password is required"))
		}, "sudo loginctl enable-linger $USER keeps it running", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.asPerson(t)
			tt.setup(th)

			require.NoError(t, th.installComputer(t), "lingering never stops the install")

			assert.Contains(t, th.out.String(), tt.want)
			assert.Equal(t, tt.sudo, th.exec.ran("sudo -n loginctl enable-linger 1000"))
			assert.True(t, th.exec.ran("systemctl --user restart nexul-computer.service"))
		})
	}
}

func TestInstallComputer_Failures(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(th *testHost)
		args    []string
		wantErr string
	}{
		{"not Linux yet", func(th *testHost) { th.GOOS = "darwin" }, nil, "runs on Linux for now"},
		{"no token", func(*testHost) {}, []string{"computer"}, "not a computer token"},
		{"a token that is no JWT", func(*testHost) {}, []string{"computer", "--token", "nxe_code"}, "not a computer token"},
		{"a token naming no address", func(*testHost) {}, []string{"computer", "--token", computerToken(t, "nexul.example.com")}, "names no instance address"},
		{"a used code", func(th *testHost) { th.instance.status["/api/runners/enroll"] = http.StatusUnauthorized }, nil, "make a new one with Add a computer"},
		{"a runner's code", func(th *testHost) {
			th.instance.status["/api/runners/enroll"] = http.StatusConflict
			th.instance.body = map[string]string{"/api/runners/enroll": `{"message":"conflict: this code enrolls a runner, not a computer","code":"runner_code"}`}
		}, nil, "refused the enrollment code: conflict: this code enrolls a runner, not a computer"},
		{"a conflict from an instance that gives no reason", func(th *testHost) {
			th.instance.status["/api/runners/enroll"] = http.StatusConflict
		}, nil, "it does not add a computer"},
		{"no user manager to run under", func(th *testHost) {
			t.Setenv("XDG_RUNTIME_DIR", "")
		}, nil, "systemd user manager is not running"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.asPerson(t)
			tt.setup(th)
			args := tt.args
			if args == nil {
				args = []string{"computer", "--token", computerToken(t, th.web.URL)}
			}

			require.ErrorContains(t, th.runInstall(t.Context(), args), tt.wantErr)

			assert.NoDirExists(t, filepath.Join(th.Home, ".local", "share", "nexul", "computer"), "a failed install leaves no runner behind")
		})
	}

	t.Run("twice", func(t *testing.T) {
		th := newTestHost(t)
		th.asPerson(t)
		require.NoError(t, th.installComputer(t))
		require.ErrorContains(t, th.installComputer(t), "already has its runner")
		assert.Len(t, th.instance.calls("/api/runners/enroll"), 1, "the second code is not spent")
	})
}

func TestInstallComputer_WaitsForTheUserManagerLingeringStarts(t *testing.T) {
	th := newTestHost(t)
	th.asPerson(t)
	t.Setenv("XDG_RUNTIME_DIR", "")
	runtime := filepath.Join(th.Paths.UserRuntime, "1000")
	th.exec.onRun = func(line string) {
		if line == "loginctl enable-linger" {
			require.NoError(t, os.MkdirAll(runtime, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(runtime, "bus"), nil, 0o600))
		}
	}

	require.NoError(t, th.installComputer(t))

	assert.Equal(t, runtime, os.Getenv("XDG_RUNTIME_DIR"), "systemctl --user reaches the manager lingering started")
}

func TestUninstallComputer_RemovesTheUserServiceAndTellsTheInstance(t *testing.T) {
	th := newTestHost(t)
	th.asPerson(t)
	require.NoError(t, th.installComputer(t))
	th.exec.calls = nil

	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}))

	remove := th.instance.calls("/api/runners/self/remove")
	require.Len(t, remove, 1)
	assert.Equal(t, "Bearer cred-laptop", remove[0].auth)
	assert.Equal(t, []string{"systemctl --user disable --now nexul-computer.service", "systemctl --user daemon-reload"}, th.exec.callsTo("systemctl"))
	assert.NoFileExists(t, filepath.Join(th.Home, ".config", "systemd", "user", "nexul-computer.service"))
	assert.NoDirExists(t, filepath.Join(th.Home, ".local", "share", "nexul", "computer"))
	assert.Contains(t, th.out.String(), "the instance was told")
}

func TestUninstallComputer_Refusals(t *testing.T) {
	th := newTestHost(t)
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}), "not as root")
	th.asPerson(t)
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}), "no Nexul runner installed")
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "extra"}), `"extra"`)
	require.NoError(t, th.installComputer(t))
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer"}), "--yes")
	assert.FileExists(t, filepath.Join(th.Home, ".local", "share", "nexul", "computer", "unit.json"))
}

func TestUninstallComputer_Detach_RunsInTheUsersOwnManager(t *testing.T) {
	th := newTestHost(t)
	th.asPerson(t)

	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--detach"}))

	self, _ := th.Executable()
	calls := th.exec.calls
	require.Len(t, calls, 1, "no sudo: the person removes their own service")
	assert.Regexp(t, `^systemd-run --user --unit nexul-uninstall-computer-[a-z2-7]{8} --collect --quiet `+self+` uninstall computer --yes$`, calls[0])
}

func TestInstallHost_AComputersCodeIsRefusedInTheInstancesWords(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.3.1")
	th.instance.status["/api/runners/enroll"] = http.StatusConflict
	th.instance.body = map[string]string{"/api/runners/enroll": `{"message":"conflict: this code adds a computer; run it with nexul install computer","code":"computer_code"}`}

	err := th.InstallHost(t.Context(), HostOptions{Kind: kindRunner, Server: th.web.URL, Name: "edge", Code: "nxe_code"})

	require.ErrorContains(t, err, "this code adds a computer; run it with nexul install computer")
}
