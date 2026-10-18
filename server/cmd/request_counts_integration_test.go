package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/dns"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/runner"
)

// TestRequestCounts_AProxyGatewaysRouterCountsReachTheStacksPage: a machine runner's counts on the bus land in the minute
// of the stack whose exposure the router is named for, the stack's requests route shows them to whoever reads the
// stack, and a router no exposure labelled adds nothing.
func TestRequestCounts_AProxyGatewaysRouterCountsReachTheStacksPage(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	wireDomainEventSubscriptions(context.Background(), bus, svc, logger)
	f := seedPermFixture(t, db, svc, store)
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, store.Services.Create(ctx, &deploy.Container{ID: "c-web", StackID: f.stack, Name: "web"}))
	require.NoError(t, store.DNS.SaveGateway(ctx, dns.Gateway{ID: "gw-1", Kind: dns.GatewayProxy, DockerNetwork: "net1", Machine: "m1", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, store.DNS.SaveExposure(ctx, dns.Exposure{ID: "e1", GatewayID: "gw-1", Hostname: "web.example.com", ServiceID: "c-web", Port: 80, ZoneID: "z1", Zone: "example.com", CreatedAt: now, UpdatedAt: now}))
	stack, err := svc.deploySvc.GetStack(as(uOwner), f.stack)
	require.NoError(t, err)
	stack.ErrorRate.Enabled, stack.DockerNetwork = true, "net"
	_, err = svc.deploySvc.UpdateStack(as(uOwner), *stack)
	require.NoError(t, err, "only a stack whose error-rate rule is on is counted")
	h := deploy.NewHandler(svc.deploySvc).Routes()
	get := func(user string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/"+f.stack+"/requests", nil).WithContext(as(user)))
		return rec
	}

	require.NoError(t, bus.Publish(ctx, runner.TopicRequestCounts, runner.RequestCountsEvent{Machine: "m1", Counts: []runner.RouterCount{
		{Router: dns.RouterName("e1") + "@file", Requests: 20, Errors: 5},
		{Router: "nexul@file", Requests: 99, Errors: 99},
	}}))

	var window deploy.RequestWindow
	require.Eventually(t, func() bool {
		rec := get(uReader)
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &window) != nil || !window.Counted {
			return false
		}
		last := window.Requests[len(window.Requests)-1]
		return last != nil && *last > 0
	}, 5*time.Second, 10*time.Millisecond)
	var requests, errors int64
	for i, r := range window.Requests {
		if r != nil {
			requests, errors = requests+*r, errors+*window.Errors5xx[i]
		}
	}
	assert.EqualValues(t, 20, requests, "the instance's own router is not the stack's")
	assert.EqualValues(t, 5, errors)
	assert.NotEqual(t, http.StatusOK, get(uPlain).Code, "a member who cannot read the stack cannot read its requests")
}
