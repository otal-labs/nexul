package pairing

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// fakeTunnels records what was torn down at Cloudflare.
type fakeTunnels struct {
	mu           sync.Mutex
	deleted      []ComputerTunnel
	deleteErr    error
	tokenDeletes int
	tokenErr     error
}

func (f *fakeTunnels) DeleteTunnel(_ context.Context, t ComputerTunnel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, t)
	return nil
}

func (f *fakeTunnels) DeleteAccessServiceToken(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tokenErr != nil {
		return f.tokenErr
	}
	f.tokenDeletes++
	return nil
}

// fakeBus records the pair failures pairing publishes.
type fakeBus struct {
	mu         sync.Mutex
	pairFailed []PairFailedEvent
}

func (b *fakeBus) Publish(_ context.Context, topic string, payload any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if topic == TopicPairFailed {
		b.pairFailed = append(b.pairFailed, payload.(PairFailedEvent))
	}
	return nil
}

func newTunnelService(repo *fakeRepo, exch *fakeExchanger, tunnels Tunnels) (*Service, *[]string) {
	var changed []string
	svc := NewService(Config{
		Repo: repo, Harnesses: registry(exch), EncryptionKey: testEncKey, Tunnels: tunnels, Tokens: newFakeTokens(),
		Now:                func() time.Time { return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) },
		OnComputersChanged: func(userID string) { changed = append(changed, userID) },
	})
	return svc, &changed
}

// tunnelComputer stores one of u1's computers as pairing through a tunnel left it.
func tunnelComputer(t *testing.T, svc *Service) string {
	t.Helper()
	now := svc.now().UTC()
	c := Computer{
		ID: ids.New(), UserID: "u1", Kind: harness.KindT3Code, Name: "Laptop", ServerURL: "https://laptop-ab12cd34.example.com",
		CreatedAt: now, UpdatedAt: now,
		Tunnel: &ComputerTunnel{TunnelID: "tun-1", Hostname: "laptop-ab12cd34.example.com", ZoneID: "z1", RecordID: "rec-1", AccessAppID: "app-1"},
	}
	require.NoError(t, svc.repo.SaveComputer(t.Context(), c))
	return c.ID
}

func TestService_RetireTunnels_DeletesEveryTunnelThenTheAccessTokenAndKeepsTheComputers(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	tunnels := &fakeTunnels{}
	svc, changed := newTunnelService(repo, &fakeExchanger{}, tunnels)
	id := tunnelComputer(t, svc)

	require.NoError(t, svc.RetireTunnels(t.Context()))
	require.Len(t, tunnels.deleted, 1)
	assert.Equal(t, "app-1", tunnels.deleted[0].AccessAppID)
	assert.Equal(t, 1, tunnels.tokenDeletes)
	stored, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err, "the computer keeps its row, links and setup")
	assert.Nil(t, stored.Tunnel)
	assert.Equal(t, TopicTunnelRemoved, repo.outbox[len(repo.outbox)-1].Topic)
	assert.Equal(t, []string{"u1"}, *changed)

	require.NoError(t, svc.RetireTunnels(t.Context()), "a later boot finds nothing left to delete")
	assert.Len(t, tunnels.deleted, 1)
}

func TestService_RetireTunnels_KeepsWhatCloudflareRefused(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	tunnels := &fakeTunnels{deleteErr: errBoom}
	svc, _ := newTunnelService(repo, &fakeExchanger{}, tunnels)
	id := tunnelComputer(t, svc)

	require.Error(t, svc.RetireTunnels(t.Context()))
	stored, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err)
	assert.NotNil(t, stored.Tunnel, "the next boot retries it")
	assert.Zero(t, tunnels.tokenDeletes, "the token outlives every tunnel its Access apps admit it to")

	tunnels.deleteErr, tunnels.tokenErr = nil, errBoom
	require.ErrorIs(t, svc.RetireTunnels(t.Context()), errBoom)
}

func TestService_RetireTunnels_Unwired(t *testing.T) {
	t.Parallel()
	svc, _ := newTunnelService(newFakeRepo(), &fakeExchanger{}, nil)
	require.ErrorIs(t, svc.RetireTunnels(t.Context()), apperrs.ErrFatal)
}

func TestService_DeleteComputer_TearsItsTunnelDownFirst(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	tunnels := &fakeTunnels{}
	svc, _ := newTunnelService(repo, &fakeExchanger{}, tunnels)
	id := tunnelComputer(t, svc)

	tunnels.deleteErr = errBoom
	require.ErrorIs(t, svc.DeleteComputer(t.Context(), "u1", id), errBoom)
	_, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err, "a failed teardown keeps the computer so removing it again retries")

	tunnels.deleteErr = nil
	require.NoError(t, svc.DeleteComputer(t.Context(), "u1", id))
	require.Len(t, tunnels.deleted, 1)
	_, err = repo.GetComputer(t.Context(), "u1", id)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.Len(t, repo.outbox, 2)
	assert.Equal(t, ComputerRemovedEvent{ComputerID: id, UserID: "u1", MembersOnly: true}, repo.outbox[0].Payload, "its owner's Your computer step may be left again")
	assert.Equal(t, TopicTunnelRemoved, repo.outbox[1].Topic)
	assert.Equal(t, id, repo.outbox[1].Payload.(TunnelChangedEvent).ComputerID)
}

func TestService_DeleteComputer_WithTunnelButTunnelsUnwired_KeepsTheComputer(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	svc, _ := newTunnelService(repo, &fakeExchanger{}, nil)
	id := tunnelComputer(t, svc)

	require.ErrorIs(t, svc.DeleteComputer(t.Context(), "u1", id), apperrs.ErrFatal)
	_, err := repo.GetComputer(t.Context(), "u1", id)
	require.NoError(t, err)
}
