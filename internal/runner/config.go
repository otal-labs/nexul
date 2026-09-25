package runner

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// RunnerConfig is the host binary's environment configuration, validated before the client dials.
type RunnerConfig struct {
	ServerURL string
	Token     string
	// SecretFile is where the server publishes the runner secret (NEXUL_RUNNER_SECRET_FILE); the instance runner
	// reads it off the shared data volume instead of carrying a copy in its env.
	SecretFile string
	RunnerID   string
	Name       string
	// Machine is the machine this runner belongs to (NEXUL_MACHINE, defaults to the host's hostname); runners
	// reporting the same machine form a dispatch pool (issue 05).
	Machine string
	// StackRoot is where this runner keeps checkouts (NEXUL_STACK_ROOT), reported so a new machine starts with it.
	StackRoot string
	LogLevel  string
	// GitToken authenticates private repo clones for repo-driven builds; optional so a fresh install's instance runner still starts.
	GitToken          string
	HeartbeatInterval time.Duration
	ConnectTimeout    time.Duration
	BackoffBase       time.Duration
	BackoffMax        time.Duration
}

// LoadRunnerConfig reads the runner env vars; a deliberate break from config.Load since it's a separate binary.
func LoadRunnerConfig() (*RunnerConfig, error) {
	cfg := &RunnerConfig{
		ServerURL:         os.Getenv("NEXUL_SERVER_WS"),
		Token:             os.Getenv("NEXUL_RUNNER_SECRET"),
		SecretFile:        os.Getenv("NEXUL_RUNNER_SECRET_FILE"),
		RunnerID:          os.Getenv("NEXUL_RUNNER_ID"),
		Name:              os.Getenv("NEXUL_RUNNER_NAME"),
		Machine:           os.Getenv("NEXUL_MACHINE"),
		StackRoot:         os.Getenv("NEXUL_STACK_ROOT"),
		LogLevel:          envOrDefault("NEXUL_LOG_LEVEL", "info"),
		GitToken:          os.Getenv("NEXUL_GIT_TOKEN"),
		HeartbeatInterval: envDuration("NEXUL_RUNNER_HEARTBEAT", 10*time.Second),
		ConnectTimeout:    envDuration("NEXUL_RUNNER_CONNECT_TIMEOUT", 10*time.Second),
		BackoffBase:       envDuration("NEXUL_RUNNER_BACKOFF_BASE", time.Second),
		BackoffMax:        envDuration("NEXUL_RUNNER_BACKOFF_MAX", 30*time.Second),
	}
	if cfg.RunnerID == "" {
		host, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("NEXUL_RUNNER_ID unset and hostname unavailable: %w", err)
		}
		cfg.RunnerID = host
	}
	if cfg.Machine == "" {
		if host, err := os.Hostname(); err == nil {
			cfg.Machine = host
		}
	}
	return cfg, nil
}

// Validate checks every required field and fail-fast invariants.
func (c *RunnerConfig) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("NEXUL_SERVER_WS is required")
	}
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("NEXUL_SERVER_WS %q: %w", c.ServerURL, err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("NEXUL_SERVER_WS must be ws:// or wss://, got %q", u.Scheme)
	}
	if c.Token == "" {
		return fmt.Errorf("NEXUL_RUNNER_SECRET or NEXUL_RUNNER_SECRET_FILE is required")
	}
	if c.RunnerID == "" {
		return fmt.Errorf("runner id is required")
	}
	if c.HeartbeatInterval <= 0 {
		return fmt.Errorf("NEXUL_RUNNER_HEARTBEAT must be positive")
	}
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("NEXUL_RUNNER_CONNECT_TIMEOUT must be positive")
	}
	if c.BackoffBase <= 0 || c.BackoffMax < c.BackoffBase {
		return fmt.Errorf("NEXUL_RUNNER_BACKOFF_BASE must be positive and <= BACKOFF_MAX")
	}
	return nil
}

// ReadSecretFile polls path until it holds a secret: the instance runner starts alongside the server, which only
// writes the file once it has booted.
func ReadSecretFile(ctx context.Context, path string) (string, error) {
	for {
		b, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("read runner secret %s: %w", path, err)
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			return s, nil
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for runner secret %s: %w", path, ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return d
}
