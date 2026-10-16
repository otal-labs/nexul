package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeHostDocker stands in for the host's Docker: Stream prints streamOut for each run of `docker events` and then
// either ends with streamErr or stays up until killed, and Output answers `ps` and `inspect` from canned text.
type fakeHostDocker struct {
	streamOut string
	streamErr error
	// endingRuns is how many runs end with streamErr before one stays up.
	endingRuns int
	ps         string
	inspect    func(ids []string) string
	psErr      error
	inspectErr error

	mu      sync.Mutex
	streams int
	outputs [][]string
	argv    []string
}

func (d *fakeHostDocker) watch() *ContainerWatch {
	return &ContainerWatch{Stream: d.stream, Output: d.output}
}

func (d *fakeHostDocker) stream(ctx context.Context, stdout, _ io.Writer, name string, args ...string) error {
	d.mu.Lock()
	d.streams++
	run := d.streams
	d.argv = append([]string{name}, args...)
	d.mu.Unlock()
	_, _ = io.WriteString(stdout, d.streamOut)
	if run <= d.endingRuns {
		return d.streamErr
	}
	<-ctx.Done()
	return ctx.Err()
}

func (d *fakeHostDocker) output(_ context.Context, args ...string) (string, error) {
	d.mu.Lock()
	d.outputs = append(d.outputs, args)
	d.mu.Unlock()
	if args[0] == "ps" {
		return d.ps, d.psErr
	}
	return d.inspect(args[3:]), d.inspectErr
}

func (d *fakeHostDocker) streamRuns() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.streams
}

func (d *fakeHostDocker) inspectCalls() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, a := range d.outputs {
		if a[0] == "inspect" {
			n++
		}
	}
	return n
}

func eventLine(action string, attrs map[string]string) string {
	b, _ := json.Marshal(map[string]any{
		"status": action, "Type": "container", "Action": action, "time": 1700000000,
		"Actor": map[string]any{"ID": "abc", "Attributes": attrs},
	})
	return string(b) + "\n"
}

func inspectLine(name, state string, exit int, oom bool, finished string, labels map[string]string) string {
	b, _ := json.Marshal(map[string]any{"name": "/" + name, "labels": labels, "state": state, "exit_code": exit, "oom": oom, "finished_at": finished})
	return string(b) + "\n"
}

// framesFrom starts a machine runner against a socket that records every frame it sends.
func framesFrom(t *testing.T, w *ContainerWatch, personal bool) <-chan Frame {
	t.Helper()
	frames := make(chan Frame, 256)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type != FrameHeartbeat && f.Type != FrameFacts {
				frames <- f
			}
		}
	})
	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Containers, c.cfg.Personal = w, personal
	c.findT3 = func(string) string { return "" }
	runClient(t, c)
	return frames
}

func collect(t *testing.T, frames <-chan Frame, n int) []Frame {
	t.Helper()
	var out []Frame
	for range n {
		out = append(out, nextFrame(t, frames))
	}
	return out
}

func TestContainerWatch_ForwardsDieOOMStartKillStopAndASnapshot(t *testing.T) {
	compose := map[string]string{"name": "shop-web-1", composeProjectLabel: "shop", composeServiceLabel: "web"}
	docker := &fakeHostDocker{
		streamOut: eventLine("die", withAttr(compose, "exitCode", "137")) +
			eventLine("oom", compose) +
			eventLine("start", compose) +
			eventLine("kill", withAttr(compose, "signal", "15")) +
			eventLine("stop", compose) +
			eventLine("destroy", compose) +
			eventLine("die", map[string]string{"name": "shop-seed-run-1", composeProjectLabel: "shop", composeOneOffLabel: "True", "exitCode": "1"}) +
			eventLine("exec_start: sh", compose) +
			eventLine("die", map[string]string{"exitCode": "1"}) +
			eventLine("die", map[string]string{"name": "api", "exitCode": "bad"}) +
			"not json\n",
		ps: "id1\nid2\n",
		inspect: func([]string) string {
			return inspectLine("shop-web-1", "running", 0, false, "0001-01-01T00:00:00Z", map[string]string{composeProjectLabel: "shop", composeServiceLabel: "web"}) +
				inspectLine("shop-db-1", "exited", 1, false, "2026-10-01T10:42:00Z", map[string]string{composeProjectLabel: "shop", composeServiceLabel: "db"})
		},
	}
	frames := collect(t, framesFrom(t, docker.watch(), false), 7)

	got := map[FrameType][]Frame{}
	for _, f := range frames {
		got[f.Type] = append(got[f.Type], f)
	}
	events := got[FrameContainerEvent]
	require.Len(t, events, 6)
	assert.Equal(t, Frame{Type: FrameContainerEvent, Container: "shop-web-1", Project: "shop", Service: "web", Action: "die", ExitCode: 137, TS: 1700000000}, events[0])
	assert.Equal(t, "oom", events[1].Action)
	assert.Equal(t, "start", events[2].Action)
	assert.Equal(t, "kill", events[3].Action)
	assert.Equal(t, "stop", events[4].Action)
	assert.Equal(t, Frame{Type: FrameContainerEvent, Container: "api", Action: "die", TS: 1700000000}, events[5], "an exit code that does not parse is zero; a run container has no compose labels")

	require.Len(t, got[FrameContainerStates], 1)
	assert.Equal(t, []ContainerState{
		{Container: "shop-web-1", Project: "shop", Service: "web", State: ContainerStateRunning},
		{Container: "shop-db-1", Project: "shop", Service: "db", State: ContainerStateExited, ExitCode: 1, FinishedAt: time.Date(2026, 10, 1, 10, 42, 0, 0, time.UTC).Unix()},
	}, got[FrameContainerStates][0].States)

	docker.mu.Lock()
	defer docker.mu.Unlock()
	assert.Equal(t, append([]string{"docker"}, containerEventsArgs...), docker.argv)
	assert.Equal(t, []string{"event=die", "event=oom", "event=start", "event=kill", "event=stop"}, filterValues(docker.argv, "event="))
}

func withAttr(attrs map[string]string, k, v string) map[string]string {
	out := map[string]string{k: v}
	for key, val := range attrs {
		out[key] = val
	}
	return out
}

func filterValues(argv []string, prefix string) []string {
	var out []string
	for _, a := range argv {
		if strings.HasPrefix(a, prefix) {
			out = append(out, a)
		}
	}
	return out
}

func TestContainerWatch_RestartsTheStreamWithBackoffAndResnapshotsEachTime(t *testing.T) {
	docker := &fakeHostDocker{
		streamErr: errors.New("exit status 1"), endingRuns: 3,
		ps:      "id1\n",
		inspect: func([]string) string { return inspectLine("api", "running", 0, false, "", nil) },
	}
	frames := framesFrom(t, docker.watch(), false)

	for range 4 {
		f := nextFrame(t, frames)
		assert.Equal(t, FrameContainerStates, f.Type, "every (re)start of the stream reports what the host holds")
	}
	assert.Equal(t, 4, docker.streamRuns(), "three ended runs were restarted, the fourth stays up")
}

func TestContainerWatch_ANonMachineRunnerWatchesNothing(t *testing.T) {
	tests := []struct {
		name     string
		watch    func(*fakeHostDocker) *ContainerWatch
		personal bool
	}{
		{"a personal runner", (*fakeHostDocker).watch, true},
		{"a client with no watch configured", func(*fakeHostDocker) *ContainerWatch { return nil }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docker := &fakeHostDocker{ps: "id1\n", inspect: func([]string) string { return "" }}
			frames := framesFrom(t, tt.watch(docker), tt.personal)
			select {
			case f := <-frames:
				t.Fatalf("unexpected %s frame", f.Type)
			case <-time.After(200 * time.Millisecond):
			}
			assert.Zero(t, docker.streamRuns())
		})
	}
}

func TestContainerStates_KeepsRunningAndCrashedContainers(t *testing.T) {
	compose := map[string]string{composeProjectLabel: "shop", composeServiceLabel: "web"}
	tests := []struct {
		name string
		line string
		want *ContainerState
	}{
		{"running", inspectLine("shop-web-1", "running", 0, false, "0001-01-01T00:00:00Z", compose), &ContainerState{Container: "shop-web-1", Project: "shop", Service: "web", State: ContainerStateRunning}},
		{"exited non-zero", inspectLine("shop-web-1", "exited", 137, false, "2026-10-01T10:42:00.5Z", compose), &ContainerState{Container: "shop-web-1", Project: "shop", Service: "web", State: ContainerStateExited, ExitCode: 137, FinishedAt: 1790851320}},
		{"restarting after a crash", inspectLine("shop-web-1", "restarting", 1, false, "2026-10-01T10:42:00Z", compose), &ContainerState{Container: "shop-web-1", Project: "shop", Service: "web", State: ContainerStateExited, ExitCode: 1, FinishedAt: 1790851320}},
		{"dead by OOM with a zero code", inspectLine("shop-web-1", "dead", 0, true, "0001-01-01T00:00:00Z", compose), &ContainerState{Container: "shop-web-1", Project: "shop", Service: "web", State: ContainerStateExited, OOM: true}},
		{"a run container has no labels", inspectLine("api", "exited", 2, false, "2026-10-01T10:42:00Z", nil), &ContainerState{Container: "api", State: ContainerStateExited, ExitCode: 2, FinishedAt: 1790851320}},
		{"exited cleanly", inspectLine("shop-web-1", "exited", 0, false, "2026-10-01T10:42:00Z", compose), nil},
		{"created, never started", inspectLine("shop-web-1", "created", 0, false, "", compose), nil},
		{"paused", inspectLine("shop-web-1", "paused", 0, false, "", compose), nil},
		{"a one-off command", inspectLine("shop-seed-1", "running", 0, false, "", map[string]string{composeOneOffLabel: "True"}), nil},
		{"no name", inspectLine("", "running", 0, false, "", nil), nil},
		{"a blank line", "\n", nil},
		{"not json", "oops\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docker := &fakeHostDocker{ps: "id1\n", inspect: func([]string) string { return tt.line }}
			c := newTestClient("ws://unused", &fakeExecutor{})
			c.cfg.Containers = docker.watch()
			got, err := c.containerStates(t.Context())
			require.NoError(t, err)
			if tt.want == nil {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, []ContainerState{*tt.want}, got)
		})
	}
}

func TestContainerStates_ReadsInChunksAndSurvivesAVanishedContainer(t *testing.T) {
	var ids []string
	for i := range 120 {
		ids = append(ids, fmt.Sprintf("id%03d", i))
	}
	docker := &fakeHostDocker{
		ps: strings.Join(ids, "\n") + "\n",
		inspect: func(chunk []string) string {
			var out strings.Builder
			for _, id := range chunk {
				out.WriteString(inspectLine("c-"+id, "running", 0, false, "", nil))
			}
			return out.String()
		},
		inspectErr: errors.New("docker inspect: No such object: id050"),
	}
	c := newTestClient("ws://unused", &fakeExecutor{})
	c.cfg.Containers = docker.watch()

	got, err := c.containerStates(t.Context())

	require.ErrorContains(t, err, "No such object")
	assert.Len(t, got, 120, "what inspect printed is kept even when it failed on one container")
	assert.Equal(t, 3, docker.inspectCalls(), "50, 50 and 20 containers")
}

func TestContainerStates_APsFailureReportsNothing(t *testing.T) {
	docker := &fakeHostDocker{psErr: errors.New("docker ps: cannot connect to the daemon"), inspect: func([]string) string { return "" }}
	c := newTestClient("ws://unused", &fakeExecutor{})
	c.cfg.Containers = docker.watch()
	got, err := c.containerStates(t.Context())
	require.ErrorContains(t, err, "cannot connect")
	assert.Empty(t, got)
	assert.Zero(t, docker.inspectCalls())
}

func TestSendContainerStates_SplitsIntoFramesThatFitTheServersLimit(t *testing.T) {
	var lines strings.Builder
	for i := range 120 {
		lines.WriteString(inspectLine(fmt.Sprintf("a-rather-long-container-name-%03d", i), "exited", 1, false, "2026-10-01T10:42:00Z",
			map[string]string{composeProjectLabel: "a-rather-long-project-name", composeServiceLabel: "a-rather-long-service-name"}))
	}
	docker := &fakeHostDocker{ps: "id1\n", inspect: func([]string) string { return lines.String() }}
	frames := framesFrom(t, docker.watch(), false)

	var sizes []int
	for range 3 {
		f := nextFrame(t, frames)
		require.Equal(t, FrameContainerStates, f.Type)
		raw, err := json.Marshal(f)
		require.NoError(t, err)
		assert.Less(t, len(raw), 64<<10, "the server closes a connection that sends a frame over its read limit")
		sizes = append(sizes, len(f.States))
	}
	assert.Equal(t, []int{50, 50, 20}, sizes)
}

func TestHostDockerOutput_NamesTheCommandThatFailed(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := hostDockerOutput(t.Context(), "ps")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker ps")
}

func TestHost_ContainerFramesBecomeInternalBusEvents(t *testing.T) {
	f := newT3Fixture(t)
	_, deploy := f.control(t, f.deploy)
	_, alice := f.control(t, f.alice)
	ctx := t.Context()
	state := ContainerState{Container: "shop-web-1", Project: "shop", Service: "web", State: ContainerStateExited, ExitCode: 137, FinishedAt: 5}

	require.NoError(t, wsjson.Write(ctx, alice, Frame{Type: FrameContainerEvent, Container: "mine", Action: ContainerActionDie, ExitCode: 1}))
	require.NoError(t, wsjson.Write(ctx, alice, Frame{Type: FrameContainerStates, States: []ContainerState{state}}))
	require.NoError(t, wsjson.Write(ctx, deploy, Frame{Type: FrameContainerEvent, Container: "shop-web-1", Project: "shop", Service: "web", Action: ContainerActionDie, ExitCode: 137}))
	require.NoError(t, wsjson.Write(ctx, deploy, Frame{Type: FrameContainerStates, States: []ContainerState{state}}))

	eventually(t, 2*time.Second, func() bool {
		return len(f.bus.topicEvents(TopicContainerEvent)) > 0 && len(f.bus.topicEvents(TopicContainerStates)) > 0
	})
	require.Len(t, f.bus.topicEvents(TopicContainerEvent), 1, "a personal runner's frames are dropped")
	require.Len(t, f.bus.topicEvents(TopicContainerStates), 1)
	var ev ContainerEvent
	require.NoError(t, json.Unmarshal(f.bus.topicEvents(TopicContainerEvent)[0].Payload, &ev))
	assert.Equal(t, ContainerEvent{Machine: "m1", Container: "shop-web-1", Project: "shop", Service: "web", Action: "die", ExitCode: 137}, ev, "the machine is the runner's own, never the frame's")
	var states ContainerStatesEvent
	require.NoError(t, json.Unmarshal(f.bus.topicEvents(TopicContainerStates)[0].Payload, &states))
	assert.Equal(t, ContainerStatesEvent{Machine: "m1", States: []ContainerState{state}}, states)
}

func TestHost_AContainerFramePublishFailureIsLoggedNotFatal(t *testing.T) {
	f := newT3Fixture(t)
	f.bus.publishErr = errors.New("bus closed")
	_, deploy := f.control(t, f.deploy)
	require.NoError(t, wsjson.Write(t.Context(), deploy, Frame{Type: FrameContainerEvent, Container: "web", Action: ContainerActionStart}))
	require.NoError(t, wsjson.Write(t.Context(), deploy, Frame{Type: FrameHeartbeat, TS: time.Now().Unix()}))
	eventually(t, 2*time.Second, func() bool { return f.repo.heartbeatCount() == 1 })
}
