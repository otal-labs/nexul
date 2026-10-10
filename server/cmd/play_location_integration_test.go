package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	"github.com/otal-labs/nexul/internal/pairing"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"
	"github.com/otal-labs/nexul/internal/t3clientv2"
	"github.com/otal-labs/nexul/internal/t3rpc"
	"github.com/otal-labs/nexul/internal/t3rpc/t3rpctest"
)

type pausedPlayTurn struct {
	*agent.Service
	ready  chan struct{}
	resume chan struct{}
	done   chan struct{}
}

func (p *pausedPlayTurn) RunTurn(ctx context.Context, req agent.TurnRequest) {
	close(p.ready)
	select {
	case <-p.resume:
	case <-ctx.Done():
		return
	}
	p.Service.RunTurn(ctx, req)
	close(p.done)
}

func TestIntegration_PlayLocation_SettingsChangedBeforeRunTurn_UsesConfirmedTarget(t *testing.T) {
	for _, change := range []string{"changed", "unlinked", "computer removed"} {
		t.Run(change, func(t *testing.T) {
			ctx := as(uOwner)
			var got harness.Target
			client := &harnesstest.Client{ListProvidersFn: func(context.Context, harness.Session) ([]harness.Provider, error) {
				return []harness.Provider{{ID: "provider", Driver: "provider", Name: "Provider"}}, nil
			}, StartTurnFn: func(_ context.Context, target harness.Target, _ string, _ harness.TurnPrompts) (harness.StartResult, error) {
				got = target
				updates := make(chan harness.Update, 1)
				updates <- harness.Update{Terminal: &harness.TurnResult{State: harness.TurnDone}}
				close(updates)
				return harness.StartResult{SessionID: "thread-confirmed", Updates: updates}, nil
			}}
			f, svc, runner, turn := newPlayLocationFixture(t, client, harness.Session{ServerURL: "https://example.com", BearerToken: "test-token"})

			_, err := runner.Run(ctx, plays.RunInput{PlayID: "play-location", TargetType: plays.TargetTicket, TargetID: f.ticket.ID, Via: plays.ViaWeb})
			require.NoError(t, err)
			<-turn.ready
			if change == "unlinked" {
				require.NoError(t, svc.ClearProjectLink(ctx, uOwner, f.ticket.ProjectID))
			}
			if change == "changed" {
				_, err = svc.SetProjectLink(ctx, uOwner, f.ticket.ProjectID, pairing.ProjectLink{ComputerID: "computer", HarnessProjectID: "checkout-b", Provider: "provider", Model: "model-b", StartIn: pairing.StartInFolder})
				require.NoError(t, err)
			}
			if change == "computer removed" {
				require.NoError(t, f.store.Pairing.DeleteComputer(ctx, uOwner, "computer"))
			}
			close(turn.resume)
			<-turn.done
			if change == "computer removed" {
				assert.Empty(t, got.Session.ComputerID, "a removed computer cannot use the old launch credential")
				return
			}
			assert.Equal(t, "computer", got.Session.ComputerID)
			assert.Equal(t, "checkout-a", got.ProjectID)
			assert.Equal(t, "model-a", got.Model)
			assert.Equal(t, []harness.OptionSetting{{ID: "effort", Value: "high"}}, got.ModelOptions)
			assert.True(t, got.Worktree)
		})
	}
}

func newPlayLocationFixture(t *testing.T, client *harnesstest.Client, session harness.Session) (permFixture, *pairing.Service, *plays.Runner, *pausedPlayTurn) {
	t.Helper()
	f := newPermFixture(t)
	ctx := as(uOwner)
	now := time.Now().UTC()
	registry := harnesstest.Registry(client)
	sealed, err := crypto.Encrypt(switchTestKey, []byte(session.BearerToken))
	require.NoError(t, err)
	require.NoError(t, f.store.Pairing.SaveComputer(ctx, pairing.Computer{ID: "computer", UserID: uOwner, Name: "Laptop", Kind: harness.KindT3Code, ServerURL: session.ServerURL, BearerToken: sealed, TokenExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Pairing.SetSetupConfirmedAt(ctx, uOwner, "computer", &now, eventbus.OutboxEvent{ID: "setup-overall", Topic: "test.setup", Payload: map[string]string{"computer_id": "computer"}}))
	require.NoError(t, f.store.Pairing.SaveProviderSetup(ctx, "computer", pairing.ProviderSetup{Provider: "provider", ConfirmedAt: &now}, now, eventbus.OutboxEvent{ID: "setup-provider", Topic: "test.setup", Payload: map[string]string{"computer_id": "computer"}}))
	svc := pairing.NewService(pairing.Config{Repo: f.store.Pairing, Harnesses: registry, EncryptionKey: switchTestKey, Projects: f.svc.accessSvc})
	_, err = svc.SetDefaults(ctx, uOwner, pairing.Defaults{DefaultComputerID: "computer", FallbackProjectID: "fallback", Provider: "provider", StartIn: pairing.StartInFolder})
	require.NoError(t, err)
	_, err = svc.SetProjectLink(ctx, uOwner, f.ticket.ProjectID, pairing.ProjectLink{ComputerID: "computer", HarnessProjectID: "checkout-a", Provider: "provider", Model: "model-a", ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}}, StartIn: pairing.StartInWorktree})
	require.NoError(t, err)
	turn := &pausedPlayTurn{Service: agent.NewService(agent.Config{Live: noopPublisher{}, Conversations: agentConversations{svc: f.svc.chatSvc}, Targets: svc, Harnesses: registry, Tickets: agentTicketReader{svc: f.svc.ticketsSvc, projects: f.store.Projects, statuses: f.store.Statuses}}), ready: make(chan struct{}), resume: make(chan struct{}), done: make(chan struct{})}
	stage := plays.StageBacklog
	play := &plays.Play{ShowWhenStage: &stage, ID: "play-location", WorkspaceID: wsDefault, Label: "Run", Type: plays.TypeTicket, Instructions: "Complete the ticket", Enabled: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, f.store.Plays.Create(ctx, play))
	runner := plays.NewRunner(plays.RunnerConfig{Plays: f.store.Plays, Trails: f.store.PlayTrails, Perm: f.svc.accessSvc, Targets: playsTargetReader{tickets: f.svc.ticketsSvc, workspace: f.svc.workspaceSvc}, Projects: playsProjectLookup{memoriesProjectLookup{projects: f.store.Projects}}, Harness: playsHarnessResolver{svc: svc}, Memories: playsMemoryReader{svc: f.svc.memoriesSvc}, Threads: playsThreads{svc: f.svc.chatSvc}, Turns: turn})
	return f, svc, runner, turn
}

func TestIntegration_PlayLocation_FailedChangeReopened_RetryLeavesTheOldThread(t *testing.T) {
	rpc := t3rpctest.New(t)
	rpc.Protocol = 2
	h := t3clientv2.NewHarness(t3rpc.Options{HTTPClient: rpc.Client(), RPCTimeout: 5 * time.Second})
	var offline atomic.Bool
	client := &harnesstest.Client{ListProvidersFn: func(context.Context, harness.Session) ([]harness.Provider, error) {
		if offline.Load() {
			return nil, errors.New("offline")
		}
		return []harness.Provider{{ID: "provider", Driver: "provider", Name: "Provider", Models: []harness.ProviderModel{{Slug: "model-a", IsDefault: true}}}}, nil
	}, StartTurnFn: h.StartTurn}
	f, svc, runner, turn := newPlayLocationFixture(t, client, rpc.Session())
	ctx := as(uOwner)
	conv, err := f.svc.chatSvc.GetOrCreateTicketThread(ctx, wsDefault, f.ticket.ID, uOwner)
	require.NoError(t, err)
	require.NoError(t, f.svc.chatSvc.SetAgentThread(ctx, conv.ID, "thread-checkout-a"))
	in := plays.RunInput{PlayID: "play-location", TargetType: plays.TargetTicket, TargetID: f.ticket.ID, Via: plays.ViaWeb, ComputerID: "computer", HarnessProjectID: "checkout-b", Model: "model-a"}
	offline.Store(true)
	_, err = runner.Run(ctx, in)
	var refusal *plays.HarnessRefusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, string(pairing.ReasonOffline), refusal.Reason)
	link, err := svc.GetProjectLink(ctx, uOwner, f.ticket.ProjectID)
	require.NoError(t, err)
	assert.Equal(t, "checkout-b", link.HarnessProjectID)
	require.NoError(t, f.svc.chatSvc.SetAgentThread(ctx, conv.ID, "thread-checkout-a"))
	offline.Store(false)
	in.ComputerID, in.HarnessProjectID = "", ""
	_, err = runner.Run(ctx, in)
	require.NoError(t, err)
	<-turn.ready
	close(turn.resume)
	old := t3rpctest.WaitFor(t, rpc.Subscribed, "old thread snapshot")
	snapshot := func(project string) map[string]any {
		return map[string]any{"kind": "snapshot", "snapshotSequence": 1, "projection": map[string]any{"thread": map[string]any{"projectId": project, "runtimeMode": "full-access", "modelSelection": map[string]any{"instanceId": "provider", "model": "model-a"}}}}
	}
	rpc.Write(t3rpctest.Chunk(old, snapshot("checkout-a")))
	create := t3rpctest.WaitFor(t, rpc.Dispatched, "first command after reopening")
	require.Equal(t, "thread.create", create["type"], "the first prompt must not reach the surviving checkout")
	assert.Equal(t, "checkout-b", create["projectId"])
	fresh := t3rpctest.WaitFor(t, rpc.Subscribed, "fresh thread")
	rpc.Write(t3rpctest.Chunk(fresh, snapshot("checkout-b")))
	dispatch := t3rpctest.WaitFor(t, rpc.Dispatched, "prompt")
	assert.Equal(t, create["threadId"], dispatch["threadId"])
	messageID := dispatch["messageId"]
	rpc.Write(t3rpctest.Chunk(fresh, map[string]any{"kind": "event", "sequence": 2, "event": map[string]any{"type": "run.created", "payload": map[string]any{"id": "run-1", "userMessageId": messageID, "rootNodeId": "root", "status": "running"}}}, map[string]any{"kind": "event", "sequence": 3, "event": map[string]any{"type": "run.updated", "payload": map[string]any{"id": "run-1", "userMessageId": messageID, "rootNodeId": "root", "status": "waiting"}}}))
	<-turn.done
	saved, err := f.svc.chatSvc.GetConversation(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, create["threadId"], saved.AgentThreadID)
}
