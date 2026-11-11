package install

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testOPToken    = "ops_acme-token-1"
	testOPTokenNew = "ops_acme-token-2"
)

func (th *testHost) feedStdin(s string) { th.In = bufio.NewReader(strings.NewReader(s)) }

// assertNoExecHolds fails when any recorded command line carries one of the secrets.
func assertNoExecHolds(t *testing.T, th *testHost, secrets ...string) {
	t.Helper()
	for _, call := range th.exec.calls {
		for _, s := range secrets {
			assert.NotContains(t, call, s, "a token must never ride an argument list")
		}
	}
}

func TestRunnerToken_SetRotateRemove_OnEachServiceManager(t *testing.T) {
	tests := []struct {
		goos    string
		restart string
		// holder is the file that carries the token to the process.
		holder func(th *testHost, dir string) string
	}{
		{"linux", "systemctl restart nexul-runner-edge.service", func(_ *testHost, dir string) string { return filepath.Join(dir, "env") }},
		{"darwin", "launchctl bootstrap", func(th *testHost, _ string) string {
			return filepath.Join(th.Paths.Services, "io.nexul.nexul-runner-edge.plist")
		}},
		{"windows", "sc.exe start nexul-runner-edge", func(_ *testHost, dir string) string { return filepath.Join(dir, "unit.json") }},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			th := newTestHost(t)
			th.hostInstalled(t, kindRunner, "edge")
			th.GOOS = tt.goos
			if tt.goos == "darwin" {
				th.Getuid = func() int { return 1000 } // a Mac refuses root
			}
			dir := filepath.Join(th.Paths.UnitRoot, "runner-edge")
			holder := tt.holder(th, dir)

			require.NoError(t, th.SetRunnerToken(t.Context(), "edge", testOPToken))
			assert.Contains(t, readFile(t, holder), testOPToken)
			assertMode(t, 0o600, holder)
			assert.True(t, th.exec.ran(tt.restart), "the runner restarts to pick the token up")
			u, err := th.loadUnit("nexul-runner-edge")
			require.NoError(t, err)
			assert.Equal(t, testOPToken, u.Env[opTokenEnv])
			assert.Contains(t, u.Env, "NEXUL_RUNNER_NAME", "the rest of the env stays")

			th.exec.calls = nil
			require.NoError(t, th.SetRunnerToken(t.Context(), "edge", testOPTokenNew))
			assert.Contains(t, readFile(t, holder), testOPTokenNew)
			assert.NotContains(t, readFile(t, holder), testOPToken)
			assert.True(t, th.exec.ran(tt.restart), "a rotated token restarts the runner too")

			th.exec.calls = nil
			require.NoError(t, th.SetRunnerToken(t.Context(), "edge", ""))
			assert.NotContains(t, readFile(t, holder), "ops_acme")
			assert.True(t, th.exec.ran(tt.restart), "a removed token restarts the runner too")
			u, err = th.loadUnit("nexul-runner-edge")
			require.NoError(t, err)
			assert.NotContains(t, u.Env, opTokenEnv)
			assert.Contains(t, u.Env, "NEXUL_RUNNER_NAME")

			assertNoExecHolds(t, th, testOPToken, testOPTokenNew)
			assert.NotContains(t, th.out.String(), "ops_acme", "the terminal never shows the token")
		})
	}
}

func TestRunnerToken_Upgrade_KeepsIt(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindRunner, "edge")
	require.NoError(t, th.SetRunnerToken(t.Context(), "edge", testOPToken))
	u, err := th.loadUnit("nexul-runner-edge")
	require.NoError(t, err)
	u.Version = "v0.2.0"
	require.NoError(t, th.saveUnit(*u))
	th.exec.calls = nil

	require.NoError(t, th.Upgrade(t.Context(), "v0.2.1"))

	assert.True(t, th.exec.ran("systemctl restart nexul-runner-edge.service"), "the upgrade did replace the unit")
	assert.Contains(t, readFile(t, filepath.Join(th.Paths.UnitRoot, "runner-edge", "env")), opTokenEnv+"="+testOPToken)
	assertNoExecHolds(t, th, testOPToken)
}

func TestRunnerToken_Status_SaysWhetherOneIsSetNeverItsValue(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindRunner, "edge")
	th.hostInstalled(t, kindRunner, "bare")
	require.NoError(t, th.SetRunnerToken(t.Context(), "edge", testOPToken))
	th.out.Reset()

	require.NoError(t, th.Status(t.Context()))

	assert.Regexp(t, `nexul-runner-edge\s+\S+\s+v0.2.1\s+1Password token set`, th.out.String())
	assert.Regexp(t, `nexul-runner-bare\s+\S+\s+v0.2.1\s+no 1Password token`, th.out.String())
	assert.NotContains(t, th.out.String(), "ops_acme")
}

func TestInstallRunner_TokenFile_StartsTheRunnerWithTheToken(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	file := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(file, []byte(testOPToken+"\n"), 0o600))

	require.NoError(t, th.runInstall(t.Context(), []string{"runner", "--server", th.web.URL, "--name", "edge", "--code", "nxe_code", "--1password-token-file", file}))

	env, err := readEnvFile(filepath.Join(th.Paths.UnitRoot, "runner-edge", "env"))
	require.NoError(t, err)
	assert.Equal(t, testOPToken, env[opTokenEnv])
	assertNoExecHolds(t, th, testOPToken)
	for _, req := range th.instance.calls("/api/runners/enroll") {
		assert.NotContains(t, req.body, testOPToken)
	}
}

func TestInstallRunner_UnusableTokenFile_FailsBeforeEnrolling(t *testing.T) {
	multi := filepath.Join(t.TempDir(), "multi")
	require.NoError(t, os.WriteFile(multi, []byte("ops_a\nNEXUL_X=1\n"), 0o600))
	for name, file := range map[string]string{"a missing file": filepath.Join(t.TempDir(), "none"), "a multi-line file": multi} {
		t.Run(name, func(t *testing.T) {
			th := newTestHost(t)
			withVersion(t, "v0.2.1")
			err := th.runInstall(t.Context(), []string{"runner", "--server", th.web.URL, "--name", "edge", "--code", "nxe_code", "--1password-token-file", file})
			require.ErrorContains(t, err, "1password-token-file")
			assert.Empty(t, th.instance.calls("/api/runners/enroll"), "the code is not spent on an install that cannot finish")
		})
	}
}

func TestRunRunner_ReadsTheTokenFromStdin(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindRunner, "edge")
	th.feedStdin(testOPToken + "\n")

	require.NoError(t, th.runRunner(t.Context(), []string{"1password", "edge"}))
	u, err := th.loadUnit("nexul-runner-edge")
	require.NoError(t, err)
	assert.Equal(t, testOPToken, u.Env[opTokenEnv])

	require.NoError(t, th.runRunner(t.Context(), []string{"1password", "edge", "--remove"}))
	u, err = th.loadUnit("nexul-runner-edge")
	require.NoError(t, err)
	assert.NotContains(t, u.Env, opTokenEnv)
}

func TestRunRunner_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		stdin   string
		wantErr string
	}{
		{"no manager", []string{}, "", "usage: nexul runner 1password"},
		{"another manager", []string{"vault", "edge"}, "", "usage: nexul runner 1password"},
		{"no name", []string{"1password", "--remove"}, "", "usage: nexul runner 1password"},
		{"the token as a flag", []string{"1password", "edge", "--token", testOPToken}, "", "flag provided but not defined"},
		{"the token as an argument", []string{"1password", "edge", testOPToken}, "", "unexpected argument"},
		{"nothing on stdin", []string{"1password", "edge"}, "", "no token given"},
		{"a token with a second variable in it", []string{"1password", "edge"}, "ops_a NEXUL_X=1\n", "not a service account token"},
		{"a runner that is not installed", []string{"1password", "ghost"}, testOPToken, `no runner named "ghost"`},
		{"a runner that is not installed, before any token is read", []string{"1password", "ghost"}, "", `no runner named "ghost"`},
		{"a name that is not a name", []string{"1password", "../x"}, testOPToken, "no runner named"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.hostInstalled(t, kindRunner, "edge")
			th.feedStdin(tt.stdin)
			err := th.runRunner(t.Context(), tt.args)
			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), testOPToken)
			assert.Empty(t, th.exec.calls, "a refused command leaves the runner running")
			assert.NotContains(t, readFile(t, filepath.Join(th.Paths.UnitRoot, "runner-edge", "env")), opTokenEnv)
		})
	}
}

func TestRunnerToken_Remove_WithNoneSet_RestartsNothing(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindRunner, "edge")

	require.NoError(t, th.SetRunnerToken(t.Context(), "edge", ""))

	assert.Contains(t, th.out.String(), "has no 1Password token")
	assert.Empty(t, th.exec.calls)
}
