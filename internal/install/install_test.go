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

	"github.com/otal-labs/nexul"
)

func TestRun_UnknownCommand_ReturnsErrUnknownCommand(t *testing.T) {
	err := Run(t.Context(), "bogus", nil)
	require.ErrorIs(t, err, ErrUnknownCommand)
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
		{"another OS", "freebsd", 0, "nexul serve"},
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

func TestWithDefaults(t *testing.T) {
	t.Run("a fresh host takes the default ports", func(t *testing.T) {
		assert.Equal(t, Options{Port: 80, LogsPort: 5080}, withDefaults(Options{}, nil))
	})
	t.Run("an existing install keeps its ports", func(t *testing.T) {
		prev := &installed{Env: map[string]string{"NEXUL_PORT": "8080", "NEXUL_LOGS_PORT": "5081"}}
		assert.Equal(t, Options{Port: 8080, LogsPort: 5081}, withDefaults(Options{}, prev))
	})
	t.Run("flags win over the existing install", func(t *testing.T) {
		prev := &installed{Env: map[string]string{"NEXUL_PORT": "8080"}}
		assert.Equal(t, Options{Port: 9000, LogsPort: 5080}, withDefaults(Options{Port: 9000}, prev))
	})
}

func TestCheckPorts(t *testing.T) {
	tests := []struct {
		name    string
		o       Options
		prev    *installed
		wantErr string
	}{
		{"both free", Options{Port: 81, LogsPort: 5080}, nil, ""},
		{"same port twice", Options{Port: 5080, LogsPort: 5080}, nil, "both 5080"},
		{"out of range", Options{Port: 70000, LogsPort: 5080}, nil, "out of range"},
		{"held by another program", Options{Port: 80, LogsPort: 5080}, nil, "--port"},
		{"held by this install", Options{Port: 80, LogsPort: 5080}, &installed{Env: map[string]string{"NEXUL_PORT": "80"}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Host{PortFree: func(port int) bool { return port != 80 }}
			err := h.checkPorts(tt.o, tt.prev)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAskPorts(t *testing.T) {
	t.Run("Enter keeps both defaults", func(t *testing.T) {
		th := newTestHost(t)
		th.In = bufio.NewReader(strings.NewReader("\n\n"))
		o, err := th.askPorts(Options{Port: 80, LogsPort: 5080}, nil)
		require.NoError(t, err)
		assert.Equal(t, Options{Port: 80, LogsPort: 5080}, o)
	})
	t.Run("a bad or busy port is asked again", func(t *testing.T) {
		th := newTestHost(t)
		th.PortFree = func(port int) bool { return port != 80 }
		th.In = bufio.NewReader(strings.NewReader("abc\n80\n8080\n\n"))
		o, err := th.askPorts(Options{Port: 80, LogsPort: 5080}, nil)
		require.NoError(t, err)
		assert.Equal(t, Options{Port: 8080, LogsPort: 5080}, o)
		assert.Contains(t, th.out.String(), `"abc" is not a port number`)
		assert.Contains(t, th.out.String(), "Port 80 is already in use")
	})
	t.Run("the input ending early is an error, not a loop", func(t *testing.T) {
		th := newTestHost(t)
		th.In = bufio.NewReader(strings.NewReader(""))
		_, err := th.askPorts(Options{Port: 80, LogsPort: 5080}, nil)
		require.ErrorContains(t, err, "read answer")
	})
}

func TestInstall_FreshHost_RunsEveryStepAndPrintsTheSummary(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "data", "nexul")

	err := th.Install(t.Context(), Options{Dir: dir, Port: th.webPort(t), LogsPort: 5080, Yes: true})
	require.NoError(t, err)

	compose, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	require.NoError(t, err)
	assert.Equal(t, nexul.Compose, compose)
	env, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "0.2.1", env["NEXUL_VERSION"])
	assert.Len(t, env["NEXUL_LOGS_PASSWORD"], 25)
	assert.NotEmpty(t, env["NEXUL_LOGS_TOKEN"])
	info, err := os.Stat(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	conf, err := readEnvFile(th.Paths.Config)
	require.NoError(t, err)
	assert.Equal(t, dir, conf["NEXUL_DIR"])

	runner, err := os.ReadFile(filepath.Join(th.Paths.BinDir, "nexul-runner"))
	require.NoError(t, err)
	assert.Equal(t, "runner-binary", string(runner))
	self, err := os.ReadFile(filepath.Join(th.Paths.BinDir, "nexul"))
	require.NoError(t, err)
	assert.Equal(t, "this-binary", string(self))
	unit, err := os.ReadFile(th.Paths.Unit)
	require.NoError(t, err)
	assert.Contains(t, string(unit), "NEXUL_RUNNER_SECRET_FILE="+filepath.Join(dir, "data", "runner-secret"))

	assert.True(t, th.exec.ran("docker compose --project-directory "+dir+" pull"))
	assert.True(t, th.exec.ran("docker compose --project-directory "+dir+" up --detach"))
	assert.True(t, th.exec.ran("systemctl restart nexul-runner.service"))
	out := th.out.String()
	assert.Contains(t, out, "Nexul v0.2.1 is running.")
	assert.Contains(t, out, env["NEXUL_LOGS_PASSWORD"])
}

func TestInstall_Rerun_KeepsSecretsAndChoices(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "data", "nexul")
	port := th.webPort(t)
	require.NoError(t, th.Install(t.Context(), Options{Dir: dir, Port: port, LogsPort: 5080, Yes: true}))
	first, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)

	th.PortFree = func(int) bool { return false }
	require.NoError(t, th.Install(t.Context(), Options{Yes: true}))

	second, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestInstall_KeptDirectory_ReusesItsSecretsAndPorts(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "kept")
	port := th.webPort(t)
	require.NoError(t, th.Install(t.Context(), Options{Dir: dir, Port: port, LogsPort: 5081, Yes: true}))
	first, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	require.NoError(t, th.Uninstall(t.Context(), false, true))
	assert.NoDirExists(t, filepath.Dir(th.Paths.Config))

	th.PortFree = func(int) bool { return false }
	require.NoError(t, th.Install(t.Context(), Options{Dir: dir, Yes: true}))

	second, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestInstall_FailingStep_StopsTheRest(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	th.set("docker compose --project-directory", "", errors.New("pull access denied"))

	err := th.Install(t.Context(), Options{Dir: filepath.Join(th.root, "n"), Port: th.webPort(t), LogsPort: 5080, Yes: true})

	require.ErrorContains(t, err, "images: pull access denied")
	assert.False(t, th.exec.ran("systemctl"), "the runner must not be installed when the stack did not start")
	assert.Contains(t, th.out.String(), "failed")
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
	th.Interactive = true
	th.In = bufio.NewReader(strings.NewReader(dir + "\n" + strconv.Itoa(th.webPort(t)) + "\n\n"))

	require.NoError(t, th.Install(t.Context(), Options{}))

	assert.FileExists(t, filepath.Join(dir, "docker-compose.yml"))
	assert.Contains(t, th.out.String(), "Press Enter to accept the default.")
}

func TestInstallRunner_WithoutSystemd_Fails(t *testing.T) {
	th := newTestHost(t)
	th.Paths.SystemdProbe = filepath.Join(th.root, "absent")
	_, err := th.installRunner(t.Context(), Options{Dir: "/data/nexul", Port: 80}, "v0.2.1")
	require.ErrorContains(t, err, "systemd")
}

func TestWaitHealthy_ServerNeverAnswers_TimesOut(t *testing.T) {
	th := newTestHost(t)
	th.HealthTimeout = 20 * time.Millisecond
	err := th.waitHealthy(t.Context(), 1)
	require.ErrorContains(t, err, "did not answer on port 1")
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

func TestWriteStack_ReplacesAPasswordOpenObserveWouldRefuse(t *testing.T) {
	th := newTestHost(t)
	dir := filepath.Join(th.root, "n")
	prev := &installed{Dir: dir, Env: map[string]string{"NEXUL_LOGS_PASSWORD": "PSZVCNBRMQNJxhk6j4uzh4zk", "NEXUL_LOGS_TOKEN": "keep-me"}}
	env, err := th.writeStack(Options{Dir: dir, Port: 80, LogsPort: 5080}, prev, "v0.2.1")
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
