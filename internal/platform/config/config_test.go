package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_AuthSecret(t *testing.T) {
	t.Run("env var wins and nothing is written", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("NEXUL_DB_PATH", filepath.Join(dir, "d.db"))
		t.Setenv("NEXUL_AUTH_SECRET", "from-env")
		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "from-env", cfg.AuthSecret)
		assert.NoFileExists(t, filepath.Join(dir, "auth-secret"))
	})

	t.Run("generated once next to the db and reused on the next load", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("NEXUL_DB_PATH", filepath.Join(dir, "data", "d.db"))
		t.Setenv("NEXUL_AUTH_SECRET", "")
		first, err := Load()
		require.NoError(t, err)
		assert.Len(t, first.AuthSecret, 64)

		b, err := os.ReadFile(filepath.Join(dir, "data", "auth-secret"))
		require.NoError(t, err)
		assert.Equal(t, first.AuthSecret+"\n", string(b))

		second, err := Load()
		require.NoError(t, err)
		assert.Equal(t, first.AuthSecret, second.AuthSecret)
	})
}

func TestLoad_LogsURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"unset means no proxy", "", "", false},
		{"http with port", "http://127.0.0.1:5080", "http://127.0.0.1:5080", false},
		{"https", "https://logs.internal", "https://logs.internal", false},
		{"no scheme", "127.0.0.1:5080", "", true},
		{"unsupported scheme", "ftp://127.0.0.1", "", true},
		{"no host", "http://", "", true},
		{"unparseable", "http://[::1", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NEXUL_DB_PATH", filepath.Join(t.TempDir(), "d.db"))
			t.Setenv("NEXUL_AUTH_SECRET", "s")
			t.Setenv("NEXUL_LOGS_URL", tt.raw)
			cfg, err := Load()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want == "" {
				assert.Nil(t, cfg.LogsURL)
				return
			}
			require.NotNil(t, cfg.LogsURL)
			assert.Equal(t, tt.want, cfg.LogsURL.String())
		})
	}
}
