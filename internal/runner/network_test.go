package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeNetworkCmd is a CommandRunner recording every `docker network connect` call, canned per network name.
type fakeNetworkCmd struct {
	calls  [][]string
	errFor map[string]error
}

func (f *fakeNetworkCmd) run(_ context.Context, _ string, _ func(string), name string, args ...string) error {
	f.calls = append(f.calls, append([]string{name}, args...))
	if len(args) < 3 {
		return nil
	}
	return f.errFor[args[2]] // args: ["network", "connect", <network>, <container>]
}

func TestJoinNetworks_RunsConnectPerNetwork(t *testing.T) {
	cmd := &fakeNetworkCmd{errFor: map[string]error{}}

	err := joinNetworks(context.Background(), cmd.run, "gateway-container", []string{"net1", "net2"})
	require.NoError(t, err)

	require.Len(t, cmd.calls, 2)
	assert.Equal(t, []string{"docker", "network", "connect", "net1", "gateway-container"}, cmd.calls[0])
	assert.Equal(t, []string{"docker", "network", "connect", "net2", "gateway-container"}, cmd.calls[1])
}

// TestJoinNetworks_AlreadyExistsIsIdempotent is the ticket's headline requirement: Docker's own
// idempotent-connect error ("endpoint with name X already exists in network Y") counts as success, since a
// redeploy or a second exposure often re-requests a network the gateway already joined.
func TestJoinNetworks_AlreadyExistsIsIdempotent(t *testing.T) {
	cmd := &fakeNetworkCmd{errFor: map[string]error{
		"net1": errors.New("Error response from daemon: endpoint with name gateway-container already exists in network net1"),
	}}

	err := joinNetworks(context.Background(), cmd.run, "gateway-container", []string{"net1"})
	require.NoError(t, err)
}

// TestJoinNetworks_RealFailureSurfacesButKeepsGoing asserts a genuine failure is reported, but every other
// network in the batch is still attempted rather than aborting at the first error.
func TestJoinNetworks_RealFailureSurfacesButKeepsGoing(t *testing.T) {
	boom := errors.New("network net1 not found")
	cmd := &fakeNetworkCmd{errFor: map[string]error{"net1": boom}}

	err := joinNetworks(context.Background(), cmd.run, "gateway-container", []string{"net1", "net2"})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	require.Len(t, cmd.calls, 2, "net2 is still attempted after net1 fails")
}

func TestJoinNetworks_SkipsBlankEntries(t *testing.T) {
	cmd := &fakeNetworkCmd{errFor: map[string]error{}}

	err := joinNetworks(context.Background(), cmd.run, "gateway-container", []string{"", "  ", "net1"})
	require.NoError(t, err)
	require.Len(t, cmd.calls, 1)
}

func TestShellExecutor_JoinNetworks_SendsSuccessResult(t *testing.T) {
	cmd := &fakeNetworkCmd{errFor: map[string]error{}}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}

	e.JoinNetworks(context.Background(), "gateway-container", []string{"net1"}, rec.send)

	last := rec.last()
	assert.Equal(t, FrameJoinNetworksResult, last.Type)
	assert.Equal(t, "gateway-container", last.GatewayContainer)
	assert.Equal(t, BuildStatusSuccess, last.Status)
	assert.Empty(t, last.Error)
}

func TestShellExecutor_JoinNetworks_SendsFailureResult(t *testing.T) {
	boom := errors.New("network net1 not found")
	cmd := &fakeNetworkCmd{errFor: map[string]error{"net1": boom}}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}

	e.JoinNetworks(context.Background(), "gateway-container", []string{"net1"}, rec.send)

	last := rec.last()
	assert.Equal(t, FrameJoinNetworksResult, last.Type)
	assert.Equal(t, BuildStatusFailed, last.Status)
	assert.Contains(t, last.Error, "net1")
}

// TestShellExecutor_Deploy_JoinsGatewayNetworks covers the deploy path's guarded one-liner: a deploy carrying
// GatewayContainer runs the join as its last step, before the terminal deploy_result.
func TestShellExecutor_Deploy_JoinsGatewayNetworks(t *testing.T) {
	cmd := &fakeCmd{}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}

	e.Deploy(context.Background(), DeployRequestedEvent{
		ID: "d1", Service: "api", Image: "img:v1", Strategy: "run",
		GatewayContainer: "gateway-container", JoinNetworks: []string{"api_default"},
	}, rec.send)

	found := false
	for _, args := range cmd.argsFor("docker") {
		if len(args) >= 4 && args[0] == "network" && args[1] == "connect" && args[2] == "api_default" && args[3] == "gateway-container" {
			found = true
		}
	}
	assert.True(t, found, "the deploy's last step joins the gateway container onto its declared networks")
	assert.Equal(t, DeployStatusHealthy, rec.last().Status)
}

// TestShellExecutor_Deploy_SkipsJoinWhenGatewayContainerEmpty is the "guarded on the field being non-empty"
// requirement: a plain deploy with no gateway never runs a network-join step at all.
func TestShellExecutor_Deploy_SkipsJoinWhenGatewayContainerEmpty(t *testing.T) {
	cmd := &fakeCmd{}
	e := newTestExecutor(cmd.run)
	rec := &frameRecorder{}

	e.Deploy(context.Background(), DeployRequestedEvent{ID: "d1", Service: "api", Image: "img:v1", Strategy: "run"}, rec.send)

	for _, args := range cmd.argsFor("docker") {
		require.False(t, len(args) >= 2 && args[0] == "network" && args[1] == "connect", "no join step without a gateway container")
	}
}
