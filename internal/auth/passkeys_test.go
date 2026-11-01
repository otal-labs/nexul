package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

const testOrigin = "https://nexul.example.com"

type fakePasskeyStore struct {
	mu     sync.Mutex
	byID   map[string]Passkey
	outbox []eventbus.OutboxEvent
}

func newFakePasskeyStore() *fakePasskeyStore {
	return &fakePasskeyStore{byID: map[string]Passkey{}}
}

func (f *fakePasskeyStore) InsertPasskey(_ context.Context, p *Passkey, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byID[p.ID]; ok {
		return apperrs.ErrConflict
	}
	f.byID[p.ID] = *p
	f.outbox = append(f.outbox, events...)
	return nil
}

func (f *fakePasskeyStore) GetPasskey(_ context.Context, id string) (*Passkey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &p, nil
}

func (f *fakePasskeyStore) ListPasskeys(_ context.Context, userID string) ([]Passkey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Passkey
	for _, p := range f.byID {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakePasskeyStore) UpdatePasskeyUse(_ context.Context, id string, credential []byte, usedAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	p.Credential, p.LastUsedAt = credential, &usedAt
	f.byID[id] = p
	return nil
}

func (f *fakePasskeyStore) RenamePasskey(_ context.Context, userID, id, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok || p.UserID != userID {
		return apperrs.ErrNotFound
	}
	p.Name = name
	f.byID[id] = p
	return nil
}

func (f *fakePasskeyStore) DeletePasskey(_ context.Context, userID, id string, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.byID[id]
	if !ok || p.UserID != userID {
		return apperrs.ErrNotFound
	}
	delete(f.byID, id)
	f.outbox = append(f.outbox, events...)
	return nil
}

// softAuthenticator is a platform authenticator in software: one ES256 key, a counter, and the user handle it holds.
type softAuthenticator struct {
	key    *ecdsa.PrivateKey
	credID []byte
	userID []byte
	count  uint32
}

func newSoftAuthenticator(t *testing.T) *softAuthenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	credID := make([]byte, 16)
	_, err = rand.Read(credID)
	require.NoError(t, err)
	return &softAuthenticator{key: key, credID: credID}
}

const (
	flagUserPresent  = 0x01
	flagUserVerified = 0x04
	flagAttested     = 0x40
)

func authenticatorData(rpID string, flags byte, count uint32, attested []byte) []byte {
	h := sha256.Sum256([]byte(rpID))
	out := append(h[:], flags)
	out = binary.BigEndian.AppendUint32(out, count)
	return append(out, attested...)
}

func clientDataJSON(t *testing.T, typ string, challenge []byte, origin string) []byte {
	t.Helper()
	out, err := json.Marshal(map[string]any{"type": typ, "challenge": base64.RawURLEncoding.EncodeToString(challenge), "origin": origin, "crossOrigin": false})
	require.NoError(t, err)
	return out
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (a *softAuthenticator) create(t *testing.T, rpID, origin string, userID string, options *protocol.PublicKeyCredentialCreationOptions) []byte {
	t.Helper()
	a.userID = []byte(userID)
	pub, err := a.key.PublicKey.Bytes()
	require.NoError(t, err)
	cose, err := webauthncbor.Marshal(webauthncose.EC2PublicKeyData{
		PublicKeyData: webauthncose.PublicKeyData{KeyType: int64(webauthncose.EllipticKey), Algorithm: int64(webauthncose.AlgES256)},
		Curve:         1, XCoord: pub[1:33], YCoord: pub[33:],
	})
	require.NoError(t, err)
	attested := append(make([]byte, 16), byte(len(a.credID)>>8), byte(len(a.credID)))
	attested = append(append(attested, a.credID...), cose...)
	attestation, err := webauthncbor.Marshal(map[string]any{
		"fmt": "none", "attStmt": map[string]any{}, "authData": authenticatorData(rpID, flagUserPresent|flagUserVerified|flagAttested, 0, attested),
	})
	require.NoError(t, err)
	out, err := json.Marshal(map[string]any{
		"id": b64(a.credID), "rawId": b64(a.credID), "type": "public-key", "clientExtensionResults": map[string]any{},
		"response": map[string]any{
			"clientDataJSON":    b64(clientDataJSON(t, "webauthn.create", options.Challenge, origin)),
			"attestationObject": b64(attestation),
		},
	})
	require.NoError(t, err)
	return out
}

func (a *softAuthenticator) get(t *testing.T, rpID, origin string, options *protocol.PublicKeyCredentialRequestOptions) []byte {
	t.Helper()
	a.count++
	data := authenticatorData(rpID, flagUserPresent|flagUserVerified, a.count, nil)
	client := clientDataJSON(t, "webauthn.get", options.Challenge, origin)
	clientHash := sha256.Sum256(client)
	digest := sha256.Sum256(append(append([]byte{}, data...), clientHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	require.NoError(t, err)
	out, err := json.Marshal(map[string]any{
		"id": b64(a.credID), "rawId": b64(a.credID), "type": "public-key", "clientExtensionResults": map[string]any{},
		"response": map[string]any{
			"clientDataJSON": b64(client), "authenticatorData": b64(data), "signature": b64(sig), "userHandle": b64(a.userID),
		},
	})
	require.NoError(t, err)
	return out
}

type passkeyHarness struct {
	s        *Service
	users    *fakeUserStore
	passkeys *fakePasskeyStore
	userID   string
}

func newPasskeyHarness(t *testing.T) passkeyHarness {
	t.Helper()
	users := newFakeUserStore()
	s := newTestService(&fakeGitHub{}, users, newFakeAllowlist(), newFakeSettings())
	passkeys := newFakePasskeyStore()
	s.cfg.Passkeys = passkeys
	u := users.insert(&Identity{UserID: "user-alice", Provider: ProviderGitHub, ProviderUserID: "1", Login: "alice", Name: "Alice"})
	return passkeyHarness{s: s, users: users, passkeys: passkeys, userID: u.ID}
}

// register adds a passkey for userID through the real ceremony and returns the authenticator holding it.
func (h passkeyHarness) register(t *testing.T, userID string) *softAuthenticator {
	t.Helper()
	ctx := WithDevice(t.Context(), Device{Label: "Firefox", Platform: "Linux"})
	options, ceremony, err := h.s.StartPasskeyRegistration(ctx, userID, testOrigin)
	require.NoError(t, err)
	a := newSoftAuthenticator(t)
	_, err = h.s.FinishPasskeyRegistration(ctx, userID, testOrigin, ceremony, a.create(t, "nexul.example.com", testOrigin, userID, options))
	require.NoError(t, err)
	return a
}

func (h passkeyHarness) signIn(t *testing.T, a *softAuthenticator, rpID, origin string) (string, error) {
	t.Helper()
	options, ceremony, err := h.s.StartPasskeyLogin(t.Context(), testOrigin)
	require.NoError(t, err)
	return h.s.FinishPasskeyLogin(t.Context(), testOrigin, ceremony, a.get(t, rpID, origin, options))
}

func TestPasskey_AddedThenUsed_SignsItsOwnerInAndRecordsTheUse(t *testing.T) {
	h := newPasskeyHarness(t)
	a := h.register(t, h.userID)

	stored, err := h.s.ListPasskeys(t.Context(), h.userID)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, "Firefox on Linux", stored[0].Name)
	assert.Equal(t, "nexul.example.com", stored[0].RPID)
	assert.Nil(t, stored[0].LastUsedAt)

	token, err := h.signIn(t, a, "nexul.example.com", testOrigin)
	require.NoError(t, err)
	assert.Equal(t, h.userID, mustVerify(t, h.s, token))

	stored, err = h.s.ListPasskeys(t.Context(), h.userID)
	require.NoError(t, err)
	assert.NotNil(t, stored[0].LastUsedAt, "a sign-in records when the passkey was last used")

	_, err = h.signIn(t, a, "nexul.example.com", testOrigin)
	require.NoError(t, err, "the stored counter moved on, so the next sign-in is not taken for a copy")
	require.Len(t, h.passkeys.outbox, 1)
	assert.Equal(t, TopicPasskeyAdded, h.passkeys.outbox[0].Topic)
}

func TestFinishPasskeyLogin_Refusals(t *testing.T) {
	tests := []struct {
		name   string
		rpID   string
		origin string
		setup  func(h passkeyHarness, a *softAuthenticator)
		want   error
	}{
		{name: "signed for another domain", rpID: "evil.example.com", origin: testOrigin, want: errPasskeyRefused},
		{name: "signed on another origin", rpID: "nexul.example.com", origin: "https://evil.example.com", want: errPasskeyRefused},
		{name: "removed passkey", rpID: "nexul.example.com", origin: testOrigin, want: errPasskeyRefused, setup: func(h passkeyHarness, a *softAuthenticator) {
			for id := range h.passkeys.byID {
				require.NoError(t, h.s.DeletePasskey(context.Background(), h.userID, id))
			}
		}},
		{name: "handle names someone else", rpID: "nexul.example.com", origin: testOrigin, want: errPasskeyRefused, setup: func(_ passkeyHarness, a *softAuthenticator) {
			a.userID = []byte("user-bob")
		}},
		{name: "counter went backwards", rpID: "nexul.example.com", origin: testOrigin, want: errPasskeyRefused, setup: func(h passkeyHarness, a *softAuthenticator) {
			_, err := h.signIn(t, a, "nexul.example.com", testOrigin)
			require.NoError(t, err)
			a.count = 0
		}},
		{name: "disabled account", rpID: "nexul.example.com", origin: testOrigin, want: errAccountDisabled, setup: func(h passkeyHarness, _ *softAuthenticator) {
			h.users.byID[h.userID].AccountStatus = AccountDisabled
		}},
		{name: "removed account", rpID: "nexul.example.com", origin: testOrigin, want: errAccountRemoved, setup: func(h passkeyHarness, _ *softAuthenticator) {
			h.users.byID[h.userID].AccountStatus = AccountRemoved
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newPasskeyHarness(t)
			h.users.insert(&Identity{UserID: "user-bob", Provider: ProviderGitHub, ProviderUserID: "2", Login: "bob"})
			a := h.register(t, h.userID)
			if tt.setup != nil {
				tt.setup(h, a)
			}
			_, err := h.signIn(t, a, tt.rpID, tt.origin)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPasskeyCeremony_IsSpentOnceAndOnlyByItsOwnFlow(t *testing.T) {
	h := newPasskeyHarness(t)
	a := h.register(t, h.userID)
	options, ceremony, err := h.s.StartPasskeyLogin(t.Context(), testOrigin)
	require.NoError(t, err)
	response := a.get(t, "nexul.example.com", testOrigin, options)
	_, err = h.s.FinishPasskeyLogin(t.Context(), testOrigin, ceremony, response)
	require.NoError(t, err)
	_, err = h.s.FinishPasskeyLogin(t.Context(), testOrigin, ceremony, response)
	assert.ErrorIs(t, err, errPasskeyCeremonyGone, "a replayed ceremony")

	_, ceremony, err = h.s.StartPasskeyRegistration(t.Context(), h.userID, testOrigin)
	require.NoError(t, err)
	_, err = h.s.FinishPasskeyLogin(t.Context(), testOrigin, ceremony, response)
	assert.ErrorIs(t, err, errPasskeyCeremonyGone, "a registration's ceremony can't finish a sign-in")

	_, ceremony, err = h.s.StartPasskeyRegistration(t.Context(), h.userID, testOrigin)
	require.NoError(t, err)
	_, err = h.s.FinishPasskeyRegistration(t.Context(), "user-bob", testOrigin, ceremony, response)
	assert.ErrorIs(t, err, errPasskeyCeremonyGone, "someone else's registration")

	_, ceremony, err = h.s.StartPasskeyLogin(t.Context(), testOrigin)
	require.NoError(t, err)
	h.s.cfg.Now = func() time.Time { return time.Unix(1_700_000_000, 0).Add(passkeyCeremonyTTL) }
	_, err = h.s.FinishPasskeyLogin(t.Context(), testOrigin, ceremony, response)
	assert.ErrorIs(t, err, errPasskeyCeremonyGone, "past five minutes")
}

func TestPasskeyCeremonies_Full_RefusesUntilOneExpires(t *testing.T) {
	c := newPasskeyCeremonies()
	now := time.Unix(1_700_000_000, 0)
	for i := range passkeyCeremonyCap {
		_, err := c.put(now, passkeyCeremony{addr: fmt.Sprintf("198.51.100.%d", i)})
		require.NoError(t, err)
	}
	_, err := c.put(now, passkeyCeremony{addr: "203.0.113.1"})
	assert.ErrorIs(t, err, apperrs.ErrRateLimited)
	_, err = c.put(now.Add(passkeyCeremonyTTL), passkeyCeremony{addr: "203.0.113.1"})
	assert.NoError(t, err, "expired ceremonies make room")
}

func TestPasskeyCeremonies_OneAddress_CantFillTheMap(t *testing.T) {
	c := newPasskeyCeremonies()
	now := time.Unix(1_700_000_000, 0)
	for range passkeyCeremoniesPerAddress {
		_, err := c.put(now, passkeyCeremony{addr: "203.0.113.1"})
		require.NoError(t, err)
	}
	_, err := c.put(now, passkeyCeremony{addr: "203.0.113.1"})
	assert.ErrorIs(t, err, apperrs.ErrRateLimited)
	_, err = c.put(now, passkeyCeremony{addr: "203.0.113.2"})
	assert.NoError(t, err, "another address still gets in")
}

func TestPasskeyHost(t *testing.T) {
	tests := []struct{ origin, want string }{
		{"https://nexul.example.com", "nexul.example.com"},
		{"https://nexul.example.com:8443", "nexul.example.com"},
		{"http://localhost:5173", "localhost"},
		{"http://nexul.example.com", ""},
		{"https://203.0.113.7", ""},
		{"http://127.0.0.1:8080", ""},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, passkeyHost(tt.origin), tt.origin)
	}
}

func TestStartPasskeyRegistration_OffHTTPS_IsRefused(t *testing.T) {
	h := newPasskeyHarness(t)
	_, _, err := h.s.StartPasskeyRegistration(t.Context(), h.userID, "http://203.0.113.7:5123")
	assert.ErrorIs(t, err, errPasskeysOff)
	h.s.cfg.Passkeys = nil
	_, _, err = h.s.StartPasskeyLogin(t.Context(), testOrigin)
	assert.ErrorIs(t, err, errPasskeysOff)
}

func TestRenameAndDeletePasskey(t *testing.T) {
	h := newPasskeyHarness(t)
	h.register(t, h.userID)
	var id string
	for k := range h.passkeys.byID {
		id = k
	}

	assert.ErrorIs(t, h.s.RenamePasskey(t.Context(), h.userID, id, "  "), apperrs.ErrInvalid)
	assert.ErrorIs(t, h.s.RenamePasskey(t.Context(), h.userID, id, string(make([]rune, passkeyNameMax+1))), apperrs.ErrInvalid)
	assert.ErrorIs(t, h.s.RenamePasskey(t.Context(), "user-bob", id, "Mine now"), apperrs.ErrNotFound)
	require.NoError(t, h.s.RenamePasskey(t.Context(), h.userID, id, " Work laptop "))
	assert.Equal(t, "Work laptop", h.passkeys.byID[id].Name)

	assert.ErrorIs(t, h.s.DeletePasskey(t.Context(), "user-bob", id), apperrs.ErrNotFound, "nobody removes another person's passkey")
	require.NoError(t, h.s.DeletePasskey(t.Context(), h.userID, id))
	assert.Empty(t, h.passkeys.byID)
	assert.Equal(t, TopicPasskeyRemoved, h.passkeys.outbox[len(h.passkeys.outbox)-1].Topic)
}

func TestHandler_PasskeyRoutes_AddThenSignInOverHTTP(t *testing.T) {
	h := newPasskeyHarness(t)
	h.s.cfg.SPAOrigin = testOrigin
	protected := protectedHandler(h.s, h.users)
	public := NewHandler(h.s).Routes()
	call := func(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := call(protected, protectedRequest(t, h.s, h.users, http.MethodPost, "/api/auth/passkeys/start", h.userID, ""))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var creation struct {
		Ceremony string                                      `json:"ceremony"`
		Options  protocol.PublicKeyCredentialCreationOptions `json:"options"`
	}
	require.NoError(t, decodeJSON(rec, &creation))
	assert.Equal(t, "nexul.example.com", creation.Options.RelyingParty.ID)

	a := newSoftAuthenticator(t)
	body, err := json.Marshal(map[string]any{"ceremony": creation.Ceremony, "credential": json.RawMessage(a.create(t, "nexul.example.com", testOrigin, h.userID, &creation.Options))})
	require.NoError(t, err)
	rec = call(protected, protectedRequest(t, h.s, h.users, http.MethodPost, "/api/auth/passkeys", h.userID, string(body)))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "credential", "the stored record never reaches a client")

	rec = call(protected, protectedRequest(t, h.s, h.users, http.MethodGet, "/api/auth/passkeys", h.userID, ""))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"available":true`)

	rec = call(public, httptest.NewRequest(http.MethodPost, "/api/auth/passkeys/login/start", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var assertion struct {
		Ceremony string                                     `json:"ceremony"`
		Options  protocol.PublicKeyCredentialRequestOptions `json:"options"`
	}
	require.NoError(t, decodeJSON(rec, &assertion))
	body, err = json.Marshal(map[string]any{"ceremony": assertion.Ceremony, "credential": json.RawMessage(a.get(t, "nexul.example.com", testOrigin, &assertion.Options))})
	require.NoError(t, err)
	rec = call(public, httptest.NewRequest(http.MethodPost, "/api/auth/passkeys/login/finish", strings.NewReader(string(body))))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var signedIn struct {
		Token string `json:"token"`
	}
	require.NoError(t, decodeJSON(rec, &signedIn))
	assert.Equal(t, h.userID, mustVerify(t, h.s, signedIn.Token))

	id := base64.RawURLEncoding.EncodeToString(a.credID)
	rec = call(protected, protectedRequest(t, h.s, h.users, http.MethodPatch, "/api/auth/passkeys/"+id, h.userID, `{"name":"Work laptop"}`))
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Equal(t, "Work laptop", h.passkeys.byID[id].Name)

	h.s.cfg.PATs = newFakePATStore()
	pat, _, err := h.s.MintPAT(t.Context(), h.userID, "ci")
	require.NoError(t, err)
	rec = doRequest(protected, http.MethodDelete, "/api/auth/passkeys/"+id, "Bearer "+pat, "")
	assert.Equal(t, http.StatusForbidden, rec.Code, "a personal access token has no say over passkeys")
	rec = call(protected, protectedRequest(t, h.s, h.users, http.MethodDelete, "/api/auth/passkeys/"+id, h.userID, ""))
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Empty(t, h.passkeys.byID)

	rec = call(public, httptest.NewRequest(http.MethodPost, "/api/auth/passkeys/login/finish", strings.NewReader(`{"ceremony":"`+strings.Repeat("x", passkeyBodyMax)+`"}`)))
	assert.Equal(t, http.StatusBadRequest, rec.Code, "an oversized body is refused before it is read")
}
