package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeDocker stands in for `docker logs` at the StreamRunner boundary: it prints its canned output on the two
// pipes, and with --follow stays running until killed.
type fakeDocker struct {
	stdout, stderr string
	killed         chan struct{}

	mu    sync.Mutex
	calls [][]string
}

func (d *fakeDocker) run(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error {
	d.mu.Lock()
	d.calls = append(d.calls, append([]string{name}, args...))
	d.mu.Unlock()
	_, _ = io.WriteString(stdout, d.stdout)
	_, _ = io.WriteString(stderr, d.stderr)
	if !slices.Contains(args, "--follow") {
		return nil
	}
	<-ctx.Done()
	close(d.killed)
	return ctx.Err()
}

func (d *fakeDocker) lastCall() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls[len(d.calls)-1]
}

// connectLogsRunner connects a real client on machine m1 to a real handler and waits until the handler sees it.
func connectLogsRunner(t *testing.T, exec Executor) *Handler {
	t.Helper()
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, newFakeBus(), repo)
	client := NewClient(ClientConfig{
		URL: srv.URL, Credential: repo.enrolled("r-logs", "logs", "m1"), Name: "logs", Logger: testLogger(), Executor: exec,
		HeartbeatInterval: 20 * time.Millisecond, ConnectTimeout: time.Second, BackoffBase: 5 * time.Millisecond, BackoffMax: 50 * time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = client.Run(ctx) }()
	eventually(t, 3*time.Second, func() bool { return h.connOnMachine("m1") != nil })
	return h
}

func drain(t *testing.T, feed *LogFeed) ([]ContainerLogLine, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	var out []ContainerLogLine
	for {
		lines, err := feed.Next(ctx)
		if err != nil {
			return out, err
		}
		out = append(out, lines...)
	}
}

func TestContainerLogs_RoundTripOverTheRunnerSocket(t *testing.T) {
	docker := &fakeDocker{
		stdout: "2026-09-30T10:00:00.000000001Z listening on :8080\n",
		stderr: "2026-09-30T10:00:01.5Z warn: slow query\n",
		killed: make(chan struct{}),
	}
	h := connectLogsRunner(t, NewShellExecutor(nil, ExecutorConfig{Streams: docker.run}, testLogger()))
	want := []ContainerLogLine{
		{TS: "2026-09-30T10:00:00.000000001Z", Stream: LogStreamStdout, Line: "listening on :8080"},
		{TS: "2026-09-30T10:00:01.5Z", Stream: LogStreamStderr, Line: "warn: slow query"},
	}

	t.Run("a snapshot returns the tail with stderr tagged, and ends", func(t *testing.T) {
		feed, err := h.OpenLogs(t.Context(), "m1", "shop-web-1", 5000, false)
		require.NoError(t, err)
		t.Cleanup(feed.Close)

		lines, err := drain(t, feed)
		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, want, lines)
		assert.Equal(t, []string{"docker", "logs", "--timestamps", "--tail", "1000", "shop-web-1"}, docker.lastCall(), "a tail past the cap asks for the cap")
	})

	t.Run("a follow keeps the process running until the viewer closes, which kills it", func(t *testing.T) {
		feed, err := h.OpenLogs(t.Context(), "m1", "shop-web-1", 200, true)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		var got []ContainerLogLine
		for len(got) < len(want) {
			lines, err := feed.Next(ctx)
			require.NoError(t, err)
			got = append(got, lines...)
		}
		assert.Equal(t, want, got)
		assert.Equal(t, []string{"docker", "logs", "--timestamps", "--tail", "200", "--follow", "shop-web-1"}, docker.lastCall())

		feed.Close()
		select {
		case <-docker.killed:
		case <-ctx.Done():
			t.Fatal("closing the feed never killed docker logs on the runner")
		}
		_, err = feed.Next(ctx)
		assert.ErrorIs(t, err, io.EOF, "a closed feed reads as ended")
	})
}

func TestContainerLogs_RunnerRefusesPastSixteenStreams_AndAClosedOneFreesItsSlot(t *testing.T) {
	exec := &fakeExecutor{logsFn: func(ctx context.Context, req Frame, send func(Frame)) {
		send(Frame{Type: FrameLogsChunk, ID: req.ID, Lines: []ContainerLogLine{{Stream: LogStreamStdout, Line: "up"}}})
		<-ctx.Done()
	}}
	h := connectLogsRunner(t, exec)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	var open []*LogFeed
	for range maxLogStreams {
		feed, err := h.OpenLogs(ctx, "m1", "web", 0, true)
		require.NoError(t, err)
		t.Cleanup(feed.Close)
		open = append(open, feed)
	}
	refused, err := h.OpenLogs(ctx, "m1", "web", 0, true)
	require.NoError(t, err)
	_, err = drain(t, refused)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "this runner already serves 16 log streams")

	open[0].Close()
	again, err := h.OpenLogs(ctx, "m1", "web", 0, true)
	require.NoError(t, err)
	t.Cleanup(again.Close)
	lines, err := again.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "up", lines[0].Line)
}

func TestContainerLogs_RunnerGoneOrDropped_EndsTheFeed(t *testing.T) {
	repo := newFakeRunnerRepo()
	h, srv := newIntegrationHandler(t, newFakeBus(), repo)

	_, err := h.OpenLogs(t.Context(), "m1", "web", 10, true)
	require.ErrorIs(t, err, apperrs.ErrRetryable, "no runner on the machine is an offline error")

	conn, _, err := dialRunner(t.Context(), srv, repo.enrolled("r-drop", "drop", "m1"), "")
	require.NoError(t, err)
	eventually(t, 3*time.Second, func() bool { return h.connOnMachine("m1") != nil })
	feed, err := h.OpenLogs(t.Context(), "m1", "web", 10, true)
	require.NoError(t, err)
	var req Frame
	require.NoError(t, wsjson.Read(t.Context(), conn, &req))
	require.Equal(t, FrameLogsRequest, req.Type)

	require.NoError(t, conn.Close(websocket.StatusNormalClosure, ""))
	_, err = drain(t, feed)
	assert.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestLogFeed_SlowViewerGetsASkippedLineInsteadOfAGrowingBuffer(t *testing.T) {
	feed := &LogFeed{ready: make(chan struct{}, 1)}
	batch := func(n int) []ContainerLogLine {
		return slices.Repeat([]ContainerLogLine{{Stream: LogStreamStdout, Line: "x"}}, n)
	}
	feed.push(batch(1500))
	feed.push(batch(1000))

	lines, err := feed.Next(t.Context())
	require.NoError(t, err)
	require.Len(t, lines, logFeedBuffer+1)
	assert.Equal(t, "… 500 lines skipped", lines[logFeedBuffer].Line)

	feed.push(batch(3))
	lines, err = feed.Next(t.Context())
	require.NoError(t, err)
	assert.Len(t, lines, 3, "a viewer that caught up gets every line again")

	feed.finish(nil)
	_, err = feed.Next(t.Context())
	assert.ErrorIs(t, err, io.EOF)
}

func TestShellExecutor_Logs_RateCapsAFollowButNeverASnapshot(t *testing.T) {
	const printed = 100
	output := func(w io.Writer) {
		for i := range printed {
			_, _ = fmt.Fprintf(w, "2026-09-30T10:00:00Z %04d %s\n", i, strings.Repeat("x", 1000))
		}
	}
	tests := []struct {
		name       string
		follow     bool
		wantCapped bool
	}{
		{"follow", true, true},
		{"snapshot", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				docker := func(ctx context.Context, stdout, _ io.Writer, _ string, args ...string) error {
					output(stdout)
					if slices.Contains(args, "--follow") {
						<-ctx.Done()
					}
					return nil
				}
				rec := &frameRecorder{}
				ctx, cancel := context.WithCancel(t.Context())
				done := make(chan struct{})
				go func() {
					NewShellExecutor(nil, ExecutorConfig{Streams: docker}, testLogger()).Logs(ctx, Frame{ID: "l1", Container: "web", Tail: printed, Follow: tt.follow}, rec.send)
					close(done)
				}()
				time.Sleep(time.Second)
				synctest.Wait()
				cancel()
				<-done

				var lines []ContainerLogLine
				for _, f := range rec.all() {
					require.NoError(t, f.Validate())
					data, err := json.Marshal(f)
					require.NoError(t, err)
					assert.Less(t, len(data), 64<<10, "every chunk fits the server's frame read limit")
					lines = append(lines, f.Lines...)
				}
				if !tt.wantCapped {
					assert.Len(t, lines, printed)
					return
				}
				kept := lines[:len(lines)-1]
				sent := 0
				for _, l := range kept {
					sent += encodedSize(l)
				}
				assert.LessOrEqual(t, sent, logRateBytes)
				assert.Equal(t, fmt.Sprintf("… %d lines skipped", printed-len(kept)), lines[len(lines)-1].Line)
			})
		})
	}
}

func TestParseLogLine(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want ContainerLogLine
	}{
		{"docker timestamp", "2026-09-30T10:00:00.123Z GET / 200", ContainerLogLine{TS: "2026-09-30T10:00:00.123Z", Stream: LogStreamStdout, Line: "GET / 200"}},
		{"no timestamp keeps the whole line", "plain words here", ContainerLogLine{Stream: LogStreamStdout, Line: "plain words here"}},
		{"an overlong line is cut", strings.Repeat("y", logLineMaxBytes+5), ContainerLogLine{Stream: LogStreamStdout, Line: strings.Repeat("y", logLineMaxBytes) + "…"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseLogLine(LogStreamStdout, tt.raw))
		})
	}
}

func TestShellExecutor_Logs_DockerFailure_EndsWithItsOwnWords(t *testing.T) {
	docker := func(_ context.Context, _, stderr io.Writer, _ string, _ ...string) error {
		_, _ = io.WriteString(stderr, "Error response from daemon: No such container: web\n")
		return errors.New("docker logs: exit status 1")
	}
	rec := &frameRecorder{}
	NewShellExecutor(nil, ExecutorConfig{Streams: docker}, testLogger()).Logs(t.Context(), Frame{ID: "l1", Container: "web"}, rec.send)

	frames := rec.all()
	require.NotEmpty(t, frames)
	assert.Equal(t, Frame{Type: FrameLogsEnd, ID: "l1", Error: "Error response from daemon: No such container: web"}, frames[len(frames)-1])
}
