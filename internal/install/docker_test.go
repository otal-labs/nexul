package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDocker(t *testing.T) {
	t.Run("a running docker reports its version", func(t *testing.T) {
		th := newTestHost(t)
		v, err := th.ensureDocker(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "29.1.3", v)
		assert.False(t, th.exec.ran("systemctl"))
	})

	t.Run("a missing docker is installed with Docker's script", func(t *testing.T) {
		th := newTestHost(t)
		th.missing("docker")
		script := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("echo installing docker"))
		}))
		t.Cleanup(script.Close)
		th.DockerScriptURL = script.URL

		v, err := th.ensureDocker(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "installed 29.1.3", v)
		assert.True(t, th.exec.ran("sh "+filepath.Join(os.TempDir(), "nexul-get-docker.sh")))
	})

	t.Run("a failing install script is reported", func(t *testing.T) {
		th := newTestHost(t)
		th.missing("docker")
		script := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
		t.Cleanup(script.Close)
		th.DockerScriptURL = script.URL
		th.set("sh ", "", errors.New("unsupported distribution"))

		_, err := th.ensureDocker(t.Context())
		require.ErrorContains(t, err, "unsupported distribution")
	})

	t.Run("docker from snap is refused", func(t *testing.T) {
		th := newTestHost(t)
		th.LookPath = func(string) (string, error) { return "/snap/bin/docker", nil }
		_, err := th.ensureDocker(t.Context())
		require.ErrorContains(t, err, "snap remove docker")
	})

	t.Run("a stopped daemon is started", func(t *testing.T) {
		th := newTestHost(t)
		th.set("docker info", "", errors.New("cannot connect to the docker daemon"))
		_, err := th.ensureDocker(t.Context())
		require.NoError(t, err)
		assert.True(t, th.exec.ran("systemctl enable --now docker"))
	})

	t.Run("a daemon that will not start is reported", func(t *testing.T) {
		th := newTestHost(t)
		th.set("docker info", "", errors.New("cannot connect"))
		th.set("systemctl enable --now docker", "", errors.New("unit not found"))
		_, err := th.ensureDocker(t.Context())
		require.ErrorContains(t, err, "daemon is not running")
	})
}

func TestEnsureCompose(t *testing.T) {
	t.Run("an installed plugin reports its version", func(t *testing.T) {
		th := newTestHost(t)
		v, err := th.ensureCompose(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "2.40.0", v)
	})

	t.Run("Ubuntu's docker.io gets docker-compose-v2 after Docker's package is not found", func(t *testing.T) {
		th := newTestHost(t)
		installed := false
		th.set("docker compose version", "", errors.New("'compose' is not a docker command"))
		th.set("apt-get install -y -qq docker-compose-plugin", "", errors.New("Unable to locate package"))
		th.exec.onRun = func(line string) {
			if line == "apt-get install -y -qq docker-compose-v2" {
				installed = true
				th.set("docker compose version", "2.37.1", nil)
			}
		}

		v, err := th.ensureCompose(t.Context())

		require.NoError(t, err)
		assert.True(t, installed)
		assert.Equal(t, "installed 2.37.1 (docker-compose-v2)", v)
		assert.Equal(t, 1, strings.Count(strings.Join(th.exec.calls, "\n"), "apt-get update"), "apt-get update runs once")
	})

	t.Run("no package works, so the plugin is downloaded and verified", func(t *testing.T) {
		th := newTestHost(t)
		th.missing("apt-get", "dnf", "yum", "zypper", "pacman", "apk")
		th.set("docker compose version", "", errors.New("missing"))
		plugin := []byte("compose-plugin")
		sum := sha256.Sum256(plugin)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, ".sha256") {
				_, _ = fmt.Fprintf(w, "%s *docker-compose-linux-x86_64\n", hex.EncodeToString(sum[:]))
				return
			}
			_, _ = w.Write(plugin)
		}))
		t.Cleanup(srv.Close)
		th.ComposeURL = srv.URL
		th.exec.onRun = func(line string) {
			if _, err := os.Stat(filepath.Join(th.Paths.ComposePlugs, "docker-compose")); err == nil {
				th.set("docker compose version", "2.40.0", nil)
			}
		}

		v, err := th.ensureCompose(t.Context())

		require.NoError(t, err)
		assert.Equal(t, "installed 2.40.0", v)
		got, err := os.ReadFile(filepath.Join(th.Paths.ComposePlugs, "docker-compose"))
		require.NoError(t, err)
		assert.Equal(t, plugin, got)
	})

	t.Run("a CPU without a compose build is reported", func(t *testing.T) {
		th := newTestHost(t)
		th.GOARCH = "riscv64"
		th.missing("apt-get", "dnf", "yum", "zypper", "pacman", "apk")
		th.set("docker compose version", "", errors.New("missing"))
		_, err := th.ensureCompose(t.Context())
		require.ErrorContains(t, err, "riscv64")
	})
}

func TestEnsurePackage(t *testing.T) {
	t.Run("a present command is left alone", func(t *testing.T) {
		th := newTestHost(t)
		v, err := th.ensurePackage(t.Context(), "git")
		require.NoError(t, err)
		assert.Equal(t, "present", v)
		assert.Empty(t, th.exec.calls)
	})
	t.Run("a missing command is installed with the host's package manager", func(t *testing.T) {
		th := newTestHost(t)
		th.missing("git", "apt-get")
		v, err := th.ensurePackage(t.Context(), "git")
		require.NoError(t, err)
		assert.Equal(t, "installed", v)
		assert.True(t, th.exec.ran("dnf install -y -q git"))
	})
	t.Run("no package manager is an error naming the command", func(t *testing.T) {
		th := newTestHost(t)
		th.missing("git", "apt-get", "dnf", "yum", "zypper", "pacman", "apk")
		_, err := th.ensurePackage(t.Context(), "git")
		require.ErrorContains(t, err, "git is missing")
	})
	t.Run("a failing apt-get update stops the install", func(t *testing.T) {
		th := newTestHost(t)
		th.missing("git")
		th.set("apt-get update", "", errors.New("temporary failure resolving"))
		_, err := th.ensurePackage(t.Context(), "git")
		require.ErrorContains(t, err, "temporary failure")
	})
}

func TestReleaseAsset(t *testing.T) {
	t.Run("a checksum mismatch keeps the old file", func(t *testing.T) {
		th := newTestHost(t)
		th.release.checksum = map[string]string{"nexul-runner-linux-amd64": strings.Repeat("0", 64)}
		dest := filepath.Join(th.root, "bin", "nexul-runner")
		require.NoError(t, os.WriteFile(dest, []byte("old"), 0o755))

		err := th.releaseAsset(t.Context(), "v0.2.1", "nexul-runner-linux-amd64", dest)

		require.ErrorContains(t, err, "checksum mismatch")
		got, readErr := os.ReadFile(dest)
		require.NoError(t, readErr)
		assert.Equal(t, "old", string(got))
		assert.NoFileExists(t, dest+".download")
	})
	t.Run("a file the checksums do not list is refused", func(t *testing.T) {
		th := newTestHost(t)
		err := th.releaseAsset(t.Context(), "v0.2.1", "nexul-runner-linux-arm64", filepath.Join(th.root, "x"))
		require.ErrorContains(t, err, "lists no checksum")
	})
	t.Run("a missing release is an HTTP error", func(t *testing.T) {
		th := newTestHost(t)
		th.ReleaseURL += "/missing"
		srv := httptest.NewServer(http.NotFoundHandler())
		t.Cleanup(srv.Close)
		th.ReleaseURL = srv.URL
		err := th.releaseAsset(t.Context(), "v9.9.9", "nexul-linux-amd64", filepath.Join(th.root, "x"))
		require.ErrorContains(t, err, "404")
	})
}

func TestTail(t *testing.T) {
	assert.Equal(t, "a\nb", tail("a\nb", 5))
	assert.Equal(t, "c\nd", tail("a\nb\nc\nd", 2))
}
