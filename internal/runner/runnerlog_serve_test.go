package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func record(msg string) string {
	return `{"time":"2026-10-16T10:00:00Z","level":"INFO","msg":"` + msg + `"}` + "\n"
}

func rendered(msg string) string { return "2026-10-16T10:00:00Z INFO " + msg }

func lineTexts(lines []ContainerLogLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Line
	}
	return out
}

// connectPersonal runs a personal runner holding log on alice's computer c-laptop against a real handler.
func connectPersonal(t *testing.T, log *RunnerLog) *streamFixture {
	t.Helper()
	f := newStreamFixture(t)
	credential := f.repo.personal("r-alice", "u-alice", "c-laptop")
	c := newTestClient(wsURL(f.srv)+"/ws/runner", &fakeExecutor{})
	c.cfg.Credential, c.cfg.Personal, c.cfg.RunnerLog = credential, true, log
	runClient(t, c)
	eventually(t, 3*time.Second, func() bool {
		_, err := f.h.computerConn(t.Context(), "c-laptop")
		return err == nil
	})
	return f
}

func newTestRunnerLog(t *testing.T, msgs ...string) *RunnerLog {
	t.Helper()
	log, err := openRunnerLog(filepath.Join(t.TempDir(), runnerLogName), 1<<20)
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	for _, m := range msgs {
		_, err := log.Write([]byte(record(m)))
		require.NoError(t, err)
	}
	return log
}

func followers(l *RunnerLog) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.followers)
}

func TestPersonalRunner_ServesItsOwnLog_WithoutDocker(t *testing.T) {
	log := newTestRunnerLog(t, "one", "two", "three")
	f := connectPersonal(t, log)

	t.Run("a snapshot is the last lines, rendered, and ends", func(t *testing.T) {
		feed, err := f.h.OpenComputerRunnerLogs(t.Context(), "c-laptop", 2, false)
		require.NoError(t, err)
		t.Cleanup(feed.Close)
		lines, err := drain(t, feed)
		require.ErrorIs(t, err, io.EOF)
		assert.Equal(t, []string{rendered("two"), rendered("three")}, lineTexts(lines))
		assert.Equal(t, "2026-10-16T10:00:00Z", lines[0].TS)
	})

	t.Run("a tail of zero returns nothing", func(t *testing.T) {
		feed, err := f.h.OpenComputerRunnerLogs(t.Context(), "c-laptop", 0, false)
		require.NoError(t, err)
		t.Cleanup(feed.Close)
		lines, err := drain(t, feed)
		require.ErrorIs(t, err, io.EOF)
		assert.Empty(t, lines)
	})

	t.Run("a follow gets the tail and then each new record, and closing it unsubscribes", func(t *testing.T) {
		feed, err := f.h.OpenComputerRunnerLogs(t.Context(), "c-laptop", 1, true)
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		var got []string
		for len(got) < 1 {
			lines, err := feed.Next(ctx)
			require.NoError(t, err)
			got = append(got, lineTexts(lines)...)
		}
		assert.Equal(t, []string{rendered("three")}, got)

		_, err = log.Write([]byte(record(`token ses_` + strings.Repeat("a", 43))))
		require.NoError(t, err)
		lines, err := feed.Next(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{rendered("token [redacted token]")}, lineTexts(lines), "a token never leaves the computer")

		assert.Equal(t, 1, followers(log))
		feed.Close()
		eventually(t, 3*time.Second, func() bool { return followers(log) == 0 })
	})
}

func TestPersonalRunner_ServesAtMostTheTailCap(t *testing.T) {
	var msgs []string
	for range maxLogTail + 20 {
		msgs = append(msgs, "line")
	}
	f := connectPersonal(t, newTestRunnerLog(t, msgs...))
	feed, err := f.h.OpenComputerRunnerLogs(t.Context(), "c-laptop", 5000, false)
	require.NoError(t, err)
	t.Cleanup(feed.Close)
	lines, err := drain(t, feed)
	require.ErrorIs(t, err, io.EOF)
	assert.Len(t, lines, maxLogTail)
}

func TestContainerLogStream_CapFromRateCapsOnlyWhatFollowsTheSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &frameRecorder{}
		out := &containerLogStream{id: "l1", send: rec.send}
		big := ContainerLogLine{Stream: LogStreamStdout, Line: strings.Repeat("x", 1000)}
		const each = 100
		for range each {
			out.addLine(big)
		}
		out.capFrom()
		for range each {
			out.addLine(big)
		}
		out.close()

		var lines []ContainerLogLine
		for _, f := range rec.all() {
			lines = append(lines, f.Lines...)
		}
		snapshot, followed := lines[:each], lines[each:]
		assert.Equal(t, slices.Repeat([]ContainerLogLine{big}, each), snapshot, "the snapshot is never rate-capped")
		kept := followed[:len(followed)-1]
		assert.Less(t, len(kept), each, "the follow is")
		assert.Equal(t, fmt.Sprintf("… %d lines skipped", each-len(kept)), followed[len(followed)-1].Line)
	})
}

func TestPersonalRunner_TheLogOfARunnerWithoutOneIsRefused(t *testing.T) {
	tests := []struct {
		name     string
		personal bool
		log      *RunnerLog
	}{
		{"a personal runner that could not open its file", true, nil},
		{"a runner that is not personal", false, newTestRunnerLog(t, "one")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answers := make(chan Frame, 1)
			srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
				_ = wsjson.Write(ctx, conn, Frame{Type: FrameLogsRequest, ID: "l1", Target: LogTargetRunner, Tail: 10})
				for {
					var f Frame
					if err := wsjson.Read(ctx, conn, &f); err != nil {
						return
					}
					if f.Type == FrameLogsEnd || f.Type == FrameLogsChunk {
						answers <- f
						return
					}
				}
			})
			c := newTestClient(wsURL(srv), &fakeExecutor{})
			c.cfg.Personal, c.cfg.RunnerLog = tt.personal, tt.log
			cancel, done := runClient(t, c)
			got := <-answers
			cancel()
			<-done
			assert.Equal(t, Frame{Type: FrameLogsEnd, ID: "l1", Error: "this runner keeps no log of its own"}, got)
		})
	}
}

func TestPersonalRunner_AnUnreadableLogEndsTheStreamWithoutNamingAPath(t *testing.T) {
	log := newTestRunnerLog(t, "one")
	require.NoError(t, os.Mkdir(log.path+".1", 0o700))
	f := connectPersonal(t, log)

	feed, err := f.h.OpenComputerRunnerLogs(t.Context(), "c-laptop", 10, false)
	require.NoError(t, err)
	t.Cleanup(feed.Close)
	_, err = drain(t, feed)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "the runner could not read its log")
	assert.NotContains(t, err.Error(), log.path)
}

func TestPersonalRunner_RefusesTheRunnerLogOfAComputerWithNoRunner(t *testing.T) {
	f := newStreamFixture(t)
	_, err := f.h.OpenComputerRunnerLogs(t.Context(), "c-none", 10, false)
	require.ErrorIs(t, err, apperrs.ErrRetryable)
}

// fakeInspect answers `docker inspect` with env and records every command it ran.
func fakeInspect(env string, err error, ran *[]string) CommandRunner {
	return func(_ context.Context, _ string, logf func(string), name string, args ...string) error {
		*ran = append(*ran, name+" "+strings.Join(args, " "))
		logf(env)
		return err
	}
}

func TestShellExecutor_Logs_MasksTheContainersOwnEnvWhenAsked(t *testing.T) {
	const secret = "hunter2-hunter2"
	output := "2026-10-16T10:00:00Z login with " + secret + " as abc and PORT 8080\n"
	env := `["PATH=/usr/bin","DB_PASSWORD=` + secret + `","SHORT=abc","PORT=8080","NOEQUALS"]`

	t.Run("each value of six characters or more is hidden, shorter ones stay", func(t *testing.T) {
		var ran []string
		docker := &fakeDocker{stdout: output, killed: make(chan struct{})}
		rec := &frameRecorder{}
		NewShellExecutor(fakeInspect(env, nil, &ran), ExecutorConfig{Streams: docker.run}, testLogger()).
			Logs(t.Context(), Frame{ID: "l1", Container: "web", Mask: true}, rec.send)

		assert.Equal(t, []string{"docker inspect --format {{json .Config.Env}} web"}, ran)
		var lines []ContainerLogLine
		for _, fr := range rec.all() {
			lines = append(lines, fr.Lines...)
		}
		require.Len(t, lines, 1)
		assert.Equal(t, "login with •••• as abc and PORT 8080", lines[0].Line)
		assert.NotContains(t, lines[0].Line, secret)
	})

	t.Run("without the flag nothing is inspected or hidden", func(t *testing.T) {
		var ran []string
		docker := &fakeDocker{stdout: output, killed: make(chan struct{})}
		rec := &frameRecorder{}
		NewShellExecutor(fakeInspect(env, nil, &ran), ExecutorConfig{Streams: docker.run}, testLogger()).
			Logs(t.Context(), Frame{ID: "l1", Container: "web"}, rec.send)
		assert.Empty(t, ran)
		assert.Contains(t, rec.all()[0].Lines[0].Line, secret)
	})

	t.Run("an env that cannot be read ends the stream before any line is read", func(t *testing.T) {
		for name, tc := range map[string]struct {
			env string
			err error
		}{
			"inspect fails":   {"", errors.New("No such container")},
			"not a JSON list": {"nope", nil},
		} {
			t.Run(name, func(t *testing.T) {
				var ran []string
				docker := &fakeDocker{stdout: output, killed: make(chan struct{})}
				rec := &frameRecorder{}
				NewShellExecutor(fakeInspect(tc.env, tc.err, &ran), ExecutorConfig{Streams: docker.run}, testLogger()).
					Logs(t.Context(), Frame{ID: "l1", Container: "web", Mask: true}, rec.send)
				assert.Equal(t, []Frame{{Type: FrameLogsEnd, ID: "l1", Error: "could not read the container's environment to hide its secrets"}}, rec.all())
				assert.Empty(t, docker.calls, "docker logs never ran, so nothing unmasked left")
			})
		}
	})
}
