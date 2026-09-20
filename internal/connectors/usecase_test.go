package connectors

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeOAuthClient is a canned OAuthClient for tests, mirroring the
// fakeProvider pattern internal/dns's own tests use.
type fakeOAuthClient struct {
	configured   bool
	authorizeURL string
	exchangeTS   *TokenSet
	exchangeErr  error
	refreshTS    *TokenSet
	refreshErr   error
	revokeErr    error
	revokedToken string
}

func (f *fakeOAuthClient) Configured() bool { return f.configured }
func (f *fakeOAuthClient) AuthorizeURL(state string) string {
	return f.authorizeURL + "?state=" + state
}
func (f *fakeOAuthClient) Exchange(context.Context, string) (*TokenSet, error) {
	return f.exchangeTS, f.exchangeErr
}
func (f *fakeOAuthClient) Refresh(context.Context, string) (*TokenSet, error) {
	return f.refreshTS, f.refreshErr
}
func (f *fakeOAuthClient) Revoke(_ context.Context, accessToken string) error {
	f.revokedToken = accessToken
	return f.revokeErr
}

// memStore is a minimal in-memory CredentialsStore for tests.
type memStore struct {
	rows map[string]Credentials
}

func newMemStore() *memStore { return &memStore{rows: map[string]Credentials{}} }

func (m *memStore) GetCredentials(_ context.Context, connectorID string) (Credentials, error) {
	c, ok := m.rows[connectorID]
	if !ok {
		return Credentials{}, apperrs.ErrNotFound
	}
	return c, nil
}

func (m *memStore) SaveCredentials(_ context.Context, c Credentials) error {
	m.rows[c.ConnectorID] = c
	return nil
}

func (m *memStore) DeleteCredentials(_ context.Context, connectorID string) error {
	delete(m.rows, connectorID)
	return nil
}

// fakeVerifier is a canned Verifier for manual-credential tests (ticket 10).
type fakeVerifier struct {
	err       error
	gotFields map[string]string
	calls     int
}

func (f *fakeVerifier) Verify(_ context.Context, fields map[string]string) error {
	f.calls++
	f.gotFields = fields
	return f.err
}

func testService(t *testing.T, oauth OAuthClient, now func() time.Time) (*Service, *memStore) {
	t.Helper()
	store := newMemStore()
	svc := NewService(Config{
		Store: store,
		Registry: []Connector{
			{ID: "github", Name: "GitHub", Description: "d", Category: "development", OAuth: oauth},
			{ID: "google", Name: "Google", Description: "d", Category: "productivity", OAuth: nil},
		},
		Now: now,
	})
	return svc, store
}

// testServiceWithVerifier is testService's sibling for ticket 10's
// manual-credential path: a livekit entry with a fake Verifier plus a
// cloudflare entry with no Manual fields, to prove SaveManualCredentials
// rejects non-manual connectors.
func testServiceWithVerifier(t *testing.T, verifier Verifier, now func() time.Time) (*Service, *memStore) {
	t.Helper()
	store := newMemStore()
	svc := NewService(Config{
		Store: store,
		Registry: []Connector{
			{ID: "livekit", Name: "LiveKit", Description: "d", Category: "communication", Manual: []CredentialField{
				{Key: "ws_url", Label: "WebSocket URL"},
				{Key: "api_key", Label: "API key"},
				{Key: "api_secret", Label: "API secret", Secret: true},
			}, Verify: verifier},
			{ID: "cloudflare", Name: "Cloudflare", Description: "d", Category: "infrastructure"},
		},
		Now: now,
	})
	return svc, store
}

func TestSaveManualCredentials_VerifiesAndStores(t *testing.T) {
	verifier := &fakeVerifier{}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	svc, store := testServiceWithVerifier(t, verifier, func() time.Time { return now })

	fields := map[string]string{"ws_url": "wss://lk.example.com", "api_key": "key1", "api_secret": "secret1"}
	status, err := svc.SaveManualCredentials(context.Background(), "livekit", "user-1", fields)
	if err != nil {
		t.Fatalf("SaveManualCredentials: %v", err)
	}
	if !status.Configured || status.ConnectedBy != "user-1" {
		t.Fatalf("status = %+v, want configured by user-1", status)
	}
	if verifier.calls != 1 {
		t.Fatalf("verifier.calls = %d, want 1", verifier.calls)
	}
	if verifier.gotFields["api_secret"] != "secret1" {
		t.Fatalf("verifier saw fields = %+v, want api_secret=secret1", verifier.gotFields)
	}

	cred, err := store.GetCredentials(context.Background(), "livekit")
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if cred.ManualFields["ws_url"] != "wss://lk.example.com" {
		t.Fatalf("stored ManualFields = %+v, want ws_url stored", cred.ManualFields)
	}

	got, err := svc.ManualCredentials(context.Background(), "livekit")
	if err != nil {
		t.Fatalf("ManualCredentials: %v", err)
	}
	if got["api_key"] != "key1" {
		t.Fatalf("ManualCredentials() = %+v, want api_key=key1", got)
	}
}

// TestSaveManualCredentials_VerifyFails_StoresNothing proves a failed
// provider verification surfaces inline (ErrInvalid, mapped to HTTP 400 by
// httpx) and nothing is written to the store.
func TestSaveManualCredentials_VerifyFails_StoresNothing(t *testing.T) {
	verifier := &fakeVerifier{err: errors.New("bad api key")}
	svc, store := testServiceWithVerifier(t, verifier, nil)

	fields := map[string]string{"ws_url": "wss://lk.example.com", "api_key": "key1", "api_secret": "secret1"}
	_, err := svc.SaveManualCredentials(context.Background(), "livekit", "user-1", fields)
	if !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("SaveManualCredentials with failing verify = %v, want ErrInvalid", err)
	}
	if !strings.Contains(err.Error(), "bad api key") {
		t.Fatalf("error = %v, want the provider's own message inline", err)
	}
	if _, err := store.GetCredentials(context.Background(), "livekit"); !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("GetCredentials after failed verify = %v, want ErrNotFound (nothing stored)", err)
	}
}

func TestSaveManualCredentials_MissingField_Rejected(t *testing.T) {
	verifier := &fakeVerifier{}
	svc, _ := testServiceWithVerifier(t, verifier, nil)

	fields := map[string]string{"ws_url": "wss://lk.example.com", "api_key": ""}
	if _, err := svc.SaveManualCredentials(context.Background(), "livekit", "user-1", fields); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("SaveManualCredentials with missing field = %v, want ErrInvalid", err)
	}
	if verifier.calls != 0 {
		t.Fatalf("verifier.calls = %d, want 0 (must reject before calling the provider)", verifier.calls)
	}
}

func TestSaveManualCredentials_NoVerifierWired_Rejected(t *testing.T) {
	svc, _ := testServiceWithVerifier(t, nil, nil)
	fields := map[string]string{"ws_url": "a", "api_key": "b", "api_secret": "c"}
	if _, err := svc.SaveManualCredentials(context.Background(), "livekit", "user-1", fields); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("SaveManualCredentials with nil Verify = %v, want ErrInvalid", err)
	}
}

func TestSaveManualCredentials_NonManualConnector_Rejected(t *testing.T) {
	svc, _ := testServiceWithVerifier(t, &fakeVerifier{}, nil)
	if _, err := svc.SaveManualCredentials(context.Background(), "cloudflare", "user-1", map[string]string{"x": "y"}); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("SaveManualCredentials(cloudflare, no Manual fields) = %v, want ErrInvalid", err)
	}
}

func TestManualCredentials_NeverConfigured_ErrNotFound(t *testing.T) {
	svc, _ := testServiceWithVerifier(t, &fakeVerifier{}, nil)
	if _, err := svc.ManualCredentials(context.Background(), "livekit"); !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("ManualCredentials before save = %v, want ErrNotFound", err)
	}
}

func TestDisconnect_ManualConnector_DeletesRowNoRevokeCall(t *testing.T) {
	verifier := &fakeVerifier{}
	svc, store := testServiceWithVerifier(t, verifier, nil)
	if err := store.SaveCredentials(context.Background(), Credentials{
		ConnectorID:  "livekit",
		ManualFields: map[string]string{"ws_url": "a", "api_key": "b", "api_secret": "c"},
	}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	if err := svc.Disconnect(context.Background(), "livekit"); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if _, err := store.GetCredentials(context.Background(), "livekit"); !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("GetCredentials after disconnect = %v, want ErrNotFound", err)
	}
}

func TestAuthorizeURL_ComingSoon_Rejected(t *testing.T) {
	svc, _ := testService(t, &fakeOAuthClient{configured: true}, nil)
	if _, err := svc.AuthorizeURL(context.Background(), "google", "state"); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("AuthorizeURL(google) = %v, want ErrInvalid", err)
	}
}

func TestAuthorizeURL_UnknownConnector_Rejected(t *testing.T) {
	svc, _ := testService(t, &fakeOAuthClient{configured: true}, nil)
	if _, err := svc.AuthorizeURL(context.Background(), "nope", "state"); !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("AuthorizeURL(nope) = %v, want ErrNotFound", err)
	}
}

func TestAuthorizeURL_NotConfigured_Rejected(t *testing.T) {
	svc, _ := testService(t, &fakeOAuthClient{configured: false}, nil)
	if _, err := svc.AuthorizeURL(context.Background(), "github", "state"); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("AuthorizeURL(unconfigured github) = %v, want ErrInvalid", err)
	}
}

func TestAuthorizeURL_Configured_ReturnsProviderURL(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true, authorizeURL: "https://example.com/authorize"}
	svc, _ := testService(t, oauth, nil)
	url, err := svc.AuthorizeURL(context.Background(), "github", "abc123")
	if err != nil {
		t.Fatalf("AuthorizeURL: %v", err)
	}
	if url != "https://example.com/authorize?state=abc123" {
		t.Fatalf("AuthorizeURL = %q, unexpected", url)
	}
}

func TestCompleteOAuth_ComingSoon_Rejected(t *testing.T) {
	svc, _ := testService(t, &fakeOAuthClient{configured: true}, nil)
	if err := svc.CompleteOAuth(context.Background(), "google", "code", "u1"); !errors.Is(err, apperrs.ErrInvalid) {
		t.Fatalf("CompleteOAuth(google) = %v, want ErrInvalid", err)
	}
}

func TestCompleteOAuth_SavesCredentials(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	oauth := &fakeOAuthClient{configured: true, exchangeTS: &TokenSet{
		AccessToken: "at1", RefreshToken: "rt1", ExpiresIn: time.Hour,
	}}
	svc, store := testService(t, oauth, func() time.Time { return now })

	if err := svc.CompleteOAuth(context.Background(), "github", "code", "user-1"); err != nil {
		t.Fatalf("CompleteOAuth: %v", err)
	}
	cred, err := store.GetCredentials(context.Background(), "github")
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if cred.AccessToken != "at1" || cred.RefreshToken != "rt1" {
		t.Fatalf("cred tokens = %+v, want at1/rt1", cred)
	}
	if cred.ConnectedBy != "user-1" {
		t.Fatalf("cred.ConnectedBy = %q, want user-1", cred.ConnectedBy)
	}
	if !cred.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("cred.ExpiresAt = %v, want %v", cred.ExpiresAt, now.Add(time.Hour))
	}

	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var githubStatus CredentialStatus
	found := false
	for _, s := range list {
		if s.Connector.ID == "github" {
			githubStatus = s.Status
			found = true
		}
	}
	if !found || !githubStatus.Configured || githubStatus.ConnectedBy != "user-1" {
		t.Fatalf("List() github status = %+v, found=%v, want configured+connected_by user-1", githubStatus, found)
	}
}

func TestList_NotConnected_ZeroStatus(t *testing.T) {
	svc, _ := testService(t, &fakeOAuthClient{configured: true}, nil)
	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(list))
	}
	for _, s := range list {
		if s.Status.Configured {
			t.Fatalf("connector %s: Status.Configured = true, want false (never connected)", s.Connector.ID)
		}
	}
}

// TestList_AppConfigured mirrors the OAuth app registration state so the frontend can offer "Set up app" before Connect.
func TestList_AppConfigured(t *testing.T) {
	for _, configured := range []bool{false, true} {
		svc, _ := testService(t, &fakeOAuthClient{configured: configured}, nil)
		list, err := svc.List(context.Background())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		for _, s := range list {
			if s.Connector.ID == "github" && s.AppConfigured != configured {
				t.Fatalf("github: AppConfigured = %v, want %v", s.AppConfigured, configured)
			}
			if s.Connector.ID == "google" && s.AppConfigured {
				t.Fatalf("google: AppConfigured = true, want false (no OAuth)")
			}
		}
	}
}

// TestList_Available reports whether a connector has a live OAuth
// implementation ("coming soon" vs real), independent of whether it is
// currently connected; the frontend needs this to render "Coming soon"
// badges correctly.
func TestList_Available(t *testing.T) {
	svc, _ := testService(t, &fakeOAuthClient{configured: true}, nil)
	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, s := range list {
		switch s.Connector.ID {
		case "github": // has OAuth wired in testService
			if !s.Available {
				t.Fatalf("github: Available = false, want true (has OAuth)")
			}
		case "google": // OAuth: nil in testService, i.e. "coming soon"
			if s.Available {
				t.Fatalf("google: Available = true, want false (coming soon)")
			}
		}
	}
}

func TestDisconnect_RevokesAndDeletes(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true}
	svc, store := testService(t, oauth, nil)
	if err := store.SaveCredentials(context.Background(), Credentials{ConnectorID: "github", AccessToken: "at1"}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	if err := svc.Disconnect(context.Background(), "github"); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if oauth.revokedToken != "at1" {
		t.Fatalf("revokedToken = %q, want at1", oauth.revokedToken)
	}
	if _, err := store.GetCredentials(context.Background(), "github"); !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("GetCredentials after disconnect = %v, want ErrNotFound", err)
	}
}

func TestDisconnect_NeverConnected_NoOp(t *testing.T) {
	svc, _ := testService(t, &fakeOAuthClient{configured: true}, nil)
	if err := svc.Disconnect(context.Background(), "github"); err != nil {
		t.Fatalf("Disconnect on never-connected connector: %v, want nil (no-op)", err)
	}
}

func TestDisconnect_RevokeErrorStillDeletesLocalRow(t *testing.T) {
	oauth := &fakeOAuthClient{configured: true, revokeErr: errors.New("provider unreachable")}
	svc, store := testService(t, oauth, nil)
	if err := store.SaveCredentials(context.Background(), Credentials{ConnectorID: "github", AccessToken: "at1"}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	if err := svc.Disconnect(context.Background(), "github"); err != nil {
		t.Fatalf("Disconnect: %v, want nil even though revoke failed (best-effort)", err)
	}
	if _, err := store.GetCredentials(context.Background(), "github"); !errors.Is(err, apperrs.ErrNotFound) {
		t.Fatalf("GetCredentials after disconnect = %v, want ErrNotFound", err)
	}
}

func TestAccessToken_ExpiredTriggersRefreshAndSave(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	oauth := &fakeOAuthClient{configured: true, refreshTS: &TokenSet{
		AccessToken: "at2", RefreshToken: "rt2", ExpiresIn: time.Hour,
	}}
	svc, store := testService(t, oauth, func() time.Time { return now })
	if err := store.SaveCredentials(context.Background(), Credentials{
		ConnectorID:  "github",
		AccessToken:  "at1-expired",
		RefreshToken: "rt1",
		ExpiresAt:    now.Add(-time.Minute), // already expired
	}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	token, err := svc.AccessToken(context.Background(), "github")
	if err != nil {
		t.Fatalf("accessToken: %v", err)
	}
	if token != "at2" {
		t.Fatalf("accessToken = %q, want at2 (refreshed)", token)
	}
	cred, err := store.GetCredentials(context.Background(), "github")
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	if cred.AccessToken != "at2" || cred.RefreshToken != "rt2" {
		t.Fatalf("saved cred after refresh = %+v, want at2/rt2", cred)
	}
	if !cred.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("cred.ExpiresAt after refresh = %v, want %v", cred.ExpiresAt, now.Add(time.Hour))
	}
}

// A pasted API token is the whole credential: no expiry, no refresh, handed to callers verbatim.
func TestAccessToken_ManualTokenReturnedAsIs(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// A refresh attempt would fail loudly, proving the manual token never reaches the OAuth path.
	oauth := &fakeOAuthClient{configured: false, refreshErr: errors.New("refresh must not run")}
	svc, store := testService(t, oauth, func() time.Time { return now })
	if err := store.SaveCredentials(context.Background(), Credentials{
		ConnectorID:  "github",
		RefreshToken: "rt-unused",
		ExpiresAt:    now.Add(-time.Minute),
		ManualFields: map[string]string{"api_token": "cf-token"},
	}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	token, err := svc.AccessToken(context.Background(), "github")
	if err != nil {
		t.Fatalf("accessToken: %v", err)
	}
	if token != "cf-token" {
		t.Fatalf("accessToken = %q, want cf-token (manual)", token)
	}
}

func TestAccessToken_NotExpired_NoRefresh(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	oauth := &fakeOAuthClient{configured: true, refreshErr: errors.New("should not be called")}
	svc, store := testService(t, oauth, func() time.Time { return now })
	if err := store.SaveCredentials(context.Background(), Credentials{
		ConnectorID: "github",
		AccessToken: "at1-still-valid",
		ExpiresAt:   now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed SaveCredentials: %v", err)
	}

	token, err := svc.AccessToken(context.Background(), "github")
	if err != nil {
		t.Fatalf("accessToken: %v", err)
	}
	if token != "at1-still-valid" {
		t.Fatalf("accessToken = %q, want unchanged at1-still-valid", token)
	}
}
