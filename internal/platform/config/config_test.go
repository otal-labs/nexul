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

	t.Run("the 1Password token is read, then gone from the environment", func(t *testing.T) {
		t.Setenv("NEXUL_DB_PATH", filepath.Join(t.TempDir(), "d.db"))
		t.Setenv("NEXUL_AUTH_SECRET", "s")
		t.Setenv("OP_SERVICE_ACCOUNT_TOKEN", "ops_acme-token")
		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "ops_acme-token", cfg.OnePasswordToken)
		_, inherited := os.LookupEnv("OP_SERVICE_ACCOUNT_TOKEN")
		assert.False(t, inherited, "a child process the server starts must not inherit the token")
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

func TestLoad_AuthSecretThatIsASecretReference_IsRefusedAtStart(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"a reference", "op://vault/item/field", true},
		{"a malformed reference", "op://vault/item", true},
		{"a URL that is not a manager's scheme is a literal", "postgres://user:pw@host/db", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NEXUL_DB_PATH", filepath.Join(t.TempDir(), "d.db"))
			t.Setenv("NEXUL_AUTH_SECRET", tt.secret)
			_, err := Load()
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, "must stay a local value")
			assert.ErrorContains(t, err, "ADR 0183")
			assert.NotContains(t, err.Error(), tt.secret, "the value never appears in the message")
		})
	}
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

func TestLoad_Local(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    bool
		wantErr bool
	}{
		{"not a boolean fails", "yes", false, true},
		{"unset means a server install", "", false, false},
		{"1 marks a desktop install", "1", true, false},
		{"true marks a desktop install", "true", true, false},
		{"0 is a server install", "0", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NEXUL_DB_PATH", filepath.Join(t.TempDir(), "d.db"))
			t.Setenv("NEXUL_AUTH_SECRET", "s")
			t.Setenv("NEXUL_LOCAL", tt.raw)
			cfg, err := Load()
			if tt.wantErr {
				require.ErrorContains(t, err, "NEXUL_LOCAL")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.Local)
		})
	}
}

func TestLoad_LogsLogin(t *testing.T) {
	t.Setenv("NEXUL_DB_PATH", filepath.Join(t.TempDir(), "d.db"))
	t.Setenv("NEXUL_LOGS_EMAIL", "nexul@example.com")
	t.Setenv("NEXUL_LOGS_PASSWORD", "pw")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "nexul@example.com", cfg.LogsEmail)
	assert.Equal(t, "pw", cfg.LogsPassword)
}
