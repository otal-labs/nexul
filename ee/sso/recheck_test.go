package sso

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var recheckNow = time.Unix(1_000_000, 0)

// withGrants seeds grants last confirmed an hour ago, the way a first hourly pass finds them.
func withGrants(h *harness, users ...string) {
	h.svc.cfg.Now = func() time.Time { return recheckNow }
	for _, u := range users {
		h.repo.grants[u] = StoredGrant{UserID: u, RefreshToken: "rt-" + u}
		h.repo.checked[u] = recheckNow.Add(-time.Hour)
	}
}

func TestRecheck_ARefusedGrant_EndsThatUsersSessionsAndLeavesTheOthers(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	withGrants(h, "alice", "bob")
	h.repo.sessions["alice"] = 2
	h.idp.refresh = map[string]error{"rt-alice": errGrantRefused}

	ended, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Equal(t, 2, ended)
	assert.Equal(t, []string{"alice"}, h.repo.refused)
	assert.NotContains(t, h.repo.grants, "alice")
	assert.Equal(t, recheckNow, h.repo.checked["bob"], "the person the provider still knows is confirmed")
}

func TestRecheck_ASuccess_StoresTheRotatedTokenAndBumpsTheCheck(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	withGrants(h, "alice", "bob")
	h.idp.rotated = map[string]string{"rt-alice": "rt-alice-2"}

	ended, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Zero(t, ended)
	assert.Equal(t, "rt-alice-2", h.repo.grants["alice"].RefreshToken)
	assert.Equal(t, "rt-bob", h.repo.grants["bob"].RefreshToken, "a provider that does not rotate leaves the token")
	assert.Equal(t, recheckNow, h.repo.checked["alice"])
	assert.Equal(t, recheckNow, h.repo.checked["bob"])
}

func TestRecheck_AGrantCheckedRecently_IsNotAskedAbout(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	withGrants(h, "alice")
	h.repo.checked["alice"] = recheckNow.Add(-time.Minute)

	_, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Empty(t, h.idp.refreshCalls)
}

func TestRecheck_AProviderThatCannotAnswer_ChangesNothingAndStopsThePass(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	withGrants(h, "alice", "bob")
	h.repo.sessions["bob"] = 1
	h.idp.refresh = map[string]error{"rt-alice": fmt.Errorf("%w: down", errProviderUnavailable)}

	ended, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

	require.ErrorIs(t, err, errProviderUnavailable)
	assert.Zero(t, ended)
	assert.Equal(t, []string{"rt-alice"}, h.idp.refreshCalls, "the pass stops at the first answer it cannot read")
	assert.Empty(t, h.repo.refused)
	assert.Len(t, h.repo.grants, 2)
	assert.Equal(t, recheckNow.Add(-time.Hour), h.repo.checked["alice"])
}

func TestRecheck_AnUnreachableProvider_StopsBeforeAnyGrantIsAsked(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	withGrants(h, "alice")
	h.svc.cfg.Connect = func(context.Context, ConnectConfig) (IdP, error) { return nil, errors.New("no route") }

	_, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

	require.Error(t, err)
	assert.Empty(t, h.repo.refused)
	assert.Contains(t, h.repo.grants, "alice")
}

func TestRecheck_UnlicensedOrUnconfigured_DoesNothing(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		licensed    bool
		unconfigure bool
	}{
		"no license":  {false, false},
		"no provider": {true, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tt.licensed)
			withGrants(h, "alice")
			h.idp.refresh = map[string]error{"rt-alice": errGrantRefused}
			if tt.unconfigure {
				h.repo.st = Settings{}
			}

			ended, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

			require.NoError(t, err)
			assert.Zero(t, ended)
			assert.Empty(t, h.idp.refreshCalls)
			assert.Contains(t, h.repo.grants, "alice")
		})
	}
}

func TestRecheck_MoreGrantsThanOneBatch_AreAllChecked(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	users := make([]string, 0, recheckBatch*2+1)
	for i := range recheckBatch*2 + 1 {
		users = append(users, fmt.Sprintf("u-%03d", i))
	}
	withGrants(h, users...)

	_, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Len(t, h.idp.refreshCalls, len(users))
}

func TestRunRecheckLoop_StopsOnContextCancel(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		h.svc.RunRecheckLoop(ctx, time.Hour, slog.New(slog.DiscardHandler))
		close(done)
	}()
	cancel()
	<-done
}

func TestComplete_AnExchangeWithARefreshToken_StoresItForThatUser(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.svc.cfg.Now = func() time.Time { return recheckNow }
	h.accounts.bySubject[issuerKey] = Account{UserID: "u-linked", Status: "active"}
	h.idp.refreshToken = "rt-new"

	_, err := h.svc.Complete(t.Context(), "state", "code")

	require.NoError(t, err)
	assert.Equal(t, StoredGrant{UserID: "u-linked", RefreshToken: "rt-new"}, h.repo.grants["u-linked"])
	assert.Equal(t, recheckNow, h.repo.checked["u-linked"])
}

func TestComplete_AnExchangeWithoutARefreshToken_LeavesTheOldGrant(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.accounts.bySubject[issuerKey] = Account{UserID: "u-linked", Status: "active"}
	h.repo.grants["u-linked"] = StoredGrant{UserID: "u-linked", RefreshToken: "rt-old"}

	_, err := h.svc.Complete(t.Context(), "state", "code")

	require.NoError(t, err)
	assert.Equal(t, "rt-old", h.repo.grants["u-linked"].RefreshToken)
}

func TestComplete_ARefusedAccount_StoresNoGrant(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.accounts.bySubject[issuerKey] = Account{UserID: "u-1", Status: "disabled"}
	h.idp.refreshToken = "rt-new"

	_, err := h.svc.Complete(t.Context(), "state", "code")

	require.ErrorIs(t, err, errDisabled)
	assert.Empty(t, h.repo.grants)
}

func TestRemove_DropsEveryGrant(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	withGrants(h, "alice")

	_, err := h.svc.Remove(asActor(t))

	require.NoError(t, err)
	assert.Empty(t, h.repo.grants)
}

func TestRecheck_AnUnreadableGrant_IsDroppedWithItsSessionsKeptAndThePassContinues(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	withGrants(h, "alice", "bob")
	h.repo.grants["alice"] = StoredGrant{UserID: "alice"}
	h.repo.sessions["alice"] = 2

	ended, err := h.svc.recheck(t.Context(), slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	assert.Zero(t, ended)
	assert.Empty(t, h.repo.refused, "no session ends over a token nobody could read")
	assert.NotContains(t, h.repo.grants, "alice")
	assert.Equal(t, []string{"rt-bob"}, h.idp.refreshCalls, "the next grant is still checked")
	assert.Equal(t, recheckNow, h.repo.checked["bob"])
}
