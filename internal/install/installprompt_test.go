package install

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installAsking runs an interactive install answered with the directory, the port, then answers.
func installAsking(t *testing.T, th *testHost, args []string, answers string) string {
	t.Helper()
	dir := filepath.Join(th.root, "data", "nexul")
	th.bootedServer(t, dir)
	th.Interactive = true
	th.feedStdin(dir + "\n" + strconv.Itoa(th.webPort(t)) + "\n" + answers)
	require.NoError(t, th.runInstall(t.Context(), args))
	return dir
}

func unitToken(t *testing.T, th *testHost, unit string) string {
	t.Helper()
	env, err := readEnvFile(filepath.Join(th.Paths.UnitRoot, unit, "env"))
	require.NoError(t, err)
	return env[opTokenEnv]
}

func installToken(t *testing.T, dir string) string {
	t.Helper()
	settings, err := readEnvFile(filepath.Join(dir, ".env"))
	require.NoError(t, err)
	return settings[opTokenEnv]
}

func TestInstall_Interactive_PastedTokensGoToTheServerAndTheBundledRunner(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	hidden := 0
	th.HideEcho = func() (func(), error) { hidden++; return func() { hidden-- }, nil }

	dir := installAsking(t, th, nil, "two words\n"+testOPToken+"\n"+testOPTokenNew+"\n")

	assert.Equal(t, testOPToken, installToken(t, dir))
	assert.Equal(t, testOPToken, unitToken(t, th, "server"))
	assert.Equal(t, testOPTokenNew, unitToken(t, th, "runner-instance"))
	out := th.out.String()
	assert.Contains(t, out, "holds spaces", "an unusable paste is asked for again")
	assert.Contains(t, out, "nexul server 1password, nexul runner 1password <name>")
	assert.NotContains(t, out, testOPToken)
	assert.NotContains(t, out, testOPTokenNew)
	assert.Zero(t, hidden, "echo is back on after every read")
	assertNoExecHolds(t, th, testOPToken, testOPTokenNew)
}

func TestInstall_Interactive_EnterSkipsBothTokens(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")

	dir := installAsking(t, th, nil, "\n\n")

	assert.Empty(t, installToken(t, dir))
	assert.Empty(t, unitToken(t, th, "server"))
	assert.Empty(t, unitToken(t, th, "runner-instance"))
}

func TestInstall_Interactive_AServerTokenFileSkipsTheServerPrompt(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	file := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(file, []byte(testOPToken+"\n"), 0o600))

	dir := installAsking(t, th, []string{"--1password-token-file", file}, testOPTokenNew+"\n")

	assert.Equal(t, testOPToken, installToken(t, dir))
	assert.Equal(t, testOPTokenNew, unitToken(t, th, "runner-instance"), "the one pasted answer was the runner's")
	assert.Equal(t, 1, strings.Count(th.out.String(), "paste a token"))
}

func TestInstall_WithoutATerminal_NeverAsksForTokens(t *testing.T) {
	th := newTestHost(t)
	withVersion(t, "v0.2.1")
	dir := filepath.Join(th.root, "data", "nexul")
	th.bootedServer(t, dir)
	th.feedStdin(testOPToken + "\n" + testOPTokenNew + "\n")

	require.NoError(t, th.runInstall(t.Context(), []string{"--dir", dir, "--port", strconv.Itoa(th.webPort(t)), "--yes"}))

	assert.NotContains(t, th.out.String(), "paste a token")
	assert.Empty(t, installToken(t, dir))
	assert.Empty(t, unitToken(t, th, "runner-instance"))
}
