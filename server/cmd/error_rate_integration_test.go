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
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/workspace"
)

// seedProxyHostname gives the fixture's stack a container behind a proxy gateway's hostname, which the rule can watch.
func seedProxyHostname(t *testing.T, f permFixture) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, f.store.Services.Create(ctx, &deploy.Container{ID: "c-web", StackID: f.stack, Name: "web"}))
	require.NoError(t, f.store.DNS.SaveGateway(ctx, dns.Gateway{ID: "gw-1", Kind: dns.GatewayProxy, DockerNetwork: "net1", Machine: "m1", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.DNS.SaveExposure(ctx, dns.Exposure{ID: "e1", GatewayID: "gw-1", Hostname: "web.example.com", ServiceID: "c-web", Port: 80, ZoneID: "z1", Zone: "example.com", CreatedAt: now, UpdatedAt: now}))
}

func switchErrorRate(t *testing.T, f permFixture, rule deploy.ErrorRateRule, crashTickets bool) error {
	t.Helper()
	stack, err := f.svc.deploySvc.GetStack(as(uOwner), f.stack)
	require.NoError(t, err)
	stack.ErrorRate, stack.CrashTickets, stack.DockerNetwork = rule, crashTickets, "net"
	_, err = f.svc.deploySvc.UpdateStack(as(uOwner), *stack)
	return err
}

// TestErrorRate_AWindowOverTheRuleNotifiesWatchersAndFilesTheCrashTicket walks the real wiring: the rule is refused for a
// stack with no hostname, accepted once one is routed; five stored minutes at 20% 5xx make the once-a-minute loop open an
// episode, and the stack's readers are told, the crash ticket is filed, and the stack read shows the episode.
func TestErrorRate_AWindowOverTheRuleNotifiesWatchersAndFilesTheCrashTicket(t *testing.T) {
	t.Parallel()
	db := mentionsTestDB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	wireDomainEventSubscriptions(context.Background(), bus, svc, logger)
	f := seedPermFixture(t, db, svc, store)
	ctx := context.Background()
	rule := deploy.ErrorRateRule{Enabled: true, Percent: 5, Minutes: 5, MinRequests: 20}

	err := switchErrorRate(t, f, rule, true)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, deploy.ReasonNoHostname)

	seedProxyHostname(t, f)
	require.NoError(t, switchErrorRate(t, f, rule, true))
	end := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Minute)
	// One minute either side of the window the loop will read, in case the clock turns over while this runs.
	for i := range 7 {
		require.NoError(t, store.RequestMinutes.Add(ctx, []deploy.RequestAdd{{StackID: f.stack, Minute: end.Add(-time.Duration(i-1) * time.Minute), Requests: 100, Errors: 20}}))
	}
	before := len(projectTickets(t, f))

	runCtx, cancel := context.WithCancel(ctx)
	relay := outbox.NewRelay(store.Outbox, bus, outbox.RelayConfig{Wake: store.Commits().Next, Logger: logger})
	var running sync.WaitGroup
	running.Go(func() { _ = relay.Run(runCtx) })
	running.Go(func() { svc.deploySvc.RunRequestCounts(runCtx, logger) })
	t.Cleanup(func() { cancel(); running.Wait() })

	inbox := func(user string) []*workspace.Notification {
		ns, _, err := store.Notifications.Page(ctx, user, workspace.InboxFilter{}, nil, paging.Window{Limit: 50})
		require.NoError(t, err)
		return ns
	}
	require.Eventually(t, func() bool {
		for _, n := range inbox(uOwner) {
			if n.Kind == workspace.KindErrorRateCrossed {
				return true
			}
		}
		return false
	}, 10*time.Second, 20*time.Millisecond)

	require.Len(t, inbox(uReader), 1, "a reader watches by default")
	assert.Equal(t, "5xx at 20% on web over the last 5 minutes, above 5%", inbox(uReader)[0].SubjectTitle)
	assert.Empty(t, inbox(uPlain), "a member without stacks:read is not told")
	assert.Equal(t, 1, outboxCount(t, db, deploy.TopicErrorRateCrossed))

	require.Eventually(t, func() bool { return len(projectTickets(t, f)) == before+1 }, 10*time.Second, 20*time.Millisecond)
	ts := projectTickets(t, f)
	assert.Equal(t, "web errors: 5xx at 20%", ts[len(ts)-1].Title)

	stack, err := svc.deploySvc.GetStack(as(uReader), f.stack)
	require.NoError(t, err)
	open, recovered, err := svc.deploySvc.ErrorRateState(as(uReader), stack)
	require.NoError(t, err)
	require.NotNil(t, open)
	assert.Nil(t, recovered)
	assert.EqualValues(t, 20, open.PeakPercent)

	rule.Enabled = false
	require.NoError(t, switchErrorRate(t, f, rule, true))
	open, _, err = svc.deploySvc.ErrorRateState(as(uReader), stack)
	require.NoError(t, err)
	assert.Nil(t, open, "switching the rule off ends the episode")
	assert.Equal(t, 0, outboxCount(t, db, deploy.TopicErrorRateRecovered), "without announcing a recovery")
}

// TestErrorRate_FramesReachWhoReadsTheStack: both frames reach the people who can load the stack and no one else.
func TestErrorRate_FramesReachWhoReadsTheStack(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	for _, topic := range []string{deploy.TopicErrorRateCrossed, deploy.TopicErrorRateRecovered} {
		ev := deploy.ErrorRateEvent{ProjectID: "project-general", StackID: f.stack}
		assert.True(t, a.allows(as(uOwner), topic, ev), "%s: the Owner", topic)
		assert.True(t, a.allows(as(uReader), topic, ev), "%s: a role that reads stacks", topic)
		assert.False(t, a.allows(as(uPlain), topic, ev), "%s: a member without stacks:read", topic)
		assert.False(t, a.allows(as(uOwner), topic, deploy.ErrorRateEvent{StackID: "gone"}), "%s: a stack that does not exist", topic)
	}
}

func TestErrorRate_TheTopicsArePushedAndScopedLikeTheStacks(t *testing.T) {
	t.Parallel()
	for _, topic := range []string{deploy.TopicErrorRateCrossed, deploy.TopicErrorRateRecovered} {
		assert.Contains(t, livePushTopics, topic)
		assert.Contains(t, liveRules, topic)
		assert.Contains(t, scopeRules, topic)
	}
}
