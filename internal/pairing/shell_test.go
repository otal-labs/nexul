package pairing

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

// fakeShells stands in for the computers' runners: it keeps each run's sink until the test ends it.
type fakeShells struct {
	mu        sync.Mutex
	runs      []ShellRun
	sinks     map[string]ShellSink
	cancelled []string
	runErr    error
	started   chan string
}

func newFakeShells() *fakeShells {
	return &fakeShells{sinks: map[string]ShellSink{}, started: make(chan string, 16)}
}

func (f *fakeShells) RunShell(_ context.Context, _ string, run ShellRun, sink ShellSink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.runErr != nil {
		return f.runErr
	}
	f.runs = append(f.runs, run)
	f.sinks[run.ID] = sink
	f.started <- run.ID
	return nil
}

func (f *fakeShells) CancelShell(_ context.Context, jobID string) error {
	f.mu.Lock()
	sink := f.sinks[jobID]
	f.cancelled = append(f.cancelled, jobID)
	f.mu.Unlock()
	if sink != nil {
		sink.Done(ShellEnd{Status: ShellCancelled})
	}
	return nil
}

func (f *fakeShells) sink(id string) ShellSink {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sinks[id]
}

func (f *fakeShells) runCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.runs)
}

func newShellService(t *testing.T) (*Service, *fakeRepo, *fakeShells) {
	t.Helper()
	repo, shells := newFakeRepo(), newFakeShells()
	repo.computers["c-laptop"] = Computer{ID: "c-laptop", UserID: "alice", Name: "laptop", ShellEnabled: true}
	s := NewService(Config{Repo: repo, Shells: shells})
	t.Cleanup(s.shellRuns.Wait)
	return s, repo, shells
}

func waitStarted(t *testing.T, shells *fakeShells) string {
	t.Helper()
	select {
	case id := <-shells.started:
		return id
	case <-time.After(2 * time.Second):
		t.Fatal("no command reached the runner")
		return ""
	}
}

func TestRunCommand_RecordsWhatRanAndHowItEnded(t *testing.T) {
	s, repo, shells := newShellService(t)

	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "go test ./...", Cwd: "~/code/app"})
	require.NoError(t, err)
	assert.Equal(t, ShellQueued, job.Status)
	assert.Equal(t, 60, job.TimeoutSeconds)
	assert.Equal(t, AdapterWeb, job.Adapter)
	id := waitStarted(t, shells)
	assert.Equal(t, job.ID, id)
	assert.Equal(t, ShellRunning, repo.shellJob(id).Status)

	shells.sink(id).Output([]containerlog.Line{{Stream: "stdout", Line: "ok  app 0.4s"}, {Stream: "stderr", Line: "warn"}})
	shells.sink(id).Done(ShellEnd{Status: ShellSucceeded})

	got, err := s.WaitShellJob(t.Context(), "alice", id)
	require.NoError(t, err)
	assert.Equal(t, ShellSucceeded, got.Status)
	require.NotNil(t, got.ExitCode)
	assert.Equal(t, 0, *got.ExitCode)
	assert.Equal(t, "ok  app 0.4s\nwarn\n", got.Output)
	assert.NotNil(t, got.EndedAt)
}

func TestRunCommand_RefusedByTheComputerIsRecordedAsRefused(t *testing.T) {
	s, repo, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "ls"})
	require.NoError(t, err)
	waitStarted(t, shells)

	shells.sink(job.ID).Done(ShellEnd{Status: ShellRefused, Error: "commands are turned off on this computer"})

	got := repo.shellJob(job.ID)
	assert.Equal(t, ShellRefused, got.Status)
	assert.Nil(t, got.ExitCode)
	assert.Contains(t, got.Error, "turned off")
}

func TestShellJobs_OnlyTheOwnerStartsOrReadsThem(t *testing.T) {
	s, _, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "whoami"})
	require.NoError(t, err)
	waitStarted(t, shells)

	_, err = s.RunCommand(t.Context(), "mallory", "c-laptop", RunCommandInput{Command: "whoami"})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.GetShellJob(t.Context(), "mallory", job.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, _, err = s.ListShellJobs(t.Context(), "mallory", "c-laptop", paging.Window{})
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.CancelShellJob(t.Context(), "mallory", job.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.OpenShellOutput(t.Context(), "mallory", job.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, 1, shells.runCount())
}

func TestRunCommand_RunsFourAtATimePerComputer(t *testing.T) {
	s, repo, shells := newShellService(t)
	var ids []string
	for range 5 {
		job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "sleep 1"})
		require.NoError(t, err)
		ids = append(ids, job.ID)
	}
	for range 4 {
		waitStarted(t, shells)
	}
	assert.Equal(t, 4, shells.runCount())
	assert.Equal(t, ShellQueued, repo.shellJob(ids[4]).Status)

	shells.sink(ids[0]).Done(ShellEnd{Status: ShellSucceeded})

	assert.Equal(t, ids[4], waitStarted(t, shells))
}

func TestRunCommand_TheSwitchRefusesNewOnesAndStopsTheRest(t *testing.T) {
	s, repo, shells := newShellService(t)
	var ids []string
	for range 5 {
		job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "sleep 9"})
		require.NoError(t, err)
		ids = append(ids, job.ID)
	}
	for range 4 {
		waitStarted(t, shells)
	}

	c, err := s.UpdateComputer(t.Context(), "alice", "c-laptop", ComputerPatch{ShellEnabled: new(false)})
	require.NoError(t, err)
	assert.False(t, c.ShellEnabled)

	for _, id := range ids {
		assert.Equal(t, ShellCancelled, repo.shellJob(id).Status, id)
	}
	assert.Len(t, shells.cancelled, 4)
	_, err = s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "ls"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestRunCommand_AComputerThatCannotRunItFailsTheJob(t *testing.T) {
	s, repo, shells := newShellService(t)
	shells.runErr = apperrs.ErrRetryable
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "ls"})
	require.NoError(t, err)

	got, err := s.WaitShellJob(t.Context(), "alice", job.ID)
	require.NoError(t, err)
	assert.Equal(t, ShellFailed, got.Status)
	assert.Equal(t, "the computer's Nexul app is not connected", repo.shellJob(job.ID).Error)
}

func TestRunCommand_ValidatesTheCommand(t *testing.T) {
	s, _, _ := newShellService(t)
	for name, in := range map[string]RunCommandInput{
		"blank":         {Command: "  "},
		"unknown shell": {Command: "ls", Shell: "zsh"},
		"over an hour":  {Command: "ls", Timeout: 2 * time.Hour},
		"too long":      {Command: strings.Repeat("x", maxShellCommand+1)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.RunCommand(t.Context(), "alice", "c-laptop", in)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestShellJob_KeepsTheLastMiBOfOutput(t *testing.T) {
	s, repo, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "yes"})
	require.NoError(t, err)
	waitStarted(t, shells)
	line := strings.Repeat("y", 1023)
	for range 1100 {
		shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: line}})
	}
	shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "last"}})
	shells.sink(job.ID).Done(ShellEnd{Status: ShellSucceeded})

	got := repo.shellJob(job.ID)
	assert.True(t, got.OutputTruncated)
	assert.LessOrEqual(t, len(got.Output), maxShellOutput)
	assert.True(t, strings.HasSuffix(got.Output, "last\n"))
}

func TestOpenShellOutput_ReplaysThenFollowsARunningJob(t *testing.T) {
	s, _, shells := newShellService(t)
	job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "build"})
	require.NoError(t, err)
	waitStarted(t, shells)
	shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "one"}})

	feed, err := s.OpenShellOutput(t.Context(), "alice", job.ID)
	require.NoError(t, err)
	defer feed.Close()
	first, err := feed.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "one", first[0].Line)

	shells.sink(job.ID).Output([]containerlog.Line{{Stream: "stdout", Line: "two"}})
	shells.sink(job.ID).Done(ShellEnd{Status: ShellSucceeded})
	second, err := feed.Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "two", second[0].Line)
	_, err = feed.Next(t.Context())
	require.ErrorIs(t, err, io.EOF)
}

func TestCancelShellJob_StopsAQueuedOneWithoutTheRunner(t *testing.T) {
	s, repo, shells := newShellService(t)
	var last *ShellJob
	for range 5 {
		job, err := s.RunCommand(t.Context(), "alice", "c-laptop", RunCommandInput{Command: "sleep 9"})
		require.NoError(t, err)
		last = job
	}
	for range 4 {
		waitStarted(t, shells)
	}

	got, err := s.CancelShellJob(t.Context(), "alice", last.ID)
	require.NoError(t, err)

	assert.Equal(t, ShellCancelled, got.Status)
	assert.Equal(t, ShellCancelled, repo.shellJob(last.ID).Status)
	assert.Empty(t, shells.cancelled)
}

func TestRunShellUpkeep_EndsUnfinishedJobsAndPurgesOldOutput(t *testing.T) {
	s, repo, _ := newShellService(t)
	old := time.Now().Add(-31 * 24 * time.Hour)
	repo.jobs["j-old"] = ShellJob{ID: "j-old", Status: ShellSucceeded, Command: "ls", Output: "secret\n", EndedAt: &old}
	repo.jobs["j-left"] = ShellJob{ID: "j-left", Status: ShellRunning, Command: "make"}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	s.RunShellUpkeep(ctx, slog.New(slog.DiscardHandler))

	assert.Equal(t, ShellFailed, repo.shellJob("j-left").Status)
	purged := repo.shellJob("j-old")
	assert.Empty(t, purged.Output)
	assert.True(t, purged.OutputPurged)
	assert.Equal(t, "ls", purged.Command)
}

func TestCommandRunTool_WaitsAndReturnsTheEndOfTheOutput(t *testing.T) {
	s, repo, shells := newShellService(t)
	go func() {
		id := waitStarted(t, shells)
		shells.sink(id).Output([]containerlog.Line{{Stream: "stdout", Line: strings.Repeat("a", mcpOutputTail)}, {Stream: "stderr", Line: "boom"}})
		shells.sink(id).Done(ShellEnd{Status: ShellFailed, ExitCode: 2})
	}()

	got, err := callTool(t, actorCtx(t, "alice"), s, "command_run", `{"computer_id":"c-laptop","command":"make test","timeout_seconds":120}`)
	require.NoError(t, err)

	r := got.(commandResult)
	assert.Equal(t, ShellFailed, r.Status)
	require.NotNil(t, r.ExitCode)
	assert.Equal(t, 2, *r.ExitCode)
	assert.True(t, r.OutputTruncated)
	assert.Len(t, r.Output, mcpOutputTail)
	assert.True(t, strings.HasSuffix(r.Output, "boom\n"))
	assert.Equal(t, AdapterMCP, repo.shellJob(r.JobID).Adapter)
}

func TestCommandRunTool_RefusesWhatItCannotWaitFor(t *testing.T) {
	s, _, shells := newShellService(t)

	_, err := callTool(t, actorCtx(t, "alice"), s, "command_run", `{"computer_id":"c-laptop","command":"sleep 600","timeout_seconds":301}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = callTool(t, actorCtx(t, "mallory"), s, "command_run", `{"computer_id":"c-laptop","command":"id"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Zero(t, shells.runCount())
}

func TestCommandRoutes_AnswerNotFoundToAnyoneElse(t *testing.T) {
	s, _, shells := newShellService(t)
	h := NewHandler(s).Routes()
	res := doRequest(h, http.MethodPost, "/api/pairing/computers/c-laptop/commands", "alice", map[string]any{"command": "uname -a"})
	require.Equal(t, http.StatusAccepted, res.Code, res.Body.String())
	var job ShellJob
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &job))
	waitStarted(t, shells)

	for _, req := range []struct{ method, path string }{
		{http.MethodPost, "/api/pairing/computers/c-laptop/commands"},
		{http.MethodGet, "/api/pairing/computers/c-laptop/commands"},
		{http.MethodGet, "/api/pairing/commands/" + job.ID},
		{http.MethodPost, "/api/pairing/commands/" + job.ID + "/cancel"},
	} {
		res := doRequest(h, req.method, req.path, "mallory", map[string]any{"command": "id"})
		assert.Equal(t, http.StatusNotFound, res.Code, req.method+" "+req.path)
	}
	res = doRequest(h, http.MethodGet, "/api/pairing/computers/c-laptop/commands", "alice", nil)
	require.Equal(t, http.StatusOK, res.Code)
	assert.Contains(t, res.Body.String(), `"total":1`)
}
