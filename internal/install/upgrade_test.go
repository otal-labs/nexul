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

// installed runs a fresh server install into <root>/data/nexul, then forgets its calls and output.
func (th *testHost) installed(t *testing.T) string {
	t.Helper()
	dir := th.serverInstall(t)
	th.out.Reset()
	th.exec.calls = nil
	return dir
}

func TestUpgrade_NothingInstalled_Fails(t *testing.T) {
	th := newTestHost(t)
	require.ErrorContains(t, th.Upgrade(t.Context(), "v0.2.2"), "nexul install")
}

func TestUpgrade_OtherVersion_SwapsTheBinaryAndReruns(t *testing.T) {
	th := newTestHost(t)
	th.installed(t)

	require.NoError(t, th.Upgrade(t.Context(), "0.2.2"))

	self, err := th.Executable()
	require.NoError(t, err)
	assert.Equal(t, "nexul-binary", readFile(t, self))
	assert.Equal(t, []string{self, "upgrade", "--version", "v0.2.2"}, th.reexec)
	assert.False(t, th.exec.ran("systemctl"), "the units are upgraded by the new binary, not this one")
}

func TestUpgrade_ThisVersion_ReplacesEveryUnitsBinaryAndRestartsIt(t *testing.T) {
	th := newTestHost(t)
	dir := th.installed(t)
	for _, name := range []string{"nexul-server", "nexul-runner-instance", "nexul-automations-instance"} {
		u, err := th.loadUnit(name)
		require.NoError(t, err)
		u.Version = "v0.2.0"
		require.NoError(t, os.WriteFile(u.Exec, []byte("old"), 0o755))
		require.NoError(t, th.saveUnit(*u))
	}
	settings, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	settings["NEXUL_VERSION"] = "0.2.0"
	require.NoError(t, writeEnvFile(filepath.Join(dir, ".env"), settings))

	require.NoError(t, th.Upgrade(t.Context(), "v0.2.1"))

	opt := th.Paths.UnitRoot
	assert.Equal(t, "nexul-server-binary", readFile(t, filepath.Join(opt, "server", "nexul-server")))
	assert.Equal(t, "nexul-runner-binary", readFile(t, filepath.Join(opt, "runner-instance", "nexul-runner")))
	assert.Equal(t, "nexul-automations-binary", readFile(t, filepath.Join(opt, "automations-instance", "nexul-automations")))
	assert.Equal(t, []string{
		"systemctl restart nexul-server.service",
		"systemctl restart nexul-runner-instance.service",
		"systemctl restart nexul-automations-instance.service",
	}, th.exec.callsTo("systemctl restart"), "OpenObserve's pin did not change, so it keeps running")
	u, err := th.loadUnit("nexul-runner-instance")
	require.NoError(t, err)
	assert.Equal(t, "v0.2.1", u.Version)
	after, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "0.2.1", after["NEXUL_VERSION"])
	assert.Contains(t, th.out.String(), "Nexul v0.2.1 is running.")
	assert.Nil(t, th.reexec)
}

func TestUpgrade_NewOpenObservePin_ReplacesAndRestartsIt(t *testing.T) {
	th := newTestHost(t)
	th.installed(t)
	th.OpenObserve = newFakeOpenObserve(t, "v1.0.5")

	require.NoError(t, th.Upgrade(t.Context(), "v0.2.1"))

	assert.Equal(t, "openobserve-v1.0.5", readFile(t, filepath.Join(th.Paths.UnitRoot, "openobserve", "openobserve")))
	assert.Equal(t, []string{"systemctl restart nexul-openobserve.service"}, th.exec.callsTo("systemctl restart"))
}

func TestUpgrade_FailingDownload_StopsAtThatUnit(t *testing.T) {
	th := newTestHost(t)
	th.installed(t)
	u, err := th.loadUnit("nexul-server")
	require.NoError(t, err)
	u.Version = "v0.2.0"
	require.NoError(t, th.saveUnit(*u))
	th.release.checksum = map[string]string{"nexul-server-linux-amd64": strings.Repeat("0", 64)}

	require.ErrorContains(t, th.Upgrade(t.Context(), "v0.2.1"), "nexul-server: checksum mismatch")
	assert.False(t, th.exec.ran("systemctl restart"))
}

func TestRunUpgrade_Detach_StartsATransientUnit(t *testing.T) {
	th := newTestHost(t)
	require.NoError(t, th.runUpgrade(t.Context(), []string{"--detach", "--version", "v0.2.2"}))

	calls := th.exec.callsTo("systemd-run")
	require.Len(t, calls, 1)
	self, _ := th.Executable()
	assert.Regexp(t, `^systemd-run --unit nexul-upgrade-[a-z2-7]{8} --collect --quiet `+self+` upgrade --version v0.2.2 --yes$`, calls[0])
	assert.Contains(t, th.out.String(), "journalctl -u nexul-upgrade-")
}

func TestUninstall(t *testing.T) {
	t.Run("nothing installed", func(t *testing.T) {
		th := newTestHost(t)
		require.ErrorContains(t, th.Uninstall(t.Context(), false, true), "not installed")
	})

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

	t.Run("removes every unit, hosts first, and keeps the install directory", func(t *testing.T) {
		th := newTestHost(t)
		dir := th.installed(t)
		th.Interactive = true
		th.In = bufio.NewReader(strings.NewReader("y\n"))

		require.NoError(t, th.Uninstall(t.Context(), false, false))

		assert.Equal(t, []string{
			"systemctl disable --now nexul-automations-instance.service",
			"systemctl disable --now nexul-runner-instance.service",
			"systemctl disable --now nexul-openobserve.service",
			"systemctl disable --now nexul-server.service",
		}, th.exec.callsTo("systemctl disable"))
		assert.Len(t, th.instance.calls("/api/runners/self/remove"), 1)
		assert.Len(t, th.instance.calls("/api/automation-hosts/self/remove"), 1)
		units, err := th.loadUnits()
		require.NoError(t, err)
		assert.Empty(t, units)
		assert.NoDirExists(t, th.Paths.UnitRoot)
		assert.NoFileExists(t, filepath.Join(th.Paths.Sudoers, "nexul-automations-instance"))
		assert.NoFileExists(t, filepath.Join(th.Paths.BinDir, "nexul"))
		assert.NoFileExists(t, th.Paths.Config)
		assert.FileExists(t, filepath.Join(dir, ".env"))
		assert.Contains(t, th.out.String(), "nexul install --dir "+dir)
	})

	t.Run("purge deletes the install directory", func(t *testing.T) {
		th := newTestHost(t)
		dir := th.installed(t)
		require.NoError(t, th.runUninstall(t.Context(), []string{"--purge", "--yes"}))
		assert.NoDirExists(t, dir)
		assert.Contains(t, th.out.String(), "DELETES "+dir)
	})

	t.Run("a remote runner alone is removed without a server install", func(t *testing.T) {
		th := newTestHost(t)
		th.hostInstalled(t, kindRunner, "edge")
		require.NoError(t, th.Uninstall(t.Context(), false, true))
		assert.NoFileExists(t, filepath.Join(th.Paths.Services, "nexul-runner-edge.service"))
		assert.Contains(t, th.out.String(), "Files ........... removed")
	})

	t.Run("a failing daemon-reload stops before the directory goes", func(t *testing.T) {
		th := newTestHost(t)
		th.installed(t)
		th.set("systemctl daemon-reload", "", errors.New("bus down"))
		require.ErrorContains(t, th.Uninstall(t.Context(), false, true), "bus down")
		assert.DirExists(t, filepath.Join(th.Paths.UnitRoot, "automations-instance"))
	})
}

func TestStatus(t *testing.T) {
	t.Run("not installed", func(t *testing.T) {
		th := newTestHost(t)
		require.ErrorContains(t, th.Status(t.Context()), "not installed")
	})
	t.Run("lists every unit with its kind, state and version", func(t *testing.T) {
		th := newTestHost(t)
		dir := th.installed(t)
		th.set("systemctl is-active", "active", nil)
		th.set("systemctl is-active nexul-runner-instance.service", "failed", nil)

		require.NoError(t, th.Status(t.Context()))

		out := th.out.String()
		assert.Contains(t, out, "Directory  "+dir)
		assert.Regexp(t, `server\s+nexul-server\s+active\s+v0.2.1`, out)
		assert.Regexp(t, `logs\s+nexul-openobserve\s+active\s+v1.0.4`, out)
		assert.Regexp(t, `runner\s+nexul-runner-instance\s+failed\s+v0.2.1`, out)
		assert.Regexp(t, `automations\s+nexul-automations-instance\s+active\s+v0.2.1`, out)
	})
	t.Run("a manager with nothing to say reports unknown", func(t *testing.T) {
		th := newTestHost(t)
		th.hostInstalled(t, kindRunner, "edge")
		require.NoError(t, th.Status(t.Context()))
		assert.Regexp(t, `runner\s+nexul-runner-edge\s+unknown`, th.out.String())
		assert.NotContains(t, th.out.String(), "Directory")
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
