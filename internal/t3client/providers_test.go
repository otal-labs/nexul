package t3client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

func TestClient_Providers_ParsesConfig(t *testing.T) {
	t.Parallel()
	c := &Client{config: json.RawMessage(`{"providers":[
		{"instanceId":"claude","driver":"claude-code","displayName":"Claude Code","enabled":true,"installed":true,
		 "models":[
			{"slug":"claude-sonnet-4-5","name":"Sonnet 4.5","isDefault":true},
			{"slug":"claude-3-5-haiku","name":"Old Haiku","isLegacy":true}
		 ]},
		{"instanceId":"opencode","driver":"opencode","enabled":true,"installed":true,"availability":"unavailable","models":[]},
		{"instanceId":"codex","driver":"codex","enabled":false,"installed":true,"models":[]}
	]}`)}

	providers, err := c.Providers()
	require.NoError(t, err)
	require.Len(t, providers, 1, "unavailable and disabled instances are dropped")
	assert.Equal(t, "claude", providers[0].ID)
	assert.Equal(t, "claude-code", providers[0].Driver, "the driver kind travels beside the instance id for the setup gate")
	assert.Equal(t, "Claude Code", providers[0].Name)
	assert.Equal(t, []harness.ProviderModel{{Slug: "claude-sonnet-4-5", Name: "Sonnet 4.5", IsDefault: true}}, providers[0].Models, "legacy models are dropped")
}

func TestClient_Providers_DriverNameFallback(t *testing.T) {
	t.Parallel()
	c := &Client{config: json.RawMessage(`{"providers":[{"instanceId":"x","driver":"opencode","enabled":true,"installed":true,"models":[]}]}`)}
	providers, err := c.Providers()
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, "opencode", providers[0].Name)
}
