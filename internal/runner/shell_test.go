package runner

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellCommandRunner_Success(t *testing.T) {
	var lines []string
	logf := func(s string) { lines = append(lines, s) }
	err := ShellCommandRunner(context.Background(), t.TempDir(), logf, "sh", "-c", "echo hi && printf 'a\\nb\\n'")
	require.NoError(t, err)
	assert.Contains(t, strings.Join(lines, ""), "hi")
	assert.Contains(t, strings.Join(lines, ""), "a\nb\n")
}

func TestShellCommandRunner_NonZeroExit(t *testing.T) {
	var lines []string
	logf := func(s string) { lines = append(lines, s) }
	err := ShellCommandRunner(context.Background(), t.TempDir(), logf, "sh", "-c", "echo before && exit 3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sh -c")
	assert.Contains(t, err.Error(), "before")
	assert.Contains(t, strings.Join(lines, ""), "before")
}

func TestShellCommandRunner_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ShellCommandRunner(ctx, t.TempDir(), discardLog, "sh", "-c", "sleep 10")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestLineWriter_SplitsLines(t *testing.T) {
	var lines []string
	var buf bytes.Buffer
	w := &lineWriter{linef: func(s string) { lines = append(lines, s) }, buf: &buf}

	n, err := w.Write([]byte("one\ntwo\nthree"))
	require.NoError(t, err)
	assert.Equal(t, 13, n)
	assert.Equal(t, []string{"one\n", "two\n", "three"}, lines)
	assert.Equal(t, "one\ntwo\nthree", buf.String())
}

func TestCmdString(t *testing.T) {
	assert.Equal(t, "docker pull img", cmdString("docker", []string{"pull", "img"}))
}
