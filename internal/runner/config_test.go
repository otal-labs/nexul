package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRunnerConfig(t *testing.T) {
	t.Setenv("NEXUL_SERVER_WS", "ws://server:8081/ws/runner")
	t.Setenv("NEXUL_RUNNER_SECRET", "shhh")
	t.Setenv("NEXUL_RUNNER_ID", "runner-a")
	t.Setenv("NEXUL_RUNNER_NAME", "alpha")
	t.Setenv("NEXUL_GIT_TOKEN", "ghp_abc")
	t.Setenv("NEXUL_RUNNER_HEARTBEAT", "5s")

	cfg, err := LoadRunnerConfig()
	require.NoError(t, err)
	assert.Equal(t, "ws://server:8081/ws/runner", cfg.ServerURL)
	assert.Equal(t, "shhh", cfg.Token)
	assert.Equal(t, "runner-a", cfg.RunnerID)
	assert.Equal(t, "alpha", cfg.Name)
	assert.Equal(t, "ghp_abc", cfg.GitToken)
	assert.Equal(t, 5*time.Second, cfg.HeartbeatInterval)
	assert.Equal(t, time.Second, cfg.BackoffBase)
	assert.Equal(t, 30*time.Second, cfg.BackoffMax)
}

func TestLoadRunnerConfig_FallsBackToHostname(t *testing.T) {
	t.Setenv("NEXUL_SERVER_WS", "ws://server")
	t.Setenv("NEXUL_RUNNER_SECRET", "shhh")
	t.Setenv("NEXUL_RUNNER_ID", "")
	cfg, err := LoadRunnerConfig()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.RunnerID)
}

func TestLoadRunnerConfig_InvalidDurationFallsBack(t *testing.T) {
	t.Setenv("NEXUL_RUNNER_HEARTBEAT", "not-a-duration")
	cfg, err := LoadRunnerConfig()
	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, cfg.HeartbeatInterval)
}

func TestRunnerConfig_Validate(t *testing.T) {
	valid := RunnerConfig{
		ServerURL: "wss://server:8081/ws/runner", Token: "shhh", RunnerID: "r1", GitToken: "ghp_abc",
		HeartbeatInterval: time.Second, ConnectTimeout: time.Second,
		BackoffBase: time.Second, BackoffMax: 30 * time.Second,
	}
	require.NoError(t, valid.Validate())

	tests := []struct {
		name   string
		mutate func(*RunnerConfig)
	}{
		{name: "missing server url", mutate: func(c *RunnerConfig) { c.ServerURL = "" }},
		{name: "server url not ws", mutate: func(c *RunnerConfig) { c.ServerURL = "https://server" }},
		{name: "server url unparsable", mutate: func(c *RunnerConfig) { c.ServerURL = "://bad" }},
		{name: "missing token", mutate: func(c *RunnerConfig) { c.Token = "" }},
		{name: "missing runner id", mutate: func(c *RunnerConfig) { c.RunnerID = "" }},
		{name: "non-positive heartbeat", mutate: func(c *RunnerConfig) { c.HeartbeatInterval = 0 }},
		{name: "non-positive connect timeout", mutate: func(c *RunnerConfig) { c.ConnectTimeout = -1 }},
		{name: "zero backoff base", mutate: func(c *RunnerConfig) { c.BackoffBase = 0 }},
		{name: "backoff max below base", mutate: func(c *RunnerConfig) { c.BackoffMax = time.Millisecond }},
	}
	t.Run("empty git token is allowed so a fresh install's runner starts", func(t *testing.T) {
		c := valid
		c.GitToken = ""
		require.NoError(t, c.Validate())
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid
			tt.mutate(&c)
			require.Error(t, c.Validate())
		})
	}
}
