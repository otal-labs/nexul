package install

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/hostcred"
)

// sudoFromAlice makes the test host root under sudo, typed by alice (uid 1000, home th.Home), whose T3 Code answers
// on its own port. What root deletes as alice through runuser is really deleted.
func (th *testHost) sudoFromAlice(t *testing.T) {
	t.Helper()
	th.Getuid = func() int { return 0 }
	t.Setenv("SUDO_USER", "alice")
	withVersion(t, "v0.3.1")
	th.missing("t3")
	th.t3.up.Store(true)
	th.aliceHas(t, ".t3/userdata/server-runtime.json", fmt.Sprintf(`{"version":1,"port":%d}`, th.t3.port()), 0o600)
	th.exec.onRun = func(line string) {
		for _, rm := range []string{"runuser -u alice -- rm -rf ", "runuser -u alice -- rm -f "} {
			if paths, ok := strings.CutPrefix(line, rm); ok {
				for _, path := range strings.Fields(paths) {
					require.NoError(t, os.RemoveAll(path))
				}
			}
		}
	}
}

// aliceHas puts a file of alice's own in her home, owned by her as everything she made is.
func (th *testHost) aliceHas(t *testing.T, rel, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(th.Home, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
	for p := path; p != th.Home; p = filepath.Dir(p) {
		th.chowned[p] = "1000:1000"
	}
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

// assertHomeIsAlices fails for anything under alice's home the install made and left root's.
func assertHomeIsAlices(t *testing.T, th *testHost) {
	t.Helper()
	err := filepath.WalkDir(th.Home, func(path string, _ fs.DirEntry, err error) error {
		if err != nil || path == th.Home {
			return err
		}
		assert.Equal(t, "1000:1000", th.chowned[path], path)
		return nil
	})
	if !os.IsNotExist(err) {
		require.NoError(t, err)
	}
}

func TestInstallComputer_RunsTheRunnerAsASystemServiceOfThePersonWhoTypedSudo(t *testing.T) {
	th := newTestHost(t)
	th.sudoFromAlice(t)

	require.NoError(t, th.installComputer(t))

	enroll := th.instance.calls("/api/runners/enroll")
	require.Len(t, enroll, 1)
	assert.Equal(t, map[string]string{"token": computerToken(t, th.web.URL+"/"), "os": "linux", "arch": "amd64", "version": "v0.3.1", "machine": "box-1"}, enroll[0].body,
		"the whole token goes to the instance it names, which checks it; the code names the runner, so no name is sent")
	dir := filepath.Join(th.Home, ".local", "share", "nexul", "computer")
	ctl := filepath.Join(th.Home, ".local", "bin", "nexul")
	assert.Equal(t, "this-binary", readFile(t, ctl), "the nexul command lands in alice's ~/.local/bin")
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
	unitFile := filepath.Join(th.Paths.Services, "nexul-computer.service")
	assert.Equal(t, `[Unit]
Description=Nexul computer runner
Wants=network-online.target
After=network-online.target

[Service]
User=alice
EnvironmentFile=`+filepath.Join(dir, "env")+`
WorkingDirectory=`+dir+`
ExecStart=`+filepath.Join(dir, "nexul-runner")+`
RestartPreventExitStatus=0
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, readFile(t, unitFile))
	assert.Equal(t, []string{"systemctl daemon-reload", "systemctl enable --now nexul-computer-cleanup.path",
		"systemctl daemon-reload", "systemctl enable nexul-computer.service", "systemctl restart nexul-computer.service"},
		th.exec.callsTo("systemctl"))
	assertHomeIsAlices(t, th)
	assert.NotContains(t, th.chowned, unitFile, "the service definition stays root's")
	requests := filepath.Join(th.root, "var-lib-nexul-computer")
	assert.Equal(t, `[Unit]
Description=Remove the Nexul computer runner when it asks

[Path]
PathExists=`+filepath.Join(requests, "remove-requested")+`
Unit=nexul-computer-cleanup.service

[Install]
WantedBy=multi-user.target
`, readFile(t, filepath.Join(th.Paths.Services, "nexul-computer-cleanup.path")))
	cleanupExe := filepath.Join(th.root, "libexec", "nexul-computer-uninstall")
	assert.Equal(t, `[Unit]
Description=Remove the Nexul computer runner

[Service]
Type=oneshot
ExecStart=`+cleanupExe+` uninstall computer --cleanup-for alice
`, readFile(t, filepath.Join(th.Paths.Services, "nexul-computer-cleanup.service")))
	assert.Equal(t, "this-binary", readFile(t, cleanupExe), "the cleanup runs root's own copy, never the one in alice's home")
	assert.NotContains(t, th.chowned, cleanupExe)
	assert.Equal(t, "1000:1000", th.chowned[requests], "only the request folder is alice's to write in")
	assert.False(t, th.exec.ran("loginctl"), "a system service needs no lingering")
	assert.False(t, th.exec.ran("docker"), "a computer needs no Docker")
	assert.NoDirExists(t, filepath.Join(th.root, "opt"), "nothing goes where the deploy services live")
	assert.Contains(t, th.out.String(), "Remove it      nexul uninstall computer")
}

func TestInstallComputer_InstallsOnlyForAPersonWhoTypedSudo(t *testing.T) {
	tests := []struct {
		name     string
		uid      int
		sudoUser string
		wantErr  string
	}{
		{"run without sudo", 1000, "", "run this with sudo"},
		{"a root login, with no SUDO_USER", 0, "", "from your own account with sudo, not as root"},
		{"sudo from root", 0, "root", "from your own account with sudo, not as root"},
		{"an account this machine cannot look up", 0, "bob", "look up bob"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			withVersion(t, "v0.3.1")
			th.Getuid = func() int { return tt.uid }
			t.Setenv("SUDO_USER", tt.sudoUser)

			require.ErrorContains(t, th.installComputer(t), tt.wantErr)

			assert.Empty(t, th.exec.calls)
			assert.Empty(t, th.instance.calls("/api/runners/enroll"), "the code is not spent")
			assert.NoDirExists(t, th.Home)
			assert.NoFileExists(t, filepath.Join(th.Paths.Services, "nexul-computer.service"))
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
		{"another account's runner on this computer", func(th *testHost) {
			require.NoError(t, os.WriteFile(filepath.Join(th.Paths.Services, "nexul-computer.service"), []byte("[Unit]\n"), 0o644))
		}, nil, "installed for another account"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.sudoFromAlice(t)
			tt.setup(th)
			args := tt.args
			if args == nil {
				args = []string{"computer", "--token", computerToken(t, th.web.URL)}
			}

			require.ErrorContains(t, th.runInstall(t.Context(), args), tt.wantErr)

			assert.NoDirExists(t, filepath.Join(th.Home, ".local", "share", "nexul", "computer"), "a failed install leaves no runner behind")
			assertHomeIsAlices(t, th)
		})
	}

	t.Run("twice", func(t *testing.T) {
		th := newTestHost(t)
		th.sudoFromAlice(t)
		require.NoError(t, th.installComputer(t))
		require.ErrorContains(t, th.installComputer(t), "already has its runner")
		assert.Len(t, th.instance.calls("/api/runners/enroll"), 1, "the second code is not spent")
	})
}

// userServiceInstall lays out a runner as an earlier release installed it: its record under alice's home and a
// systemd user unit, with no system service.
func userServiceInstall(t *testing.T, th *testHost) (dir, userUnit string) {
	t.Helper()
	dir = filepath.Join(th.Home, ".local", "share", "nexul", "computer")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	cred := filepath.Join(dir, "credential")
	require.NoError(t, os.WriteFile(cred, []byte("cred-laptop\n"), 0o600))
	u := Unit{Name: "nexul-computer", Kind: kindComputer, Host: "computer-ab12cd34", Version: "v0.3.0", Dir: dir, WorkDir: dir,
		Exec: filepath.Join(dir, "nexul-runner"), Env: map[string]string{"NEXUL_SERVER_URL": th.web.URL, "NEXUL_CREDENTIAL_FILE": cred}}
	data, err := json.Marshal(u)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unit.json"), data, 0o600))
	userUnit = filepath.Join(th.Home, ".config", "systemd", "user", "nexul-computer.service")
	require.NoError(t, os.MkdirAll(filepath.Dir(userUnit), 0o755))
	require.NoError(t, os.WriteFile(userUnit, []byte("[Unit]\n"), 0o644))
	return dir, userUnit
}

func TestInstallComputer_MovesAUserServiceToTheSystemServiceKeepingItsRunner(t *testing.T) {
	th := newTestHost(t)
	th.sudoFromAlice(t)
	dir, userUnit := userServiceInstall(t, th)

	require.NoError(t, th.installComputer(t))

	assert.Empty(t, th.instance.calls("/api/runners/enroll"), "the runner keeps its credential, so the computer keeps its row")
	assert.Equal(t, "cred-laptop\n", readFile(t, filepath.Join(dir, "credential")))
	assert.True(t, th.exec.ran("runuser -u alice -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user disable --now nexul-computer.service"))
	assert.NoFileExists(t, userUnit)
	assert.Contains(t, readFile(t, filepath.Join(th.Paths.Services, "nexul-computer.service")), "\nUser=alice\n")
	assert.Contains(t, readFile(t, filepath.Join(dir, "unit.json")), `"user": "alice"`)
	assert.Equal(t, "1000:1000", th.chowned[filepath.Join(dir, "unit.json")], "the rewritten record stays alice's")
	assert.Contains(t, th.out.String(), "moved from your user service")
	assert.FileExists(t, filepath.Join(th.Paths.Services, "nexul-computer-cleanup.path"), "a moved runner can ask for its removal too")
}

// assertNothingLeft fails for any piece of a computer's install still on the machine.
func assertNothingLeft(t *testing.T, th *testHost) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(th.Paths.Services, "nexul-computer.service"),
		filepath.Join(th.Paths.Services, "nexul-computer-cleanup.path"),
		filepath.Join(th.Paths.Services, "nexul-computer-cleanup.service"),
		filepath.Join(th.root, "libexec", "nexul-computer-uninstall"),
		filepath.Join(th.root, "var-lib-nexul-computer"),
		filepath.Join(th.Home, ".local", "share", "nexul", "computer"),
		filepath.Join(th.Home, ".local", "bin", "nexul"),
		filepath.Join(th.Home, ".config", "systemd", "user", "nexul-computer.service"),
	} {
		assert.NoFileExists(t, path)
		assert.NoDirExists(t, path)
	}
}

func TestUninstallComputer_UnderSudo_RemovesEverythingAndTellsTheInstance(t *testing.T) {
	th := newTestHost(t)
	th.sudoFromAlice(t)
	require.NoError(t, th.installComputer(t))

	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}))

	remove := th.instance.calls("/api/runners/self/remove")
	require.Len(t, remove, 1)
	assert.Equal(t, "Bearer cred-laptop", remove[0].auth)
	assertNothingLeft(t, th)
	assert.True(t, th.exec.ran("runuser -u alice -- rm -rf"), "alice's files go as alice, so a link in her home leads root nowhere")
	assert.Contains(t, th.out.String(), "the instance was told")
}

func TestUninstallComputer_RemovesAUserServiceAnEarlierReleaseLeft(t *testing.T) {
	th := newTestHost(t)
	th.sudoFromAlice(t)
	userServiceInstall(t, th)

	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}))

	assert.True(t, th.exec.ran("runuser -u alice -- env XDG_RUNTIME_DIR=/run/user/1000 systemctl --user disable --now nexul-computer.service"))
	assertNothingLeft(t, th)
}

func TestUninstallComputer_TheCleanupRemovesOnlyTheInstallWhateverTheRequestSays(t *testing.T) {
	th := newTestHost(t)
	th.sudoFromAlice(t)
	require.NoError(t, th.installComputer(t))
	elsewhere := filepath.Join(th.root, "etc", "precious")
	require.NoError(t, os.MkdirAll(elsewhere, 0o755))
	request := filepath.Join(th.root, "var-lib-nexul-computer", "remove-requested")
	require.NoError(t, os.WriteFile(request, []byte(elsewhere+"\n--cleanup-for bob\n"), 0o600))
	secret := filepath.Join(elsewhere, "secret")
	require.NoError(t, os.WriteFile(secret, []byte("root's secret"), 0o600))
	record := filepath.Join(th.Home, ".local", "share", "nexul", "computer", "unit.json")
	require.NoError(t, os.WriteFile(record, []byte(`{"name":"nexul-computer","dir":"`+elsewhere+`","env":{"NEXUL_CREDENTIAL_FILE":"`+secret+`","NEXUL_SERVER_URL":"`+th.web.URL+`"}}`), 0o600))
	t.Setenv("SUDO_USER", "")
	th.exec.calls = nil

	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--cleanup-for", "alice"}))

	assertNothingLeft(t, th)
	assert.FileExists(t, secret, "neither the request nor alice's record steers what root removes")
	assert.Empty(t, th.instance.calls("/api/runners/self/remove"), "root sends no credential anywhere alice's record names")
	require.NoError(t, th.runUninstall(t.Context(), []string{"computer", "--cleanup-for", "alice"}), "a second run finds nothing left and is fine")
}

func TestUninstallComputer_AsThePerson_AsksTheCleanup(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantTold int
	}{
		{"a removed runner, with --detach", []string{"computer", "--detach"}, 0},
		{"the person, without sudo", []string{"computer", "--yes"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.sudoFromAlice(t)
			require.NoError(t, th.installComputer(t))
			th.Getuid = func() int { return 1000 }
			th.exec.calls = nil

			require.NoError(t, th.runUninstall(t.Context(), tt.args))

			assert.FileExists(t, filepath.Join(th.root, "var-lib-nexul-computer", "remove-requested"))
			assert.Empty(t, th.exec.calls, "no sudo is tried: the root path unit does the removal")
			assert.Len(t, th.instance.calls("/api/runners/self/remove"), tt.wantTold)
		})
	}
}

func TestUninstallComputer_Refusals(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.3.1")
	t.Setenv("SUDO_USER", "")
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}), "not as root")
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "--cleanup-for", "bob"}), "look up bob")
	th.Getuid = func() int { return 1000 }
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "--cleanup-for", "alice"}), "runs as root")
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "--detach"}), "sudo ~/.local/bin/nexul uninstall computer",
		"a computer installed without the cleanup says how to remove it")
	th.sudoFromAlice(t)
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "--yes"}), "no Nexul runner installed for alice")
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer", "extra"}), `"extra"`)
	require.NoError(t, th.installComputer(t))
	require.ErrorContains(t, th.runUninstall(t.Context(), []string{"computer"}), "--yes")
	assert.FileExists(t, filepath.Join(th.Home, ".local", "share", "nexul", "computer", "unit.json"))
}

func TestInstallHost_AComputersCodeIsRefusedInTheInstancesWords(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.3.1")
	th.instance.status["/api/runners/enroll"] = http.StatusConflict
	th.instance.body = map[string]string{"/api/runners/enroll": `{"message":"conflict: this code adds a computer; run it with nexul install computer","code":"computer_code"}`}

	err := th.InstallHost(t.Context(), HostOptions{Kind: kindRunner, Server: th.web.URL, Name: "edge", Code: "nxe_code"})

	require.ErrorContains(t, err, "this code adds a computer; run it with nexul install computer")
}
