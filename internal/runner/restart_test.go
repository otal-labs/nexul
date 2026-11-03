package runner

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestRestartComputerRunner_AsksOnlyARunnerNewEnoughToRestart(t *testing.T) {
	for _, v := range []string{"", "v0.3.60-beta", lastWithoutRestart} {
		t.Run("version "+v, func(t *testing.T) {
			f := newT3Fixture(t)
			conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version="+v)
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.CloseNow() })
			eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

			require.ErrorIs(t, f.h.RestartComputerRunner(t.Context(), "c-laptop"), ErrCannotRestart)
		})
	}
}

func TestRestartComputerRunner_ReturnsWhatTheRunnerAnswered(t *testing.T) {
	f := newT3Fixture(t)
	conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version=v0.3.78-beta")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })
	eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

	answer := func(reply Frame) error {
		got := make(chan error, 1)
		go func() { got <- f.h.RestartComputerRunner(t.Context(), "c-laptop") }()
		var req Frame
		for req.Type != FrameRestart {
			require.NoError(t, wsjson.Read(t.Context(), conn, &req))
		}
		reply.ID = req.ID
		require.NoError(t, wsjson.Write(t.Context(), conn, reply))
		return <-got
	}

	require.NoError(t, answer(Frame{Type: FrameRestartResult}))
	err = answer(Frame{Type: FrameRestartResult, Error: "find the Nexul app's own binary: gone"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "own binary")
}

func TestRestartComputerRunner_OfflineComputer_IsRetryable(t *testing.T) {
	f := newT3Fixture(t)

	require.ErrorIs(t, f.h.RestartComputerRunner(t.Context(), "c-laptop"), apperrs.ErrRetryable)
}

// TestPersonalRunner_AnswersThenRestartsItself: the runner answers before it restarts, so the server hears it agreed,
// and restarts its own binary with the arguments it was started with.
func TestPersonalRunner_AnswersThenRestartsItself(t *testing.T) {
	restarted := make(chan string, 1)
	stubRestart(t, func() (string, error) { return "/opt/nexul/nexul-runner", nil }, func(exe string, _, _ []string) error {
		restarted <- exe
		return nil
	})

	got := restartAnswer(t)

	assert.Equal(t, Frame{Type: FrameRestartResult, ID: "r1"}, got)
	select {
	case exe := <-restarted:
		assert.Equal(t, "/opt/nexul/nexul-runner", exe)
	case <-time.After(10 * time.Second):
		t.Fatal("the runner never restarted")
	}
}

func TestPersonalRunner_NoBinaryToRestart_SaysWhyAndStays(t *testing.T) {
	stubRestart(t, func() (string, error) { return "", assert.AnError }, func(string, []string, []string) error {
		t.Error("restarted without a binary")
		return nil
	})

	got := restartAnswer(t)

	assert.Contains(t, got.Error, "find the Nexul app's own binary")
}

func stubRestart(t *testing.T, exe func() (string, error), reexec func(string, []string, []string) error) {
	t.Helper()
	origExe, origReexec := runnerExecutableFn, reexecFn
	runnerExecutableFn, reexecFn = exe, reexec
	t.Cleanup(func() { runnerExecutableFn, reexecFn = origExe, origReexec })
}

// restartAnswer asks a personal runner to restart and returns its answer.
func restartAnswer(t *testing.T) Frame {
	t.Helper()
	answer := make(chan Frame, 1)
	personalClient(t, t.TempDir(), "", nil, func(ctx context.Context, conn *websocket.Conn) {
		_ = wsjson.Write(ctx, conn, Frame{Type: FrameRestart, ID: "r1"})
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameRestartResult {
				answer <- f
				return
			}
		}
	})
	select {
	case f := <-answer:
		return f
	case <-time.After(30 * time.Second):
		t.Fatal("no restart_result within 30s")
		return Frame{}
	}
}
