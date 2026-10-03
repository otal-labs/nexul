package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/t3rpc"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

var switchTestKey = []byte("0123456789abcdef0123456789abcdef")

// pairingOverRegistry wires pairing to the production harness registry against a fake T3 Code, on real SQLite.
func pairingOverRegistry(t *testing.T, f *t3rpctest.Server) (*pairing.Service, *storage.Store) {
	t.Helper()
	s := storage.New(mentionsTestDB(t), switchTestKey)
	seedMentionsUser(t, s, "u1")
	var svc *pairing.Service
	harnesses := harnessRegistry(t3rpc.Options{HTTPClient: f.Client(), RPCTimeout: 5 * time.Second}, func(ctx context.Context, sess harness.Session, to harness.Kind) error {
		return svc.SwitchHarness(ctx, sess, to)
	})
	svc = pairing.NewService(pairing.Config{Repo: s.Pairing, Harnesses: harnesses, EncryptionKey: switchTestKey})
	return svc, s
}

func countSwitched(t *testing.T, s *storage.Store) int {
	t.Helper()
	entries, err := s.Outbox.Unpublished(t.Context(), 50)
	require.NoError(t, err)
	n := 0
	for _, e := range entries {
		if e.Topic == pairing.TopicHarnessSwitched {
			n++
		}
	}
	return n
}

func TestIntegration_FreshPairOnProtocol2_StoresTheNewKindWithOneToken(t *testing.T) {
	f := t3rpctest.New(t)
	f.Protocol = 2
	svc, s := pairingOverRegistry(t, f)

	c, err := svc.Pair(t.Context(), "u1", harness.KindT3Code, "Laptop", f.URL, f.PairToken)
	require.NoError(t, err)

	assert.Equal(t, harness.KindT3CodeV2, c.Kind)
	assert.Equal(t, int32(1), f.Exchanges.Load(), "the single-use token is spent once, by the client that can use it")
	stored, err := s.Pairing.GetComputer(t.Context(), "u1", c.ID)
	require.NoError(t, err)
	assert.Equal(t, harness.KindT3CodeV2, stored.Kind)
	assert.Zero(t, countSwitched(t, s), "a computer paired fresh never switched")
}

func TestIntegration_Protocol1ComputerAfterTheUpdate_SwitchesOnItsFirstCallWithOneEvent(t *testing.T) {
	f := t3rpctest.New(t)
	f.Protocol = 2
	svc, s := pairingOverRegistry(t, f)
	sealed, err := crypto.Encrypt(switchTestKey, []byte(f.Session().BearerToken))
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, s.Pairing.SaveComputer(t.Context(), pairing.Computer{
		ID: "c1", UserID: "u1", Kind: harness.KindT3Code, Name: "Laptop", ServerURL: f.URL, BearerToken: sealed,
		TokenExpiresAt: now.Add(time.Hour), HarnessVersion: "0.0.33", CreatedAt: now, UpdatedAt: now,
	}))

	for range 2 {
		providers, err := svc.ListProviders(t.Context(), "u1", "c1")
		require.NoError(t, err, "the call is retried on the protocol-2 client, so nobody re-pairs")
		require.Len(t, providers, 1)
	}

	stored, err := s.Pairing.GetComputer(t.Context(), "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, harness.KindT3CodeV2, stored.Kind)
	assert.Equal(t, "0.0.34", stored.HarnessVersion)
	assert.Equal(t, 1, countSwitched(t, s))
}
