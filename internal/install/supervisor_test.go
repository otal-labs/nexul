//go:build !windows

package install

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncedLog is a slog target the test reads once the supervisor has returned.
type syncedLog struct{ bytes.Buffer }

func (l *syncedLog) starts(child string) int {
	return strings.Count(l.String(), `"msg":"child started","child":"`+child+`"`)
}

// supervised is a supervisor running helper children, with the pipe they print their start lines on.
type supervised struct {
	t      *testing.T
	lines  *bufio.Scanner
	stop   chan os.Signal
	result chan bool
	log    *syncedLog
	w      *os.File
	exe    string
}

// startSupervisor runs build's children; copied puts the helper binary in a folder of its own, for a child that replaces it.
func startSupervisor(t *testing.T, copied bool, build func(s *supervised) []childSpec) *supervised {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	exe, err := os.Executable()
	require.NoError(t, err)
	if copied {
		self := exe
		exe = filepath.Join(t.TempDir(), "child")
		require.NoError(t, copyFile(self, exe))
	}
	s := &supervised{t: t, lines: bufio.NewScanner(r), stop: make(chan os.Signal, 1), result: make(chan bool, 1), log: &syncedLog{}, w: w, exe: exe}
	sup := &childSupervisor{log: slog.New(slog.NewJSONHandler(&s.log.Buffer, nil)), children: build(s), grace: time.Hour}
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.result <- sup.run(t.Context(), s.stop)
	}()
	t.Cleanup(func() { // a test that failed early still ends its children before the pipe closes
		select {
		case s.stop <- syscall.SIGTERM:
		default:
		}
		<-done
	})
	return s
}

// child is a helper process in role, whose output goes to the supervised pipe.
func (s *supervised) child(name, role string, ends func() bool, env ...string) childSpec {
	return childSpec{name: name, ends: ends, command: func() *exec.Cmd {
		cmd := exec.Command(s.exe)
		cmd.Env = append(os.Environ(), append([]string{helperRole + "=" + role, "GORACE=atexit_sleep_ms=0"}, env...)...) // the race detector's exit sleep is a second per child
		cmd.Stdout = s.w
		return cmd
	}}
}

// line waits for the next line a child prints.
func (s *supervised) line() string {
	s.t.Helper()
	require.True(s.t, s.lines.Scan(), "a child printed nothing more: %v", s.lines.Err())
	return s.lines.Text()
}

func pidOf(t *testing.T, line string) int {
	t.Helper()
	var pid int
	_, err := fmt.Sscanf(line[strings.Index(line, "pid="):], "pid=%d", &pid)
	require.NoError(t, err, line)
	return pid
}

// TestSupervisor_ARunnerThatReexecsItself_IsNotRestartedAndTheStopReachesTheNewBinary: the runner's update swaps its
// binary and re-execs in place, which keeps its pid. The supervisor has to see one process throughout, never start a
// second, and deliver its stop signal to the new image.
func TestSupervisor_ARunnerThatReexecsItself_IsNotRestartedAndTheStopReachesTheNewBinary(t *testing.T) {
	s := startSupervisor(t, true, func(s *supervised) []childSpec { return []childSpec{s.child("runner", "reexec", nil)} })

	before, after := s.line(), s.line()
	require.Contains(t, after, "gen=1", "the second line is the new binary's")
	require.Equal(t, pidOf(t, before), pidOf(t, after), "the same process, not a restarted one")
	s.stop <- syscall.SIGTERM
	assert.Equal(t, fmt.Sprintf("stopped pid=%d", pidOf(t, after)), s.line(), "the new binary got the stop")

	assert.False(t, <-s.result, "a stop is not a removal")
	assert.Equal(t, 1, s.log.starts("runner"), "the re-exec was not an exit to start again after")
	assert.NotContains(t, s.log.String(), "starting it again")
}

// TestSupervisor_AChildThatExits_IsStartedAgainAloneNoMatterWhyItExited: either process exiting is restarted without
// touching the other, and a runner's clean exit is not a removal unless it asked for one.
func TestSupervisor_AChildThatExits_IsStartedAgainAloneNoMatterWhyItExited(t *testing.T) {
	tests := []struct {
		name         string
		flaky, other string
		exit         string
	}{
		{"the runner fails", "runner", "t3", "1"},
		{"T3 Code fails", "t3", "runner", "1"},
		{"the runner exits cleanly with no removal asked", "runner", "t3", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "marker")
			request := filepath.Join(t.TempDir(), "remove-requested")
			s := startSupervisor(t, false, func(s *supervised) []childSpec {
				flaky := s.child(tt.flaky, "flaky", nil, "HELPER_MARKER="+marker, "HELPER_EXIT="+tt.exit)
				if tt.flaky == "runner" {
					flaky.ends = func() bool { return fileExists(request) }
				}
				return []childSpec{flaky, s.child(tt.other, "serve", nil)}
			})

			var flakyPIDs []int
			for range 3 {
				line := s.line()
				if strings.HasPrefix(line, "flaky") {
					flakyPIDs = append(flakyPIDs, pidOf(t, line))
				}
			}
			require.Len(t, flakyPIDs, 2, "the one that exited was started again, the other was not")
			assert.NotEqual(t, flakyPIDs[0], flakyPIDs[1])
			s.stop <- syscall.SIGTERM
			s.line()
			s.line()

			assert.False(t, <-s.result)
			assert.Equal(t, 2, s.log.starts(tt.flaky))
			assert.Equal(t, 1, s.log.starts(tt.other))
		})
	}
}

// TestSupervisor_ARunnerThatExitsAfterItsRemovalWasAsked_EndsTheSupervisorAndStopsT3Code: removal is the one exit that
// is not restarted, and it takes T3 Code down with it.
func TestSupervisor_ARunnerThatExitsAfterItsRemovalWasAsked_EndsTheSupervisorAndStopsT3Code(t *testing.T) {
	request := filepath.Join(t.TempDir(), "remove-requested")
	s := startSupervisor(t, false, func(s *supervised) []childSpec {
		return []childSpec{
			s.child("runner", "remove", func() bool { return fileExists(request) }, "HELPER_REQUEST="+request),
			s.child("t3", "serve", nil),
		}
	})
	var runner int
	for range 2 {
		if line := s.line(); strings.HasPrefix(line, "remove") {
			runner = pidOf(t, line)
		}
	}
	require.NoError(t, syscall.Kill(runner, syscall.SIGUSR1), "the runner is removed once T3 Code is up")

	assert.True(t, strings.HasPrefix(s.line(), "stopped"), "T3 Code was stopped")
	assert.True(t, <-s.result, "the supervisor ended because the runner was removed")
	assert.Equal(t, 1, s.log.starts("runner"))
	assert.Equal(t, 1, s.log.starts("t3"))
}
