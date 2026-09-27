package runner

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"time"
)

// RunnerConfig is the host binary's environment configuration, validated before the client dials.
type RunnerConfig struct {
	// ServerURL is the instance's http(s):// base (NEXUL_SERVER_URL); the WebSocket URL derives from it.
	ServerURL string
	// CredentialFile holds the runner's own credential (NEXUL_CREDENTIAL_FILE), written by `nexul install`.
	CredentialFile string
	Credential     string
	// EnrollCodeFile, when the credential file does not exist yet, holds a code to enroll with on first start
	// (NEXUL_ENROLL_CODE_FILE); only dev stacks set it, since `nexul install` enrolls everywhere else.
	EnrollCodeFile string
	// EnrollWait bounds how long a first start waits for the code file and the server to answer.
	EnrollWait time.Duration
	// Name is the runner's unit name (NEXUL_RUNNER_NAME), used only to uninstall itself once removed.
	Name string
	// StackRoot is where checkouts live when a job names no stack root of its own (NEXUL_STACK_ROOT).
	StackRoot string
	// Ctl is the `nexul` command the runner upgrades and uninstalls itself through (NEXUL_CTL).
	Ctl      string
	LogLevel string
	// GitToken authenticates private repo clones for repo-driven builds; optional so a fresh install's instance runner still starts.
	GitToken          string
	HeartbeatInterval time.Duration
	ConnectTimeout    time.Duration
	BackoffBase       time.Duration
	BackoffMax        time.Duration
}

// LoadRunnerConfig reads the runner env vars and its credential file; a deliberate break from config.Load since
// it's a separate binary.
func LoadRunnerConfig() (*RunnerConfig, error) {
	cfg := &RunnerConfig{
		ServerURL:         os.Getenv("NEXUL_SERVER_URL"),
		CredentialFile:    os.Getenv("NEXUL_CREDENTIAL_FILE"),
		EnrollCodeFile:    os.Getenv("NEXUL_ENROLL_CODE_FILE"),
		EnrollWait:        5 * time.Minute,
		Name:              os.Getenv("NEXUL_RUNNER_NAME"),
		StackRoot:         os.Getenv("NEXUL_STACK_ROOT"),
		Ctl:               envOrDefault("NEXUL_CTL", "nexul"),
		LogLevel:          envOrDefault("NEXUL_LOG_LEVEL", "info"),
		GitToken:          os.Getenv("NEXUL_GIT_TOKEN"),
		HeartbeatInterval: envDuration("NEXUL_RUNNER_HEARTBEAT", 10*time.Second),
		ConnectTimeout:    envDuration("NEXUL_RUNNER_CONNECT_TIMEOUT", 10*time.Second),
		BackoffBase:       envDuration("NEXUL_RUNNER_BACKOFF_BASE", time.Second),
		BackoffMax:        envDuration("NEXUL_RUNNER_BACKOFF_MAX", 30*time.Second),
	}
	if cfg.CredentialFile == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(cfg.CredentialFile)
	if errors.Is(err, fs.ErrNotExist) && cfg.EnrollCodeFile != "" {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read NEXUL_CREDENTIAL_FILE: %w", err)
	}
	cfg.Credential = strings.TrimSpace(string(b))
	return cfg, nil
}

// Validate checks every required field and fail-fast invariants.
func (c *RunnerConfig) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("NEXUL_SERVER_URL is required")
	}
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("NEXUL_SERVER_URL %q: %w", c.ServerURL, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("NEXUL_SERVER_URL must be an http:// or https:// URL, got %q", c.ServerURL)
	}
	if c.Credential == "" {
		return fmt.Errorf("NEXUL_CREDENTIAL_FILE is required and must hold the runner's credential")
	}
	if c.Name == "" {
		return fmt.Errorf("NEXUL_RUNNER_NAME is required")
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

// WSURL is the runner endpoint under ServerURL: ws(s)://<host>[<path>]/ws/runner. Call after Validate.
func (c *RunnerConfig) WSURL() string {
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return ""
	}
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	u.Path = strings.TrimRight(u.Path, "/") + "/ws/runner"
	u.RawQuery, u.Fragment = "", ""
	return u.String()
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
