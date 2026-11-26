package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/platform/config"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/inprocess"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil/migrated"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/runner"
)

// TestIntegration_StackPlays_RunAsTheirPickerAndStopWhenTheirRightGoes wires a stack play end to end: a stack auto play
// runs as whoever saved it, a crash queues a run as that person with no computer named, and a crash after the person
// loses the right to run plays is "didn't run" with the reason.
func TestIntegration_StackPlays_RunAsTheirPickerAndStopWhenTheirRightGoes(t *testing.T) {
	t.Parallel()
	c := newSharingCast(t)
	ctx := t.Context()
	_, err := c.db.Exec(`UPDATE roles SET permissions = '["plays:read","plays:run","autoplays:read","autoplays:write"]' WHERE id = 'role-member'`)
	require.NoError(t, err)
	for _, user := range []string{"u-alice", "u-dave"} {
		require.NoError(t, c.store.Access.Set(ctx, "project", "project-general", user, grant("projects:open"), nil))
	}
	now := time.Now()
	require.NoError(t, c.store.Stacks.Create(ctx, &deploy.Stack{ID: "stack-shop", ProjectID: "project-general", Name: "shop", Slug: "shop", Machine: "m1",
		Strategy: deploy.StrategyCompose, CreatedAt: now, UpdatedAt: now}))
	play, err := c.svc.playsSvc.Create(as("u-carol"), "workspace-default", plays.CreateInput{Label: "Look into crashes", Type: plays.TypeStack, Enabled: true})
	require.NoError(t, err)
	own, err := c.svc.playsSvc.CreateAutoPlay(as("u-alice"), "workspace-default", play.ID, plays.AutoPlayInput{Moment: plays.MomentStackServiceCrashed})
	require.NoError(t, err)
	shared, err := c.svc.playsSvc.CreateAutoPlay(as("u-dave"), "workspace-default", play.ID, plays.AutoPlayInput{Moment: plays.MomentStackServiceCrashed})
	require.NoError(t, err)
	assert.Equal(t, []any{"u-alice", "u-dave"}, []any{own.RunAs, shared.RunAs})
	_, err = c.svc.playsSvc.UpdateAutoPlay(as("u-dave"), "workspace-default", play.ID, shared.ID, true, plays.AutoPlayInput{Moment: plays.MomentStackServiceCrashed})
	require.NoError(t, err)

	crash := func() {
		raw, err := json.Marshal(map[string]any{
			"workspace_id": "workspace-default", "project_id": "project-general", "stack_id": "stack-shop", "stack_name": "shop",
			"service": "api", "outage_id": ids.New(),
		})
		require.NoError(t, err)
		require.NoError(t, c.svc.playsRunner.HandleAutoPlayMoment(ctx, eventbus.Event{ID: ids.New(), Topic: "service.container_stopped", Payload: raw}))
	}
	crash()
	items, err := c.store.PlayQueue.ListQueueByTarget(ctx, plays.TargetStack, "stack-shop")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, []any{plays.QueueQueued, "u-dave", plays.RunOnPerson, "api"}, []any{items[0].Status, items[0].PersonID, items[0].RunOn, items[0].Service}, items[0].Reason)
	assert.Equal(t, []any{[]string{}, ""}, []any{items[0].ComputerIDs, items[0].ComputerID}, "no computer is named; placement picks one from dave's order")

	_, err = c.db.Exec(`UPDATE roles SET permissions = '["plays:read","autoplays:read","autoplays:write"]' WHERE id = 'role-member'`)
	require.NoError(t, err)
	// The live wiring's dispatcher may be mid-pass on the run; once it lets go the run is queued again or gone.
	require.Eventually(t, func() bool {
		it, err := c.store.PlayQueue.GetQueueItem(ctx, items[0].ID)
		require.NoError(t, err)
		if it.Status == plays.QueueDispatching {
			return false
		}
		if it.Status != plays.QueueQueued {
			return true
		}
		it.Status = plays.QueueCancelled
		return c.store.PlayQueue.MoveQueueItem(ctx, it, plays.QueueQueued) == nil
	}, 5*time.Second, 10*time.Millisecond, "the waiting run is out of the way for the next crash")
	first := items[0].ID
	crash()
	items, err = c.store.PlayQueue.ListQueueByTarget(ctx, plays.TargetStack, "stack-shop")
	require.NoError(t, err)
	require.Len(t, items, 2)
	didnt := slices.IndexFunc(items, func(it *plays.QueueItem) bool { return it.ID != first && it.Status == plays.QueueDidntRun })
	require.GreaterOrEqual(t, didnt, 0, "the second crash did not run")
	assert.Contains(t, items[didnt].Reason, "may not run")
}

// TestStackPlays_ACrashTheRunnerSeesReachesTheAutoPlayWaitingForIt: a container dying on a machine opens the outage,
// whose event is the moment a switched-on stack auto play queues its run for, naming the service and the outage.
func TestStackPlays_ACrashTheRunnerSeesReachesTheAutoPlayWaitingForIt(t *testing.T) {
	t.Parallel()
	db := migrated.DB(t)
	store := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bus := inprocess.New(inprocess.Options{Logger: logger, DedupeStore: store.ProcessedEvents, DeadLetterStore: store.DeadLetters})
	t.Cleanup(func() { _ = bus.Close() })
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "nexul.db"), HTTPAddr: "127.0.0.1:0", AuthSecret: "0123456789abcdef0123456789abcdef"}
	svc := wireCoreServices(cfg, store, []byte("0123456789abcdef0123456789abcdef"), bus, logger)
	wireDomainEventSubscriptions(context.Background(), bus, svc, logger)
	wireLiveHubAndAgent(context.Background(), bus, store, svc, logger)
	f := seedPermFixture(t, db, svc, store)
	ctx := context.Background()
	now := time.Now().UTC()
	play := &plays.Play{ID: ids.New(), WorkspaceID: "workspace-default", Label: "Look into crashes", Type: plays.TypeStack, Enabled: true, ExcludedProjectIDs: []string{}, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, store.Plays.Create(ctx, play, plays.Revision{}))
	require.NoError(t, store.Plays.CreateAutoPlay(ctx, &plays.AutoPlay{
		ID: ids.New(), PlayID: play.ID, WorkspaceID: play.WorkspaceID, Enabled: true, Moment: plays.MomentStackServiceCrashed,
		Conditions: plays.Conditions{Match: plays.MatchAll, Groups: []plays.Group{}},
		Priority:   plays.Priority{Rules: []plays.PriorityRule{}, Otherwise: plays.LevelNormal},
		RunOn:      plays.RunOnPerson, RunAs: uOwner, CreatedAt: now, UpdatedAt: now,
	}))

	require.NoError(t, bus.Publish(ctx, runner.TopicContainerEvent, runner.ContainerEvent{Machine: "m1", Container: "web", Action: runner.ContainerActionDie, ExitCode: 137}))

	require.Eventually(t, func() bool { return outboxCount(t, db, deploy.TopicContainerStopped) == 1 }, 5*time.Second, 10*time.Millisecond)
	// The relay is not running here, so the stored event is handed to the bus as it would deliver it.
	var payload []byte
	require.NoError(t, db.QueryRow(`SELECT payload FROM outbox WHERE topic = ?`, deploy.TopicContainerStopped).Scan(&payload))
	require.NoError(t, bus.Publish(ctx, deploy.TopicContainerStopped, json.RawMessage(payload)))

	var items []*plays.QueueItem
	require.Eventually(t, func() bool {
		var err error
		items, err = store.PlayQueue.ListQueueByTarget(ctx, plays.TargetStack, f.stack)
		return err == nil && len(items) == 1
	}, 5*time.Second, 10*time.Millisecond)
	open, err := svc.deploySvc.OpenOutages(ctx, f.stack)
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, []any{plays.MomentStackServiceCrashed, "web", open[0].ID, uOwner, "project-general"},
		[]any{items[0].Moment, items[0].Service, items[0].OutageID, items[0].PersonID, items[0].ProjectID})
}

// TestStackPlays_TheRunnerReadsAStackAsWhoRunsIt: facts need no actor, the target read is the person's own, and a
// run's frames reach whoever can load the stack.
func TestStackPlays_TheRunnerReadsAStackAsWhoRunsIt(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	ctx := t.Context()

	facts, err := playsFacts{store: f.store}.Facts(ctx, plays.TargetStack, f.stack)
	require.NoError(t, err)
	assert.Equal(t, plays.Facts{ProjectID: "project-general", StackID: f.stack}, facts)
	_, err = playsFacts{store: f.store}.Facts(ctx, plays.TargetStack, "gone")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	reader := playsTargetReader{deploy: f.svc.deploySvc}
	got, err := reader.GetStack(as(uOwner), f.stack)
	require.NoError(t, err)
	assert.Equal(t, "project-general", got.ProjectID)
	assert.NotEmpty(t, got.Name)
	_, err = reader.GetStack(as(uPlain), f.stack)
	require.Error(t, err, "a person who cannot read the stack cannot run a play on it")

	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	frame := plays.Trail{ID: "tr-1", WorkspaceID: "workspace-default", TargetType: plays.TargetStack, TargetID: f.stack, StarterID: uPlain}
	assert.True(t, a.allows(as(uOwner), plays.TopicPlayRun, frame), "the Owner reads the stack")
	assert.False(t, a.allows(as(uPlain), plays.TopicPlayRun, frame), "even the starter hears nothing of a stack they cannot load")
}
