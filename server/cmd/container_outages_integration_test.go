package main

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/config"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/runner"
)

// TestOutages_TheRunnersContainerEventsReachDeploy: a machine runner's die on the bus opens the outage of the stack that
// owns the container, the stack read shows it, and a start followed by a snapshot of the container running ends it.
func TestOutages_TheRunnersContainerEventsReachDeploy(t *testing.T) {
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

	// Events are handled in order, so by the time the stack's outage shows, the stranger's event has been dealt with.
	require.NoError(t, bus.Publish(ctx, runner.TopicContainerEvent, runner.ContainerEvent{Machine: "m1", Container: "stranger", Action: runner.ContainerActionDie, ExitCode: 1}))
	require.NoError(t, bus.Publish(ctx, runner.TopicContainerEvent, runner.ContainerEvent{Machine: "m1", Container: "web", Action: runner.ContainerActionDie, ExitCode: 137}))

	require.Eventually(t, func() bool {
		open, err := svc.deploySvc.OpenOutages(ctx, f.stack)
		return err == nil && len(open) == 1
	}, 5*time.Second, 10*time.Millisecond)
	open, err := svc.deploySvc.OpenOutages(ctx, f.stack)
	require.NoError(t, err)
	assert.Equal(t, "web", open[0].Service)
	assert.Equal(t, 137, open[0].ExitCode)
	assert.Equal(t, 1, outboxCount(t, db, deploy.TopicContainerStopped), "the stranger raised nothing")

	require.NoError(t, bus.Publish(ctx, runner.TopicContainerStates, runner.ContainerStatesEvent{
		Machine: "m1", States: []runner.ContainerState{{Container: "web", State: runner.ContainerStateRunning}},
	}))
	require.Eventually(t, func() bool {
		open, err := svc.deploySvc.OpenOutages(ctx, f.stack)
		return err == nil && len(open) == 0
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, 1, outboxCount(t, db, deploy.TopicContainerRecovered))
}

func outboxCount(t *testing.T, db *sql.DB, topic string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outbox WHERE topic = ?`, topic).Scan(&n))
	return n
}

// TestOutages_FramesReachWhoReadsTheStack: both outage frames reach the people who can load the stack and no one else.
func TestOutages_FramesReachWhoReadsTheStack(t *testing.T) {
	f := newPermFixture(t)
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	for _, topic := range []string{deploy.TopicContainerStopped, deploy.TopicContainerRecovered} {
		ev := deploy.OutageEvent{ProjectID: "project-general", StackID: f.stack, Service: "web"}
		assert.True(t, a.allows(as(uOwner), topic, ev), "%s: the Owner", topic)
		assert.True(t, a.allows(as(uReader), topic, ev), "%s: a role that reads stacks", topic)
		assert.False(t, a.allows(as(uPlain), topic, ev), "%s: a member without stacks:read", topic)
		assert.False(t, a.allows(as(uOwner), topic, deploy.OutageEvent{StackID: "gone"}), "%s: a stack that does not exist", topic)
	}
}

func TestOutages_TheOutageTopicsArePushedAndScopedLikeTheStacks(t *testing.T) {
	for _, topic := range []string{deploy.TopicContainerStopped, deploy.TopicContainerRecovered} {
		assert.Contains(t, livePushTopics, topic)
		assert.Contains(t, liveRules, topic)
		assert.Contains(t, scopeRules, topic)
	}
}
