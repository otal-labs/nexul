package main

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/runner"
)

type routesPushed struct {
	mu    sync.Mutex
	calls []string
}

func (p *routesPushed) JoinNetworks(context.Context, string, string, []string) error { return nil }

func (p *routesPushed) SetRoutes(_ context.Context, machine, container, _ string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, machine+"/"+container)
	return nil
}

func (p *routesPushed) got() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.calls...)
}

// TestProxyRoutes_ARunnerConnectingGetsItsMachinesProxyRoutes: the connect event names a machine by id, and the proxy
// gateways on it, only, are sent their routes again.
func TestProxyRoutes_ARunnerConnectingGetsItsMachinesProxyRoutes(t *testing.T) {
	db := mentionsTestDB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	pushed := &routesPushed{}
	svc.dnsSvc.SetRunnerJoin(pushed)
	wireDomainEventSubscriptions(context.Background(), bus, svc, logger)
	f := seedPermFixture(t, db, svc, store)
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, store.Machines.Create(ctx, &runner.Machine{ID: "machine-1", Name: "m1", FirstSeen: now, LastSeen: now}))
	require.NoError(t, store.Machines.Create(ctx, &runner.Machine{ID: "machine-2", Name: "m2", FirstSeen: now, LastSeen: now}))
	require.NoError(t, store.Services.Create(ctx, &deploy.Container{ID: "c-gw", StackID: f.stack, Name: "gw", ContainerName: "traefik-1"}))
	require.NoError(t, store.DNS.SaveGateway(ctx, dns.Gateway{ID: "gw-1", Kind: dns.GatewayProxy, DockerNetwork: "net1", Machine: "m1", ServiceID: f.stack, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.DNS.SaveGateway(ctx, dns.Gateway{ID: "gw-2", Kind: dns.GatewayProxy, DockerNetwork: "net2", Machine: "m2", ServiceID: f.stack, CreatedAt: now, UpdatedAt: now}))

	require.NoError(t, bus.Publish(ctx, runner.TopicRunnerConnected, runner.RunnerConnectedEvent{RunnerID: "r-1", MachineID: "machine-1"}))

	require.Eventually(t, func() bool { return len(pushed.got()) > 0 }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, []string{"m1/traefik-1"}, pushed.got(), "only the connecting runner's machine")
}
