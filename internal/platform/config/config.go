package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	DBPath   string
	HTTPAddr string
	LogLevel string
	// AuthSecret signs sessions and derives the at-rest key; NEXUL_AUTH_SECRET, else generated once into <db dir>/auth-secret.
	AuthSecret string
	SPAOrigin  string
	// DevLogin bypasses GitHub OAuth with a fixed local session; must stay unset in production.
	DevLogin bool
	// RunnerSecret seeds the stored runner secret at boot; empty generates one on first runner connection.
	RunnerSecret string
	// OTLPEndpoint is the OTLP/HTTP base URL logs are shipped to (e.g. http://openobserve:5080/api/default); empty keeps logs on stderr only.
	OTLPEndpoint string
	OTLPUser     string
	OTLPToken    string
}

func Load() (*Config, error) {
	cfg := &Config{
		DBPath:     envOrDefault("NEXUL_DB_PATH", "./data/nexul.db"),
		HTTPAddr:   envOrDefault("NEXUL_HTTP_ADDR", ":8080"),
		LogLevel:   envOrDefault("NEXUL_LOG_LEVEL", "info"),
		AuthSecret: os.Getenv("NEXUL_AUTH_SECRET"),
		SPAOrigin:  envOrDefault("NEXUL_SPA_ORIGIN", ""),
		DevLogin:   os.Getenv("NEXUL_DEV_LOGIN") == "true",

		RunnerSecret: os.Getenv("NEXUL_RUNNER_SECRET"),

		OTLPEndpoint: os.Getenv("NEXUL_OTLP_ENDPOINT"),
		OTLPUser:     os.Getenv("NEXUL_OTLP_USER"),
		OTLPToken:    os.Getenv("NEXUL_OTLP_TOKEN"),
	}
	if cfg.AuthSecret == "" {
		secret, err := loadOrCreateSecret(filepath.Join(filepath.Dir(cfg.DBPath), "auth-secret"))
		if err != nil {
			return nil, err
		}
		cfg.AuthSecret = secret
	}
	return cfg, nil
}

// loadOrCreateSecret reads the secret at path, generating and persisting one (0600) when the file is absent or empty.
func loadOrCreateSecret(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read auth secret %s: %w", path, err)
	}
	if s := strings.TrimSpace(string(b)); s != "" {
		return s, nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate auth secret: %w", err)
	}
	secret := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create auth secret dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write auth secret %s: %w", path, err)
	}
	return secret, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
