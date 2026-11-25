package composite

import (
	"context"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

// maxCommandTimeout caps a command an agent runs, which it waits for in one tool call.
const maxCommandTimeout = 5 * time.Minute

// ComputerCommands runs commands on paired computers: the pairing use-cases.
type ComputerCommands interface {
	RunCommand(ctx context.Context, userID, computerID string, in shelljob.Input) (*shelljob.Job, error)
	WaitShellJob(ctx context.Context, userID, jobID string) (*shelljob.Job, error)
}

// MachineCommands runs commands on machines' runners: the runner use-cases.
type MachineCommands interface {
	RunCommand(ctx context.Context, runnerID string, in shelljob.Input) (*shelljob.Job, error)
	WaitShellJob(ctx context.Context, jobID string) (*shelljob.Job, error)
}

// CommandTools returns command_run, which reaches a computer or a machine's runner (ADR 0209).
func CommandTools(computers ComputerCommands, machines MachineCommands) []mcptool.Tool {
	return []mcptool.Tool{commandRunTool(computers, machines)}
}

type commandRunIn struct {
	ComputerID     string `json:"computer_id,omitempty" jsonschema:"One of your computers, from computer_list."`
	RunnerID       string `json:"runner_id,omitempty" jsonschema:"A machine's runner, from machine_list, in place of computer_id."`
	Command        string `json:"command" jsonschema:"The command line, run by bash -c (else sh) on Linux and macOS, or pwsh -Command (else Windows PowerShell) on Windows. No input is sent to it."`
	Shell          string `json:"shell,omitempty" enum:"bash,sh,pwsh,powershell" jsonschema:"A shell other than the default."`
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

func commandRunTool(computers ComputerCommands, machines MachineCommands) mcptool.Tool {
	return mcptool.New("command_run", "Run command",
		"Runs one command on one of your computers, or one shared with you by computers:run_commands, through its "+
			"Nexul app as the person it runs as there; or, with runner_id, on a machine's runner, as the user it runs as "+
			"(root on Linux), which takes machines:shell. It waits for the end. Use it to reach another computer than "+
			"the one you run on, or a server. Returns status (succeeded, failed, timed_out, cancelled, or refused when "+
			"the computer or runner turned commands off), exit_code, and the last 64 KB of output with stdout and stderr "+
			"interleaved. Every command is recorded with who ran it; computer_list or machine_list with id and commands "+
			"reads the record back.",
		mcptool.Hints{},
		func(ctx context.Context, in commandRunIn) (any, error) {
			if (in.ComputerID == "") == (in.RunnerID == "") {
				return nil, fmt.Errorf("%w: name computer_id or runner_id, not both", apperrs.ErrInvalid)
			}
			timeout := time.Duration(in.TimeoutSeconds) * time.Second
			if timeout > maxCommandTimeout {
				return nil, fmt.Errorf("%w: timeout_seconds is at most %d over MCP", apperrs.ErrInvalid, int(maxCommandTimeout/time.Second))
			}
			cmd := shelljob.Input{Command: in.Command, Shell: in.Shell, Cwd: in.Cwd, Timeout: timeout, Adapter: shelljob.AdapterMCP}
			job, err := runCommand(ctx, computers, machines, in, cmd)
			if err != nil {
				return nil, err
			}
			job.TailForAgent()
			return commandResult{
				JobID: job.ID, Status: job.Status, ExitCode: job.ExitCode, Error: job.Error, Output: job.Output,
				OutputTruncated: job.OutputTruncated,
			}, nil
		})
}

// runCommand starts the command where in names and waits for it to end.
func runCommand(ctx context.Context, computers ComputerCommands, machines MachineCommands, in commandRunIn, cmd shelljob.Input) (*shelljob.Job, error) {
	if in.RunnerID != "" {
		job, err := machines.RunCommand(ctx, in.RunnerID, cmd)
		if err != nil {
			return nil, err
		}
		wait, cancel := waitFor(ctx, job)
		defer cancel()
		return machines.WaitShellJob(wait, job.ID)
	}
	a, _ := identity.ActorFromCtx(ctx)
	job, err := computers.RunCommand(ctx, a.ID, in.ComputerID, cmd)
	if err != nil {
		return nil, err
	}
	wait, cancel := waitFor(ctx, job)
	defer cancel()
	return computers.WaitShellJob(wait, a.ID, job.ID)
}

// waitFor bounds the wait for job to its timeout and the grace its runner has to answer.
func waitFor(ctx context.Context, job *shelljob.Job) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second+shelljob.AnswerGrace)
}
