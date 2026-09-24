package pairing

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
)

// fakeBus records what the tunnel watch publishes.
type fakeBus struct {
	mu     sync.Mutex
	events []TunnelStatusChangedEvent
	err    error
}

func (b *fakeBus) Publish(_ context.Context, topic string, payload any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if topic == TopicTunnelStatusChanged {
		b.events = append(b.events, payload.(TunnelStatusChangedEvent))
	}
	return b.err
}

func (b *fakeBus) published() []TunnelStatusChangedEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]TunnelStatusChangedEvent(nil), b.events...)
}

type watchFixture struct {
	svc     *Service
	repo    *fakeRepo
	tunnels *fakeTunnels
	exch    *fakeExchanger
	bus     *fakeBus
	now     *time.Time
}

func newWatchFixture(t *testing.T) watchFixture {
	t.Helper()
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	f := watchFixture{repo: newFakeRepo(), tunnels: &fakeTunnels{status: "inactive"}, exch: &fakeExchanger{version: "0.0.40"}, bus: &fakeBus{}, now: &now}
	f.svc = NewService(Config{
		Repo: f.repo, Harnesses: registry(f.exch), EncryptionKey: testEncKey, Tunnels: f.tunnels, Bus: f.bus,
		Tokens: newFakeTokens(), Now: func() time.Time { return *f.now },
	})
	return f
}

func (f watchFixture) watched() int {
	f.svc.watchMu.Lock()
	defer f.svc.watchMu.Unlock()
	return len(f.svc.watching)
}

func TestTunnelWatch_PublishesEachChangeUntilBothChecksPass(t *testing.T) {
	t.Parallel()
	f := newWatchFixture(t)
	c, err := f.svc.CreateComputerTunnel(t.Context(), "u1", harness.KindT3Code, "Laptop", 3773)
	require.NoError(t, err)
	require.Equal(t, 1, f.watched(), "creating the tunnel starts the watch")

	f.svc.pollTunnels(t.Context())
	f.svc.pollTunnels(t.Context())
	require.Len(t, f.bus.published(), 1, "an unchanged status is not pushed twice")
	assert.Equal(t, TunnelStatusChangedEvent{ComputerID: c.ID, UserID: "u1", Tunnel: "inactive"}, f.bus.published()[0])

	f.tunnels.status = "healthy"
	f.exch.versionErr = errBoom
	f.svc.pollTunnels(t.Context())
	require.Len(t, f.bus.published(), 2)
	assert.False(t, f.bus.published()[1].HarnessReachable)
	assert.Equal(t, 1, f.watched())

	f.exch.versionErr = nil
	f.svc.pollTunnels(t.Context())
	require.Len(t, f.bus.published(), 3)
	assert.Equal(t, TunnelStatusChangedEvent{ComputerID: c.ID, UserID: "u1", Tunnel: "healthy", HarnessReachable: true, HarnessVersion: "0.0.40"}, f.bus.published()[2])
	assert.Zero(t, f.watched(), "a connected tunnel stops being polled")
}

func TestTunnelWatch_StatusReadRestartsTheWatchOnlyWhileNotConnected(t *testing.T) {
	t.Parallel()
	f := newWatchFixture(t)
	c, err := f.svc.CreateComputerTunnel(t.Context(), "u1", harness.KindT3Code, "Laptop", 3773)
	require.NoError(t, err)

	*f.now = f.now.Add(tunnelWatchWindow + time.Second)
	f.svc.pollTunnels(t.Context())
	assert.Zero(t, f.watched(), "an unattended wait expires")
	assert.Empty(t, f.bus.published())

	_, err = f.svc.ComputerTunnelStatus(t.Context(), "u1", c.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, f.watched(), "reading the status again restarts the watch")

	f.svc.stopWatching(c.ID)
	f.tunnels.status = "healthy"
	status, err := f.svc.ComputerTunnelStatus(t.Context(), "u1", c.ID)
	require.NoError(t, err)
	assert.True(t, status.Connected())
	assert.Zero(t, f.watched(), "nothing left to wait for")
}

func TestTunnelWatch_ForgetsARemovedComputerAndSurvivesFailures(t *testing.T) {
	t.Parallel()
	f := newWatchFixture(t)
	c, err := f.svc.CreateComputerTunnel(t.Context(), "u1", harness.KindT3Code, "Laptop", 3773)
	require.NoError(t, err)

	f.tunnels.readErr = errBoom
	f.svc.pollTunnels(t.Context())
	assert.Equal(t, 1, f.watched(), "a failed read keeps watching")
	assert.Empty(t, f.bus.published())

	f.tunnels.readErr = nil
	f.bus.err = errBoom
	f.svc.pollTunnels(t.Context())
	assert.Len(t, f.bus.published(), 1, "a failed publish is logged, never fatal")

	require.NoError(t, f.svc.DeleteComputer(t.Context(), "u1", c.ID))
	f.svc.pollTunnels(t.Context())
	assert.Zero(t, f.watched())
}

func TestTunnelWatch_WithoutABusStaysSilent(t *testing.T) {
	t.Parallel()
	svc, _ := newTunnelService(newFakeRepo(), &fakeExchanger{}, &fakeTunnels{status: "inactive"})
	tunnelComputer(t, svc)
	svc.pollTunnels(t.Context())
	svc.watchMu.Lock()
	defer svc.watchMu.Unlock()
	assert.Len(t, svc.watching, 1)
}

func TestRunTunnelWatch_PollsOnEachTickAndStopsWithTheContext(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		f := newWatchFixture(t)
		_, err := f.svc.CreateComputerTunnel(t.Context(), "u1", harness.KindT3Code, "Laptop", 3773)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			f.svc.RunTunnelWatch(ctx)
			close(done)
		}()
		time.Sleep(tunnelWatchInterval)
		synctest.Wait()
		assert.Len(t, f.bus.published(), 1)

		cancel()
		<-done
	})
}
