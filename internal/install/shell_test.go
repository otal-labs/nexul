package install

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellSwitch_SetsTheComputersOwnOptInAndRestartsItsRunner(t *testing.T) {
	th := newTestHost(t)
	th.sudoFromAlice(t)
	require.NoError(t, th.runInstall(t.Context(), []string{"computer", "--no-shell", "--token", computerToken(t, th.web.URL+"/")}))
	envFile := filepath.Join(th.Home, ".local", "share", "nexul", "computer", "env")
	env, err := readEnvFile(envFile)
	require.NoError(t, err)
	assert.Equal(t, "off", env["NEXUL_SHELL"], "--no-shell installs it with commands off")

	before := len(th.exec.callsTo("systemctl"))
	require.NoError(t, th.runShellSwitch(t.Context(), []string{"on"}))
	calls := th.exec.callsTo("systemctl")
	require.Greater(t, len(calls), before)
	assert.Equal(t, "systemctl restart nexul-computer.service", calls[len(calls)-1])
	env, err = readEnvFile(envFile)
	require.NoError(t, err)
	assert.NotContains(t, env, "NEXUL_SHELL")
	assert.Contains(t, th.out.String(), "Commands from Nexul are on on this computer; its runner restarted.")

	require.NoError(t, th.runShellSwitch(t.Context(), []string{"off"}))
	env, err = readEnvFile(envFile)
	require.NoError(t, err)
	assert.Equal(t, "off", env["NEXUL_SHELL"])
	assert.Equal(t, "personal", env["NEXUL_RUNNER_MODE"], "the rest of the unit is kept")
}

func TestShellSwitch_RefusesWhatItCannotDo(t *testing.T) {
	th := newTestHost(t)
	require.ErrorContains(t, th.runShellSwitch(t.Context(), []string{"maybe"}), "usage: nexul shell on|off")
	require.ErrorContains(t, th.runShellSwitch(t.Context(), nil), "usage")
	th.sudoFromAlice(t)
	require.ErrorContains(t, th.runShellSwitch(t.Context(), []string{"off"}), "no Nexul runner installed")
}
