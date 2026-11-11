package install

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomationsToken_PerWorkspace_SetRotateRemove(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindAutomations, "edge")
	envFile := filepath.Join(th.Paths.UnitRoot, "automations-edge", "env")

	require.NoError(t, th.SetAutomationsToken(t.Context(), "edge", "acme", testOPToken))
	require.NoError(t, th.SetAutomationsToken(t.Context(), "edge", "acme-ops", testOPTokenNew))

	u, err := th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	assert.Equal(t, testOPToken, u.Env["NEXUL_OP_TOKEN_ACME"])
	assert.Equal(t, testOPTokenNew, u.Env["NEXUL_OP_TOKEN_ACME_OPS"], "a dashed slug gets a name an env file can hold")
	assert.Contains(t, u.Env, "NEXUL_AUTOMATIONS_HOST_NAME", "the rest of the env stays")
	assertMode(t, 0o600, envFile)
	assert.True(t, th.exec.ran("systemctl restart nexul-automations-edge.service"), "the host restarts to pick a token up")

	th.exec.calls = nil
	th.out.Reset()
	require.NoError(t, th.Status(t.Context()))
	assert.Regexp(t, `nexul-automations-edge\s+\S+\s+v0.2.1\s+1Password token for acme, acme-ops`, th.out.String())

	require.NoError(t, th.SetAutomationsToken(t.Context(), "edge", "acme", ""))
	u, err = th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	assert.NotContains(t, u.Env, "NEXUL_OP_TOKEN_ACME", "removing one workspace's token leaves the other's")
	assert.Equal(t, testOPTokenNew, u.Env["NEXUL_OP_TOKEN_ACME_OPS"])

	assertNoExecHolds(t, th, testOPToken, testOPTokenNew)
	assert.NotContains(t, th.out.String(), "ops_acme", "the terminal never shows a token")
}

func TestAutomationsToken_Upgrade_KeepsEveryWorkspaceToken(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindAutomations, "edge")
	require.NoError(t, th.SetAutomationsToken(t.Context(), "edge", "acme", testOPToken))
	require.NoError(t, th.SetAutomationsToken(t.Context(), "edge", "globex", testOPTokenNew))
	u, err := th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	u.Version = "v0.2.0"
	require.NoError(t, th.saveUnit(*u))
	th.exec.calls = nil

	require.NoError(t, th.Upgrade(t.Context(), "v0.2.1"))

	assert.True(t, th.exec.ran("systemctl restart nexul-automations-edge.service"), "the upgrade did replace the unit")
	env := readFile(t, filepath.Join(th.Paths.UnitRoot, "automations-edge", "env"))
	assert.Contains(t, env, "NEXUL_OP_TOKEN_ACME="+testOPToken)
	assert.Contains(t, env, "NEXUL_OP_TOKEN_GLOBEX="+testOPTokenNew)
}

func TestRunAutomations_RefusesANewTokenButStillRemovesOne(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindAutomations, "edge")
	th.feedStdin(testOPToken + "\n")

	require.ErrorContains(t, th.runAutomations(t.Context(), []string{"1password", "edge", "--workspace", "acme"}), "can't hold a 1Password token yet")
	u, err := th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	assert.NotContains(t, u.Env, "NEXUL_OP_TOKEN_ACME")

	require.NoError(t, th.SetAutomationsToken(t.Context(), "edge", "acme", testOPToken))
	require.NoError(t, th.runAutomations(t.Context(), []string{"1password", "edge", "--workspace", "acme", "--remove"}))
	u, err = th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	assert.NotContains(t, u.Env, "NEXUL_OP_TOKEN_ACME")
}

func TestRunAutomations_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		stdin   string
		wantErr string
	}{
		{"no manager", []string{}, "", "usage: nexul automations 1password"},
		{"another manager", []string{"vault", "edge"}, "", "usage: nexul automations 1password"},
		{"no host", []string{"1password", "--workspace", "acme"}, "", "usage: nexul automations 1password"},
		{"no workspace", []string{"1password", "edge"}, testOPToken, "--workspace is required"},
		{"a workspace that is not a slug", []string{"1password", "edge", "--workspace", "Acme_Ops"}, testOPToken, "--workspace is required"},
		{"the token as a flag", []string{"1password", "edge", "--workspace", "acme", "--token", testOPToken}, "", "flag provided but not defined"},
		{"the token as an argument", []string{"1password", "edge", "--workspace", "acme", testOPToken}, "", "unexpected argument"},
		{"nothing on stdin", []string{"1password", "edge", "--workspace", "acme"}, "", "no token given"},
		{"a host that is not installed", []string{"1password", "ghost", "--workspace", "acme"}, testOPToken, `no automations named "ghost"`},
		{"a runner's name", []string{"1password", "run", "--workspace", "acme"}, testOPToken, `no automations named "run"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.hostInstalled(t, kindAutomations, "edge")
			th.hostInstalled(t, kindRunner, "run")
			th.feedStdin(tt.stdin)
			err := th.runAutomations(t.Context(), tt.args)
			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), testOPToken)
			assert.Empty(t, th.exec.calls, "a refused command leaves the host running")
			assert.NotContains(t, readFile(t, filepath.Join(th.Paths.UnitRoot, "automations-edge", "env")), "NEXUL_OP_TOKEN")
		})
	}
}
