package runner

import (
	"context"
	"io"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// TestPersonalRunner_WithDocker_AnswersContainerFrames: once its facts found Docker, a personal runner lists its
// containers and streams their logs, and still refuses every deploy frame.
func TestPersonalRunner_WithDocker_AnswersContainerFrames(t *testing.T) {
	answers := make(chan []Frame, 1)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameFacts {
				break
			}
		}
		for _, f := range []Frame{{Type: FrameDiscover, ID: "x1"}, {Type: FrameLogsRequest, ID: "l1", Container: "shop-web-1", Tail: 20}, {Type: FrameAssignDeploy, ID: "d1", Service: "api", Image: "api:1"}} {
			_ = wsjson.Write(ctx, conn, f)
		}
		var got []Frame
		for len(got) < 4 {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type != FrameHeartbeat && f.Type != FrameFacts {
				got = append(got, f)
			}
		}
		answers <- got
	})
	exec := &fakeExecutor{
		discoverFn: func(context.Context) (DiscoverReport, error) {
			return DiscoverReport{Containers: []DiscoveredContainer{{Name: "shop-web-1", Image: "shop:1", Status: "running"}}}, nil
		},
		logsFn: func(_ context.Context, req Frame, send func(Frame)) {
			send(Frame{Type: FrameLogsChunk, ID: req.ID, Lines: []ContainerLogLine{{TS: "t1", Stream: LogStreamStdout, Line: "listening"}}})
			send(Frame{Type: FrameLogsEnd, ID: req.ID})
		},
	}
	c := newTestClient(wsURL(srv), exec)
	c.cfg.Personal = true
	c.dockerFacts = func(context.Context) *DockerFacts { return &DockerFacts{Available: true, Version: "27.3.1"} }
	cancel, done := runClient(t, c)

	got := <-answers
	cancel()
	<-done
	assert.ElementsMatch(t, []Frame{
		{Type: FrameDeployResult, ID: "d1", Status: DeployStatusFailed, Error: "a personal runner does not run this"},
		{Type: FrameDiscoverResult, ID: "x1", Containers: []DiscoveredContainer{{Name: "shop-web-1", Image: "shop:1", Status: "running"}}},
		{Type: FrameLogsChunk, ID: "l1", Lines: []ContainerLogLine{{TS: "t1", Stream: LogStreamStdout, Line: "listening"}}},
		{Type: FrameLogsEnd, ID: "l1"},
	}, got)
	assert.Empty(t, exec.deploys)
}

func TestHandler_ComputerContainers(t *testing.T) {
	f := newStreamFixture(t)
	alice := f.repo.personal("r-alice", "u-alice", "c-laptop")
	f.repo.enrolled("r-build", "build-box", "prod")
	frames, conn := f.control(t, alice)
	ctx := t.Context()

	t.Run("lists through the computer's own runner", func(t *testing.T) {
		result := make(chan []DiscoveredContainer, 1)
		go func() {
			got, err := f.h.ComputerContainers(ctx, "c-laptop")
			assert.NoError(t, err)
			result <- got
		}()
		req := nextFrame(t, frames)
		require.Equal(t, FrameDiscover, req.Type)
		require.NoError(t, wsjson.Write(ctx, conn, Frame{Type: FrameDiscoverResult, ID: req.ID, Containers: []DiscoveredContainer{{Name: "db", Image: "postgres:17", Status: "exited"}}}))
		assert.Equal(t, []DiscoveredContainer{{Name: "db", Image: "postgres:17", Status: "exited"}}, <-result)
	})

	t.Run("a refusal from the runner is the caller's conflict", func(t *testing.T) {
		result := make(chan error, 1)
		go func() {
			_, err := f.h.ComputerContainers(ctx, "c-laptop")
			result <- err
		}()
		req := nextFrame(t, frames)
		require.NoError(t, wsjson.Write(ctx, conn, Frame{Type: FrameDiscoverResult, ID: req.ID, Error: noDocker}))
		err := <-result
		require.ErrorIs(t, err, apperrs.ErrConflict)
		assert.Contains(t, err.Error(), noDocker)
	})

	t.Run("a computer whose runner is not connected is retryable", func(t *testing.T) {
		_, err := f.h.ComputerContainers(ctx, "c-desktop")
		require.ErrorIs(t, err, apperrs.ErrRetryable)
		_, err = f.h.OpenComputerLogs(ctx, "c-desktop", "db", 10, false, false)
		require.ErrorIs(t, err, apperrs.ErrRetryable)
	})

	t.Run("logs stream from the computer's runner and end with it", func(t *testing.T) {
		feed, err := f.h.OpenComputerLogs(ctx, "c-laptop", "db", 50, false, false)
		require.NoError(t, err)
		defer feed.Close()
		req := nextFrame(t, frames)
		assert.Equal(t, Frame{Type: FrameLogsRequest, ID: req.ID, Container: "db", Tail: 50}, req)
		require.NoError(t, wsjson.Write(ctx, conn, Frame{Type: FrameLogsChunk, ID: req.ID, Lines: []ContainerLogLine{{TS: "t1", Stream: LogStreamStderr, Line: "ready"}}}))
		require.NoError(t, wsjson.Write(ctx, conn, Frame{Type: FrameLogsEnd, ID: req.ID}))
		lines, err := feed.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []ContainerLogLine{{TS: "t1", Stream: LogStreamStderr, Line: "ready"}}, lines)
		_, err = feed.Next(ctx)
		assert.ErrorIs(t, err, io.EOF)
	})
}
