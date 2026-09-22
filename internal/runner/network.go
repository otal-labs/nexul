package runner

import (
	"context"
	"fmt"
	"strings"
)

// alreadyConnectedSubstring is Docker's own idempotent-connect error text ("Error response from daemon:
// endpoint with name <container> already exists in network <network>"); matching on it lets a redeploy or a
// second exposure re-request a network the gateway already joined without that counting as a failure.
const alreadyConnectedSubstring = "already exists in network"

// JoinNetworks implements Executor.JoinNetworks: handles a standalone join_networks request from the server,
// reporting the outcome on a join_networks_result frame.
func (e *ShellExecutor) JoinNetworks(ctx context.Context, gatewayContainer string, networks []string, send func(Frame)) {
	result := Frame{Type: FrameJoinNetworksResult, GatewayContainer: gatewayContainer, Status: BuildStatusSuccess}
	if err := joinNetworks(ctx, e.cmd, gatewayContainer, networks); err != nil {
		e.log.Warn("join_networks failed", "gateway_container", gatewayContainer, "error", err)
		result.Status = BuildStatusFailed
		result.Error = err.Error()
	}
	send(result)
}

// joinGatewayNetworks runs the deploy's last step: join req.GatewayContainer onto req.JoinNetworks. Errors
// are logged, not surfaced — a routing nicety must never fail a deploy that otherwise succeeded.
func (e *ShellExecutor) joinGatewayNetworks(ctx context.Context, req DeployRequestedEvent, logs *logStream) {
	defer logs.flush()
	stream := logs.step(LogPhaseDeploy, "docker network connect "+strings.Join(req.JoinNetworks, ",")+" "+req.GatewayContainer)
	if err := joinNetworks(ctx, e.cmd, req.GatewayContainer, req.JoinNetworks); err != nil {
		e.log.Warn("gateway network join failed", "id", req.ID, "gateway_container", req.GatewayContainer, "error", err)
		stream("gateway network join failed: " + err.Error() + "\n")
	}
}

// joinNetworks runs `docker network connect <network> <gatewayContainer>` for each network, in order,
// treating "already exists" as success (idempotent). It returns the first real failure, if any, but
// still attempts every network rather than stopping at the first error.
func joinNetworks(ctx context.Context, cmd CommandRunner, gatewayContainer string, networks []string) error {
	if cmd == nil {
		cmd = ShellCommandRunner
	}
	var firstErr error
	for _, network := range networks {
		network = strings.TrimSpace(network)
		if network == "" {
			continue
		}
		err := cmd(ctx, "", discardLog, "docker", "network", "connect", network, gatewayContainer)
		if err == nil || alreadyConnected(err) {
			continue
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("join network %s: %w", network, err)
		}
	}
	return firstErr
}

func alreadyConnected(err error) bool {
	return err != nil && strings.Contains(err.Error(), alreadyConnectedSubstring)
}
