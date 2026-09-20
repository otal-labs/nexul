package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSettings_Configured proves Configured() is false until both the GitHub OAuth App client ID and secret are set (T1; consumed by the bootstrap flow, T2/T3, to decide whether to fall back to env vars).
func TestSettings_Configured(t *testing.T) {
	t.Parallel()

	assert.False(t, Settings{}.Configured(), "unset")
	assert.False(t, Settings{GitHubOAuthClientID: "id-only"}.Configured(), "client id alone")
	assert.False(t, Settings{GitHubOAuthClientSecret: "secret-only"}.Configured(), "secret alone")
	assert.True(t, Settings{GitHubOAuthClientID: "id", GitHubOAuthClientSecret: "secret"}.Configured())
}

// TestSettingsStore_SetGitHubOAuth_RoundTrips proves the SettingsStore contract: SetGitHubOAuth persists the client ID/secret and Get returns them back unchanged (the fake here is plaintext in-memory; the real encrypted round-trip through SQLite is covered by internal/platform/storage.TestSettingsRepo_SetGitHubOAuth_RoundTrip, which also asserts the secret is never stored as plaintext).
func TestSettingsStore_SetGitHubOAuth_RoundTrips(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := newFakeSettings()

	before, err := store.Get(ctx)
	require.NoError(t, err)
	assert.False(t, before.Configured())

	_, err = store.SetGitHubOAuth(ctx, "client-id-123", "client-secret-abc")
	require.NoError(t, err)

	after, err := store.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "client-id-123", after.GitHubOAuthClientID)
	assert.Equal(t, "client-secret-abc", after.GitHubOAuthClientSecret)
	assert.True(t, after.Configured())
}
