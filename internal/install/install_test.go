package install

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_UnknownCommand_ReturnsErrUnknownCommand(t *testing.T) {
	err := Run(t.Context(), "bogus", nil)
	require.ErrorIs(t, err, ErrUnknownCommand)
}

func TestRun_ServiceHostWithoutAUnit_IsAUsageError(t *testing.T) {
	require.ErrorContains(t, Run(t.Context(), "service-host", nil), "usage")
}

func TestServiceHost_OffWindows_Refuses(t *testing.T) {
	th := newTestHost(t)
	require.ErrorContains(t, th.serviceHost("nexul-server"), "not Windows")
}

func TestCheckUser(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		uid     int
		wantErr string
	}{
		{"linux as root", "linux", 0, ""},
		{"linux not root", "linux", 1000, "as root"},
		{"macOS as the user", "darwin", 501, ""},
		{"macOS with sudo", "darwin", 0, "not with sudo"},
		{"windows", "windows", -1, ""},
		{"another OS", "freebsd", 0, "nexul-server"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{GOOS: tt.goos, Getuid: func() int { return tt.uid }}
			err := h.checkUser()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestInstallTag(t *testing.T) {
	t.Run("the flag wins and gains a v", func(t *testing.T) {
		tag, err := installTag("0.2.1")
		require.NoError(t, err)
		assert.Equal(t, "v0.2.1", tag)
	})
	t.Run("a release build installs its own version", func(t *testing.T) {
		withVersion(t, "v0.2.1-beta.3")
		tag, err := installTag("")
		require.NoError(t, err)
		assert.Equal(t, "v0.2.1-beta.3", tag)
	})
	t.Run("a dev build without a flag has nothing to install", func(t *testing.T) {
		withVersion(t, "dev")
		_, err := installTag("")
		require.ErrorContains(t, err, "--version")
	})
}

func TestCheckPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		prev    *installed
		wantErr string
	}{
		{"free", 81, nil, ""},
		{"out of range", 70000, nil, "out of range"},
		{"held by another program", 80, nil, "--port"},
		{"held by this install", 80, &installed{Env: map[string]string{"NEXUL_PORT": "80"}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{PortFree: func(port int) bool { return port != 80 }}
			err := h.checkPort(tt.port, tt.prev)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAskPort(t *testing.T) {
	t.Run("Enter keeps the default", func(t *testing.T) {
		th := newTestHost(t)
		th.In = bufio.NewReader(strings.NewReader("\n"))
		port, err := th.askPort("Web port", 80, nil)
		require.NoError(t, err)
		assert.Equal(t, 80, port)
	})
	t.Run("a bad or busy port is asked again", func(t *testing.T) {
		th := newTestHost(t)
		th.PortFree = func(port int) bool { return port != 80 }
		th.In = bufio.NewReader(strings.NewReader("abc\n80\n8080\n"))
		port, err := th.askPort("Web port", 80, nil)
		require.NoError(t, err)
		assert.Equal(t, 8080, port)
		assert.Contains(t, th.out.String(), `"abc" is not a port number`)
		assert.Contains(t, th.out.String(), "Port 80 is already in use")
	})
	t.Run("once the default is busy, Enter takes the next free port", func(t *testing.T) {
		th := newTestHost(t)
		th.PortFree = func(port int) bool { return port != 80 && port != 8080 }
		th.In = bufio.NewReader(strings.NewReader("\n\n"))
		port, err := th.askPort("Web port", 80, nil)
		require.NoError(t, err)
		assert.Equal(t, 8081, port)
		assert.Contains(t, th.out.String(), "[8081]")
	})
	t.Run("nothing free nearby keeps the old default", func(t *testing.T) {
		th := newTestHost(t)
		th.PortFree = func(int) bool { return false }
		assert.Equal(t, 80, th.freePortAfter(80, 80))
		assert.Equal(t, 3000, th.freePortAfter(65535, 3000))
	})
	t.Run("the input ending early is an error, not a loop", func(t *testing.T) {
		th := newTestHost(t)
		_, err := th.askPort("Web port", 80, nil)
		require.ErrorContains(t, err, "read answer")
	})
}

// serverInstall runs a fresh, booted install into <root>/data/nexul and returns the directory.
func (th *testHost) serverInstall(t *testing.T) string {
	t.Helper()
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "data", "nexul")
	th.bootedServer(t, dir)
	require.NoError(t, th.Install(t.Context(), Options{Dir: dir, Port: th.webPort(t), Yes: true}))
	return dir
}

func TestInstall_FreshHost_RunsEveryUnitAsAService(t *testing.T) {
	th := newTestHost(t)
	th.set("id -u nexul", "", errors.New("no such user"))
	dir := th.serverInstall(t)
	port := th.webPort(t)

	settings, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "0.2.1", settings["NEXUL_VERSION"])
	assert.Equal(t, strconv.Itoa(port), settings["NEXUL_PORT"])
	assert.Equal(t, "15080", settings["NEXUL_LOGS_PORT"])
	assert.Equal(t, "15081", settings["NEXUL_LOGS_GRPC_PORT"])
	assert.Len(t, settings["NEXUL_LOGS_PASSWORD"], 25)
	for _, sub := range []string{"data", "logs", "stacks"} {
		assert.DirExists(t, filepath.Join(dir, sub))
	}

	opt := th.Paths.UnitRoot
	assert.Equal(t, `[Unit]
Description=Nexul server
Wants=network-online.target
After=network-online.target

[Service]
User=nexul
Group=nexul
EnvironmentFile=`+filepath.Join(opt, "server", "env")+`
WorkingDirectory=`+filepath.Join(dir, "data")+`
ExecStart=`+filepath.Join(opt, "server", "nexul-server")+`
AmbientCapabilities=CAP_NET_BIND_SERVICE
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, readFile(t, filepath.Join(th.Paths.Services, "nexul-server.service")))
	assert.Equal(t, "nexul-server-binary", readFile(t, filepath.Join(opt, "server", "nexul-server")))
	serverEnv, err := readEnvFile(filepath.Join(opt, "server", "env"))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"NEXUL_HTTP_ADDR":     ":" + strconv.Itoa(port),
		"NEXUL_DB_PATH":       filepath.Join(dir, "data", "nexul.db"),
		"NEXUL_LOG_LEVEL":     "info",
		"NEXUL_LOGS_URL":      "http://127.0.0.1:15080",
		"NEXUL_OTLP_ENDPOINT": "http://127.0.0.1:15080/openobserve/api/default",
		"NEXUL_OTLP_USER":     "nexul@nexul.local",
		"NEXUL_OTLP_TOKEN":    settings["NEXUL_LOGS_TOKEN"],
	}, serverEnv)

	logs := readFile(t, filepath.Join(th.Paths.Services, "nexul-openobserve.service"))
	assert.Contains(t, logs, "User=nexul\n")
	assert.Contains(t, logs, "MemoryMax=1G\n")
	assert.Equal(t, "openobserve-v1.0.4", readFile(t, filepath.Join(opt, "openobserve", "openobserve")))
	logsEnv, err := readEnvFile(filepath.Join(opt, "openobserve", "env"))
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", logsEnv["ZO_HTTP_ADDR"])
	assert.Equal(t, "15080", logsEnv["ZO_HTTP_PORT"])
	assert.Equal(t, "127.0.0.1", logsEnv["ZO_GRPC_ADDR"])
	assert.Equal(t, "15081", logsEnv["ZO_GRPC_PORT"])
	assert.Equal(t, "/openobserve", logsEnv["ZO_BASE_URI"])
	assert.Equal(t, filepath.Join(dir, "logs"), logsEnv["ZO_DATA_DIR"])
	assert.Equal(t, settings["NEXUL_LOGS_PASSWORD"], logsEnv["ZO_ROOT_USER_PASSWORD"])

	runner := th.instance.calls("/api/runners/enroll")
	require.Len(t, runner, 1)
	assert.Equal(t, map[string]string{"code": "nxe_runner", "name": "instance", "os": "linux", "arch": "amd64", "version": "v0.2.1", "stack_root": dir, "machine": "box-1"}, runner[0].body)
	require.Len(t, th.instance.calls("/api/automation-hosts/enroll"), 1)
	assert.FileExists(t, filepath.Join(th.Paths.Services, "nexul-runner-instance.service"))
	assert.FileExists(t, filepath.Join(th.Paths.Services, "nexul-automations-instance.service"))
	runnerEnv, err := readEnvFile(filepath.Join(opt, "runner-instance", "env"))
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:"+strconv.Itoa(port), runnerEnv["NEXUL_SERVER_URL"])

	assert.True(t, th.exec.ran("useradd --system --user-group --no-create-home --shell /usr/sbin/nologin nexul"))
	assert.True(t, th.exec.ran("chown -R nexul:nexul "+filepath.Join(dir, "data")+" "+filepath.Join(dir, "logs")))
	assert.Equal(t, []string{
		"systemctl restart nexul-openobserve.service",
		"systemctl restart nexul-server.service",
		"systemctl restart nexul-runner-instance.service",
		"systemctl restart nexul-automations-instance.service",
	}, th.exec.callsTo("systemctl restart"))
	assert.False(t, th.exec.ran("docker compose --project-directory"), "nothing runs in compose")
	assert.Equal(t, "this-binary", readFile(t, filepath.Join(th.Paths.BinDir, "nexul")))
	conf, err := readEnvFile(th.Paths.Config)
	require.NoError(t, err)
	assert.Equal(t, dir, conf["NEXUL_DIR"])

	out := th.out.String()
	assert.Contains(t, out, "Nexul v0.2.1 is running.")
	assert.Contains(t, out, "openobserve/  user nexul@nexul.local")
	assert.Contains(t, out, settings["NEXUL_LOGS_PASSWORD"])
}

func TestInstall_Rerun_KeepsSecretsPortsAndTheBundledHosts(t *testing.T) {
	th := newTestHost(t)
	dir := th.serverInstall(t)
	first, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)

	th.PortFree = func(int) bool { return false }
	require.NoError(t, th.Install(t.Context(), Options{Yes: true}))

	second, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Len(t, th.instance.calls("/api/runners/enroll"), 1, "the bundled runner is not enrolled twice")
	assert.Contains(t, th.out.String(), "nexul-runner-instance, already installed")
}

func TestInstall_KeptDirectory_ReusesItsSecretsAndPorts(t *testing.T) {
	th := newTestHost(t)
	dir := th.serverInstall(t)
	first, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	require.NoError(t, th.Uninstall(t.Context(), false, true))
	assert.NoFileExists(t, th.Paths.Config)

	th.PortFree = func(int) bool { return false }
	require.NoError(t, th.Install(t.Context(), Options{Dir: dir, Yes: true}))

	second, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestInstall_ComposeInstallDirectory_IsRefused(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "old")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services: {}\n"), 0o644))

	err := th.Install(t.Context(), Options{Dir: dir, Port: th.webPort(t), Yes: true})

	require.ErrorContains(t, err, "run that release's `nexul uninstall` first")
	assert.Empty(t, th.exec.calls)
}

func TestInstall_FailingStep_StopsTheRest(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	th.set("systemctl restart nexul-openobserve.service", "", errors.New("unit failed"))

	err := th.Install(t.Context(), Options{Dir: filepath.Join(th.root, "n"), Port: th.webPort(t), Yes: true})

	require.ErrorContains(t, err, "logs: unit failed")
	assert.NoFileExists(t, filepath.Join(th.Paths.Services, "nexul-server.service"), "the server must not start without its log store")
	assert.Contains(t, th.out.String(), "failed")
}

func TestInstall_ServerWritesNoCode_FailsNamingTheFile(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	th.HealthTimeout = 20 * time.Millisecond
	dir := filepath.Join(th.root, "n")

	err := th.Install(t.Context(), Options{Dir: dir, Port: th.webPort(t), Yes: true})

	require.ErrorContains(t, err, filepath.Join(dir, "data", "enroll", "runner-instance"))
}

func TestInstall_NotRoot_ChangesNothing(t *testing.T) {
	th := newTestHost(t)
	th.Getuid = func() int { return 1000 }
	require.Error(t, th.Install(t.Context(), Options{Yes: true}))
	assert.Empty(t, th.exec.calls)
}

func TestInstall_PortTaken_FailsBeforeAnyStep(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	th.PortFree = func(int) bool { return false }
	require.ErrorContains(t, th.Install(t.Context(), Options{Yes: true}), "already in use")
	assert.Empty(t, th.exec.calls)
}

func TestInstall_Interactive_AsksThenInstalls(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "asked")
	th.bootedServer(t, dir)
	th.Interactive = true
	th.In = bufio.NewReader(strings.NewReader(dir + "\n" + strconv.Itoa(th.webPort(t)) + "\n"))

	require.NoError(t, th.Install(t.Context(), Options{}))

	assert.FileExists(t, filepath.Join(dir, ".env"))
	assert.Contains(t, th.out.String(), "Press Enter to accept the default.")
}

func TestRunInstall_Flags(t *testing.T) {
	t.Run("an unknown flag is refused", func(t *testing.T) {
		th := newTestHost(t)
		require.Error(t, th.runInstall(t.Context(), []string{"--bogus"}))
	})
	t.Run("the logs port is not a flag", func(t *testing.T) {
		th := newTestHost(t)
		require.Error(t, th.runInstall(t.Context(), []string{"--logs-port", "5080"}))
	})
	t.Run("a stray argument is refused", func(t *testing.T) {
		th := newTestHost(t)
		require.ErrorContains(t, th.runInstall(t.Context(), []string{"server", "extra"}), `"extra"`)
	})
	t.Run("server is the default component", func(t *testing.T) {
		th := newTestHost(t)
		withVersion(t, "v0.2.1")
		dir := filepath.Join(th.root, "s")
		th.bootedServer(t, dir)
		require.NoError(t, th.runInstall(t.Context(), []string{"server", "--dir", dir, "--port", strconv.Itoa(th.webPort(t)), "--yes"}))
		assert.FileExists(t, filepath.Join(th.Paths.Services, "nexul-server.service"))
	})
}

func TestPickLogsPorts_SkipsTheWebPortAndEachOther(t *testing.T) {
	ports := []int{8080, 15080, 15080, 15081}
	h := &Host{LocalPort: func() (int, error) {
		p := ports[0]
		ports = ports[1:]
		return p, nil
	}}
	env := map[string]string{}
	require.NoError(t, h.pickLogsPorts(env, 8080))
	assert.Equal(t, map[string]string{"NEXUL_LOGS_PORT": "15080", "NEXUL_LOGS_GRPC_PORT": "15081"}, env)

	h.LocalPort = func() (int, error) { return 0, errors.New("no ports") }
	require.ErrorContains(t, h.pickLogsPorts(map[string]string{}, 80), "no ports")
}

func TestWaitHealthy_ServerNeverAnswers_TimesOutWithWhereToLook(t *testing.T) {
	th := newTestHost(t)
	th.HealthTimeout = 20 * time.Millisecond
	err := th.waitHealthy(t.Context(), 1, th.newUnit(kindServer, "", "v0.2.1"))
	require.ErrorContains(t, err, "did not answer on port 1; see `journalctl -u nexul-server`")
}

func TestLogHint(t *testing.T) {
	th := newTestHost(t)
	u := th.newUnit(kindServer, "", "v1")
	th.GOOS = "darwin"
	assert.Contains(t, th.logHint(u), filepath.Join(th.Paths.Logs, "nexul-server.log"))
	th.GOOS = "windows"
	assert.Contains(t, th.logHint(u), "service.log")
}

func TestReadEnvFile_SkipsCommentsAndBlanks(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte("# comment\n\nA=1\n B = two \nnot a pair\n"), 0o600))
	env, err := readEnvFile(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"A": "1", "B": "two"}, env)
}

func TestLoadInstall_ConfigWithoutDir_IsAnError(t *testing.T) {
	th := newTestHost(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(th.Paths.Config), 0o755))
	require.NoError(t, os.WriteFile(th.Paths.Config, []byte("OTHER=1\n"), 0o600))
	_, err := th.loadInstall()
	require.ErrorContains(t, err, "NEXUL_DIR")
}

func TestRandomPassword_MeetsTheLogsPasswordRule(t *testing.T) {
	for range 50 {
		p := randomPassword()
		require.Len(t, p, 25)
		assert.True(t, validLogsPassword(p), p)
		assert.NotContains(t, p, "$")
	}
}

func TestValidLogsPassword(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{"DevLogs-rotate-me-1", true},
		{"PSZVCNBRMQNJxhk6j4uzh4zk", false},
		{"Aa1-", false},
		{"alllower-1", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, validLogsPassword(tt.password), tt.password)
	}
}

func TestWriteSettings_ReplacesAPasswordOpenObserveWouldRefuse(t *testing.T) {
	th := newTestHost(t)
	dir := filepath.Join(th.root, "n")
	prev := &installed{Dir: dir, Env: map[string]string{"NEXUL_LOGS_PASSWORD": "PSZVCNBRMQNJxhk6j4uzh4zk", "NEXUL_LOGS_TOKEN": "keep-me"}}
	env, err := th.writeSettings(t.Context(), Options{Dir: dir, Port: 80}, prev, "v0.2.1")
	require.NoError(t, err)
	assert.True(t, validLogsPassword(env["NEXUL_LOGS_PASSWORD"]))
	assert.Equal(t, "keep-me", env["NEXUL_LOGS_TOKEN"])
}

func TestRandomSecret_MixesCaseAndDigits(t *testing.T) {
	for range 50 {
		s := randomSecret()
		require.Len(t, s, 24)
		assert.True(t, strings.ContainsAny(s, "234567"))
		assert.NotEqual(t, strings.ToLower(s), s)
		assert.NotEqual(t, strings.ToUpper(s), s)
	}
}

func TestSiteURL(t *testing.T) {
	assert.Equal(t, "http://203.0.113.4/", siteURL("203.0.113.4", 80))
	assert.Equal(t, "http://203.0.113.4:8080/", siteURL("203.0.113.4", 8080))
}

func TestChildEnv_DropsNexulVariables(t *testing.T) {
	got := childEnv([]string{"PATH=/usr/bin", "NEXUL_VERSION=v0.2.1", "HOME=/root", "NEXUL_PORT=81"})
	assert.Equal(t, []string{"PATH=/usr/bin", "HOME=/root"}, got)
}

func TestExecCommander_FailureCarriesTheOutput(t *testing.T) {
	out, err := execCommander{}.Run(t.Context(), "sh", "-c", "echo boom; exit 3")
	require.ErrorContains(t, err, "boom")
	assert.Equal(t, "boom", out)
}

func TestLocalPort_ReturnsAPortThatCanBeBound(t *testing.T) {
	port, err := localPort()
	require.NoError(t, err)
	assert.Positive(t, port)
}
