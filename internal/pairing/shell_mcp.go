package pairing

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

const (
	// maxMCPCommandTimeout caps a command an agent runs, which it waits for in one tool call.
	maxMCPCommandTimeout = 5 * time.Minute
	// mcpOutputTail is the end of a command's output an agent reads back.
	mcpOutputTail = 64 << 10
	// mcpCommandsListed is how many recent commands computer_list returns with commands.
	mcpCommandsListed = 10
)

type commandRunIn struct {
	ComputerID     string `json:"computer_id" jsonschema:"One of your computers, from computer_list."`
	Command        string `json:"command" jsonschema:"The command line, run by bash -c (else sh) on Linux and macOS, or pwsh -Command (else Windows PowerShell) on Windows. No input is sent to it."`
	Shell          string `json:"shell,omitempty" enum:"bash,sh,pwsh,powershell" jsonschema:"A shell other than the computer's default."`
	Cwd            string `json:"cwd,omitempty" jsonschema:"The folder it starts in, for example ~/code/app; omit for the home folder."`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"Seconds it may run before it and everything it started are killed, 1 to 300; 60 when omitted."`
}

// commandResult is a command as an agent reads it back: how it ended and the end of its output.
type commandResult struct {
	JobID           string `json:"job_id"`
	Status          string `json:"status"`
	ExitCode        *int   `json:"exit_code,omitempty"`
	Error           string `json:"error,omitempty"`
	Output          string `json:"output"`
	OutputTruncated bool   `json:"output_truncated,omitempty"`
}

func commandRunTool(s *Service) mcptool.Tool {
	return mcptool.New("command_run", "Run command on computer",
		"Runs one command on one of your computers, or one shared with you by computers:run_commands, through its "+
			"Nexul app, as the person it runs as there, and waits "+
			"for it to end. Use it to reach another computer than the one you run on, for example to test on a "+
			"Windows machine. Returns status (succeeded, failed, timed_out, cancelled, or refused when the computer "+
			"turned commands off with nexul shell off), exit_code, and the last 64 KB of output with stdout and "+
			"stderr interleaved. Every command is recorded with who ran it; computer_list with id and commands reads "+
			"the record back.",
		mcptool.Hints{},
		func(ctx context.Context, in commandRunIn) (any, error) {
			timeout := time.Duration(in.TimeoutSeconds) * time.Second
			if timeout > maxMCPCommandTimeout {
				return nil, fmt.Errorf("%w: timeout_seconds is at most %d over MCP", apperrs.ErrInvalid, int(maxMCPCommandTimeout/time.Second))
			}
			userID := mcpActorID(ctx)
			job, err := s.RunCommand(ctx, userID, in.ComputerID, RunCommandInput{
				Command: in.Command, Shell: in.Shell, Cwd: in.Cwd, Timeout: timeout, Adapter: AdapterMCP,
			})
			if err != nil {
				return nil, err
			}
			wait, cancel := context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second+shellAnswerGrace)
			defer cancel()
			if job, err = s.WaitShellJob(wait, userID, job.ID); err != nil {
				return nil, err
			}
			return toCommandResult(*job), nil
		})
}

func toCommandResult(job ShellJob) commandResult {
	r := commandResult{JobID: job.ID, Status: job.Status, ExitCode: job.ExitCode, Error: job.Error, Output: job.Output, OutputTruncated: job.OutputTruncated}
	if len(r.Output) > mcpOutputTail {
		r.Output, r.OutputTruncated = r.Output[len(r.Output)-mcpOutputTail:], true
	}
	return r
}

type computerCommandsIn struct {
	JobID string `json:"job_id,omitempty" jsonschema:"One command's job_id, to read it with its output; omit for the newest 10 without output."`
}

// commandsFor reads one command with its output, or a computer's newest commands without it.
func commandsFor(ctx context.Context, s *Service, userID string, c Computer, in computerCommandsIn) ([]ShellJob, error) {
	if in.JobID == "" {
		jobs, _, err := s.ListShellJobs(ctx, userID, c.ID, paging.Window{Limit: mcpCommandsListed})
		return jobs, err
	}
	job, err := s.GetShellJob(ctx, userID, in.JobID)
	if err != nil {
		return nil, err
	}
	if job.ComputerID != c.ID {
		return nil, fmt.Errorf("%w: command %s did not run on %s", apperrs.ErrNotFound, in.JobID, c.Name)
	}
	r := toCommandResult(*job)
	job.Output, job.OutputTruncated = r.Output, r.OutputTruncated
	return []ShellJob{*job}, nil
}
