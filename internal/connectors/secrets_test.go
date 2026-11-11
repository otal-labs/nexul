package connectors

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/secretref"
	"github.com/otal-labs/nexul/internal/platform/secretref/secretreftest"
)

const secretRef = "op://Acme/lk/secret"

// secretsOver opens one manager from open and returns the cache the service resolves through.
func secretsOver(t *testing.T, open func(context.Context) (secretref.Resolver, error)) *secretref.Cache {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	managers := secretref.NewManagers(slog.New(slog.NewTextHandler(io.Discard, nil)), secretref.Setup{Scheme: "op", Open: open})
	managers.SetRetry(time.Hour, time.Hour)
	var wg sync.WaitGroup
	managers.Start(ctx, &wg)
	t.Cleanup(func() { cancel(); wg.Wait() })
	return secretref.NewCache(managers, nil)
}

func serving(res secretref.Resolver) func(context.Context) (secretref.Resolver, error) {
	return func(context.Context) (secretref.Resolver, error) { return res, nil }
}

func unreachable(context.Context) (secretref.Resolver, error) { return nil, secretref.ErrUnreachable }

func livekitFields(secret string) map[string]string {
	return map[string]string{"ws_url": "wss://lk.example.com", "api_key": "key1", "api_secret": secret}
}

func serviceWithSecrets(t *testing.T, verifier Verifier, secrets SecretValues) (*Service, *memStore) {
	t.Helper()
	store := newMemStore()
	svc := NewService(Config{
		Store: store,
		Registry: []Connector{
			{ID: "livekit", Name: "LiveKit", Manual: []CredentialField{
				{Key: "ws_url", Label: "WebSocket URL"},
				{Key: "api_key", Label: "API key"},
				{Key: "api_secret", Label: "API secret", Secret: true},
			}, Verify: verifier},
			{ID: "cloudflare", Name: "Cloudflare", Manual: []CredentialField{{Key: "api_token", Label: "API token", Secret: true}},
				Verify: &fakeVerifier{}, OAuth: &fakeOAuthClient{}},
		},
		Secrets: secrets,
	})
	return svc, store
}

func TestSaveManualCredentials_VerifiesTheResolvedSecretAndStoresTheReference(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{secretRef: "real-secret"}}
	verifier := &fakeVerifier{}
	svc, store := serviceWithSecrets(t, verifier, secretsOver(t, serving(fake)))

	_, err := svc.SaveManualCredentials(t.Context(), "livekit", "u1", livekitFields(secretRef))

	require.NoError(t, err)
	assert.Equal(t, "real-secret", verifier.gotFields["api_secret"], "the live check sees the value, not the reference")
	assert.Equal(t, secretRef, store.rows["livekit"].ManualFields["api_secret"], "the database keeps the reference")
}

func TestSaveManualCredentials_RefusedWhileTheManagerIsUnreachable_StoringNothing(t *testing.T) {
	verifier := &fakeVerifier{}
	svc, store := serviceWithSecrets(t, verifier, secretsOver(t, unreachable))

	_, err := svc.SaveManualCredentials(t.Context(), "livekit", "u1", livekitFields(secretRef))

	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "1Password unreachable")
	assert.Zero(t, verifier.calls)
	assert.Empty(t, store.rows)
}

func TestAccessTokenAndManualCredentials_ReturnTheResolvedValueAndLeaveTheStoredReference(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{secretRef: "real-secret", "op://Acme/cf/token": "cf-token"}}
	svc, store := serviceWithSecrets(t, &fakeVerifier{}, secretsOver(t, serving(fake)))
	store.rows["livekit"] = Credentials{ConnectorID: "livekit", ManualFields: livekitFields(secretRef)}
	store.rows["cloudflare"] = Credentials{ConnectorID: "cloudflare", ManualFields: map[string]string{"api_token": "op://Acme/cf/token"}}

	fields, err := svc.ManualCredentials(t.Context(), "livekit")
	require.NoError(t, err)
	token, err := svc.AccessToken(t.Context(), "cloudflare")
	require.NoError(t, err)

	assert.Equal(t, "real-secret", fields["api_secret"])
	assert.Equal(t, "cf-token", token)
	assert.Equal(t, secretRef, store.rows["livekit"].ManualFields["api_secret"])
}

func TestList_AnUnreachableManagerMarksOnlyTheConnectorHoldingTheReference(t *testing.T) {
	svc, store := serviceWithSecrets(t, &fakeVerifier{}, secretsOver(t, unreachable))
	store.rows["livekit"] = Credentials{ConnectorID: "livekit", ManualFields: livekitFields(secretRef)}
	store.rows["cloudflare"] = Credentials{ConnectorID: "cloudflare", ManualFields: map[string]string{"api_token": "literal-token"}}

	list, err := svc.List(t.Context())
	require.NoError(t, err)

	byID := map[string]ConnectorStatus{}
	for _, c := range list {
		byID[c.Connector.ID] = c
	}
	assert.Equal(t, &SecretProblem{Manager: "op", State: secretref.ManagerUnreachable}, byID["livekit"].Status.Secret)
	assert.Nil(t, byID["cloudflare"].Status.Secret)
	token, err := svc.AccessToken(t.Context(), "cloudflare")
	require.NoError(t, err)
	assert.Equal(t, "literal-token", token, "the other connector keeps working")
}

func TestResolvedAppConfigs_ResolveForTheConsumerAndNeverTouchTheStoredRow(t *testing.T) {
	fake := &secretreftest.Fake{Values: map[string]string{"op://Acme/gh/secret": "client-secret", "op://Acme/gh/key": "pem"}}
	raw := newMemAppConfigStore()
	require.NoError(t, raw.SetAppConfig(t.Context(), AppConfig{ConnectorID: "github", ClientID: "id", ClientSecret: "op://Acme/gh/secret"}))
	require.NoError(t, raw.SetPrivateKey(t.Context(), "github", "op://Acme/gh/key"))
	resolved := ResolvedAppConfigs{AppConfigStore: raw, Secrets: secretsOver(t, serving(fake))}

	cfg, err := resolved.GetAppConfig(t.Context(), "github")
	require.NoError(t, err)
	stored, err := raw.GetAppConfig(t.Context(), "github")
	require.NoError(t, err)

	assert.Equal(t, "client-secret", cfg.ClientSecret)
	assert.Equal(t, "pem", cfg.PrivateKey)
	assert.Equal(t, "op://Acme/gh/secret", stored.ClientSecret)
	assert.Equal(t, "op://Acme/gh/key", stored.PrivateKey)
}

func TestResolvedAppConfigs_AnUnresolvableSecretFailsTheReadInsteadOfHandingBackTheReference(t *testing.T) {
	raw := newMemAppConfigStore()
	require.NoError(t, raw.SetAppConfig(t.Context(), AppConfig{ConnectorID: "github", ClientID: "id", ClientSecret: "op://Acme/gh/secret"}))
	resolved := ResolvedAppConfigs{AppConfigStore: raw, Secrets: secretsOver(t, unreachable)}

	_, err := resolved.GetAppConfig(t.Context(), "github")

	require.ErrorIs(t, err, secretref.ErrUnreachable)
}
