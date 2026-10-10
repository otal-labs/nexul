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

	watch := newWatchFixture(t)
	c, err := watch.svc.CreateComputerTunnel(ctx, "u1", harness.KindT3Code, "Laptop", 3773)
	require.NoError(t, err)
	watch.svc.pollTunnels(ctx)
	require.NoError(t, watch.svc.DeleteComputer(ctx, "u1", c.ID))
	for _, e := range watch.repo.outbox {
		check(e.Topic, e.Payload)
	}
	for _, status := range watch.bus.published() {
		check(TopicTunnelStatusChanged, status)
	}

	svc, repo, v1, _ := twoKinds(t)
	paired, err := svc.Pair(ctx, "u1", harness.KindT3Code, "Laptop", "https://h.example.com", "tok")
	require.NoError(t, err)
	v1.result.Kind = harness.KindT3CodeV2
	_, err = svc.Repair(ctx, "u1", paired.ID, "Laptop", "https://h.example.com", "tok2")
	require.NoError(t, err)
	for _, e := range repo.outbox {
		check(e.Topic, e.Payload)
	}

	for _, topic := range Topics() {
		assert.True(t, seen[topic.Name], "no flow here publishes %s; add one", topic.Name)
	}
}
