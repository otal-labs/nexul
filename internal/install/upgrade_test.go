package install

import (
	"bufio"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installed runs a fresh install into <root>/nexul for the tests that start from an installed host.
func (th *testHost) installed(t *testing.T) string {
	t.Helper()
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "nexul")
	require.NoError(t, th.Install(t.Context(), Options{Dir: dir, Port: th.webPort(t), LogsPort: 5080, Yes: true}))
	th.out.Reset()
	th.exec.calls = nil
	return dir
}

func TestUpgrade_NotInstalled_Fails(t *testing.T) {
	th := newTestHost(t)
	require.ErrorContains(t, th.Upgrade(t.Context(), "v0.2.2"), "nexul install")
}

func TestUpgrade_OtherVersion_SwapsTheBinaryAndReruns(t *testing.T) {
	th := newTestHost(t)
	th.installed(t)

	require.NoError(t, th.Upgrade(t.Context(), "0.2.2"))

	self, err := th.Executable()
	require.NoError(t, err)
	got, err := os.ReadFile(self)
	require.NoError(t, err)
	assert.Equal(t, "nexul-binary", string(got))
	assert.Equal(t, []string{self, "upgrade", "--version", "v0.2.2"}, th.reexec)
	assert.False(t, th.exec.ran("docker compose"), "the stack is upgraded by the new binary, not this one")
}

func TestUpgrade_ThisVersion_WritesTheStackAndRestartsIt(t *testing.T) {
	th := newTestHost(t)
	dir := th.installed(t)
	env, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	env["NEXUL_VERSION"] = "0.2.0"
	require.NoError(t, writeEnvFile(filepath.Join(dir, ".env"), env))

	require.NoError(t, th.Upgrade(t.Context(), "v0.2.1"))

	after, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "0.2.1", after["NEXUL_VERSION"])
	assert.Equal(t, env["NEXUL_LOGS_PASSWORD"], after["NEXUL_LOGS_PASSWORD"])
	assert.True(t, th.exec.ran("docker compose --project-directory "+dir+" pull"))
	assert.True(t, th.exec.ran("docker compose --project-directory "+dir+" up --detach"))
	assert.Contains(t, th.out.String(), "Nexul v0.2.1 is running.")
	assert.Nil(t, th.reexec)
}

func TestUninstall(t *testing.T) {
	t.Run("without a terminal and without --yes nothing is removed", func(t *testing.T) {
		th := newTestHost(t)
		th.installed(t)
		require.ErrorContains(t, th.Uninstall(t.Context(), false, false), "--yes")
		assert.Empty(t, th.exec.calls)
	})

	t.Run("answering no cancels", func(t *testing.T) {
		th := newTestHost(t)
		th.installed(t)
		th.Interactive = true
		th.In = bufio.NewReader(strings.NewReader("n\n"))
		require.ErrorContains(t, th.Uninstall(t.Context(), false, false), "cancelled")
		assert.Empty(t, th.exec.calls)
	})

	t.Run("keeps the install directory so install brings it back", func(t *testing.T) {
		th := newTestHost(t)
		dir := th.installed(t)
		th.Interactive = true
		th.In = bufio.NewReader(strings.NewReader("y\n"))

		require.NoError(t, th.Uninstall(t.Context(), false, false))

		assert.True(t, th.exec.ran("docker compose --project-directory "+dir+" down"))
		assert.True(t, th.exec.ran("systemctl disable --now nexul-runner.service"))
		assert.NoFileExists(t, th.Paths.Unit)
		assert.NoFileExists(t, filepath.Join(th.Paths.BinDir, "nexul-runner"))
		assert.NoFileExists(t, filepath.Join(th.Paths.BinDir, "nexul"))
		assert.NoFileExists(t, th.Paths.Config)
		assert.FileExists(t, filepath.Join(dir, ".env"))
		assert.Contains(t, th.out.String(), "nexul install --dir "+dir)
	})

	t.Run("purge deletes the install directory", func(t *testing.T) {
		th := newTestHost(t)
		dir := th.installed(t)
		require.NoError(t, th.Uninstall(t.Context(), true, true))
		assert.NoDirExists(t, dir)
		assert.Contains(t, th.out.String(), "DELETES "+dir)
	})

	t.Run("a failing compose down stops before removing anything", func(t *testing.T) {
		th := newTestHost(t)
		th.installed(t)
		th.set("docker compose", "", errors.New("daemon down"))
		require.ErrorContains(t, th.Uninstall(t.Context(), false, true), "daemon down")
		assert.FileExists(t, th.Paths.Unit)
	})
}

func TestStatus(t *testing.T) {
	t.Run("not installed", func(t *testing.T) {
		th := newTestHost(t)
		require.ErrorContains(t, th.Status(t.Context()), "not installed")
	})
	t.Run("prints the version, directory, runner and services", func(t *testing.T) {
		th := newTestHost(t)
		dir := th.installed(t)
		th.set("systemctl is-active", "active", nil)
		th.set("docker compose --project-directory "+dir+" ps", "server\trunning\tUp 2 minutes", nil)

		require.NoError(t, th.Status(t.Context()))

		out := th.out.String()
		assert.Contains(t, out, "Nexul v0.2.1")
		assert.Contains(t, out, "Directory  "+dir)
		assert.Contains(t, out, "Runner     active")
		assert.Contains(t, out, "server\trunning\tUp 2 minutes")
	})
	t.Run("a docker failure still prints the rest", func(t *testing.T) {
		th := newTestHost(t)
		th.installed(t)
		th.set("docker compose", "", errors.New("permission denied"))
		require.NoError(t, th.Status(t.Context()))
		assert.Contains(t, th.out.String(), "unavailable: permission denied")
	})
}

func TestCommandFlags(t *testing.T) {
	t.Run("install rejects an unknown flag", func(t *testing.T) {
		th := newTestHost(t)
		require.Error(t, th.runInstall(t.Context(), []string{"--bogus"}))
	})
	t.Run("upgrade passes --version through", func(t *testing.T) {
		th := newTestHost(t)
		require.ErrorContains(t, th.runUpgrade(t.Context(), []string{"--version", "0.2.2"}), "not installed")
	})
	t.Run("uninstall passes --purge and --yes through", func(t *testing.T) {
		th := newTestHost(t)
		dir := th.installed(t)
		require.NoError(t, th.runUninstall(t.Context(), []string{"--purge", "--yes"}))
		assert.NoDirExists(t, dir)
	})
}

func TestResolveTarget(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/otal-labs/nexul/releases/latest":
			_, _ = w.Write([]byte(`{"tag_name": "v0.2.3"}`))
		case "/repos/otal-labs/nexul/releases":
			_, _ = w.Write([]byte(`[{"tag_name": "v0.2.4-beta.2", "prerelease": true}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)

	tests := []struct {
		name    string
		version string
		flag    string
		want    string
	}{
		{"the flag wins", "v0.2.1", "0.1.0", "v0.1.0"},
		{"a stable install follows stable", "v0.2.1", "", "v0.2.3"},
		{"a beta install follows beta", "v0.2.1-beta.1", "", "v0.2.4-beta.2"},
		{"a dev build follows stable", "dev", "", "v0.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.ReleaseAPI = api.URL
			withVersion(t, tt.version)
			got, err := th.resolveTarget(t.Context(), tt.flag)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("an unreachable API is an error naming the channel", func(t *testing.T) {
		th := newTestHost(t)
		th.ReleaseAPI = api.URL + "/nowhere"
		withVersion(t, "v0.2.1")
		_, err := th.resolveTarget(t.Context(), "")
		require.ErrorContains(t, err, "newest stable release")
	})
}
