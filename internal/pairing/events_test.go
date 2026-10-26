package pairing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TestComputerEvents_AreMembersOnly drives every flow that publishes a computer event and holds each one to the mark
// integrations and automations skip (eventbus.MembersOnly), so a person's computer never leaves their own sockets.
func TestComputerEvents_AreMembersOnly(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	seen := map[string]bool{}
	check := func(topic string, payload any) {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		assert.True(t, eventbus.MembersOnly(raw), "%s reaches integrations and automations", topic)
		seen[topic] = true
	}

	setup := newSetupFixture(t)
	setup.start(t)
	_, err := setup.svc.UnconfirmSetup(ctx, "u1", setup.computer.ID)
	require.NoError(t, err)
	for _, e := range setup.repo.outbox {
		check(e.Topic, e.Payload)
	}
	for _, frame := range setup.bus.frames {
		check(TopicSetupTurnActivity, frame)
	}

	tunnelRepo := newFakeRepo()
	tunnelSvc, _ := newTunnelService(tunnelRepo, &fakeExchanger{}, &fakeTunnels{})
	tunnelComputer(t, tunnelSvc)
	require.NoError(t, tunnelSvc.RetireTunnels(ctx))
	require.NoError(t, tunnelSvc.DeleteComputer(ctx, "u1", tunnelComputer(t, tunnelSvc)))
	for _, e := range tunnelRepo.outbox {
		check(e.Topic, e.Payload)
	}

	svc, repo, v1, _ := twoKinds(t)
	paired, err := pairAt(svc, ctx, "u1", harness.KindT3Code, "Laptop", "https://h.example.com", "tok")
	require.NoError(t, err)
	v1.result.Kind = harness.KindT3CodeV2
	_, err = svc.pair(ctx, *paired, "https://h.example.com", "tok2")
	require.NoError(t, err)
	for _, e := range repo.outbox {
		check(e.Topic, e.Payload)
	}

	facts := newFactsFixture(t)
	require.NoError(t, facts.svc.HandleFactsReported(ctx, reportEvent(t, laptopReport("answering", 5e10))))
	for _, e := range facts.repo.outbox {
		check(e.Topic, e.Payload)
	}

	runner := newRunnerFixture(t, Computer{Name: "Laptop"})
	runner.runners.tokenErr = errBoom
	_, err = runner.svc.PairComputer(ctx, "u-alice", "c-laptop")
	require.Error(t, err)
	for _, failed := range runner.bus.pairFailed {
		check(TopicPairFailed, failed)
	}

	moving := newRunnerFixture(t, Computer{Name: "Laptop"})
	moving.repo.toMove = []Computer{{ID: "c-laptop", UserID: "u-alice"}}
	require.NoError(t, moving.svc.AnnounceMoves(ctx))
	for _, e := range moving.repo.outbox {
		check(e.Topic, e.Payload)
	}

	placed := newPlacementFixture(t)
	placedBus := &fakeBus{}
	placed.svc.bus = placedBus
	placed.add(t, "a", nil)
	require.NoError(t, placed.svc.ReorderComputers(ctx, "u1", []string{"a"}))
	for _, p := range placedBus.placements {
		check(TopicPlacementChanged, p)
	}
	placed.svc.PublishPresence("u1", "a", "connected")
	for _, p := range placedBus.presence {
		check(TopicPresenceChanged, p)
	}

	// Retired with pairing through a tunnel, still declared because the published contract only grows (ADR 0044).
	retired := map[string]bool{TopicTunnelCreated: true, TopicTunnelStatusChanged: true}
	for _, topic := range Topics() {
		assert.True(t, seen[topic.Name] || retired[topic.Name], "no flow here publishes %s; add one", topic.Name)
	}
}
