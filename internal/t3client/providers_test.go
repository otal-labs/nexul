package t3client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

// The fixture is trimmed from real T3 provider snapshots (Claude and OpenCode instances).
func TestClient_Providers_ParsesConfig(t *testing.T) {
	t.Parallel()
	c := &Client{config: json.RawMessage(`{"providers":[
		{"instanceId":"claudeAgent","driver":"claudeAgent","displayName":"Claude","enabled":true,"installed":true,
		 "models":[
			{"slug":"claude-opus-5-5","name":"Claude Opus 5.5","badge":"new","isCustom":false,"capabilities":{"optionDescriptors":[
				{"id":"effort","label":"Reasoning","type":"select","options":[
					{"id":"low","label":"Low"},{"id":"medium","label":"Medium","isDefault":true},
					{"id":"ultracode","label":"Ultracode","description":"xhigh effort plus multi-agent workflow orchestration"},
					{"id":"ultrathink","label":"Ultrathink"}],"promptInjectedValues":["ultrathink"]},
				{"id":"fastMode","label":"Fast Mode","type":"boolean"},
				{"id":"contextWindow","label":"Context Window","type":"select","options":[
					{"id":"200k","label":"200k"},{"id":"1m","label":"1M","isDefault":true}]}]}},
			{"slug":"claude-fable-5","name":"Claude Fable 5","isCustom":false,"isLegacy":true,"capabilities":null}
		 ]},
		{"instanceId":"opencode","driver":"opencode","enabled":true,"installed":true,
		 "models":[{"slug":"github-copilot/claude-haiku-4.5","name":"Claude Haiku 4.5 (latest)","subProvider":"GitHub Copilot","isCustom":false,
			"capabilities":{"optionDescriptors":[{"id":"agent","label":"Agent","type":"select","options":[
				{"id":"build","label":"Build"},{"id":"plan","label":"Plan"}],"currentValue":"build"}]}}]},
		{"instanceId":"cursor","driver":"cursor","enabled":true,"installed":true,"availability":"unavailable","models":[]},
		{"instanceId":"codex","driver":"codex","enabled":false,"installed":true,"models":[]}
	]}`)}

	providers, err := c.Providers()
	require.NoError(t, err)
	require.Len(t, providers, 2, "unavailable and disabled instances are dropped")
	assert.Equal(t, "claudeAgent", providers[0].Driver, "the driver kind travels beside the instance id for the setup gate")
	assert.Equal(t, []harness.ProviderModel{
		{Slug: "claude-opus-5-5", Name: "Claude Opus 5.5", IsNew: true, Options: []harness.ModelOption{
			{ID: "effort", Label: "Reasoning", Type: harness.OptionSelect, Choices: []harness.OptionChoice{
				{ID: "low", Label: "Low"}, {ID: "medium", Label: "Medium", IsDefault: true},
				{ID: "ultracode", Label: "Ultracode", Description: "xhigh effort plus multi-agent workflow orchestration"},
			}},
			{ID: "fastMode", Label: "Fast Mode", Type: harness.OptionSwitch},
			{ID: "contextWindow", Label: "Context Window", Type: harness.OptionSelect, Choices: []harness.OptionChoice{
				{ID: "200k", Label: "200k"}, {ID: "1m", Label: "1M", IsDefault: true},
			}},
		}},
		{Slug: "claude-fable-5", Name: "Claude Fable 5", IsLegacy: true},
	}, providers[0].Models, "prompt-injected choices are dropped; legacy models stay, flagged")
	assert.Equal(t, []harness.ProviderModel{
		{Slug: "github-copilot/claude-haiku-4.5", Name: "Claude Haiku 4.5 (latest)", SubProvider: "GitHub Copilot", Options: []harness.ModelOption{
			{ID: "agent", Label: "Agent", Type: harness.OptionSelect, Choices: []harness.OptionChoice{
				{ID: "build", Label: "Build", IsDefault: true}, {ID: "plan", Label: "Plan"},
			}},
		}},
	}, providers[1].Models, "with no marked default, the current value is the default")
}

func TestClient_Providers_DriverNameFallback(t *testing.T) {
	t.Parallel()
	c := &Client{config: json.RawMessage(`{"providers":[{"instanceId":"x","driver":"opencode","enabled":true,"installed":true,"models":[]}]}`)}
	providers, err := c.Providers()
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, "opencode", providers[0].Name)
}
