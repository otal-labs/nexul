package install

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hostInstalled installs a runner or automations host enrolled with the fake instance.
func (th *testHost) hostInstalled(t *testing.T, kind, name string) {
	t.Helper()
	withVersion(t, "v0.2.1")
	require.NoError(t, th.InstallHost(t.Context(), HostOptions{Kind: kind, Server: th.web.URL + "/", Name: name, Code: "nxe_code"}))
	th.out.Reset()
	th.exec.calls = nil
}

func TestHostOptionsValidate(t *testing.T) {
	ok := HostOptions{Kind: kindRunner, Server: "https://nexul.example.com", Name: "edge-1", Code: "nxe_x"}
	tests := []struct {
		name    string
		change  func(*HostOptions)
		wantErr string
	}{
		{"valid", func(*HostOptions) {}, ""},
		{"upper case name", func(o *HostOptions) { o.Name = "Edge" }, "not a valid name"},
		{"leading dash", func(o *HostOptions) { o.Name = "-edge" }, "not a valid name"},
		{"33 characters", func(o *HostOptions) { o.Name = "a23456789012345678901234567890123" }, "not a valid name"},
		{"path in the name", func(o *HostOptions) { o.Name = "../etc" }, "not a valid name"},
		{"server without a scheme", func(o *HostOptions) { o.Server = "nexul.example.com" }, "http:// or https://"},
		{"no code", func(o *HostOptions) { o.Code = "" }, "--code is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := ok
			tt.change(&o)
			err := o.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestInstallHost_Runner_EnrollsAndRunsAsRootWithItsOwnDirectory(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")

	require.NoError(t, th.runInstall(t.Context(), []string{"runner", "--server", th.web.URL + "/", "--name", "edge", "--code", "nxe_code", "--git-token", "ghp_x"}))

	dir := filepath.Join(th.Paths.UnitRoot, "runner-edge")
	enroll := th.instance.calls("/api/runners/enroll")
	require.Len(t, enroll, 1)
	assert.Equal(t, map[string]string{"code": "nxe_code", "name": "edge", "os": "linux", "arch": "amd64", "version": "v0.2.1", "stack_root": dir, "machine": "box-1"}, enroll[0].body)
	assert.Equal(t, "nexul-runner-binary", readFile(t, filepath.Join(dir, "nexul-runner")))
	assert.Equal(t, "cred-edge\n", readFile(t, filepath.Join(dir, "credential")))
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
		"NEXUL_RUNNER_NAME":     "edge",
		"NEXUL_STACK_ROOT":      dir,
		"NEXUL_GIT_TOKEN":       "ghp_x",
		"NEXUL_CTL":             filepath.Join(th.Paths.BinDir, "nexul"),
	}, env)
	assert.Equal(t, `[Unit]
Description=Nexul runner edge
Wants=network-online.target docker.service
After=network-online.target docker.service

[Service]
EnvironmentFile=`+filepath.Join(dir, "env")+`
WorkingDirectory=`+dir+`
ExecStart=`+filepath.Join(dir, "nexul-runner")+`
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, readFile(t, filepath.Join(th.Paths.Services, "nexul-runner-edge.service")))
	assert.True(t, th.exec.ran("docker version"), "a runner needs Docker")
	assert.Equal(t, []string{"systemctl daemon-reload", "systemctl enable nexul-runner-edge.service", "systemctl restart nexul-runner-edge.service"}, th.exec.callsTo("systemctl"))
	assert.Contains(t, th.out.String(), "nexul uninstall runner edge")
}

func TestInstallHost_Automations_RunsAsNexulWithItsSudoersRule(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")

	require.NoError(t, th.runInstall(t.Context(), []string{"automations", "--server", th.web.URL, "--name", "jobs", "--code", "nxe_code"}))

	dir := filepath.Join(th.Paths.UnitRoot, "automations-jobs")
	enroll := th.instance.calls("/api/automation-hosts/enroll")
	require.Len(t, enroll, 1)
	assert.NotContains(t, enroll[0].body, "stack_root")
	assert.Equal(t, "box-1", enroll[0].body["machine"], "filed under the host it runs on")
	assert.Contains(t, th.out.String(), "The automations host jobs is running")
	env, err := readEnvFile(filepath.Join(dir, "env"))
	require.NoError(t, err)
	ctl := filepath.Join(th.Paths.BinDir, "nexul")
	assert.Equal(t, map[string]string{
		"NEXUL_SERVER_URL":            th.web.URL,
		"NEXUL_CREDENTIAL_FILE":       filepath.Join(dir, "credential"),
		"NEXUL_AUTOMATIONS_HOST_NAME": "jobs",
		"NEXUL_CTL":                   ctl,
	}, env)
	unit := readFile(t, filepath.Join(th.Paths.Services, "nexul-automations-jobs.service"))
	assert.Contains(t, unit, "User=nexul\nGroup=nexul\n")
	assert.NotContains(t, unit, "docker.service")
	sudoers := filepath.Join(th.Paths.Sudoers, "nexul-automations-jobs")
	assert.Equal(t, "nexul ALL=(root) NOPASSWD: "+ctl+" uninstall automations jobs --detach\n", readFile(t, sudoers))
	info, err := os.Stat(sudoers)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o440), info.Mode().Perm())
	assert.True(t, th.exec.ran("visudo -cf "+sudoers+".new"))
	assert.True(t, th.exec.ran("chown -R nexul:nexul "+dir))
	assert.False(t, th.exec.ran("docker"), "an automations host does not need Docker")
}

func TestInstallHost_Failures(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{"an unknown, used or expired code", http.StatusUnauthorized, "unknown, already used or expired"},
		{"a code made for another name", http.StatusConflict, "made for a different name"},
		{"any other answer", http.StatusInternalServerError, "500"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			withVersion(t, "v0.2.1")
			th.instance.status["/api/runners/enroll"] = tt.status

			err := th.InstallHost(t.Context(), HostOptions{Kind: kindRunner, Server: th.web.URL, Name: "edge", Code: "nxe_code"})

			require.ErrorContains(t, err, tt.wantErr)
			assert.NoDirExists(t, filepath.Join(th.Paths.UnitRoot, "runner-edge"), "a failed install leaves nothing behind")
			assert.False(t, th.exec.ran("systemctl"))
		})
	}

	t.Run("an answer without a credential", func(t *testing.T) {
		th := newTestHost(t)
		withVersion(t, "v0.2.1")
		th.instance.status["/api/automation-hosts/enroll"] = http.StatusOK
		err := th.InstallHost(t.Context(), HostOptions{Kind: kindAutomations, Server: th.web.URL, Name: "jobs", Code: "nxe_code"})
		require.ErrorContains(t, err, "holds no credential")
	})

	t.Run("an unreachable instance", func(t *testing.T) {
		th := newTestHost(t)
		withVersion(t, "v0.2.1")
		err := th.InstallHost(t.Context(), HostOptions{Kind: kindAutomations, Server: "http://127.0.0.1:1", Name: "jobs", Code: "nxe_code"})
		require.ErrorContains(t, err, "enroll with http://127.0.0.1:1")
	})

	t.Run("a sudoers rule visudo refuses", func(t *testing.T) {
		th := newTestHost(t)
		withVersion(t, "v0.2.1")
		th.set("visudo", "", errors.New("syntax error"))
		err := th.InstallHost(t.Context(), HostOptions{Kind: kindAutomations, Server: th.web.URL, Name: "jobs", Code: "nxe_code"})
		require.ErrorContains(t, err, "did not validate")
		assert.NoFileExists(t, filepath.Join(th.Paths.Sudoers, "nexul-automations-jobs"))
		assert.NoFileExists(t, filepath.Join(th.Paths.Sudoers, "nexul-automations-jobs.new"))
	})

	t.Run("the same name twice", func(t *testing.T) {
		th := newTestHost(t)
		th.hostInstalled(t, kindRunner, "edge")
		err := th.InstallHost(t.Context(), HostOptions{Kind: kindRunner, Server: th.web.URL, Name: "edge", Code: "nxe_other"})
		require.ErrorContains(t, err, "already installed here")
		assert.FileExists(t, filepath.Join(th.Paths.UnitRoot, "runner-edge", "unit.json"), "the installed runner is untouched")
	})

	t.Run("not root", func(t *testing.T) {
		th := newTestHost(t)
		th.Getuid = func() int { return 1000 }
		err := th.InstallHost(t.Context(), HostOptions{Kind: kindRunner, Server: th.web.URL, Name: "edge", Code: "nxe_code"})
		require.ErrorContains(t, err, "as root")
	})

	t.Run("a stray argument", func(t *testing.T) {
		th := newTestHost(t)
		require.ErrorContains(t, th.runInstall(t.Context(), []string{"runner", "edge"}), `"edge"`)
	})
}

func TestUninstallHost_TellsTheInstanceAndRemovesEverything(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindAutomations, "jobs")

	require.NoError(t, th.runUninstall(t.Context(), []string{"automations", "jobs", "--yes"}))

	remove := th.instance.calls("/api/automation-hosts/self/remove")
	require.Len(t, remove, 1)
	assert.Equal(t, "Bearer cred-jobs", remove[0].auth)
	assert.True(t, th.exec.ran("systemctl disable --now nexul-automations-jobs.service"))
	assert.NoFileExists(t, filepath.Join(th.Paths.Services, "nexul-automations-jobs.service"))
	assert.NoFileExists(t, filepath.Join(th.Paths.Sudoers, "nexul-automations-jobs"))
	assert.NoDirExists(t, filepath.Join(th.Paths.UnitRoot, "automations-jobs"))
	assert.Contains(t, th.out.String(), "the instance was told")
}

func TestUninstallHost_TheInstanceBeingGoneDoesNotStopTheRemoval(t *testing.T) {
	tests := []struct {
		name  string
		setup func(th *testHost)
		want  string
	}{
		{"already removed there", func(th *testHost) { th.instance.status["/api/runners/self/remove"] = http.StatusUnauthorized }, "had already removed it"},
		{"an error there", func(th *testHost) { th.instance.status["/api/runners/self/remove"] = http.StatusBadGateway }, "502"},
		{"unreachable", func(th *testHost) { th.web.Close() }, "was not told"},
		{"no credential", func(th *testHost) {
			_ = os.Remove(filepath.Join(th.Paths.UnitRoot, "runner-edge", "credential"))
		}, "no credential"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.hostInstalled(t, kindRunner, "edge")
			tt.setup(th)

			require.NoError(t, th.UninstallHost(t.Context(), kindRunner, "edge", true))

			assert.NoDirExists(t, filepath.Join(th.Paths.UnitRoot, "runner-edge"))
			assert.Contains(t, th.out.String(), tt.want)
		})
	}
}

func TestUninstallHost_Refusals(t *testing.T) {
	th := newTestHost(t)
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"runner"}), "usage: nexul uninstall runner <name>")
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"runner", "--yes"}), "usage")
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"runner", "edge", "extra"}), `"extra"`)
	require.ErrorContains(t, th.UninstallHost(t.Context(), kindRunner, "edge", true), `no runner named "edge"`)
	require.ErrorContains(t, th.UninstallHost(t.Context(), kindRunner, "../../etc", true), "no runner named")
	th.hostInstalled(t, kindRunner, "edge")
	require.ErrorContains(t, th.UninstallHost(t.Context(), kindRunner, "edge", false), "--yes")
	assert.DirExists(t, filepath.Join(th.Paths.UnitRoot, "runner-edge"))
}

func TestUninstallHost_Detach(t *testing.T) {
	t.Run("root on Linux runs it in a transient unit with --yes", func(t *testing.T) {
		th := newTestHost(t)
		require.NoError(t, th.runUninstall(t.Context(), []string{"runner", "edge", "--detach"}))
		calls := th.exec.callsTo("systemd-run")
		require.Len(t, calls, 1)
		self, _ := th.Executable()
		assert.Regexp(t, `^systemd-run --unit nexul-uninstall-[a-z2-7]{8} --collect --quiet `+self+` uninstall runner edge --yes$`, calls[0])
	})
	t.Run("a non-root caller on Linux goes through sudo -n exactly as called", func(t *testing.T) {
		th := newTestHost(t)
		th.Getuid = func() int { return 998 }
		require.NoError(t, th.runUninstall(t.Context(), []string{"automations", "jobs", "--detach"}))
		self, _ := th.Executable()
		assert.Equal(t, []string{"sudo -n " + self + " uninstall automations jobs --detach"}, th.exec.calls)
	})
	t.Run("macOS starts a detached process", func(t *testing.T) {
		th := newTestHost(t)
		th.GOOS = "darwin"
		require.NoError(t, th.runUninstall(t.Context(), []string{"runner", "edge", "--detach"}))
		self, _ := th.Executable()
		log := filepath.Join(th.Paths.UnitRoot, "nexul-uninstall.log")
		assert.Equal(t, [][]string{{self, log, "uninstall", "runner", "edge", "--yes"}}, th.detached)
		assert.Empty(t, th.exec.calls)
	})
	t.Run("a failed start is reported", func(t *testing.T) {
		th := newTestHost(t)
		th.GOOS = "windows"
		th.StartDetached = func(string, []string, string) error { return errors.New("access denied") }
		require.ErrorContains(t, th.runUpgrade(t.Context(), []string{"--detach"}), "access denied")
	})
}

func TestIsDetachFlag(t *testing.T) {
	for _, arg := range []string{"--detach", "-detach", "--detach=true"} {
		assert.True(t, isDetachFlag(arg), arg)
	}
	for _, arg := range []string{"detach", "--detached", "--yes"} {
		assert.False(t, isDetachFlag(arg), arg)
	}
}

func TestEnsureServiceUser_UseraddFailure_IsReported(t *testing.T) {
	th := newTestHost(t)
	th.set("id -u nexul", "", errors.New("no such user"))
	th.set("useradd", "", errors.New("useradd: cannot lock /etc/passwd"))
	_, err := th.ensureServiceUser(t.Context())
	require.ErrorContains(t, err, "create the nexul user: useradd: cannot lock")
}
