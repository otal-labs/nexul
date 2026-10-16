package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Compose labels and the limits a machine runner's container watch works under.
const (
	composeOneOffLabel = "com.docker.compose.oneoff"
	// containerChunk keeps one container_states frame, and one docker inspect call, well under the server's frame limit.
	containerChunk = 50
	// containerStreamSteady is how long a docker events stream must have run for its restart to start the backoff over.
	containerStreamSteady = time.Minute
	containerCommandWait  = 30 * time.Second
)

// containerEventsArgs is the one Docker event stream a machine runner keeps open: deaths, OOM kills and starts.
var containerEventsArgs = []string{
	"events", "--filter", "type=container", "--filter", "event=die", "--filter", "event=oom", "--filter", "event=start", "--format", "{{json .}}",
}

// inspectTemplate projects only what a snapshot needs, so no container environment is ever read.
const inspectTemplate = `{"name":{{json .Name}},"labels":{{json .Config.Labels}},"state":{{json .State.Status}},` +
	`"exit_code":{{.State.ExitCode}},"oom":{{.State.OOMKilled}},"finished_at":{{json .State.FinishedAt}}}`

// ContainerWatch is how a machine runner reads its host's containers: Stream runs `docker events`, Output any other
// docker command. A client without one watches nothing.
type ContainerWatch struct {
	Stream StreamRunner
	Output func(ctx context.Context, args ...string) (string, error)
}

// HostContainerWatch reads the host's own Docker.
func HostContainerWatch() *ContainerWatch {
	return &ContainerWatch{Stream: ShellStreamRunner, Output: hostDockerOutput}
}

// hostDockerOutput returns what a docker command printed, even when it failed after printing some of it: inspecting
// several containers one of which just went still answers for the rest.
func hostDockerOutput(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, containerCommandWait)
	defer cancel()
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return string(out), fmt.Errorf("docker %s: %w", args[0], err)
	}
	return string(out), nil
}

// watchContainers keeps the host's container events flowing to the server until the connection ends. Each time the
// stream (re)starts it also sends a snapshot, so a death while it was down is still reported.
func (c *Client) watchContainers(ctx context.Context, conn *websocket.Conn) {
	backoff := NewBackoff(c.cfg.BackoffBase, c.cfg.BackoffMax, rand.New(rand.NewSource(time.Now().UnixNano())))
	attempt := 0
	for {
		started := time.Now()
		err := c.streamContainerEvents(ctx, conn)
		if ctx.Err() != nil {
			return
		}
		if time.Since(started) > containerStreamSteady {
			attempt = 0
		}
		attempt++
		c.log.Warn("docker events ended; restarting", "runner", c.cfg.Name, "error", err, "attempt", attempt)
		if !backoff.Wait(ctx, attempt) {
			return
		}
	}
}

// streamContainerEvents runs one `docker events` process to its end, forwarding each event, with a snapshot sent
// beside it so no start-up gap goes unreported.
func (c *Client) streamContainerEvents(ctx context.Context, conn *websocket.Conn) error {
	var snapshot sync.WaitGroup
	defer snapshot.Wait()
	snapshot.Go(func() { c.sendContainerStates(ctx, conn) })
	var lastErr string
	stdout := &lineSplitter{emit: func(raw string) { c.forwardContainerEvent(ctx, conn, raw) }}
	stderr := &lineSplitter{emit: func(line string) { lastErr = line }}
	err := c.cfg.Containers.Stream(ctx, stdout, stderr, "docker", containerEventsArgs...)
	stdout.flush()
	stderr.flush()
	if err != nil && lastErr != "" {
		return fmt.Errorf("%w: %s", err, lastErr)
	}
	return err
}

// dockerEvent is the part of a `docker events --format '{{json .}}'` line the runner reads.
type dockerEvent struct {
	Action string `json:"Action"`
	Actor  struct {
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
	Time int64 `json:"time"`
}

// forwardContainerEvent sends one event line as a container_event; lines that are not a die, oom or start of a
// container that compose did not start for a one-off command are dropped.
func (c *Client) forwardContainerEvent(ctx context.Context, conn *websocket.Conn, raw string) {
	var ev dockerEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		c.log.Warn("docker event unparsable", "runner", c.cfg.Name, "error", err)
		return
	}
	attrs := ev.Actor.Attributes
	if !oneOf(ev.Action, ContainerActionDie, ContainerActionOOM, ContainerActionStart) || attrs["name"] == "" || oneOff(attrs) {
		return
	}
	exit, err := strconv.Atoi(attrs["exitCode"])
	if err != nil && attrs["exitCode"] != "" {
		c.log.Warn("docker event exit code unparsable", "runner", c.cfg.Name, "container", attrs["name"], "exit_code", attrs["exitCode"])
	}
	c.sendFrame(ctx, conn, Frame{
		Type: FrameContainerEvent, Container: attrs["name"], Project: attrs[composeProjectLabel], Service: attrs[composeServiceLabel],
		Action: ev.Action, ExitCode: exit, TS: ev.Time,
	})
}

func oneOff(labels map[string]string) bool {
	return strings.EqualFold(labels[composeOneOffLabel], "true")
}

// sendContainerStates reports the host's containers in frames of containerChunk entries.
func (c *Client) sendContainerStates(ctx context.Context, conn *websocket.Conn) {
	states, err := c.containerStates(ctx)
	if err != nil {
		c.log.Warn("container snapshot failed", "runner", c.cfg.Name, "error", err)
	}
	for chunk := range slices.Chunk(states, containerChunk) {
		c.sendFrame(ctx, conn, Frame{Type: FrameContainerStates, States: chunk})
	}
}

// containerStates inspects every container on the host and keeps those that are running and those that are down for a
// reason a crash explains.
func (c *Client) containerStates(ctx context.Context) ([]ContainerState, error) {
	listed, err := c.cfg.Containers.Output(ctx, "ps", "-a", "-q", "--no-trunc")
	if err != nil {
		return nil, err
	}
	var states []ContainerState
	var firstErr error
	for chunk := range slices.Chunk(strings.Fields(listed), containerChunk) {
		out, err := c.cfg.Containers.Output(ctx, append([]string{"inspect", "--format", inspectTemplate}, chunk...)...)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		for line := range strings.Lines(out) {
			if st, ok := c.containerState(line); ok {
				states = append(states, st)
			}
		}
	}
	return states, firstErr
}

// inspectedContainer is one line of the inspect template's output.
type inspectedContainer struct {
	Name       string            `json:"name"`
	Labels     map[string]string `json:"labels"`
	State      string            `json:"state"`
	ExitCode   int               `json:"exit_code"`
	OOM        bool              `json:"oom"`
	FinishedAt string            `json:"finished_at"`
}

func (c *Client) containerState(line string) (ContainerState, bool) {
	if strings.TrimSpace(line) == "" {
		return ContainerState{}, false
	}
	var in inspectedContainer
	if err := json.Unmarshal([]byte(line), &in); err != nil {
		c.log.Warn("container inspect line unparsable", "runner", c.cfg.Name, "error", err)
		return ContainerState{}, false
	}
	name := strings.TrimPrefix(in.Name, "/")
	if name == "" || oneOff(in.Labels) {
		return ContainerState{}, false
	}
	st := ContainerState{Container: name, Project: in.Labels[composeProjectLabel], Service: in.Labels[composeServiceLabel]}
	if in.State == "running" {
		st.State = ContainerStateRunning
		return st, true
	}
	if !oneOf(in.State, "exited", "dead", "restarting") || (in.ExitCode == 0 && !in.OOM) {
		return ContainerState{}, false
	}
	st.State, st.ExitCode, st.OOM = ContainerStateExited, in.ExitCode, in.OOM
	if finished, err := time.Parse(time.RFC3339Nano, in.FinishedAt); err == nil && finished.Unix() > 0 {
		st.FinishedAt = finished.Unix()
	}
	return st, true
}
