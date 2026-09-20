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
