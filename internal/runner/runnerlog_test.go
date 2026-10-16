package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/logging"
)

// logLine is a 30 byte record, so a limit of 100 rotates on the fourth.
func logLine(n int) string { return fmt.Sprintf("%029d\n", n) }

func writeLines(t *testing.T, l *RunnerLog, from, to int) {
	t.Helper()
	for n := from; n <= to; n++ {
		_, err := l.Write([]byte(logLine(n)))
		require.NoError(t, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func TestRunnerLog_RotatesAtTheLimitAndKeepsOnePreviousFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner.log")
	l, err := openRunnerLog(path, 100)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	writeLines(t, l, 1, 3)
	_, err = os.Stat(path + ".1")
	assert.ErrorIs(t, err, os.ErrNotExist, "at the limit, not past it, nothing rotates")

	writeLines(t, l, 4, 4)
	assert.Equal(t, logLine(1)+logLine(2)+logLine(3), readFile(t, path+".1"))
	assert.Equal(t, logLine(4), readFile(t, path))

	writeLines(t, l, 5, 7)
	assert.Equal(t, logLine(4)+logLine(5)+logLine(6), readFile(t, path+".1"), "the second rotation replaces the previous file")
	assert.Equal(t, logLine(7), readFile(t, path))
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	assert.Len(t, entries, 2, "one previous file, never more")

	if runtime.GOOS == "windows" {
		return
	}
	for _, p := range []string{path, path + ".1"} {
		info, err := os.Stat(p)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), p)
	}
}

func TestOpenRunnerLog_IsOwnerOnlyAndKeepsWhatARestartLeft(t *testing.T) {
	dir := t.TempDir()
	first, err := OpenRunnerLog(dir)
	require.NoError(t, err)
	writeLines(t, first, 1, 1)
	require.NoError(t, first.Close())

	second, err := OpenRunnerLog(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Close() })
	writeLines(t, second, 2, 2)
	assert.Equal(t, logLine(1)+logLine(2), readFile(t, filepath.Join(dir, runnerLogName)))
	assert.Equal(t, int64(2*len(logLine(1))), second.size, "the size carries over, so the limit counts the old lines")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, runnerLogName))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestOpenRunnerLog_ADirectoryThatIsGoneIsAnError(t *testing.T) {
	_, err := OpenRunnerLog(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestRunnerLog_OpenReadsTheLastLinesAcrossARotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner.log")
	l, err := openRunnerLog(path, 100)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	lines, _, err := l.Open(5, false)
	require.NoError(t, err)
	assert.Empty(t, lines, "a log with nothing in it")

	writeLines(t, l, 1, 5) // the file holds 4 and 5, the previous one 1 to 3
	tests := []struct {
		name string
		n    int
		want []string
	}{
		{"none", 0, nil},
		{"inside the file", 1, []string{logLine(5)}},
		{"all of the file and one more", 3, []string{logLine(3), logLine(4), logLine(5)}},
		{"more than both hold", 50, []string{logLine(1), logLine(2), logLine(3), logLine(4), logLine(5)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, follower, err := l.Open(tt.n, false)
			require.NoError(t, err)
			assert.Nil(t, follower)
			want := make([]string, len(tt.want))
			for i, w := range tt.want {
				want[i] = strings.TrimSuffix(w, "\n")
			}
			assert.ElementsMatch(t, want, got)
			assert.Equal(t, len(want), len(got))
		})
	}
}

func TestRunnerLog_OpenFailsWhenAFileCannotBeRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner.log")
	l, err := openRunnerLog(path, 100)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	require.NoError(t, os.Mkdir(path+".1", 0o700))

	_, _, err = l.Open(10, true)
	require.Error(t, err)
	assert.Empty(t, l.followers, "a failed open subscribes nobody")
}

func TestRunnerLog_RotationThatFailsStillWritesAndStillFollows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner.log")
	l, err := openRunnerLog(path, 100)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	require.NoError(t, os.MkdirAll(filepath.Join(path+".1", "keep"), 0o700), "a rename onto a directory with something in it fails")
	_, follower, err := l.Open(0, true)
	require.NoError(t, err)

	writeLines(t, l, 1, 3)
	_, err = l.Write([]byte(logLine(4)))
	require.Error(t, err, "the failed rotation is reported")
	assert.Contains(t, readFile(t, path), logLine(4), "and the line still lands in the file")
	for n := 1; n <= 4; n++ {
		assert.Equal(t, strings.TrimSuffix(logLine(n), "\n"), <-follower.Lines())
	}
}

func TestRunnerLog_FollowerGetsOnlyLinesAfterItsTailAndStopsOnClose(t *testing.T) {
	l, err := openRunnerLog(filepath.Join(t.TempDir(), "runner.log"), 1<<20)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	writeLines(t, l, 1, 2)

	tail, follower, err := l.Open(10, true)
	require.NoError(t, err)
	assert.Len(t, tail, 2)
	writeLines(t, l, 3, 3)
	assert.Equal(t, strings.TrimSuffix(logLine(3), "\n"), <-follower.Lines(), "line 3 once, and neither 1 nor 2 again")

	follower.Close()
	follower.Close()
	assert.Empty(t, l.followers)
	writeLines(t, l, 4, 4)
	select {
	case line := <-follower.Lines():
		t.Fatalf("a closed follower still got %q", line)
	default:
	}
}

func TestRunnerLog_AFollowerThatFallsBehindCountsWhatItMissed(t *testing.T) {
	l, err := openRunnerLog(filepath.Join(t.TempDir(), "runner.log"), 1<<30)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	_, follower, err := l.Open(0, true)
	require.NoError(t, err)

	writeLines(t, l, 1, followerBuffer+7)
	assert.Equal(t, 7, follower.Dropped())
	assert.Zero(t, follower.Dropped(), "reading the count resets it")
}

func TestRunnerLog_TeeLoggerWritesJSONLinesBesideStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runner.log")
	l, err := openRunnerLog(path, 1<<20)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	logger := logging.NewTee("info", l)

	logger.Debug("below the level")
	logger.With("runner", "alpha").Info("connected", "attempt", 2)

	lines, _, err := l.Open(10, false)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	rendered := renderRunnerLine(lines[0])
	assert.Regexp(t, `^\S+ INFO connected attempt=2 runner=alpha$`, rendered.Line)
	assert.Equal(t, LogStreamStdout, rendered.Stream)
	assert.NotEmpty(t, rendered.TS)
}

func TestRenderRunnerLine(t *testing.T) {
	token := "ses_" + strings.Repeat("a", 43)
	tests := []struct {
		name       string
		raw        string
		wantLine   string
		wantStream string
	}{
		{"keys sorted after time, level and message", `{"time":"2026-10-16T10:00:00Z","level":"INFO","msg":"runner connected","runner":"alpha","attempt":3}`, "2026-10-16T10:00:00Z INFO runner connected attempt=3 runner=alpha", LogStreamStdout},
		{"an error goes to stderr", `{"time":"2026-10-16T10:00:00Z","level":"ERROR","msg":"boom"}`, "2026-10-16T10:00:00Z ERROR boom", LogStreamStderr},
		{"a value with a space is quoted, a nested one is JSON", `{"time":"t","level":"WARN","msg":"m","err":"dial tcp: refused","g":{"a":1},"empty":""}`, `t WARN m empty="" err="dial tcp: refused" g={"a":1}`, LogStreamStdout},
		{"a token in a value is hidden", `{"time":"t","level":"INFO","msg":"dial","url":"wss://example.com/ws?token=` + token + `"}`, `t INFO dial url="wss://example.com/ws?token=[redacted token]"`, LogStreamStdout},
		{"a line that is not JSON passes through, its tokens hidden", "plain " + token, "plain [redacted token]", LogStreamStdout},
		{"an overlong line is cut", `{"time":"t","level":"INFO","msg":"` + strings.Repeat("x", logLineMaxBytes) + `"}`, "", LogStreamStdout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderRunnerLine(tt.raw)
			assert.Equal(t, tt.wantStream, got.Stream)
			if tt.wantLine == "" {
				assert.True(t, strings.HasSuffix(got.Line, "…"))
				assert.Len(t, got.Line, logLineMaxBytes+len("…"))
				return
			}
			assert.Equal(t, tt.wantLine, got.Line)
		})
	}
}
