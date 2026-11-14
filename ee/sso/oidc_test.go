package sso

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeProvider is an OIDC provider that signs real ID tokens, so the verification under test is the library's own.
type fakeProvider struct {
	srv  *httptest.Server
	key  *rsa.PrivateKey
	mu   sync.Mutex
	form url.Values
	// claims overrides what the next ID token says; iss, aud, exp and nonce default to a valid token.
	claims map[string]any
	// signWith signs with another key when set, a token the provider's key set does not vouch for.
	signWith *rsa.PrivateKey
	// rejectClient answers the code exchange as an unauthorized client.
	rejectClient bool
	// scopes is what discovery advertises as scopes_supported; none when nil.
	scopes []string
	// refreshToken is issued beside the access token on every grant.
	refreshToken string
	// refreshStatus and refreshError, when set, answer a refresh_token grant as that error response.
	refreshStatus int
	refreshError  string
	// hits counts the requests each path has answered.
	hits map[string]int
}

func newFakeProvider(t *testing.T) *fakeProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	p := &fakeProvider{key: key, hits: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		doc := map[string]any{
			"issuer": p.srv.URL, "authorization_endpoint": p.srv.URL + "/authorize", "token_endpoint": p.srv.URL + "/token",
			"jwks_uri": p.srv.URL + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"},
		}
		p.mu.Lock()
		if p.scopes != nil {
			doc["scopes_supported"] = p.scopes
		}
		p.mu.Unlock()
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "k1",
			"n": b64(key.N.Bytes()), "e": b64(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", p.token)
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		p.hits[r.URL.Path]++
		p.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (p *fakeProvider) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p.mu.Lock()
	p.form = r.PostForm
	claims := map[string]any{"iss": p.srv.URL, "aud": "client-1", "sub": "sub-1", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()}
	for k, v := range p.claims {
		claims[k] = v
	}
	refreshToken, refreshStatus, refreshError := p.refreshToken, p.refreshStatus, p.refreshError
	p.mu.Unlock()
	if p.form.Get("grant_type") == "refresh_token" {
		w.Header().Set("Content-Type", "application/json")
		if refreshStatus != 0 {
			w.WriteHeader(refreshStatus)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": refreshError})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at2", "token_type": "Bearer", "refresh_token": refreshToken})
		return
	}
	if p.rejectClient {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_client"})
		return
	}
	signer := p.key
	if p.signWith != nil {
		signer = p.signWith
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "id_token": sign(signer, claims), "refresh_token": refreshToken})
}

func sign(key *rsa.PrivateKey, claims map[string]any) string {
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "k1", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	signing := b64(header) + "." + b64(payload)
	sum := sha256.Sum256([]byte(signing))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	return signing + "." + b64(sig)
}

func (p *fakeProvider) connect(t *testing.T) IdP {
	t.Helper()
	idp, err := dial(http.DefaultClient, []byte("server-secret"), time.Now)(t.Context(), ConnectConfig{
		Issuer: p.srv.URL, ClientID: "client-1", ClientSecret: "s3cret", RedirectURI: "http://nexul.test/auth/oidc/callback",
	})
	require.NoError(t, err)
	return idp
}

// nonceOf starts a sign-in the way the handler does and returns the nonce the provider must echo.
func nonceOf(t *testing.T, idp IdP, state string) string {
	t.Helper()
	u, err := url.Parse(idp.AuthURL(state))
	require.NoError(t, err)
	return u.Query().Get("nonce")
}

func TestProvider_Exchange_VerifiesTheIDToken(t *testing.T) {
	t.Parallel()
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tests := []struct {
		name     string
		claims   func(nonce string) map[string]any
		signWith *rsa.PrivateKey
		wantErr  bool
	}{
		{"a valid token yields its claims", func(n string) map[string]any {
			return map[string]any{"nonce": n, "email": "alice@example.com", "email_verified": true, "name": "Alice"}
		}, nil, false},
		{"another sign-in's nonce is refused", func(string) map[string]any { return map[string]any{"nonce": "someone-elses"} }, nil, true},
		{"a token for another client is refused", func(n string) map[string]any { return map[string]any{"nonce": n, "aud": "client-2"} }, nil, true},
		{"an expired token is refused", func(n string) map[string]any {
			return map[string]any{"nonce": n, "exp": time.Now().Add(-time.Hour).Unix()}
		}, nil, true},
		{"a token from another issuer is refused", func(n string) map[string]any { return map[string]any{"nonce": n, "iss": "https://evil.example.com"} }, nil, true},
		{"a token the key set does not vouch for is refused", func(n string) map[string]any { return map[string]any{"nonce": n} }, other, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := newFakeProvider(t)
			idp := p.connect(t)
			p.signWith = tt.signWith
			p.claims = tt.claims(nonceOf(t, idp, "state-1"))
			grant, err := idp.Exchange(t.Context(), "code-1", "state-1")
			if tt.wantErr {
				require.ErrorIs(t, err, apperrs.ErrUnauthorized)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, Claims{Subject: "sub-1", Email: "alice@example.com", EmailVerified: true, Name: "Alice"}, grant.Claims)
		})
	}
}

func TestProvider_Exchange_ProvesTheCodeWithTheStatesPKCEVerifier(t *testing.T) {
	t.Parallel()
	p := newFakeProvider(t)
	idp := p.connect(t)
	u, err := url.Parse(idp.AuthURL("state-1"))
	require.NoError(t, err)
	challenge := u.Query().Get("code_challenge")
	require.Equal(t, "S256", u.Query().Get("code_challenge_method"))
	p.claims = map[string]any{"nonce": u.Query().Get("nonce")}

	_, err = idp.Exchange(t.Context(), "code-1", "state-1")
	require.NoError(t, err)

	sum := sha256.Sum256([]byte(p.form.Get("code_verifier")))
	assert.Equal(t, challenge, b64(sum[:]), "the verifier sent with the code hashes to the challenge the sign-in started with")
}

func TestProvider_Exchange_RejectedClientIsToldApart(t *testing.T) {
	t.Parallel()
	p := newFakeProvider(t)
	p.rejectClient = true
	_, err := p.connect(t).Exchange(t.Context(), "code-1", "state-1")
	require.ErrorIs(t, err, errClientRejected, "so a re-read secret reference gets its one retry")
}

func TestProvider_Exchange_EmailVerifiedAsAStringIsRead(t *testing.T) {
	t.Parallel()
	for value, want := range map[any]bool{"true": true, "false": false, true: true, nil: false} {
		p := newFakeProvider(t)
		idp := p.connect(t)
		p.claims = map[string]any{"nonce": nonceOf(t, idp, "s"), "email_verified": value}
		grant, err := idp.Exchange(t.Context(), "c", "s")
		require.NoError(t, err)
		assert.Equal(t, want, grant.Claims.EmailVerified, "%v", value)
	}
}

func TestProvider_Exchange_RequestsOfflineAccessOnlyWhenTheProviderAdvertisesIt(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		scopes []string
		want   bool
	}{
		"advertised":     {[]string{"openid", "email", "offline_access"}, true},
		"not advertised": {[]string{"openid", "email"}, false},
		"no list":        {nil, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := newFakeProvider(t)
			p.scopes = tt.scopes
			u, err := url.Parse(p.connect(t).AuthURL("state-1"))
			require.NoError(t, err)
			assert.Equal(t, tt.want, strings.Contains(" "+u.Query().Get("scope")+" ", " offline_access "))
		})
	}
}

func TestProvider_Exchange_ReturnsTheRefreshTokenTheProviderIssued(t *testing.T) {
	t.Parallel()
	p := newFakeProvider(t)
	p.refreshToken = "rt-1"
	idp := p.connect(t)
	p.claims = map[string]any{"nonce": nonceOf(t, idp, "s")}
	grant, err := idp.Exchange(t.Context(), "c", "s")
	require.NoError(t, err)
	assert.Equal(t, "rt-1", grant.RefreshToken)
}

func TestProvider_Refresh_MapsTheProvidersAnswerToWhatTheRecheckDoes(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		status  int
		code    string
		wantErr error
	}{
		"invalid_grant as 400 is refused": {http.StatusBadRequest, "invalid_grant", errGrantRefused},
		"invalid_grant as 401 is refused": {http.StatusUnauthorized, "invalid_grant", errGrantRefused},
		"another 400 keeps everything":    {http.StatusBadRequest, "invalid_request", errProviderUnavailable},
		"a bare 401 keeps everything":     {http.StatusUnauthorized, "", errProviderUnavailable},
		"invalid_client keeps everything": {http.StatusUnauthorized, "invalid_client", errProviderUnavailable},
		"invalid_client as 400 keeps":     {http.StatusBadRequest, "invalid_client", errProviderUnavailable},
		"a 5xx keeps everything":          {http.StatusBadGateway, "", errProviderUnavailable},
		"a 429 keeps everything":          {http.StatusTooManyRequests, "", errProviderUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := newFakeProvider(t)
			p.refreshStatus, p.refreshError = tt.status, tt.code
			_, err := p.connect(t).Refresh(t.Context(), "rt-1")
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestProvider_Refresh_AnUnreachableProviderKeepsEverything(t *testing.T) {
	t.Parallel()
	p := newFakeProvider(t)
	idp := p.connect(t)
	p.srv.Close()
	_, err := idp.Refresh(t.Context(), "rt-1")
	require.ErrorIs(t, err, errProviderUnavailable)
}

func TestProvider_Refresh_ReturnsTheRotatedTokenOrTheOldOne(t *testing.T) {
	t.Parallel()
	p := newFakeProvider(t)
	idp := p.connect(t)
	p.refreshToken = "rt-2"
	got, err := idp.Refresh(t.Context(), "rt-1")
	require.NoError(t, err)
	assert.Equal(t, "rt-2", got)
	assert.Equal(t, "refresh_token", p.form.Get("grant_type"))
	assert.Equal(t, "rt-1", p.form.Get("refresh_token"))

	p.refreshToken = ""
	got, err = idp.Refresh(t.Context(), "rt-1")
	require.NoError(t, err)
	assert.Equal(t, "rt-1", got, "a provider that does not rotate leaves the token as it was")
}

func TestDial_SignInsReuseOneDiscoveryAndKeySet_UntilItAgesOrACheckAsksFresh(t *testing.T) {
	t.Parallel()
	fp := newFakeProvider(t)
	now := time.Now()
	connect := dial(http.DefaultClient, []byte("server-secret"), func() time.Time { return now })
	cfg := ConnectConfig{Issuer: fp.srv.URL, ClientID: "client-1", ClientSecret: "s3cret", RedirectURI: "http://nexul.test/auth/oidc/callback"}
	signIn := func(t *testing.T, state string) {
		t.Helper()
		idp, err := connect(t.Context(), cfg)
		require.NoError(t, err, "the sign-in's start")
		nonce := nonceOf(t, idp, state)
		idp, err = connect(t.Context(), cfg)
		require.NoError(t, err, "its callback")
		fp.mu.Lock()
		fp.claims = map[string]any{"nonce": nonce}
		fp.mu.Unlock()
		_, err = idp.Exchange(t.Context(), "code", state)
		require.NoError(t, err)
	}
	hits := func() (discovery, keys int) {
		fp.mu.Lock()
		defer fp.mu.Unlock()
		return fp.hits["/.well-known/openid-configuration"], fp.hits["/jwks"]
	}

	signIn(t, "state-1")
	signIn(t, "state-2")
	discovery, keys := hits()
	assert.Equal(t, 1, discovery, "two sign-ins, four connects, one discovery")
	assert.Equal(t, 1, keys)

	fresh := cfg
	fresh.Fresh = true
	_, err := connect(t.Context(), fresh)
	require.NoError(t, err)
	discovery, _ = hits()
	assert.Equal(t, 2, discovery, "a check reaches the provider now")

	now = now.Add(discoveryTTL)
	signIn(t, "state-3")
	discovery, keys = hits()
	assert.Equal(t, 3, discovery, "an hour-old discovery is fetched again")
	assert.Equal(t, 2, keys, "with its key set")
}

func TestDial_UnreachableIssuer_IsRefusedWithTheAddress(t *testing.T) {
	t.Parallel()
	_, err := dial(http.DefaultClient, nil, time.Now)(t.Context(), ConnectConfig{Issuer: "http://127.0.0.1:1"})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "http://127.0.0.1:1")
}

func TestCheckIssuer(t *testing.T) {
	t.Parallel()
	for issuer, ok := range map[string]bool{
		"https://login.example.com":     true,
		"https://login.example.com/x/y": true,
		"http://localhost:5556/dex":     true,
		"http://127.0.0.1:8080":         true,
		"http://login.example.com":      false,
		"login.example.com":             false,
		"https://":                      false,
		"https://user:pw@example.com":   false,
		"https://example.com?a=b":       false,
		"ftp://localhost":               false,
	} {
		err := checkIssuer(issuer)
		assert.Equal(t, ok, err == nil, issuer)
	}
}
