package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
)

func openConnectorAppConfigStore(t *testing.T) *Store {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return New(db, testEncKey)
}

// TestConnectorAppConfigRepo_GetAppConfig_NeverSet proves a connector whose
// app config has never been set returns a clean not-configured state (zero
// AppConfig, Configured() == false), not an error — this is a settings knob,
// not a connection with a missing-vs-present error distinction.
func TestConnectorAppConfigRepo_GetAppConfig_NeverSet(t *testing.T) {
	t.Parallel()
	s := openConnectorAppConfigStore(t)
	ctx := context.Background()

	got, err := s.ConnectorAppConfig.GetAppConfig(ctx, "github")
	require.NoError(t, err)
	assert.Equal(t, "github", got.ConnectorID)
	assert.False(t, got.Configured())
	assert.Equal(t, "", got.ClientID)
	assert.Equal(t, "", got.ClientSecret)
	assert.Equal(t, "", got.BaseURL)
}

// TestConnectorAppConfigRepo_SetGet_RoundTrip proves the app config
// round-trips through storage and that the client secret is genuinely
// encrypted at rest (raw-column ciphertext assertion, same convention as
// auth_repo_test.go's TestSettingsRepo_SetGitHubOAuth_RoundTrip).
func TestConnectorAppConfigRepo_SetGet_RoundTrip(t *testing.T) {
	t.Parallel()
	s := openConnectorAppConfigStore(t)
	ctx := context.Background()

	const clientID = "Iv1.app-client-id"
	const clientSecret = "super-secret-app-value"

	cfg := connectors.AppConfig{
		ConnectorID:  "github",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      "",
		AppSlug:      "nexul-test-app",
	}
	require.NoError(t, s.ConnectorAppConfig.SetAppConfig(ctx, cfg))

	got, err := s.ConnectorAppConfig.GetAppConfig(ctx, "github")
	require.NoError(t, err)
	assert.Equal(t, clientID, got.ClientID)
	assert.Equal(t, clientSecret, got.ClientSecret)
	assert.Equal(t, "", got.BaseURL)
	assert.Equal(t, "nexul-test-app", got.AppSlug)
	assert.True(t, got.Configured())

	// Verify the secret is genuinely encrypted at rest: the raw column must
	// not contain the plaintext.
	var rawSecret string
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT client_secret FROM connector_app_config WHERE connector_id = ?`, "github").Scan(&rawSecret))
	assert.NotEqual(t, clientSecret, rawSecret, "client secret must not be stored as plaintext")
	assert.NotEmpty(t, rawSecret)

	// Rotating the secret (self-hosted base_url set this time) overwrites
	// rather than erroring or leaving a second row.
	cfg2 := connectors.AppConfig{
		ConnectorID:  "github",
		ClientID:     "Iv1.new-client-id",
		ClientSecret: "rotated-secret-value",
		BaseURL:      "https://ghe.example.com",
		AppSlug:      "rotated-slug",
	}
	require.NoError(t, s.ConnectorAppConfig.SetAppConfig(ctx, cfg2))

	got2, err := s.ConnectorAppConfig.GetAppConfig(ctx, "github")
	require.NoError(t, err)
	assert.Equal(t, cfg2.ClientID, got2.ClientID)
	assert.Equal(t, cfg2.ClientSecret, got2.ClientSecret)
	assert.Equal(t, cfg2.BaseURL, got2.BaseURL)
	assert.Equal(t, cfg2.AppSlug, got2.AppSlug)

	var count int
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM connector_app_config WHERE connector_id = ?`, "github").Scan(&count))
	assert.Equal(t, 1, count, "rotating must overwrite the single row, not add a second")
}

// TestConnectorAppConfigRepo_SetGet_MultipleConnectorsIndependent proves each
// connector's app config is stored independently (github's rows don't leak
// into gitlab's, or vice versa) — generic to any connector.
func TestConnectorAppConfigRepo_SetGet_MultipleConnectorsIndependent(t *testing.T) {
	t.Parallel()
	s := openConnectorAppConfigStore(t)
	ctx := context.Background()

	require.NoError(t, s.ConnectorAppConfig.SetAppConfig(ctx, connectors.AppConfig{
		ConnectorID: "github", ClientID: "gh-id", ClientSecret: "gh-secret",
	}))

	gitlab, err := s.ConnectorAppConfig.GetAppConfig(ctx, "gitlab")
	require.NoError(t, err)
	assert.False(t, gitlab.Configured())

	github, err := s.ConnectorAppConfig.GetAppConfig(ctx, "github")
	require.NoError(t, err)
	assert.True(t, github.Configured())
}
