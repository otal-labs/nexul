package connectors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// memAppConfigStore is a minimal in-memory AppConfigStore for tests.
type memAppConfigStore struct {
	rows map[string]AppConfig
}

func newMemAppConfigStore() *memAppConfigStore {
	return &memAppConfigStore{rows: map[string]AppConfig{}}
}

func (m *memAppConfigStore) GetAppConfig(_ context.Context, connectorID string) (AppConfig, error) {
	c, ok := m.rows[connectorID]
	if !ok {
		return AppConfig{ConnectorID: connectorID}, nil
	}
	return c, nil
}

func (m *memAppConfigStore) SetAppConfig(_ context.Context, c AppConfig) error {
	m.rows[c.ConnectorID] = c
	return nil
}

// fakeOwnerGate is a canned OwnerGate for tests: userID "owner" holds the
// instance-admin bit, everyone else doesn't.
type fakeOwnerGate struct {
	ownerIDs map[string]bool
	err      error
}

func (f *fakeOwnerGate) CanCreateWorkspace(_ context.Context, userID string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.ownerIDs[userID], nil
}

func testAppConfigService(t *testing.T, owner OwnerGate) (*Service, *memAppConfigStore) {
	t.Helper()
	appStore := newMemAppConfigStore()
	svc := NewService(Config{
		Store:          newMemStore(),
		AppConfigStore: appStore,
		Owner:          owner,
		Registry: []Connector{
			{ID: "github", Name: "GitHub", Description: "d", Category: "development"},
			{ID: "google", Name: "Google", Description: "d", Category: "productivity"},
		},
	})
	return svc, appStore
}

// fakeAppVerifier is an OAuth client whose live app check answers with err.
type fakeAppVerifier struct {
	fakeOAuthClient
	err error
}

func (f *fakeAppVerifier) VerifyApp(context.Context, AppConfig) error { return f.err }

// A rejected live check stores nothing and surfaces the provider's reason.
func TestSetAppConfig_VerifierRejects_NothingStored(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}
	appStore := newMemAppConfigStore()
	svc := NewService(Config{
		Store:          newMemStore(),
		AppConfigStore: appStore,
		Owner:          owner,
		Registry: []Connector{
			{ID: "github", Name: "GitHub", Description: "d", Category: "development", OAuth: &fakeAppVerifier{err: fmt.Errorf("%w: GitHub rejected the client secret", apperrs.ErrInvalid)}},
		},
	})

	_, err := svc.SetAppConfig(context.Background(), "owner-1", "github", "Iv1.abc", "wrong", "", "my-app")
	if err == nil || !errors.Is(err, apperrs.ErrInvalid) || !strings.Contains(err.Error(), "client secret") {
		t.Fatalf("SetAppConfig err = %v, want the verifier's ErrInvalid", err)
	}
	if got, _ := appStore.GetAppConfig(context.Background(), "github"); got.Configured() {
		t.Fatalf("rejected app config must not be stored, got %+v", got)
	}
}

// TestSetAppConfig_OwnerOnly_NonOwnerRejected proves a non-owner caller is
// rejected with ErrForbidden and never reaches storage.
func TestSetAppConfig_OwnerOnly_NonOwnerRejected(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}
	svc, store := testAppConfigService(t, owner)

	_, err := svc.SetAppConfig(context.Background(), "not-the-owner", "github", "client-id", "client-secret", "", "")
	if !errors.Is(err, apperrs.ErrForbidden) {
		t.Fatalf("SetAppConfig (non-owner) = %v, want ErrForbidden", err)
	}
	if got, _ := store.GetAppConfig(context.Background(), "github"); got.Configured() {
		t.Fatalf("non-owner call must not reach storage, got %+v", got)
	}
}

// TestSetAppConfig_OwnerOnly_EmptyUserRejected proves an unauthenticated
// caller (no user id in context) is rejected, mirroring oauthStart's own
// empty-userID check.
func TestSetAppConfig_OwnerOnly_EmptyUserRejected(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}
	svc, _ := testAppConfigService(t, owner)

	_, err := svc.SetAppConfig(context.Background(), "", "github", "client-id", "client-secret", "", "")
	if !errors.Is(err, apperrs.ErrUnauthorized) {
		t.Fatalf("SetAppConfig (no user) = %v, want ErrUnauthorized", err)
	}
}

// TestSetAppConfig_Owner_RoundTrips proves the owner path stores and
// retrieves the app config, including the optional base_url and
// app_slug (T13b, needed since GitHub's install+authorize flow can't derive
// the App's slug from client_id).
func TestSetAppConfig_Owner_RoundTrips(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}
	svc, store := testAppConfigService(t, owner)

	st, err := svc.SetAppConfig(context.Background(), "owner-1", "github", "Iv1.abc", "shh-secret", "https://ghe.example.com", "nexul-test-app")
	if err != nil {
		t.Fatalf("SetAppConfig: %v", err)
	}
	if !st.Configured || st.ClientID != "Iv1.abc" || st.BaseURL != "https://ghe.example.com" || st.AppSlug != "nexul-test-app" {
		t.Fatalf("SetAppConfig status = %+v, want configured with client id/base url/app slug, no secret", st)
	}

	got, err := store.GetAppConfig(context.Background(), "github")
	if err != nil {
		t.Fatalf("GetAppConfig: %v", err)
	}
	if got.ClientSecret != "shh-secret" {
		t.Fatalf("stored client secret = %q, want shh-secret", got.ClientSecret)
	}
	if got.AppSlug != "nexul-test-app" {
		t.Fatalf("stored app slug = %q, want nexul-test-app", got.AppSlug)
	}
}

// TestSetAppConfig_UnconfiguredConnector_CleanNotConfiguredState proves
// getting an unset connector's config returns a clean not-configured state
// rather than an error.
func TestSetAppConfig_UnconfiguredConnector_CleanNotConfiguredState(t *testing.T) {
	_, store := testAppConfigService(t, &fakeOwnerGate{})

	got, err := store.GetAppConfig(context.Background(), "github")
	if err != nil {
		t.Fatalf("GetAppConfig(never-set) = %v, want nil error", err)
	}
	if got.Configured() {
		t.Fatalf("GetAppConfig(never-set).Configured() = true, want false")
	}
}

// TestSetAppConfig_MissingClientSecret_Rejected proves an empty client
// secret (or client id) is rejected as invalid input, not silently stored.
func TestSetAppConfig_MissingClientSecret_Rejected(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}
	svc, _ := testAppConfigService(t, owner)

	if _, err := svc.SetAppConfig(context.Background(), "owner-1", "github", "Iv1.abc", "", "", ""); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("SetAppConfig (empty secret) = %v, want ErrInvalid", err)
	}
	if _, err := svc.SetAppConfig(context.Background(), "owner-1", "github", "", "secret", "", ""); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("SetAppConfig (empty client id) = %v, want ErrInvalid", err)
	}
}

// TestSetAppConfig_UnknownConnector_Rejected proves an unknown connector id
// is rejected rather than silently persisted.
func TestSetAppConfig_UnknownConnector_Rejected(t *testing.T) {
	owner := &fakeOwnerGate{ownerIDs: map[string]bool{"owner-1": true}}
	svc, _ := testAppConfigService(t, owner)

	if _, err := svc.SetAppConfig(context.Background(), "owner-1", "nope", "id", "secret", "", ""); !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("SetAppConfig (unknown connector) = %v, want ErrNotFound", err)
	}
}
