package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRunnerConfig_UnreadableCredentialFile_Fails(t *testing.T) {
	t.Setenv("NEXUL_CREDENTIAL_FILE", filepath.Join(t.TempDir(), "missing"))
	_, err := LoadRunnerConfig()
	require.ErrorContains(t, err, "NEXUL_CREDENTIAL_FILE")
}

func TestLoadRunnerConfig(t *testing.T) {
	credFile := filepath.Join(t.TempDir(), "credential")
	require.NoError(t, os.WriteFile(credFile, []byte("nxr_abc\n"), 0o600))
	t.Setenv("NEXUL_SERVER_URL", "https://nexul.example.com")
	t.Setenv("NEXUL_CREDENTIAL_FILE", credFile)
	t.Setenv("NEXUL_RUNNER_NAME", "alpha")
	t.Setenv("NEXUL_STACK_ROOT", "/srv/stacks")
	t.Setenv("NEXUL_GIT_TOKEN", "ghp_abc")
	t.Setenv("NEXUL_CTL", "/usr/local/bin/nexul")
	t.Setenv("NEXUL_RUNNER_HEARTBEAT", "5s")

	cfg, err := LoadRunnerConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://nexul.example.com", cfg.ServerURL)
	assert.Equal(t, "nxr_abc", cfg.Credential)
	assert.Equal(t, "alpha", cfg.Name)
	assert.Equal(t, "/srv/stacks", cfg.StackRoot)
	assert.Equal(t, "ghp_abc", cfg.GitToken)
	assert.Equal(t, "/usr/local/bin/nexul", cfg.Ctl)
	assert.Equal(t, 5*time.Second, cfg.HeartbeatInterval)
	assert.Equal(t, time.Second, cfg.BackoffBase)
	assert.Equal(t, 30*time.Second, cfg.BackoffMax)
}

func TestLoadRunnerConfig_Defaults(t *testing.T) {
	t.Setenv("NEXUL_CREDENTIAL_FILE", "")
	t.Setenv("NEXUL_CTL", "")
	t.Setenv("NEXUL_RUNNER_HEARTBEAT", "not-a-duration")
	cfg, err := LoadRunnerConfig()
	require.NoError(t, err)
	assert.Equal(t, "nexul", cfg.Ctl)
	assert.Equal(t, 10*time.Second, cfg.HeartbeatInterval)
	assert.Empty(t, cfg.Credential)
}

func TestRunnerConfig_Validate(t *testing.T) {
	valid := RunnerConfig{
		ServerURL: "https://server:8443", Credential: "nxr_abc", Name: "alpha", GitToken: "ghp_abc",
		HeartbeatInterval: time.Second, ConnectTimeout: time.Second,
		BackoffBase: time.Second, BackoffMax: 30 * time.Second,
	}
	require.NoError(t, valid.Validate())

	tests := []struct {
		name   string
		mutate func(*RunnerConfig)
	}{
		{name: "missing server url", mutate: func(c *RunnerConfig) { c.ServerURL = "" }},
		{name: "server url is a websocket url", mutate: func(c *RunnerConfig) { c.ServerURL = "wss://server/ws/runner" }},
		{name: "server url without host", mutate: func(c *RunnerConfig) { c.ServerURL = "https://" }},
		{name: "server url unparsable", mutate: func(c *RunnerConfig) { c.ServerURL = "://bad" }},
		{name: "missing credential", mutate: func(c *RunnerConfig) { c.Credential = "" }},
		{name: "missing name", mutate: func(c *RunnerConfig) { c.Name = "" }},
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

func TestRunnerConfig_WSURL(t *testing.T) {
	tests := []struct{ server, want string }{
		{"https://nexul.example.com", "wss://nexul.example.com/ws/runner"},
		{"http://127.0.0.1:8080/", "ws://127.0.0.1:8080/ws/runner"},
		{"https://example.com/nexul?x=1", "wss://example.com/nexul/ws/runner"},
	}
	for _, tt := range tests {
		c := RunnerConfig{ServerURL: tt.server}
		assert.Equal(t, tt.want, c.WSURL(), tt.server)
	}
}
