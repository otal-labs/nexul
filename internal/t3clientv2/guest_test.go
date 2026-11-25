package t3clientv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestGuestArgs_GiveTheAgentOnlyNexulWithTheGuestsToken(t *testing.T) {
	claude, err := guestArgs("claudeAgent", "https://nexul.example.com/mcp")
	require.NoError(t, err)
	assert.Equal(t, `--strict-mcp-config --mcp-config '{"mcpServers":{"nexul":{"headers":{"Authorization":"Bearer ${NEXUL_GUEST_TOKEN}"},"type":"http","url":"https://nexul.example.com/mcp"}}}'`, claude)

	codex, err := guestArgs("codex", "https://nexul.example.com/mcp")
	require.NoError(t, err)
	assert.Equal(t, `-c 'mcp_servers.nexul.url="https://nexul.example.com/mcp"' -c 'mcp_servers.nexul.http_headers={}' -c 'mcp_servers.nexul.bearer_token_env_var="NEXUL_GUEST_TOKEN"'`, codex)

	_, err = guestArgs("opencode", "https://nexul.example.com/mcp")
	require.ErrorIs(t, err, apperrs.ErrConflict)
	_, err = guestArgs("codex", "https://nexul.example.com/mcp' --rm")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestGuestInstance_CopiesTheOwnersProviderButNeverItsSecrets(t *testing.T) {
	own := map[string]providerInstance{"codex_work": {
		Driver: "codex", Environment: []envVar{{Name: "HTTPS_PROXY", Value: "http://proxy:3128"}},
		Config: map[string]any{"homePath": "/home/fahad/.codex-work", "launchArgs": "--search"},
	}}

	inst, err := guestInstance(own, "codex_work", "https://nexul.example.com/mcp", "nxp_guest", "Onik on Fahad's PC")
	require.NoError(t, err)

	assert.Equal(t, "codex", inst.Driver)
	assert.Equal(t, "Onik on Fahad's PC", inst.DisplayName)
	assert.Equal(t, "/home/fahad/.codex-work", inst.Config["homePath"], "the owner's provider sign-in stays the one it runs on")
	assert.Contains(t, inst.Config["launchArgs"], "--search -c ")
	assert.Equal(t, []envVar{{Name: "HTTPS_PROXY", Value: "http://proxy:3128"}, {Name: guestTokenEnv, Value: "nxp_guest", Sensitive: true}}, inst.Environment)
	assert.Equal(t, "nxp_guest", inst.Environment[1].Value, "the token rides in the environment")
	assert.NotContains(t, inst.Config["launchArgs"], "nxp_guest", "never in the arguments")

	builtIn, err := guestInstance(nil, "claudeAgent", "https://nexul.example.com/mcp", "nxp_guest", "x")
	require.NoError(t, err)
	assert.Equal(t, "claudeAgent", builtIn.Driver, "a driver with no settings of its own is its built-in instance")

	own["codex_work"] = providerInstance{Driver: "codex", Environment: []envVar{{Name: "OPENAI_API_KEY", Sensitive: true, ValueRedacted: true}}}
	_, err = guestInstance(own, "codex_work", "https://nexul.example.com/mcp", "nxp_guest", "x")
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestOpenGuest_AddsTheInstanceAndCloseGuestRemovesIt(t *testing.T) {
	f, h := newFake(t, protocol)
	f.SessionPermissions = []string{"orchestration:operate", "providers:manage"}
	s := f.Session()

	id, err := h.OpenGuest(t.Context(), s, "claudeAgent", "https://nexul.example.com/mcp", "nxp_guest", "Onik on Fahad's PC")
	require.NoError(t, err)

	created := <-f.SettingsUpdates
	mutation := created["providerInstanceMutation"].(map[string]any)
	assert.Equal(t, "create", mutation["operation"])
	assert.Equal(t, id, mutation["instanceId"])
	providers, err := h.ListProviders(t.Context(), s)
	require.NoError(t, err)
	assert.True(t, slicesHasID(providers, id), "a run can start on it at once")

	require.NoError(t, h.CloseGuest(t.Context(), s, id))
	removed := (<-f.SettingsUpdates)["providerInstanceMutation"].(map[string]any)
	assert.Equal(t, map[string]any{"operation": "remove", "instanceId": id}, removed)
	require.ErrorIs(t, h.CloseGuest(t.Context(), s, "claudeAgent"), apperrs.ErrInvalid, "only an instance Nexul made")
}

func TestOpenGuest_ASessionWithoutProvidersManageIsToldToPairAgain(t *testing.T) {
	f, h := newFake(t, protocol)
	f.SessionPermissions = []string{"orchestration:operate"}

	_, err := h.OpenGuest(t.Context(), f.Session(), "claudeAgent", "https://nexul.example.com/mcp", "t", "x")

	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "pair the computer again")
}

func slicesHasID(providers []harness.Provider, id string) bool {
	for _, p := range providers {
		if p.ID == id {
			return true
		}
	}
	return false
}
