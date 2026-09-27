package dns

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// twoAccounts is a token that reaches two Cloudflare accounts, each owning one zone, with a tunnel provider per account.
type twoAccounts struct {
	svc     *Service
	repo    *fakeRepo
	dns     *fakeProvider
	tunnels map[string]*fakeTunnelProvider
	asked   []string
}

func newTwoAccounts(t *testing.T) *twoAccounts {
	t.Helper()
	ta := &twoAccounts{
		repo: newFakeRepo(),
		dns:  newFakeProvider(),
		tunnels: map[string]*fakeTunnelProvider{
			"acct-law":  {account: "acct-law", tunnels: map[string]*Tunnel{}},
			"acct-otal": {account: "acct-otal", tunnels: map[string]*Tunnel{}},
		},
	}
	ta.dns.zones = []Zone{
		{ID: "z-law", Name: "law.example", AccountID: "acct-law", AccountName: "Law"},
		{ID: "z-otal", Name: "otal.dev", AccountID: "acct-otal", AccountName: "Otal"},
	}
	ta.svc = NewService(Config{
		Repo:          ta.repo,
		Provider:      ta.dns,
		EncryptionKey: testKey(),
		Settings:      &fakeSettings{instanceURL: "https://nexul.otal.dev"},
		Tokens:        &fakeTokenProvider{token: "at"},
		NewTunnelProvider: func(_ context.Context, _, accountID string) (TunnelProvider, error) {
			ta.asked = append(ta.asked, accountID)
			return ta.tunnels[accountID], nil
		},
		Now: time.Now,
	})
	return ta
}

func TestCreateTunnel_Accounts(t *testing.T) {
	t.Run("several accounts and no choice names them instead of guessing", func(t *testing.T) {
		ta := newTwoAccounts(t)
		_, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod"})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.ErrorContains(t, err, "Law (acct-law), Otal (acct-otal)")
		assert.Empty(t, ta.asked, "nothing is created at Cloudflare")
	})
	t.Run("the chosen account is where the tunnel is created", func(t *testing.T) {
		ta := newTwoAccounts(t)
		got, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod", AccountID: "acct-otal"})
		require.NoError(t, err)
		assert.Equal(t, "acct-otal", got.AccountID)
		assert.Len(t, ta.tunnels["acct-otal"].tunnels, 1)
		assert.Empty(t, ta.tunnels["acct-law"].tunnels)
	})
	t.Run("a single account needs no choice", func(t *testing.T) {
		ta := newTwoAccounts(t)
		ta.dns.zones = ta.dns.zones[1:]
		got, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod"})
		require.NoError(t, err)
		assert.Equal(t, "acct-otal", got.AccountID)
	})
	t.Run("a retry reuses the same-named tunnel only in the same account", func(t *testing.T) {
		ta := newTwoAccounts(t)
		first, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod", AccountID: "acct-law"})
		require.NoError(t, err)
		again, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod", AccountID: "acct-law"})
		require.NoError(t, err)
		assert.Equal(t, first.ID, again.ID)
		other, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod", AccountID: "acct-otal"})
		require.NoError(t, err)
		assert.Equal(t, "acct-otal", other.AccountID)
	})
}

func TestTunnelOperations_UseTheTunnelsAccount(t *testing.T) {
	t.Run("a tracked tunnel is read in its own account", func(t *testing.T) {
		ta := newTwoAccounts(t)
		created, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod", AccountID: "acct-otal"})
		require.NoError(t, err)
		ta.asked = nil
		_, err = ta.svc.TunnelStatus(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"acct-otal"}, ta.asked)
	})
	t.Run("an untracked tunnel is found in whichever account has it", func(t *testing.T) {
		ta := newTwoAccounts(t)
		ta.tunnels["acct-otal"].tunnels["found"] = &Tunnel{ID: "found", Name: "running", AccountID: "acct-otal"}
		got, err := ta.svc.ComputerTunnelStatus(t.Context(), "found")
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Equal(t, []string{"acct-law", "acct-otal"}, ta.asked, "the account that answers is the one used")
	})
	t.Run("a computer tunnel is created in the instance zone's account", func(t *testing.T) {
		ta := newTwoAccounts(t)
		tp, err := ta.svc.tunnelProviderIn(t.Context(), ta.mustInstanceZone(t).AccountID)
		require.NoError(t, err)
		assert.Same(t, ta.tunnels["acct-otal"], tp)
	})
}

func (ta *twoAccounts) mustInstanceZone(t *testing.T) Zone {
	t.Helper()
	z, err := ta.svc.instanceZone(t.Context(), ta.dns)
	require.NoError(t, err)
	return z
}

func TestRouteTunnelHostname_RefusesAnotherAccountsZone(t *testing.T) {
	ta := newTwoAccounts(t)
	created, err := ta.svc.CreateTunnel(t.Context(), CreateTunnelInput{Name: "prod", AccountID: "acct-law"})
	require.NoError(t, err)

	_, err = ta.svc.RouteTunnelHostname(t.Context(), RouteTunnelInput{
		TunnelID: created.ID, Hostname: "nexul.otal.dev", ZoneID: "z-otal", Zone: "otal.dev", Service: "http://x:1",
	})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "otal.dev belongs to the Cloudflare account Otal")
	records, err := ta.dns.ListRecords(t.Context(), "z-otal")
	require.NoError(t, err)
	assert.Empty(t, records, "no dead CNAME is left behind")

	_, err = ta.svc.RouteTunnelHostname(t.Context(), RouteTunnelInput{
		TunnelID: created.ID, Hostname: "app.law.example", ZoneID: "z-law", Zone: "law.example", Service: "http://x:1",
	})
	require.NoError(t, err)
}
