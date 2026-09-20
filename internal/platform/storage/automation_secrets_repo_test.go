package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomationSecretsRepo_SetAndAll(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	require.NoError(t, s.AutomationSecrets.Set(context.Background(), "API_KEY", "sk-123", now))

	all, err := s.AutomationSecrets.All(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "sk-123", all["API_KEY"])
}

func TestAutomationSecretsRepo_ValueIsEncryptedAtRest(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.AutomationSecrets.Set(context.Background(), "API_KEY", "sk-123-plaintext-marker", time.Now().UTC()))

	var stored string
	require.NoError(t, s.db.QueryRow(`SELECT value FROM automation_secrets WHERE name = ?`, "API_KEY").Scan(&stored))
	assert.NotContains(t, stored, "sk-123-plaintext-marker", "the raw column must hold ciphertext, never the plaintext value")
}

func TestAutomationSecretsRepo_SetReplacesWholesalePreservingCreatedAt(t *testing.T) {
	s := newTestStore(t)
	created := time.Unix(1000, 0).UTC()
	require.NoError(t, s.AutomationSecrets.Set(context.Background(), "API_KEY", "old", created))
	updated := time.Unix(2000, 0).UTC()
	require.NoError(t, s.AutomationSecrets.Set(context.Background(), "API_KEY", "new", updated))

	all, err := s.AutomationSecrets.All(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "new", all["API_KEY"])

	list, err := s.AutomationSecrets.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.True(t, list[0].CreatedAt.Equal(created), "created_at survives a replace")
	assert.True(t, list[0].UpdatedAt.Equal(updated))
}

func TestAutomationSecretsRepo_List_NeverExposesValue(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.AutomationSecrets.Set(context.Background(), "API_KEY", "sk-123", time.Now().UTC()))

	list, err := s.AutomationSecrets.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "API_KEY", list[0].Name)
	// SecretMeta has no value field to leak — this is a shape guarantee,
	// not a runtime check.
}

func TestAutomationSecretsRepo_Delete(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.AutomationSecrets.Set(context.Background(), "API_KEY", "sk-123", time.Now().UTC()))
	require.NoError(t, s.AutomationSecrets.Delete(context.Background(), "API_KEY"))

	all, err := s.AutomationSecrets.All(context.Background())
	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestAutomationSecretsRepo_DeleteUnknownName_NoOp(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.AutomationSecrets.Delete(context.Background(), "NEVER_SET"))
}
