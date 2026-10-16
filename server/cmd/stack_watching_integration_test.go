package main

import (
	"context"
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
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TestStackWatching_AnOutageNotifiesTheWatchersWhoReadTheStack: with the real wiring, a service going down and coming
// back reaches everyone who reads the stack and has not opted out, and no one who has lost the read by a rule.
func TestStackWatching_AnOutageNotifiesTheWatchersWhoReadTheStack(t *testing.T) {
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
	outage := deploy.OutageEvent{
		WorkspaceID: "workspace-default", ProjectID: "project-general", StackID: f.stack, StackName: "web", Service: "web", ExitCode: 137, OOM: true,
	}
	inbox := func(user string) []*workspace.Notification {
		ns, _, err := store.Notifications.Page(ctx, user, workspace.InboxFilter{}, nil, paging.Window{Limit: 50})
		require.NoError(t, err)
		return ns
	}
	// The Owner always reads the stack, so their notice arriving means the whole fan-out has been written.
	delivered := func(kind workspace.Kind) {
		require.Eventually(t, func() bool {
			for _, n := range inbox(uOwner) {
				if n.Kind == kind {
					return true
				}
			}
			return false
		}, 5*time.Second, 10*time.Millisecond)
	}

	_, err := svc.deploySvc.SetWatching(as(uPlain), f.stack, false)
	require.Error(t, err, "someone who cannot read the stack cannot change their watching")
	watching, err := svc.deploySvc.SetWatching(as(uOverwrite), f.stack, false)
	require.NoError(t, err)
	assert.False(t, watching)

	require.NoError(t, bus.Publish(ctx, deploy.TopicContainerStopped, outage))
	delivered(workspace.KindServiceStopped)

	require.Len(t, inbox(uReader), 1, "a reader watches by default")
	assert.Equal(t, "web crashed on web: exit 137, out of memory", inbox(uReader)[0].SubjectTitle)
	assert.Equal(t, workspace.SubjectStack, inbox(uReader)[0].SubjectType)
	assert.Empty(t, inbox(uOverwrite), "an opt-out is not told")
	assert.Empty(t, inbox(uPlain), "a member without stacks:read is not told")
	assert.Empty(t, inbox(uWriter), "a member without stacks:read is not told")

	require.NoError(t, store.Access.Set(ctx, "stack", f.stack, uReader, nil, grant("stacks:read")))
	require.NoError(t, bus.Publish(ctx, deploy.TopicContainerRecovered, outage))
	delivered(workspace.KindServiceRecovered)

	assert.Len(t, inbox(uReader), 1, "a rule that took the read away before the event fires leaves them out of the recovery")
	watching, err = svc.deploySvc.SetWatching(as(uOverwrite), f.stack, true)
	require.NoError(t, err)
	assert.True(t, watching)
}

// TestStackWatching_CanStackFollowsTheRulesWhereTheyLive: a branch deployment answers through its base stack's rules, and
// a stack outside every project, which no workspace's members read, answers no to everyone.
func TestStackWatching_CanStackFollowsTheRulesWhereTheyLive(t *testing.T) {
	f := newPermFixture(t)
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-feat", ProjectID: "project-general", Name: "web-feat", Slug: "web-feat", Machine: "m1",
		Strategy: deploy.StrategyRun, DerivedFrom: f.stack, Branch: "feat", CreatedAt: now, UpdatedAt: now}))
	access := f.svc.accessSvc

	assert.True(t, access.CanStack(ctx, uReader, "stack-feat", permissions.StacksRead))
	require.NoError(t, f.store.Access.Set(ctx, "stack", f.stack, uReader, nil, grant("stacks:read")))
	assert.False(t, access.CanStack(ctx, uReader, f.stack, permissions.StacksRead), "the rule on the base")
	assert.False(t, access.CanStack(ctx, uReader, "stack-feat", permissions.StacksRead), "its branch deployment follows")
	assert.True(t, access.CanStack(ctx, uOwner, "stack-feat", permissions.StacksRead))
	assert.False(t, access.CanStack(ctx, uOwner, f.infra, permissions.StacksRead), "an instance stack belongs to no workspace's members")
	assert.False(t, access.CanStack(ctx, uOwner, "gone", permissions.StacksRead))
}
