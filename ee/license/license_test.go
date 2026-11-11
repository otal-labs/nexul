package license

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const testURL = "https://nexul.example.com"

var testNow = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

type fakeRepo struct {
	mu     sync.Mutex
	key    string
	setBy  string
	events []eventbus.OutboxEvent
	err    error
}

func (f *fakeRepo) Key(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.key, f.err
}

func (f *fakeRepo) SetKey(_ context.Context, key, setBy string, _ time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.key, f.setBy = key, setBy
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) DeleteKey(_ context.Context, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.key = ""
	f.events = append(f.events, evts...)
	return nil
}

// fakeGate lets the owner through everything and the member through reads only.
type fakeGate struct{}

func (fakeGate) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	a, _ := identity.ActorFromCtx(ctx)
	if a.ID == "owner" || action == permissions.InstanceRead {
		return nil
	}
	return apperrs.ErrForbidden
}

func asOwner(t *testing.T) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: "owner"})
}

func asMember(t *testing.T) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: "member"})
}

type issuer struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

func newIssuer(t *testing.T) issuer {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return issuer{pub: pub, priv: priv}
}

func (is issuer) key(t *testing.T, opts ...func(*Claims)) string {
	t.Helper()
	c := Claims{Licensee: "Acme", InstanceURL: testURL, Features: Features, ExpiresAt: testNow.Add(365 * 24 * time.Hour), IssuedAt: testNow}
	for _, o := range opts {
		o(&c)
	}
	k, err := Sign(c, is.priv)
	require.NoError(t, err)
	return k
}

func expiresIn(d time.Duration) func(*Claims) {
	return func(c *Claims) { c.ExpiresAt = testNow.Add(d) }
}

func newTestService(is issuer, repo *fakeRepo, instanceURL string) *Service {
	s := NewService(repo, func(context.Context) (string, error) { return instanceURL, nil }).WithGate(fakeGate{})
	s.pub = is.pub
	s.now = func() time.Time { return testNow }
	return s
}

func TestPublicKey_IsTheBuiltInIssuerKey(t *testing.T) {
	assert.Len(t, PublicKey(), ed25519.PublicKeySize)
}

func TestParse_RefusesWhatIsNotAValidKey(t *testing.T) {
	is := newIssuer(t)
	good := is.key(t)
	payload, sig, _ := strings.Cut(strings.TrimPrefix(good, keyPrefix), ".")
	enc := base64.RawURLEncoding
	tampered := keyPrefix + enc.EncodeToString([]byte(strings.Replace(mustDecode(t, payload), "Acme", "Evil", 1))) + "." + sig
	unsignedClaims, _ := json.Marshal(Claims{Licensee: "Acme"})
	emptyClaims := keyPrefix + enc.EncodeToString(unsignedClaims) + "." + enc.EncodeToString(ed25519.Sign(is.priv, unsignedClaims))

	tests := []struct {
		name string
		key  string
		want error
	}{
		{"no prefix", "hello", errNotAKey},
		{"no signature", keyPrefix + payload, errNotAKey},
		{"payload not base64", keyPrefix + "!!." + sig, errNotAKey},
		{"signature not base64", keyPrefix + payload + ".!!", errNotAKey},
		{"changed after signing", tampered, errBadSignature},
		{"signed by another issuer", newIssuer(t).key(t), errBadSignature},
		{"signed but not json", keyPrefix + enc.EncodeToString([]byte("x")) + "." + enc.EncodeToString(ed25519.Sign(is.priv, []byte("x"))), errNotAKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.key, is.pub)
			require.ErrorIs(t, err, tt.want)
			assert.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
	t.Run("signed claims missing their fields", func(t *testing.T) {
		_, err := Parse(emptyClaims, is.pub)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.Contains(t, err.Error(), "instance's public URL")
	})
	t.Run("a valid key with surrounding whitespace", func(t *testing.T) {
		c, err := Parse("\n  "+good+"\n", is.pub)
		require.NoError(t, err)
		assert.Equal(t, "Acme", c.Licensee)
		assert.Equal(t, Features, c.Features)
	})
}

func mustDecode(t *testing.T, s string) string {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(s)
	require.NoError(t, err)
	return string(b)
}

func TestSign_RefusesIncompleteClaims(t *testing.T) {
	is := newIssuer(t)
	base := Claims{Licensee: "Acme", InstanceURL: testURL, Features: Features, ExpiresAt: testNow}
	tests := []struct {
		name   string
		change func(*Claims)
		want   string
	}{
		{"no licensee", func(c *Claims) { c.Licensee = " " }, "licensee"},
		{"no url", func(c *Claims) { c.InstanceURL = "" }, "public URL"},
		{"not a web url", func(c *Claims) { c.InstanceURL = "ftp://nexul.example.com" }, "public URL"},
		{"no features", func(c *Claims) { c.Features = nil }, "at least one feature"},
		{"no expiry", func(c *Claims) { c.ExpiresAt = time.Time{} }, "expiry"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := base
			tt.change(&c)
			_, err := Sign(c, is.priv)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestEvaluate_StateFollowsTheAddressAndTheClock(t *testing.T) {
	c := Claims{Licensee: "Acme", InstanceURL: testURL, Features: Features}
	tests := []struct {
		name        string
		expiresIn   time.Duration
		instanceURL string
		want        State
	}{
		{"well before expiry", 60 * 24 * time.Hour, testURL, StateActive},
		{"one minute outside the notice", NoticeWindow + time.Minute, testURL, StateActive},
		{"exactly 14 days left", NoticeWindow, testURL, StateExpiring},
		{"one day left", 24 * time.Hour, testURL, StateExpiring},
		{"the moment it expires", 0, testURL, StateExpired},
		{"after expiry", -time.Hour, testURL, StateExpired},
		{"same address, other case and a trailing slash", 60 * 24 * time.Hour, "HTTPS://Nexul.Example.com/", StateActive},
		{"another address", 60 * 24 * time.Hour, "https://other.example.com", StateInvalid},
		{"another scheme", 60 * 24 * time.Hour, "http://nexul.example.com", StateInvalid},
		{"no address set", 60 * 24 * time.Hour, "", StateInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.ExpiresAt = testNow.Add(tt.expiresIn)
			st := evaluate(c, tt.instanceURL, testNow)
			assert.Equal(t, tt.want, st.State)
			assert.Equal(t, tt.want == StateActive || tt.want == StateExpiring, st.Unlocks(FeatureSSO))
		})
	}
	t.Run("the problem names both addresses", func(t *testing.T) {
		st := evaluate(c, "https://other.example.com", testNow)
		assert.Equal(t, "This key is for https://nexul.example.com, but this instance's address is https://other.example.com.", st.Problem)
		assert.Contains(t, evaluate(c, "", testNow).Problem, "no address set")
	})
}

func TestStatus_Unlocks_OnlyTheKeysFeatures(t *testing.T) {
	st := Status{State: StateActive, Features: []Feature{FeatureSSO}}
	assert.True(t, st.Unlocks(FeatureSSO))
	assert.False(t, st.Unlocks(FeatureBranding))
	assert.False(t, Status{State: StateNone}.Unlocks(FeatureSSO))
}

func TestService_Set_RefusesWithTheReason(t *testing.T) {
	is := newIssuer(t)
	tests := []struct {
		name string
		ctx  func(*testing.T) context.Context
		key  string
		want error
		msg  string
	}{
		{"a member", asMember, is.key(t), apperrs.ErrForbidden, ""},
		{"not a key", asOwner, "paste", apperrs.ErrInvalid, "not a Nexul license key"},
		{"a bad signature", asOwner, newIssuer(t).key(t), apperrs.ErrInvalid, "signature does not check out"},
		{"another instance", asOwner, is.key(t, func(c *Claims) { c.InstanceURL = "https://other.example.com" }), apperrs.ErrInvalid, "This key is for https://other.example.com"},
		{"already expired", asOwner, is.key(t, expiresIn(-24*time.Hour)), apperrs.ErrInvalid, "expired on 10 Nov 2026"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			_, err := newTestService(is, repo, testURL).Set(tt.ctx(t), tt.key)
			require.ErrorIs(t, err, tt.want)
			assert.Contains(t, err.Error(), tt.msg)
			assert.Empty(t, repo.key, "a refused key is never stored")
			assert.Empty(t, repo.events)
		})
	}
}

func TestService_SetThenRemove_StoresTheKeyAndAnnouncesEachChange(t *testing.T) {
	is := newIssuer(t)
	repo := &fakeRepo{}
	s := newTestService(is, repo, testURL)

	st, err := s.Set(asOwner(t), "  "+is.key(t, expiresIn(10*24*time.Hour))+"\n")
	require.NoError(t, err)
	assert.Equal(t, StateExpiring, st.State, "a key inside the notice window is still accepted")
	assert.Equal(t, "Acme", st.Licensee)
	assert.True(t, strings.HasPrefix(repo.key, keyPrefix), "stored trimmed")
	assert.Equal(t, "owner", repo.setBy)
	assert.True(t, s.Enabled(t.Context(), FeatureBranding))

	got, err := s.Status(asMember(t))
	require.NoError(t, err)
	assert.Equal(t, st, got)

	st, err = s.Remove(asOwner(t))
	require.NoError(t, err)
	assert.Equal(t, StateNone, st.State)
	assert.False(t, s.Enabled(t.Context(), FeatureBranding))

	st, err = s.Remove(asOwner(t))
	require.NoError(t, err, "removing no key changes nothing")
	assert.Equal(t, StateNone, st.State)

	require.Len(t, repo.events, 2)
	set := repo.events[0].Payload.(ChangedEvent)
	assert.Equal(t, TopicChanged, repo.events[0].Topic)
	assert.Equal(t, "Acme", set.Licensee)
	assert.Equal(t, "owner", set.ChangedBy)
	assert.Equal(t, testNow, set.ChangedAt)
	assert.True(t, repo.events[1].Payload.(ChangedEvent).Removed)

	_, err = s.Remove(asMember(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestService_StoredKeyThatStoppedChecking_UnlocksNothingAndSaysWhy(t *testing.T) {
	is := newIssuer(t)
	t.Run("the instance moved to another address", func(t *testing.T) {
		repo := &fakeRepo{key: is.key(t)}
		s := newTestService(is, repo, "https://moved.example.com")
		st, err := s.Status(asOwner(t))
		require.NoError(t, err)
		assert.Equal(t, StateInvalid, st.State)
		assert.Contains(t, st.Problem, "https://moved.example.com")
		assert.Equal(t, "Acme", st.Licensee, "who it was for still shows")
		assert.False(t, s.Enabled(t.Context(), FeatureSSO))
	})
	t.Run("the key no longer verifies", func(t *testing.T) {
		repo := &fakeRepo{key: newIssuer(t).key(t)}
		st, err := newTestService(is, repo, testURL).Status(asOwner(t))
		require.NoError(t, err)
		assert.Equal(t, StateInvalid, st.State)
		assert.True(t, strings.HasPrefix(st.Problem, "This key's signature does not check out"), st.Problem)
	})
	t.Run("it expired: the features stop, the key stays", func(t *testing.T) {
		repo := &fakeRepo{key: is.key(t, expiresIn(-time.Hour))}
		s := newTestService(is, repo, testURL)
		st, err := s.Status(asOwner(t))
		require.NoError(t, err)
		assert.Equal(t, StateExpired, st.State)
		assert.False(t, s.Enabled(t.Context(), FeatureSSO))
		assert.NotEmpty(t, repo.key)
	})
}

func TestService_ReadFailures(t *testing.T) {
	is := newIssuer(t)
	boom := errors.New("disk")
	t.Run("the repo fails: no feature runs", func(t *testing.T) {
		s := newTestService(is, &fakeRepo{err: boom}, testURL)
		assert.False(t, s.Enabled(t.Context(), FeatureSSO))
		_, err := s.Status(asOwner(t))
		require.ErrorIs(t, err, boom)
		_, err = s.Remove(asOwner(t))
		require.ErrorIs(t, err, boom)
	})
	t.Run("the instance url cannot be read", func(t *testing.T) {
		repo := &fakeRepo{key: is.key(t)}
		s := newTestService(is, repo, testURL)
		s.instanceURL = func(context.Context) (string, error) { return "", boom }
		_, err := s.Status(asOwner(t))
		require.ErrorIs(t, err, boom)
		_, err = s.Set(asOwner(t), is.key(t))
		require.ErrorIs(t, err, boom)
	})
	t.Run("no gate wired refuses a person", func(t *testing.T) {
		s := NewService(&fakeRepo{}, func(context.Context) (string, error) { return testURL, nil })
		_, err := s.Status(asOwner(t))
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
}

func TestHandler_Routes(t *testing.T) {
	is := newIssuer(t)
	repo := &fakeRepo{}
	h := NewHandler(newTestService(is, repo, testURL)).Routes()
	do := func(ctx context.Context, method, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, method, "/api/instance/license", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := do(asOwner(t), http.MethodPut, `{`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = do(asOwner(t), http.MethodPut, `{"key":"nope"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), `"key":["this is not a Nexul license key"]`)

	rec = do(asMember(t), http.MethodPut, `{"key":"`+is.key(t)+`"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = do(asOwner(t), http.MethodPut, `{"key":"`+is.key(t)+`"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"state":"active"`)

	rec = do(asMember(t), http.MethodGet, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"licensee":"Acme"`)

	rec = do(asMember(t), http.MethodDelete, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = do(asOwner(t), http.MethodDelete, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"state":"none","licensee":"","instance_url":"","features":[]}`, rec.Body.String())

	s := newTestService(is, &fakeRepo{err: errors.New("disk")}, testURL)
	rec = httptest.NewRecorder()
	NewHandler(s).Routes().ServeHTTP(rec, httptest.NewRequestWithContext(asOwner(t), http.MethodGet, "/api/instance/license", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	rec = httptest.NewRecorder()
	NewHandler(s).Routes().ServeHTTP(rec, httptest.NewRequestWithContext(asOwner(t), http.MethodDelete, "/api/instance/license", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestInstanceUpdateTool(t *testing.T) {
	is := newIssuer(t)
	repo := &fakeRepo{}
	tool := MCPTools(newTestService(is, repo, testURL))[0]
	require.Equal(t, "instance_update", tool.Name)
	call := func(ctx context.Context, args string) (Status, error) {
		out, err := tool.Call(ctx, json.RawMessage(args))
		if err != nil {
			return Status{}, err
		}
		return out.(instanceSettings).License, nil
	}

	_, err := call(asMember(t), `{"license_key":"`+is.key(t)+`"}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = call(asOwner(t), `{"license_key":"nope"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	st, err := call(asOwner(t), `{"license_key":"`+is.key(t)+`"}`)
	require.NoError(t, err)
	assert.Equal(t, StateActive, st.State)

	st, err = call(asMember(t), `{}`)
	require.NoError(t, err, "an omitted field keeps the key and reads it back")
	assert.Equal(t, StateActive, st.State)
	assert.NotEmpty(t, repo.key)

	st, err = call(asOwner(t), `{"license_key":""}`)
	require.NoError(t, err)
	assert.Equal(t, StateNone, st.State)
	assert.Empty(t, repo.key)
}
