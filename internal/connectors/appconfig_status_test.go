package connectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestSetAppConfig_BlankSecretKeepsTheStoredOne(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}
	svc, store := testAppConfigService(t, owner)
	_, err := svc.SetAppConfig(t.Context(), "owner-1", "github", "Iv1.abc", "shh", "", "old-slug")
	require.NoError(t, err)

	got, err := svc.SetAppConfig(t.Context(), "owner-1", "github", "Iv1.abc", "", "", "new-slug")
	require.NoError(t, err)

	assert.Equal(t, "new-slug", got.AppSlug)
	cfg, err := store.GetAppConfig(t.Context(), "github")
	require.NoError(t, err)
	assert.Equal(t, "shh", cfg.ClientSecret, "the stored secret survives an edit without one")
}

func TestAppConfigStatus(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}

	t.Run("an unknown connector is not found", func(t *testing.T) {
		svc, _ := testAppConfigService(t, owner)
		_, err := svc.AppConfigStatus(t.Context(), "bogus")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("never registered reads as not configured", func(t *testing.T) {
		svc, _ := testAppConfigService(t, owner)
		got, err := svc.AppConfigStatus(t.Context(), "github")
		require.NoError(t, err)
		assert.False(t, got.Configured)
	})
	t.Run("a registered app reads back without its secret", func(t *testing.T) {
		svc, _ := testAppConfigService(t, owner)
		_, err := svc.SetAppConfig(t.Context(), "owner-1", "github", "Iv1.abc", "shh", "https://ghe.example.com", "my-app")
		require.NoError(t, err)
		got, err := svc.AppConfigStatus(t.Context(), "github")
		require.NoError(t, err)
		assert.Equal(t, AppConfigStatus{Configured: true, ClientID: "Iv1.abc", BaseURL: "https://ghe.example.com", AppSlug: "my-app"}, got)
	})
	t.Run("without a store nothing is configured", func(t *testing.T) {
		svc := NewService(Config{Store: newMemStore(), Owner: owner, Registry: []Connector{{ID: "github", Name: "GitHub"}}})
		got, err := svc.AppConfigStatus(t.Context(), "github")
		require.NoError(t, err)
		assert.False(t, got.Configured)
	})
}
