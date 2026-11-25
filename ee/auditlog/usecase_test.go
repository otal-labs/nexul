package auditlog

import (
	"context"
	"errors"
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
	err    error
}

func (r *fakeRepo) Get(context.Context) (Settings, error) { return r.st, r.err }
func (r *fakeRepo) Set(_ context.Context, s Settings, e ...eventbus.OutboxEvent) error {
	r.st = s
	r.events = append(r.events, e...)
	return nil
}
func (r *fakeRepo) Delete(_ context.Context, e ...eventbus.OutboxEvent) error {
	r.st = Settings{}
	r.events = append(r.events, e...)
	return nil
}

type fakeLicenses struct{ on bool }

func (l *fakeLicenses) Enabled(_ context.Context, f license.Feature) bool {
	return l.on && f == license.FeatureAudit
}

type fakeGate struct{ refuse permissions.Action }

func (g fakeGate) RequireAnywhere(_ context.Context, a permissions.Action) error {
	if a == g.refuse {
		return apperrs.ErrForbidden
	}
	return nil
}

type fakeSecrets map[string]error

func (s fakeSecrets) Read(_ context.Context, stored string) (string, error) {
	if err := s[stored]; err != nil {
		return "", err
	}
	return "resolved", nil
}

type harness struct {
	svc      *Service
	repo     *fakeRepo
	licenses *fakeLicenses
}

var now = time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC)

func newHarness(t *testing.T, unlocked bool) harness {
	t.Helper()
	h := harness{repo: &fakeRepo{}, licenses: &fakeLicenses{on: unlocked}}
	h.svc = NewService(Config{
		Repo: h.repo, Licenses: h.licenses, Gate: fakeGate{},
		Secrets: fakeSecrets{"op://vault/missing/secret": errors.New("item not found")},
		Now:     func() time.Time { return now },
	})
	return h
}

func asOwner(t *testing.T) context.Context {
	t.Helper()
	return identity.WithActor(t.Context(), identity.Actor{ID: "owner-1"})
}

func ptr[T any](v T) *T { return &v }

func TestUpdate_RefusesWhatTheSettingCannotHold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		patch Patch
		want  string
	}{
		{"days below 45", Patch{RetentionDays: ptr(44)}, "45 to 1095"},
		{"days above the cap", Patch{RetentionDays: ptr(1096)}, "45 to 1095"},
		{"zero days", Patch{RetentionDays: ptr(0)}, "45 to 1095"},
		{"negative days", Patch{RetentionDays: ptr(-30)}, "45 to 1095"},
		{"an http destination", Patch{StreamURL: ptr("http://logs.example.com/nexul")}, "must use https"},
		{"credentials in the destination", Patch{StreamURL: ptr("https://alice:hunter2@logs.example.com")}, "no user or password"},
		{"a destination with no host", Patch{StreamURL: ptr("https:///nexul")}, "absolute https URL"},
		{"a relative destination", Patch{StreamURL: ptr("logs.example.com")}, "absolute https URL"},
		{"a secret reference that does not resolve", Patch{StreamURL: ptr("https://logs.example.com"), StreamSecret: ptr("op://vault/missing/secret")}, "stream secret: could not be resolved"},
		{"a malformed secret reference", Patch{StreamURL: ptr("https://logs.example.com"), StreamSecret: ptr("op://vault")}, "not a valid secret reference"},
		{"a secret with no destination", Patch{StreamSecret: ptr("s3cret")}, "set stream_url too"},
		{"switched on with no destination", Patch{StreamEnabled: ptr(true)}, "needs a stream_url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, true)

			_, err := h.svc.Update(asOwner(t), tt.patch)

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), tt.want)
			assert.Empty(t, h.repo.events, "nothing is saved")
		})
	}
}

func TestUpdate_KeyLocked_IsRefusedAndSavesNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)

	_, err := h.svc.Update(asOwner(t), Patch{RetentionDays: ptr(365)})

	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "unlocks audit")
	assert.Equal(t, Settings{}, h.repo.st)
}

func TestUpdate_WithoutInstanceWrite_IsRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.svc.cfg.Gate = fakeGate{refuse: permissions.InstanceWrite}

	_, err := h.svc.Update(asOwner(t), Patch{RetentionDays: ptr(365)})
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = h.svc.Remove(asOwner(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestView_KeyLocked_StillReadsTheSavedDays(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)
	h.repo.st = Settings{RetentionDays: 400, StreamURL: "https://logs.example.com", StreamSecret: "s3cret", StreamEnabled: true}

	v, err := h.svc.View(asOwner(t))
	require.NoError(t, err)

	assert.Equal(t, View{RetentionDays: 400, StreamURL: "https://logs.example.com", SecretSet: true, StreamEnabled: true}, v,
		"locked, the setting stays and nothing streams")
}

func TestView_WithoutInstanceRead_IsRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.svc.cfg.Gate = fakeGate{refuse: permissions.InstanceRead}

	_, err := h.svc.View(asOwner(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
}

func TestUpdate_SavesTheDaysAndPublishesTheChangeWithoutTheSecret(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	v, err := h.svc.Update(asOwner(t), Patch{RetentionDays: ptr(365)})
	require.NoError(t, err)

	assert.Equal(t, Saved{View: View{Unlocked: true, RetentionDays: 365}}, v)
	assert.Equal(t, Settings{RetentionDays: 365, UpdatedBy: "owner-1", UpdatedAt: now}, h.repo.st)
	require.Len(t, h.repo.events, 1)
	assert.Equal(t, TopicSettingsChanged, h.repo.events[0].Topic)
	assert.Equal(t, SettingsChangedEvent{RetentionDays: 365, ChangedBy: "owner-1", ChangedAt: now}, h.repo.events[0].Payload)
}

func TestUpdate_ADestinationWithoutASecret_GetsOneShownOnce(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	v, err := h.svc.Update(asOwner(t), Patch{StreamURL: ptr(" https://logs.example.com/nexul "), StreamEnabled: ptr(true)})
	require.NoError(t, err)

	assert.Len(t, v.NewSecret, 64)
	assert.Equal(t, v.NewSecret, h.repo.st.StreamSecret)
	assert.True(t, v.SecretSet)
	assert.True(t, v.Streaming)
	assert.Equal(t, "https://logs.example.com/nexul", h.repo.st.StreamURL)
	assert.Equal(t, SettingsChangedEvent{RetentionDays: 45, StreamURL: "https://logs.example.com/nexul", StreamEnabled: true, ChangedBy: "owner-1", ChangedAt: now},
		h.repo.events[0].Payload, "the event never carries the secret")

	again, err := h.svc.Update(asOwner(t), Patch{RetentionDays: ptr(90)})
	require.NoError(t, err)
	assert.Empty(t, again.NewSecret, "a later save neither makes nor shows a secret")
	assert.Equal(t, v.NewSecret, h.repo.st.StreamSecret)
}

func TestUpdate_TheOwnersSecretOrReference_IsKeptAsTyped(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	v, err := h.svc.Update(asOwner(t), Patch{StreamURL: ptr("http://127.0.0.1:9000/hook"), StreamSecret: ptr("op://vault/item/secret")})
	require.NoError(t, err)

	assert.Empty(t, v.NewSecret)
	assert.Equal(t, "op://vault/item/secret", h.repo.st.StreamSecret)
	assert.False(t, v.StreamEnabled, "saving a destination does not switch the stream on")
}

func TestUpdate_AnOmittedFieldKeepsItsValue(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st = Settings{RetentionDays: 400, StreamURL: "https://logs.example.com", StreamSecret: "s3cret", StreamEnabled: true}

	_, err := h.svc.Update(asOwner(t), Patch{StreamEnabled: ptr(false)})
	require.NoError(t, err)

	assert.Equal(t, 400, h.repo.st.RetentionDays)
	assert.Equal(t, "https://logs.example.com", h.repo.st.StreamURL)
	assert.Equal(t, "s3cret", h.repo.st.StreamSecret)
	assert.False(t, h.repo.st.StreamEnabled)
}

func TestUpdate_AnEmptyStreamURL_RemovesTheDestination(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st = Settings{RetentionDays: 400, StreamURL: "https://logs.example.com", StreamSecret: "s3cret", StreamEnabled: true}

	v, err := h.svc.Update(asOwner(t), Patch{StreamURL: ptr("")})
	require.NoError(t, err)

	assert.Equal(t, Settings{RetentionDays: 400, UpdatedBy: "owner-1", UpdatedAt: now}, h.repo.st)
	assert.False(t, v.Streaming)
}

func TestUpdate_ARepoThatCannotBeRead_IsReturned(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.err = errors.New("database is locked")

	_, err := h.svc.Update(asOwner(t), Patch{RetentionDays: ptr(90)})
	require.ErrorIs(t, err, h.repo.err)
	_, err = h.svc.View(asOwner(t))
	require.ErrorIs(t, err, h.repo.err)
	_, err = h.svc.Remove(asOwner(t))
	require.ErrorIs(t, err, h.repo.err)
	_, err = h.svc.RetentionDays(t.Context())
	require.ErrorIs(t, err, h.repo.err)
}

func TestRetentionDays_SavedDaysHoldWhetherOrNotTheKeyUnlocksAudit(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	days, err := h.svc.RetentionDays(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 45, days, "unset keeps the free 45 days")

	h.repo.st.RetentionDays = 730
	h.licenses.on = false
	days, err = h.svc.RetentionDays(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 730, days, "a lapsed key keeps what the feature made (ADR 0184)")
}

func TestRemove_WorksLockedAndReturnsTo45Days(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)
	h.repo.st = Settings{RetentionDays: 730, StreamURL: "https://logs.example.com", StreamSecret: "s3cret"}

	v, err := h.svc.Remove(asOwner(t))
	require.NoError(t, err)

	assert.Equal(t, View{RetentionDays: 45}, v)
	assert.Equal(t, Settings{}, h.repo.st)
	require.Len(t, h.repo.events, 1)
	assert.Equal(t, SettingsChangedEvent{RetentionDays: 45, Removed: true, ChangedBy: "owner-1", ChangedAt: now}, h.repo.events[0].Payload)
	days, err := h.svc.RetentionDays(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 45, days)

	_, err = h.svc.Remove(asOwner(t))
	require.NoError(t, err)
	assert.Len(t, h.repo.events, 1, "removing nothing publishes nothing")
}
