package install

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const workspaceLookupPath = "/api/automation-hosts/self/workspace"

// instanceAnswers has the fake instance name the workspace id a slug resolves to, and the workspace ids whose
// automations are placed on the asking host.
func (th *testHost) instanceAnswers(slug, id string, placed ...string) {
	var rows []string
	for _, p := range placed {
		rows = append(rows, fmt.Sprintf(`{"id":%q,"slug":"slug-of-%s"}`, p, p))
	}
	th.instance.get[workspaceLookupPath] = fmt.Sprintf(`{"workspace":{"id":%q,"slug":%q},"placed":[%s]}`, id, slug, strings.Join(rows, ","))
}

func TestWorkspaceTokenEnv_RoundTripsAnyID(t *testing.T) {
	// The same names automations/test/managers.test.ts reads back on the host.
	assert.Equal(t, "NEXUL_OP_TOKEN_776F726B73706163652D64656661756C74", workspaceTokenEnv("workspace-default"))
	assert.Equal(t, "NEXUL_OP_TOKEN_30313961366633652D376331642D376232612D396630302D316332643365346635613662", workspaceTokenEnv("019a6f3e-7c1d-7b2a-9f00-1c2d3e4f5a6b"))
	for _, id := range []string{"workspace-default", "019a6f3e-7c1d-7b2a-9f00-1c2d3e4f5a6b", "Acme_Ops", "ws.1/ü"} {
		got, ok := tokenWorkspace(workspaceTokenEnv(id))
		assert.True(t, ok, id)
		assert.Equal(t, id, got)
	}
	for _, name := range []string{"NEXUL_OP_TOKEN_", "NEXUL_OP_TOKEN_ACME", "NEXUL_OP_TOKEN_776f", "NEXUL_OP_TOKEN_777", "OP_SERVICE_ACCOUNT_TOKEN"} {
		_, ok := tokenWorkspace(name)
		assert.False(t, ok, name)
	}
}

func TestAutomationsToken_SetRotateRemove_KeyedByWorkspaceID(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindAutomations, "edge")
	th.instanceAnswers("acme", "ws-acme", "ws-acme")
	envFile := filepath.Join(th.Paths.UnitRoot, "automations-edge", "env")
	key := workspaceTokenEnv("ws-acme")

	th.feedStdin(testOPToken + "\n")
	require.NoError(t, th.runAutomations(t.Context(), []string{"1password", "edge", "--workspace", "acme"}))
	u, err := th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	assert.Equal(t, testOPToken, u.Env[key], "the token is filed under the workspace's id, never its slug")
	assert.Contains(t, u.Env, "NEXUL_AUTOMATIONS_HOST_NAME", "the rest of the env stays")
	assertMode(t, 0o600, envFile)
	assert.True(t, th.exec.ran("systemctl restart nexul-automations-edge.service"), "the host restarts to pick a token up")
	asked := th.instance.calls(workspaceLookupPath)
	require.Len(t, asked, 1)
	assert.Equal(t, "Bearer cred-edge", asked[0].auth, "the command asks as the host, with its own credential")
	assert.Equal(t, "slug=acme", asked[0].query)

	th.feedStdin(testOPTokenNew + "\n")
	require.NoError(t, th.runAutomations(t.Context(), []string{"1password", "edge", "--workspace", "acme"}))
	u, err = th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	assert.Equal(t, testOPTokenNew, u.Env[key], "the same workspace's token rotates")

	th.exec.calls = nil
	th.out.Reset()
	require.NoError(t, th.Status(t.Context()))
	assert.Regexp(t, `nexul-automations-edge\s+\S+\s+v0.2.1\s+1Password token for workspace ws-acme`, th.out.String())

	require.NoError(t, th.runAutomations(t.Context(), []string{"1password", "edge", "--remove"}))
	u, err = th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	assert.NotContains(t, u.Env, key)
	assert.Len(t, th.instance.calls(workspaceLookupPath), 2, "removing asks the instance nothing")

	assertNoExecHolds(t, th, testOPToken, testOPTokenNew)
	assert.NotContains(t, th.out.String(), "ops_acme", "the terminal never shows a token")
}

func TestAutomationsToken_RefusesASecondWorkspace(t *testing.T) {
	tests := []struct {
		name    string
		held    string
		slug    string
		answer  func(th *testHost)
		wantErr string
	}{
		{
			name: "a token for another workspace already held", held: "ws-acme", slug: "globex",
			answer:  func(th *testHost) { th.instanceAnswers("globex", "ws-globex") },
			wantErr: "already holds a 1Password token for another workspace (ws-acme)",
		},
		{
			name: "the slug renamed onto a new workspace", held: "ws-acme", slug: "acme",
			answer:  func(th *testHost) { th.instanceAnswers("acme", "ws-new-acme", "ws-acme") },
			wantErr: "already holds a 1Password token for another workspace (ws-acme)",
		},
		{
			name: "another workspace's automations placed on the host", slug: "acme",
			answer:  func(th *testHost) { th.instanceAnswers("acme", "ws-acme", "ws-acme", "ws-globex") },
			wantErr: "has automations of the workspace slug-of-ws-globex placed on it",
		},
		{
			name: "a slug the instance doesn't know", slug: "acme",
			answer: func(th *testHost) {
				th.instance.get[workspaceLookupPath] = `{"error":"not found"}`
				th.instance.status[workspaceLookupPath] = http.StatusNotFound
			},
			wantErr: `the instance has no workspace with the slug "acme"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := newTestHost(t)
			th.hostInstalled(t, kindAutomations, "edge")
			if tt.held != "" {
				u, err := th.loadUnit("nexul-automations-edge")
				require.NoError(t, err)
				u.Env[workspaceTokenEnv(tt.held)] = testOPToken
				require.NoError(t, th.saveUnit(*u))
			}
			tt.answer(th)
			th.feedStdin(testOPTokenNew + "\n")

			err := th.runAutomations(t.Context(), []string{"1password", "edge", "--workspace", tt.slug})

			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), testOPTokenNew)
			assert.Empty(t, th.exec.calls, "a refused token leaves the host running")
			assert.NotContains(t, readFile(t, filepath.Join(th.Paths.UnitRoot, "automations-edge", "env")), testOPTokenNew)
		})
	}
}

func TestAutomationsToken_Upgrade_KeepsTheToken(t *testing.T) {
	th := newTestHost(t)
	th.hostInstalled(t, kindAutomations, "edge")
	th.instanceAnswers("acme", "ws-acme")
	require.NoError(t, th.SetAutomationsToken(t.Context(), "edge", "acme", testOPToken))
	u, err := th.loadUnit("nexul-automations-edge")
	require.NoError(t, err)
	u.Version = "v0.2.0"
	require.NoError(t, th.saveUnit(*u))
	th.exec.calls = nil

	require.NoError(t, th.Upgrade(t.Context(), "v0.2.1"))

	assert.True(t, th.exec.ran("systemctl restart nexul-automations-edge.service"), "the upgrade did replace the unit")
	env := readFile(t, filepath.Join(th.Paths.UnitRoot, "automations-edge", "env"))
	assert.Contains(t, env, workspaceTokenEnv("ws-acme")+"="+testOPToken)
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
		{"a workspace to remove", []string{"1password", "edge", "--workspace", "acme", "--remove"}, "", "--remove takes no --workspace"},
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
			th.instanceAnswers("acme", "ws-acme")
			th.feedStdin(tt.stdin)
			err := th.runAutomations(t.Context(), tt.args)
			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), testOPToken)
			assert.Empty(t, th.exec.calls, "a refused command leaves the host running")
			assert.NotContains(t, readFile(t, filepath.Join(th.Paths.UnitRoot, "automations-edge", "env")), "NEXUL_OP_TOKEN")
		})
	}
}
