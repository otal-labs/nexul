package install

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// onMac makes the test host a Mac with no Docker and Homebrew at /opt/homebrew, unless the test changes that.
func onMac(t *testing.T, th *testHost) {
	t.Helper()
	th.GOOS, th.GOARCH = "darwin", "arm64"
	th.Getuid = func() int { return 501 }
	th.lookPath(map[string]string{"brew": "/opt/homebrew/bin/brew", "open": "/usr/bin/open"})
	th.set("/opt/homebrew/bin/brew --prefix docker-compose", "/opt/homebrew/opt/docker-compose", nil)
}

// lookPath makes exactly the given commands resolvable; every other lookup fails.
func (th *testHost) lookPath(found map[string]string) {
	th.LookPath = func(file string) (string, error) {
		if path, ok := found[file]; ok {
			return path, nil
		}
		return "", errors.New("not found")
	}
}

func TestEnsureDockerMac(t *testing.T) {
	t.Run("a running engine is used as it is", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
		v, err := th.ensureDockerMac(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "29.1.3", v)
		assert.False(t, th.exec.ran("/opt/homebrew/bin/brew"))
	})

	t.Run("no docker installs Colima with Homebrew and starts it at login", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)

		v, err := th.ensureDockerMac(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "installed Colima, Docker 29.1.3", v)
		assert.True(t, th.exec.ran("/opt/homebrew/bin/brew install colima docker docker-compose"))
		assert.True(t, th.exec.ran("colima start --cpu 2 --memory 4 --disk 30"))
		assert.True(t, th.exec.ran("colima stop"))
		assert.True(t, th.exec.ran("/opt/homebrew/bin/brew services start colima"))
		target, err := os.Readlink(filepath.Join(th.Paths.UserPlugins, "docker-compose"))
		require.NoError(t, err)
		assert.Equal(t, "/opt/homebrew/opt/docker-compose/bin/docker-compose", target)
	})

	t.Run("no Homebrew and no terminal says where to get Homebrew", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.lookPath(map[string]string{})
		_, err := th.ensureDockerMac(t.Context())
		require.ErrorContains(t, err, "brew.sh")
		assert.Empty(t, th.exec.calls)
	})

	t.Run("no Homebrew with a terminal runs Homebrew's installer attached", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.lookPath(map[string]string{})
		th.Interactive = true
		script := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("echo installing homebrew"))
		}))
		t.Cleanup(script.Close)
		th.BrewInstallURL = script.URL
		th.exec.onRun = func(line string) {
			if strings.HasPrefix(line, "/bin/bash") {
				th.lookPath(map[string]string{"/opt/homebrew/bin/brew": "/opt/homebrew/bin/brew"})
			}
		}

		_, err := th.ensureDockerMac(t.Context())

		require.NoError(t, err)
		assert.True(t, th.exec.ran("/bin/bash "+filepath.Join(os.TempDir(), "nexul-brew-install.sh")))
		assert.True(t, th.exec.ran("/opt/homebrew/bin/brew install colima"))
		assert.Contains(t, th.out.String(), "asks for your password")
	})

	t.Run("a stopped Colima is started through brew services", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.lookPath(map[string]string{"docker": "/opt/homebrew/bin/docker", "colima": "/opt/homebrew/bin/colima", "brew": "/opt/homebrew/bin/brew"})
		started := false
		th.set("docker info", "", errors.New("cannot connect"))
		th.exec.onRun = func(line string) {
			if line == "/opt/homebrew/bin/brew services start colima" {
				started = true
				th.set("docker info", "", nil)
			}
		}

		v, err := th.ensureDockerMac(t.Context())

		require.NoError(t, err)
		assert.True(t, started)
		assert.Equal(t, "started Colima, Docker 29.1.3", v)
		assert.False(t, th.exec.ran("/opt/homebrew/bin/brew install"))
	})

	t.Run("a stopped Docker Desktop is opened", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
		require.NoError(t, os.MkdirAll(th.Paths.DockerApp, 0o755))
		th.set("docker info", "", errors.New("cannot connect"))
		th.exec.onRun = func(line string) {
			if strings.HasPrefix(line, "open -a") {
				th.set("docker info", "", nil)
			}
		}

		v, err := th.ensureDockerMac(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "started Docker Desktop 29.1.3", v)
	})

	t.Run("a stopped engine that is neither is left to the user", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
		th.set("docker info", "", errors.New("cannot connect"))
		_, err := th.ensureDockerMac(t.Context())
		require.ErrorContains(t, err, "start your Docker engine")
	})

	t.Run("an engine that never comes up times out", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.HealthTimeout = 20 * time.Millisecond
		th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
		require.NoError(t, os.MkdirAll(th.Paths.DockerApp, 0o755))
		th.set("docker info", "", errors.New("cannot connect"))
		_, err := th.ensureDockerMac(t.Context())
		require.ErrorContains(t, err, "did not start")
	})
}

func TestEnsureComposeMac(t *testing.T) {
	t.Run("an installed plugin reports its version", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		v, err := th.ensureComposeMac(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "2.40.0", v)
	})
	t.Run("a missing plugin comes from Homebrew and is linked", func(t *testing.T) {
		th := newTestHost(t)
		onMac(t, th)
		th.set("docker compose version", "", errors.New("'compose' is not a docker command"))
		th.exec.onRun = func(line string) {
			if line == "/opt/homebrew/bin/brew install docker-compose" {
				th.set("docker compose version", "2.40.0", nil)
			}
		}
		v, err := th.ensureComposeMac(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "installed 2.40.0", v)
		assert.FileExists(t, filepath.Join(th.Paths.UserPlugins, "docker-compose"), "the link exists even though its target is fake")
	})
}

func TestEnsureDockerWindows(t *testing.T) {
	t.Run("no Docker Desktop explains where to get it", func(t *testing.T) {
		th := newTestHost(t)
		th.GOOS = "windows"
		th.lookPath(map[string]string{})
		_, err := th.ensureDockerWindows(t.Context())
		require.ErrorContains(t, err, "winget install -e --id Docker.DockerDesktop")
		assert.Empty(t, th.exec.calls, "nothing is installed on Windows")
	})
	t.Run("Docker Desktop not running says to start it", func(t *testing.T) {
		th := newTestHost(t)
		th.GOOS = "windows"
		th.set("docker info", "", errors.New("pipe not found"))
		_, err := th.ensureDockerWindows(t.Context())
		require.ErrorContains(t, err, "start Docker Desktop")
	})
	t.Run("a running Docker Desktop reports its version", func(t *testing.T) {
		th := newTestHost(t)
		th.GOOS = "windows"
		v, err := th.ensureDockerWindows(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "29.1.3", v)
	})
	t.Run("a missing compose plugin says to update Docker Desktop", func(t *testing.T) {
		th := newTestHost(t)
		th.GOOS = "windows"
		th.set("docker compose version", "", errors.New("not a docker command"))
		_, err := th.ensureComposeWindows(t.Context())
		require.ErrorContains(t, err, "update Docker Desktop")
	})
}

func TestInstall_Mac_RunsEveryUnitAsALaunchAgentOfTheUser(t *testing.T) {
	th := newTestHost(t)
	onMac(t, th)
	onMacServices(th)
	th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.Home, "nexul")
	th.bootedServer(t, dir)

	require.NoError(t, th.Install(t.Context(), Options{Port: th.webPort(t), Yes: true}))

	for _, name := range []string{"nexul-server", "nexul-openobserve", "nexul-runner-instance", "nexul-automations-instance"} {
		assert.FileExists(t, filepath.Join(th.Paths.Services, "io.nexul."+name+".plist"))
		assert.True(t, th.exec.ran("launchctl bootstrap gui/501 "+filepath.Join(th.Paths.Services, "io.nexul."+name+".plist")), name)
	}
	assert.Equal(t, "nexul-server-binary", readFile(t, filepath.Join(th.Paths.UnitRoot, "server", "nexul-server")))
	assert.DirExists(t, filepath.Join(dir, "data"), "a native server keeps its data on the host")
	assert.False(t, th.exec.ran("systemctl"), "there is no systemd on a Mac")
	assert.False(t, th.exec.ran("useradd"), "units run as the installing user")
	assert.False(t, th.exec.ran("chown"))
	out := th.out.String()
	assert.Contains(t, out, "http://localhost:")
	assert.Contains(t, out, "runs on this computer")
	assert.Contains(t, out, "Setup code     nxs_setup")
	serverEnv, err := readEnvFile(filepath.Join(th.Paths.UnitRoot, "server", "env"))
	require.NoError(t, err)
	assert.Equal(t, "1", serverEnv["NEXUL_LOCAL"])
}

func TestInstall_Windows_RegistersServicesAndCopiesTheCommand(t *testing.T) {
	th := newTestHost(t)
	th.GOOS = "windows"
	th.Getuid = func() int { return -1 }
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.Home, "nexul")
	th.bootedServer(t, dir)

	require.NoError(t, th.Install(t.Context(), Options{Port: th.webPort(t), Yes: true}))

	ctl := filepath.Join(th.Paths.BinDir, "nexul.exe")
	assert.Equal(t, "this-binary", readFile(t, ctl))
	assert.True(t, th.exec.ran(`sc.exe create nexul-server binPath= "`+ctl+`" service-host nexul-server`))
	assert.True(t, th.exec.ran("sc.exe start nexul-runner-instance"))
	assert.Equal(t, "nexul-runner-binary", readFile(t, filepath.Join(th.Paths.UnitRoot, "runner-instance", "nexul-runner.exe")))
	assert.Equal(t, "openobserve-v1.0.4", readFile(t, filepath.Join(th.Paths.UnitRoot, "openobserve", "openobserve.exe")))
	u, err := th.loadUnit("nexul-runner-instance")
	require.NoError(t, err)
	assert.Equal(t, ctl, u.Env["NEXUL_CTL"])
}

func TestUninstall_Windows_LeavesTheRunningCommandAndSaysSo(t *testing.T) {
	th := newTestHost(t)
	th.GOOS = "windows"
	th.Getuid = func() int { return -1 }
	withVersion(t, "v0.2.1")
	th.bootedServer(t, filepath.Join(th.Home, "nexul"))
	require.NoError(t, th.Install(t.Context(), Options{Port: th.webPort(t), Yes: true}))
	th.set("sc.exe query", "        STATE              : 1  STOPPED\n", nil)

	require.NoError(t, th.Uninstall(t.Context(), false, true))

	assert.True(t, th.exec.ran("sc.exe delete nexul-server"))
	assert.FileExists(t, filepath.Join(th.Paths.BinDir, "nexul.exe"))
	assert.Contains(t, th.out.String(), "yourself")
}

func TestPlatformNames(t *testing.T) {
	win := &Host{GOOS: "windows", GOARCH: "amd64", Home: `C:\Users\onik`}
	assert.Equal(t, "nexul-windows-amd64.exe", win.assetName("nexul"))
	assert.Equal(t, "nexul.exe", win.commandName())
	assert.Equal(t, filepath.Join(`C:\Users\onik`, "nexul"), win.defaultDir())

	mac := &Host{GOOS: "darwin", GOARCH: "arm64", Home: "/Users/onik"}
	assert.Equal(t, "nexul-darwin-arm64", mac.assetName("nexul"))
	assert.Equal(t, "/Users/onik/nexul", mac.defaultDir())
	assert.Equal(t, "localhost", mac.PublicIP())

	linux := &Host{GOOS: "linux"}
	assert.Equal(t, DefaultDir, linux.defaultDir())
	noEnv := func(string) string { return "" }
	macPaths := pathsFor("darwin", "/Users/onik", noEnv)
	assert.Equal(t, "/Users/onik/Library/Application Support/nexul/nexul.conf", macPaths.Config)
	assert.Equal(t, "/Users/onik/Library/LaunchAgents", macPaths.Services)
	linuxPaths := pathsFor("linux", "/root", noEnv)
	assert.Equal(t, "/etc/nexul/nexul.conf", linuxPaths.Config)
	assert.Equal(t, "/opt/nexul", linuxPaths.UnitRoot)
	winPaths := pathsFor("windows", `C:\Users\onik`, func(k string) string {
		return map[string]string{"ProgramData": `D:\Data`}[k]
	})
	assert.Equal(t, filepath.Join(`D:\Data`, "Nexul"), winPaths.UnitRoot)
	assert.Equal(t, filepath.Join(`C:\Program Files`, "Nexul"), winPaths.BinDir)
}

func TestReplaceFile_MovesOverTheOldFile(t *testing.T) {
	dir := t.TempDir()
	src, dest := filepath.Join(dir, "new"), filepath.Join(dir, "nexul")
	require.NoError(t, os.WriteFile(src, []byte("new"), 0o755))
	require.NoError(t, os.WriteFile(dest, []byte("old"), 0o755))
	require.NoError(t, replaceFile(src, dest))
	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))
	assert.NoFileExists(t, src)
}
