package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	// OTLPEndpoint is the OTLP/HTTP base URL logs are shipped to (e.g. http://openobserve:5080/api/default); empty keeps logs on stderr only.
	OTLPEndpoint string
	OTLPUser     string
	OTLPToken    string
	// LogsURL is the local log store's base URL (NEXUL_LOGS_URL); when set the server proxies /openobserve/ to it.
	LogsURL *url.URL
	// Local marks a desktop install (NEXUL_LOCAL), where first run stays on http://localhost instead of a domain.
	Local bool
	// SiteURL (NEXUL_SITE_URL) serves the install scripts a computer's command fetches; empty is nexul.io. For testing
	// an instance against its own build.
	SiteURL string
	// ReleaseURL (NEXUL_RELEASE_URL) is where those scripts download the release from; empty is GitHub's.
	ReleaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		DBPath:     envOrDefault("NEXUL_DB_PATH", "./data/nexul.db"),
		HTTPAddr:   envOrDefault("NEXUL_HTTP_ADDR", ":8080"),
		LogLevel:   envOrDefault("NEXUL_LOG_LEVEL", "info"),
		AuthSecret: os.Getenv("NEXUL_AUTH_SECRET"),
		SPAOrigin:  envOrDefault("NEXUL_SPA_ORIGIN", ""),
		DevLogin:   os.Getenv("NEXUL_DEV_LOGIN") == "true",

		OTLPEndpoint: os.Getenv("NEXUL_OTLP_ENDPOINT"),
		OTLPUser:     os.Getenv("NEXUL_OTLP_USER"),
		OTLPToken:    os.Getenv("NEXUL_OTLP_TOKEN"),
		SiteURL:      os.Getenv("NEXUL_SITE_URL"),
		ReleaseURL:   os.Getenv("NEXUL_RELEASE_URL"),
	}
	logsURL, err := parseLogsURL(os.Getenv("NEXUL_LOGS_URL"))
	if err != nil {
		return nil, err
	}
	cfg.LogsURL = logsURL
	if cfg.Local, err = parseOptionalBool("NEXUL_LOCAL"); err != nil {
		return nil, err
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

func parseLogsURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("NEXUL_LOGS_URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("NEXUL_LOGS_URL %q: want an http(s)://host:port URL", raw)
	}
	return u, nil
}

func parseOptionalBool(key string) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return false, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s %q: want 1, true, 0 or false", key, raw)
	}
	return v, nil
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
