package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunShell_Windows_RunsPowerShellAndReportsTheExitCode(t *testing.T) {
	t.Setenv("NEXUL_CREDENTIAL_FILE", `C:\secret\credential`)
	c := newTestClient("ws://unused", &fakeExecutor{})
	out := &shellFrames{}

	result := c.runShell(t.Context(), Frame{ID: "j1", Script: `Write-Output "hello"; [Console]::Error.WriteLine("oops"); Write-Output "cred=$env:NEXUL_CREDENTIAL_FILE"; exit 3`, TimeoutSeconds: 30}, out.send)

	assert.Equal(t, Frame{Type: FrameShellResult, ID: "j1", Status: ShellStatusFailed, ExitCode: 3}, result)
	assert.Contains(t, out.output(), "stdout:hello\n")
	assert.Contains(t, out.output(), "stderr:oops\n")
	assert.Contains(t, out.output(), "stdout:cred=\n")
}

func TestRunShell_Windows_TimeoutKillsTheWholeProcessTree(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "child-still-running")
	c := newTestClient("ws://unused", &fakeExecutor{})
	child := `Start-Sleep 4; New-Item -ItemType File '` + marker + `'`
	script := `Start-Process -NoNewWindow powershell -ArgumentList '-NoProfile','-Command',"` + strings.ReplaceAll(child, `"`, "`\"") + `"; Start-Sleep 60`
	start := time.Now()

	result := c.runShell(t.Context(), Frame{ID: "j1", Script: script, TimeoutSeconds: 2}, func(Frame) {})

	assert.Equal(t, ShellStatusTimedOut, result.Status)
	assert.Less(t, time.Since(start), 10*time.Second)
	time.Sleep(5 * time.Second)
	_, err := os.Stat(marker)
	assert.True(t, os.IsNotExist(err), "the child powershell outlived the job")
}
