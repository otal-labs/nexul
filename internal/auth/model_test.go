package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSettings_Configured proves Configured() is false until both the GitHub OAuth App client ID and secret are set (T1; consumed by the bootstrap flow, T2/T3, to decide whether to fall back to env vars).
func TestSettings_Configured(t *testing.T) {
	t.Parallel()

	assert.False(t, Settings{}.Configured(), "unset")
	assert.False(t, Settings{GitHubOAuthClientID: "id-only"}.Configured(), "client id alone")
	assert.False(t, Settings{GitHubOAuthClientSecret: "secret-only"}.Configured(), "secret alone")
	assert.True(t, Settings{GitHubOAuthClientID: "id", GitHubOAuthClientSecret: "secret"}.Configured())
}

func TestPublicUsername(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		provider Provider
		username string
		want     string
	}{
		{"GitHub is its login", ProviderGitHub, "", "lena"},
		{"Discord is its username, never the email login", ProviderDiscord, "lena_d", "lena_d"},
		{"Discord without a stored username is its name", ProviderDiscord, "", "Lena"},
		{"Google has none", ProviderGoogle, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PublicUsername(tt.provider, "lena", "Lena", tt.username))
		})
	}
}
