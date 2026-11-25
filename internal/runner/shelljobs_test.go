package runner

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shellFrames collects what one job sent.
type shellFrames struct {
	mu     sync.Mutex
	frames []Frame
}

func (s *shellFrames) send(f Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames = append(s.frames, f)
}

func (s *shellFrames) output() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	for _, f := range s.frames {
		for _, l := range f.Lines {
			b.WriteString(l.Stream + ":" + l.Line + "\n")
		}
	}
	return b.String()
}

func runTestShell(t *testing.T, script string, timeout int) (Frame, *shellFrames) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the scripts are POSIX shell")
	}
	c := newTestClient("ws://unused", &fakeExecutor{})
	out := &shellFrames{}
	result := c.runShell(t.Context(), Frame{ID: "j1", Script: script, TimeoutSeconds: timeout}, out.send)
	return result, out
}

func TestRunShell_ReportsOutputPerStreamAndTheExitCode(t *testing.T) {
	result, out := runTestShell(t, "echo hello; echo oops >&2; exit 3", 10)

	assert.Equal(t, Frame{Type: FrameShellResult, ID: "j1", Status: ShellStatusFailed, ExitCode: 3}, result)
	assert.Contains(t, out.output(), "stdout:hello\n")
	assert.Contains(t, out.output(), "stderr:oops\n")
	for _, f := range out.frames {
		assert.Equal(t, FrameShellOutput, f.Type)
	}
}

func TestRunShell_SucceedsInTheHomeFolderWithoutNexulSettings(t *testing.T) {
	t.Setenv("NEXUL_CREDENTIAL_FILE", "/secret/credential")
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result, out := runTestShell(t, `pwd; echo "cred=${NEXUL_CREDENTIAL_FILE:-none}"`, 10)

	assert.Equal(t, ShellStatusSucceeded, result.Status)
	assert.Contains(t, out.output(), "stdout:"+home+"\n")
	assert.Contains(t, out.output(), "stdout:cred=none\n")
}

func TestRunShell_TimeoutKillsTheWholeProcessTree(t *testing.T) {
	marker := t.TempDir() + "/child-still-running"
	start := time.Now()

	result, _ := runTestShell(t, "(sleep 3; touch "+marker+") & sleep 30", 1)

	assert.Equal(t, ShellStatusTimedOut, result.Status)
	assert.Less(t, time.Since(start), 5*time.Second)
	time.Sleep(3 * time.Second)
	assert.NoFileExists(t, marker, "the background child outlived the job")
}

func TestRunShell_CancelledByTheServer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the script is POSIX shell")
	}
	c := newTestClient("ws://unused", &fakeExecutor{})
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)

	result := c.runShell(ctx, Frame{ID: "j1", Script: "sleep 30", TimeoutSeconds: 60}, func(Frame) {})

	assert.Equal(t, ShellStatusCancelled, result.Status)
}

func TestRunShell_ANamedShellThatIsNotInstalledFailsToStart(t *testing.T) {
	if _, err := exec.LookPath("pwsh"); err == nil || runtime.GOOS == "windows" {
		t.Skip("pwsh is installed here")
	}
	c := newTestClient("ws://unused", &fakeExecutor{})

	result := c.runShell(t.Context(), Frame{ID: "j1", Shell: "pwsh", Script: "Get-Date", TimeoutSeconds: 5}, func(Frame) {})

	assert.Equal(t, ShellStatusFailed, result.Status)
	assert.Contains(t, result.Error, "only on Windows")
}

func TestPersonalRunner_ReportsCommandsOffAndRefusesThem(t *testing.T) {
	facts := make(chan *Facts, 1)
	answer := make(chan Frame, 1)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameFacts {
				facts <- f.Facts
				_ = wsjson.Write(ctx, conn, Frame{Type: FrameShellRun, ID: "j1", Script: "echo hi", TimeoutSeconds: 5})
			}
			if f.Type == FrameShellResult {
				answer <- f
				return
			}
		}
	})
	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Personal = true
	cancel, done := runClient(t, c)
	defer func() { cancel(); <-done }()

	assert.Equal(t, &ShellFacts{Enabled: false}, (<-facts).Shell)
	f := <-answer
	assert.Equal(t, ShellStatusRefused, f.Status)
	assert.Contains(t, f.Error, "nexul shell on")
}

func TestValidateShellRun(t *testing.T) {
	tests := []struct {
		name string
		f    Frame
		ok   bool
	}{
		{"default shell", Frame{Type: FrameShellRun, ID: "j", Script: "ls", TimeoutSeconds: 60}, true},
		{"pwsh", Frame{Type: FrameShellRun, ID: "j", Shell: "pwsh", Script: "ls", TimeoutSeconds: 3600}, true},
		{"blank script", Frame{Type: FrameShellRun, ID: "j", Script: "  ", TimeoutSeconds: 60}, false},
		{"unknown shell", Frame{Type: FrameShellRun, ID: "j", Shell: "zsh", Script: "ls", TimeoutSeconds: 60}, false},
		{"no timeout", Frame{Type: FrameShellRun, ID: "j", Script: "ls"}, false},
		{"over an hour", Frame{Type: FrameShellRun, ID: "j", Script: "ls", TimeoutSeconds: 3601}, false},
		{"result without status", Frame{Type: FrameShellResult, ID: "j"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ok, tt.f.Validate() == nil)
		})
	}
}

// recordingSink is a ShellSink that keeps what reached it.
type recordingSink struct {
	mu    sync.Mutex
	lines []ContainerLogLine
	done  chan ShellResult
}

func newRecordingSink() *recordingSink { return &recordingSink{done: make(chan ShellResult, 1)} }

func (s *recordingSink) Output(lines []ContainerLogLine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = append(s.lines, lines...)
}

func (s *recordingSink) Done(r ShellResult) { s.done <- r }

func TestHandler_RunShell(t *testing.T) {
	t.Run("an old runner is refused before anything is sent", func(t *testing.T) {
		f := newT3Fixture(t)
		conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version="+lastWithoutShell)
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.CloseNow() })
		eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

		require.ErrorIs(t, f.h.RunShell(t.Context(), "c-laptop", ShellRun{ID: "j1", Script: "ls", Timeout: time.Minute}, newRecordingSink()), ErrCannotRunShell)
	})

	t.Run("output and the result reach the sink; a disconnect ends a running job", func(t *testing.T) {
		f := newT3Fixture(t)
		conn, _, err := dialRunner(t.Context(), f.srv, f.alice, "/ws/runner?version=v0.3.101-beta")
		require.NoError(t, err)
		eventually(t, 2*time.Second, func() bool { return len(f.h.Runners()) == 1 })

		first := newRecordingSink()
		require.NoError(t, f.h.RunShell(t.Context(), "c-laptop", ShellRun{ID: "j1", Script: "ls", Timeout: time.Minute}, first))
		var req Frame
		for req.Type != FrameShellRun {
			require.NoError(t, wsjson.Read(t.Context(), conn, &req))
		}
		assert.Equal(t, 60, req.TimeoutSeconds)
		require.NoError(t, wsjson.Write(t.Context(), conn, Frame{Type: FrameShellOutput, ID: "j1", Lines: []ContainerLogLine{{TS: "t", Stream: LogStreamStdout, Line: "a.txt"}}}))
		require.NoError(t, wsjson.Write(t.Context(), conn, Frame{Type: FrameShellResult, ID: "j1", Status: ShellStatusSucceeded}))
		assert.Equal(t, ShellResult{Status: ShellStatusSucceeded}, <-first.done)
		assert.Equal(t, "a.txt", first.lines[0].Line)

		second := newRecordingSink()
		require.NoError(t, f.h.RunShell(t.Context(), "c-laptop", ShellRun{ID: "j2", Script: "sleep 99", Timeout: time.Minute}, second))
		_ = conn.CloseNow()
		got := <-second.done
		assert.Equal(t, ShellStatusFailed, got.Status)
		assert.Contains(t, got.Error, "disconnected")
	})
}
