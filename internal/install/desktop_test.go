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

func TestInstall_Mac_LayersTheDesktopStack(t *testing.T) {
	th := newTestHost(t)
	onMac(t, th)
	th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
	withVersion(t, "v0.2.1")

	require.NoError(t, th.Install(t.Context(), Options{Port: th.webPort(t), LogsPort: 5080, Yes: true}))

	dir := filepath.Join(th.Home, "nexul")
	assert.FileExists(t, filepath.Join(dir, "docker-compose.yml"))
	assert.FileExists(t, filepath.Join(dir, "docker-compose.desktop.yml"))
	assert.NoDirExists(t, filepath.Join(dir, "data"), "a desktop install keeps its data in Docker volumes")
	env, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "docker-compose.yml:docker-compose.desktop.yml", env["COMPOSE_FILE"])
	assert.Equal(t, dir, env["NEXUL_STACK_ROOT"])
	assert.False(t, th.exec.ran("systemctl"), "there is no systemd on a Mac")
	assert.NoFileExists(t, th.Paths.Unit)
	out := th.out.String()
	assert.Contains(t, out, "container, part of the stack")
	assert.Contains(t, out, "http://localhost:")
	assert.Contains(t, out, "Docker volumes nexul_nexul-data")
}

func TestInstall_Windows_UsesItsPathsAndSeparator(t *testing.T) {
	th := newTestHost(t)
	th.GOOS = "windows"
	th.Getuid = func() int { return -1 }
	withVersion(t, "v0.2.1")

	require.NoError(t, th.Install(t.Context(), Options{Port: th.webPort(t), LogsPort: 5080, Yes: true}))

	env, err := readEnvFile(filepath.Join(th.Home, "nexul", ".env"))
	require.NoError(t, err)
	assert.Equal(t, "docker-compose.yml;docker-compose.desktop.yml", env["COMPOSE_FILE"])
	assert.Equal(t, DefaultDir, env["NEXUL_STACK_ROOT"])
	assert.FileExists(t, filepath.Join(th.Paths.BinDir, "nexul.exe"))
}

func TestUninstall_Desktop_Purge_RemovesTheVolumes(t *testing.T) {
	th := newTestHost(t)
	onMac(t, th)
	th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
	withVersion(t, "v0.2.1")
	require.NoError(t, th.Install(t.Context(), Options{Port: th.webPort(t), LogsPort: 5080, Yes: true}))
	th.exec.calls = nil

	require.NoError(t, th.Uninstall(t.Context(), true, true))

	assert.True(t, th.exec.ran("docker compose --project-directory "+filepath.Join(th.Home, "nexul")+" down --remove-orphans --volumes"))
	assert.False(t, th.exec.ran("systemctl"))
	assert.Contains(t, th.out.String(), "removed with the stack")
}

func TestStatus_Desktop_PointsAtTheRunnerContainer(t *testing.T) {
	th := newTestHost(t)
	onMac(t, th)
	th.lookPath(map[string]string{"docker": "/usr/local/bin/docker"})
	withVersion(t, "v0.2.1")
	require.NoError(t, th.Install(t.Context(), Options{Port: th.webPort(t), LogsPort: 5080, Yes: true}))
	th.out.Reset()

	require.NoError(t, th.Status(t.Context()))

	assert.Contains(t, th.out.String(), "Runner     container (see Services)")
	assert.False(t, th.exec.ran("systemctl"))
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
	assert.Equal(t, "/Users/onik/Library/Application Support/nexul/nexul.conf", pathsFor("darwin", "/Users/onik").Config)
	assert.Equal(t, "/etc/nexul/nexul.conf", pathsFor("linux", "/root").Config)
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
