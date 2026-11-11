package install

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerToken_SetRotateRemove_ReachTheEnvFileAndTheUnitAndRestartTheServer(t *testing.T) {
	th := newTestHost(t)
	dir := th.installed(t)
	envFile := filepath.Join(dir, ".env")
	unitEnv := filepath.Join(th.Paths.UnitRoot, "server", "env")
	restart := "systemctl restart nexul-server.service"

	require.NoError(t, th.SetServerToken(t.Context(), testOPToken))
	assert.Contains(t, readFile(t, envFile), opTokenEnv+"="+testOPToken)
	assertMode(t, 0o600, envFile)
	assert.Contains(t, readFile(t, unitEnv), opTokenEnv+"="+testOPToken)
	assertMode(t, 0o600, unitEnv)
	assert.Contains(t, readFile(t, unitEnv), "NEXUL_DB_PATH=", "the rest of the server's env stays")
	assert.True(t, th.exec.ran(restart))

	th.exec.calls = nil
	require.NoError(t, th.SetServerToken(t.Context(), testOPTokenNew))
	assert.Contains(t, readFile(t, envFile), testOPTokenNew)
	assert.NotContains(t, readFile(t, envFile), testOPToken)
	assert.NotContains(t, readFile(t, unitEnv), testOPToken)
	assert.True(t, th.exec.ran(restart), "a rotated token restarts the server too")

	th.exec.calls = nil
	require.NoError(t, th.SetServerToken(t.Context(), ""))
	assert.NotContains(t, readFile(t, envFile), "ops_acme")
	assert.NotContains(t, readFile(t, unitEnv), "ops_acme")
	assert.Contains(t, readFile(t, envFile), "NEXUL_PORT=", "the install's other settings stay")
	assert.True(t, th.exec.ran(restart), "a removed token restarts the server too")

	assertNoExecHolds(t, th, testOPToken, testOPTokenNew)
	assert.NotContains(t, th.out.String(), "ops_acme", "the terminal never shows the token")
}

func TestServerToken_Upgrade_KeepsItInBothFiles(t *testing.T) {
	th := newTestHost(t)
	dir := th.installed(t)
	require.NoError(t, th.SetServerToken(t.Context(), testOPToken))
	u, err := th.loadUnit("nexul-server")
	require.NoError(t, err)
	u.Version = "v0.2.0"
	require.NoError(t, th.saveUnit(*u))
	settings, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	settings["NEXUL_VERSION"] = "0.2.0"
	require.NoError(t, writeEnvFile(filepath.Join(dir, ".env"), settings))

	require.NoError(t, th.Upgrade(t.Context(), "v0.2.1"))

	after, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "0.2.1", after["NEXUL_VERSION"], "the upgrade did rewrite the file")
	assert.Equal(t, testOPToken, after[opTokenEnv])
	unit, err := th.loadUnit("nexul-server")
	require.NoError(t, err)
	assert.Equal(t, testOPToken, unit.Env[opTokenEnv])
	assert.Equal(t, "v0.2.1", unit.Version)
}

func TestInstallServer_TokenFile_StartsTheServerWithTheToken(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "data", "nexul")
	th.bootedServer(t, dir)
	file := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(file, []byte(testOPToken+"\n"), 0o600))

	require.NoError(t, th.runInstall(t.Context(), []string{"--dir", dir, "--port", strconv.Itoa(th.webPort(t)), "--yes", "--1password-token-file", file}))

	settings, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, testOPToken, settings[opTokenEnv])
	env, err := readEnvFile(filepath.Join(th.Paths.UnitRoot, "server", "env"))
	require.NoError(t, err)
	assert.Equal(t, testOPToken, env[opTokenEnv])
	runnerEnv, err := readEnvFile(filepath.Join(th.Paths.UnitRoot, "runner-instance", "env"))
	require.NoError(t, err)
	assert.NotContains(t, runnerEnv, opTokenEnv, "the bundled runner has its own token")
	assertNoExecHolds(t, th, testOPToken)
}

func TestInstallServer_UnusableTokenFile_FailsBeforeInstalling(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	err := th.runInstall(t.Context(), []string{"--yes", "--1password-token-file", filepath.Join(t.TempDir(), "none")})
	require.ErrorContains(t, err, "1password-token-file")
	assert.Empty(t, th.exec.calls, "nothing was installed")
}

func TestRunServer_ReadsTheTokenFromStdinAndRefusesWhatItCannotUse(t *testing.T) {
	th := newTestHost(t)
	require.ErrorIs(t, th.runServer(t.Context(), []string{"1password", "--remove"}), errNoServer, "nothing to remove from before an install")

	th.installed(t)
	th.feedStdin(testOPToken + "\n")
	require.NoError(t, th.runServer(t.Context(), []string{"1password"}))
	u, err := th.loadUnit("nexul-server")
	require.NoError(t, err)
	assert.Equal(t, testOPToken, u.Env[opTokenEnv])

	require.NoError(t, th.runServer(t.Context(), []string{"1password", "--remove"}))
	u, err = th.loadUnit("nexul-server")
	require.NoError(t, err)
	assert.NotContains(t, u.Env, opTokenEnv)

	require.ErrorContains(t, th.runServer(t.Context(), []string{"vault"}), "usage: nexul server 1password")
	require.ErrorContains(t, th.runServer(t.Context(), []string{"1password", testOPToken}), "never taken from the command line")
	th.feedStdin("ops_a b\n")
	require.ErrorContains(t, th.runServer(t.Context(), []string{"1password"}), "not a service account token")
}
