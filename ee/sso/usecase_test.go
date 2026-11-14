package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/license"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type fakeRepo struct {
	st     Settings
	events []eventbus.OutboxEvent
	// kept is the users TurnOnOnly was told to keep signed in; nil until it runs.
	kept   []string
	grants map[string]StoredGrant
	// checked is when each grant was last confirmed.
	checked map[string]time.Time
	// refused is the users whose grant RefuseGrant was told to end, with the sessions each had.
	refused  []string
	sessions map[string]int
}

func (r *fakeRepo) Get(context.Context) (Settings, error) { return r.st, nil }
func (r *fakeRepo) Set(_ context.Context, s Settings, e ...eventbus.OutboxEvent) error {
	r.st = s
	r.events = append(r.events, e...)
	return nil
}
func (r *fakeRepo) TurnOnOnly(_ context.Context, s Settings, keepUsers []string, e ...eventbus.OutboxEvent) error {
	r.st = s
	r.kept = append([]string{}, keepUsers...)
	r.events = append(r.events, e...)
	return nil
}
func (r *fakeRepo) SaveGrant(_ context.Context, userID, token string, at time.Time) error {
	r.grants[userID] = StoredGrant{UserID: userID, RefreshToken: token}
	r.checked[userID] = at
	return nil
}
func (r *fakeRepo) StaleGrants(_ context.Context, before time.Time, limit int) ([]StoredGrant, error) {
	out := []StoredGrant{}
	for _, id := range slices.Sorted(maps.Keys(r.grants)) {
		if r.checked[id].Before(before) && len(out) < limit {
			out = append(out, r.grants[id])
		}
	}
	return out, nil
}
func (r *fakeRepo) ConfirmGrant(_ context.Context, userID, token string, at time.Time) error {
	if token != "" {
		r.grants[userID] = StoredGrant{UserID: userID, RefreshToken: token}
	}
	r.checked[userID] = at
	return nil
}
func (r *fakeRepo) DropGrant(_ context.Context, userID string) error {
	delete(r.grants, userID)
	return nil
}
func (r *fakeRepo) RefuseGrant(_ context.Context, userID string) (int, error) {
	delete(r.grants, userID)
	r.refused = append(r.refused, userID)
	return r.sessions[userID], nil
}
func (r *fakeRepo) Delete(_ context.Context, e ...eventbus.OutboxEvent) error {
	r.st = Settings{}
	r.grants = map[string]StoredGrant{}
	r.events = append(r.events, e...)
	return nil
}

type fakeAccounts struct {
	bySubject map[string]Account
	byEmail   map[string]Account
	roles     map[string]bool
	linked    map[string]Identity
	created   map[string]string
}

func newAccounts() *fakeAccounts {
	return &fakeAccounts{
		bySubject: map[string]Account{}, byEmail: map[string]Account{}, roles: map[string]bool{"member": true},
		linked: map[string]Identity{}, created: map[string]string{},
	}
}

func (a *fakeAccounts) BySubject(_ context.Context, key string) (Account, error) {
	if acct, ok := a.bySubject[key]; ok {
		return acct, nil
	}
	return Account{}, apperrs.ErrNotFound
}

func (a *fakeAccounts) ByEmail(_ context.Context, email string) (Account, error) {
	if acct, ok := a.byEmail[email]; ok {
		return acct, nil
	}
	return Account{}, apperrs.ErrNotFound
}

func (a *fakeAccounts) Link(_ context.Context, userID string, id Identity) error {
	a.linked[userID] = id
	return nil
}

func (a *fakeAccounts) Create(_ context.Context, userID string, id Identity, roleID string, _ time.Time) error {
	a.created[userID] = roleID
	a.linked[userID] = id
	return nil
}

func (a *fakeAccounts) CanJoinAs(_ context.Context, roleID string) (bool, error) {
	return a.roles[roleID], nil
}

type fakeLicenses struct{ on bool }

func (l fakeLicenses) Enabled(_ context.Context, f license.Feature) bool {
	return l.on && f == license.FeatureSSO
}

type fakeGate struct{ err error }

func (g fakeGate) RequireAnywhere(context.Context, permissions.Action) error { return g.err }

type fakeIdP struct {
	claims       Claims
	refreshToken string
	err          error
	errOnce      error
	calls        int
	// refresh answers a Refresh by the token it is given: the new token, or the error; a token not listed returns itself.
	refresh      map[string]error
	rotated      map[string]string
	refreshCalls []string
}

func (f *fakeIdP) AuthURL(state string) string { return "https://idp.test/authorize?state=" + state }
func (f *fakeIdP) Exchange(context.Context, string, string) (Grant, error) {
	f.calls++
	if f.errOnce != nil && f.calls == 1 {
		return Grant{}, f.errOnce
	}
	return Grant{Claims: f.claims, RefreshToken: f.refreshToken}, f.err
}
func (f *fakeIdP) Refresh(_ context.Context, token string) (string, error) {
	f.refreshCalls = append(f.refreshCalls, token)
	if err := f.refresh[token]; err != nil {
		return "", err
	}
	if next, ok := f.rotated[token]; ok {
		return next, nil
	}
	return token, nil
}

type fakeSecrets struct {
	values  map[string]string
	refresh string
}

func (s fakeSecrets) Value(_ context.Context, stored string) (string, error) {
	if v, ok := s.values[stored]; ok {
		return v, nil
	}
	return stored, nil
}
func (s fakeSecrets) Read(ctx context.Context, stored string) (string, error) {
	if _, ok := s.values[stored]; !ok {
		return "", errors.New("item not found")
	}
	return s.Value(ctx, stored)
}
func (s fakeSecrets) Refresh(context.Context, string) (string, bool) {
	return s.refresh, s.refresh != ""
}

type harness struct {
	svc      *Service
	repo     *fakeRepo
	accounts *fakeAccounts
	idp      *fakeIdP
	// secrets seen by Connect, in order.
	connected []ConnectConfig
}

const verifiedSecret = "s3cret"

func newHarness(t *testing.T, licensed bool) *harness {
	t.Helper()
	h := &harness{
		repo: &fakeRepo{grants: map[string]StoredGrant{}, checked: map[string]time.Time{}, sessions: map[string]int{}, st: Settings{
			IssuerURL: "https://login.example.com", ClientID: "client-1", ClientSecret: verifiedSecret, DisplayName: "Acme SSO", JoinRoleID: "member",
		}},
		accounts: newAccounts(),
		idp:      &fakeIdP{claims: Claims{Subject: "sub-1", Email: "alice@example.com", EmailVerified: true, Name: "Alice"}},
	}
	h.svc = NewService(Config{
		Repo: h.repo, Accounts: h.accounts, Licenses: fakeLicenses{on: licensed}, Gate: fakeGate{},
		InstanceURL:     func(context.Context) (string, error) { return "https://nexul.example.com/", nil },
		OwnerHasPasskey: func(context.Context) (bool, error) { return true, nil },
		InstanceOwners:  func(context.Context) ([]string, error) { return []string{"owner-1"}, nil },
		Connect: func(_ context.Context, c ConnectConfig) (IdP, error) {
			h.connected = append(h.connected, c)
			return h.idp, nil
		},
	})
	return h
}

func asActor(t *testing.T) context.Context {
	t.Helper()
	return identity.WithActor(t.Context(), identity.Actor{ID: "owner-1"})
}

const issuerKey = "https://login.example.com#sub-1"

func TestComplete_VerifiedEmailOfAnExistingAccount_LinksItAndSignsIn(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-alice", Status: "active"}

	userID, err := h.svc.Complete(t.Context(), "state", "code")

	require.NoError(t, err)
	assert.Equal(t, "u-alice", userID)
	assert.Equal(t, Identity{Key: issuerKey, Email: "alice@example.com", Name: "Alice"}, h.accounts.linked["u-alice"])
	assert.Empty(t, h.accounts.created)
}

func TestComplete_AKnownSubject_SignsInWithoutMatchingByEmail(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.accounts.bySubject[issuerKey] = Account{UserID: "u-linked", Status: "active"}
	h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-someone-else", Status: "active"}

	userID, err := h.svc.Complete(t.Context(), "state", "code")

	require.NoError(t, err)
	assert.Equal(t, "u-linked", userID)
	assert.Empty(t, h.accounts.linked, "nothing is linked again")
}

func TestComplete_NoAccount_CreatesOneJoiningAsTheChosenRole(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	userID, err := h.svc.Complete(t.Context(), "state", "code")

	require.NoError(t, err)
	assert.Equal(t, "member", h.accounts.created[userID])
	assert.Equal(t, "alice@example.com", h.accounts.linked[userID].Email)
}

func TestComplete_AnEmailTheProviderDoesNotVouchFor_MakesAndLinksNothing(t *testing.T) {
	t.Parallel()
	for name, claims := range map[string]Claims{
		"unverified":    {Subject: "sub-1", Email: "alice@example.com", EmailVerified: false},
		"no email":      {Subject: "sub-1", EmailVerified: true},
		"blank subject": {Email: "alice@example.com", EmailVerified: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, true)
			h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-alice", Status: "active"}
			h.idp.claims = claims

			_, err := h.svc.Complete(t.Context(), "state", "code")

			require.ErrorIs(t, err, errUnverified)
			assert.Empty(t, h.accounts.linked)
			assert.Empty(t, h.accounts.created)
		})
	}
}

func TestComplete_AnAccountThatMayNotSignIn_IsRefusedWhicheverWayItIsFound(t *testing.T) {
	t.Parallel()
	for status, want := range map[string]error{"disabled": errDisabled, "removed": errRemoved} {
		t.Run("by subject, "+status, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, true)
			h.accounts.bySubject[issuerKey] = Account{UserID: "u-1", Status: status}
			_, err := h.svc.Complete(t.Context(), "state", "code")
			require.ErrorIs(t, err, want)
		})
		t.Run("by email, "+status, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, true)
			h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-1", Status: status}
			_, err := h.svc.Complete(t.Context(), "state", "code")
			require.ErrorIs(t, err, want)
			assert.Empty(t, h.accounts.linked, "a refused account is not linked")
		})
	}
}

func TestComplete_AJoinRoleNobodyMayHoldAnyMore_CreatesNoAccount(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.accounts.roles["member"] = false

	_, err := h.svc.Complete(t.Context(), "state", "code")

	require.ErrorIs(t, err, errNotReady)
	assert.Empty(t, h.accounts.created)
}

func TestStartAndComplete_WithoutALicense_AreRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)
	_, _, err := h.svc.Start(t.Context())
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = h.svc.Complete(t.Context(), "state", "code")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, h.connected, "the provider is never contacted")
}

func TestStart_ReadsTheSecretReferenceAndBuildsTheRedirectURI(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st.ClientSecret = "op://vault/item/field"
	h.svc.cfg.Secrets = fakeSecrets{values: map[string]string{"op://vault/item/field": "resolved"}}

	target, state, err := h.svc.Start(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "https://idp.test/authorize?state="+state, target)
	assert.Len(t, state, 32)
	assert.Equal(t, ConnectConfig{Issuer: "https://login.example.com", ClientID: "client-1", ClientSecret: "resolved", RedirectURI: "https://nexul.example.com/auth/oidc/callback"}, h.connected[0],
		"a sign-in may use the cached discovery")
}

func TestComplete_AProviderThatRejectsTheClient_RetriesOnceWithTheReReadSecret(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st.ClientSecret = "op://vault/item/field"
	h.svc.cfg.Secrets = fakeSecrets{values: map[string]string{"op://vault/item/field": "stale"}, refresh: "fresh"}
	h.idp.errOnce = errClientRejected

	_, err := h.svc.Complete(t.Context(), "state", "code")

	require.NoError(t, err)
	require.Len(t, h.connected, 2)
	assert.Equal(t, "fresh", h.connected[1].ClientSecret)
}

func TestComplete_ARejectedClientWithNothingToReRead_IsNotRetried(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.idp.err = errClientRejected
	_, err := h.svc.Complete(t.Context(), "state", "code")
	require.ErrorIs(t, err, errClientRejected)
	assert.Equal(t, 1, h.idp.calls)
}

func patch(fields map[string]any) Patch {
	var p Patch
	b, _ := json.Marshal(fields)
	_ = json.Unmarshal(b, &p)
	return p
}

func TestUpdate_Refusals_StoreNothing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		licensed bool
		gate     error
		start    Settings
		patch    map[string]any
		connect  error
		passkey  bool
		want     error
	}{
		{"without a license", false, nil, Settings{}, full(), nil, true, apperrs.ErrForbidden},
		{"without instance:write", true, apperrs.ErrForbidden, Settings{}, full(), nil, true, apperrs.ErrForbidden},
		{"an incomplete provider", true, nil, Settings{}, map[string]any{"issuer_url": "https://login.example.com"}, nil, true, apperrs.ErrInvalid},
		{"an insecure issuer", true, nil, Settings{}, with(full(), "issuer_url", "http://login.example.com"), nil, true, apperrs.ErrInvalid},
		{"a provider that does not answer", true, nil, Settings{}, full(), fmt.Errorf("%w: cannot reach", apperrs.ErrInvalid), true, apperrs.ErrInvalid},
		{"the Owner role", true, nil, Settings{}, with(full(), "join_role_id", "owner"), nil, true, apperrs.ErrInvalid},
		{"a name too long for a button", true, nil, Settings{}, with(full(), "display_name", "An extremely long identity provider name here"), nil, true, apperrs.ErrInvalid},
		{"a secret reference that does not resolve", true, nil, Settings{}, with(full(), "client_secret", "op://vault/missing/field"), nil, true, apperrs.ErrInvalid},
		{"only, with no Owner passkey to fall back on", true, nil, Settings{}, with(full(), "only", true), nil, false, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tt.licensed)
			h.repo.st = tt.start
			h.accounts.roles["owner"] = false
			h.svc.cfg.Gate = fakeGate{err: tt.gate}
			h.svc.cfg.Secrets = fakeSecrets{values: map[string]string{}}
			h.svc.cfg.OwnerHasPasskey = func(context.Context) (bool, error) { return tt.passkey, nil }
			h.svc.cfg.Connect = func(context.Context, ConnectConfig) (IdP, error) { return h.idp, tt.connect }

			_, err := h.svc.Update(asActor(t), patch(tt.patch))

			require.ErrorIs(t, err, tt.want)
			assert.Equal(t, tt.start, h.repo.st)
			assert.Empty(t, h.repo.events)
		})
	}
}

func full() map[string]any {
	return map[string]any{
		"issuer_url": "https://login.example.com", "client_id": "client-1", "client_secret": verifiedSecret,
		"display_name": "Acme SSO", "join_role_id": "member",
	}
}

func with(m map[string]any, key string, value any) map[string]any {
	m[key] = value
	return m
}

func TestUpdate_SavesTheProviderAndPublishesAChangeWithoutTheSecret(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st = Settings{}

	v, err := h.svc.Update(asActor(t), patch(full()))

	require.NoError(t, err)
	assert.True(t, v.Active)
	assert.Equal(t, "https://nexul.example.com/auth/oidc/callback", v.RedirectURI)
	assert.Equal(t, verifiedSecret, h.repo.st.ClientSecret)
	require.Len(t, h.repo.events, 1)
	assert.Equal(t, TopicChanged, h.repo.events[0].Topic)
	payload, _ := json.Marshal(h.repo.events[0].Payload)
	assert.NotContains(t, string(payload), verifiedSecret)
	assert.Equal(t, "owner-1", h.repo.st.UpdatedBy)
}

func TestUpdate_AnOmittedFieldKeepsItsValue_AndABlankSecretKeepsTheSavedSecret(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	saved := h.repo.st

	_, err := h.svc.Update(asActor(t), patch(map[string]any{"display_name": "Acme", "client_secret": "  "}))

	require.NoError(t, err)
	want := saved
	want.DisplayName = "Acme"
	want.UpdatedBy, want.UpdatedAt = h.repo.st.UpdatedBy, h.repo.st.UpdatedAt
	assert.Equal(t, want, h.repo.st)
	assert.Empty(t, h.connected, "an unchanged issuer and secret are not reached again")
}

func TestUpdate_OnlyOn_NeedsTheOwnerPasskeyOnlyWhenTurningItOn(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st.Only = true
	h.svc.cfg.OwnerHasPasskey = func(context.Context) (bool, error) { return false, nil }

	_, err := h.svc.Update(asActor(t), patch(map[string]any{"display_name": "Acme"}))
	require.NoError(t, err, "an instance already SSO-only can still be edited")
	v, err := h.svc.Update(asActor(t), patch(map[string]any{"only": false}))
	require.NoError(t, err)
	assert.False(t, v.Only)
}

func TestUpdate_TurningOnlyOn_SignsOutEveryoneButSSOSessionsAndInstanceOwners(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	_, err := h.svc.Update(asActor(t), patch(map[string]any{"display_name": "Acme"}))
	require.NoError(t, err)
	assert.Nil(t, h.repo.kept, "a save that leaves the switch off signs nobody out")

	v, err := h.svc.Update(asActor(t), patch(map[string]any{"only": true}))
	require.NoError(t, err)
	assert.True(t, v.Only)
	assert.Equal(t, []string{"owner-1"}, h.repo.kept)

	h.repo.kept = nil
	_, err = h.svc.Update(asActor(t), patch(map[string]any{"display_name": "Acme SSO"}))
	require.NoError(t, err)
	assert.Nil(t, h.repo.kept, "a save while the switch is already on signs nobody out again")
}

func TestUpdate_TurningOnlyOn_FailsWithoutSavingWhenTheOwnersCannotBeRead(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	saved := h.repo.st
	h.svc.cfg.InstanceOwners = func(context.Context) ([]string, error) { return nil, errors.New("disk gone") }

	_, err := h.svc.Update(asActor(t), patch(map[string]any{"only": true}))

	require.Error(t, err)
	assert.Equal(t, saved, h.repo.st)
	assert.Empty(t, h.repo.events)
}

func TestRemove_WorksWithoutALicense_SoALapsedKeyNeverTrapsAProvider(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)

	v, err := h.svc.Remove(asActor(t))

	require.NoError(t, err)
	assert.False(t, v.Configured)
	assert.Equal(t, Settings{}, h.repo.st)
	require.Len(t, h.repo.events, 1)
}

func TestPublic_ShowsTheButtonOnlyWhileLicensedAndConfigured(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		licensed bool
		st       func(*Settings)
		want     Public
	}{
		{"licensed and configured", true, func(*Settings) {}, Public{Active: true, Name: "Acme SSO"}},
		{"the only sign-in", true, func(s *Settings) { s.Only = true }, Public{Active: true, Name: "Acme SSO", Only: true}},
		{"a lapsed license hides it, and the switch with it", false, func(s *Settings) { s.Only = true }, Public{}},
		{"a half-set provider", true, func(s *Settings) { s.ClientID = "" }, Public{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tt.licensed)
			tt.st(&h.repo.st)
			assert.Equal(t, tt.want, h.svc.Public(t.Context()))
			assert.Equal(t, tt.want.Only, h.svc.OnlySSO(t.Context()))
		})
	}
}

func TestView_KeepsTheSavedSettingsWhenTheLicenseLapses_AndNeverCarriesTheSecret(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)

	v, err := h.svc.View(asActor(t))

	require.NoError(t, err)
	assert.True(t, v.Configured)
	assert.False(t, v.Unlocked)
	assert.False(t, v.Active)
	assert.True(t, v.SecretSet)
	b, _ := json.Marshal(v)
	assert.NotContains(t, string(b), verifiedSecret)
}

func TestCheck_RunsOneNamedCheckAndStoresNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.svc.cfg.Secrets = fakeSecrets{values: map[string]string{"op://vault/item/field": "x"}}

	detail, err := h.svc.Check(asActor(t), "issuer", patch(map[string]any{"issuer_url": "https://idp.example.com"}))
	require.NoError(t, err)
	assert.Equal(t, "Found the sign-in endpoints at idp.example.com", detail)
	require.Len(t, h.connected, 1)
	assert.True(t, h.connected[0].Fresh, "a check reaches the provider now, never a cached discovery")

	detail, err = h.svc.Check(asActor(t), "secret", patch(map[string]any{"client_secret": "op://vault/item/field"}))
	require.NoError(t, err)
	assert.Equal(t, "Read from the secrets manager", detail)

	_, err = h.svc.Check(asActor(t), "secret", patch(map[string]any{"client_secret": "op://vault/missing/field"}))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = h.svc.Check(asActor(t), "issuer", patch(map[string]any{"issuer_url": "http://idp.example.com"}))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = h.svc.Check(asActor(t), "role", Patch{})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Empty(t, h.repo.events)

	h.svc.cfg.Licenses = fakeLicenses{on: false}
	_, err = h.svc.Check(asActor(t), "issuer", Patch{})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}
